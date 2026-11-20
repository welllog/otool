package storage

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

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

func resetProfileKeyCache() {
	keyMu.Lock()
	keyLoaded = false
	cachedKey = nil
	keyLoadErr = nil
	keyMu.Unlock()
}

func TestProfileEncryptDecrypt(t *testing.T) {
	resetProfileKeyCache()
	t.Setenv("HOME", t.TempDir())
	orig := "my_secret_password_123!@#"
	enc, err := EncryptSecret(orig)
	if err != nil {
		t.Fatal(err)
	}
	if enc == orig {
		t.Fatalf("expected encrypted string to differ from original")
	}
	dec, err := DecryptSecret(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != orig {
		t.Fatalf("expected decrypted string %s, got %s", orig, dec)
	}
	keyInfo, err := os.Stat(filepath.Join(GetConfigDir(), "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	if keyInfo.Mode().Perm()&0077 != 0 {
		t.Fatalf("secret.key is readable by other users: %o", keyInfo.Mode().Perm())
	}
}

func TestLocalProfileConfigUsesRandomSecretKey(t *testing.T) {
	resetProfileKeyCache()
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := GetConfigDir()
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte(`{"version":1,"last_used_id":"local_default","profiles":[{"id":"local_default","name":"local","type":"local"}]}`)
	if err := os.WriteFile(filepath.Join(cfgDir, "storage_profiles.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := EncryptSecret("secret"); err != nil {
		t.Fatal(err)
	}
	key, err := os.ReadFile(filepath.Join(cfgDir, "secret.key"))
	if err != nil {
		t.Fatal(err)
	}
	hostname, _ := os.Hostname()
	user, _ := os.UserHomeDir()
	legacy := sha256.Sum256([]byte(fmt.Sprintf("otool-salt-v1:%s:%s", hostname, user)))
	if bytes.Equal(key, legacy[:]) {
		t.Fatal("new local profile configuration reused the legacy hostname/home key")
	}
}

func TestLocalStorage(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otool_local_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	subDir := filepath.Join(tmpDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	f1 := filepath.Join(tmpDir, "test1.jpg")
	f2 := filepath.Join(subDir, "test2.png")
	f3 := filepath.Join(tmpDir, "not_image.txt")

	_ = os.WriteFile(f1, []byte("fake_image_1"), 0644)
	_ = os.WriteFile(f2, []byte("fake_image_2"), 0644)
	_ = os.WriteFile(f3, []byte("text"), 0644)

	store := NewLocalStorage("test")

	// 1. CheckPermission
	w, d, err := store.CheckPermission(context.Background(), tmpDir)
	if err != nil || !w || !d {
		t.Fatalf("expected write and delete permission, got w=%v d=%v err=%v", w, d, err)
	}

	// 2. List recursive
	var count int
	err = store.List(context.Background(), tmpDir, true, func(item FileItem) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected 2 image files, got %d", count)
	}

	// 3. List non-recursive
	count = 0
	err = store.List(context.Background(), tmpDir, false, func(item FileItem) error {
		count++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 image file in top dir, got %d", count)
	}

	// 4. OpenRange
	rc, err := store.OpenRange(context.Background(), f1, 5, 5)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 10)
	n, _ := rc.Read(buf)
	_ = rc.Close()
	if n != 5 || string(buf[:n]) != "image" {
		t.Fatalf("expected 'image', got '%s'", string(buf[:n]))
	}
}

func TestLocalStorageSkipsTrashDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "otool_trash_skip_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	_ = os.WriteFile(filepath.Join(tmpDir, "keep.jpg"), []byte("keep"), 0644)
	trashDir := filepath.Join(tmpDir, TrashDirName)
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(trashDir, "deleted.jpg"), []byte("deleted"), 0644)
	for _, name := range []string{".Trashes", ".Trash-1000"} {
		external := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(external, 0755); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(external, "deleted.jpg"), []byte("deleted"), 0644)
	}

	store := NewLocalStorage("trash_test")
	var count int
	if err := store.List(context.Background(), tmpDir, true, func(item FileItem) error {
		count++
		if filepath.Base(filepath.Dir(item.Path)) == TrashDirName {
			t.Fatalf("trash file should not be listed: %s", item.Path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected only the non-trash image, got %d", count)
	}
}

func TestCacheKeyIncludesETag(t *testing.T) {
	now := time.Now()
	a := GenerateCacheKey("webdav", "/photo/a.jpg", 1024, now, "etag-a")
	b := GenerateCacheKey("webdav", "/photo/a.jpg", 1024, now, "etag-b")
	if a == b {
		t.Fatal("expected different ETags to produce different cache keys")
	}
}

func TestCacheManager(t *testing.T) {
	cm := GetCacheManager()
	key := GenerateCacheKey("test", "/path/to/img.jpg", 1024, time.Now())

	data := []byte("fake_webp_bytes")
	if err := cm.SaveThumbnail(key, data); err != nil {
		t.Fatal(err)
	}

	got, ok := cm.GetThumbnail(key)
	if !ok || string(got) != string(data) {
		t.Fatalf("failed to retrieve cached thumbnail")
	}

	if err := cm.RemoveThumbnail(key); err != nil {
		t.Fatal(err)
	}

	_, ok = cm.GetThumbnail(key)
	if ok {
		t.Fatalf("expected thumbnail to be removed")
	}
}

func TestFingerprintCache(t *testing.T) {
	cm := GetCacheManager()
	key := GenerateCacheKey("test_fp", "/path/to/photo.jpg", 2048, time.Now())

	fp := CachedFingerprint{
		Width:       1920,
		Height:      1080,
		Format:      "JPEG",
		ContentHash: 0x1122334455667788,
		PHash:       0x12345678,
		DHash:       0x87654321,
		BlurScore:   85.5,
		HasEXIF:     true,
		CameraModel: "TestCam",
	}

	cm.SaveFingerprint(key, fp)

	retrieved, ok := cm.GetFingerprint(key)
	if !ok {
		t.Fatalf("expected fingerprint to be found in cache")
	}
	if retrieved.Width != 1920 || retrieved.CameraModel != "TestCam" || retrieved.ContentHash != fp.ContentHash {
		t.Fatalf("retrieved fingerprint mismatch: %+v", retrieved)
	}

	if err := cm.FlushFingerprints(); err != nil {
		t.Fatalf("failed to flush fingerprints: %v", err)
	}
}

func TestCachePruneKeepsFingerprintWhenThumbnailExpires(t *testing.T) {
	cm := GetCacheManager()
	key := GenerateCacheKey("prune_test", "/path/to/expired.jpg", 4096, time.Now())

	if err := cm.SaveThumbnail(key, []byte("expired-thumb")); err != nil {
		t.Fatal(err)
	}
	cm.SaveFingerprint(key, CachedFingerprint{Width: 1024, Height: 768})
	if !cm.HasThumbnail(key) {
		t.Fatal("expected thumbnail to exist before prune")
	}

	thumbPath := filepath.Join(cm.CacheDir(), key+".jpg")
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(thumbPath, old, old); err != nil {
		t.Fatal(err)
	}

	result, err := cm.PruneCache(0, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if result.RemovedThumbnails < 1 {
		t.Fatalf("expected at least one pruned thumbnail, got %+v", result)
	}
	if cm.HasThumbnail(key) {
		t.Fatal("expected expired thumbnail to be removed")
	}
	if _, ok := cm.GetFingerprint(key); !ok {
		t.Fatal("expected fingerprint to survive thumbnail pruning")
	}
}

func TestCacheStatsAndClear(t *testing.T) {
	cm := GetCacheManager()
	key := GenerateCacheKey("stats_test", "/path/to/stats.jpg", 2048, time.Now())
	if err := cm.SaveThumbnail(key, []byte("stats-thumb")); err != nil {
		t.Fatal(err)
	}
	cm.SaveFingerprint(key, CachedFingerprint{Width: 800, Height: 600})

	stats, err := cm.RefreshCacheStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.ThumbnailCount < 1 || stats.TotalBytes <= 0 || stats.FingerprintCount < 1 {
		t.Fatalf("unexpected cache stats: %+v", stats)
	}

	if err := cm.ClearCache(); err != nil {
		t.Fatal(err)
	}
	stats, err = cm.RefreshCacheStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalBytes != 0 || stats.ThumbnailCount != 0 || stats.FingerprintCount != 0 {
		t.Fatalf("expected empty cache after clear, got %+v", stats)
	}
}

func TestCacheFilesArePrivate(t *testing.T) {
	cm := GetCacheManager()
	dirInfo, err := os.Stat(cm.CacheDir())
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm()&0077 != 0 {
		t.Fatalf("cache directory is readable by other users: %o", dirInfo.Mode().Perm())
	}

	key := GenerateCacheKey("private_test", "/path/to/private.jpg", 1024, time.Now())
	if err := cm.SaveThumbnail(key, []byte("private-thumb")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cm.RemoveThumbnail(key) })
	thumbInfo, err := os.Stat(filepath.Join(cm.CacheDir(), key+".jpg"))
	if err != nil {
		t.Fatal(err)
	}
	if thumbInfo.Mode().Perm()&0077 != 0 {
		t.Fatalf("thumbnail is readable by other users: %o", thumbInfo.Mode().Perm())
	}

	cm.SaveFingerprint(key, CachedFingerprint{Width: 100, Height: 80})
	if err := cm.FlushFingerprints(); err != nil {
		t.Fatal(err)
	}
	fpInfo, err := os.Stat(filepath.Join(cm.CacheDir(), fingerprintsFile))
	if err != nil {
		t.Fatal(err)
	}
	if fpInfo.Mode().Perm()&0077 != 0 {
		t.Fatalf("fingerprint file is readable by other users: %o", fpInfo.Mode().Perm())
	}
}

func TestOldFingerprintSchemaIsIgnored(t *testing.T) {
	cm := GetCacheManager()
	key := GenerateCacheKey("schema_test", "/path/to/schema.jpg", 1024, time.Now())
	cm.SaveFingerprint(key, CachedFingerprint{Width: 100, Height: 80})
	cm.mu.Lock()
	fp := cm.fingerprints[key]
	fp.Version = 0
	cm.fingerprints[key] = fp
	cm.mu.Unlock()
	if _, ok := cm.GetFingerprint(key); ok {
		t.Fatal("expected old-schema fingerprint to be ignored")
	}
}

// TestPruneCacheRemovesStaleTmpFiles covers the cleanup of write-temporary
// leftovers: a killed process can leave *.tmp files in the cache directory
// that regular thumbnail pruning never removed, while they still counted
// toward the size cap. The fresh .tmp must survive (live-writer guard) and
// regular thumbnails must be untouched.
func TestPruneCacheRemovesStaleTmpFiles(t *testing.T) {
	cm := GetCacheManager()
	if err := cm.ClearCache(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cm.ClearCache() })

	dir := cm.CacheDir()
	stale := filepath.Join(dir, "stale-thumb.jpg.tmp")
	fresh := filepath.Join(dir, "fresh-thumb.jpg.tmp")
	key := GenerateCacheKey("tmp_test", "/path/to/a.jpg", 10, time.Now())
	thumb := filepath.Join(dir, key+".jpg")
	for _, p := range []string{stale, fresh, thumb} {
		if err := os.WriteFile(p, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	result, err := cm.PruneCache(0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("expected stale tmp to be pruned, stat err: %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Fatalf("expected fresh tmp kept as live-writer guard, stat err: %v", err)
	}
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("expected regular thumbnail untouched, stat err: %v", err)
	}
	if result.RemovedThumbnails < 1 || result.FreedBytes < 1 {
		t.Fatalf("expected stale tmp counted in prune result, got %+v", result)
	}
}

// TestCachePruneClearFlushConcurrency hammers PruneCache, ClearCache,
// FlushFingerprints, SaveFingerprint, and thumbnail readers concurrently.
// Every fingerprints.json writer must hold fsMu before mu (the lock order
// documented on CacheManager); PruneCache's eviction phase taking only mu
// used to race ClearCache's directory sweep and could rewrite
// fingerprints.json after the user cleared the cache. A lock-order
// violation deadlocks and trips the test timeout; -race flags any
// unprotected shared state.
func TestCachePruneClearFlushConcurrency(t *testing.T) {
	cm := GetCacheManager()

	// Keep a small, bounded key space so flushes stay tiny and the stress
	// loop stays fast.
	const iterations = 40
	var seq atomic.Int64
	run := func(fn func()) {
		for i := 0; i < iterations; i++ {
			fn()
		}
	}

	var wg sync.WaitGroup
	start := func(fn func()) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			run(fn)
		}()
	}

	start(func() {
		_, _ = cm.PruneCache(0, 0)
	})
	start(func() {
		_ = cm.ClearCache()
	})
	start(func() {
		_ = cm.FlushFingerprints()
	})
	start(func() {
		n := seq.Add(1) % 64
		cm.SaveFingerprint(GenerateCacheKey("stress", fmt.Sprint(n), 1, time.Now()), CachedFingerprint{Width: 1, Height: 1})
	})
	start(func() {
		cm.HasThumbnail("stress-missing")
		_, _ = cm.GetThumbnail("stress-missing")
	})
	wg.Wait()

	// Deterministic tail check: after a successful clear with no concurrent
	// writers, no fingerprint file may linger on disk.
	if err := cm.ClearCache(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cm.CacheDir(), fingerprintsFile)); !os.IsNotExist(err) {
		t.Fatalf("expected no fingerprints file after clear, stat err: %v", err)
	}
}
