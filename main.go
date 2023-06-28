package main

import (
	"embed"

	"github.com/wailsapp/wails/v2/pkg/logger"
	"github.com/wailsapp/wails/v2/pkg/options/linux"
	"github.com/wailsapp/wails/v2/pkg/options/mac"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"github.com/welllog/otool/internal"
	"github.com/welllog/otool/internal/log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

const version = "v0.0.2"

//go:embed all:frontend/build
var assets embed.FS

func main() {
	customLogger := log.New()
	// Create an instance of the app structure
	app := internal.NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:             "otool",
		Width:             1024,
		Height:            768,
		Frameless:         false, // 无边框应用
		StartHidden:       false, // 启动时隐藏窗口
		HideWindowOnClose: false, // 关闭时隐藏窗口
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		LogLevel:         logger.DEBUG,
		Logger:           customLogger.InsideLogger(),
		OnStartup:        app.Startup,
		Bind: []interface{}{
			app,
			app.EncryptSrv,
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false, // 网页透明
			WindowIsTranslucent:  true,  // 窗口半透明
			DisableWindowIcon:    false, // 禁用窗口图标 true将删除标题栏左上角的图标
		},
		Mac: &mac.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  true,
			About: &mac.AboutInfo{
				Title:   "otool version",
				Message: version,
			},
		},
		Linux: &linux.Options{
			WindowIsTranslucent: true,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
