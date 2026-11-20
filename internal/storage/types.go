package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
)

// TrashDirName is the in-tree quarantine directory used by storage drivers that
// cannot move files to the system trash (or for WebDAV).
const TrashDirName = ".otool_trash"

// PartialListError reports directories or files that were skipped during a
// scan while still allowing the readable part of the tree to be analyzed.
type PartialListError struct {
	SkippedPaths []string
	SkippedFiles int
}

func (e *PartialListError) Error() string {
	if len(e.SkippedPaths) == 0 && e.SkippedFiles == 0 {
		return "部分文件无法读取"
	}
	parts := make([]string, 0, 2)
	if len(e.SkippedPaths) == 1 {
		parts = append(parts, fmt.Sprintf("目录无法读取: %s", e.SkippedPaths[0]))
	} else if len(e.SkippedPaths) > 1 {
		parts = append(parts, fmt.Sprintf("%d 个目录无法读取（如 %s）", len(e.SkippedPaths), e.SkippedPaths[0]))
	}
	if e.SkippedFiles > 0 {
		parts = append(parts, fmt.Sprintf("%d 个文件无法读取", e.SkippedFiles))
	}
	return strings.Join(parts, "；")
}

// Supported image extensions
var ImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".jpe":  true,
	".jfif": true,
	".png":  true,
	".webp": true,
	".avif": true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	".heic": true,
	".heif": true,
	".raw":  true,
	".dng":  true,
	".cr2":  true,
	".nef":  true,
	".arw":  true,
}

// DecodableImageExtensions are formats for which this build has an image
// decoder registered. Exact mode can still hash files with other extensions,
// but similar-mode analysis needs a real decoder.
var DecodableImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".jpe":  true,
	".jfif": true,
	".png":  true,
	".webp": true,
	".avif": true,
	".gif":  true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
}

// IsImageFile returns whether the filename matches common image extensions.
func IsImageFile(name string) bool {
	idx := strings.LastIndex(name, ".")
	if idx <= 0 {
		return false
	}
	ext := strings.ToLower(name[idx:])
	return ImageExtensions[ext]
}

// IsDecodableImageFile reports whether similar-mode analysis can decode it.
func IsDecodableImageFile(name string) bool {
	idx := strings.LastIndex(name, ".")
	if idx <= 0 {
		return false
	}
	ext := strings.ToLower(name[idx:])
	return DecodableImageExtensions[ext]
}

// FileItem represents standard file metadata across local and remote storage.
type FileItem struct {
	Path      string    `json:"path"`
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	IsDir     bool      `json:"is_dir"`
	ETag      string    `json:"etag,omitempty"`
	StorageID string    `json:"storage_id,omitempty"`
}

// Storage is the unified storage driver interface.
type Storage interface {
	// Driver returns driver name ("local", "webdav", "s3")
	Driver() string

	// List streams files in dir to onFile callback. If recursive is true, scans all subdirectories.
	List(ctx context.Context, dir string, recursive bool, onFile func(item FileItem) error) error

	// Open opens the full file for reading.
	Open(ctx context.Context, path string) (io.ReadCloser, error)

	// OpenRange opens a partial range of the file.
	OpenRange(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error)

	// Stat retrieves metadata for a single file.
	Stat(ctx context.Context, path string) (FileItem, error)

	// CheckPermission runs a canary probe to test write and delete capabilities.
	CheckPermission(ctx context.Context, targetDir string) (writable bool, deletable bool, err error)

	// MoveToTrash moves the file to a safe trash / quarantine location instead of hard deleting.
	MoveToTrash(ctx context.Context, path string) error

	// Delete permanently deletes the file.
	Delete(ctx context.Context, path string) error
}
