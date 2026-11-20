package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

// webdavTrashServer fakes a WebDAV server for trash tests. Like real servers
// (nginx/Apache/Sabre), MKCOL on an existing collection returns 405; gowebdav
// must normalize that into success for repeated MoveToTrash calls to work.
type webdavTrashServer struct {
	mu    sync.Mutex
	dirs  map[string]bool
	files map[string]bool
	srv   *httptest.Server
}

func newWebdavTrashServer(t *testing.T, files map[string]bool) *webdavTrashServer {
	t.Helper()
	s := &webdavTrashServer{
		dirs:  map[string]bool{"/photo": true},
		files: files,
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *webdavTrashServer) handle(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := r.URL.Path
	switch r.Method {
	case "MKCOL":
		if s.dirs[p] {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.dirs[p] = true
		w.WriteHeader(http.StatusCreated)
	case "MOVE":
		if !s.files[p] {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		delete(s.files, p)
		if dest := r.Header.Get("Destination"); dest != "" {
			if u, err := url.Parse(dest); err == nil {
				s.files[u.Path] = true
			}
		}
		w.WriteHeader(http.StatusCreated)
	case "DELETE":
		if s.files[p] {
			delete(s.files, p)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	case "GET":
		if s.files[p] {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *webdavTrashServer) hasFile(p string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.files[p]
}

// TestWebdavMoveToTrashRepeatedly locks in the repeated-trash behavior: the
// quarantine directory already exists from the second call on, and MKCOL then
// returns 405, which gowebdav treats as success (requests.go normalizes
// 405 -> 201). If a future library upgrade or webdav.go change breaks that,
// every WebDAV cleanup after the first photo would fail.
func TestWebdavMoveToTrashRepeatedly(t *testing.T) {
	s := newWebdavTrashServer(t, map[string]bool{
		"/photo/a.jpg": true,
		"/photo/b.jpg": true,
		"/photo/c.jpg": true,
	})

	store := NewWebDAVStorage("trash_test", s.srv.URL, "u", "p", "/")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	for _, p := range []string{"/photo/a.jpg", "/photo/b.jpg", "/photo/c.jpg"} {
		if err := store.MoveToTrash(ctx, p); err != nil {
			t.Fatalf("MoveToTrash(%s) failed after earlier calls: %v", p, err)
		}
		if s.hasFile(p) {
			t.Fatalf("%s still present after MoveToTrash", p)
		}
	}

	// Fail-closed sanity check: trashing a file that is no longer at its
	// original path must return an error, not a silent success.
	if err := store.MoveToTrash(ctx, "/photo/a.jpg"); err == nil {
		t.Fatal("trashing a nonexistent file should fail")
	}
}
