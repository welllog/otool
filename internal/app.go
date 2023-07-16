package internal

import (
	"context"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/golib/randz"
	"github.com/welllog/otool/internal/errx"
	"github.com/welllog/otool/internal/srvs"
)

// App struct
type App struct {
	ctx        context.Context
	EncryptSrv *srvs.Encrypt
	ImageSrv   *srvs.Image
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		EncryptSrv: &srvs.Encrypt{},
		ImageSrv:   &srvs.Image{},
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.ImageSrv.Ctx = ctx
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

func (a *App) warnAlert(msg string) {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:         runtime.WarningDialog,
		Title:        "warn",
		Message:      msg,
		CancelButton: "关闭",
	})
}
