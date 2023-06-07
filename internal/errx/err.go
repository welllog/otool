package errx

import (
	"fmt"

	"github.com/welllog/olog"
)

func Logf(format string, a ...any) error {
	olog.Log(olog.Record{
		Level:       olog.ERROR,
		CallerSkip:  1,
		MsgOrFormat: format,
		MsgArgs:     a,
	})
	return fmt.Errorf(format, a...)
}

func Log(err error) error {
	if err != nil {
		olog.Log(olog.Record{
			Level:       olog.ERROR,
			CallerSkip:  1,
			MsgOrFormat: err.Error(),
		})
	}
	return err
}

func LogStr(s string, err error) (string, error) {
	if err != nil {
		olog.Log(olog.Record{
			Level:       olog.ERROR,
			CallerSkip:  1,
			MsgOrFormat: err.Error(),
		})
	}
	return s, err
}
