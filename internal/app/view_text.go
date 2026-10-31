package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

type TextOption struct {
	Name  string
	Value string
	Group string
}

var textEncOptions = []TextOption{
	{Name: "AES-256", Value: "opensslAesEnc", Group: "加密"},
	{Name: "Base64", Value: "base64Enc", Group: "编码"},
	{Name: "URL", Value: "urlEnc", Group: "编码"},
	{Name: "Hex", Value: "hexEnc", Group: "编码"},
	{Name: "Octal", Value: "octEnc", Group: "编码"},
	{Name: "Unicode", Value: "unicodeEnc", Group: "编码"},
	{Name: "UTF-16", Value: "utf16Enc", Group: "编码"},
	{Name: "UPPER", Value: "upper", Group: "转换"},
	{Name: "MD5", Value: "md5", Group: "哈希"},
	{Name: "SHA-1", Value: "sha1", Group: "哈希"},
	{Name: "SHA-224", Value: "sha224", Group: "哈希"},
	{Name: "SHA-256", Value: "sha256", Group: "哈希"},
	{Name: "SHA-384", Value: "sha384", Group: "哈希"},
	{Name: "SHA-512", Value: "sha512", Group: "哈希"},
	{Name: "SHA-512/224", Value: "sha512_224", Group: "哈希"},
	{Name: "SHA-512/256", Value: "sha512_256", Group: "哈希"},
}

var textDecOptions = []TextOption{
	{Name: "AES-256", Value: "opensslAesDec", Group: "解密"},
	{Name: "Base64", Value: "base64Dec", Group: "解码"},
	{Name: "URL", Value: "urlDec", Group: "解码"},
	{Name: "Hex", Value: "hexDec", Group: "解码"},
	{Name: "Octal", Value: "octDec", Group: "解码"},
	{Name: "Unicode", Value: "unicodeDec", Group: "解码"},
	{Name: "UTF-16", Value: "utf16Dec", Group: "解码"},
	{Name: "lower", Value: "lower", Group: "转换"},
}

func isHashOpt(op string) bool {
	switch op {
	case "md5", "sha1", "sha224", "sha256", "sha384", "sha512", "sha512_224", "sha512_256":
		return true
	default:
		return false
	}
}

func (a *App) viewText(c *ui.Context) {
	t := c.Theme()

	ui.Column(c).Gap(14).Children(func() {
		// 1. Top Split: Input (Left) & Output (Right)
		ui.Row(c).Gap(14).AlignItems(ui.Start).Children(func() {
			// Left: Input Card (50% width)
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				Card(c, "输入源文本", IconText, func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						PillBadge(c, fmt.Sprintf("LEN: %d", len([]rune(a.textIn))))
						if ui.Button(c, "清空").Disabled(a.textIn == "").Clicked() {
							a.textIn = ""
							c.Toast("已清空输入内容")
						}
					})
				}, func() {
					ui.TextArea(c, &a.textIn).
						Placeholder("在此粘贴或输入需要处理的文本内容...").
						Height(160).
						Font("monospace")
				})
			})

			// Right: Output Card (50% width)
			ui.Column(c).Grow(1).Basis(0).Children(func() {
				Card(c, "处理结果", IconCheck, func() {
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						PillBadge(c, fmt.Sprintf("LEN: %d", len([]rune(a.textOut))))
						if ui.Button(c, "回填到输入").Disabled(a.textOut == "").Clicked() {
							a.textIn = a.textOut
							c.Toast("结果已回填到输入框")
						}
						CopyButton(c, a.textOut)
					})
				}, func() {
					ui.TextArea(c, &a.textOut).
						ReadOnly(true).
						Placeholder("处理结果将在此显示...").
						Height(160).
						Font("monospace")
				})
			})
		})

		// 2. Bottom: Operation & Configuration Card
		Card(c, "转换算法与配置", nil, func() {
			curOptName := a.textOpt
			for _, opt := range textEncOptions {
				if opt.Value == a.textOpt {
					curOptName = opt.Name
					break
				}
			}
			if curOptName == a.textOpt {
				for _, opt := range textDecOptions {
					if opt.Value == a.textOpt {
						curOptName = opt.Name
						break
					}
				}
			}
			PillBadge(c, "当前: "+curOptName)
		}, func() {
			ui.Column(c).Gap(12).Children(func() {
				// Operation Buttons Grid
				ui.Row(c).Gap(16).AlignItems(ui.Start).Children(func() {
					// Encoder column
					ui.Column(c).Grow(1).Basis(0).Gap(6).Children(func() {
						SectionHeader(c, "编码 / 加密 / 哈希", ui.Hex("#2563eb"))
						ui.Row(c).Gap(5).Wrap().Children(func() {
							for _, opt := range textEncOptions {
								o := opt
								selected := a.textOpt == o.Value
								btn := ui.Row(c).
									Padding(5, 10).
									Radius(7).
									Gap(4).
									AlignItems(ui.Center).
									Border(1, t.Border).
									Cursor(ui.CursorPointer)

								if selected {
									btn.Background(ui.Hex("#2563eb")).Border(1, ui.Hex("#2563eb"))
								} else {
									btn.Background(t.Surface)
								}

								if btn.Clicked() {
									a.textOpt = o.Value
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

					// Decoder column
					ui.Column(c).Grow(1).Basis(0).Gap(6).Children(func() {
						SectionHeader(c, "解码 / 解密 / 转换", ui.Hex("#16a34a"))
						ui.Row(c).Gap(5).Wrap().Children(func() {
							for _, opt := range textDecOptions {
								o := opt
								selected := a.textOpt == o.Value
								btn := ui.Row(c).
									Padding(5, 10).
									Radius(7).
									Gap(4).
									AlignItems(ui.Center).
									Border(1, t.Border).
									Cursor(ui.CursorPointer)

								if selected {
									btn.Background(ui.Hex("#16a34a")).Border(1, ui.Hex("#16a34a"))
								} else {
									btn.Background(t.Surface)
								}

								if btn.Clicked() {
									a.textOpt = o.Value
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

				// Hash HMAC Option
				if isHashOpt(a.textOpt) {
					ui.Row(c).
						Padding(8, 12).
						Radius(8).
						Background(t.Surface).
						Border(1, t.Border).
						AlignItems(ui.Center).
						Justify(ui.SpaceBetween).
						Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "附加 HMAC 签名").FontSize(12).Bold().TextColor(t.Text)
								ui.Text(c, "(使用密钥对哈希进行数字签名)").FontSize(11).TextColor(t.TextMuted)
							})
							ui.Checkbox(c, &a.textCheckHmac, "启用")
						})
				}

				// Secret Key Field (for AES or Hash+HMAC)
				needSecret := a.textOpt == "opensslAesEnc" || a.textOpt == "opensslAesDec" || (isHashOpt(a.textOpt) && a.textCheckHmac)
				if needSecret {
					ui.Column(c).
						Padding(10, 12).
						Radius(8).
						Background(t.Surface).
						Border(1, t.Border).
						Gap(6).
						Children(func() {
							ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Children(func() {
								ui.Text(c, "处理密钥 (Secret Key)").FontSize(11).Bold().TextColor(t.Accent)
								eyeBtn := ui.Row(c).Gap(4).AlignItems(ui.Center).Cursor(ui.CursorPointer)
								if eyeBtn.Clicked() {
									a.textSecretVisible = !a.textSecretVisible
								}
								eyeBtn.Children(func() {
									if a.textSecretVisible {
										ui.Icon(c, IconEyeOff).Size(13, 13).TextColor(t.TextMuted)
										ui.Text(c, "隐藏").FontSize(10).TextColor(t.TextMuted)
									} else {
										ui.Icon(c, IconEye).Size(13, 13).TextColor(t.TextMuted)
										ui.Text(c, "显示").FontSize(10).TextColor(t.TextMuted)
									}
								})
							})

							input := ui.TextInput(c, &a.textSecretKey).
								Placeholder("在此输入加解密或 HMAC 密钥...").
								Font("monospace")
							if !a.textSecretVisible {
								input.Password()
							}
						})
				}

				// Actions Row
				ui.Row(c).Justify(ui.SpaceBetween).AlignItems(ui.Center).Padding(4, 0, 0).Children(func() {
					if ui.Button(c, "清空全部 (含密钥)").Clicked() {
						a.textIn = ""
						a.textOut = ""
						a.textSecretKey = ""
						c.Toast("已重置所有文本与密钥")
					}

					if ui.PrimaryButton(c, "立即执行处理").Width(160).Height(36).Clicked() {
						a.executeTextTransform(c)
					}
				})
			})
		})
	})
}

func (a *App) executeTextTransform(c *ui.Context) {
	if strings.TrimSpace(a.textIn) == "" {
		c.Toast("请输入需要处理的文本")
		return
	}
	if a.textOpt == "" {
		c.Toast("请先选择一个操作")
		return
	}

	cs := a.cryptoSrv
	var res string
	var err error

	if isHashOpt(a.textOpt) && a.textCheckHmac {
		if a.textSecretKey == "" {
			c.Toast("使用 HMAC 请输入密钥")
			return
		}
		res, err = cs.Hmac(a.textIn, a.textSecretKey, a.textOpt)
	} else {
		switch a.textOpt {
		case "opensslAesEnc":
			if a.textSecretKey == "" {
				c.Toast("AES-256 加密需要输入密钥")
				return
			}
			res, err = cs.OpenSSLAesEnc(a.textIn, a.textSecretKey)
		case "opensslAesDec":
			if a.textSecretKey == "" {
				c.Toast("AES-256 解密需要输入密钥")
				return
			}
			res, err = cs.OpenSSLAesDec(a.textIn, a.textSecretKey)
		case "base64Enc":
			res = cs.Base64Enc(a.textIn)
		case "base64Dec":
			res = cs.Base64Dec(a.textIn)
		case "urlEnc":
			res = cs.UrlEnc(a.textIn)
		case "urlDec":
			res, err = cs.UrlDec(a.textIn)
		case "octEnc":
			res = cs.OctEnc(a.textIn)
		case "octDec":
			res = cs.OctDec(a.textIn)
		case "hexEnc":
			res = cs.HexEnc(a.textIn)
		case "hexDec":
			res = cs.HexDec(a.textIn)
		case "unicodeEnc":
			res = cs.UnicodeEnc(a.textIn)
		case "unicodeDec":
			res = cs.UnicodeDec(a.textIn)
		case "utf16Enc":
			res = cs.Utf16Enc(a.textIn)
		case "utf16Dec":
			res = cs.Utf16Dec(a.textIn)
		case "upper":
			res = cs.Upper(a.textIn)
		case "lower":
			res = cs.Lower(a.textIn)
		case "md5":
			res = cs.Md5(a.textIn)
		case "sha1":
			res = cs.Sha1(a.textIn)
		case "sha224":
			res = cs.Sha224(a.textIn)
		case "sha256":
			res = cs.Sha256(a.textIn)
		case "sha384":
			res = cs.Sha384(a.textIn)
		case "sha512":
			res = cs.Sha512(a.textIn)
		case "sha512_224":
			res = cs.Sha512_224(a.textIn)
		case "sha512_256":
			res = cs.Sha512_256(a.textIn)
		default:
			c.Toast("未知的操作选项")
			return
		}
	}

	if err != nil {
		c.Toast("处理失败: " + err.Error())
		return
	}

	a.textOut = res
	c.Toast("处理完成")
}

