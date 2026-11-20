package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
)

// withContext bounds a blocking gowebdav call by ctx. The library predates
// context support; the underlying HTTP request will finish at its 30s timeout
// if cancellation wins, but the scanner can stop waiting immediately.
func withContext[T any](ctx context.Context, fn func() (T, error)) (T, error) {
	type result struct {
		value T
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		value, err := fn()
		ch <- result{value: value, err: err}
	}()
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case r := <-ch:
		return r.value, r.err
	}
}

func withContextReadCloser(ctx context.Context, fn func() (io.ReadCloser, error)) (io.ReadCloser, error) {
	type result struct {
		value io.ReadCloser
		err   error
	}
	ch := make(chan result, 1)
	go func() {
		value, err := fn()
		ch <- result{value: value, err: err}
	}()
	select {
	case <-ctx.Done():
		// If the HTTP request completes after cancellation, close the stream
		// instead of leaking it.
		go func() {
			r := <-ch
			if r.value != nil {
				_ = r.value.Close()
			}
		}()
		return nil, ctx.Err()
	case r := <-ch:
		return r.value, r.err
	}
}

func withContextErr(ctx context.Context, fn func() error) error {
	_, err := withContext(ctx, func() (struct{}, error) {
		return struct{}{}, fn()
	})
	return err
}

func webdavETag(fi os.FileInfo) string {
	switch f := fi.(type) {
	case gowebdav.File:
		return f.ETag()
	case *gowebdav.File:
		return f.ETag()
	default:
		return ""
	}
}

type WebDAVStorage struct {
	id     string
	client *gowebdav.Client
	root   string
}

func NewWebDAVStorage(id, rawURL, user, password, root string) *WebDAVStorage {
	if id == "" {
		id = "webdav"
	}
	client := gowebdav.NewClient(rawURL, user, password)
	client.SetTimeout(30 * time.Second)

	if root == "" {
		root = "/"
	}
	if !strings.HasPrefix(root, "/") {
		root = "/" + root
	}

	return &WebDAVStorage{
		id:     id,
		client: client,
		root:   root,
	}
}

func (s *WebDAVStorage) Driver() string {
	return DriverWebDAV
}

func (s *WebDAVStorage) List(ctx context.Context, dir string, recursive bool, onFile func(item FileItem) error) error {
	if dir == "" {
		dir = s.root
	}
	if !strings.HasPrefix(dir, "/") {
		dir = "/" + dir
	}

	queue := []string{dir}
	var skippedDirs []string

	for len(queue) > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentDir := queue[0]
		queue = queue[1:]

		files, err := withContext(ctx, func() ([]os.FileInfo, error) {
			return s.client.ReadDir(currentDir)
		})
		if err != nil {
			// Cancellation is never a "partial directory" warning: return the
			// context error so callers can report a stopped scan.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return err
			}
			// A failure at the requested root must be visible: otherwise a typo
			// or auth problem looks like an empty photo library.
			if currentDir == dir {
				return fmt.Errorf("读取远程目录失败: %w", err)
			}
			// Remember inaccessible subdirectories so the UI can tell the
			// user that the scan was incomplete instead of silently hiding it.
			skippedDirs = append(skippedDirs, currentDir)
			continue
		}

		for _, f := range files {
			fullPath := path.Join(currentDir, f.Name())

			if f.IsDir() {
				if isJunkDirName(f.Name()) {
					continue
				}
				if recursive {
					queue = append(queue, fullPath)
				}
				continue
			}

			if !IsImageFile(f.Name()) {
				continue
			}

			etag := webdavETag(f)

			item := FileItem{
				Path:      fullPath,
				Name:      f.Name(),
				Size:      f.Size(),
				ModTime:   f.ModTime(),
				IsDir:     false,
				ETag:      etag,
				StorageID: s.id,
			}

			if err := onFile(item); err != nil {
				return err
			}
		}
	}

	if len(skippedDirs) > 0 {
		return &PartialListError{SkippedPaths: skippedDirs}
	}
	return nil
}

func (s *WebDAVStorage) Open(ctx context.Context, p string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rc, err := withContextReadCloser(ctx, func() (io.ReadCloser, error) {
		return s.client.ReadStream(p)
	})
	if err != nil {
		return nil, err
	}
	return newContextReadCloser(ctx, rc), nil
}

func (s *WebDAVStorage) OpenRange(ctx context.Context, p string, offset, length int64) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rc, err := withContextReadCloser(ctx, func() (io.ReadCloser, error) {
		return s.client.ReadStreamRange(p, offset, length)
	})
	if err != nil {
		return nil, err
	}
	return newContextReadCloser(ctx, rc), nil
}

type contextReadCloser struct {
	ctx  context.Context
	rc   io.ReadCloser
	stop func() bool
}

func newContextReadCloser(ctx context.Context, rc io.ReadCloser) *contextReadCloser {
	r := &contextReadCloser{ctx: ctx, rc: rc}
	// Closing the body unblocks a Read that is already stuck in the HTTP
	// stack; contextReadCloser.Read only protects the entry to each Read.
	r.stop = context.AfterFunc(ctx, func() {
		_ = rc.Close()
	})
	return r
}

func (r *contextReadCloser) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.rc.Read(p)
}

func (r *contextReadCloser) Close() error {
	if r.stop != nil {
		r.stop()
	}
	return r.rc.Close()
}

func (s *WebDAVStorage) Stat(ctx context.Context, p string) (FileItem, error) {
	if err := ctx.Err(); err != nil {
		return FileItem{}, err
	}
	fi, err := withContext(ctx, func() (os.FileInfo, error) {
		return s.client.Stat(p)
	})
	if err != nil {
		return FileItem{}, err
	}

	etag := webdavETag(fi)

	return FileItem{
		Path:      p,
		Name:      fi.Name(),
		Size:      fi.Size(),
		ModTime:   fi.ModTime(),
		IsDir:     fi.IsDir(),
		ETag:      etag,
		StorageID: s.id,
	}, nil
}

func (s *WebDAVStorage) CheckPermission(ctx context.Context, targetDir string) (writable bool, deletable bool, err error) {
	if err := ctx.Err(); err != nil {
		return false, false, err
	}
	if targetDir == "" {
		targetDir = s.root
	}
	if !strings.HasPrefix(targetDir, "/") {
		targetDir = "/" + targetDir
	}

	probePath := path.Join(targetDir, fmt.Sprintf(".otool_perm_test_%d.tmp", time.Now().UnixNano()))

	// Test write
	err = withContextErr(ctx, func() error {
		return s.client.WriteStream(probePath, bytes.NewReader([]byte("test")), 0644)
	})
	if err != nil {
		return false, false, fmt.Errorf("WebDAV 目录不可写 (可能账号只有只读权限): %w", err)
	}
	writable = true

	// Test delete
	err = withContextErr(ctx, func() error {
		return s.client.Remove(probePath)
	})
	if err != nil {
		return true, false, fmt.Errorf("WebDAV 具有写入权限但缺少删除权限: %w", err)
	}
	deletable = true

	return true, true, nil
}

func (s *WebDAVStorage) MoveToTrash(ctx context.Context, p string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parent := path.Dir(p)
	trashDir := path.Join(parent, TrashDirName)

	if err := withContextErr(ctx, func() error {
		return s.client.MkdirAll(trashDir, 0700)
	}); err != nil {
		return fmt.Errorf("创建远程隔离目录失败: %w", err)
	}

	base := path.Base(p)
	dest := path.Join(trashDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), base))
	return withContextErr(ctx, func() error {
		return s.client.Rename(p, dest, true)
	})
}

func (s *WebDAVStorage) Delete(ctx context.Context, p string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return withContextErr(ctx, func() error {
		return s.client.Remove(p)
	})
}
