package log

import (
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/welllog/olog"
)

const (
	_defCallerSkip = 2
)

type Logger struct {
	olog.Logger
}

type insideLogger struct{}

func New() *Logger {
	olog.SetLoggerOptions(
		olog.WithLoggerEncode(olog.PLAIN),
		olog.WithLoggerTimeFormat("2006/01/02 15:04:05"),
	)
	return &Logger{
		olog.GetLogger(),
	}
}

func (l *Logger) InsideLogger() logger.Logger {
	return insideLogger{}
}

func (i insideLogger) Print(message string) {
	olog.Log(olog.Record{
		Level:       olog.NOTICE,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		MsgOrFormat: message,
	})
}

func (i insideLogger) Trace(message string) {
	olog.Log(olog.Record{
		Level:       olog.TRACE,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		MsgOrFormat: message,
	})
}

func (i insideLogger) Debug(message string) {
	olog.Log(olog.Record{
		Level:       olog.DEBUG,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		MsgOrFormat: message,
	})
}

func (i insideLogger) Info(message string) {
	olog.Log(olog.Record{
		Level:       olog.INFO,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		MsgOrFormat: message,
	})
}

func (i insideLogger) Warning(message string) {
	olog.Log(olog.Record{
		Level:       olog.WARN,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		MsgOrFormat: message,
	})
}

func (i insideLogger) Error(message string) {
	olog.Log(olog.Record{
		Level:       olog.ERROR,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		MsgOrFormat: message,
	})
}

func (i insideLogger) Fatal(message string) {
	olog.Log(olog.Record{
		Level:       olog.FATAL,
		Caller:      olog.Disable,
		CallerSkip:  _defCallerSkip,
		Stack:       olog.Enable,
		MsgOrFormat: message,
	})
}
