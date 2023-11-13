package internal

import (
	"context"
	"os"
	"path/filepath"

	stdrt "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/golib/goz"
	"github.com/welllog/golib/mapz"
	"github.com/welllog/golib/randz"
	"github.com/welllog/otool/internal/errx"
	"github.com/welllog/otool/internal/srvs"
)

// App struct
type App struct {
	version    string
	ctx        context.Context
	gow        *goz.Limiter
	EncryptSrv *srvs.Encrypt
	ImageSrv   *srvs.Image
}

// NewApp creates a new App application struct
func NewApp(version string) *App {
	gow := goz.NewLimiter(stdrt.GOMAXPROCS(0))
	return &App{
		version:    version,
		gow:        gow,
		EncryptSrv: &srvs.Encrypt{Gow: gow},
		ImageSrv:   &srvs.Image{Gow: gow, Kv: mapz.NewSafeKV[string, any](10)},
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.ImageSrv.Ctx = ctx
	a.EncryptSrv.Ctx = ctx
}

func (a *App) OnShutdown(ctx context.Context) {
	a.gow.Wait()
}

func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		ShowHiddenFiles: true,
	})
}

func (a *App) OpenDirectoryDialog() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		ShowHiddenFiles: true,
	})
}

func (a *App) SaveFileDialog() (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		ShowHiddenFiles:      true,
		CanCreateDirectories: true,
	})
}

func (a *App) DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		_ = errx.Log(err)
		return ""
	}
	desk := filepath.Join(home, "Desktop")
	f, err := os.Stat(desk)
	if err != nil || !f.IsDir() {
		return home
	}

	return desk
}

func (a *App) RandId() string {
	return randz.Id().Base32()
}

func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func (a *App) Version() string {
	return a.version
}

func (a *App) warnAlert(msg string) {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:         runtime.WarningDialog,
		Title:        "warn",
		Message:      msg,
		CancelButton: "关闭",
	})
}
