package internal

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/otool/internal/srvs"
)

// App struct
type App struct {
	ctx        context.Context
	EncryptSrv *srvs.Encrypt
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		EncryptSrv: &srvs.Encrypt{},
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
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

func (a *App) warnAlert(msg string) {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:         runtime.WarningDialog,
		Title:        "warn",
		Message:      msg,
		CancelButton: "关闭",
	})
}
