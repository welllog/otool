package srvs

import (
	"context"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type NotifyEvent struct {
	Info string `json:"info"`
	Type string `json:"type"`
}

type streamProgress struct {
	n     int64
	size  int64
	event string
	ctx   context.Context
}

func (s *streamProgress) Write(b []byte) (int, error) {
	s.n += int64(len(b))
	s.progress()
	return len(b), nil
}

func (s *streamProgress) progress() {
	p := s.n * 100 / s.size
	runtime.EventsEmit(s.ctx, s.event, p)
}

func (s *streamProgress) close() {
	runtime.EventsEmit(s.ctx, s.event, -1)
}

type countProgress struct {
	n     int
	count int
	event string
	ctx   context.Context
}

func (c *countProgress) Incr() {
	c.n++
	p := c.n * 100 / c.count
	runtime.EventsEmit(c.ctx, c.event, p)
}

func (c *countProgress) close() {
	runtime.EventsEmit(c.ctx, c.event, -1)
}

type taskRecord struct {
	progress countProgress
	errCh    chan error
	done     sync.WaitGroup
	name     string
}

func notify(ctx context.Context, e NotifyEvent) {
	runtime.EventsEmit(ctx, "notify", e)
}

var fileNameReplacer = strings.NewReplacer(
	"\\", "_",
	":", "_",
	"*", "_",
	"?", "_",
	"\"", "_",
	"<", "_",
	">", "_",
	"|", "_",
	"/", "_",
)

func fileName(str string) string {
	str = strings.Trim(str, " ")
	// \ : * ? " < > | /
	return fileNameReplacer.Replace(str)
}
