package dedup

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/welllog/otool/internal/storage"
)

type Engine struct {
	cacheOnce sync.Once
	cacheMgr  *storage.CacheManager
}

func NewEngine() *Engine {
	return &Engine{}
}

// cache initializes the process-wide cache manager lazily. Loading the
// fingerprint JSON at app startup is unnecessary for users who never open the
// dedup page, and can be significant on a large photo library.
func (e *Engine) cache() *storage.CacheManager {
	e.cacheOnce.Do(func() {
		e.cacheMgr = storage.GetCacheManager()
	})
	return e.cacheMgr
}

func itemCacheKey(store storage.Storage, item storage.FileItem) string {
	sID := item.StorageID
	if sID == "" {
		sID = store.Driver()
	}
	return storage.GenerateCacheKey(sID, item.Path, item.Size, item.ModTime, item.ETag)
}

// decodeMemoryGate bounds the total estimated memory used by concurrent image
// decodes. A 50MP photo can need ~200MB while being decoded; without a gate,
// several workers could easily exhaust several GB even though the final work
// image is only 256px.
type decodeMemoryGate struct {
	mu     sync.Mutex
	cond   *sync.Cond
	budget int64
	used   int64
}

func newDecodeMemoryGate(budget int64) *decodeMemoryGate {
	if budget <= 0 {
		budget = 512 << 20
	}
	g := &decodeMemoryGate{budget: budget}
	g.cond = sync.NewCond(&g.mu)
	return g
}

func (g *decodeMemoryGate) acquire(ctx context.Context, want int64) (int64, bool) {
	if want <= 0 {
		return 0, false
	}
	if want > g.budget {
		return 0, false
	}

	// sync.Cond.Wait cannot observe a context by itself. Broadcast on
	// cancellation so a worker blocked behind a long-running decode wakes up,
	// re-checks ctx.Err and leaves instead of waiting for the next release.
	stopCtxWake := context.AfterFunc(ctx, func() {
		g.mu.Lock()
		g.cond.Broadcast()
		g.mu.Unlock()
	})
	defer stopCtxWake()

	g.mu.Lock()
	defer g.mu.Unlock()
	for g.used+want > g.budget {
		if ctx.Err() != nil {
			return 0, false
		}
		g.cond.Wait()
	}
	if ctx.Err() != nil {
		return 0, false
	}
	g.used += want
	return want, true
}

func (g *decodeMemoryGate) release(reserved int64) {
	if reserved <= 0 {
		return
	}
	g.mu.Lock()
	g.used -= reserved
	if g.used < 0 {
		g.used = 0
	}
	g.mu.Unlock()
	g.cond.Broadcast()
}

// RunScan executes the full deduplication pipeline.
func (e *Engine) RunScan(ctx context.Context, store storage.Storage, opts ScanOptions, onProgress func(p ScanProgress)) ([]*DuplicateGroup, error) {
	if opts.WorkerCount <= 0 {
		opts.WorkerCount = defaultWorkerCount()
	}
	if opts.SimilarDistance < 0 {
		opts.SimilarDistance = 8
	}
	if opts.SimilarDistance > 15 {
		return nil, fmt.Errorf("相似度阈值必须在 0~15 之间")
	}
	if opts.MemoryBudget <= 0 {
		opts.MemoryBudget = 512 << 20
	}

	report := func(p ScanProgress) {
		if onProgress != nil {
			onProgress(p)
		}
	}

	if opts.TargetDir == "" && store.Driver() == storage.DriverLocal {
		return nil, fmt.Errorf("扫描目标目录不能为空")
	}

	// 1. Permission check
	if !opts.ReadOnly {
		w, d, err := store.CheckPermission(ctx, opts.TargetDir)
		if err != nil {
			return nil, fmt.Errorf("存储权限检查未通过: %w", err)
		}
		if !w || !d {
			return nil, fmt.Errorf("存储权限检查未通过: 缺少写入或删除权限 (可勾选“只读扫描”仅分析)")
		}
	}

	// 2. Discover files
	report(ScanProgress{
		Phase:      "discovering",
		PhaseTitle: "正在扫描目录结构并检索图片...",
	})

	var discoveredItems []storage.FileItem
	skippedDirs := 0
	skippedFiles := 0
	err := store.List(ctx, opts.TargetDir, opts.Recursive, func(item storage.FileItem) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		discoveredItems = append(discoveredItems, item)
		if len(discoveredItems)%25 == 0 {
			report(ScanProgress{
				Phase:           "discovering",
				PhaseTitle:      fmt.Sprintf("正在检索图片 (已发现 %d 张)...", len(discoveredItems)),
				TotalDiscovered: len(discoveredItems),
			})
		}
		return nil
	})
	if err != nil {
		var partial *storage.PartialListError
		if errors.As(err, &partial) {
			// Unreadable subdirectories are reported as a warning, not a fatal
			// error: the rest of the photo library is still scanned.
			skippedDirs = len(partial.SkippedPaths)
			skippedFiles = partial.SkippedFiles
		} else {
			return nil, fmt.Errorf("扫描目录失败: %w", err)
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Similar mode needs a real image decoder. Exact mode can still hash
	// HEIC/RAW files byte-for-byte, so only filter the perceptual path.
	if opts.Mode != ModeExact {
		filtered := discoveredItems[:0]
		for _, item := range discoveredItems {
			if storage.IsDecodableImageFile(item.Name) {
				filtered = append(filtered, item)
			} else {
				skippedFiles++
			}
		}
		discoveredItems = filtered
	}

	totalDiscovered := len(discoveredItems)
	if totalDiscovered < 2 {
		title := "扫描完成"
		if skippedFiles > 0 {
			title += fmt.Sprintf("（%d 张图片无法读取或格式不支持，已跳过）", skippedFiles)
		}
		if skippedDirs > 0 {
			title += fmt.Sprintf("，%d 个目录无法读取已跳过", skippedDirs)
		}
		report(ScanProgress{
			Phase:           "done",
			PhaseTitle:      title,
			TotalDiscovered: totalDiscovered,
			SkippedCount:    skippedFiles,
			SkippedDirs:     skippedDirs,
			Percent:         1.0,
			IsComplete:      true,
		})
		return nil, nil
	}

	// 3. Exact Mode vs Similar Mode pipelines
	if opts.Mode == ModeExact {
		return e.runExactScan(ctx, store, discoveredItems, opts, report, skippedDirs, skippedFiles)
	}

	return e.runSimilarScan(ctx, store, discoveredItems, opts, report, skippedDirs, skippedFiles)
}

func skippedDirsSuffix(n int) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("（%d 个目录无法读取，已跳过）", n)
}

// fillGroupContentHashes computes the full-file hash for photos that only have
// perceptual fingerprints (similar mode). The hash is required before deletion
// so a same-size/same-mtime replacement cannot be mistaken for the scanned
// photo. Exact mode already carries the hash from the content-hash phase.
func (e *Engine) fillGroupContentHashes(ctx context.Context, store storage.Storage, groups []*DuplicateGroup, workerCount int) (int, error) {
	if len(groups) == 0 {
		return 0, nil
	}
	tasks := make(chan *PhotoMeta, 0)
	var wg sync.WaitGroup
	var skipped int64

	if workerCount < 1 {
		workerCount = 1
	}
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for photo := range tasks {
				select {
				case <-ctx.Done():
					return
				default:
				}
				if photo.ContentHash != 0 {
					continue
				}
				rc, err := store.Open(ctx, photo.Path)
				if err != nil {
					atomic.AddInt64(&skipped, 1)
					continue
				}
				hash, hashErr := ComputeFullContentHashContext(ctx, rc)
				_ = rc.Close()
				if hashErr != nil {
					atomic.AddInt64(&skipped, 1)
					continue
				}
				photo.ContentHash = hash
				// Persist the hash so the next scan can reuse it from the
				// fingerprint cache instead of re-reading group members from
				// the store (a full download for remote storage).
				if photo.CacheKey != "" {
					e.cache().UpdateFingerprintContentHash(photo.CacheKey, hash)
				}
			}
		}()
	}

	sendErr := make(chan error, 1)
	go func() {
		defer close(tasks)
		for _, group := range groups {
			for _, photo := range group.Photos {
				select {
				case <-ctx.Done():
					sendErr <- ctx.Err()
					return
				case tasks <- photo:
				}
			}
		}
		sendErr <- nil
	}()

	wg.Wait()
	if err := <-sendErr; err != nil {
		return int(atomic.LoadInt64(&skipped)), err
	}
	return int(atomic.LoadInt64(&skipped)), nil
}

// analyzeItem opens an item, spools remote/non-seekable streams to a temporary
// file, probes its dimensions, acquires a decode-memory budget and decodes it.
func (e *Engine) analyzeItem(ctx context.Context, store storage.Storage, item storage.FileItem, gate *decodeMemoryGate, wantThumb bool) (*AnalysisResult, error) {
	rc, err := store.Open(ctx, item.Path)
	if err != nil {
		return nil, err
	}

	var reader io.ReadSeeker
	var spool *os.File
	var closers []io.Closer
	if rs, ok := rc.(io.ReadSeeker); ok {
		reader = rs
		closers = append(closers, rc)
		defer rc.Close()
	} else {
		spool, err = os.CreateTemp("", "otool-dedup-*")
		if err != nil {
			_ = rc.Close()
			return nil, err
		}
		spoolName := spool.Name()
		defer func() {
			_ = spool.Close()
			_ = os.Remove(spoolName)
		}()

		if _, err := io.Copy(spool, rc); err != nil {
			_ = rc.Close()
			return nil, err
		}
		_ = rc.Close()
		if _, err := spool.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		reader = spool
		closers = append(closers, spool)
	}
	// Closing the underlying file unblocks a long image.Decode when the user
	// stops a scan; without this, a 50MP decode can outlive the cancellation.
	stopClose := context.AfterFunc(ctx, func() {
		for _, c := range closers {
			_ = c.Close()
		}
	})
	defer stopClose()

	probe, err := ProbeImage(&contextReadSeeker{ctx: ctx, rs: reader})
	if err != nil {
		return nil, err
	}

	estimated := estimateDecodeMemory(probe.Width, probe.Height, probe.BytesPerPixel)
	// Never let a single image reserve more than the whole decode budget:
	// attempting it risks the process being OOM killed instead of simply
	// skipping one unreadable photo.
	if estimated > gate.budget {
		return nil, fmt.Errorf("图片尺寸过大，已跳过解码: %dx%d", probe.Width, probe.Height)
	}

	reserved, ok := gate.acquire(ctx, estimated)
	if !ok {
		return nil, ctx.Err()
	}
	defer gate.release(reserved)

	if wantThumb {
		return AnalyzeImageReaderContext(ctx, reader, item.Name)
	}
	return AnalyzeImageReaderContextNoThumb(ctx, reader, item.Name)
}

// runExactScan executes an ultra-fast pipelined exact duplicate check:
// Size clustering -> Partial block xxHash -> Full stream xxHash -> one decode
// per confirmed group (all copies in a group are byte-identical).
func (e *Engine) runExactScan(
	ctx context.Context,
	store storage.Storage,
	items []storage.FileItem,
	opts ScanOptions,
	report func(p ScanProgress),
	skippedDirs int,
	skippedFiles int,
) ([]*DuplicateGroup, error) {
	totalDiscovered := len(items)
	skippedCount := int64(skippedFiles)
	// Exact groups whose representative failed to decode keep all their
	// photos in the results (equality is proven by the byte hash); they are
	// reported separately instead of being miscounted as skipped photos.
	previewFailed := int64(0)

	// Step 1: Filter by file size
	sizeMap := make(map[int64][]storage.FileItem)
	for _, item := range items {
		sizeMap[item.Size] = append(sizeMap[item.Size], item)
	}
	var sizeCandidates []storage.FileItem
	for _, group := range sizeMap {
		if len(group) >= 2 {
			sizeCandidates = append(sizeCandidates, group...)
		}
	}

	if len(sizeCandidates) < 2 {
		report(ScanProgress{
			Phase:           "done",
			PhaseTitle:      "未发现完全相同的重复图片" + skippedDirsSuffix(skippedDirs),
			TotalDiscovered: totalDiscovered,
			SkippedCount:    int(atomic.LoadInt64(&skippedCount)),
			SkippedDirs:     skippedDirs,
			Percent:         1.0,
			IsComplete:      true,
		})
		return nil, nil
	}

	report(ScanProgress{
		Phase:           "filtering",
		PhaseTitle:      fmt.Sprintf("正在进行首尾分段哈希初筛 (%d 张候选)...", len(sizeCandidates)),
		TotalDiscovered: totalDiscovered,
		TotalToAnalyze:  len(sizeCandidates),
		SkippedDirs:     skippedDirs,
		Percent:         0.2,
	})

	// Step 2: Partial Hash (head, middle, tail 4KB) using range requests.
	type partialResult struct {
		item storage.FileItem
		ph   uint64
		err  error
	}

	partialChan := make(chan storage.FileItem, len(sizeCandidates))
	for _, it := range sizeCandidates {
		partialChan <- it
	}
	close(partialChan)

	var (
		pwg            sync.WaitGroup
		partialResChan = make(chan partialResult, len(sizeCandidates))
		pWorkerCount   = opts.WorkerCount
	)
	if pWorkerCount > len(sizeCandidates) {
		pWorkerCount = len(sizeCandidates)
	}
	if pWorkerCount < 1 {
		pWorkerCount = 1
	}

	for w := 0; w < pWorkerCount; w++ {
		pwg.Add(1)
		go func() {
			defer pwg.Done()
			for it := range partialChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				ph, err := ComputePartialHashFromStore(ctx, store, it.Path, it.Size)
				partialResChan <- partialResult{item: it, ph: ph, err: err}
			}
		}()
	}

	pwg.Wait()
	close(partialResChan)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Group by (Size + PartialHash)
	type sizeHashKey struct {
		size int64
		ph   uint64
	}
	partialMap := make(map[sizeHashKey][]storage.FileItem)
	for res := range partialResChan {
		if res.err != nil {
			atomic.AddInt64(&skippedCount, 1)
			continue
		}
		k := sizeHashKey{size: res.item.Size, ph: res.ph}
		partialMap[k] = append(partialMap[k], res.item)
	}

	var fullHashCandidates []storage.FileItem
	for _, grp := range partialMap {
		if len(grp) >= 2 {
			fullHashCandidates = append(fullHashCandidates, grp...)
		}
	}

	if len(fullHashCandidates) < 2 {
		report(ScanProgress{
			Phase:           "done",
			PhaseTitle:      "未发现完全相同的重复图片" + skippedDirsSuffix(skippedDirs),
			TotalDiscovered: totalDiscovered,
			SkippedCount:    int(atomic.LoadInt64(&skippedCount)),
			SkippedDirs:     skippedDirs,
			Percent:         1.0,
			IsComplete:      true,
		})
		return nil, nil
	}

	// Step 3: Stream Full Content xxHash
	report(ScanProgress{
		Phase:           "analyzing",
		PhaseTitle:      fmt.Sprintf("正在校验完整文件哈希 (%d 张候选)...", len(fullHashCandidates)),
		TotalDiscovered: totalDiscovered,
		TotalToAnalyze:  len(fullHashCandidates),
		SkippedDirs:     skippedDirs,
		Percent:         0.5,
	})

	type fullHashResult struct {
		item        storage.FileItem
		contentHash uint64
		err         error
	}

	fullChan := make(chan storage.FileItem, len(fullHashCandidates))
	for _, it := range fullHashCandidates {
		fullChan <- it
	}
	close(fullChan)

	var (
		fwg           sync.WaitGroup
		fullResChan   = make(chan fullHashResult, len(fullHashCandidates))
		fullProcessed int64
		totFull       = len(fullHashCandidates)
		fWorkerCount  = opts.WorkerCount
	)
	if fWorkerCount > totFull {
		fWorkerCount = totFull
	}
	if fWorkerCount < 1 {
		fWorkerCount = 1
	}

	for w := 0; w < fWorkerCount; w++ {
		fwg.Add(1)
		go func() {
			defer fwg.Done()
			for it := range fullChan {
				select {
				case <-ctx.Done():
					return
				default:
				}
				rc, err := store.Open(ctx, it.Path)
				if err != nil {
					fullResChan <- fullHashResult{item: it, err: err}
					continue
				}
				ch, err := ComputeFullContentHashContext(ctx, rc)
				_ = rc.Close()
				fullResChan <- fullHashResult{item: it, contentHash: ch, err: err}

				done := atomic.AddInt64(&fullProcessed, 1)
				if done%5 == 0 || int(done) == totFull {
					pct := 0.5 + 0.3*(float64(done)/float64(totFull))
					report(ScanProgress{
						Phase:           "analyzing",
						PhaseTitle:      fmt.Sprintf("正在校验文件哈希 (%d/%d)...", done, totFull),
						TotalDiscovered: totalDiscovered,
						TotalToAnalyze:  totFull,
						CurrentAnalyzed: int(done),
						CurrentFile:     it.Name,
						SkippedDirs:     skippedDirs,
						Percent:         pct,
					})
				}
			}
		}()
	}

	fwg.Wait()
	close(fullResChan)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Group by full content hash
	exactDupMap := make(map[uint64][]storage.FileItem)
	for res := range fullResChan {
		if res.err != nil {
			atomic.AddInt64(&skippedCount, 1)
			continue
		}
		if res.contentHash != 0 {
			exactDupMap[res.contentHash] = append(exactDupMap[res.contentHash], res.item)
		}
	}

	type exactGroup struct {
		hash  uint64
		items []storage.FileItem
	}
	var exactGroups []exactGroup
	var confirmedDupCount int
	for hash, grp := range exactDupMap {
		if len(grp) >= 2 {
			sort.Slice(grp, func(i, j int) bool { return grp[i].Path < grp[j].Path })
			exactGroups = append(exactGroups, exactGroup{hash: hash, items: grp})
			confirmedDupCount += len(grp)
		}
	}
	sort.Slice(exactGroups, func(i, j int) bool { return exactGroups[i].hash < exactGroups[j].hash })

	if len(exactGroups) == 0 {
		report(ScanProgress{
			Phase:           "done",
			PhaseTitle:      "未发现完全相同的重复图片" + skippedDirsSuffix(skippedDirs),
			TotalDiscovered: totalDiscovered,
			SkippedCount:    int(atomic.LoadInt64(&skippedCount)),
			SkippedDirs:     skippedDirs,
			Percent:         1.0,
			IsComplete:      true,
		})
		return nil, nil
	}

	// Step 4: Decode one representative per exact group and clone its metadata
	// to the other byte-identical copies. Decoding every copy was a huge CPU
	// and memory waste for groups containing dozens of duplicates.
	report(ScanProgress{
		Phase:           "analyzing",
		PhaseTitle:      fmt.Sprintf("正在生成重复项预览图 (%d 组 / %d 张)...", len(exactGroups), confirmedDupCount),
		TotalDiscovered: totalDiscovered,
		TotalToAnalyze:  len(exactGroups),
		SkippedDirs:     skippedDirs,
		Percent:         0.85,
	})

	type exactMetaResult struct {
		hash   uint64
		photos []*PhotoMeta
	}
	metaChan := make(chan exactGroup, len(exactGroups))
	for _, g := range exactGroups {
		metaChan <- g
	}
	close(metaChan)

	var (
		mwg        sync.WaitGroup
		metaResCh  = make(chan exactMetaResult, len(exactGroups))
		mDone      int64
		totMeta    = len(exactGroups)
		metaWorker = opts.WorkerCount
		gate       = newDecodeMemoryGate(opts.MemoryBudget)
	)
	if metaWorker > totMeta {
		metaWorker = totMeta
	}
	if metaWorker < 1 {
		metaWorker = 1
	}

	for w := 0; w < metaWorker; w++ {
		mwg.Add(1)
		go func() {
			defer mwg.Done()
			for g := range metaChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				rep := g.items[0]
				res, analyzeErr := e.analyzeItem(ctx, store, rep, gate, true)
				if analyzeErr != nil {
					// Exact equality is proven by the byte hash; an unsupported
					// HEIC/RAW or corrupt file must not remove the group, but
					// report that preview metadata could not be generated.
					res = nil
					atomic.AddInt64(&previewFailed, 1)
				}

				photos := make([]*PhotoMeta, 0, len(g.items))
				for _, it := range g.items {
					ck := itemCacheKey(store, it)
					pm := &PhotoMeta{
						StorageID:   it.StorageID,
						ETag:        it.ETag,
						Path:        it.Path,
						Name:        it.Name,
						Size:        it.Size,
						ModTime:     it.ModTime,
						ContentHash: g.hash,
						CacheKey:    ck,
					}
					if res != nil {
						pm.Width = res.Width
						pm.Height = res.Height
						pm.Format = res.Format
						pm.PHash = res.PHash
						pm.DHash = res.DHash
						pm.BlurScore = res.BlurScore
						pm.HasEXIF = res.HasEXIF
						pm.CameraModel = res.CameraModel
						pm.DateTimeOrig = res.DateTimeOrig
						if len(res.ThumbnailBytes) > 0 {
							_ = e.cache().SaveThumbnail(ck, res.ThumbnailBytes)
						}
						// The group is byte-identical, so the representative's
						// fingerprint is valid for every copy. Populate the cache
						// here so a later similar-mode scan can reuse it.
						e.cache().SaveFingerprint(ck, storage.CachedFingerprint{
							Width:        res.Width,
							Height:       res.Height,
							Format:       res.Format,
							ContentHash:  g.hash,
							PHash:        res.PHash,
							DHash:        res.DHash,
							BlurScore:    res.BlurScore,
							HasEXIF:      res.HasEXIF,
							CameraModel:  res.CameraModel,
							DateTimeOrig: res.DateTimeOrig,
						})
					}
					photos = append(photos, pm)
				}
				metaResCh <- exactMetaResult{hash: g.hash, photos: photos}

				done := atomic.AddInt64(&mDone, 1)
				if done%2 == 0 || int(done) == totMeta {
					report(ScanProgress{
						Phase:           "analyzing",
						PhaseTitle:      fmt.Sprintf("正在生成重复项预览图 (%d/%d 组)...", done, totMeta),
						TotalDiscovered: totalDiscovered,
						TotalToAnalyze:  totMeta,
						CurrentAnalyzed: int(done),
						CurrentFile:     rep.Name,
						SkippedDirs:     skippedDirs,
						Percent:         0.85 + 0.1*(float64(done)/float64(totMeta)),
					})
				}
			}
		}()
	}

	mwg.Wait()
	close(metaResCh)

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	metaResults := make(map[uint64][]*PhotoMeta, len(exactGroups))
	for res := range metaResCh {
		metaResults[res.hash] = res.photos
	}

	// Build DuplicateGroups in deterministic hash order.
	var groups []*DuplicateGroup
	var totalWasted int64
	var idCounter uint64

	for i, g := range exactGroups {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		photos := metaResults[g.hash]
		if len(photos) < 2 {
			continue
		}
		for _, pm := range photos {
			idCounter++
			pm.ID = idCounter
		}
		dg := &DuplicateGroup{
			GroupID: fmt.Sprintf("exact_%d", i+1),
			Mode:    ModeExact,
			Photos:  photos,
		}
		EvaluateAndScoreGroup(dg)
		totalWasted += dg.TotalWastedBytes
		groups = append(groups, dg)
	}

	finalTitle := fmt.Sprintf("精确去重扫描完成，发现 %d 组完全重复照片", len(groups))
	if skipped := int(atomic.LoadInt64(&skippedCount)); skipped > 0 {
		finalTitle += fmt.Sprintf("（%d 张图片无法读取或哈希校验失败，已跳过）", skipped)
	}
	if skippedDirs > 0 {
		finalTitle += fmt.Sprintf("，%d 个目录无法读取已跳过", skippedDirs)
	}
	previewFailedGroups := int(atomic.LoadInt64(&previewFailed))
	if previewFailedGroups > 0 {
		finalTitle += fmt.Sprintf("，%d 组未能生成预览信息（照片仍保留在结果中）", previewFailedGroups)
	}
	_ = e.cache().FlushFingerprints()
	report(ScanProgress{
		Phase:               "done",
		PhaseTitle:          finalTitle,
		TotalDiscovered:     totalDiscovered,
		TotalToAnalyze:      totMeta,
		CurrentAnalyzed:     totMeta,
		SkippedCount:        int(atomic.LoadInt64(&skippedCount)),
		SkippedDirs:         skippedDirs,
		PreviewFailedGroups: previewFailedGroups,
		FoundGroups:         len(groups),
		WastedBytes:         totalWasted,
		Percent:             1.0,
		IsComplete:          true,
	})

	return groups, nil
}

// runSimilarScan executes perceptual duplicate check with persistent fingerprint cache & BK-Tree.
func (e *Engine) runSimilarScan(
	ctx context.Context,
	store storage.Storage,
	items []storage.FileItem,
	opts ScanOptions,
	report func(p ScanProgress),
	skippedDirs int,
	skippedFiles int,
) ([]*DuplicateGroup, error) {
	totalDiscovered := len(items)
	totalToAnalyze := totalDiscovered

	report(ScanProgress{
		Phase:           "analyzing",
		PhaseTitle:      "正在并行读取与计算图片视觉指纹...",
		TotalDiscovered: totalDiscovered,
		TotalToAnalyze:  totalToAnalyze,
		SkippedDirs:     skippedDirs,
	})

	type workTask struct {
		id   uint64
		item storage.FileItem
	}

	taskChan := make(chan workTask, totalToAnalyze)
	for i, item := range items {
		taskChan <- workTask{id: uint64(i + 1), item: item}
	}
	close(taskChan)

	var (
		analyzedPhotos []*PhotoMeta
		photosMu       sync.Mutex
		processedCount int64
		skippedCount   = int64(skippedFiles)
		wg             sync.WaitGroup
		gate           = newDecodeMemoryGate(opts.MemoryBudget)
	)

	workerCount := opts.WorkerCount
	if workerCount > totalToAnalyze {
		workerCount = totalToAnalyze
	}
	if workerCount < 1 {
		workerCount = 1
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for task := range taskChan {
				select {
				case <-ctx.Done():
					return
				default:
				}

				item := task.item
				cacheKey := itemCacheKey(store, item)

				// 1. Check persistent fingerprint cache. A missing thumbnail is
				// no reason to discard a valid fingerprint: the UI can regenerate
				// previews lazily, while invalidating the fingerprint would force
				// a full download/decode of the original.
				if cachedFp, ok := e.cache().GetFingerprint(cacheKey); ok &&
					cachedFp.Width > 0 && cachedFp.Height > 0 {
					meta := &PhotoMeta{
						ID:           task.id,
						StorageID:    item.StorageID,
						ETag:         item.ETag,
						Path:         item.Path,
						Name:         item.Name,
						Size:         item.Size,
						ModTime:      item.ModTime,
						Width:        cachedFp.Width,
						Height:       cachedFp.Height,
						Format:       cachedFp.Format,
						ContentHash:  cachedFp.ContentHash,
						PHash:        cachedFp.PHash,
						DHash:        cachedFp.DHash,
						BlurScore:    cachedFp.BlurScore,
						HasEXIF:      cachedFp.HasEXIF,
						CameraModel:  cachedFp.CameraModel,
						DateTimeOrig: cachedFp.DateTimeOrig,
						CacheKey:     cacheKey,
					}
					photosMu.Lock()
					analyzedPhotos = append(analyzedPhotos, meta)
					photosMu.Unlock()

					done := atomic.AddInt64(&processedCount, 1)
					if done%10 == 0 || int(done) == totalToAnalyze {
						pct := float64(done) / float64(totalToAnalyze)
						report(ScanProgress{
							Phase:           "analyzing",
							PhaseTitle:      fmt.Sprintf("正在检索图片特征 (%d/%d, 已复用缓存)...", done, totalToAnalyze),
							TotalDiscovered: totalDiscovered,
							TotalToAnalyze:  totalToAnalyze,
							CurrentAnalyzed: int(done),
							CurrentFile:     item.Name,
							SkippedDirs:     skippedDirs,
							Percent:         pct * 0.9,
						})
					}
					continue
				}

				// 2. Fetch/stream and analyze
				res, err := e.analyzeItem(ctx, store, item, gate, false)
				if err != nil {
					atomic.AddInt64(&skippedCount, 1)
					done := atomic.AddInt64(&processedCount, 1)
					if done%5 == 0 || int(done) == totalToAnalyze {
						report(ScanProgress{
							Phase:           "analyzing",
							PhaseTitle:      fmt.Sprintf("正在分析图片 (%d/%d, 已跳过 %d 张)...", done, totalToAnalyze, atomic.LoadInt64(&skippedCount)),
							TotalDiscovered: totalDiscovered,
							TotalToAnalyze:  totalToAnalyze,
							CurrentAnalyzed: int(done),
							CurrentFile:     item.Name,
							SkippedDirs:     skippedDirs,
							Percent:         float64(done) / float64(totalToAnalyze) * 0.9,
						})
					}
					continue
				}

				// Save only the fingerprint here. Thumbnails are generated lazily
				// for photos that actually end up in a duplicate group.
				e.cache().SaveFingerprint(cacheKey, storage.CachedFingerprint{
					Width:        res.Width,
					Height:       res.Height,
					Format:       res.Format,
					ContentHash:  res.ContentHash,
					PHash:        res.PHash,
					DHash:        res.DHash,
					BlurScore:    res.BlurScore,
					HasEXIF:      res.HasEXIF,
					CameraModel:  res.CameraModel,
					DateTimeOrig: res.DateTimeOrig,
				})

				meta := &PhotoMeta{
					ID:           task.id,
					StorageID:    item.StorageID,
					ETag:         item.ETag,
					Path:         item.Path,
					Name:         item.Name,
					Size:         item.Size,
					ModTime:      item.ModTime,
					Width:        res.Width,
					Height:       res.Height,
					Format:       res.Format,
					ContentHash:  res.ContentHash,
					PHash:        res.PHash,
					DHash:        res.DHash,
					BlurScore:    res.BlurScore,
					HasEXIF:      res.HasEXIF,
					CameraModel:  res.CameraModel,
					DateTimeOrig: res.DateTimeOrig,
					CacheKey:     cacheKey,
				}

				photosMu.Lock()
				analyzedPhotos = append(analyzedPhotos, meta)
				photosMu.Unlock()

				done := atomic.AddInt64(&processedCount, 1)
				if done%5 == 0 || int(done) == totalToAnalyze {
					pct := float64(done) / float64(totalToAnalyze)
					report(ScanProgress{
						Phase:           "analyzing",
						PhaseTitle:      fmt.Sprintf("正在分析图片 (%d/%d)...", done, totalToAnalyze),
						TotalDiscovered: totalDiscovered,
						TotalToAnalyze:  totalToAnalyze,
						CurrentAnalyzed: int(done),
						CurrentFile:     item.Name,
						SkippedDirs:     skippedDirs,
						Percent:         pct * 0.9,
					})
				}
			}
		}()
	}

	wg.Wait()

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Flush persistent fingerprint cache to disk
	_ = e.cache().FlushFingerprints()

	// 3. Metric space clustering with BK-Tree
	report(ScanProgress{
		Phase:           "clustering",
		PhaseTitle:      "正在进行 BK-Tree 度量空间聚类分析...",
		TotalDiscovered: totalDiscovered,
		TotalToAnalyze:  totalToAnalyze,
		CurrentAnalyzed: len(analyzedPhotos),
		SkippedDirs:     skippedDirs,
		Percent:         0.95,
	})

	rawClusters := ClusterSimilarPhotosContext(ctx, analyzedPhotos, opts.SimilarDistance)
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 4. Build and score final DuplicateGroups
	var groups []*DuplicateGroup
	var totalWasted int64

	for i, cluster := range rawClusters {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		grp := &DuplicateGroup{
			GroupID: fmt.Sprintf("group_%d", i+1),
			Mode:    opts.Mode,
			Photos:  cluster,
		}
		EvaluateAndScoreGroup(grp)
		totalWasted += grp.TotalWastedBytes
		groups = append(groups, grp)
	}

	// Similar groups only have perceptual hashes at this point. Compute a full
	// content hash for their members so deletion can prove the file is still
	// the exact photo that was scanned.
	hashSkipped, hashErr := e.fillGroupContentHashes(ctx, store, groups, opts.WorkerCount)
	if hashErr != nil {
		return nil, hashErr
	}
	skippedCount += int64(hashSkipped)
	// The analysis phase flushed before clustering. The content hashes learned
	// afterwards by fillGroupContentHashes need their own flush, otherwise they
	// stay in memory and the next scan still re-reads every group member.
	_ = e.cache().FlushFingerprints()

	finalTitle := fmt.Sprintf("分析完成，发现 %d 组重复照片", len(groups))
	if skipped := int(atomic.LoadInt64(&skippedCount)); skipped > 0 {
		finalTitle += fmt.Sprintf("（%d 张图片无法读取或解码，已跳过）", skipped)
	}
	if skippedDirs > 0 {
		finalTitle += fmt.Sprintf("，%d 个目录无法读取已跳过", skippedDirs)
	}

	report(ScanProgress{
		Phase:           "done",
		PhaseTitle:      finalTitle,
		TotalDiscovered: totalDiscovered,
		TotalToAnalyze:  totalToAnalyze,
		CurrentAnalyzed: len(analyzedPhotos),
		SkippedCount:    int(atomic.LoadInt64(&skippedCount)),
		SkippedDirs:     skippedDirs,
		FoundGroups:     len(groups),
		WastedBytes:     totalWasted,
		Percent:         1.0,
		IsComplete:      true,
	})

	return groups, nil
}
