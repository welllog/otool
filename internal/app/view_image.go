package app

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/welllog/otool/internal/services"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

var resizeModes = []struct {
	Name  string
	Value int
}{
	{Name: "保持原尺寸", Value: services.OpOriginal},
	{Name: "固定宽度缩放", Value: services.OpFixedWidth},
	{Name: "最大宽度缩放", Value: services.OpMaxWidth},
	{Name: "固定宽高裁剪", Value: services.OpFixedWH},
	{Name: "最大宽高缩放", Value: services.OpMaxWH},
	{Name: "固定高度缩放", Value: services.OpFixedHeight},
	{Name: "最大高度缩放", Value: services.OpMaxHeight},
	{Name: "按百分比缩放", Value: services.OpPercentage},
	{Name: "自定义裁剪", Value: services.OpCustomCrop},
}

var imageEncoders = []string{"png", "jpg", "webp", "gif", "avif", "bmp", "tiff"}

var pngCompressOptions = []struct {
	Name  string
	Value int
}{
	{Name: "默认级别", Value: 0},
	{Name: "不压缩", Value: -1},
	{Name: "速度优先", Value: -2},
	{Name: "压缩优先", Value: -3},
}

var gifDropOptions = []struct {
	Name  string
	Value int
}{
	{Name: "不处理", Value: 0},
	{Name: "抽取 1/2", Value: 2},
	{Name: "抽取 1/3", Value: 3},
	{Name: "抽取 1/4", Value: 4},
	{Name: "抽取 1/5", Value: 5},
	{Name: "抽取 1/6", Value: 6},
	{Name: "抽取 1/7", Value: 7},
	{Name: "抽取 1/8", Value: 8},
}

var avifEncoderOptions = []struct {
	Name  string
	Value string
}{
	{Name: "自动 (推荐)", Value: "auto"},
	{Name: "快速 (gav1d)", Value: "gav1d"},
	{Name: "高质量 (libavif)", Value: "libavif"},
}

func (a *App) viewImage(c *ui.Context) {
	// 顶部功能分段选择：精细处理与裁剪 vs 文件夹全量批处理
	ui.Row(c).Margin(0, 0, 10, 0).Children(func() {
		ui.Segmented(c, &a.imgTab, "图片精细处理与裁剪", "文件夹全量批处理")
	})

	if a.imgTab == 0 {
		a.viewImageFine(c)
	} else {
		a.viewImageFolder(c)
	}
}

func (a *App) viewImageFine(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Gap(16).Children(func() {
		// 1. Image Files Section
		Card(c, fmt.Sprintf("待处理图片列表 (%d 张)", len(a.imgFiles)), IconImage, func() {
			if len(a.imgFiles) > 0 {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					if ui.Button(c, "清空列表").Disabled(a.imgIsProcessing).Clicked() {
						a.imgFiles = nil
						a.imgInfos = nil
						a.imgThumbs = nil
						a.imgOpts.CropX = 0
						a.imgOpts.CropY = 0
						a.imgOpts.CropWidth = 0
						a.imgOpts.CropHeight = 0
						c.Toast("已清空图片列表")
					}
				})
			}
		}, func() {
			// Drop zone & file selector
			dropZone := ui.Row(c).
				Padding(14, 16).
				Radius(10).
				Background(t.Surface).
				Border(1, t.Border).
				AlignItems(ui.Center).
				Justify(ui.SpaceBetween)

			if dropped := dropZone.DroppedFiles(); len(dropped) > 0 {
				a.addImages(dropped)
			}
			if dropZone.FileDragOver() {
				dropZone.Border(2, t.Accent)
			}

			dropZone.Children(func() {
				ui.Column(c).Gap(4).Children(func() {
					ui.Text(c, "支持拖拽单张或多张图片至此处添加").FontSize(12).Bold().TextColor(t.Text)
					ui.Text(c, "格式支持 JPG, PNG, GIF, BMP, TIFF, WEBP, AVIF").FontSize(11).TextColor(t.TextMuted)
				})

				if ui.Button(c, "添加图片文件").Disabled(a.imgIsProcessing).Clicked() {
					go func() {
						paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
							Title:    "选择图片文件",
							Multiple: true,
							Filters: []mygo.FileFilter{
								{Name: "图片文件", Extensions: []string{"jpg", "jpeg", "png", "gif", "bmp", "tiff", "tif", "webp", "avif"}},
							},
						})
						if err == nil && len(paths) > 0 {
							a.addImages(paths)
						}
					}()
				}
			})

			// Render thumbnail list
			if len(a.imgInfos) > 0 {
				ui.Column(c).Padding(8, 0, 0).Gap(8).Children(func() {
					for idx, info := range a.imgInfos {
						i := idx
						item := info
						ui.Row(c).
							Padding(8, 12).
							Radius(8).
							Background(t.Surface).
							Border(1, t.Border).
							AlignItems(ui.Center).
							Justify(ui.SpaceBetween).
							Children(func() {
								// Left: Thumbnail + info
								ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
									if i < len(a.imgThumbs) && a.imgThumbs[i] != nil {
										ui.Image(c, a.imgThumbs[i]).Size(40, 40).Radius(4)
									}
									ui.Column(c).Gap(2).Children(func() {
										ui.Text(c, item.Name).FontSize(12).Bold().TextColor(t.Text).MaxLines(1)
										ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
											PillBadge(c, item.Format)
											ui.Textf(c, "%d × %d", item.Width, item.Height).FontSize(11).TextColor(t.TextMuted)
											ui.Text(c, item.Size).FontSize(11).TextColor(t.TextMuted)
											if item.Frames > 1 {
												ui.Textf(c, "%d 帧动图", item.Frames).FontSize(11).TextColor(t.Accent)
											}
										})
									})
								})

								// Right: Delete button
								if ui.Button(c, "移除").Disabled(a.imgIsProcessing).Clicked() {
									a.removeImage(i)
								}
							})
					}
				})
			}
		})

		// 2. Options Grid: Resize & Parameters
		ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
			// Resize Card
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				Card(c, "尺寸调整", nil, func() {
					if len(a.imgInfos) > 0 {
						if ui.Button(c, "重置原图尺寸").Clicked() {
							a.imgOpts.Width = a.imgInfos[0].Width
							a.imgOpts.Height = a.imgInfos[0].Height
							a.imgOpts.Percent = 100
							c.Toast("已重置尺寸参数")
						}
					}
				}, func() {
					ui.Column(c).Gap(14).Children(func() {
						SectionHeader(c, "缩放与裁剪模式", ui.Hex("#2563eb"))
						ui.Row(c).Gap(6).Wrap().Children(func() {
							for _, m := range resizeModes {
								mode := m
								selected := a.imgOpts.Op == mode.Value
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
									a.imgOpts.Op = mode.Value
								}

								btn.Children(func() {
									textColor := t.Text
									if selected {
										textColor = ui.RGB(255, 255, 255)
									}
									ui.Text(c, mode.Name).FontSize(11).Bold().TextColor(textColor)
								})
							}
						})

						// Conditional Inputs based on Op
						if a.imgOpts.Op == services.OpFixedWidth || a.imgOpts.Op == services.OpMaxWidth ||
							a.imgOpts.Op == services.OpFixedWH || a.imgOpts.Op == services.OpMaxWH {
							ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "宽度 (PX):").FontSize(12).Bold().TextColor(t.Text)
								wVal := float64(a.imgOpts.Width)
								if ui.NumberInput(c, &wVal, 1, 10000, 10).Changed() {
									a.imgOpts.Width = int(wVal)
								}
							})
						}

						if a.imgOpts.Op == services.OpFixedHeight || a.imgOpts.Op == services.OpMaxHeight ||
							a.imgOpts.Op == services.OpFixedWH || a.imgOpts.Op == services.OpMaxWH {
							ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "高度 (PX):").FontSize(12).Bold().TextColor(t.Text)
								hVal := float64(a.imgOpts.Height)
								if ui.NumberInput(c, &hVal, 1, 10000, 10).Changed() {
									a.imgOpts.Height = int(hVal)
								}
							})
						}

						if a.imgOpts.Op == services.OpPercentage {
							ui.Row(c).Gap(12).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "比例 (%):").FontSize(12).Bold().TextColor(t.Text)
								pVal := float64(a.imgOpts.Percent)
								if ui.NumberInput(c, &pVal, 1, 500, 5).Changed() {
									a.imgOpts.Percent = int(pVal)
								}
								ui.Textf(c, "%d%%", a.imgOpts.Percent).FontSize(12).TextColor(t.Accent)
							})
						}

						if a.imgOpts.Op == services.OpCustomCrop {
							ui.Column(c).Gap(10).Children(func() {
								// Visual Crop Selection Trigger
								ui.Row(c).
									Padding(10, 12).
									Radius(8).
									Background(t.Surface).
									Border(1, t.Border).
									AlignItems(ui.Center).
									Justify(ui.SpaceBetween).
									Children(func() {
										ui.Column(c).Gap(2).Children(func() {
											ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
												ui.Icon(c, IconImage).Size(14, 14).TextColor(t.Accent)
												ui.Text(c, "自由选区状态:").FontSize(11).Bold().TextColor(t.Text)
											})
											if a.imgOpts.CropWidth > 0 && a.imgOpts.CropHeight > 0 {
												ui.Textf(c, "%d × %d px (起点: %d, %d)", a.imgOpts.CropWidth, a.imgOpts.CropHeight, a.imgOpts.CropX, a.imgOpts.CropY).
													FontSize(12).
													Bold().
													TextColor(ui.Hex("#16a34a"))
											} else {
												ui.Text(c, "尚未设定裁剪选区 (点击右侧按钮进行可视化框选)").
													FontSize(11).
													TextColor(t.TextMuted)
											}
										})

										ui.Row(c).Gap(6).Children(func() {
											if a.imgOpts.CropWidth > 0 && a.imgOpts.CropHeight > 0 {
												if ui.Button(c, "清除选区").Clicked() {
													a.imgOpts.CropX = 0
													a.imgOpts.CropY = 0
													a.imgOpts.CropWidth = 0
													a.imgOpts.CropHeight = 0
													c.Toast("已清除选区")
												}
											}
											btn := ui.PrimaryButton(c, "📐 打开可视化裁剪选区")
											if btn.Clicked() {
												a.openCropModal(c)
											}
										})
									})

								// Coordinate fine-tuning inputs
								ui.Text(c, "像素坐标微调 (可直接修改):").FontSize(11).TextColor(t.TextMuted)
								ui.Row(c).Gap(10).Children(func() {
									ui.Text(c, "X:").FontSize(11).TextColor(t.TextMuted)
									xVal := float64(a.imgOpts.CropX)
									if ui.NumberInput(c, &xVal, 0, 10000, 1).Changed() {
										a.imgOpts.CropX = int(xVal)
									}
									ui.Text(c, "Y:").FontSize(11).TextColor(t.TextMuted)
									yVal := float64(a.imgOpts.CropY)
									if ui.NumberInput(c, &yVal, 0, 10000, 1).Changed() {
										a.imgOpts.CropY = int(yVal)
									}
									ui.Text(c, "宽:").FontSize(11).TextColor(t.TextMuted)
									cwVal := float64(a.imgOpts.CropWidth)
									if ui.NumberInput(c, &cwVal, 0, 10000, 10).Changed() {
										a.imgOpts.CropWidth = int(cwVal)
									}
									ui.Text(c, "高:").FontSize(11).TextColor(t.TextMuted)
									chVal := float64(a.imgOpts.CropHeight)
									if ui.NumberInput(c, &chVal, 0, 10000, 10).Changed() {
										a.imgOpts.CropHeight = int(chVal)
									}
								})
							})
						}
					})
				})
			})

			// Encoder Configuration Card
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				a.renderEncoderSettingsCard(c)
			})
		})

		// 3. Output Folder & Execution Card
		Card(c, "保存与执行", nil, nil, func() {
			ui.Column(c).Gap(12).Children(func() {
				// Output Folder
				ui.Row(c).
					Padding(10, 14).
					Radius(8).
					Background(t.Surface).
					Border(1, t.Border).
					AlignItems(ui.Center).
					Justify(ui.SpaceBetween).
					Children(func() {
						displayPath := a.imgOutPath
						if displayPath == "" {
							displayPath = "未设置保存目录 (请选择)"
						}
						ui.Column(c).Gap(2).Grow(1).Children(func() {
							ui.Text(c, "转换后图片保存目录:").FontSize(11).TextColor(t.TextMuted)
							ui.Text(c, displayPath).FontSize(12).Bold().TextColor(t.Text).MaxLines(1)
						})

						if ui.Button(c, "更改保存目录").Disabled(a.imgIsProcessing).Clicked() {
							go func() {
								paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
									Title:     "选择转换后图片保存目录",
									Directory: true,
								})
								if err == nil && len(paths) > 0 {
									a.update(func() {
										a.imgOutPath = paths[0]
									})
								}
							}()
						}
					})

				// Progress bar if processing
				if a.imgShowProgress {
					ui.Column(c).Gap(6).Children(func() {
						ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Spinner(c)
								ui.Textf(c, "正在处理: %s", a.imgCurFile).FontSize(12).TextColor(t.Accent)
							})
							ui.Textf(c, "%d / %d (%d%%)", a.imgDoneCount, len(a.imgFiles), a.imgProgress).
								FontSize(12).
								Bold().
								TextColor(t.Accent)
						})
						ui.Progress(c, float64(a.imgProgress)/100.0)
					})
				}

				// Execute Action Button
				ui.Row(c).Justify(ui.Center).Padding(6, 0, 0).Children(func() {
					btn := ui.PrimaryButton(c, fmt.Sprintf("开始批量转换 (%d 张图片)", len(a.imgFiles))).
						Width(280).
						Height(42).
						Disabled(a.imgIsProcessing || len(a.imgFiles) == 0 || a.imgOutPath == "")

					if btn.Clicked() {
						a.startBatchTransform(c)
					}
				})
			})
		})

		// 4. Render Crop Modal if open
		a.renderCropModal(c)
	})
}

func (a *App) addImages(paths []string) {
	go func() {
		for _, p := range paths {
			if slices.Contains(a.imgFiles, p) {
				continue
			}
			info, err := services.DecodeImageInfo(p)
			if err == nil {
				a.update(func() {
					a.imgFiles = append(a.imgFiles, p)
					a.imgInfos = append(a.imgInfos, info)
					if info.Thumbnail != nil {
						a.imgThumbs = append(a.imgThumbs, ui.NewBitmap(info.Thumbnail))
					} else {
						a.imgThumbs = append(a.imgThumbs, nil)
					}
					if len(a.imgFiles) == 1 {
						a.imgOpts.Width = info.Width
						a.imgOpts.Height = info.Height
						a.imgOpts.CropWidth = info.Width
						a.imgOpts.CropHeight = info.Height
					}
				})
			}
		}
	}()
}

func (a *App) removeImage(index int) {
	if index >= 0 && index < len(a.imgFiles) {
		a.imgFiles = append(a.imgFiles[:index], a.imgFiles[index+1:]...)
		a.imgInfos = append(a.imgInfos[:index], a.imgInfos[index+1:]...)
		a.imgThumbs = append(a.imgThumbs[:index], a.imgThumbs[index+1:]...)
		if len(a.imgFiles) == 0 {
			a.imgOpts.CropX = 0
			a.imgOpts.CropY = 0
			a.imgOpts.CropWidth = 0
			a.imgOpts.CropHeight = 0
		}
	}
}

func (a *App) startBatchTransform(c *ui.Context) {
	if len(a.imgFiles) == 0 {
		c.Toast("请先添加图片文件")
		return
	}
	if a.imgOutPath == "" {
		c.Toast("请选择转换保存目录")
		return
	}

	a.imgIsProcessing = true
	a.imgShowProgress = true
	a.imgProgress = 0
	a.imgDoneCount = 0

	files := slices.Clone(a.imgFiles)
	opts := a.imgOpts
	opts.OutPath = a.imgOutPath

	go func() {
		success, fail := services.BatchTransform(files, opts, func(done, total int, current string) {
			pct := done * 100 / total
			a.update(func() {
				a.imgProgress = pct
				a.imgDoneCount = done
				a.imgCurFile = current
			})
		})

		a.update(func() {
			a.imgIsProcessing = false
			a.imgShowProgress = false
			msg := fmt.Sprintf("批量转换完成: 成功 %d, 失败 %d", success, fail)
			a.Toast(msg)
		})
	}()
}

func (a *App) renderEncoderSettingsCard(c *ui.Context) {
	t := c.Theme()
	Card(c, "格式与编码参数", nil, nil, func() {
		ui.Column(c).Gap(14).Children(func() {
			SectionHeader(c, "目标保存格式", ui.Hex("#16a34a"))
			ui.Row(c).Gap(6).Wrap().Children(func() {
				for _, enc := range imageEncoders {
					e := enc
					selected := a.imgOpts.Encoder == e
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
						a.imgOpts.Encoder = e
					}

					btn.Children(func() {
						textColor := t.Text
						if selected {
							textColor = ui.RGB(255, 255, 255)
						}
						ui.Text(c, strings.ToUpper(e)).FontSize(11).Bold().TextColor(textColor)
					})
				}
			})

			// Detailed options per format
			switch a.imgOpts.Encoder {
			case "jpg", "jpeg":
				ui.Column(c).Gap(8).Children(func() {
					SectionHeader(c, fmt.Sprintf("JPEG 压缩质量: %d", a.imgOpts.JpgQuality), ui.Hex("#2563eb"))
					qVal := float64(a.imgOpts.JpgQuality)
					if ui.Slider(c, &qVal, 1, 100).Changed() {
						a.imgOpts.JpgQuality = int(qVal)
					}
				})

			case "png":
				ui.Column(c).Gap(8).Children(func() {
					SectionHeader(c, "PNG 压缩策略", ui.Hex("#2563eb"))
					ui.Row(c).Gap(6).Wrap().Children(func() {
						for _, pc := range pngCompressOptions {
							p := pc
							selected := a.imgOpts.PngCompression == p.Value
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
								a.imgOpts.PngCompression = p.Value
							}

							btn.Children(func() {
								textColor := t.Text
								if selected {
									textColor = ui.RGB(255, 255, 255)
								}
								ui.Text(c, p.Name).FontSize(11).TextColor(textColor)
							})
						}
					})
				})

			case "webp":
				ui.Column(c).Gap(10).Children(func() {
					SectionHeader(c, fmt.Sprintf("WebP 质量: %d", a.imgOpts.WebpQuality), ui.Hex("#2563eb"))
					qVal := float64(a.imgOpts.WebpQuality)
					if ui.Slider(c, &qVal, 1, 100).Changed() {
						a.imgOpts.WebpQuality = int(qVal)
					}
					ui.Checkbox(c, &a.imgOpts.WebpLossless, "无损压缩 (Lossless)")
					if a.imgOpts.WebpLossless {
						ui.Checkbox(c, &a.imgOpts.WebpRgbInTransparent, "保留透明区域 RGB")
					}
				})

			case "gif":
				ui.Column(c).Gap(10).Children(func() {
					SectionHeader(c, fmt.Sprintf("色彩数量: %d", a.imgOpts.GifNumColors), ui.Hex("#2563eb"))
					cVal := float64(a.imgOpts.GifNumColors)
					if ui.Slider(c, &cVal, 2, 256).Changed() {
						a.imgOpts.GifNumColors = int(cVal)
					}

					SectionHeader(c, "抽帧处理", ui.Hex("#2563eb"))
					ui.Row(c).Gap(6).Wrap().Children(func() {
						for _, gd := range gifDropOptions {
							g := gd
							selected := a.imgOpts.GifDropRate == g.Value
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

							if btn.Clicked() {
								a.imgOpts.GifDropRate = g.Value
							}

							btn.Children(func() {
								textColor := t.Text
								if selected {
									textColor = ui.RGB(255, 255, 255)
								}
								ui.Text(c, g.Name).FontSize(10).TextColor(textColor)
							})
						}
					})
					ui.Checkbox(c, &a.imgOpts.GifDrawOnBefore, "在前一帧上绘制")
				})

			case "avif":
				ui.Column(c).Gap(10).Children(func() {
					SectionHeader(c, fmt.Sprintf("AVIF 图像质量: %d", a.imgOpts.AvifQuality), ui.Hex("#2563eb"))
					qVal := float64(a.imgOpts.AvifQuality)
					if ui.Slider(c, &qVal, 1, 100).Changed() {
						a.imgOpts.AvifQuality = int(qVal)
					}

					SectionHeader(c, fmt.Sprintf("Alpha 质量: %d", a.imgOpts.AvifQualityAlpha), ui.Hex("#2563eb"))
					aqVal := float64(a.imgOpts.AvifQualityAlpha)
					if ui.Slider(c, &aqVal, 1, 100).Changed() {
						a.imgOpts.AvifQualityAlpha = int(aqVal)
					}

					SectionHeader(c, fmt.Sprintf("编码速度: %d (0最高质量, 10最快)", a.imgOpts.AvifSpeed), ui.Hex("#2563eb"))
					spVal := float64(a.imgOpts.AvifSpeed)
					if ui.Slider(c, &spVal, 0, 10).Changed() {
						a.imgOpts.AvifSpeed = int(spVal)
					}

					SectionHeader(c, "编码器核心", ui.Hex("#2563eb"))
					ui.Row(c).Gap(6).Wrap().Children(func() {
						for _, ae := range avifEncoderOptions {
							enc := ae
							selected := a.imgOpts.AvifEncoder == enc.Value
							btn := ui.Row(c).
								Padding(4, 10).
								Radius(6).
								Border(1, t.Border).
								Cursor(ui.CursorPointer)

							if selected {
								btn.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
							} else {
								btn.Background(t.Surface)
							}

							if btn.Clicked() {
								a.imgOpts.AvifEncoder = enc.Value
							}

							btn.Children(func() {
								textColor := t.Text
								if selected {
									textColor = ui.RGB(255, 255, 255)
								}
								ui.Text(c, enc.Name).FontSize(11).TextColor(textColor)
							})
						}
					})
				})

			default:
				ui.Row(c).Padding(12).Center().Children(func() {
					ui.Text(c, "此格式采用标准压缩算法，无需额外配置参数").
						FontSize(12).
						TextColor(t.TextMuted)
				})
			}
		})
	})
}

func (a *App) viewImageFolder(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Gap(16).Children(func() {
		// 1. Source Folder & Save Destination Card
		Card(c, "文件夹与存储策略", IconFolder, nil, func() {
			ui.Column(c).Gap(14).Children(func() {
				// 源文件夹拖拽与选择
				srcDropZone := ui.Row(c).
					Padding(14, 16).
					Radius(10).
					Background(t.Surface).
					Border(1, t.Border).
					AlignItems(ui.Center).
					Justify(ui.SpaceBetween)

				if dropped := srcDropZone.DroppedFiles(); len(dropped) > 0 {
					p := dropped[0]
					if fi, err := os.Stat(p); err == nil {
						if fi.IsDir() {
							a.setFolderSrc(p)
						} else {
							a.setFolderSrc(filepath.Dir(p))
						}
					}
				}
				if srcDropZone.FileDragOver() {
					srcDropZone.Border(2, t.Accent)
				}

				srcDropZone.Children(func() {
					ui.Column(c).Gap(3).Grow(1).Children(func() {
						ui.Text(c, "源文件夹 (支持拖拽文件夹至此处)").FontSize(12).Bold().TextColor(t.Text)
						srcDisplay := a.folderSrcPath
						if srcDisplay == "" {
							srcDisplay = "尚未选择文件夹 (请点击右侧选择或拖入)"
						}
						col := t.TextMuted
						if a.folderSrcPath != "" {
							col = t.Accent
						}
						ui.Text(c, srcDisplay).FontSize(12).TextColor(col).MaxLines(1)
					})

					if ui.Button(c, "选择源文件夹").Disabled(a.folderIsProcessing).Clicked() {
						go func() {
							paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
								Title:     "选择待转换图片所在的源文件夹",
								Directory: true,
							})
							if err == nil && len(paths) > 0 {
								a.update(func() {
									a.setFolderSrc(paths[0])
								})
							}
						}()
					}
				})

				// 存储模式切换：另存到新目录 vs 直接替换原文件
				SectionHeader(c, "输出目标与存储模式", ui.Hex("#2563eb"))
				ui.Row(c).Gap(10).Children(func() {
					// 模式 1: 另存新目录
					modeSaveNew := ui.Row(c).
						Padding(8, 14).
						Radius(8).
						Border(1, t.Border).
						Cursor(ui.CursorPointer).
						AlignItems(ui.Center).
						Gap(8)
					if !a.folderReplaceOrig {
						modeSaveNew.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
					} else {
						modeSaveNew.Background(t.Surface)
					}
					if modeSaveNew.Clicked() {
						a.folderReplaceOrig = false
					}
					modeSaveNew.Children(func() {
						col := t.Text
						if !a.folderReplaceOrig {
							col = ui.RGB(255, 255, 255)
						}
						ui.Text(c, "另存至指定新目录 (推荐)").FontSize(12).Bold().TextColor(col)
					})

					// 模式 2: 直接替换原文件
					modeReplace := ui.Row(c).
						Padding(8, 14).
						Radius(8).
						Border(1, t.Border).
						Cursor(ui.CursorPointer).
						AlignItems(ui.Center).
						Gap(8)
					if a.folderReplaceOrig {
						modeReplace.Background(ui.Hex("#dc2626")).Border(1, ui.Hex("#dc2626"))
					} else {
						modeReplace.Background(t.Surface)
					}
					if modeReplace.Clicked() {
						a.folderReplaceOrig = true
					}
					modeReplace.Children(func() {
						col := t.Text
						if a.folderReplaceOrig {
							col = ui.RGB(255, 255, 255)
						}
						ui.Text(c, "直接替换原文件 (原图将被删除/覆盖)").FontSize(12).Bold().TextColor(col)
					})
				})

				// 针对模式展开的配置项
				if !a.folderReplaceOrig {
					// 另存模式下的目标目录和保持目录树
					ui.Row(c).
						Padding(10, 14).
						Radius(8).
						Background(t.Surface).
						Border(1, t.Border).
						AlignItems(ui.Center).
						Justify(ui.SpaceBetween).
						Children(func() {
							displayOut := a.folderOutPath
							if displayOut == "" {
								if a.folderSrcPath != "" {
									displayOut = a.folderSrcPath + "_converted"
								} else {
									displayOut = "默认存至: 原目录同级 [原文件夹名]_converted"
								}
							}
							ui.Column(c).Gap(2).Grow(1).Children(func() {
								ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
									ui.Text(c, "转换后图片保存目录:").FontSize(11).TextColor(t.TextMuted)
									if a.folderSrcPath != "" && !a.folderOutCustom {
										PillBadge(c, "默认原目录旁")
									}
								})
								ui.Text(c, displayOut).FontSize(12).Bold().TextColor(t.Text).MaxLines(1)
							})

							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								if a.folderOutCustom && a.folderSrcPath != "" {
									if ui.Button(c, "恢复默认").Disabled(a.folderIsProcessing).Clicked() {
										a.folderOutPath = a.folderSrcPath + "_converted"
										a.folderOutCustom = false
									}
								}

								if ui.Button(c, "更改保存目录").Disabled(a.folderIsProcessing).Clicked() {
									go func() {
										paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
											Title:     "选择全量图片保存目录",
											Directory: true,
										})
										if err == nil && len(paths) > 0 {
											a.update(func() {
												a.folderOutPath = paths[0]
												a.folderOutCustom = true
											})
										}
									}()
								}
							})
						})

					ui.Checkbox(c, &a.folderKeepTree, "保持原有的子目录层级结构 (镜像目录树)")
				} else {
					// 替换模式下的警告提示
					ui.Row(c).
						Padding(12, 16).
						Radius(8).
						Background(ui.RGBA(239, 68, 68, 25)).
						Border(1, ui.Hex("#ef4444")).
						AlignItems(ui.Center).
						Gap(12).
						Children(func() {
							ui.Text(c, "⚠️").FontSize(24)
							ui.Column(c).Gap(4).Grow(1).Children(func() {
								ui.Text(c, "高风险提示：直接替换原文件模式").FontSize(13).Bold().TextColor(ui.Hex("#dc2626"))
								ui.Text(c, "1. 与目标格式同名的文件将被直接覆盖写入。\n2. 格式变更的旧文件（如原 PNG 转 WebP 成功后）原文件将被永久删除！\n3. 此操作不可撤销，请务必确认重要原图已做好备份。").
									FontSize(11).
									TextColor(t.Text)
							})
						})
				}
			})
		})

		// 2. Filter & Skip Strategy Card
		Card(c, "遍历与过滤规则", nil, nil, func() {
			ui.Column(c).Gap(10).Children(func() {
				ui.Row(c).Gap(20).Wrap().Children(func() {
					ui.Checkbox(c, &a.folderRecursive, "递归遍历所有子文件夹 (深度扫描)")
					ui.Checkbox(c, &a.folderSkipSame, "跳过同格式文件的重新压缩 (另存时直接复制，替换时跳过)")
				})
				ui.Text(c, "• 开启后：另存新目录时将对同格式图片直接快速复制（零画质损失并保持相册完整）；替换模式下则完全跳过。").
					FontSize(11).
					TextColor(t.TextMuted)
				ui.Text(c, "• 遇到不受支持的文件格式或损坏图片，系统将自动静默跳过，不会中断任务。").
					FontSize(11).
					TextColor(t.TextMuted)
			})
		})

		// 3. Encoder Configuration Card (复用)
		a.renderEncoderSettingsCard(c)

		// 4. Execution & Progress Card
		Card(c, "执行全量转换", nil, nil, func() {
			ui.Column(c).Gap(12).Children(func() {
				// 进度条与状态展示
				if a.folderShowProgress {
					ui.Column(c).Gap(6).Children(func() {
						ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								if a.folderIsProcessing {
									ui.Spinner(c)
								}
								ui.Textf(c, "当前状态: %s", a.folderCurFile).FontSize(12).TextColor(t.Accent)
							})
							ui.Textf(c, "%d / %d (%d%%)", a.folderDoneCount, a.folderTotalCount, a.folderProgress).
								FontSize(12).
								Bold().
								TextColor(t.Accent)
						})
						ui.Progress(c, float64(a.folderProgress)/100.0)

						// 统计数量徽标
						ui.Row(c).Gap(12).Padding(4, 0, 0).Children(func() {
							ui.Textf(c, "成功: %d", a.folderSuccessCount).FontSize(11).TextColor(ui.Hex("#16a34a"))
							if a.folderCopiedCount > 0 {
								ui.Textf(c, "(含免转码直通复制: %d)", a.folderCopiedCount).FontSize(11).TextColor(ui.Hex("#2563eb"))
							}
							if a.folderSkippedCount > 0 {
								ui.Textf(c, "跳过: %d", a.folderSkippedCount).FontSize(11).TextColor(t.TextMuted)
							}
							ui.Textf(c, "失败: %d", a.folderFailCount).FontSize(11).TextColor(ui.Hex("#dc2626"))
						})
					})
				}

				// 开始按钮
				ui.Row(c).Justify(ui.Center).Padding(6, 0, 0).Children(func() {
					btnText := "开始全量批量转换"
					if a.folderReplaceOrig {
						btnText = "⚠️ 开始全量替换原文件"
					}
					btn := ui.PrimaryButton(c, btnText).
						Width(280).
						Height(42).
						Disabled(a.folderIsProcessing || a.folderSrcPath == "")

					if btn.Clicked() {
						if a.folderReplaceOrig {
							a.folderConfirmModal = true
						} else {
							a.startFolderBatchTransform(c)
						}
					}
				})
			})
		})

		// 5. 危险操作确认弹窗
		a.renderFolderConfirmModal(c)
	})
}

func (a *App) renderFolderConfirmModal(c *ui.Context) {
	if !a.folderConfirmModal {
		return
	}
	t := c.Theme()

	ui.Modal(c, &a.folderConfirmModal, func() {
		ui.Column(c).
			Width(440).
			Padding(20).
			Gap(16).
			Children(func() {
				// 头部
				ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "⚠️").FontSize(22)
					ui.Text(c, "全量替换原文件确认").FontSize(16).Bold().TextColor(ui.Hex("#dc2626"))
				})

				// 提示正文
				ui.Column(c).Gap(8).Children(func() {
					ui.Text(c, "您即将以【直接替换原文件】模式批量转换以下目录中的所有图片：").
						FontSize(12).
						TextColor(t.Text)
					ui.Text(c, a.folderSrcPath).
						FontSize(12).
						Bold().
						TextColor(t.Accent).
						MaxLines(2)
					ui.Text(c, "重要须知：\n1. 与目标格式扩展名相同的文件将被直接覆盖写入。\n2. 格式变更的旧文件（如原 png 转为 webp 成功后）将被永久删除。\n3. 此操作不可撤销，请务必确认重要数据已做好备份！").
						FontSize(11).
						TextColor(t.TextMuted)
				})

				// 底部按钮
				ui.Row(c).Justify(ui.End).Gap(10).Children(func() {
					if ui.Button(c, "取消").Clicked() {
						a.folderConfirmModal = false
					}
					confirmBtn := ui.Button(c, "我已知晓风险，确认替换").
						Background(ui.Hex("#dc2626")).
						Border(1, ui.Hex("#dc2626"))
					if confirmBtn.Clicked() {
						a.folderConfirmModal = false
						a.startFolderBatchTransform(c)
					}
				})
			})
	})
}

func (a *App) startFolderBatchTransform(c *ui.Context) {
	if a.folderSrcPath == "" {
		c.Toast("请先选择源文件夹")
		return
	}
	finfo, err := os.Stat(a.folderSrcPath)
	if err != nil || !finfo.IsDir() {
		c.Toast("源文件夹路径无效或不存在")
		return
	}
	targetOut := a.folderOutPath
	if targetOut == "" && a.folderSrcPath != "" {
		targetOut = a.folderSrcPath + "_converted"
	}
	if !a.folderReplaceOrig && targetOut == "" {
		c.Toast("请设置保存目标目录")
		return
	}
	if !a.folderReplaceOrig && filepath.Clean(targetOut) == filepath.Clean(a.folderSrcPath) {
		c.Toast("另存目标目录不能与源文件夹相同，若要原地替换请选择【直接替换原文件】模式")
		return
	}

	a.folderIsProcessing = true
	a.folderShowProgress = true
	a.folderProgress = 0
	a.folderDoneCount = 0
	a.folderTotalCount = 0
	a.folderSuccessCount = 0
	a.folderCopiedCount = 0
	a.folderFailCount = 0
	a.folderSkippedCount = 0
	a.folderCurFile = "正在扫描文件夹..."

	folderImgOpts := a.imgOpts
	// 文件夹全量格式转换必须保留原图原始尺寸，绝不执行单图模式下残留的缩放或裁剪
	folderImgOpts.Op = services.OpOriginal
	folderImgOpts.Width = 0
	folderImgOpts.Height = 0
	folderImgOpts.CropX = 0
	folderImgOpts.CropY = 0
	folderImgOpts.CropWidth = 0
	folderImgOpts.CropHeight = 0
	folderImgOpts.Percent = 100

	opts := services.FolderBatchOptions{
		SrcDir:          a.folderSrcPath,
		OutDir:          targetOut,
		ReplaceOriginal: a.folderReplaceOrig,
		Recursive:       a.folderRecursive,
		KeepDirTree:     a.folderKeepTree,
		SkipSameFormat:  a.folderSkipSame,
		ImageOpts:       folderImgOpts,
	}

	go func() {
		res, err := services.BatchConvertFolder(opts, func(done, total int, current string) {
			pct := 0
			if total > 0 {
				pct = done * 100 / total
			}
			a.update(func() {
				a.folderProgress = pct
				a.folderDoneCount = done
				a.folderTotalCount = total
				a.folderCurFile = filepath.Base(current)
			})
		})

		a.update(func() {
			a.folderIsProcessing = false
			if err != nil {
				a.Toast(fmt.Sprintf("批量转换失败: %v", err))
				return
			}
			a.folderSuccessCount = res.SuccessFiles
			a.folderCopiedCount = res.CopiedFiles
			a.folderFailCount = res.FailedFiles
			a.folderSkippedCount = res.SkippedFiles
			a.folderCurFile = "全量转换完成"

			toastMsg := fmt.Sprintf("全量转换完成! 成功 %d", res.SuccessFiles)
			if res.CopiedFiles > 0 {
				toastMsg += fmt.Sprintf(" (含直通复制 %d)", res.CopiedFiles)
			}
			if res.SkippedFiles > 0 {
				toastMsg += fmt.Sprintf(", 跳过 %d", res.SkippedFiles)
			}
			if res.FailedFiles > 0 {
				toastMsg += fmt.Sprintf(", 失败 %d", res.FailedFiles)
			}
			a.Toast(toastMsg)
		})
	}()
}

func (a *App) openCropModal(c *ui.Context) {
	if len(a.imgFiles) == 0 {
		c.Toast("请先添加一张图片文件")
		return
	}
	if len(a.imgFiles) > 1 {
		c.Toast("自由裁剪仅支持单张图片，请保留一张")
		return
	}

	info := a.imgInfos[0]
	a.imgCropOrigW = info.Width
	a.imgCropOrigH = info.Height
	a.imgCropTarget = info.Path

	// 如果未设定过选区，默认取 80% 居中选区
	if a.imgOpts.CropWidth <= 0 || a.imgOpts.CropHeight <= 0 {
		w := int(float64(info.Width) * 0.8)
		h := int(float64(info.Height) * 0.8)
		if w <= 0 {
			w = info.Width
		}
		if h <= 0 {
			h = info.Height
		}
		a.imgCropTempW = w
		a.imgCropTempH = h
		a.imgCropTempX = (info.Width - w) / 2
		a.imgCropTempY = (info.Height - h) / 2
	} else {
		a.imgCropTempW = a.imgOpts.CropWidth
		a.imgCropTempH = a.imgOpts.CropHeight
		a.imgCropTempX = a.imgOpts.CropX
		a.imgCropTempY = a.imgOpts.CropY
	}

	a.imgCropModalOpen = true
}

func (a *App) renderCropModal(c *ui.Context) {
	if !a.imgCropModalOpen || len(a.imgThumbs) == 0 || a.imgThumbs[0] == nil {
		return
	}
	t := c.Theme()

	ui.Modal(c, &a.imgCropModalOpen, func() {
		ui.Column(c).
			Width(520).
			Padding(18).
			Gap(14).
			Children(func() {
				// 1. Header
				ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, IconImage).Size(20, 20).TextColor(t.Accent)
						ui.Text(c, "自由裁剪可视选区").FontSize(15).Bold().TextColor(t.Text)
					})
					PillBadge(c, fmt.Sprintf("原图: %d × %d px", a.imgCropOrigW, a.imgCropOrigH))
				})

				// 2. Interactive Preview Canvas
				origW := float32(max(a.imgCropOrigW, 1))
				origH := float32(max(a.imgCropOrigH, 1))
				maxW, maxH := float32(480), float32(260)
				scale := min(maxW/origW, maxH/origH)
				dispW := origW * scale
				dispH := origH * scale

				// Clamp crop coords
				if a.imgCropTempW > a.imgCropOrigW {
					a.imgCropTempW = a.imgCropOrigW
				}
				if a.imgCropTempH > a.imgCropOrigH {
					a.imgCropTempH = a.imgCropOrigH
				}
				if a.imgCropTempX < 0 {
					a.imgCropTempX = 0
				}
				if a.imgCropTempY < 0 {
					a.imgCropTempY = 0
				}
				if a.imgCropTempX+a.imgCropTempW > a.imgCropOrigW {
					a.imgCropTempX = a.imgCropOrigW - a.imgCropTempW
				}
				if a.imgCropTempY+a.imgCropTempH > a.imgCropOrigH {
					a.imgCropTempY = a.imgCropOrigH - a.imgCropTempH
				}

				boxX := float32(a.imgCropTempX) * scale
				boxY := float32(a.imgCropTempY) * scale
				boxW := float32(a.imgCropTempW) * scale
				boxH := float32(a.imgCropTempH) * scale

				ui.Row(c).
					Justify(ui.Center).
					Children(func() {
						cursor := ui.CursorCrosshair
						if a.imgCropDragMode == 1 {
							cursor = ui.CursorGrab
						} else if a.imgCropDragMode == 2 {
							cursor = ui.CursorResizeNWSE
						}

						board := ui.Box(c).
							Size(dispW, dispH).
							Radius(8).
							Background(ui.Hex("#0f172a")).
							Clip().
							Cursor(cursor)

						// 平滑连续拖拽事件处理器 (支持选区平移、右下角手柄缩放、空白处框选)
						board.HandleInput(func(ev ui.InputEvent) bool {
							switch ev.Kind {
							case ui.InputPointerDown:
								if ev.Button != 0 {
									return false
								}
								a.imgCropDragStartX = ev.X
								a.imgCropDragStartY = ev.Y
								a.imgCropInitX = a.imgCropTempX
								a.imgCropInitY = a.imgCropTempY
								a.imgCropInitW = a.imgCropTempW
								a.imgCropInitH = a.imgCropTempH

								// 检查是否点在右下角手柄区域 (选区右下角 22x22 像素热区)
								handleSize := float32(22)
								handleX := boxX + boxW - handleSize
								handleY := boxY + boxH - handleSize
								if ev.X >= handleX && ev.X <= boxX+boxW+6 && ev.Y >= handleY && ev.Y <= boxY+boxH+6 {
									a.imgCropDragMode = 2 // Resize SE Corner
									return true
								}

								// 检查是否点在选区框内部 (平移拖动模式)
								if ev.X >= boxX && ev.X <= boxX+boxW && ev.Y >= boxY && ev.Y <= boxY+boxH {
									a.imgCropDragMode = 1 // Move Box
									return true
								}

								// 点在外部空白处：开始拖拽划出新选区
								a.imgCropDragMode = 3 // Draw New Box
								origX := int(ev.X / scale)
								origY := int(ev.Y / scale)
								if origX < 0 {
									origX = 0
								}
								if origY < 0 {
									origY = 0
								}
								if origX > a.imgCropOrigW {
									origX = a.imgCropOrigW
								}
								if origY > a.imgCropOrigH {
									origY = a.imgCropOrigH
								}
								a.imgCropTempX = origX
								a.imgCropTempY = origY
								a.imgCropTempW = 10
								a.imgCropTempH = 10
								return true

							case ui.InputPointerMove:
								if a.imgCropDragMode == 0 {
									return false
								}
								dx := ev.X - a.imgCropDragStartX
								dy := ev.Y - a.imgCropDragStartY
								origDx := int(dx / scale)
								origDy := int(dy / scale)

								switch a.imgCropDragMode {
								case 1: // 选区整体平移，绝对平滑连续跟踪
									newX := a.imgCropInitX + origDx
									newY := a.imgCropInitY + origDy
									if newX < 0 {
										newX = 0
									}
									if newY < 0 {
										newY = 0
									}
									if newX+a.imgCropInitW > a.imgCropOrigW {
										newX = a.imgCropOrigW - a.imgCropInitW
									}
									if newY+a.imgCropInitH > a.imgCropOrigH {
										newY = a.imgCropOrigH - a.imgCropInitH
									}
									if newX < 0 {
										newX = 0
									}
									if newY < 0 {
										newY = 0
									}
									a.imgCropTempX = newX
									a.imgCropTempY = newY
									return true

								case 2: // 右下角缩放手柄拖拽
									newW := a.imgCropInitW + origDx
									newH := a.imgCropInitH + origDy
									if newW < 20 {
										newW = 20
									}
									if newH < 20 {
										newH = 20
									}
									if a.imgCropInitX+newW > a.imgCropOrigW {
										newW = a.imgCropOrigW - a.imgCropInitX
									}
									if a.imgCropInitY+newH > a.imgCropOrigH {
										newH = a.imgCropOrigH - a.imgCropInitY
									}
									a.imgCropTempW = newW
									a.imgCropTempH = newH
									return true

								case 3: // 画布直接拖拽划定新选区矩形
									startX := int(a.imgCropDragStartX / scale)
									startY := int(a.imgCropDragStartY / scale)
									curX := int(ev.X / scale)
									curY := int(ev.Y / scale)

									minX := min(startX, curX)
									minY := min(startY, curY)
									maxX := max(startX, curX)
									maxY := max(startY, curY)

									if minX < 0 {
										minX = 0
									}
									if minY < 0 {
										minY = 0
									}
									if maxX > a.imgCropOrigW {
										maxX = a.imgCropOrigW
									}
									if maxY > a.imgCropOrigH {
										maxY = a.imgCropOrigH
									}

									w := maxX - minX
									h := maxY - minY
									if w >= 10 && h >= 10 {
										a.imgCropTempX = minX
										a.imgCropTempY = minY
										a.imgCropTempW = w
										a.imgCropTempH = h
									}
									return true
								}

							case ui.InputPointerUp:
								a.imgCropDragMode = 0
								return true
							}
							return false
						})

						board.Children(func() {
							// 原图底图
							ui.Image(c, a.imgThumbs[0]).Size(dispW, dispH)

							// 半透明暗色蒙层
							ui.Box(c).Absolute().Fill().Background(ui.RGBA(0, 0, 0, 0.5))

							// 高亮选区框 (内含微调虚线与九宫格)
							ui.Box(c).
								Absolute().
								Left(boxX).
								Top(boxY).
								Size(boxW, boxH).
								Border(2, ui.Hex("#3b82f6")).
								Background(ui.RGBA(59, 130, 246, 0.12)).
								Cursor(ui.CursorGrab).
								Children(func() {
									// 九宫格参考线
									ui.Box(c).Absolute().LeftPercent(33.3).Top(0).Bottom(0).Width(1).Background(ui.RGBA(255, 255, 255, 0.3))
									ui.Box(c).Absolute().LeftPercent(66.6).Top(0).Bottom(0).Width(1).Background(ui.RGBA(255, 255, 255, 0.3))
									ui.Box(c).Absolute().TopPercent(33.3).Left(0).Right(0).Height(1).Background(ui.RGBA(255, 255, 255, 0.3))
									ui.Box(c).Absolute().TopPercent(66.6).Left(0).Right(0).Height(1).Background(ui.RGBA(255, 255, 255, 0.3))

									// 右下角拖动手柄 (Resize Handle)
									ui.Box(c).
										Absolute().
										Right(-4).
										Bottom(-4).
										Size(12, 12).
										Radius(3).
										Background(ui.Hex("#ffffff")).
										Border(2, ui.Hex("#2563eb")).
										Shadow(0, 2, 4, 0, ui.RGBA(0, 0, 0, 0.3)).
										Cursor(ui.CursorResizeNWSE)
								})
						})
					})

				// 交互指引提示
				ui.Row(c).Justify(ui.Center).Children(func() {
					ui.Text(c, "💡 选区内按住鼠标可平滑拖动平移；拖拽右下角方块可缩放大小；在空白处拖拽可直接框选新区域。").
						FontSize(11).
						TextColor(t.TextMuted)
				})

				// 3. 预设比例快捷按钮
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "快捷预设:").FontSize(11).Bold().TextColor(t.TextMuted)

					// 1:1
					if ui.Button(c, "1:1 正方").Clicked() {
						side := min(a.imgCropOrigW, a.imgCropOrigH)
						a.imgCropTempW = side
						a.imgCropTempH = side
						a.imgCropTempX = (a.imgCropOrigW - side) / 2
						a.imgCropTempY = (a.imgCropOrigH - side) / 2
					}
					// 4:3
					if ui.Button(c, "4:3 标准").Clicked() {
						w := a.imgCropOrigW
						h := w * 3 / 4
						if h > a.imgCropOrigH {
							h = a.imgCropOrigH
							w = h * 4 / 3
						}
						a.imgCropTempW = w
						a.imgCropTempH = h
						a.imgCropTempX = (a.imgCropOrigW - w) / 2
						a.imgCropTempY = (a.imgCropOrigH - h) / 2
					}
					// 16:9
					if ui.Button(c, "16:9 宽屏").Clicked() {
						w := a.imgCropOrigW
						h := w * 9 / 16
						if h > a.imgCropOrigH {
							h = a.imgCropOrigH
							w = h * 16 / 9
						}
						a.imgCropTempW = w
						a.imgCropTempH = h
						a.imgCropTempX = (a.imgCropOrigW - w) / 2
						a.imgCropTempY = (a.imgCropOrigH - h) / 2
					}
					// 80% 居中
					if ui.Button(c, "居中 80%").Clicked() {
						w := int(float64(a.imgCropOrigW) * 0.8)
						h := int(float64(a.imgCropOrigH) * 0.8)
						a.imgCropTempW = w
						a.imgCropTempH = h
						a.imgCropTempX = (a.imgCropOrigW - w) / 2
						a.imgCropTempY = (a.imgCropOrigH - h) / 2
					}
					// 100% 全图
					if ui.Button(c, "全选 100%").Clicked() {
						a.imgCropTempX = 0
						a.imgCropTempY = 0
						a.imgCropTempW = a.imgCropOrigW
						a.imgCropTempH = a.imgCropOrigH
					}
				})

				// 4. 精细微调滑块 (起点与尺寸)
				ui.Column(c).Gap(8).Padding(4, 0).Children(func() {
					// X, Y
					ui.Row(c).Gap(16).Children(func() {
						ui.Column(c).Grow(1).Gap(4).Children(func() {
							ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
								ui.Text(c, "起点 X 偏移").FontSize(11).TextColor(t.TextMuted)
								ui.Textf(c, "%d px", a.imgCropTempX).FontSize(11).Bold().TextColor(t.Text)
							})
							xSlider := float64(a.imgCropTempX)
							maxX := float64(max(a.imgCropOrigW-a.imgCropTempW, 0))
							if ui.Slider(c, &xSlider, 0, maxX).Changed() {
								a.imgCropTempX = int(xSlider)
							}
						})

						ui.Column(c).Grow(1).Gap(4).Children(func() {
							ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
								ui.Text(c, "起点 Y 偏移").FontSize(11).TextColor(t.TextMuted)
								ui.Textf(c, "%d px", a.imgCropTempY).FontSize(11).Bold().TextColor(t.Text)
							})
							ySlider := float64(a.imgCropTempY)
							maxY := float64(max(a.imgCropOrigH-a.imgCropTempH, 0))
							if ui.Slider(c, &ySlider, 0, maxY).Changed() {
								a.imgCropTempY = int(ySlider)
							}
						})
					})

					// Width, Height
					ui.Row(c).Gap(16).Children(func() {
						ui.Column(c).Grow(1).Gap(4).Children(func() {
							ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
								ui.Text(c, "选区宽度").FontSize(11).TextColor(t.TextMuted)
								ui.Textf(c, "%d px", a.imgCropTempW).FontSize(11).Bold().TextColor(t.Accent)
							})
							wSlider := float64(a.imgCropTempW)
							if ui.Slider(c, &wSlider, 10, float64(a.imgCropOrigW)).Changed() {
								a.imgCropTempW = int(wSlider)
								if a.imgCropTempX+a.imgCropTempW > a.imgCropOrigW {
									a.imgCropTempX = a.imgCropOrigW - a.imgCropTempW
								}
							}
						})

						ui.Column(c).Grow(1).Gap(4).Children(func() {
							ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
								ui.Text(c, "选区高度").FontSize(11).TextColor(t.TextMuted)
								ui.Textf(c, "%d px", a.imgCropTempH).FontSize(11).Bold().TextColor(t.Accent)
							})
							hSlider := float64(a.imgCropTempH)
							if ui.Slider(c, &hSlider, 10, float64(a.imgCropOrigH)).Changed() {
								a.imgCropTempH = int(hSlider)
								if a.imgCropTempY+a.imgCropTempH > a.imgCropOrigH {
									a.imgCropTempY = a.imgCropOrigH - a.imgCropTempH
								}
							}
						})
					})
				})

				// 5. Footer Actions
				ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Padding(4, 0, 0).Children(func() {
					ui.Textf(c, "最终选区: %d × %d (起点: %d, %d)", a.imgCropTempW, a.imgCropTempH, a.imgCropTempX, a.imgCropTempY).
						FontSize(12).
						Bold().
						TextColor(ui.Hex("#16a34a"))

					ui.Row(c).Gap(8).Children(func() {
						if ui.Button(c, "取消").Clicked() {
							a.imgCropModalOpen = false
						}
						applyBtn := ui.PrimaryButton(c, "确定应用裁剪")
						if applyBtn.Clicked() {
							a.imgOpts.CropX = a.imgCropTempX
							a.imgOpts.CropY = a.imgCropTempY
							a.imgOpts.CropWidth = a.imgCropTempW
							a.imgOpts.CropHeight = a.imgCropTempH
							a.imgCropModalOpen = false
							c.Toast(fmt.Sprintf("已设置裁剪选区: %d × %d", a.imgOpts.CropWidth, a.imgOpts.CropHeight))
						}
					})
				})
			})
	})
}
