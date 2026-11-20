package dedup

import (
	"runtime"
	"time"
)

// DedupMode specifies whether to find exact duplicates or perceptual near-duplicates.
type DedupMode int

const (
	ModeExact   DedupMode = 0 // 精确重复 (字节哈希)
	ModeSimilar DedupMode = 1 // 相似/连拍去重 (感知哈希 pHash/dHash)
)

// PhotoMeta represents the analyzed metadata, fingerprints, and quality score for a photo.
type PhotoMeta struct {
	ID           uint64    `json:"id"`
	StorageID    string    `json:"storage_id"`
	ETag         string    `json:"etag,omitempty"`
	Path         string    `json:"path"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	Format       string    `json:"format"`
	ContentHash  uint64    `json:"content_hash"` // xxhash for exact check
	PHash        uint64    `json:"phash"`        // Perception hash (64-bit)
	DHash        uint64    `json:"dhash"`        // Difference hash (64-bit)
	BlurScore    float64   `json:"blur_score"`   // Laplacian variance (sharpness)
	Score        float64   `json:"score"`        // Quality score (0~100)
	ScoreReasons []string  `json:"score_reasons"`
	HasEXIF      bool      `json:"has_exif"`
	CameraModel  string    `json:"camera_model,omitempty"`
	DateTimeOrig time.Time `json:"date_time_orig,omitempty"`
	CacheKey     string    `json:"cache_key"`
	Selected     bool      `json:"selected"` // UI state: whether user checked this to DELETE
}

// DuplicateGroup represents a cluster of exact duplicate or visually similar photos.
type DuplicateGroup struct {
	GroupID           string       `json:"group_id"`
	Mode              DedupMode    `json:"mode"`
	SimilarityPercent int          `json:"similarity_percent"` // 0~100%
	BestPickID        uint64       `json:"best_pick_id"`       // The photo recommended to KEEP
	Photos            []*PhotoMeta `json:"photos"`
	TotalWastedBytes  int64        `json:"total_wasted_bytes"` // Bytes saved if keeping BestPick and deleting rest
}

// ScanProgress carries real-time status updates from the scanning pipeline to the UI.
type ScanProgress struct {
	Phase           string // "discovering", "filtering", "analyzing", "clustering", "done"
	PhaseTitle      string // e.g. "正在扫描文件目录...", "正在计算图片指纹..."
	TotalDiscovered int    // Total files found so far
	TotalToAnalyze  int    // Candidate files for deep hash
	CurrentAnalyzed int    // Files analyzed so far
	SkippedCount    int    // Files that could not be read or decoded
	SkippedDirs     int    // Directories that could not be read
	// PreviewFailedGroups counts exact-mode groups whose representative could
	// not be decoded. Those photos are NOT skipped: they stay in the results
	// with their byte hash, only preview metadata and thumbnails are missing.
	PreviewFailedGroups int
	FoundGroups         int     // Number of duplicate groups found
	WastedBytes         int64   // Total duplicate bytes
	CurrentFile         string  // Currently processing filename
	Percent             float64 // Overall progress 0.0 ~ 1.0
	IsComplete          bool
	Error               error
}

// ScanOptions configures the dedup scan job.
type ScanOptions struct {
	StorageID       string
	TargetDir       string
	Recursive       bool
	Mode            DedupMode
	SimilarDistance int   // Hamming distance threshold (default 8, range 0~15)
	WorkerCount     int   // Concurrent processing workers (default min(4, NumCPU))
	MemoryBudget    int64 // Estimated decoded-image memory budget in bytes (default 512MB)
	PriorityQuality bool  // Prioritize high-resolution/original EXIF
	ReadOnly        bool  // Scan-only mode without delete permission
}

// DefaultScanOptions returns recommended default options.
func DefaultScanOptions() ScanOptions {
	return ScanOptions{
		Recursive:       true,
		Mode:            ModeSimilar,
		SimilarDistance: 8,
		WorkerCount:     defaultWorkerCount(),
		MemoryBudget:    512 << 20,
		PriorityQuality: true,
	}
}

func defaultWorkerCount() int {
	n := runtime.NumCPU()
	if n < 1 {
		return 1
	}
	if n > 4 {
		return 4
	}
	return n
}
