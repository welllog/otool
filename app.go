package main

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/otool/internal"
	"net/url"
)

// App struct
type App struct {
	ctx    context.Context
	logger logger.Logger
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		logger: logger.NewDefaultLogger(),
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

func (a *App) OpenSSLAesEnc(in, secret string) (string, error) {
	return internal.OpenSSLAesEncToStr(in, secret)
}

func (a *App) OpenSSLAesDec(in, secret string) (string, error) {
	return internal.OpenSSlAesDecToStr(in, secret)
}

func (a *App) Md5(in string) string {
	h := md5.Sum(internal.StringToBytes(in))
	return hex.EncodeToString(h[:])
}

func (a *App) Sha1(in string) string {
	h := sha1.Sum(internal.StringToBytes(in))
	return hex.EncodeToString(h[:])
}

func (a *App) Sha256(in string) string {
	h := sha256.Sum256(internal.StringToBytes(in))
	return hex.EncodeToString(h[:])
}

func (a *App) Base64Enc(in string) string {
	return base64.StdEncoding.EncodeToString(internal.StringToBytes(in))
}

func (a *App) Base64Dec(in string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return "", err
	}
	return internal.BytesToString(b), nil
}

func (a *App) UrlEnc(in string) string {
	return url.QueryEscape(in)
}

func (a *App) UrlDec(in string) (string, error) {
	return url.QueryUnescape(in)
}

func (a *App) warnAlert(msg string) {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:         runtime.WarningDialog,
		Title:        "warn",
		Message:      msg,
		CancelButton: "关闭",
	})
}
