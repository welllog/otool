package app

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// Card creates a structured card element with header, optional icon/extra, and content body.
func Card(c *ui.Context, title string, icon *ui.SVG, extra func(), body func()) ui.Element {
	t := c.Theme()
	return ui.Column(c).
		Padding(18).
		Gap(12).
		Radius(12).
		Background(t.Background).
		Border(1, t.Border).
		Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.04)).
		Children(func() {
			if title != "" || extra != nil {
				ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if icon != nil {
							ui.Icon(c, icon).Size(18, 18).TextColor(t.Accent)
						}
						if title != "" {
							ui.Text(c, title).FontSize(14).Bold().TextColor(t.Text)
						}
					})
					if extra != nil {
						extra()
					}
				})
			}
			body()
		})
}

// CopyButton creates a compact button that writes text to clipboard and shows toast.
func CopyButton(c *ui.Context, text string) {
	t := c.Theme()
	btn := ui.Row(c).
		Padding(4, 10).
		Radius(6).
		Gap(4).
		AlignItems(ui.Center).
		Background(t.Surface).
		Border(1, t.Border).
		Cursor(ui.CursorPointer)

	if btn.Clicked() {
		if text != "" {
			mygo.Clipboard.WriteText(text)
			c.Toast("已复制到剪贴板")
		} else {
			c.Toast("没有可复制的内容")
		}
	}

	btn.Children(func() {
		ui.Icon(c, IconCopy).Size(13, 13).TextColor(t.TextMuted)
		ui.Text(c, "复制").FontSize(11).TextColor(t.TextMuted)
	})
}

// PillBadge renders a small rounded indicator badge.
func PillBadge(c *ui.Context, label string) {
	t := c.Theme()
	ui.Row(c).
		Padding(2, 8).
		Radius(999).
		Background(t.Surface).
		Border(1, t.Border).
		Children(func() {
			ui.Text(c, label).FontSize(10).Bold().TextColor(t.TextMuted)
		})
}

// SectionHeader displays a category label with a colored dot.
func SectionHeader(c *ui.Context, label string, dotColor ui.Color) {
	ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(dotColor)
		ui.Text(c, label).FontSize(11).Bold().TextColor(dotColor)
	})
}

