package main

import (
	"log"

	"github.com/welllog/otool/internal/app"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	AppName    = "otool"
	AppVersion = "v0.2.0"
)

func main() {
	application := app.NewApp()

	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:     AppName + " " + AppVersion,
			Width:     1080,
			Height:    760,
			MinWidth:  840,
			MinHeight: 600,
			Content:   ui.View(application.View),
		})
		application.SetWindow(win)
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

