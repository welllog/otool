package app

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"sync"
	"time"

	_ "github.com/SeriousBug/webp-go-pure/std"
	"github.com/welllog/otool/internal/services"
	"github.com/welllog/otool/internal/services/dedup"
	"github.com/welllog/otool/internal/storage"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const dedupGroupPageSize = 50

// dedupVisibleGroupCount returns how many result groups are currently expanded
// for review. Results are paginated so huge scans stay responsive; cleaning
// groups beyond this window requires the explicit authorization in the
// confirm dialog.
func (a *App) dedupVisibleGroupCount() int {
	n := len(a.dedupGroups)
	if a.dedupVisibleGroupLimit <= 0 || a.dedupVisibleGroupLimit >= n {
		return n
	}
	return a.dedupVisibleGroupLimit
}

// dedupSelectionStats counts selected photos and their bytes within the first
// groupCount groups. groupCount is clamped to the slice length.
func dedupSelectionStats(groups []*dedup.DuplicateGroup, groupCount int) (int, int64) {
	if groupCount > len(groups) {
		groupCount = len(groups)
	}
	if groupCount < 0 {
		groupCount = 0
	}
	var count int
	var bytes int64
	for _, g := range groups[:groupCount] {
		for _, p := range g.Photos {
			if p.Selected {
				count++
				bytes += p.Size
			}
		}
	}
	return count, bytes
}

func anyDedupSelection(groups []*dedup.DuplicateGroup) bool {
	for _, g := range groups {
		for _, p := range g.Photos {
			if p.Selected {
				return true
			}
		}
	}
	return false
}

func defaultDedupProfiles() *storage.ProfilesFile {
	return &storage.ProfilesFile{
		Version:    1,
		LastUsedID: "local_default",
		Profiles: []storage.Profile{
			{
				ID:        "local_default",
				Name:      "本地磁盘",
				Type:      storage.DriverLocal,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
	}
}

func (a *App) ensureDedupCacheMaintenance() {
	if a.dedupCacheMaintenance {
		return
	}
	a.dedupCacheMaintenance = true
	// Called on the UI thread; set the flag synchronously so a scan started in
	// the same event cannot slip in before the queued update runs.
	a.dedupCacheBusy = true
	go func() {
		_, _ = storage.GetCacheManager().PruneCache(storage.DefaultCacheMaxBytes, storage.DefaultCacheMaxAge)
		a.update(func() {
			a.dedupCacheBusy = false
			a.refreshDedupCacheStats()
		})
	}()
}

func (a *App) ensureDedupProfilesLoaded() {
	if a.dedupProfiles != nil && len(a.dedupProfiles.Profiles) > 0 {
		a.ensureDedupCacheMaintenance()
		return
	}
	pf, err := storage.LoadProfiles()
	if err != nil {
		// Do not silently replace a corrupt/unreadable config on disk. Keep an
		// in-memory fallback so the page remains usable, but disable saving
		// until the user fixes or retries loading the original file.
		a.dedupProfilesLoadErr = err.Error()
		pf = defaultDedupProfiles()
	} else if pf == nil || len(pf.Profiles) == 0 {
		pf = defaultDedupProfiles()
	}
	a.dedupProfiles = pf

	// Match last used
	for i, p := range pf.Profiles {
		if p.ID == pf.LastUsedID {
			a.dedupActiveProfileIdx = i
			break
		}
	}
	if a.dedupActiveProfileIdx < 0 || a.dedupActiveProfileIdx >= len(pf.Profiles) {
		a.dedupActiveProfileIdx = 0
	}

	if a.dedupActiveProfileIdx >= 0 && a.dedupActiveProfileIdx < len(pf.Profiles) {
		active := pf.Profiles[a.dedupActiveProfileIdx]
		if active.Type == storage.DriverLocal || active.Type == "" {
			if active.Local.DefaultPath != "" {
				a.dedupScanOpts.TargetDir = active.Local.DefaultPath
			}
		}
	}

	if a.dedupScanOpts.TargetDir == "" {
		a.dedupScanOpts.TargetDir = defaultUserPath()
	}

	a.ensureDedupCacheMaintenance()
}

func (a *App) saveDedupProfiles() error {
	if a.dedupProfiles == nil {
		return fmt.Errorf("配置尚未加载")
	}
	if a.dedupProfilesLoadErr != "" {
		err := fmt.Errorf("配置文件读取失败，已禁止保存以避免覆盖原配置")
		a.Toast(err.Error())
		return err
	}
	if err := storage.SaveProfiles(a.dedupProfiles); err != nil {
		a.Toast("保存配置失败: " + err.Error())
		return err
	}
	return nil
}

func (a *App) retryLoadDedupProfiles() {
	// Retrying replaces the profile/target configuration. Drop any result set
	// and pinned scan store so the UI cannot show new profiles beside old
	// results and delete through an obsolete storage session.
	a.dedupProfiles = nil
	a.dedupProfilesLoadErr = ""
	a.dedupActiveProfileIdx = 0
	a.dedupScanOpts.TargetDir = ""
	a.dedupGroups = nil
	a.dedupVisibleGroupLimit = dedupGroupPageSize
	a.dedupScanStore = nil
	a.dedupScanProfileID = ""
	a.dedupScanReadOnly = false
	a.dedupSelectedDeleteCount = 0
	a.dedupSelectedDeleteBytes = 0
	a.resetDedupThumbCache()
	a.ensureDedupProfilesLoaded()
}

func (a *App) getActiveDedupStorage() (storage.Storage, string, error) {
	a.ensureDedupProfilesLoaded()

	if len(a.dedupProfiles.Profiles) == 0 {
		return nil, "", fmt.Errorf("没有可用的存储配置")
	}

	if a.dedupActiveProfileIdx < 0 || a.dedupActiveProfileIdx >= len(a.dedupProfiles.Profiles) {
		a.dedupActiveProfileIdx = 0
	}

	p := a.dedupProfiles.Profiles[a.dedupActiveProfileIdx]

	switch p.Type {
	case storage.DriverWebDAV:
		pass, err := storage.DecryptSecret(p.WebDAV.Password)
		if err != nil {
			return nil, "", fmt.Errorf("解密 WebDAV 密码失败: %w", err)
		}
		store := storage.NewWebDAVStorage(p.ID, p.WebDAV.URL, p.WebDAV.Username, pass, p.WebDAV.RootPath)
		return store, p.WebDAV.RootPath, nil
	case storage.DriverLocal, "":
		store := storage.NewLocalStorage(p.ID)
		target := p.Local.DefaultPath
		if target == "" {
			target = defaultUserPath()
		}
		return store, target, nil
	default:
		return nil, "", fmt.Errorf("暂不支持的存储类型: %s", p.Type)
	}
}

// scheduleDedupCacheStatsRefresh coalesces cache-stat scans while thumbnail
// workers are writing files. The timer is UI-thread owned.
func (a *App) scheduleDedupCacheStatsRefresh() {
	if a.dedupCacheStatsRefreshTimer != nil {
		return
	}
	delay := time.Second - time.Since(a.dedupCacheStatsLastRefresh)
	if delay < 0 {
		delay = 0
	}
	a.dedupCacheStatsRefreshTimer = time.AfterFunc(delay, func() {
		a.update(func() {
			a.dedupCacheStatsRefreshTimer = nil
			a.refreshDedupCacheStats()
		})
	})
}

func (a *App) refreshDedupCacheStats() {
	if a.dedupCacheStatsLoading {
		a.dedupCacheStatsDirty = true
		a.dedupCacheStatsLastRefresh = time.Now()
		a.scheduleDedupCacheStatsRefresh()
		return
	}
	a.dedupCacheStatsLoading = true
	a.dedupCacheStatsLastRefresh = time.Now()
	a.dedupCacheStatsRequest++
	requestID := a.dedupCacheStatsRequest
	go func() {
		stats, err := storage.GetCacheManager().RefreshCacheStats()
		a.update(func() {
			if requestID != a.dedupCacheStatsRequest {
				return
			}
			a.dedupCacheStatsLoading = false
			if err == nil {
				a.dedupCacheStatsBytes = stats.TotalBytes
				a.dedupCacheStatsCount = stats.ThumbnailCount
				a.dedupCacheStatsFpCount = stats.FingerprintCount
			}
			// Mark loaded even on failure so a broken cache directory does not
			// trigger a new directory scan on every rendered frame.
			a.dedupCacheStatsLoaded = true
			if a.dedupCacheStatsDirty {
				a.dedupCacheStatsDirty = false
				a.refreshDedupCacheStats()
			}
		})
	}()
}

func (a *App) pruneDedupCache() {
	if a.dedupCacheBusy || a.dedupIsScanning || a.dedupIsDeleting {
		return
	}
	a.dedupCacheBusy = true
	go func() {
		result, err := storage.GetCacheManager().PruneCache(storage.DefaultCacheMaxBytes, storage.DefaultCacheMaxAge)
		a.update(func() {
			a.dedupCacheBusy = false
		})
		if err != nil {
			a.Toast("清理缓存失败: " + err.Error())
			a.update(a.refreshDedupCacheStats)
			return
		}
		if result.RemovedThumbnails == 0 && result.RemovedFingerprints == 0 {
			a.Toast("缓存未超过限制，无需清理")
			return
		}
		a.Toast(fmt.Sprintf("已清理 %d 张过期/超额缩略图、%d 条指纹缓存，释放 %s",
			result.RemovedThumbnails, result.RemovedFingerprints, services.FormatFileSize(result.FreedBytes)))
		a.update(a.refreshDedupCacheStats)
	}()
}

func (a *App) requestClearDedupCache() {
	if a.dedupCacheBusy || a.dedupIsScanning || a.dedupIsDeleting || a.dedupCacheStatsBytes == 0 {
		return
	}
	a.dedupShowCacheClearModal = true
}

func (a *App) executeClearDedupCache() {
	if a.dedupCacheBusy || a.dedupIsScanning || a.dedupIsDeleting {
		return
	}
	a.dedupCacheBusy = true
	// Invalidate any in-flight stats scan so it cannot overwrite the zeroed
	// snapshot after ClearCache completes.
	a.dedupCacheStatsRequest++
	a.dedupCacheStatsLoading = false
	// Cancel in-flight thumbnail loaders before deleting the cache so a late
	// completion cannot write a fresh file back after ClearCache.
	a.resetDedupThumbCache()
	oldBytes := a.dedupCacheStatsBytes
	oldCount := a.dedupCacheStatsCount
	go func() {
		err := storage.GetCacheManager().ClearCache()
		if err != nil {
			a.update(func() {
				a.dedupCacheBusy = false
			})
			a.Toast("清空缓存失败: " + err.Error())
			a.update(a.refreshDedupCacheStats)
			return
		}
		a.update(func() {
			a.dedupCacheBusy = false
			a.dedupShowCacheClearModal = false
			a.dedupCacheStatsBytes = 0
			a.dedupCacheStatsCount = 0
			a.dedupCacheStatsFpCount = 0
			a.dedupCacheStatsLoaded = true
			a.dedupCacheStatsDirty = false
			a.dedupCacheStatsLastRefresh = time.Now()
		})
		a.Toast(fmt.Sprintf("已清空本地缓存：%d 张缩略图，释放 %s", oldCount, services.FormatFileSize(oldBytes)))
	}()
}

func (a *App) resetDedupThumbCache() {
	a.dedupThumbMu.Lock()
	defer a.dedupThumbMu.Unlock()

	a.dedupThumbs = make(map[string]*ui.Bitmap)
	a.dedupThumbOrder = nil
	a.dedupThumbPending = make(map[string]bool)
	a.dedupThumbMissing = make(map[string]bool)
	a.dedupThumbGeneration++
	if a.dedupThumbCancel != nil {
		close(a.dedupThumbCancel)
	}
	a.dedupThumbCancel = make(chan struct{})
}

func (a *App) dropDedupThumbs(keys []string) {
	if len(keys) == 0 {
		return
	}
	a.dedupThumbMu.Lock()
	defer a.dedupThumbMu.Unlock()

	for _, key := range keys {
		delete(a.dedupThumbs, key)
		delete(a.dedupThumbPending, key)
		// A loader may already be decoding one of these keys. Mark it as gone
		// so a late completion is discarded instead of resurrecting the bitmap.
		a.dedupThumbMissing[key] = true
	}
	if len(a.dedupThumbOrder) == 0 {
		return
	}
	filtered := a.dedupThumbOrder[:0]
	for _, key := range a.dedupThumbOrder {
		if _, ok := a.dedupThumbs[key]; ok {
			filtered = append(filtered, key)
		}
	}
	a.dedupThumbOrder = filtered
}

const (
	// Keep enough decoded thumbnails to cover a large result set. The old
	// limit of 128 caused the full-page view to evict and reload thumbnails
	// on nearly every frame once more than 128 photos were shown. 512 decoded
	// 200px thumbnails is roughly 50 MB of RGBA pixels.
	dedupThumbCacheLimit  = 512
	dedupThumbLoadWorkers = 3
	dedupThumbQueueSize   = 512
)

type dedupThumbRequest struct {
	key        string
	path       string
	name       string
	store      storage.Storage
	generation uint64
	cancel     <-chan struct{}
}

func (a *App) ensureDedupThumbLoaders() {
	a.dedupThumbWorkersOnce.Do(func() {
		a.dedupThumbRequests = make(chan dedupThumbRequest, dedupThumbQueueSize)
		for i := 0; i < dedupThumbLoadWorkers; i++ {
			go a.dedupThumbLoader()
		}
	})
}

func (a *App) dedupThumbLoader() {
	for req := range a.dedupThumbRequests {
		select {
		case <-req.cancel:
			// The result set changed; don't waste IO decoding stale thumbs.
			a.update(func() {
				a.dedupThumbMu.Lock()
				if req.generation == a.dedupThumbGeneration {
					delete(a.dedupThumbPending, req.key)
				}
				a.dedupThumbMu.Unlock()
			})
			continue
		default:
		}

		img, ok, wroteCache := a.loadDedupThumb(req)
		a.update(func() {
			a.dedupThumbMu.Lock()
			defer a.dedupThumbMu.Unlock()
			if req.generation != a.dedupThumbGeneration {
				return
			}
			delete(a.dedupThumbPending, req.key)
			if _, gone := a.dedupThumbMissing[req.key]; gone {
				return
			}
			if !ok || img == nil {
				a.dedupThumbMissing[req.key] = true
				return
			}
			a.storeDedupThumbLocked(req.key, ui.NewBitmap(img))
			if wroteCache {
				a.dedupCacheStatsLoaded = false
				a.dedupCacheStatsDirty = true
				a.scheduleDedupCacheStatsRefresh()
			}
		})
	}
}

func dedupThumbContext(req dedupThumbRequest) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if req.cancel == nil {
		return ctx, cancel
	}
	stop := make(chan struct{})
	go func() {
		select {
		case <-req.cancel:
			cancel()
		case <-stop:
		}
	}()
	return ctx, func() {
		close(stop)
		cancel()
	}
}

func (a *App) dedupThumbStillWanted(req dedupThumbRequest) bool {
	a.dedupThumbMu.Lock()
	defer a.dedupThumbMu.Unlock()
	if req.generation != a.dedupThumbGeneration {
		return false
	}
	_, gone := a.dedupThumbMissing[req.key]
	return !gone
}

func (a *App) loadDedupThumb(req dedupThumbRequest) (image.Image, bool, bool) {
	if !a.dedupThumbStillWanted(req) {
		return nil, false, false
	}
	cache := storage.GetCacheManager()
	if raw, ok := cache.GetThumbnail(req.key); ok && len(raw) > 0 {
		if img, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
			return img, true, false
		}
	}

	// The fingerprint may have survived a thumbnail prune. Regenerate the
	// preview from the original file on demand instead of showing nothing.
	if req.store == nil || req.path == "" {
		return nil, false, false
	}
	ctx, cancel := dedupThumbContext(req)
	defer cancel()
	raw, err := dedup.GenerateThumbnailFromStore(ctx, req.store, req.path, req.name)
	if err != nil || len(raw) == 0 {
		return nil, false, false
	}
	if !a.dedupThumbStillWanted(req) {
		return nil, false, false
	}
	wroteCache := cache.SaveThumbnail(req.key, raw) == nil
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, false, false
	}
	return img, true, wroteCache
}

func (a *App) touchDedupThumbLocked(cacheKey string) {
	for i, key := range a.dedupThumbOrder {
		if key != cacheKey {
			continue
		}
		copy(a.dedupThumbOrder[i:], a.dedupThumbOrder[i+1:])
		a.dedupThumbOrder[len(a.dedupThumbOrder)-1] = cacheKey
		return
	}
}

func (a *App) storeDedupThumbLocked(cacheKey string, bmp *ui.Bitmap) {
	if bmp == nil {
		return
	}
	if _, ok := a.dedupThumbs[cacheKey]; ok {
		a.touchDedupThumbLocked(cacheKey)
		return
	}
	a.dedupThumbs[cacheKey] = bmp
	a.dedupThumbOrder = append(a.dedupThumbOrder, cacheKey)

	// Bounded true LRU: move cache hits to the end and evict only the oldest.
	for len(a.dedupThumbOrder) > dedupThumbCacheLimit {
		oldest := a.dedupThumbOrder[0]
		a.dedupThumbOrder = a.dedupThumbOrder[1:]
		delete(a.dedupThumbs, oldest)
	}
}

// getDedupThumbBitmap returns a decoded bitmap for the current frame, or nil
// while the asynchronous loader is fetching it. It never performs disk IO or
// image decoding on the UI thread.
func (a *App) getDedupThumbBitmap(photo *dedup.PhotoMeta) (*ui.Bitmap, bool) {
	if photo == nil || photo.CacheKey == "" {
		return nil, false
	}
	cacheKey := photo.CacheKey

	a.dedupThumbMu.Lock()
	defer a.dedupThumbMu.Unlock()

	if bmp, ok := a.dedupThumbs[cacheKey]; ok && bmp != nil {
		a.touchDedupThumbLocked(cacheKey)
		return bmp, false
	}
	if a.dedupThumbMissing[cacheKey] {
		return nil, false
	}
	if a.dedupThumbPending[cacheKey] {
		return nil, true
	}

	a.ensureDedupThumbLoaders()
	req := dedupThumbRequest{
		key:        cacheKey,
		path:       photo.Path,
		name:       photo.Name,
		store:      a.dedupScanStore,
		generation: a.dedupThumbGeneration,
		cancel:     a.dedupThumbCancel,
	}
	select {
	case a.dedupThumbRequests <- req:
		a.dedupThumbPending[cacheKey] = true
	default:
		// Queue is full: leave the key unqueued so a later frame retries.
	}
	return nil, true
}

func (a *App) recalculateDedupStats() {
	var count int
	var bytesTotal int64

	for _, g := range a.dedupGroups {
		var groupSelected int64
		for _, p := range g.Photos {
			if p.Selected {
				count++
				groupSelected += p.Size
				bytesTotal += p.Size
			}
		}
		g.TotalWastedBytes = groupSelected
	}

	a.dedupSelectedDeleteCount = count
	a.dedupSelectedDeleteBytes = bytesTotal
}

func (a *App) applyRecommendedKeepAll() {
	for _, g := range a.dedupGroups {
		for _, p := range g.Photos {
			if p.ID == g.BestPickID {
				p.Selected = false // Keep
			} else {
				p.Selected = true // Delete
			}
		}
	}
	a.recalculateDedupStats()
	a.Toast("已为所有重复组应用最佳推荐 (保留最高画质)")
}

func cloneDedupGroupsForDeletion(groups []*dedup.DuplicateGroup) []*dedup.DuplicateGroup {
	result := make([]*dedup.DuplicateGroup, 0, len(groups))
	for _, g := range groups {
		photos := make([]*dedup.PhotoMeta, 0, len(g.Photos))
		for _, p := range g.Photos {
			clone := *p
			photos = append(photos, &clone)
		}
		groupClone := *g
		groupClone.Photos = photos
		result = append(result, &groupClone)
	}
	return result
}

// dedupWebdavPasswordSaveError guards against silently overwriting a stored
// WebDAV password with an empty one. The edit form is prefilled with the
// decrypted password, so an empty field while editing a profile that has a
// stored password means the user cleared it or decryption failed and the
// password was never re-entered. New profiles and originally password-less
// profiles may legitimately save an empty password (anonymous WebDAV).
func dedupWebdavPasswordSaveError(editingIdx int, profiles []storage.Profile, typed string) string {
	if editingIdx < 0 || editingIdx >= len(profiles) {
		return ""
	}
	if typed != "" {
		return ""
	}
	if profiles[editingIdx].WebDAV.Password != "" {
		return "密码不能为空：请保留已填入的原密码，或重新输入 WebDAV 密码后再保存，以免覆盖现有连接"
	}
	return ""
}

func dedupFileStillMatches(ctx context.Context, store storage.Storage, p *dedup.PhotoMeta) bool {
	fi, err := store.Stat(ctx, p.Path)
	if err != nil || fi.IsDir {
		return false
	}
	if p.Size <= 0 || fi.Size != p.Size {
		return false
	}
	if p.ETag != "" && fi.ETag != "" && p.ETag != fi.ETag {
		return false
	}
	if !p.ModTime.IsZero() {
		if fi.ModTime.IsZero() || !p.ModTime.Equal(fi.ModTime) {
			return false
		}
	}
	// Never delete a photo when the scan has no full-content hash to prove its
	// identity. Similar-mode scans fill these hashes after clustering, so this
	// normally only rejects stale/corrupt UI state rather than valid photos.
	if p.ContentHash == 0 {
		return false
	}
	// Re-hash before deleting so a same-size/same-mtime replacement cannot be
	// removed. This applies to both exact and similar groups.
	rc, err := store.Open(ctx, p.Path)
	if err != nil {
		return false
	}
	hash, err := dedup.ComputeFullContentHashContext(ctx, rc)
	_ = rc.Close()
	if err != nil || hash != p.ContentHash {
		return false
	}
	return true
}

func (a *App) startDedupScan() {
	if a.dedupIsScanning {
		return
	}
	if a.dedupIsDeleting {
		a.Toast("正在清理照片，请等待清理完成后再扫描")
		return
	}
	if a.dedupCacheBusy {
		a.Toast("正在处理缓存，请稍后再开始扫描")
		return
	}
	if a.dedupProfilesLoadErr != "" {
		a.Toast("配置文件读取失败，请先修复配置或重试读取后再扫描")
		return
	}
	store, targetDir, err := a.getActiveDedupStorage()
	if err != nil {
		a.Toast("初始化存储失败: " + err.Error())
		return
	}
	if a.dedupCacheBusy {
		a.Toast("正在维护缓存，请稍后再开始扫描")
		return
	}

	profileID := ""
	if a.dedupActiveProfileIdx >= 0 && a.dedupActiveProfileIdx < len(a.dedupProfiles.Profiles) {
		profileID = a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].ID
	}

	a.dedupScanOpts.TargetDir = targetDir
	if a.dedupSelectedModeIdx == 0 {
		a.dedupScanOpts.Mode = dedup.ModeSimilar
		switch a.dedupSelectedDistanceIdx {
		case 0:
			a.dedupScanOpts.SimilarDistance = 5 // 严格
		case 1:
			a.dedupScanOpts.SimilarDistance = 8 // 标准
		case 2:
			a.dedupScanOpts.SimilarDistance = 12 // 宽松连拍
		}
	} else {
		a.dedupScanOpts.Mode = dedup.ModeExact
	}
	scanOpts := a.dedupScanOpts // snapshot: UI settings may change during the scan

	ctx, cancel := context.WithCancel(context.Background())
	a.dedupScanCancel = cancel
	a.dedupIsScanning = true
	a.dedupProgress = dedup.ScanProgress{Phase: "starting", PhaseTitle: "准备扫描..."}
	a.dedupGroups = nil
	a.dedupVisibleGroupLimit = dedupGroupPageSize
	a.dedupScanStore = store
	a.dedupScanProfileID = profileID
	a.dedupScanReadOnly = scanOpts.ReadOnly
	a.resetDedupThumbCache()

	var progressMu sync.Mutex
	var lastProgress dedup.ScanProgress
	lastProgressAt := time.Now()

	reportProgress := func(p dedup.ScanProgress) {
		progressMu.Lock()
		lastProgress = p
		now := time.Now()
		// Engine workers can report every few files. Throttle main-loop posts;
		// the final progress is committed together with the result groups.
		if now.Sub(lastProgressAt) < 100*time.Millisecond {
			progressMu.Unlock()
			return
		}
		lastProgressAt = now
		progressMu.Unlock()
		a.update(func() {
			a.dedupProgress = p
		})
	}

	go func() {
		defer func() {
			a.update(func() {
				a.dedupIsScanning = false
			})
		}()

		groups, err := a.dedupEngine.RunScan(ctx, store, scanOpts, reportProgress)
		if err != nil {
			if ctx.Err() != nil {
				a.Toast("已停止扫描")
			} else {
				a.Toast("扫描失败: " + err.Error())
			}
			return
		}
		if ctx.Err() != nil {
			a.Toast("已停止扫描")
			return
		}

		progressMu.Lock()
		finalProgress := lastProgress
		skipped := lastProgress.SkippedCount
		skippedDirs := lastProgress.SkippedDirs
		progressMu.Unlock()

		a.update(func() {
			a.dedupProgress = finalProgress
			a.dedupGroups = groups
			a.dedupVisibleGroupLimit = dedupGroupPageSize
			a.resetDedupThumbCache()
			a.recalculateDedupStats()
			a.refreshDedupCacheStats()
			// Publish the busy state together with the result frame so the
			// delete button cannot act before post-scan cache maintenance starts.
			a.dedupCacheBusy = true
		})

		// A scan can write many thumbnails in one pass. Enforce the cache
		// ceiling in the background as soon as the scan finishes instead of
		// waiting for the next app launch.
		go func() {
			_, _ = storage.GetCacheManager().PruneCache(storage.DefaultCacheMaxBytes, storage.DefaultCacheMaxAge)
			a.update(func() {
				a.dedupCacheBusy = false
				a.refreshDedupCacheStats()
			})
		}()

		dirNote := ""
		if skippedDirs > 0 {
			dirNote = fmt.Sprintf("，%d 个目录无法读取已跳过", skippedDirs)
		}
		// Exact-mode groups whose representative could not be decoded keep
		// their photos in the results; tell the user previews are missing
		// instead of miscounting those photos as skipped.
		previewNote := ""
		if finalProgress.PreviewFailedGroups > 0 {
			previewNote = fmt.Sprintf("，%d 组未能生成预览信息（照片仍保留在结果中）", finalProgress.PreviewFailedGroups)
		}
		switch {
		case len(groups) == 0 && skipped > 0:
			a.Toast(fmt.Sprintf("扫描完成，未发现重复或相似照片（%d 张图片无法读取或解码，已跳过%s）", skipped, dirNote))
		case len(groups) == 0:
			a.Toast("扫描完成，未发现重复或相似照片" + dirNote)
		case skipped > 0:
			a.Toast(fmt.Sprintf("扫描完成，发现 %d 组重复照片（%d 张图片无法读取或解码，已跳过%s%s）", len(groups), skipped, dirNote, previewNote))
		default:
			a.Toast(fmt.Sprintf("扫描完成，发现 %d 组重复照片%s%s", len(groups), dirNote, previewNote))
		}
	}()
}

func (a *App) cancelDedupScan() {
	if a.dedupScanCancel != nil {
		a.dedupScanCancel()
	}
}

// executeDedupDeletion deletes the currently selected photos. cleanAll picks
// the scope: when false, only groups already expanded for review are touched
// and the remaining groups keep their selections for a later batch; when true,
// every group — including ones never expanded — is cleaned as authorized by
// the confirm dialog.
func (a *App) executeDedupDeletion(cleanAll bool) {
	if a.dedupIsDeleting || a.dedupIsScanning || a.dedupCacheBusy || len(a.dedupGroups) == 0 {
		return
	}

	// Always use the store that produced the scan results. The user is allowed
	// to switch profiles in the picker, but cleanup must never be redirected to
	// a different server/local directory that merely shares the same paths.
	store := a.dedupScanStore
	if store == nil {
		a.Toast("清理会话已失效，请重新扫描后再清理")
		return
	}
	if a.dedupScanReadOnly || a.dedupScanOpts.ReadOnly {
		a.Toast("当前扫描为只读模式，无法执行清理")
		return
	}

	// Freeze the selection before starting the worker. The confirmation modal
	// may be dismissed while the worker is running, so it must never read the
	// live checkboxes again.
	allGroups := cloneDedupGroupsForDeletion(a.dedupGroups)
	visible := a.dedupVisibleGroupCount()
	if cleanAll || visible > len(allGroups) {
		visible = len(allGroups)
	}
	// Groups past the expanded review window keep their selections for a later
	// batch; they must survive the result rebuild untouched.
	deferred := allGroups[visible:]
	groups := allGroups[:visible]
	if !anyDedupSelection(groups) {
		if cleanAll {
			a.Toast("没有勾选任何待清理照片")
		} else {
			a.Toast("已展开审阅的分组中没有勾选待清理照片")
		}
		return
	}
	for _, g := range groups {
		if len(g.Photos) == 0 {
			continue
		}
		selected := 0
		for _, p := range g.Photos {
			if !p.Selected {
				continue
			}
			selected++
			if a.dedupScanProfileID != "" && p.StorageID != "" && p.StorageID != a.dedupScanProfileID {
				a.Toast("扫描结果中的存储源与清理会话不一致，已取消清理")
				return
			}
		}
		if selected == len(g.Photos) {
			a.Toast("每组至少需要保留 1 张照片，请取消勾选每组中的一张后重试")
			return
		}
	}

	trashOnly := a.dedupConfirmTrashOnly
	a.dedupIsDeleting = true
	ctx := context.Background()

	go func() {
		defer func() {
			a.update(func() {
				a.dedupIsDeleting = false
				a.dedupShowConfirmModal = false
			})
		}()

		var deletedCount, failedCount int
		var freedBytes int64
		var deletedCacheKeys []string
		cacheMgr := storage.GetCacheManager()

		// Build new groups from immutable snapshots instead of mutating the
		// slices currently being rendered by the UI thread.
		newGroups := make([]*dedup.DuplicateGroup, 0, len(groups))
		for _, g := range groups {
			remainingPhotos := make([]*dedup.PhotoMeta, 0, len(g.Photos))
			for _, p := range g.Photos {
				if p.Selected {
					// Never delete based on a stale path: use a bounded per-file
					// context so a hung remote Stat/Open cannot freeze cleanup.
					opCtx, cancelOp := context.WithTimeout(ctx, 60*time.Second)
					matches := dedupFileStillMatches(opCtx, store, p)
					if !matches {
						cancelOp()
						failedCount++
						clone := *p
						remainingPhotos = append(remainingPhotos, &clone)
						continue
					}
					var delErr error
					if trashOnly {
						delErr = store.MoveToTrash(opCtx, p.Path)
					} else {
						delErr = store.Delete(opCtx, p.Path)
					}
					cancelOp()

					if delErr == nil {
						deletedCount++
						freedBytes += p.Size
						if p.CacheKey != "" {
							_ = cacheMgr.RemoveThumbnail(p.CacheKey)
							deletedCacheKeys = append(deletedCacheKeys, p.CacheKey)
						}
						continue
					}
					failedCount++
				}

				clone := *p
				remainingPhotos = append(remainingPhotos, &clone)
			}

			if len(remainingPhotos) >= 2 {
				ng := &dedup.DuplicateGroup{
					GroupID: g.GroupID,
					Mode:    g.Mode,
					Photos:  remainingPhotos,
				}
				// Preserve the user's explicit keep/delete choices after a
				// partial cleanup; only the scores and BestPick are refreshed.
				dedup.EvaluateAndScoreGroupPreserveSelection(ng)
				newGroups = append(newGroups, ng)
			}
		}
		// Groups outside the cleaned scope were never reviewed in this round;
		// reattach them unchanged so their selections survive for later.
		newGroups = append(newGroups, deferred...)

		// RemoveThumbnail marks the fingerprint map dirty; flush once after the
		// whole batch instead of writing fingerprints.json per deleted photo.
		_ = cacheMgr.FlushFingerprints()

		a.update(func() {
			a.dedupGroups = newGroups
			a.dropDedupThumbs(deletedCacheKeys)
			a.recalculateDedupStats()
			a.refreshDedupCacheStats()
		})

		actionName := "移入回收站/隔离区"
		if !trashOnly {
			actionName = "彻底删除"
		}

		msg := fmt.Sprintf("清理完成！已%s %d 张废片，释放 %s 空间", actionName, deletedCount, services.FormatFileSize(freedBytes))
		if failedCount > 0 {
			msg += fmt.Sprintf("；%d 张删除失败，已保留在列表中", failedCount)
		}
		a.Toast(msg)
	}()
}

func (a *App) viewDedup(c *ui.Context) {
	t := c.Theme()
	a.ensureDedupProfilesLoaded()
	if a.dedupActiveProfileIdx < 0 || a.dedupActiveProfileIdx >= len(a.dedupProfiles.Profiles) {
		a.dedupActiveProfileIdx = 0
	}

	ui.Column(c).Gap(16).Children(func() {
		// 1. Storage Source and Target Card
		Card(c, "存储源与扫描目标", IconFolder, func() {
			if a.dedupProfilesLoadErr != "" {
				ui.Row(c).
					WidthPercent(100).
					Padding(8, 10).
					Radius(8).
					Background(ui.Hex("#fef2f2")).
					Justify(ui.SpaceBetween).
					AlignItems(ui.Center).
					Children(func() {
						ui.Text(c, "⚠️ 配置文件读取失败，当前使用临时默认配置（不会覆盖原文件）: "+a.dedupProfilesLoadErr).
							FontSize(11).
							TextColor(ui.Hex("#b91c1c")).
							MaxLines(2).
							Grow(1)
						if ui.Button(c, "重试读取").Disabled(a.dedupIsScanning || a.dedupIsDeleting).Clicked() {
							a.retryLoadDedupProfiles()
						}
					})
			}
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "➕ 新建网盘连接").Disabled(a.dedupIsScanning || a.dedupIsDeleting || len(a.dedupGroups) > 0 || a.dedupProfilesLoadErr != "").Clicked() {
					a.dedupEditingIdx = -1
					a.dedupEditName = "我的 WebDAV"
					a.dedupEditType = storage.DriverWebDAV
					a.dedupEditWebdavURL = "https://"
					a.dedupEditWebdavUser = ""
					a.dedupEditWebdavPass = ""
					a.dedupEditWebdavRoot = "/"
					a.dedupEditTestMsg = ""
					a.dedupShowConfigModal = true
				}
				if a.dedupActiveProfileIdx >= 0 &&
					a.dedupActiveProfileIdx < len(a.dedupProfiles.Profiles) &&
					a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].Type == storage.DriverWebDAV {
					if ui.Button(c, "✏️ 编辑当前配置").Disabled(a.dedupIsScanning || a.dedupIsDeleting || len(a.dedupGroups) > 0 || a.dedupProfilesLoadErr != "").Clicked() {
						p := a.dedupProfiles.Profiles[a.dedupActiveProfileIdx]
						a.dedupEditingIdx = a.dedupActiveProfileIdx
						a.dedupEditName = p.Name
						a.dedupEditType = p.Type
						a.dedupEditWebdavURL = p.WebDAV.URL
						a.dedupEditWebdavUser = p.WebDAV.Username
						a.dedupEditWebdavRoot = p.WebDAV.RootPath
						a.dedupEditTestMsg = ""
						if pass, err := storage.DecryptSecret(p.WebDAV.Password); err != nil {
							a.dedupEditWebdavPass = ""
							a.dedupEditTestMsg = "⚠️ 原密码解密失败，请重新输入 WebDAV 密码"
						} else {
							a.dedupEditWebdavPass = pass
						}
						a.dedupShowConfigModal = true
					}
				}
				if ui.Button(c, "📂 打开配置所在目录").Clicked() {
					_ = storage.OpenConfigFolder()
					c.Toast("已在系统文件管理器中打开配置目录")
				}
			})
		}, func() {
			ui.Column(c).Gap(12).Children(func() {
				// Profile selector
				profileNames := make([]string, len(a.dedupProfiles.Profiles))
				nameCounts := make(map[string]int)
				rawNames := make([]string, len(a.dedupProfiles.Profiles))
				for i, p := range a.dedupProfiles.Profiles {
					typeName := "本地"
					if p.Type == storage.DriverWebDAV {
						typeName = "WebDAV"
					}
					rawNames[i] = fmt.Sprintf("%s (%s)", p.Name, typeName)
					nameCounts[rawNames[i]]++
				}
				for i, name := range rawNames {
					// Make duplicate display names selectable; otherwise both
					// entries map to the first profile with that name.
					if nameCounts[name] > 1 {
						profileNames[i] = fmt.Sprintf("%s #%d", name, i+1)
					} else {
						profileNames[i] = name
					}
				}

				selectedName := profileNames[a.dedupActiveProfileIdx]
				profileLocked := a.dedupIsScanning || a.dedupIsDeleting || len(a.dedupGroups) > 0 || a.dedupProfilesLoadErr != ""
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "存储源:").FontSize(13).Bold().TextColor(t.Text)
					if ui.Select(c, &selectedName, profileNames).Disabled(profileLocked).Changed() {
						for i, name := range profileNames {
							if name == selectedName {
								a.dedupActiveProfileIdx = i
								a.dedupProfiles.LastUsedID = a.dedupProfiles.Profiles[i].ID
								// Keep the local target in sync with the selected
								// profile instead of reusing the previous profile's path.
								if a.dedupProfiles.Profiles[i].Type == storage.DriverLocal ||
									a.dedupProfiles.Profiles[i].Type == "" {
									if p := a.dedupProfiles.Profiles[i].Local.DefaultPath; p != "" {
										a.dedupScanOpts.TargetDir = p
									} else {
										a.dedupScanOpts.TargetDir = defaultUserPath()
									}
								}
								a.saveDedupProfiles()
								break
							}
						}
					}
				})

				// Target directory row
				if a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].Type == storage.DriverLocal ||
					a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].Type == "" {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "扫描目录:").FontSize(13).TextColor(t.TextMuted)
						if ui.TextInput(c, &a.dedupScanOpts.TargetDir).
							Placeholder("选择或输入待去重照片所在文件夹路径...").
							Grow(1).Disabled(a.dedupIsScanning || a.dedupIsDeleting || a.dedupCacheBusy || a.dedupProfilesLoadErr != "").Changed() {
							a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].Local.DefaultPath = a.dedupScanOpts.TargetDir
							a.saveDedupProfiles()
						}

						if ui.Button(c, "浏览文件夹").Disabled(a.dedupIsScanning || a.dedupIsDeleting || a.dedupCacheBusy || a.dedupProfilesLoadErr != "").Clicked() {
							go func() {
								paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
									Title:     "选择待去重照片所在的文件夹",
									Directory: true,
								})
								if err == nil && len(paths) > 0 {
									a.update(func() {
										a.dedupScanOpts.TargetDir = paths[0]
										a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].Local.DefaultPath = paths[0]
										a.saveDedupProfiles()
									})
								}
							}()
						}
					})
				} else {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "远程目录:").FontSize(13).TextColor(t.TextMuted)
						if ui.TextInput(c, &a.dedupProfiles.Profiles[a.dedupActiveProfileIdx].WebDAV.RootPath).
							Placeholder("如 /photo/2024").
							Grow(1).Disabled(a.dedupIsScanning || a.dedupIsDeleting || a.dedupCacheBusy || a.dedupProfilesLoadErr != "").Changed() {
							a.saveDedupProfiles()
						}

						if ui.Button(c, "⚡ 测试连接与删除权限").Disabled(a.dedupIsScanning || a.dedupIsDeleting || a.dedupCacheBusy).Clicked() {
							// Resolve the profile on the UI thread, then hand only
							// the immutable storage handle to the worker.
							store, rootPath, err := a.getActiveDedupStorage()
							if err != nil {
								a.Toast("创建客户端失败: " + err.Error())
							} else {
								go func() {
									ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
									defer cancel()
									w, d, err := store.CheckPermission(ctx, rootPath)
									if err != nil {
										a.Toast("权限测试未通过: " + err.Error())
									} else if !w || !d {
										a.Toast("权限测试未通过: 缺少写入或删除权限")
									} else {
										a.Toast("🟢 测试成功: 具备读写与删除权限")
									}
								}()
							}
						}
					})
				}

				// Transparent config file and cache display row
				ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Wrap().Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "📄 配置文件:").FontSize(11).TextColor(t.TextMuted)
						ui.Text(c, storage.GetConfigPath()).FontSize(11).Bold().TextColor(t.TextMuted)
					})

					// Cache stats and cleanup buttons. Stats refresh runs in the
					// background; the UI only reads the cached snapshot.
					if !a.dedupCacheStatsLoaded && !a.dedupCacheStatsLoading {
						// Debounce the full-directory stats scan while thumbnail
						// workers are writing cache files continuously.
						if a.dedupCacheStatsDirty {
							a.scheduleDedupCacheStatsRefresh()
						} else {
							a.refreshDedupCacheStats()
						}
					}
					cBytes, cCount, cFpCount := a.dedupCacheStatsBytes, a.dedupCacheStatsCount, a.dedupCacheStatsFpCount
					cacheLocked := a.dedupCacheBusy || a.dedupIsScanning || a.dedupIsDeleting
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("💾 本地缓存: %s · 缩略图 %d 张 · 指纹 %d 条", services.FormatFileSize(cBytes), cCount, cFpCount)).
							FontSize(11).TextColor(t.TextMuted)

						if ui.Button(c, "📂 查看").Clicked() {
							go func() {
								if err := storage.GetCacheManager().OpenCacheFolder(); err != nil {
									a.Toast("打开缓存目录失败: " + err.Error())
									return
								}
								a.Toast("已在文件管理器中打开缓存目录")
							}()
						}

						if ui.Button(c, "🧹 清理过期").Disabled(cacheLocked).Clicked() {
							a.pruneDedupCache()
						}

						if ui.Button(c, "🗑️ 清空缓存").Disabled(cacheLocked || cBytes == 0).Clicked() {
							a.requestClearDedupCache()
						}
					})
				})
			})
		})

		// 2. Scan Options & Actions Card
		Card(c, "去重模式与参数设置", IconDedup, nil, func() {
			ui.Column(c).Gap(14).Children(func() {
				// Mode selection
				ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "检测算法:").FontSize(13).Bold().TextColor(t.Text)
					ui.Segmented(c, &a.dedupSelectedModeIdx, "智能相似/连拍去重 (推荐)", "完全精确去重 (字节级)").Disabled(a.dedupIsScanning || a.dedupIsDeleting)
				})

				// Similar threshold
				if a.dedupSelectedModeIdx == 0 {
					ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "灵敏度:").FontSize(13).TextColor(t.TextMuted)
						ui.Segmented(c, &a.dedupSelectedDistanceIdx, "严格相似 (阈值 5)", "标准相似 (阈值 8)", "宽松连拍 (阈值 12)").Disabled(a.dedupIsScanning || a.dedupIsDeleting)
					})
				}

				// Checkboxes & Buttons
				ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Gap(16).AlignItems(ui.Center).Children(func() {
						ui.Checkbox(c, &a.dedupScanOpts.Recursive, "包含所有子目录 (深度递归扫描)").Disabled(a.dedupIsScanning || a.dedupIsDeleting)
						ui.Checkbox(c, &a.dedupScanOpts.ReadOnly, "只读扫描 (仅分析，不检查删除权限)").Disabled(a.dedupIsScanning || a.dedupIsDeleting)
					})

					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if !a.dedupIsScanning {
							btn := ui.Button(c, "开始极速扫描").
								Size(140, 36).
								Disabled(a.dedupIsDeleting || a.dedupCacheBusy)
							if btn.Clicked() {
								a.startDedupScan()
							}
						} else {
							if ui.Button(c, "停止扫描").Size(120, 36).Clicked() {
								a.cancelDedupScan()
							}
						}
					})
				})
			})
		})

		// 3. Scan Progress Section (Displayed when scanning)
		if a.dedupIsScanning {
			Card(c, "扫描进度", IconRefresh, nil, func() {
				ui.Column(c).Gap(8).Children(func() {
					ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
						ui.Text(c, a.dedupProgress.PhaseTitle).FontSize(13).Bold().TextColor(t.Accent)
						pctText := fmt.Sprintf("%.0f%%", a.dedupProgress.Percent*100)
						ui.Text(c, pctText).FontSize(13).Bold().TextColor(t.Accent)
					})

					// Progress Bar
					barWidth := float32(a.dedupProgress.Percent)
					if barWidth > 1.0 {
						barWidth = 1.0
					}
					ui.Box(c).
						Height(8).
						Radius(4).
						Background(t.Surface).
						Children(func() {
							ui.Box(c).
								FillHeight().
								WidthPercent(barWidth * 100).
								Radius(4).
								Background(t.Accent)
						})

					if a.dedupProgress.CurrentFile != "" {
						ui.Text(c, "正在处理: "+a.dedupProgress.CurrentFile).FontSize(11).TextColor(t.TextMuted).MaxLines(1)
					}
				})
			})
		}

		// 4. Results Review Section (Displayed when groups exist)
		if len(a.dedupGroups) > 0 {
			// Summary Actions Banner
			ui.Row(c).
				Padding(12, 16).
				Radius(10).
				Background(t.Surface).
				Border(1, t.Border).
				Justify(ui.SpaceBetween).
				AlignItems(ui.Center).
				Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Text(c, fmt.Sprintf("发现 %d 组重复照片", len(a.dedupGroups))).FontSize(14).Bold().TextColor(t.Text)
						ui.Text(c, "·").TextColor(t.TextMuted)
						ui.Text(c, fmt.Sprintf("已选 %d 张待清理", a.dedupSelectedDeleteCount)).FontSize(13).TextColor(ui.Hex("#dc2626"))
						ui.Text(c, "·").TextColor(t.TextMuted)
						ui.Text(c, fmt.Sprintf("预计释放 %s", services.FormatFileSize(a.dedupSelectedDeleteBytes))).FontSize(13).Bold().TextColor(ui.Hex("#16a34a"))
					})

					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if ui.Button(c, "👑 一键应用推荐保留").Disabled(a.dedupIsDeleting).Clicked() {
							a.applyRecommendedKeepAll()
						}
						if ui.Button(c, "🔄 反选").Disabled(a.dedupIsDeleting).Clicked() {
							for _, g := range a.dedupGroups {
								for _, p := range g.Photos {
									if p.ID == g.BestPickID {
										p.Selected = false // never let bulk invert select the recommended keeper
										continue
									}
									p.Selected = !p.Selected
								}
							}
							a.recalculateDedupStats()
							a.Toast("已反选（每组推荐保留项不会被勾选）")
						}
						delBtnText := fmt.Sprintf("🗑️ 开始清理 (%d 张)", a.dedupSelectedDeleteCount)
						if ui.Button(c, delBtnText).Disabled(a.dedupSelectedDeleteCount == 0 || a.dedupScanReadOnly || a.dedupScanOpts.ReadOnly || a.dedupIsDeleting || a.dedupIsScanning || a.dedupCacheBusy).Clicked() {
							if a.dedupScanReadOnly || a.dedupScanOpts.ReadOnly {
								a.Toast("当前扫描为只读模式，无法执行清理")
							} else {
								a.dedupConfirmTrashOnly = true
								a.dedupShowConfirmModal = true
							}
						}
					})
				})

			// Render Duplicate Groups in pages. Building every group and every
			// photo at once makes large result sets slow even with async previews.
			visibleGroupCount := a.dedupVisibleGroupCount()
			if visibleGroupCount < len(a.dedupGroups) {
				ui.Text(c, fmt.Sprintf("为保持界面流畅，当前展开前 %d 组（共 %d 组）；未展开的分组已按推荐预选，可在清理确认弹窗中选择分批清理或一并清理。",
					visibleGroupCount, len(a.dedupGroups))).
					FontSize(11).TextColor(t.TextMuted)
			}
			for groupIdx := 0; groupIdx < visibleGroupCount; groupIdx++ {
				grp := a.dedupGroups[groupIdx]
				gTitle := fmt.Sprintf("第 %d 组 (共 %d 张) · 相似度: %d%% · 可释放 %s",
					groupIdx+1, len(grp.Photos), grp.SimilarityPercent, services.FormatFileSize(grp.TotalWastedBytes))

				Card(c, gTitle, IconImage, nil, func() {
					ui.Row(c).Gap(12).Wrap().Children(func() {
						for _, p := range grp.Photos {
							photo := p
							isBest := photo.ID == grp.BestPickID

							ui.Column(c).
								Width(210).
								Padding(10).
								Radius(8).
								Background(t.Surface).
								Border(1, t.Border).
								Gap(8).
								Children(func() {
									// Thumbnail with on-demand async load and LRU limit
									bmp, thumbLoading := a.getDedupThumbBitmap(photo)
									if bmp != nil {
										ui.Image(c, bmp).Size(190, 130).Radius(6)
									} else {
										placeholder := "无预览图"
										if thumbLoading {
											placeholder = "加载中..."
										}
										ui.Box(c).Size(190, 130).Radius(6).Background(t.Border).Center().Children(func() {
											ui.Text(c, placeholder).FontSize(11).TextColor(t.TextMuted)
										})
									}

									// Badge
									if isBest {
										ui.Row(c).Padding(2, 6).Radius(4).Background(ui.Hex("#dcfce7")).Children(func() {
											ui.Text(c, "👑 推荐保留 (画质最佳)").FontSize(10).Bold().TextColor(ui.Hex("#16a34a"))
										})
									} else {
										ui.Row(c).Padding(2, 6).Radius(4).Background(ui.Hex("#fee2e2")).Children(func() {
											ui.Text(c, "❌ 待清理废片").FontSize(10).Bold().TextColor(ui.Hex("#dc2626"))
										})
									}

									// Checkbox to select for delete
									if ui.Checkbox(c, &photo.Selected, "标记为清理").Disabled(a.dedupIsDeleting).Changed() {
										a.recalculateDedupStats()
									}

									// Filename & Path
									ui.Column(c).Gap(2).Children(func() {
										ui.Text(c, photo.Name).FontSize(11).Bold().TextColor(t.Text).MaxLines(1)
										ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
											ui.Text(c, photo.Path).FontSize(9).TextColor(t.TextMuted).MaxLines(1).Grow(1)
											CopyButton(c, photo.Path)
										})
									})

									// Metadata Info
									ui.Column(c).Gap(2).Children(func() {
										ui.Text(c, fmt.Sprintf("分辨率: %dx%d", photo.Width, photo.Height)).FontSize(11).TextColor(t.Text)
										ui.Text(c, fmt.Sprintf("大小: %s", services.FormatFileSize(photo.Size))).FontSize(11).TextColor(t.TextMuted)
										ui.Text(c, fmt.Sprintf("清晰度: %.1f", photo.BlurScore)).FontSize(11).TextColor(t.TextMuted)
										if photo.CameraModel != "" {
											ui.Text(c, "相机: "+photo.CameraModel).FontSize(10).TextColor(t.TextMuted).MaxLines(1)
										}
										for _, reason := range photo.ScoreReasons {
											ui.Text(c, "• "+reason).FontSize(10).TextColor(t.Accent).MaxLines(1)
										}
										// Diagnostic hashes
										ui.Text(c, fmt.Sprintf("指纹: %08x / %08x", uint32(photo.PHash), uint32(photo.DHash))).
											FontSize(9).Font("monospace").TextColor(t.TextMuted)
									})
								})
						}
					})
				})
			}

			if visibleGroupCount < len(a.dedupGroups) {
				remaining := len(a.dedupGroups) - visibleGroupCount
				ui.Row(c).Justify(ui.Center).Padding(8).Children(func() {
					if ui.Button(c, fmt.Sprintf("加载更多分组（还有 %d 组）", remaining)).Clicked() {
						a.dedupVisibleGroupLimit += dedupGroupPageSize
					}
				})
			}
		}
	})

	// Modals
	a.viewDedupConfigModal(c)
	a.viewDedupConfirmModal(c)
	a.viewDedupCacheClearModal(c)
}

func (a *App) viewDedupConfigModal(c *ui.Context) {
	if !a.dedupShowConfigModal {
		return
	}
	t := c.Theme()

	ui.Modal(c, &a.dedupShowConfigModal, func() {
		ui.Column(c).Width(480).Padding(24).Gap(16).Background(t.Background).Radius(12).Children(func() {
			ui.Text(c, "配置 WebDAV 远程存储").FontSize(16).Bold().TextColor(t.Text)

			ui.Column(c).Gap(10).Children(func() {
				ui.Text(c, "配置名称:").FontSize(12).Bold().TextColor(t.Text)
				ui.TextInput(c, &a.dedupEditName).Placeholder("例如: 家用群晖 NAS")

				ui.Text(c, "WebDAV URL:").FontSize(12).Bold().TextColor(t.Text)
				ui.TextInput(c, &a.dedupEditWebdavURL).Placeholder("https://nas.example.com:5006")

				ui.Row(c).Gap(10).Children(func() {
					ui.Column(c).Grow(1).Gap(4).Children(func() {
						ui.Text(c, "账号:").FontSize(12).Bold().TextColor(t.Text)
						ui.TextInput(c, &a.dedupEditWebdavUser).Placeholder("admin")
					})
					ui.Column(c).Grow(1).Gap(4).Children(func() {
						ui.Text(c, "密码:").FontSize(12).Bold().TextColor(t.Text)
						input := ui.TextInput(c, &a.dedupEditWebdavPass).Placeholder("密码...")
						if !a.dedupEditWebdavPassVis {
							input.Password()
						}
					})
				})

				ui.Text(c, "远程根目录:").FontSize(12).Bold().TextColor(t.Text)
				ui.TextInput(c, &a.dedupEditWebdavRoot).Placeholder("/photo")

				if a.dedupEditTestMsg != "" {
					ui.Text(c, a.dedupEditTestMsg).FontSize(11).TextColor(t.Accent)
				}
			})

			ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "⚡ 测试权限").Disabled(a.dedupEditTesting).Clicked() {
					// Snapshot mutable editor fields before entering the
					// goroutine; reading them live races with the UI.
					testURL := a.dedupEditWebdavURL
					testUser := a.dedupEditWebdavUser
					testPass := a.dedupEditWebdavPass
					testRoot := a.dedupEditWebdavRoot
					a.dedupEditTesting = true
					a.dedupEditTestMsg = "正在测试连接与权限..."
					go func() {
						store := storage.NewWebDAVStorage("test", testURL, testUser, testPass, testRoot)
						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()
						w, d, err := store.CheckPermission(ctx, testRoot)
						a.update(func() {
							a.dedupEditTesting = false
							switch {
							case err != nil:
								a.dedupEditTestMsg = "❌ 测试未通过: " + err.Error()
							case !w || !d:
								a.dedupEditTestMsg = "❌ 测试未通过: 缺少写入或删除权限"
							default:
								a.dedupEditTestMsg = "🟢 测试成功: 具备读写与删除权限"
							}
						})
					}()
				}

				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.dedupShowConfigModal = false
					}
					if ui.Button(c, "保存配置").Disabled(a.dedupProfilesLoadErr != "").Clicked() {
						// Editing an existing connection must never replace a
						// stored password with an empty one: the field is
						// prefilled with the decrypted password, so an empty
						// value means the user cleared it, or decryption
						// failed and it was never re-entered.
						if msg := dedupWebdavPasswordSaveError(a.dedupEditingIdx, a.dedupProfiles.Profiles, a.dedupEditWebdavPass); msg != "" {
							a.Toast(msg)
							return
						}
						encPass, encErr := storage.EncryptSecret(a.dedupEditWebdavPass)
						if encErr != nil {
							a.Toast("保存配置失败: " + encErr.Error())
							return
						}
						newProf := storage.Profile{
							ID:        fmt.Sprintf("webdav_%d", time.Now().UnixNano()),
							Name:      a.dedupEditName,
							Type:      storage.DriverWebDAV,
							UpdatedAt: time.Now(),
							WebDAV: storage.WebDAVConfig{
								URL:      a.dedupEditWebdavURL,
								Username: a.dedupEditWebdavUser,
								Password: encPass,
								RootPath: a.dedupEditWebdavRoot,
							},
						}

						if a.dedupEditingIdx >= 0 && a.dedupEditingIdx < len(a.dedupProfiles.Profiles) {
							newProf.ID = a.dedupProfiles.Profiles[a.dedupEditingIdx].ID
							newProf.CreatedAt = a.dedupProfiles.Profiles[a.dedupEditingIdx].CreatedAt
							a.dedupProfiles.Profiles[a.dedupEditingIdx] = newProf
						} else {
							newProf.CreatedAt = time.Now()
							a.dedupProfiles.Profiles = append(a.dedupProfiles.Profiles, newProf)
							a.dedupActiveProfileIdx = len(a.dedupProfiles.Profiles) - 1
						}
						a.dedupProfiles.LastUsedID = newProf.ID
						if err := a.saveDedupProfiles(); err != nil {
							return
						}
						a.dedupShowConfigModal = false
						a.Toast("配置已保存")
					}
				})
			})
		})
	})
}

func (a *App) viewDedupConfirmModal(c *ui.Context) {
	if !a.dedupShowConfirmModal {
		return
	}
	t := c.Theme()

	ui.Modal(c, &a.dedupShowConfirmModal, func() {
		ui.Column(c).Width(440).Padding(24).Gap(16).Background(t.Background).Radius(12).Children(func() {
			ui.Text(c, "确认清理废片照片").FontSize(16).Bold().TextColor(t.Text)

			totalCount, totalBytes := dedupSelectionStats(a.dedupGroups, len(a.dedupGroups))
			visibleCount := a.dedupVisibleGroupCount()
			expandedSel, expandedBytes := dedupSelectionStats(a.dedupGroups, visibleCount)
			unexpanded := len(a.dedupGroups) - visibleCount

			if unexpanded > 0 {
				ui.Text(c, fmt.Sprintf("已展开审阅 %d 组（勾选 %d 张，共 %s）；另有 %d 组未在界面展开审阅（已按推荐预选，勾选 %d 张）。",
					visibleCount, expandedSel, services.FormatFileSize(expandedBytes), unexpanded, totalCount-expandedSel)).
					FontSize(13).TextColor(t.Text)
				ui.Text(c, "建议优先「仅清理已展开审阅的分组」逐批复核；确认信任系统推荐保留结果时，再选择一并清理全部分组。").
					FontSize(11).TextColor(t.TextMuted)
			} else {
				ui.Text(c, fmt.Sprintf("即将清理已选中的 %d 张废片，预计可释放 %s 空间。",
					totalCount, services.FormatFileSize(totalBytes))).
					FontSize(13).TextColor(t.Text)
			}

			ui.Column(c).Gap(8).Padding(10).Radius(8).Background(t.Surface).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					isTrash := a.dedupConfirmTrashOnly
					if ui.Checkbox(c, &isTrash, "移至系统回收站 / 隔离目录 (推荐，可安全恢复)").Disabled(a.dedupIsDeleting).Changed() {
						a.dedupConfirmTrashOnly = isTrash
					}
				})
			})

			ui.Row(c).Justify(ui.End).Gap(8).Wrap().Children(func() {
				if ui.Button(c, "取消").Disabled(a.dedupIsDeleting).Clicked() {
					a.dedupShowConfirmModal = false
				}
				if unexpanded > 0 {
					if ui.Button(c, fmt.Sprintf("仅清理已展开 %d 组 (%d 张)", visibleCount, expandedSel)).
						Disabled(a.dedupIsDeleting || expandedSel == 0).Clicked() {
						a.executeDedupDeletion(false)
					}
					if ui.Button(c, fmt.Sprintf("一并清理全部 %d 组 (%d 张)", len(a.dedupGroups), totalCount)).
						Disabled(a.dedupIsDeleting || totalCount == 0).Clicked() {
						a.executeDedupDeletion(true)
					}
				} else {
					if ui.Button(c, "确认执行清理").Disabled(a.dedupIsDeleting).Clicked() {
						a.executeDedupDeletion(true)
					}
				}
			})
		})
	})
}

func (a *App) viewDedupCacheClearModal(c *ui.Context) {
	if !a.dedupShowCacheClearModal {
		return
	}
	t := c.Theme()

	ui.Modal(c, &a.dedupShowCacheClearModal, func() {
		ui.Column(c).Width(440).Padding(24).Gap(16).Background(t.Background).Radius(12).Children(func() {
			ui.Text(c, "清空本地缓存").FontSize(16).Bold().TextColor(t.Text)
			ui.Text(c, fmt.Sprintf("将删除 %d 张缩略图和 %d 条指纹缓存，共 %s。此操作不会修改原始照片。",
				a.dedupCacheStatsCount, a.dedupCacheStatsFpCount, services.FormatFileSize(a.dedupCacheStatsBytes))).
				FontSize(13).TextColor(t.Text)
			ui.Text(c, "清空后下次扫描需要重新读取和解码图片，但不会影响照片本身。").
				FontSize(11).TextColor(t.TextMuted)

			ui.Row(c).Justify(ui.End).Gap(8).Children(func() {
				if ui.Button(c, "取消").Disabled(a.dedupCacheBusy).Clicked() {
					a.dedupShowCacheClearModal = false
				}
				if ui.Button(c, "确认清空").Disabled(a.dedupCacheBusy).Clicked() {
					a.executeClearDedupCache()
				}
			})
		})
	})
}
