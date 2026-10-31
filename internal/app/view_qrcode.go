package app

import (
	"path/filepath"
	"strings"

	"github.com/welllog/otool/internal/services"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var qrRecoverOptions = []struct {
	Name  string
	Desc  string
	Value int
}{
	{Name: "L (7%)", Desc: "低", Value: 0},
	{Name: "M (15%)", Desc: "中", Value: 1},
	{Name: "Q (25%)", Desc: "高", Value: 2},
	{Name: "H (30%)", Desc: "最高 (Logo必选)", Value: 3},
}

var qrStyleOptions = []struct {
	Name  string
	Value int
}{
	{Name: "标准方块", Value: int(services.QrShapeSquare)},
	{Name: "精致圆点", Value: int(services.QrShapeDots)},
	{Name: "圆角流动", Value: int(services.QrShapeRounded)},
}

var qrFormatOptions = []struct {
	Name  string
	Value string
}{
	{Name: "PNG (推荐)", Value: "png"},
	{Name: "JPEG", Value: "jpeg"},
	{Name: "SVG 矢量", Value: "svg"},
}

// 预设高对比度扫码安全前景色主题
var qrThemeColors = []struct {
	Name string
	Hex  string
}{
	{Name: "曜石黑", Hex: "#000000"},
	{Name: "商务蓝", Hex: "#1e40af"},
	{Name: "深邃青", Hex: "#0e7490"},
	{Name: "森林绿", Hex: "#166534"},
	{Name: "典雅紫", Hex: "#6b21a8"},
	{Name: "中国红", Hex: "#991b1b"},
	{Name: "极简炭", Hex: "#334155"},
}

func (a *App) viewQRCode(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Gap(14).Children(func() {
		// Top Tab Switcher: 解析二维码 vs 生成二维码
		ui.Row(c).Justify(ui.Center).Children(func() {
			ui.Segmented(c, &a.qrTab, "解析二维码", "生成二维码").
				Label("QR Code Mode").
				Width(300)
		})

		if a.qrTab == 0 {
			// ================== DECODE TAB (左右并排紧凑布局) ==================
			ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
				// Left: Upload & Drop Area
				ui.Column(c).Grow(1).Basis(0).Children(func() {
					Card(c, "上传/拖拽二维码图片", IconUpload, nil, func() {
						dropZone := ui.Column(c).
							Padding(22).
							Radius(10).
							Background(t.Surface).
							Border(1, t.Border).
							Center().
							Gap(10).
							Cursor(ui.CursorPointer)

						if files := dropZone.DroppedFiles(); len(files) > 0 {
							a.qrDecodeFile = files[0]
							a.triggerQrDecode(files[0])
						}
						if dropZone.FileDragOver() {
							dropZone.Border(2, t.Accent)
						}

						dropZone.Children(func() {
							ui.Icon(c, IconQRCode).Size(36, 36).TextColor(t.Accent)
							ui.Text(c, "拖拽二维码图片到此处，或点击下方选择").FontSize(13).Bold().TextColor(t.Text)
							ui.Text(c, "基于 gozxing 工业级纯 Go 引擎，支持倾斜/反色/模糊识别").FontSize(11).TextColor(t.TextMuted)

							if a.qrDecodeFile != "" {
								ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(6, 0, 0).Children(func() {
									if a.qrDecodeThumb != nil {
										ui.Image(c, a.qrDecodeThumb).Size(40, 40).Radius(6)
									}
									ui.Column(c).Gap(2).Children(func() {
										ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
											ui.Icon(c, IconCheck).Size(12, 12).TextColor(ui.Hex("#16a34a"))
											ui.Text(c, "识别目标").FontSize(10).Bold().TextColor(ui.Hex("#16a34a"))
										})
										ui.Text(c, filepath.Base(a.qrDecodeFile)).
											FontSize(12).
											Bold().
											TextColor(t.Text).
											MaxLines(1)
									})
								})
							}

							ui.Row(c).Padding(6, 0, 0).Children(func() {
								if ui.Button(c, "浏览二维码图片").Clicked() {
									go func() {
										paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
											Title: "选择二维码图片",
											Filters: []mygo.FileFilter{
												{Name: "图片文件", Extensions: []string{"png", "jpg", "jpeg", "webp", "gif", "bmp"}},
											},
										})
										if err == nil && len(paths) > 0 {
											a.update(func() {
												a.qrDecodeFile = paths[0]
												a.triggerQrDecode(paths[0])
											})
										}
									}()
								}
							})
						})
					})
				})

				// Right: Decode Result
				ui.Column(c).Grow(1).Basis(0).Children(func() {
					Card(c, "解析内容", IconText, func() {
						if a.qrDecodeText != "" {
							CopyButton(c, a.qrDecodeText)
						}
					}, func() {
						if a.qrIsDecoding {
							ui.Row(c).Center().Padding(36).Gap(8).Children(func() {
								ui.Spinner(c)
								ui.Text(c, "gozxing 引擎正在深度解析二维码...").FontSize(12).TextColor(t.Accent)
							})
						} else {
							ui.TextArea(c, &a.qrDecodeText).
								ReadOnly(true).
								Placeholder("解析出的二维码文本内容将在此处显示...").
								Height(160).
								Font("monospace")
						}
					})
				})
			})

		} else {
			// ================== ENCODE TAB (左右对开紧凑布局) ==================
			ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
				// LEFT COLUMN: Controls & Options (Grow 3)
				ui.Column(c).Grow(3).Basis(0).Gap(14).Children(func() {
					Card(c, "二维码生成参数", IconQRCode, nil, func() {
						ui.Column(c).Gap(12).Children(func() {
							// 1. Text input
							SectionHeader(c, "二维码文本 / 链接 (Content)", ui.Hex("#2563eb"))
							ui.TextArea(c, &a.qrEncodeText).
								Placeholder("在此输入要生成二维码的文本、网址链接或任意字符串...").
								Height(70)

							// 2. Row: Shape Style & Export Format
							ui.Row(c).Gap(14).AlignItems(ui.Start).Children(func() {
								// Visual Style
								ui.Column(c).Grow(1).Basis(0).Gap(6).Children(func() {
									SectionHeader(c, "视觉样式 (Shape Style)", ui.Hex("#2563eb"))
									ui.Row(c).Gap(6).Wrap().Children(func() {
										for _, opt := range qrStyleOptions {
											o := opt
											selected := a.qrStyle == o.Value
											btn := ui.Row(c).
												Padding(5, 10).
												Radius(6).
												Border(1, t.Border).
												Cursor(ui.CursorPointer)

											if selected {
												btn.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
											} else {
												btn.Background(t.Surface)
											}

											if btn.Clicked() {
												a.qrStyle = o.Value
											}

											btn.Children(func() {
												textColor := t.Text
												if selected {
													textColor = ui.RGB(255, 255, 255)
												}
												ui.Text(c, o.Name).FontSize(11).Bold().TextColor(textColor)
											})
										}
									})
								})

								// Export Format
								ui.Column(c).Grow(1).Basis(0).Gap(6).Children(func() {
									SectionHeader(c, "导出格式 (Format)", ui.Hex("#2563eb"))
									ui.Row(c).Gap(6).Wrap().Children(func() {
										for _, opt := range qrFormatOptions {
											o := opt
											selected := a.qrFormat == o.Value
											btn := ui.Row(c).
												Padding(5, 10).
												Radius(6).
												Border(1, t.Border).
												Cursor(ui.CursorPointer)

											if selected {
												btn.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
											} else {
												btn.Background(t.Surface)
											}

											if btn.Clicked() {
												a.qrFormat = o.Value
											}

											btn.Children(func() {
												textColor := t.Text
												if selected {
													textColor = ui.RGB(255, 255, 255)
												}
												ui.Text(c, o.Name).FontSize(11).Bold().TextColor(textColor)
											})
										}
									})
								})
							})

							// 3. Foreground Theme Color Selector (前景色主题选择)
							SectionHeader(c, "前景色主题 (Theme Color)", ui.Hex("#2563eb"))
							ui.Row(c).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
								for _, col := range qrThemeColors {
									cl := col
									isSelected := strings.EqualFold(a.qrFgColor, cl.Hex)
									pill := ui.Row(c).
										Padding(4, 9).
										Radius(6).
										Border(1, t.Border).
										Gap(6).
										AlignItems(ui.Center).
										Cursor(ui.CursorPointer)

									if isSelected {
										pill.Background(ui.Hex("#eff6ff")).Border(1.5, ui.Hex("#2563eb"))
									} else {
										pill.Background(t.Surface).Border(1, t.Border)
									}

									if pill.Clicked() {
										a.qrFgColor = cl.Hex
									}

									pill.Children(func() {
										// Color dot indicator
										ui.Row(c).
											Size(10, 10).
											Radius(5).
											Background(ui.Hex(cl.Hex)).
											Border(1, ui.Hex("#cbd5e1"))

										txtColor := t.Text
										if isSelected {
											txtColor = ui.Hex("#1d4ed8")
										}
										ui.Text(c, cl.Name).FontSize(11).Bold().TextColor(txtColor)
									})
								}

								// Custom Hex input
								ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
									ui.Text(c, "自定义:").FontSize(11).TextColor(t.TextMuted)
									ui.TextInput(c, &a.qrFgColor).Width(78).Placeholder("#000000")
								})
							})

							// 4. Row: Error Correction Level & Center Logo (并排两列，布局对称舒展)
							ui.Row(c).Gap(14).AlignItems(ui.Start).Children(func() {
								// Error Correction
								ui.Column(c).Grow(1).Basis(0).Gap(6).Children(func() {
									SectionHeader(c, "容错率等级 (Recovery Level)", ui.Hex("#2563eb"))
									hasLogo := a.qrLogoPath != ""
									ui.Row(c).Gap(4).Wrap().Children(func() {
										for _, opt := range qrRecoverOptions {
											o := opt
											selected := a.qrRecover == o.Value
											if hasLogo {
												selected = (o.Value == 3) // 强制 H
											}
											btn := ui.Row(c).
												Padding(4, 8).
												Radius(6).
												Border(1, t.Border).
												Cursor(ui.CursorPointer)

											if selected {
												btn.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
											} else {
												btn.Background(t.Surface)
											}

											if !hasLogo && btn.Clicked() {
												a.qrRecover = o.Value
											}

											btn.Children(func() {
												textColor := t.Text
												if selected {
													textColor = ui.RGB(255, 255, 255)
												}
												ui.Text(c, o.Name).FontSize(10).Bold().TextColor(textColor)
											})
										}
									})
									if hasLogo {
										ui.Row(c).Gap(4).AlignItems(ui.Center).Padding(2, 0, 0).Children(func() {
											ui.Icon(c, IconCheck).Size(11, 11).TextColor(ui.Hex("#16a34a"))
											ui.Text(c, "已自动锁定最高容错率 H(30%)").
												FontSize(10).
												TextColor(ui.Hex("#16a34a"))
										})
									}
								})

								// Center Logo (单层精致按钮，支持点击选择与文件拖拽)
								ui.Column(c).Grow(1).Basis(0).Gap(6).Children(func() {
									SectionHeader(c, "中心 Logo 图标 (可选)", ui.Hex("#2563eb"))
									if a.qrLogoPath == "" {
										btn := ui.Row(c).
											Padding(7, 14).
											Radius(6).
											Background(t.Surface).
											Border(1, t.Border).
											AlignItems(ui.Center).
											Justify(ui.Center).
											Gap(6).
											Cursor(ui.CursorPointer)

										if files := btn.DroppedFiles(); len(files) > 0 {
											a.setQRLogo(files[0])
										}
										if btn.FileDragOver() {
											btn.Border(2, ui.Hex("#2563eb")).Background(ui.Hex("#eff6ff"))
										} else if btn.Hovered() {
											btn.Border(1, t.Accent)
										}

										if btn.Clicked() {
											go func() {
												paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
													Title: "选择 Logo 图片",
													Filters: []mygo.FileFilter{
														{Name: "图片文件", Extensions: []string{"png", "jpg", "jpeg", "webp"}},
													},
												})
												if err == nil && len(paths) > 0 {
													a.setQRLogo(paths[0])
												}
											}()
										}

										btn.Children(func() {
											ui.Icon(c, IconUpload).Size(14, 14).TextColor(t.Accent)
											ui.Text(c, "点击或拖拽 Logo 图片").FontSize(11).Bold().TextColor(t.Text)
										})
									} else {
										// Show Selected Logo info (已选状态下同样支持直接拖拽新图片进行替换)
										card := ui.Row(c).
											Padding(5, 10).
											Radius(6).
											Background(t.Surface).
											Border(1, t.Border).
											AlignItems(ui.Center).
											Gap(8)

										if files := card.DroppedFiles(); len(files) > 0 {
											a.setQRLogo(files[0])
										}
										if card.FileDragOver() {
											card.Border(2, ui.Hex("#2563eb")).Background(ui.Hex("#eff6ff"))
										}

										card.Children(func() {
											// Left area: Thumb + File name (支持超长文本安全截断，不挤压右侧)
											ui.Row(c).Gap(6).AlignItems(ui.Center).Grow(1).Shrink(1).MinWidth(0).Children(func() {
												if a.qrLogoThumb != nil {
													ui.Image(c, a.qrLogoThumb).Size(22, 22).Radius(4).Shrink(0)
												}
												ui.Text(c, filepath.Base(a.qrLogoPath)).
													FontSize(11).
													Bold().
													TextColor(t.Text).
													MaxLines(1).
													SingleLine().
													Grow(1).
													Shrink(1)
											})

											// Right area: Compact remove button (紧凑型操作胶囊，彻底消除高度与宽度溢出)
											rmBtn := ui.Row(c).
												Padding(3, 7).
												Radius(4).
												Background(ui.Hex("#fee2e2")).
												Border(1, ui.Hex("#fca5a5")).
												Cursor(ui.CursorPointer).
												Shrink(0)

											if rmBtn.Hovered() {
												rmBtn.Background(ui.Hex("#fecaca"))
											}
											if rmBtn.Clicked() {
												a.qrLogoPath = ""
												a.qrLogoThumb = nil
											}
											rmBtn.Children(func() {
												ui.Text(c, "移除").FontSize(10).Bold().TextColor(ui.Hex("#dc2626"))
											})
										})
									}
								})
							})

							// 5. Output Size (独立整行展开，宽敞从容，彻底根治加号截断与文字换行)
							SectionHeader(c, "输出尺寸 (Size)", ui.Hex("#2563eb"))
							ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
								ui.NumberInput(c, &a.qrSize, 64, 1024, 32).Width(130)

								ui.Text(c, "常用尺寸:").FontSize(11).TextColor(t.TextMuted)
								for _, sz := range []float64{256, 512, 1024} {
									s := sz
									szSelected := a.qrSize == s
									p := ui.Row(c).Padding(3, 8).Radius(4).Border(1, t.Border).Cursor(ui.CursorPointer)
									if szSelected {
										p.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
									} else {
										p.Background(t.Surface)
									}
									if p.Clicked() {
										a.qrSize = s
									}
									p.Children(func() {
										color := t.Text
										if szSelected {
											color = ui.RGB(255, 255, 255)
										}
										ui.Textf(c, "%.0f px", s).FontSize(11).Bold().TextColor(color)
									})
								}
							})

							// 5. Save Path & Generate Action
							SectionHeader(c, "保存目录", ui.Hex("#2563eb"))
							ui.Row(c).
								Padding(6, 12).
								Radius(6).
								Background(t.Surface).
								Border(1, t.Border).
								AlignItems(ui.Center).
								Justify(ui.SpaceBetween).
								Children(func() {
									displayPath := a.qrSavePath
									if displayPath == "" {
										displayPath = "未设置保存目录 (仅在界面中预览)"
									}
									ui.Text(c, displayPath).FontSize(11).TextColor(t.Text).MaxLines(1).Grow(1)

									if ui.Button(c, "更改保存目录").Clicked() {
										go func() {
											paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
												Title:     "选择二维码图片保存目录",
												Directory: true,
											})
											if err == nil && len(paths) > 0 {
												a.update(func() {
													a.qrSavePath = paths[0]
												})
											}
										}()
									}
								})

							ui.Row(c).Justify(ui.Center).Padding(4, 0, 0).Children(func() {
								btn := ui.PrimaryButton(c, "立即生成二维码").
									Width(220).
									Height(36).
									Disabled(strings.TrimSpace(a.qrEncodeText) == "")

								if btn.Clicked() {
									a.triggerQrEncode(c)
								}
							})
						})
					})
				})

				// RIGHT COLUMN: Preview Card (Grow 2)
				ui.Column(c).Grow(2).Basis(0).Children(func() {
					Card(c, "二维码实时预览", IconQRCode, func() {
						if a.qrLastSavedName != "" {
							PillBadge(c, strings.ToUpper(a.qrFormat))
						}
					}, func() {
						ui.Column(c).
							Center().
							Padding(16).
							Gap(12).
							Children(func() {
								if a.qrPreviewBitmap != nil {
									ui.Image(c, a.qrPreviewBitmap).Size(190, 190).Radius(8)

									if a.qrLastSavedName != "" && a.qrSavePath != "" {
										ui.Column(c).Center().Gap(4).Children(func() {
											ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
												ui.Icon(c, IconCheck).Size(12, 12).TextColor(ui.Hex("#16a34a"))
												ui.Text(c, "已成功保存到本地").FontSize(11).Bold().TextColor(ui.Hex("#16a34a"))
											})
											ui.Text(c, filepath.Join(a.qrSavePath, a.qrLastSavedName)).
												FontSize(10).
												TextColor(t.TextMuted).
												MaxLines(2)
										})
									}
								} else {
									ui.Column(c).Center().Padding(40, 20).Gap(8).Children(func() {
										ui.Icon(c, IconQRCode).Size(48, 48).TextColor(t.TextMuted)
										ui.Text(c, "输入内容后点击“立即生成”").FontSize(12).Bold().TextColor(t.TextMuted)
										ui.Text(c, "支持圆点/圆角/多彩主题/Logo与SVG").FontSize(10).TextColor(t.TextMuted)
									})
								}
							})
					})
				})
			})
		}
	})
}

func (a *App) triggerQrDecode(filePath string) {
	a.update(func() {
		a.qrIsDecoding = true
		a.qrDecodeText = ""
	})

	go func() {
		content, err := services.QrDecodeFile(filePath)
		info, _ := services.DecodeImageInfo(filePath)

		a.update(func() {
			a.qrIsDecoding = false
			if info != nil && info.Thumbnail != nil {
				a.qrDecodeThumb = ui.NewBitmap(info.Thumbnail)
			}
			if err != nil {
				a.qrDecodeText = ""
				a.Toast("二维码识别失败: " + err.Error())
			} else {
				a.qrDecodeText = content
				a.Toast("二维码识别成功")
			}
		})
	}()
}

func (a *App) triggerQrEncode(c *ui.Context) {
	text := strings.TrimSpace(a.qrEncodeText)
	if text == "" {
		c.Toast("请输入二维码内容")
		return
	}

	size := int(a.qrSize)
	if size <= 0 {
		size = 256
	}

	format := services.QrExportFormat(a.qrFormat)
	if format == "" {
		format = services.QrFormatPNG
	}

	opts := services.QrGenerateOptions{
		Text:     text,
		Style:    services.QrShapeStyle(a.qrStyle),
		Recover:  a.qrRecover,
		Size:     size,
		Format:   format,
		FgColor:  a.qrFgColor,
		LogoPath: a.qrLogoPath,
		SaveDir:  a.qrSavePath,
	}

	res, err := services.QrGenerate(opts)
	if err != nil {
		c.Toast("生成二维码失败: " + err.Error())
		return
	}

	bitmap, err := ui.DecodeBitmap(res.PreviewPNG)
	if err != nil {
		c.Toast("解析二维码预览失败: " + err.Error())
		return
	}

	a.qrPreviewBitmap = bitmap
	a.qrLastSavedName = res.SaveName

	if a.qrSavePath != "" && res.SavedPath != "" {
		c.Toast("二维码已成功保存: " + res.SaveName)
	} else {
		c.Toast("二维码生成成功")
	}
}

func (a *App) setQRLogo(filePath string) {
	ext := strings.ToLower(filepath.Ext(filePath))
	validExts := map[string]bool{
		".png":  true,
		".jpg":  true,
		".jpeg": true,
		".webp": true,
		".bmp":  true,
		".gif":  true,
	}
	if !validExts[ext] {
		a.Toast("请选择有效的图片文件 (PNG, JPG, WEBP 等)")
		return
	}

	info, err := services.DecodeImageInfo(filePath)
	if err != nil {
		a.Toast("加载 Logo 失败: " + err.Error())
		return
	}

	a.update(func() {
		a.qrLogoPath = filePath
		if info != nil && info.Thumbnail != nil {
			a.qrLogoThumb = ui.NewBitmap(info.Thumbnail)
		}
		a.qrRecover = 3 // 自动锁定最高容错率 H(30%)
	})
	a.Toast("Logo 图标已加载")
}
