package storage

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type LocalStorage struct {
	id string
}

func NewLocalStorage(id string) *LocalStorage {
	if id == "" {
		id = "local"
	}
	return &LocalStorage{id: id}
}

func (s *LocalStorage) Driver() string {
	return DriverLocal
}

func isJunkDirName(name string) bool {
	lower := strings.ToLower(name)
	switch lower {
	case TrashDirName, ".trash", ".trashes", "#recycle", "@recycle", "@eadir", ".@__thumb", "$recycle.bin":
		return true
	}
	return strings.HasPrefix(lower, ".trash-")
}

func isTrashDirName(name string) bool {
	return isJunkDirName(name)
}

func isUnderDir(child, parent string) bool {
	child = filepath.Clean(child)
	parent = filepath.Clean(parent)
	return child == parent || strings.HasPrefix(child, parent+string(os.PathSeparator))
}

func isSystemTrashPath(p string) bool {
	clean := filepath.Clean(p)
	home, err := os.UserHomeDir()
	if err == nil {
		trashRoots := []string{
			filepath.Join(home, ".Trash"),
			filepath.Join(home, ".local", "share", "Trash"),
		}
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			trashRoots = append(trashRoots, filepath.Join(xdg, "Trash"))
		}
		for _, trash := range trashRoots {
			if clean == trash || strings.HasPrefix(clean, trash+string(os.PathSeparator)) {
				return true
			}
		}
	}
	if runtime.GOOS == "windows" {
		if vol := filepath.VolumeName(clean); vol != "" {
			recycle := filepath.Join(vol, "$Recycle.Bin")
			if clean == recycle || strings.HasPrefix(clean, recycle+string(os.PathSeparator)) {
				return true
			}
		}
	}
	// External volumes on macOS and Linux keep trash in a volume-root
	// directory rather than in the user's home trash.
	if runtime.GOOS == "darwin" && strings.HasPrefix(clean, "/Volumes/") {
		rest := strings.TrimPrefix(clean, "/Volumes/")
		volume := strings.SplitN(rest, string(os.PathSeparator), 2)[0]
		if volume != "" {
			trash := filepath.Join("/Volumes", volume, ".Trashes")
			if clean == trash || strings.HasPrefix(clean, trash+string(os.PathSeparator)) {
				return true
			}
		}
	}
	return false
}

func (s *LocalStorage) List(ctx context.Context, dir string, recursive bool, onFile func(item FileItem) error) error {
	if strings.TrimSpace(dir) == "" {
		return fmt.Errorf("目标目录不能为空")
	}
	cleanDir := filepath.Clean(dir)
	if resolved, err := filepath.EvalSymlinks(cleanDir); err == nil {
		cleanDir = resolved
	}
	cacheDir := filepath.Clean(defaultCacheDir())
	if fi, err := os.Stat(cleanDir); err != nil {
		return fmt.Errorf("目标目录不存在或无法访问: %w", err)
	} else if !fi.IsDir() {
		return fmt.Errorf("目标路径不是文件夹: %s", cleanDir)
	}

	var skippedDirs []string
	skippedFiles := 0
	walkErr := filepath.WalkDir(cleanDir, func(path string, d fs.DirEntry, err error) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err != nil {
			if path == cleanDir {
				return err
			}
			if d != nil && d.IsDir() {
				// Remember the unreadable directory instead of silently
				// returning a partial list that looks complete.
				skippedDirs = append(skippedDirs, path)
				return filepath.SkipDir
			}
			skippedFiles++
			return nil
		}

		if d.IsDir() {
			// Never rescan our own quarantine directory, OS/NAS trash, or the
			// application's thumbnail cache: otherwise deleted photos or cached
			// previews become "duplicates" again on the next scan.
			if isJunkDirName(d.Name()) || isSystemTrashPath(path) || isUnderDir(path, cacheDir) {
				return filepath.SkipDir
			}
			if !recursive && path != cleanDir {
				return filepath.SkipDir
			}
			return nil
		}

		if !IsImageFile(d.Name()) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			skippedFiles++
			return nil
		}

		item := FileItem{
			Path:      path,
			Name:      d.Name(),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsDir:     false,
			StorageID: s.id,
		}

		return onFile(item)
	})
	if walkErr != nil {
		return walkErr
	}
	if len(skippedDirs) > 0 || skippedFiles > 0 {
		return &PartialListError{SkippedPaths: skippedDirs, SkippedFiles: skippedFiles}
	}
	return nil
}

func (s *LocalStorage) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (s *LocalStorage) OpenRange(ctx context.Context, path string, offset, length int64) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	if offset > 0 {
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			return nil, err
		}
	}

	var r io.Reader = f
	if length > 0 {
		r = io.LimitReader(f, length)
	}

	return &rangeReadCloser{Reader: r, Closer: f}, nil
}

func (s *LocalStorage) Stat(ctx context.Context, path string) (FileItem, error) {
	if err := ctx.Err(); err != nil {
		return FileItem{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return FileItem{}, err
	}
	return FileItem{
		Path:      path,
		Name:      info.Name(),
		Size:      info.Size(),
		ModTime:   info.ModTime(),
		IsDir:     info.IsDir(),
		StorageID: s.id,
	}, nil
}

func (s *LocalStorage) CheckPermission(ctx context.Context, targetDir string) (writable bool, deletable bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, false, err
	}
	if fi, err := os.Stat(targetDir); err != nil || !fi.IsDir() {
		return false, false, fmt.Errorf("目标目录不存在或不是文件夹: %s", targetDir)
	}

	probePath := filepath.Join(targetDir, fmt.Sprintf(".otool_perm_test_%d.tmp", time.Now().UnixNano()))

	// Test write
	f, err := os.OpenFile(probePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return false, false, fmt.Errorf("缺少写入权限: %w", err)
	}
	writable = true
	_ = f.Close()

	// Test delete
	if err := os.Remove(probePath); err != nil {
		return true, false, fmt.Errorf("具有写入权限但缺少删除权限: %w", err)
	}
	deletable = true

	return true, true, nil
}

func (s *LocalStorage) MoveToTrash(ctx context.Context, path string) error {
	// Attempt OS Trash on macOS first
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err == nil {
			trashDir := filepath.Join(home, ".Trash")
			if fi, err := os.Stat(trashDir); err == nil && fi.IsDir() {
				base := filepath.Base(path)
				dest := filepath.Join(trashDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), base))
				if err := os.Rename(path, dest); err == nil {
					_ = os.Chmod(dest, 0600)
					return nil
				}
			}
		}
	}

	// Fallback to local .otool_trash subdirectory in the parent directory
	parentDir := filepath.Dir(path)
	trashDir := filepath.Join(parentDir, TrashDirName)
	if err := os.MkdirAll(trashDir, 0700); err != nil {
		return fmt.Errorf("创建回收隔离目录失败: %w", err)
	}
	_ = os.Chmod(trashDir, 0700)

	base := filepath.Base(path)
	dest := filepath.Join(trashDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), base))
	if err := os.Rename(path, dest); err != nil {
		return err
	}
	_ = os.Chmod(dest, 0600)
	return nil
}

func (s *LocalStorage) Delete(ctx context.Context, path string) error {
	return os.Remove(path)
}

type rangeReadCloser struct {
	io.Reader
	io.Closer
}
