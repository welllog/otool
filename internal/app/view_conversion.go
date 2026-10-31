package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/welllog/otool/internal/services"

	"github.com/egoist/mygo/ui"
)

type BaseOption struct {
	Name  string
	Value int
}

var commonBases = []BaseOption{
	{Name: "二进制 (2)", Value: 2},
	{Name: "八进制 (8)", Value: 8},
	{Name: "十进制 (10)", Value: 10},
	{Name: "十六进制 (16)", Value: 16},
	{Name: "32进制 (32)", Value: 32},
	{Name: "36进制 (36)", Value: 36},
}

var allBaseNames = func() []string {
	names := make([]string, 35)
	for i := 2; i <= 36; i++ {
		names[i-2] = fmt.Sprintf("%d 进制", i)
	}
	return names
}()

func (a *App) viewConversion(c *ui.Context) {
	t := c.Theme()

	// Reactive calculation
	res, err := services.ConvertBase(a.convInText, a.convInBase, a.convOutBase)
	if err != nil {
		a.convOutText = "Error"
		a.convHasError = true
	} else {
		a.convOutText = res
		a.convHasError = false
	}

	ui.Column(c).Gap(16).Children(func() {
		// Grid of Input and Output Cards
		ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
			// Input Side
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				Card(c, "输入源 (Source Base)", IconConversion, nil, func() {
					ui.Column(c).Gap(14).Children(func() {
						// Quick Buttons
						SectionHeader(c, fmt.Sprintf("原始进制: %d 进制", a.convInBase), ui.Hex("#2563eb"))
						ui.Row(c).Gap(6).Wrap().Children(func() {
							for _, b := range commonBases {
								base := b
								selected := a.convInBase == base.Value
								btn := ui.Row(c).
									Padding(6, 12).
									Radius(8).
									Border(1, t.Border).
									Cursor(ui.CursorPointer)

								if selected {
									btn.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
								} else {
									btn.Background(t.Surface)
								}

								if btn.Clicked() {
									a.convInBase = base.Value
									a.convInSelectStr = fmt.Sprintf("%d 进制", base.Value)
								}

								btn.Children(func() {
									textColor := t.Text
									if selected {
										textColor = ui.RGB(255, 255, 255)
									}
									ui.Text(c, base.Name).FontSize(11).Bold().TextColor(textColor)
								})
							}
						})

						// Detailed Select Dropdown (2..36)
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "更多进制:").FontSize(12).TextColor(t.TextMuted)
							if ui.Select(c, &a.convInSelectStr, allBaseNames).Changed() {
								parts := strings.Split(a.convInSelectStr, " ")
								if len(parts) > 0 {
									if v, err := strconv.Atoi(parts[0]); err == nil {
										a.convInBase = v
									}
								}
							}
						})

						// Input value
						SectionHeader(c, "待转换数值 (Input)", ui.Hex("#2563eb"))
						ui.TextInput(c, &a.convInText).
							Placeholder("输入数值，例如 255 或 FF...").
							Font("monospace").
							FontSize(16).
							Bold()
					})
				})
			})

			// Output Side
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				Card(c, "目标结果 (Target Base)", IconConversion, func() {
					if !a.convHasError && a.convOutText != "" && a.convOutText != "0" {
						CopyButton(c, a.convOutText)
					}
				}, func() {
					ui.Column(c).Gap(14).Children(func() {
						// Quick Buttons
						SectionHeader(c, fmt.Sprintf("目标进制: %d 进制", a.convOutBase), ui.Hex("#16a34a"))
						ui.Row(c).Gap(6).Wrap().Children(func() {
							for _, b := range commonBases {
								base := b
								selected := a.convOutBase == base.Value
								btn := ui.Row(c).
									Padding(6, 12).
									Radius(8).
									Border(1, t.Border).
									Cursor(ui.CursorPointer)

								if selected {
									btn.Background(ui.Hex("#16a34a")).Border(1, ui.Hex("#16a34a"))
								} else {
									btn.Background(t.Surface)
								}

								if btn.Clicked() {
									a.convOutBase = base.Value
									a.convOutSelectStr = fmt.Sprintf("%d 进制", base.Value)
								}

								btn.Children(func() {
									textColor := t.Text
									if selected {
										textColor = ui.RGB(255, 255, 255)
									}
									ui.Text(c, base.Name).FontSize(11).Bold().TextColor(textColor)
								})
							}
						})

						// Detailed Select Dropdown (2..36)
						ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
							ui.Text(c, "更多进制:").FontSize(12).TextColor(t.TextMuted)
							if ui.Select(c, &a.convOutSelectStr, allBaseNames).Changed() {
								parts := strings.Split(a.convOutSelectStr, " ")
								if len(parts) > 0 {
									if v, err := strconv.Atoi(parts[0]); err == nil {
										a.convOutBase = v
									}
								}
							}
						})

						// Output value
						SectionHeader(c, "转换结果 (Output)", ui.Hex("#16a34a"))
						outDisplay := a.convOutText
						resInput := ui.TextInput(c, &outDisplay).
							ReadOnly(true).
							Font("monospace").
							FontSize(16).
							Bold()

						if a.convHasError {
							resInput.TextColor(ui.Hex("#dc2626"))
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, IconWarn).Size(14, 14).TextColor(ui.Hex("#dc2626"))
								ui.Text(c, "输入数值包含当前进制非法字符").FontSize(11).Bold().TextColor(ui.Hex("#dc2626"))
							})
						} else {
							resInput.TextColor(ui.Hex("#16a34a"))
						}
					})
				})
			})
		})

		// Tip Card
		ui.Row(c).Justify(ui.Center).Padding(8, 0, 0).Children(func() {
			ui.Text(c, "💡 支持任意 2 到 36 进制间的实时转换，输入数值自动解析并同步更新。").
				FontSize(12).
				TextColor(t.TextMuted)
		})
	})
}

