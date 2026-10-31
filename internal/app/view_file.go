package app

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

type FileOption struct {
	Name  string
	Value string
}

var fileEncOptions = []FileOption{
	{Name: "AES-256 加密", Value: "encrypt"},
	{Name: "MD5", Value: "md5"},
	{Name: "SHA-1", Value: "sha1"},
	{Name: "SHA-224", Value: "sha224"},
	{Name: "SHA-256", Value: "sha256"},
	{Name: "SHA-384", Value: "sha384"},
	{Name: "SHA-512", Value: "sha512"},
}

var fileDecOptions = []FileOption{
	{Name: "AES-256 解密", Value: "decrypt"},
}

func (a *App) viewFile(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Gap(16).Children(func() {
		// 1. Mode Card
		Card(c, "处理模式", IconFile, nil, func() {
			ui.Column(c).Gap(14).Children(func() {
				// Encrypt & Hash
				SectionHeader(c, "加密与哈希 (Encrypt & Hash)", ui.Hex("#2563eb"))
				ui.Row(c).Gap(6).Wrap().Children(func() {
					for _, opt := range fileEncOptions {
						o := opt
						selected := a.fileOpt == o.Value
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

						if !a.fileIsProcessing && btn.Clicked() {
							a.fileOpt = o.Value
							a.fileOutText = ""
						}

						btn.Children(func() {
							textColor := t.Text
							if selected {
								textColor = ui.RGB(255, 255, 255)
							}
							ui.Text(c, o.Name).FontSize(12).Bold().TextColor(textColor)
						})
					}
				})

				// Decrypt
				SectionHeader(c, "解密与还原 (Decrypt)", ui.Hex("#16a34a"))
				ui.Row(c).Gap(6).Wrap().Children(func() {
					for _, opt := range fileDecOptions {
						o := opt
						selected := a.fileOpt == o.Value
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

						if !a.fileIsProcessing && btn.Clicked() {
							a.fileOpt = o.Value
							a.fileOutText = ""
						}

						btn.Children(func() {
							textColor := t.Text
							if selected {
								textColor = ui.RGB(255, 255, 255)
							}
							ui.Text(c, o.Name).FontSize(12).Bold().TextColor(textColor)
						})
					}
				})

				// Secret Key field if encrypt or decrypt
				if a.fileOpt == "encrypt" || a.fileOpt == "decrypt" {
					ui.Column(c).
						Padding(12, 14).
						Radius(8).
						Background(t.Surface).
						Border(1, t.Border).
						Gap(8).
						Children(func() {
							ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "密钥 (Secret Key)").FontSize(12).Bold().TextColor(t.Accent)
								eyeBtn := ui.Row(c).Gap(4).AlignItems(ui.Center).Cursor(ui.CursorPointer)
								if eyeBtn.Clicked() {
									a.fileSecretVisible = !a.fileSecretVisible
								}
								eyeBtn.Children(func() {
									if a.fileSecretVisible {
										ui.Icon(c, IconEyeOff).Size(14, 14).TextColor(t.TextMuted)
										ui.Text(c, "隐藏").FontSize(11).TextColor(t.TextMuted)
									} else {
										ui.Icon(c, IconEye).Size(14, 14).TextColor(t.TextMuted)
										ui.Text(c, "显示").FontSize(11).TextColor(t.TextMuted)
									}
								})
							})

							input := ui.TextInput(c, &a.fileSecretKey).
								Placeholder("在此输入加解密密钥...").
								Font("monospace")
							if !a.fileSecretVisible {
								input.Password()
							}
						})
				}
			})
		})

		// 2. File Selection Card (Supports Drag and Drop & Browse)
		Card(c, "文件路径", IconFolder, nil, func() {
			dropZone := ui.Row(c).
				Padding(12, 16).
				Radius(10).
				Background(t.Surface).
				Border(1, t.Border).
				AlignItems(ui.Center).
				Justify(ui.SpaceBetween)

			if files := dropZone.DroppedFiles(); len(files) > 0 {
				a.filePath = files[0]
				a.fileOutText = ""
				c.Toast("已选择文件: " + filepath.Base(files[0]))
			}
			if dropZone.FileDragOver() {
				dropZone.Border(2, t.Accent)
			}

			dropZone.Children(func() {
				ui.Column(c).Gap(4).Grow(1).Children(func() {
					ui.Text(c, "支持直接将文件拖拽至此，或点击右侧浏览").FontSize(11).TextColor(t.TextMuted)
					displayPath := a.filePath
					if displayPath == "" {
						displayPath = "未选择任何文件"
					}
					ui.Text(c, displayPath).FontSize(13).Bold().TextColor(t.Text).MaxLines(1)
				})

				if ui.Button(c, "浏览文件").Disabled(a.fileIsProcessing).Clicked() {
					go func() {
						paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
							Title: "选择需要处理的文件",
						})
						if err == nil && len(paths) > 0 {
							a.update(func() {
								a.filePath = paths[0]
								a.fileOutText = ""
							})
						}
					}()
				}
			})
		})

		// 3. Progress Bar (if processing)
		if a.fileShowProgress {
			Card(c, "处理进度", nil, nil, func() {
				ui.Column(c).Gap(8).Children(func() {
					ui.Row(c).Justify(ui.SpaceBetween).Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Spinner(c)
							ui.Text(c, "正在处理中...").FontSize(12).TextColor(t.Accent)
						})
						ui.Textf(c, "%d%%", a.fileProgress).FontSize(13).Bold().TextColor(t.Accent)
					})
					ui.Progress(c, float64(a.fileProgress)/100.0)
				})
			})
		}

		// 4. Output Result (for Hash modes)
		if a.fileOutText != "" {
			Card(c, "计算结果", nil, func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					PillBadge(c, fmt.Sprintf("LEN: %d", len([]rune(a.fileOutText))))
					CopyButton(c, a.fileOutText)
				})
			}, func() {
				ui.TextArea(c, &a.fileOutText).
					ReadOnly(true).
					Height(80).
					Font("monospace")
			})
		}

		// 5. Execution Button
		ui.Row(c).Justify(ui.Center).Padding(8, 0, 0).Children(func() {
			btn := ui.PrimaryButton(c, "立即开始处理").
				Width(240).
				Height(42).
				Disabled(a.fileIsProcessing || a.filePath == "")

			if btn.Clicked() {
				a.startFileProcessing(c)
			}
		})

		// Modal Confirmation for Encryption
		if a.fileConfirmModal {
			ui.Modal(c, &a.fileConfirmModal, func() {
				ui.Column(c).Gap(14).Padding(8).Children(func() {
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, IconWarn).Size(28, 28).TextColor(ui.Hex("#dc2626"))
						ui.Text(c, "确认加密该文件?").FontSize(18).Bold().TextColor(t.Text)
					})

					ui.Text(c, "加密操作将直接替换原文件。如果密钥丢失，文件内容将永久无法找回，请务必妥善保管密钥！").
						FontSize(13).
						TextColor(t.TextMuted)

					ui.Textf(c, "目标文件: %s", filepath.Base(a.filePath)).
						FontSize(12).
						Bold().
						TextColor(t.Text)

					ui.Row(c).Gap(10).Justify(ui.End).Padding(8, 0, 0).Children(func() {
						if ui.Button(c, "取消").Clicked() {
							a.fileConfirmModal = false
						}
						confirmBtn := ui.Button(c, "确认并开始加密")
						confirmBtn.Background(ui.Hex("#dc2626")).TextColor(ui.RGB(255, 255, 255))
						if confirmBtn.Clicked() {
							a.fileConfirmModal = false
							a.executeFileEncrypt()
						}
					})
				})
			})
		}
	})
}

func (a *App) startFileProcessing(c *ui.Context) {
	if a.filePath == "" {
		c.Toast("请先选择需要处理的文件")
		return
	}

	if a.fileOpt == "encrypt" {
		if a.fileSecretKey == "" {
			c.Toast("请填写文件加密密钥")
			return
		}
		a.fileConfirmModal = true
		return
	}

	if a.fileOpt == "decrypt" {
		if a.fileSecretKey == "" {
			c.Toast("请填写文件解密密钥")
			return
		}
		a.executeFileDecrypt()
		return
	}

	// Hash calculation
	a.executeFileHash()
}

func (a *App) executeFileEncrypt() {
	a.fileIsProcessing = true
	a.fileShowProgress = true
	a.fileProgress = 0
	filePath := a.filePath
	secret := a.fileSecretKey

	go func() {
		err := a.cryptoSrv.EncryptFile(filePath, secret, func(pct int64) {
			a.update(func() {
				a.fileProgress = pct
			})
		})

		a.update(func() {
			a.fileIsProcessing = false
			a.fileShowProgress = false
			if err != nil {
				a.Toast("文件加密失败: " + err.Error())
			} else {
				a.Toast("文件加密处理完成")
			}
		})
	}()
}

func (a *App) executeFileDecrypt() {
	a.fileIsProcessing = true
	a.fileShowProgress = true
	a.fileProgress = 0
	filePath := a.filePath
	secret := a.fileSecretKey

	go func() {
		err := a.cryptoSrv.DecryptFile(filePath, secret, func(pct int64) {
			a.update(func() {
				a.fileProgress = pct
			})
		})

		a.update(func() {
			a.fileIsProcessing = false
			a.fileShowProgress = false
			if err != nil {
				a.Toast("文件解密失败: " + err.Error())
			} else {
				a.Toast("文件解密处理完成")
			}
		})
	}()
}

func (a *App) executeFileHash() {
	a.fileIsProcessing = true
	filePath := a.filePath
	op := a.fileOpt

	go func() {
		var res string
		var err error

		switch op {
		case "md5":
			res, err = a.cryptoSrv.Md5File(filePath)
		case "sha1":
			res, err = a.cryptoSrv.Sha1File(filePath)
		case "sha224":
			res, err = a.cryptoSrv.Sha224File(filePath)
		case "sha256":
			res, err = a.cryptoSrv.Sha256File(filePath)
		case "sha384":
			res, err = a.cryptoSrv.Sha384File(filePath)
		case "sha512":
			res, err = a.cryptoSrv.Sha512File(filePath)
		default:
			err = fmt.Errorf("未知哈希操作: %s", op)
		}

		a.update(func() {
			a.fileIsProcessing = false
			if err != nil {
				a.Toast("哈希计算失败: " + err.Error())
			} else {
				a.fileOutText = res
				a.Toast("计算完成")
			}
		})
	}()
}
