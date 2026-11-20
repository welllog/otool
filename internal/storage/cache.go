package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cespare/xxhash/v2"
)

const (
	cacheSubDir      = "otool/thumbnails"
	fingerprintsFile = "fingerprints.json"

	// DefaultCacheMaxBytes, DefaultCacheMaxAge and DefaultCacheMaxFingerprints
	// keep the local thumbnail and fingerprint cache bounded without requiring
	// a settings dialog. Fingerprints are deliberately small and expensive to
	// regenerate, so they are capped by count rather than tied to thumbnails.
	DefaultCacheMaxBytes        int64 = 512 << 20
	DefaultCacheMaxAge                = 30 * 24 * time.Hour
	DefaultCacheMaxFingerprints       = 50000

	// cacheSchemaVersion invalidates fingerprints produced by older analysis
	// algorithms (for example a changed pHash/orientation implementation).
	cacheSchemaVersion = 2

	// staleTmpCleanupAge gates the removal of leftover write-temporary files
	// in PruneCache. A killed process can leave *.tmp files behind; they are
	// far older than any live writer's WriteFile→Rename window, so deleting
	// them cannot disturb an in-flight save.
	staleTmpCleanupAge = time.Hour
)

// CachedFingerprint holds cached analysis metadata for an image to avoid re-decoding.
type CachedFingerprint struct {
	Version      int       `json:"version,omitempty"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Format       string    `json:"format"`
	ContentHash  uint64    `json:"content_hash"`
	PHash        uint64    `json:"phash"`
	DHash        uint64    `json:"dhash"`
	BlurScore    float64   `json:"blur_score"`
	HasEXIF      bool      `json:"has_exif"`
	CameraModel  string    `json:"camera_model,omitempty"`
	DateTimeOrig time.Time `json:"date_time_orig,omitempty"`
	LastUsed     time.Time `json:"last_used,omitempty"`
}

// CacheStats is a snapshot of the on-disk cache.
type CacheStats struct {
	TotalBytes       int64
	FileCount        int
	ThumbnailCount   int
	FingerprintCount int
}

// PruneResult reports what an automatic or manual prune removed.
type PruneResult struct {
	RemovedThumbnails   int
	RemovedFingerprints int
	FreedBytes          int64
}

type CacheManager struct {
	mu           sync.RWMutex // protects fingerprints, fpDirty, stats
	fsMu         sync.RWMutex // serializes cache files; acquire before mu, never after
	cacheDir     string
	fingerprints map[string]CachedFingerprint
	fpLoaded     bool
	fpDirty      bool
	stats        CacheStats
	statsValid   bool
}

var (
	defaultCacheMgr *CacheManager
	cacheOnce       sync.Once
)

func defaultCacheDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, cacheSubDir)
}

func GetCacheManager() *CacheManager {
	cacheOnce.Do(func() {
		dir := defaultCacheDir()
		_ = os.MkdirAll(dir, 0700)
		// Tighten permissions on directories created by older versions.
		_ = os.Chmod(dir, 0700)
		defaultCacheMgr = &CacheManager{
			cacheDir:     dir,
			fingerprints: make(map[string]CachedFingerprint),
		}
		defaultCacheMgr.loadFingerprints()
		defaultCacheMgr.mu.Lock()
		defaultCacheMgr.evictFingerprintsLocked(DefaultCacheMaxFingerprints)
		defaultCacheMgr.mu.Unlock()
	})
	return defaultCacheMgr
}

// GenerateCacheKey generates a 64-bit hex hash key for a unique file version.
// The optional etag is used by remote storage drivers whose ModTime precision
// can be too coarse to detect an in-place content replacement.
func GenerateCacheKey(storageID, path string, size int64, modTime time.Time, etag ...string) string {
	raw := fmt.Sprintf("%s:%s:%d:%d", storageID, path, size, modTime.UnixNano())
	if len(etag) > 0 && etag[0] != "" {
		raw += ":" + etag[0]
	}
	h := xxhash.Sum64String(raw)
	return fmt.Sprintf("%016x", h)
}

func (c *CacheManager) CacheDir() string {
	return c.cacheDir
}

// OpenCacheFolder opens the thumbnail cache directory in the OS file manager.
func (c *CacheManager) OpenCacheFolder() error {
	dir := c.cacheDir
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0700)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", dir)
	case "windows":
		cmd = exec.Command("explorer.exe", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

func (c *CacheManager) getFilePath(key string) string {
	// Prefer .jpg, fallback to .webp
	jpgPath := filepath.Join(c.cacheDir, key+".jpg")
	if _, err := os.Stat(jpgPath); err == nil {
		return jpgPath
	}
	webpPath := filepath.Join(c.cacheDir, key+".webp")
	if _, err := os.Stat(webpPath); err == nil {
		return webpPath
	}
	return jpgPath
}

// HasThumbnail checks whether a cached thumbnail exists without reading its
// contents. The dedup scanner only needs this lightweight existence check.
func (c *CacheManager) HasThumbnail(key string) bool {
	c.fsMu.RLock()
	defer c.fsMu.RUnlock()

	for _, ext := range []string{".jpg", ".webp"} {
		if _, err := os.Stat(filepath.Join(c.cacheDir, key+ext)); err == nil {
			return true
		}
	}
	return false
}

// GetThumbnail reads cached thumbnail data if it exists.
func (c *CacheManager) GetThumbnail(key string) ([]byte, bool) {
	c.fsMu.RLock()
	defer c.fsMu.RUnlock()

	p := c.getFilePath(key)
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	now := time.Now()
	_ = os.Chtimes(p, now, now)
	return data, true
}

// SaveThumbnail saves thumbnail bytes to disk.
func (c *CacheManager) SaveThumbnail(key string, data []byte) error {
	c.fsMu.Lock()
	defer c.fsMu.Unlock()

	if err := os.MkdirAll(c.cacheDir, 0700); err != nil {
		return err
	}
	p := filepath.Join(c.cacheDir, key+".jpg")
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmp, p); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	// Remove a legacy .webp thumbnail for the same key so a key has at most
	// one on-disk thumbnail.
	_ = os.Remove(filepath.Join(c.cacheDir, key+".webp"))
	_ = os.Chmod(p, 0600)

	c.mu.Lock()
	c.statsValid = false
	c.mu.Unlock()
	return nil
}

// RemoveThumbnail removes a specific thumbnail file and its fingerprint entry.
func (c *CacheManager) RemoveThumbnail(key string) error {
	c.fsMu.Lock()
	defer c.fsMu.Unlock()

	var firstErr error
	for _, ext := range []string{".jpg", ".webp"} {
		p := filepath.Join(c.cacheDir, key+ext)
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}

	c.mu.Lock()
	if _, ok := c.fingerprints[key]; ok {
		delete(c.fingerprints, key)
		c.fpDirty = true
	}
	c.statsValid = false
	c.mu.Unlock()
	return firstErr
}

func (c *CacheManager) loadFingerprints() {
	fpPath := filepath.Join(c.cacheDir, fingerprintsFile)
	data, err := os.ReadFile(fpPath)
	if err != nil {
		c.fpLoaded = true
		return
	}
	var m map[string]CachedFingerprint
	if err := json.Unmarshal(data, &m); err == nil && m != nil {
		c.fingerprints = m
	}
	c.fpLoaded = true
}

// GetFingerprint retrieves cached fingerprint info if exists. Cache hits also
// refresh the LRU timestamp used by the fingerprint-count cap.
func (c *CacheManager) GetFingerprint(key string) (CachedFingerprint, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	fp, ok := c.fingerprints[key]
	if !ok {
		return CachedFingerprint{}, false
	}
	if fp.Version != cacheSchemaVersion {
		delete(c.fingerprints, key)
		c.fpDirty = true
		c.statsValid = false
		return CachedFingerprint{}, false
	}
	if now := time.Now(); fp.LastUsed.IsZero() || now.Sub(fp.LastUsed) > time.Hour {
		fp.LastUsed = now
		c.fingerprints[key] = fp
		c.fpDirty = true
	}
	return fp, true
}

// evictFingerprintsLocked removes the oldest-used entries until at most
// target remain. The caller must hold c.mu.
func (c *CacheManager) evictFingerprintsLocked(target int) int {
	if target < 0 {
		target = 0
	}
	if len(c.fingerprints) <= target {
		return 0
	}
	keys := make([]string, 0, len(c.fingerprints))
	for key := range c.fingerprints {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return c.fingerprints[keys[i]].LastUsed.Before(c.fingerprints[keys[j]].LastUsed)
	})
	removed := 0
	for _, key := range keys[:len(keys)-target] {
		delete(c.fingerprints, key)
		removed++
	}
	if removed > 0 {
		c.fpDirty = true
		c.statsValid = false
	}
	return removed
}

// SaveFingerprint caches fingerprint info for a key. The in-memory index is
// bounded during scanning, not only when PruneCache eventually runs.
func (c *CacheManager) SaveFingerprint(key string, fp CachedFingerprint) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict in batches so a huge scan cannot grow the map without bound and
	// does not need to sort on every single insertion.
	if len(c.fingerprints) >= DefaultCacheMaxFingerprints {
		target := DefaultCacheMaxFingerprints - 1024
		if target < 0 {
			target = 0
		}
		c.evictFingerprintsLocked(target)
	}
	fp.Version = cacheSchemaVersion
	fp.LastUsed = time.Now()
	c.fingerprints[key] = fp
	c.fpDirty = true
	c.statsValid = false
}

// UpdateFingerprintContentHash stores a late-computed full-content hash into
// an existing fingerprint entry. Similar-mode scans only learn the hash after
// clustering; persisting it lets the next scan reuse it instead of re-reading
// group members from the store. It never creates entries, so a fingerprint
// evicted mid-scan is simply rebuilt on the next full analysis.
func (c *CacheManager) UpdateFingerprintContentHash(key string, hash uint64) bool {
	if key == "" || hash == 0 {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	fp, ok := c.fingerprints[key]
	if !ok || fp.Version != cacheSchemaVersion {
		return false
	}
	fp.ContentHash = hash
	c.fingerprints[key] = fp
	c.fpDirty = true
	c.statsValid = false
	return true
}

// FlushFingerprints writes memory fingerprints to disk. Fingerprint file
// writes share fsMu with thumbnail files so ClearCache cannot race a flush.
func (c *CacheManager) FlushFingerprints() error {
	c.fsMu.Lock()
	defer c.fsMu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.flushFingerprintsLocked()
}

func (c *CacheManager) flushFingerprintsLocked() error {
	if !c.fpDirty {
		return nil
	}
	data, err := json.Marshal(c.fingerprints)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(c.cacheDir, 0700); err != nil {
		return err
	}
	fpPath := filepath.Join(c.cacheDir, fingerprintsFile)
	tmpPath := fpPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, fpPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Chmod(fpPath, 0600)
	c.fpDirty = false
	return nil
}

// GetCacheStats returns the last known cache stats. If no valid snapshot is
// available it performs one scan; callers on the UI thread should prefer
// RefreshCacheStats in a background goroutine.
func (c *CacheManager) GetCacheStats() (CacheStats, error) {
	c.mu.RLock()
	if c.statsValid {
		stats := c.stats
		c.mu.RUnlock()
		return stats, nil
	}
	c.mu.RUnlock()

	return c.RefreshCacheStats()
}

// RefreshCacheStats scans the cache directory and refreshes the cached stats.
func (c *CacheManager) RefreshCacheStats() (CacheStats, error) {
	c.fsMu.RLock()
	entries, err := os.ReadDir(c.cacheDir)
	c.fsMu.RUnlock()
	if err != nil {
		if os.IsNotExist(err) {
			c.mu.Lock()
			c.stats = CacheStats{}
			c.statsValid = true
			c.mu.Unlock()
			return CacheStats{}, nil
		}
		return CacheStats{}, err
	}

	var stats CacheStats
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		stats.TotalBytes += info.Size()
		stats.FileCount++
		if _, ok := thumbnailKeyFromName(entry.Name()); ok {
			stats.ThumbnailCount++
		}
	}

	c.mu.Lock()
	stats.FingerprintCount = len(c.fingerprints)
	c.stats = stats
	c.statsValid = true
	c.mu.Unlock()
	return stats, nil
}

// ClearCache deletes every cache file and resets in-memory fingerprint state.
func (c *CacheManager) ClearCache() error {
	c.fsMu.Lock()
	defer c.fsMu.Unlock()

	entries, err := os.ReadDir(c.cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.mu.Lock()
			c.fingerprints = make(map[string]CachedFingerprint)
			c.fpDirty = false
			c.stats = CacheStats{}
			c.statsValid = true
			c.mu.Unlock()
			return nil
		}
		c.mu.Lock()
		c.statsValid = false
		c.mu.Unlock()
		return err
	}

	var firstErr error
	for _, entry := range entries {
		p := filepath.Join(c.cacheDir, entry.Name())
		var removeErr error
		if entry.IsDir() {
			removeErr = os.RemoveAll(p)
		} else {
			removeErr = os.Remove(p)
		}
		if removeErr != nil && !os.IsNotExist(removeErr) && firstErr == nil {
			firstErr = removeErr
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	// Always reset the in-memory index. If an on-disk file could not be
	// removed, flush an empty fingerprints file so old entries cannot be
	// resurrected on the next launch, and leave stats marked stale.
	c.fingerprints = make(map[string]CachedFingerprint)
	c.fpDirty = false
	if firstErr != nil {
		c.fpDirty = true
		c.statsValid = false
		if flushErr := c.flushFingerprintsLocked(); flushErr != nil {
			return firstErr
		}
		return firstErr
	}
	c.stats = CacheStats{}
	c.statsValid = true
	return nil
}

func thumbnailKeyFromName(name string) (string, bool) {
	ext := strings.ToLower(filepath.Ext(name))
	if ext != ".jpg" && ext != ".webp" {
		return "", false
	}
	key := strings.TrimSuffix(name, filepath.Ext(name))
	if key == "" {
		return "", false
	}
	return key, true
}

type fileAccessEntry struct {
	path    string
	size    int64
	modTime time.Time
}

// PruneCache ensures cache does not exceed maxBytes and removes files older than maxAge.
// Fingerprints are intentionally independent from thumbnails: pruning a
// thumbnail must not throw away an expensive-to-compute fingerprint. The
// fingerprint map is bounded separately by DefaultCacheMaxFingerprints.
func (c *CacheManager) PruneCache(maxBytes int64, maxAge time.Duration) (PruneResult, error) {
	var result PruneResult

	c.fsMu.RLock()
	entries, err := os.ReadDir(c.cacheDir)
	c.fsMu.RUnlock()
	if err != nil && !os.IsNotExist(err) {
		return result, err
	}

	var (
		list      []fileAccessEntry
		expired   []fileAccessEntry
		totalSize int64
	)
	if err == nil {
		now := time.Now()
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			info, infoErr := e.Info()
			if infoErr != nil {
				continue
			}

			name := e.Name()
			if name == fingerprintsFile {
				if info.Mode().Perm() != 0600 {
					_ = os.Chmod(filepath.Join(c.cacheDir, name), 0600)
				}
				totalSize += info.Size()
				continue
			}
			// Interrupted writes (a killed process) leave *.tmp files behind
			// that regular thumbnail pruning can never remove, even though
			// they count toward the size cap. Delete them through the same
			// expired list, guarded by age so an in-flight WriteFile→Rename
			// is never disturbed.
			if strings.HasSuffix(name, ".tmp") && now.Sub(info.ModTime()) > staleTmpCleanupAge {
				tmpPath := filepath.Join(c.cacheDir, name)
				if info.Mode().Perm() != 0600 {
					_ = os.Chmod(tmpPath, 0600)
				}
				totalSize += info.Size()
				expired = append(expired, fileAccessEntry{path: tmpPath, size: info.Size(), modTime: info.ModTime()})
				continue
			}
			if _, ok := thumbnailKeyFromName(name); !ok {
				totalSize += info.Size()
				continue
			}

			fullPath := filepath.Join(c.cacheDir, name)
			if info.Mode().Perm() != 0600 {
				_ = os.Chmod(fullPath, 0600)
			}
			entry := fileAccessEntry{path: fullPath, size: info.Size(), modTime: info.ModTime()}
			totalSize += entry.size
			if maxAge > 0 && now.Sub(entry.modTime) > maxAge {
				expired = append(expired, entry)
				continue
			}
			list = append(list, entry)
		}
	}

	// Delete expired files and enforce the size cap in a short write-locked
	// section; directory scanning/stat work above must not block thumbnail
	// readers for the whole cache.
	c.fsMu.Lock()
	for _, item := range expired {
		if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
			list = append(list, item)
			continue
		}
		result.RemovedThumbnails++
		result.FreedBytes += item.size
		totalSize -= item.size
	}
	if maxBytes > 0 && totalSize > maxBytes {
		sort.Slice(list, func(i, j int) bool {
			return list[i].modTime.Before(list[j].modTime)
		})
		for _, item := range list {
			if totalSize <= maxBytes {
				break
			}
			if err := os.Remove(item.path); err != nil {
				continue
			}
			result.RemovedThumbnails++
			result.FreedBytes += item.size
			totalSize -= item.size
		}
	}
	c.fsMu.Unlock()

	// Bound the fingerprint index independently. Hold fsMu through the map
	// update and flush so a concurrent Flush/Clear cannot resurrect old data.
	// Lock order matches FlushFingerprints/ClearCache: fsMu first, then mu.
	// Taking only mu would let the flush race ClearCache's directory sweep
	// (which deletes files before it takes mu) and rewrite fingerprints.json
	// after the user cleared the cache.
	c.fsMu.Lock()
	defer c.fsMu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, fp := range c.fingerprints {
		if fp.Version != cacheSchemaVersion {
			delete(c.fingerprints, key)
			result.RemovedFingerprints++
			c.fpDirty = true
		}
	}
	if removed := c.evictFingerprintsLocked(DefaultCacheMaxFingerprints); removed > 0 {
		result.RemovedFingerprints += removed
	}
	if c.fpDirty {
		if flushErr := c.flushFingerprintsLocked(); flushErr != nil {
			return result, flushErr
		}
	}
	c.statsValid = false
	return result, nil
}
