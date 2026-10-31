package app

import (
	"github.com/egoist/mygo/ui"
)

type NavItem struct {
	ID    int
	Title string
	Icon  *ui.SVG
}

var navItems = []NavItem{
	{ID: PageText, Title: "文本编解码", Icon: IconText},
	{ID: PageFile, Title: "文件编解码", Icon: IconFile},
	{ID: PageConversion, Title: "进制转换", Icon: IconConversion},
	{ID: PageImage, Title: "图片转换", Icon: IconImage},
	{ID: PageQRCode, Title: "二维码工具", Icon: IconQRCode},
}

func (a *App) sidebar(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).
		Width(220).
		Shrink(0).
		Background(t.Surface).
		Border(0, t.Border).
		Padding(16, 12).
		Justify(ui.SpaceBetween).
		Children(func() {
			// Top: Brand & Menu
			ui.Column(c).Gap(18).Children(func() {
				// Brand Header
				ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(4, 8).Children(func() {
					ui.Box(c).
						Size(32, 32).
						Radius(8).
						Background(ui.Hex("#2563eb")).
						Center().
						Children(func() {
							ui.Icon(c, IconLogo).Size(18, 18).TextColor(ui.RGB(255, 255, 255))
						})

					ui.Column(c).Gap(1).Children(func() {
						ui.Text(c, "OTOOL").FontSize(16).Bold().TextColor(t.Text)
						ui.Text(c, "工具箱").FontSize(10).TextColor(t.TextMuted)
					})
				})

				// Menu Items
				ui.Column(c).Gap(4).Children(func() {
					ui.Text(c, "功能模块").
						FontSize(10).
						Bold().
						TextColor(t.TextMuted).
						Padding(4, 10, 4)

					for _, item := range navItems {
						it := item
						active := a.currentPage == it.ID

						btn := ui.Row(c).
							Padding(9, 12).
							Radius(8).
							Gap(10).
							AlignItems(ui.Center).
							Cursor(ui.CursorPointer)

						if active {
							btn.Background(ui.Hex("#2563eb")).
								Shadow(0, 4, 12, 0, ui.RGBA(37, 99, 235, 0.25))
						}

						if btn.Clicked() {
							a.currentPage = it.ID
						}

						btn.Children(func() {
							iconColor := t.TextMuted
							textColor := t.Text
							if active {
								iconColor = ui.RGB(255, 255, 255)
								textColor = ui.RGB(255, 255, 255)
							}
							ui.Icon(c, it.Icon).Size(16, 16).TextColor(iconColor)
							ui.Text(c, it.Title).FontSize(13).Bold().TextColor(textColor)
						})
					}
				})
			})

			// Bottom: Footer status / version
			ui.Column(c).Gap(4).Padding(4, 8).Children(func() {
				ui.Text(c, "v0.2.0 • Native GPU").FontSize(11).Bold().TextColor(t.TextMuted)
				ui.Text(c, "Pure Go • egoist/mygo").FontSize(10).TextColor(t.TextMuted)
			})
		})
}
