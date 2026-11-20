package dedup

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/welllog/otool/internal/storage"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestMain isolates HOME/XDG_CACHE_HOME so scans that persist the fingerprint
// cache never touch the developer's real user cache directory.
func TestMain(m *testing.M) {
	oldHome, hadHome := os.LookupEnv("HOME")
	oldCache, hadCache := os.LookupEnv("XDG_CACHE_HOME")
	home, err := os.MkdirTemp("", "otool-test-home-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	code := m.Run()
	if hadHome {
		_ = os.Setenv("HOME", oldHome)
	} else {
		_ = os.Unsetenv("HOME")
	}
	if hadCache {
		_ = os.Setenv("XDG_CACHE_HOME", oldCache)
	} else {
		_ = os.Unsetenv("XDG_CACHE_HOME")
	}
	_ = os.RemoveAll(home)
	os.Exit(code)
}

// createTestImage creates a simple in-memory JPEG image with shapes.
func createTestImage(w, h int, drawShapes bool) []byte {
	m := image.NewRGBA(image.Rect(0, 0, w, h))
	// Fill white
	draw.Draw(m, m.Bounds(), &image.Uniform{color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)

	if drawShapes {
		// Draw a structured, non-monotonic pattern so both pHash and dHash have
		// meaningful entropy (a plain gradient makes dHash all ones).
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				checker := uint8(((x/7 + y/9) % 2) * 80)
				m.Set(x, y, color.RGBA{
					uint8((x * 255 / w) ^ int(checker)),
					uint8((y * 255 / h) ^ int(checker)),
					uint8(((x+y)*255/(w+h) + int(checker)) / 2),
					255,
				})
			}
		}
	}

	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, m, &jpeg.Options{Quality: 90})
	return buf.Bytes()
}

func TestDecodeMemoryGateCancellation(t *testing.T) {
	gate := newDecodeMemoryGate(100)
	reserved, ok := gate.acquire(context.Background(), 100)
	if !ok || reserved != 100 {
		t.Fatalf("expected to reserve the whole budget, got %d, %v", reserved, ok)
	}
	defer gate.release(reserved)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan bool, 1)
	go func() {
		_, ok := gate.acquire(ctx, 100)
		done <- ok
	}()

	// The second acquire must be waiting for the budget to be released.
	select {
	case ok := <-done:
		t.Fatalf("acquire returned before cancellation: %v", ok)
	case <-time.After(20 * time.Millisecond):
	}

	cancel()
	select {
	case ok := <-done:
		if ok {
			t.Fatal("expected canceled acquire to fail")
		}
	case <-time.After(time.Second):
		t.Fatal("acquire did not wake up after context cancellation")
	}
}

func TestLaplacianVariance(t *testing.T) {
	// Sharp image with shapes
	sharpBytes := createTestImage(200, 200, true)
	sharpImg, _ := jpeg.Decode(bytes.NewReader(sharpBytes))
	sharpScore := ComputeLaplacianVariance(sharpImg)

	// Blank / smooth image
	smoothBytes := createTestImage(200, 200, false)
	smoothImg, _ := jpeg.Decode(bytes.NewReader(smoothBytes))
	smoothScore := ComputeLaplacianVariance(smoothImg)

	if sharpScore <= smoothScore {
		t.Fatalf("expected sharp image score (%f) to be greater than smooth image score (%f)", sharpScore, smoothScore)
	}
}

func TestBKTreeAndCluster(t *testing.T) {
	photos := []*PhotoMeta{
		{ID: 1, PHash: 0x1111111111111111, DHash: 0x2222222222222222, Width: 1920, Height: 1080, Size: 1000},
		{ID: 2, PHash: 0x1111111111111113, DHash: 0x2222222222222223, Width: 1280, Height: 720, Size: 500},  // dist = 1 to #1
		{ID: 3, PHash: 0xFFFFFFFFFFFFFFFF, DHash: 0xFFFFFFFFFFFFFFFF, Width: 1920, Height: 1080, Size: 800}, // very far
	}

	clusters := ClusterSimilarPhotos(photos, 5)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster group, got %d", len(clusters))
	}
	if len(clusters[0]) != 2 {
		t.Fatalf("expected cluster to have 2 photos, got %d", len(clusters[0]))
	}

	grp := &DuplicateGroup{
		GroupID: "test_grp",
		Mode:    ModeSimilar,
		Photos:  clusters[0],
	}
	EvaluateAndScoreGroup(grp)

	if grp.BestPickID != 1 {
		t.Fatalf("expected photo 1 to be BestPick due to higher resolution, got %d", grp.BestPickID)
	}
	if grp.Photos[0].Selected != false {
		t.Fatalf("expected BestPick to have Selected=false (keep)")
	}
	if grp.Photos[1].Selected != true {
		t.Fatalf("expected non-BestPick to have Selected=true (marked for delete)")
	}
}

func TestPreventTransitiveChaining(t *testing.T) {
	// A-B = 5, B-C = 5, A-C = 10: A and C must not be merged through B.
	base := uint64(0xAAAAAAAA00000000)
	pA := &PhotoMeta{ID: 1, PHash: base, DHash: base, Width: 100, Height: 100}
	pB := &PhotoMeta{ID: 2, PHash: base ^ 0x1F, DHash: base ^ 0x1F, Width: 100, Height: 100}
	pC := &PhotoMeta{ID: 3, PHash: base ^ 0x3FF, DHash: base ^ 0x3FF, Width: 100, Height: 100}

	clusters := ClusterSimilarPhotos([]*PhotoMeta{pA, pB, pC}, 5)
	for _, c := range clusters {
		if len(c) == 3 {
			t.Fatalf("expected chaining to be prevented, but got a 3-photo cluster containing incompatible A and C!")
		}
	}
}

func TestSimilarThresholdIsRespected(t *testing.T) {
	base := uint64(0xAAAAAAAAAAAAAAAA)
	a := &PhotoMeta{ID: 1, PHash: base, DHash: base, Width: 100, Height: 100}
	// pHash distance 10, dHash distance 3; with a 5-distance budget this pair
	// must not be accepted just because one of the two hashes is close.
	b := &PhotoMeta{ID: 2, PHash: base ^ 0x3FF, DHash: base ^ 0x7, Width: 100, Height: 100}

	dist, similar := ArePhotosVisuallySimilar(a, b, 5)
	if similar {
		t.Fatalf("expected distance %d to be rejected at threshold 5", dist)
	}
	if dist <= 5 {
		t.Fatalf("expected combined distance > 5, got %d", dist)
	}

	// One hash very close must not make an otherwise divergent pair acceptable.
	c := &PhotoMeta{ID: 3, PHash: base ^ 0x3, DHash: base ^ 0x1FFFF, Width: 100, Height: 100}
	if d, ok := ArePhotosVisuallySimilar(a, c, 5); ok {
		t.Fatalf("expected near-identical pHash but divergent dHash to be rejected at threshold 5, got %d", d)
	}
}

func TestBlackImagesCanBeClustered(t *testing.T) {
	// A pure black image legitimately has pHash == 0 and dHash == 0; zero must
	// not be treated as a missing/invalid hash.
	a := &PhotoMeta{ID: 1, PHash: 0, DHash: 0, Width: 100, Height: 100}
	b := &PhotoMeta{ID: 2, PHash: 0, DHash: 0, Width: 100, Height: 100}

	clusters := ClusterSimilarPhotos([]*PhotoMeta{a, b}, 8)
	if len(clusters) != 1 || len(clusters[0]) != 2 {
		t.Fatalf("expected identical black images to form one cluster, got %#v", clusters)
	}
}

func TestEvaluatePreservesSelection(t *testing.T) {
	best := &PhotoMeta{ID: 1, PHash: 1, DHash: 1, Width: 4000, Height: 3000, Size: 3000}
	kept := &PhotoMeta{ID: 2, PHash: 2, DHash: 2, Width: 2000, Height: 1500, Size: 2000}
	pending := &PhotoMeta{ID: 3, PHash: 3, DHash: 3, Width: 1000, Height: 750, Size: 1000}
	kept.Selected = false
	pending.Selected = true

	group := &DuplicateGroup{GroupID: "g", Mode: ModeSimilar, Photos: []*PhotoMeta{best, kept, pending}}
	EvaluateAndScoreGroupPreserveSelection(group)

	for _, p := range group.Photos {
		if p.ID == kept.ID && p.Selected {
			t.Fatalf("user-kept photo was unexpectedly marked for deletion")
		}
		if p.ID == pending.ID && !p.Selected {
			t.Fatalf("pending photo selection was unexpectedly cleared")
		}
	}
}

// TestEvaluateBestPickPrefersUnselectedKeeper covers the rebuild path after a
// partial cleanup: if the user marked the highest-scoring photo for deletion,
// the crown must go to the best-scoring keeper instead of contradicting its
// own cleanup checkbox.
func TestEvaluateBestPickPrefersUnselectedKeeper(t *testing.T) {
	top := &PhotoMeta{ID: 1, PHash: 1, DHash: 1, Width: 4000, Height: 3000, Size: 3000, Selected: true}
	mid := &PhotoMeta{ID: 2, PHash: 2, DHash: 2, Width: 2000, Height: 1500, Size: 2000}
	low := &PhotoMeta{ID: 3, PHash: 3, DHash: 3, Width: 1000, Height: 750, Size: 1000}

	grp := &DuplicateGroup{GroupID: "g", Mode: ModeSimilar, Photos: []*PhotoMeta{top, mid, low}}
	EvaluateAndScoreGroupPreserveSelection(grp)

	if grp.BestPickID != mid.ID {
		t.Fatalf("expected best-scoring keeper (ID %d) as BestPick, got %d", mid.ID, grp.BestPickID)
	}
	if top.Selected != true || mid.Selected != false || low.Selected != false {
		t.Fatal("preserve mode must not alter user selections")
	}

	// The unselected-keeper preference must not leak into normal scoring:
	// a fresh group still crowns the highest-scoring photo.
	freshTop := &PhotoMeta{ID: 11, PHash: 1, DHash: 1, Width: 4000, Height: 3000, Size: 3000}
	freshMid := &PhotoMeta{ID: 12, PHash: 2, DHash: 2, Width: 2000, Height: 1500, Size: 2000}
	grp2 := &DuplicateGroup{GroupID: "g2", Mode: ModeSimilar, Photos: []*PhotoMeta{freshTop, freshMid}}
	EvaluateAndScoreGroup(grp2)

	if grp2.BestPickID != freshTop.ID {
		t.Fatalf("normal scoring must crown the highest-scoring photo, got %d", grp2.BestPickID)
	}
	if freshMid.Selected != true {
		t.Fatal("normal scoring must preselect non-best photos for deletion")
	}
}

func TestOrientationRejection(t *testing.T) {
	// Portrait photo (e.g. 3000x4000) vs Landscape photo (e.g. 4032x3024)
	portrait := &PhotoMeta{ID: 1, PHash: 0x1111111111111111, DHash: 0x2222222222222222, Width: 3000, Height: 4000}
	landscape := &PhotoMeta{ID: 2, PHash: 0x111111111111111E, DHash: 0x222222222222222E, Width: 4032, Height: 3024} // dist 4 (0001 ^ 1110 = 1111)

	_, similar := ArePhotosVisuallySimilar(portrait, landscape, 8)
	if similar {
		t.Fatalf("expected portrait and landscape photos with dist > 3 to NOT be similar!")
	}
}

func TestEngineEndToEnd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otool_engine_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create 2 identical/near-identical images
	imgData1 := createTestImage(300, 300, true)
	img1Dec, _ := jpeg.Decode(bytes.NewReader(imgData1))
	var buf2 bytes.Buffer
	_ = jpeg.Encode(&buf2, img1Dec, &jpeg.Options{Quality: 70})
	imgData2 := buf2.Bytes()

	_ = os.WriteFile(filepath.Join(tmpDir, "photo1.jpg"), imgData1, 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "photo1_copy.jpg"), imgData2, 0644)

	res1, err1 := AnalyzeImage(imgData1, "photo1.jpg")
	res2, err2 := AnalyzeImage(imgData2, "photo1_copy.jpg")
	t.Logf("res1: err=%v, phash=%x, dhash=%x", err1, res1.PHash, res1.DHash)
	t.Logf("res2: err=%v, phash=%x, dhash=%x", err2, res2.PHash, res2.DHash)
	t.Logf("dist: %d", HammingDistance(res1.PHash, res2.PHash))

	store := storage.NewLocalStorage("local_test")
	eng := NewEngine()

	opts := DefaultScanOptions()
	opts.TargetDir = tmpDir
	opts.Mode = ModeSimilar
	opts.SimilarDistance = 10

	var lastPhase string
	groups, err := eng.RunScan(context.Background(), store, opts, func(p ScanProgress) {
		lastPhase = p.Phase
		if p.Error != nil {
			t.Logf("progress err: %v", p.Error)
		}
	})

	t.Logf("discovered: err=%v, groups=%d", err, len(groups))

	if err != nil {
		t.Fatalf("scan error: %v", err)
	}
	if lastPhase != "done" {
		t.Fatalf("expected final phase 'done', got '%s'", lastPhase)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 duplicate group found, got %d", len(groups))
	}
	for _, p := range groups[0].Photos {
		if p.ContentHash == 0 {
			t.Fatal("expected similar-mode groups to have full content hashes for safe deletion")
		}
	}

	// The late-computed hashes must be written back into the fingerprint cache
	// and flushed to disk, so the next scan does not re-read group members.
	cm := storage.GetCacheManager()
	raw, err := os.ReadFile(filepath.Join(cm.CacheDir(), "fingerprints.json"))
	if err != nil {
		t.Fatalf("fingerprints.json not flushed after similar scan: %v", err)
	}
	var persisted map[string]storage.CachedFingerprint
	if err := json.Unmarshal(raw, &persisted); err != nil {
		t.Fatal(err)
	}
	for _, p := range groups[0].Photos {
		if p.CacheKey == "" {
			t.Fatal("expected a cache key for every similar group photo")
		}
		fp, ok := cm.GetFingerprint(p.CacheKey)
		if !ok || fp.ContentHash != p.ContentHash {
			t.Fatalf("expected the late-computed hash %x to be cached, ok=%v cached=%x", p.ContentHash, ok, fp.ContentHash)
		}
		if persisted[p.CacheKey].ContentHash != p.ContentHash {
			t.Fatalf("expected the hash to be persisted to fingerprints.json, got %x", persisted[p.CacheKey].ContentHash)
		}
	}

	best := groups[0].BestPickID
	if groups[0].Photos[0].ID != best {
		t.Fatalf("expected first photo in group to be best pick")
	}
}

// TestFillGroupContentHashesUpdatesFingerprintCache verifies that the hashes
// learned after clustering are written back into the fingerprint cache, so a
// later scan reuses them instead of re-reading (re-downloading) group
// members, and that no partial entry is fabricated for unknown keys.
func TestFillGroupContentHashesUpdatesFingerprintCache(t *testing.T) {
	tmpDir := t.TempDir()
	data := createTestImage(80, 60, true)
	path := filepath.Join(tmpDir, "photo.jpg")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	want, err := ComputeFullContentHash(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	store := storage.NewLocalStorage("hash_cache_test")
	key := storage.GenerateCacheKey("hash_cache_test", path, fi.Size(), fi.ModTime())
	missingKey := storage.GenerateCacheKey("hash_cache_test", path, 1, fi.ModTime())

	cm := storage.GetCacheManager()
	cm.SaveFingerprint(key, storage.CachedFingerprint{Width: 80, Height: 60, Format: "jpeg", PHash: 1, DHash: 2})
	if fp, ok := cm.GetFingerprint(key); !ok || fp.ContentHash != 0 {
		t.Fatalf("expected a seeded cache entry without a content hash, ok=%v hash=%x", ok, fp.ContentHash)
	}

	groups := []*DuplicateGroup{{
		GroupID: "g",
		Mode:    ModeSimilar,
		Photos: []*PhotoMeta{
			{ID: 1, StorageID: "hash_cache_test", Path: path, Size: fi.Size(), ModTime: fi.ModTime(), CacheKey: key},
			{ID: 2, StorageID: "hash_cache_test", Path: path, Size: fi.Size(), ModTime: fi.ModTime(), CacheKey: missingKey},
		},
	}}

	eng := NewEngine()
	skipped, err := eng.fillGroupContentHashes(context.Background(), store, groups, 2)
	if err != nil {
		t.Fatalf("fillGroupContentHashes failed: %v", err)
	}
	if skipped != 0 {
		t.Fatalf("expected no skipped photos, got %d", skipped)
	}
	for _, p := range groups[0].Photos {
		if p.ContentHash != want {
			t.Fatalf("photo %d hash mismatch: got %x want %x", p.ID, p.ContentHash, want)
		}
	}
	fp, ok := cm.GetFingerprint(key)
	if !ok {
		t.Fatal("fingerprint vanished from cache")
	}
	if fp.ContentHash != want {
		t.Fatalf("cached content hash not updated: got %x want %x", fp.ContentHash, want)
	}
	if _, ok := cm.GetFingerprint(missingKey); ok {
		t.Fatal("fill must not fabricate cache entries for keys that were never analyzed")
	}
}

func TestEngineExactMode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otool_exact_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content1 := createTestImage(150, 150, true)
	content2 := createTestImage(150, 150, false)

	// Two identical files
	_ = os.WriteFile(filepath.Join(tmpDir, "file1.jpg"), content1, 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "file1_exact_copy.jpg"), content1, 0644)
	// One different file
	_ = os.WriteFile(filepath.Join(tmpDir, "file2.jpg"), content2, 0644)

	store := storage.NewLocalStorage("exact_test")
	eng := NewEngine()

	opts := DefaultScanOptions()
	opts.TargetDir = tmpDir
	opts.Mode = ModeExact

	groups, err := eng.RunScan(context.Background(), store, opts, nil)
	if err != nil {
		t.Fatalf("exact scan failed: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 exact duplicate group, got %d", len(groups))
	}
	if len(groups[0].Photos) != 2 {
		t.Fatalf("expected 2 photos in exact group, got %d", len(groups[0].Photos))
	}
	if groups[0].SimilarityPercent != 100 {
		t.Fatalf("expected 100%% similarity, got %d", groups[0].SimilarityPercent)
	}
	for _, p := range groups[0].Photos {
		if p.Width <= 0 || p.Height <= 0 {
			t.Fatalf("expected representative metadata to be cloned to every exact copy, got %dx%d", p.Width, p.Height)
		}
		if p.CacheKey == "" {
			t.Fatal("expected a cache key for every exact copy")
		}
	}
}

func TestComputePartialHashFromStoreMatchesSeekerHash(t *testing.T) {
	tmpDir := t.TempDir()
	data := make([]byte, 20000)
	for i := range data {
		data[i] = byte(i*31 + i/7)
	}
	path := filepath.Join(tmpDir, "sample.jpg")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	store := storage.NewLocalStorage("partial_test")
	got, err := ComputePartialHashFromStore(context.Background(), store, path, int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	want, err := ComputePartialHash(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("range-based partial hash mismatch: got %x want %x", got, want)
	}
}

func TestEngineExactModeKeepsUndecodableFiles(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otool_exact_heic_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Exact mode only needs the byte hash; these files cannot be decoded by
	// image.Decode, but they are still byte-for-byte duplicates.
	payload := []byte("not-a-real-heic-but-identical-bytes")
	_ = os.WriteFile(filepath.Join(tmpDir, "photo1.heic"), payload, 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "photo1_copy.heic"), payload, 0644)

	store := storage.NewLocalStorage("exact_heic_test")
	eng := NewEngine()
	opts := DefaultScanOptions()
	opts.TargetDir = tmpDir
	opts.Mode = ModeExact

	var finalProgress ScanProgress
	groups, err := eng.RunScan(context.Background(), store, opts, func(p ScanProgress) {
		if p.IsComplete {
			finalProgress = p
		}
	})
	if err != nil {
		t.Fatalf("exact scan failed: %v", err)
	}
	if len(groups) != 1 || len(groups[0].Photos) != 2 {
		t.Fatalf("expected undecodable byte-identical files to stay in one exact group, got %#v", groups)
	}
	// The failed representative decode must be reported as a preview failure,
	// not miscounted as skipped photos: both files are in the results.
	if finalProgress.PreviewFailedGroups != 1 {
		t.Fatalf("expected PreviewFailedGroups=1, got %d", finalProgress.PreviewFailedGroups)
	}
	if finalProgress.SkippedCount != 0 {
		t.Fatalf("preview failure must not count as skipped photos, got %d", finalProgress.SkippedCount)
	}
}

func TestBKTreeIdenticalHashGrouping(t *testing.T) {
	tree := NewBKTree()
	tree.Insert(1, 0x1234567812345678)
	tree.Insert(2, 0x1234567812345678) // Same hash
	tree.Insert(3, 0x1234567812345679) // Distance 1

	var matches []uint64
	tree.Search(0x1234567812345678, 0, func(ids []uint64, dist int) {
		matches = append(matches, ids...)
	})

	if len(matches) != 2 {
		t.Fatalf("expected 2 IDs with exact same hash, got %d", len(matches))
	}
}

// TestMergeCompatibleClustersSingletonHandling locks in the singleton-skip
// optimization in mergeCompatibleClusters: singleton×singleton pairs are
// provably unmergeable (the greedy phase would already have absorbed them),
// but a singleton that is compatible with every member of a multi cluster
// must still merge, and cancellation must exit the pass immediately.
func TestMergeCompatibleClustersSingletonHandling(t *testing.T) {
	base := uint64(0xAAAAAAAAAAAAAAAA)
	newPhoto := func(id uint64, delta uint64) *PhotoMeta {
		return &PhotoMeta{ID: id, PHash: base ^ delta, DHash: base ^ delta, Width: 100, Height: 100}
	}
	pA, pB, pS := newPhoto(1, 0), newPhoto(2, 0x1), newPhoto(3, 0x3)

	// A singleton compatible with the whole multi cluster must merge into it,
	// in either slice order.
	for name, order := range map[string][][]*PhotoMeta{
		"multi-first":     {{pA, pB}, {pS}},
		"singleton-first": {{pS}, {pA, pB}},
	} {
		merged := mergeCompatibleClusters(context.Background(), order, 8)
		if len(merged) != 1 || len(merged[0]) != 3 {
			t.Fatalf("%s: expected singleton to merge into the multi cluster, got %d clusters", name, len(merged))
		}
	}

	// Singleton-only input is returned unchanged via the fast path.
	singles := [][]*PhotoMeta{{newPhoto(10, 0x55)}, {newPhoto(11, 0x77)}, {newPhoto(12, 0x99)}}
	out := mergeCompatibleClusters(context.Background(), singles, 8)
	if len(out) != 3 || len(out[0]) != 1 || len(out[1]) != 1 || len(out[2]) != 1 {
		t.Fatalf("expected singleton-only input unchanged, got %d clusters", len(out))
	}

	// Cancellation returns nil so the engine can report a stopped scan.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got := mergeCompatibleClusters(ctx, [][]*PhotoMeta{{pA, pB}, {pS}}, 8); got != nil {
		t.Fatalf("expected nil on cancelled context, got %v", got)
	}
}

func TestAnalyzeImageReaderNoThumbnail(t *testing.T) {
	data := createTestImage(120, 90, true)
	res, err := AnalyzeImageReaderContextNoThumb(context.Background(), bytes.NewReader(data), "no-thumb.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ThumbnailBytes) != 0 {
		t.Fatalf("expected no thumbnail bytes, got %d", len(res.ThumbnailBytes))
	}
	if res.Width != 120 || res.Height != 90 {
		t.Fatalf("unexpected dimensions: %dx%d", res.Width, res.Height)
	}
}

func TestGenerateThumbnailFromStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "thumb-source.jpg")
	data := createTestImage(180, 120, true)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	store := storage.NewLocalStorage("thumb_test")
	thumb, err := GenerateThumbnailFromStore(context.Background(), store, path, "thumb-source.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := jpeg.Decode(bytes.NewReader(thumb)); err != nil {
		t.Fatalf("generated thumbnail is not valid JPEG: %v", err)
	}
}

func TestComputeFullContentHashContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ComputeFullContentHashContext(ctx, bytes.NewReader(bytes.Repeat([]byte{1}, 1024))); err == nil {
		t.Fatal("expected canceled context to stop hashing")
	}
}

func TestSimilarScanSkipsUnsupportedFormats(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.heic", "b.heic"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("not-a-heic"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	store := storage.NewLocalStorage("unsupported_test")
	eng := NewEngine()
	opts := DefaultScanOptions()
	opts.TargetDir = dir
	opts.Mode = ModeSimilar
	var final ScanProgress
	groups, err := eng.RunScan(context.Background(), store, opts, func(p ScanProgress) {
		if p.IsComplete {
			final = p
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected no groups, got %d", len(groups))
	}
	if final.SkippedCount != 2 {
		t.Fatalf("expected 2 unsupported files skipped, got %+v", final)
	}
}

func TestEstimateDecodeMemoryDoesNotOverflow(t *testing.T) {
	for _, tc := range []struct{ w, h int }{
		{1 << 30, 1 << 30},
		{1<<31 - 1, 1<<31 - 1},
		{200_000_000, 2},
	} {
		if got := EstimateDecodeMemory(tc.w, tc.h); got <= 0 {
			t.Fatalf("EstimateDecodeMemory(%d,%d) overflowed: %d", tc.w, tc.h, got)
		}
	}
	if got := EstimateDecodeMemory(0, 0); got != 64<<20 {
		t.Fatalf("unexpected zero-dimension estimate: %d", got)
	}
}

type readerOnly struct{ io.Reader }

func TestComputePartialHashNonSeekerMatchesSeeker(t *testing.T) {
	data := make([]byte, 20000)
	for i := range data {
		data[i] = byte(i*17 + i/3)
	}
	for _, size := range []int64{13000, 20000, 50000} {
		payload := data
		if int64(len(payload)) < size {
			payload = make([]byte, size)
			copy(payload, data)
			for i := len(data); i < len(payload); i++ {
				payload[i] = byte(i * 19)
			}
		}
		want, err := ComputePartialHash(bytes.NewReader(payload[:size]), size)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ComputePartialHash(readerOnly{bytes.NewReader(payload[:size])}, size)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("non-seeker partial hash mismatch for size %d: got %x want %x", size, got, want)
		}
	}
}
