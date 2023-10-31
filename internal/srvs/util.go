package srvs

import (
	"context"
	"strings"

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

func notify(ctx context.Context, e NotifyEvent) {
	runtime.EventsEmit(ctx, "notify", e)
}

var fileNameReplacer = strings.NewReplacer(
	"\\", "-",
	":", "-",
	"*", "-",
	"?", "-",
	"\"", "-",
	"<", "-",
	">", "-",
	"|", "-",
	"/", "-",
)

func fileName(str string) string {
	str = strings.Trim(str, " ")
	// \ : * ? " < > | /
	return fileNameReplacer.Replace(str)
}
