package services

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/makiuchi-d/gozxing"
	gozxingqr "github.com/makiuchi-d/gozxing/qrcode"
	qrcode "github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"github.com/yeqown/go-qrcode/writer/standard/shapes"
	"github.com/welllog/golib/randz"
	"github.com/welllog/golib/strz"
)

var fileNameReplacer = strings.NewReplacer(
	"\\", "_",
	":", "_",
	"*", "_",
	"?", "_",
	"\"", "_",
	"<", "_",
	">", "_",
	"|", "_",
	"/", "_",
	"\n", "_",
	"\r", "_",
)

func sanitizeFileName(str string) string {
	str = strings.TrimSpace(str)
	return fileNameReplacer.Replace(str)
}

// QrShapeStyle defines the visual appearance of QR code blocks.
type QrShapeStyle int

const (
	QrShapeSquare  QrShapeStyle = 0 // 经典直角方块 (最高通用兼容性)
	QrShapeDots    QrShapeStyle = 1 // 现代精致圆点 (数据块圆点化 + 标准定位角)
	QrShapeRounded QrShapeStyle = 2 // 柔和圆角流动 (圆角定位角 + 圆角数据块)
)

// QrExportFormat defines the output file format.
type QrExportFormat string

const (
	QrFormatPNG  QrExportFormat = "png"
	QrFormatJPEG QrExportFormat = "jpeg"
	QrFormatJPG  QrExportFormat = "jpg"
	QrFormatSVG  QrExportFormat = "svg"
)

// QrGenerateOptions provides full configurations for generating a QR code.
type QrGenerateOptions struct {
	Text     string         // 二维码内容文本 / URL
	Style    QrShapeStyle   // 视觉样式：Square, Dots, Rounded
	Recover  int            // 容错率：0: L(7%), 1: M(15%), 2: Q(25%), 3: H(30%)
	Size     int            // 输出目标像素大小 (默认 256)
	Format   QrExportFormat // 导出格式："png", "jpeg", "svg"
	FgColor  string         // 前景色主题 Hex (如 "#000000", "#1e40af")
	LogoPath string         // 中心 Logo 图片路径 (可选)
	SaveDir  string         // 保存目录 (若不为空则自动持久化到本地)
}

// QrGenerateResult returns the generated assets and metadata.
type QrGenerateResult struct {
	SaveName   string // 保存的文件名 (含后缀)
	SavedPath  string // 完整持久化路径
	Bytes      []byte // 导出格式的实际数据流 (SVG 为文本字节，PNG/JPEG 为二进制图)
	PreviewPNG []byte // 专供 UI 实时预览渲染的高清 PNG 位图数据
}

type nopBufferCloser struct {
	*bytes.Buffer
}

func (n nopBufferCloser) Close() error {
	return nil
}

// QrGenerate is the unified entrypoint for creating high-compatibility, modern QR codes.
func QrGenerate(opts QrGenerateOptions) (*QrGenerateResult, error) {
	if strings.TrimSpace(opts.Text) == "" {
		return nil, errors.New("二维码内容不能为空")
	}

	if opts.Size <= 0 {
		opts.Size = 256
	}

	opts.Format = QrExportFormat(strings.ToLower(string(opts.Format)))
	if opts.Format == "" {
		opts.Format = QrFormatPNG
	}

	// 1. Error Correction Level mapping
	// 关键兼容性底线：一旦挂载 Logo，必须强制提升至 H(30%)，避免遮挡导致扫码失效
	ec := qrcode.ErrorCorrectionMedium
	if opts.LogoPath != "" {
		ec = qrcode.ErrorCorrectionHighest
	} else {
		switch opts.Recover {
		case 0:
			ec = qrcode.ErrorCorrectionLow
		case 1:
			ec = qrcode.ErrorCorrectionMedium
		case 2:
			ec = qrcode.ErrorCorrectionQuart
		case 3:
			ec = qrcode.ErrorCorrectionHighest
		default:
			ec = qrcode.ErrorCorrectionMedium
		}
	}

	qrc, err := qrcode.NewWith(opts.Text, qrcode.WithErrorCorrectionLevel(ec))
	if err != nil {
		return nil, fmt.Errorf("生成二维码矩阵失败: %w", err)
	}

	dim := qrc.Dimension()
	if dim <= 0 {
		dim = 33
	}
	// Calculate block pixel width to roughly target opts.Size
	qrWidth := opts.Size / (dim + 8)
	if qrWidth < 3 {
		qrWidth = 3
	}
	if qrWidth > 40 {
		qrWidth = 40
	}

	// 2. Prepare logo image if specified
	var logoImg image.Image
	if opts.LogoPath != "" {
		openedLogo, err := imaging.Open(opts.LogoPath)
		if err != nil {
			return nil, fmt.Errorf("加载 Logo 图片失败: %w", err)
		}
		// 动态等比缩放 Logo，严格保持在二维码尺寸的 18% 以内 (不超过 1/5 容错阈值)，保证 100% 扫码通过率
		maxLogoPix := int(float64(qrWidth*dim) * 0.18)
		if maxLogoPix < 16 {
			maxLogoPix = 16
		}
		if openedLogo.Bounds().Dx() > maxLogoPix || openedLogo.Bounds().Dy() > maxLogoPix {
			openedLogo = imaging.Fit(openedLogo, maxLogoPix, maxLogoPix, imaging.Lanczos)
		}
		logoImg = openedLogo
	}

	// 3. Normalize foreground color (default to classic black #000000)
	fgHex := strings.TrimSpace(opts.FgColor)
	if fgHex == "" {
		fgHex = "#000000"
	} else if !strings.HasPrefix(fgHex, "#") {
		fgHex = "#" + fgHex
	}

	buildStandardWriterOpts := func(isJPEG bool) []standard.ImageOption {
		targetFmt := standard.PNG_FORMAT
		if isJPEG {
			targetFmt = standard.JPEG_FORMAT
		}
		stdOpts := []standard.ImageOption{
			standard.WithQRWidth(uint8(qrWidth)),
			standard.WithBorderWidth(20),
			standard.WithBuiltinImageEncoder(targetFmt),
			standard.WithFgColorRGBHex(fgHex),
		}

		// Apply Shape Styles
		switch opts.Style {
		case QrShapeDots:
			// 数据块圆点化 (0.85 比例)，Finder Pattern 保留标准方形，确保 100% 扫码检出
			shapeDot := shapes.Assemble(shapes.SquareFinder(), shapes.CircleBlocks(0.85))
			stdOpts = append(stdOpts, standard.WithCustomShape(shapeDot))
		case QrShapeRounded:
			// 圆角 Finder + 圆角流动 LiquidBlock，现代优雅风格
			shapeRound := shapes.Assemble(shapes.RoundedFinder(), shapes.LiquidBlock())
			stdOpts = append(stdOpts, standard.WithCustomShape(shapeRound))
		default:
			// 标准方块：无需额外参数，标准直角渲染
		}

		// Apply Logo
		if logoImg != nil {
			stdOpts = append(stdOpts,
				standard.WithLogoImage(logoImg),
				standard.WithLogoSafeZone(),
			)
		}

		return stdOpts
	}

	// 4. Always generate a high-definition PNG buffer for UI preview
	previewBuf := &bytes.Buffer{}
	pngWriter := standard.NewWithWriter(nopBufferCloser{previewBuf}, buildStandardWriterOpts(false)...)
	if err := qrc.Save(pngWriter); err != nil {
		return nil, fmt.Errorf("渲染二维码 PNG 失败: %w", err)
	}
	previewPNGBytes := previewBuf.Bytes()

	// 5. Generate Target Output Bytes based on opts.Format
	var targetBytes []byte
	var ext string

	switch opts.Format {
	case QrFormatJPEG, QrFormatJPG:
		ext = ".jpg"
		jpegBuf := &bytes.Buffer{}
		jpegWriter := standard.NewWithWriter(nopBufferCloser{jpegBuf}, buildStandardWriterOpts(true)...)
		if err := qrc.Save(jpegWriter); err != nil {
			return nil, fmt.Errorf("渲染二维码 JPEG 失败: %w", err)
		}
		targetBytes = jpegBuf.Bytes()

	case QrFormatSVG:
		ext = ".svg"
		svgBuf := &bytes.Buffer{}
		sw := &svgWriter{
			buf:      svgBuf,
			style:    opts.Style,
			size:     opts.Size,
			fgColor:  fgHex,
			logoImg:  logoImg,
			logoPath: opts.LogoPath,
		}
		if err := qrc.Save(sw); err != nil {
			return nil, fmt.Errorf("渲染二维码 SVG 失败: %w", err)
		}
		targetBytes = svgBuf.Bytes()

	case QrFormatPNG:
		fallthrough
	default:
		ext = ".png"
		targetBytes = previewPNGBytes
	}

	// 6. Assemble Result & Handle Saving if SaveDir is requested
	prefix := sanitizeFileName(strz.Sub(opts.Text, 0, 10))
	if prefix == "" {
		prefix = "qrcode"
	}
	saveName := fmt.Sprintf("%s_%s_%d_%d%s", prefix, randz.String(5), opts.Recover, opts.Size, ext)
	var fullSavedPath string

	if opts.SaveDir != "" {
		fullSavedPath = filepath.Join(opts.SaveDir, saveName)
		_, err := os.Stat(fullSavedPath)
		if err == nil || errors.Is(err, fs.ErrExist) {
			return nil, fmt.Errorf("文件已存在: %s", fullSavedPath)
		}
		if err := os.WriteFile(fullSavedPath, targetBytes, 0644); err != nil {
			return nil, fmt.Errorf("保存二维码文件失败: %w", err)
		}
	}

	return &QrGenerateResult{
		SaveName:   saveName,
		SavedPath:  fullSavedPath,
		Bytes:      targetBytes,
		PreviewPNG: previewPNGBytes,
	}, nil
}

// QrEncodeInMemory generates a standard QR code PNG image in memory (backward compatible).
func QrEncodeInMemory(text string, recover, size int) ([]byte, error) {
	res, err := QrGenerate(QrGenerateOptions{
		Text:    text,
		Style:   QrShapeSquare,
		Recover: recover,
		Size:    size,
		Format:  QrFormatPNG,
	})
	if err != nil {
		return nil, err
	}
	return res.PreviewPNG, nil
}

// QrEncodeAndSave generates a standard QR code, saves it to savePath if provided, and returns filename and bytes.
func QrEncodeAndSave(text, savePath string, recover, size int) (string, []byte, error) {
	res, err := QrGenerate(QrGenerateOptions{
		Text:    text,
		Style:   QrShapeSquare,
		Recover: recover,
		Size:    size,
		Format:  QrFormatPNG,
		SaveDir: savePath,
	})
	if err != nil {
		return "", nil, err
	}
	return res.SaveName, res.PreviewPNG, nil
}

// QrDecodeBytes decodes QR code payloads from image bytes using industrial-grade gozxing.
func QrDecodeBytes(b []byte) (string, error) {
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return "", fmt.Errorf("图像解码失败: %w", err)
	}

	bmp, err := gozxing.NewBinaryBitmapFromImage(img)
	if err != nil {
		return "", fmt.Errorf("构建二值化位图失败: %w", err)
	}

	hints := make(map[gozxing.DecodeHintType]interface{})
	hints[gozxing.DecodeHintType_TRY_HARDER] = true
	hints[gozxing.DecodeHintType_ALSO_INVERTED] = true

	reader := gozxingqr.NewQRCodeReader()
	res, err := reader.Decode(bmp, hints)
	if err != nil {
		return "", fmt.Errorf("未能识别出二维码内容: %w", err)
	}

	return res.GetText(), nil
}

// QrDecodeFile decodes a QR code directly from a file path.
func QrDecodeFile(pathName string) (string, error) {
	b, err := os.ReadFile(pathName)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %w", err)
	}
	return QrDecodeBytes(b)
}

// ==================== PURE GO SVG WRITER ====================

type svgWriter struct {
	buf      *bytes.Buffer
	style    QrShapeStyle
	size     int
	fgColor  string
	logoImg  image.Image
	logoPath string
}

func (s *svgWriter) Write(mat qrcode.Matrix) error {
	w := mat.Width()
	h := mat.Height()
	padding := 4
	totalW := w + padding*2
	totalH := h + padding*2

	fg := s.fgColor
	if fg == "" {
		fg = "#000000"
	}

	s.buf.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 %d %d" width="%d" height="%d">`+"\n", totalW, totalH, s.size, s.size))
	s.buf.WriteString(fmt.Sprintf(`  <rect width="%d" height="%d" fill="#ffffff"/>`+"\n", totalW, totalH))

	// Finder Pattern boundaries (7x7 blocks at corners)
	isFinder := func(x, y int) bool {
		if x < 7 && y < 7 {
			return true
		}
		if x >= w-7 && y < 7 {
			return true
		}
		if x < 7 && y >= h-7 {
			return true
		}
		return false
	}

	// Center Logo Safe Zone boundaries (middle ~20% of area)
	hasLogo := s.logoImg != nil
	centerX, centerY := w/2, h/2
	logoRadius := w / 5 // Radius of 1/5 of grid
	isLogoZone := func(x, y int) bool {
		if !hasLogo {
			return false
		}
		return x >= centerX-logoRadius && x <= centerX+logoRadius &&
			y >= centerY-logoRadius && y <= centerY+logoRadius
	}

	for y := 0; y < h; y++ {
		row := mat.Row(y)
		for x := 0; x < w; x++ {
			if !row[x].IsSet() {
				continue
			}
			// Skip blocks covered by center Logo safe zone
			if isLogoZone(x, y) {
				continue
			}

			posX := x + padding
			posY := y + padding

			if isFinder(x, y) {
				// Finder blocks remain square or gently rounded for reliable scanning
				if s.style == QrShapeRounded {
					s.buf.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="1" height="1" rx="0.2" ry="0.2" fill="%s"/>`+"\n", posX, posY, fg))
				} else {
					s.buf.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="1" height="1" fill="%s"/>`+"\n", posX, posY, fg))
				}
			} else {
				switch s.style {
				case QrShapeDots:
					s.buf.WriteString(fmt.Sprintf(`  <circle cx="%.2f" cy="%.2f" r="0.43" fill="%s"/>`+"\n", float64(posX)+0.5, float64(posY)+0.5, fg))
				case QrShapeRounded:
					s.buf.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="1" height="1" rx="0.32" ry="0.32" fill="%s"/>`+"\n", posX, posY, fg))
				default:
					s.buf.WriteString(fmt.Sprintf(`  <rect x="%d" y="%d" width="1" height="1" fill="%s"/>`+"\n", posX, posY, fg))
				}
			}
		}
	}

	// Embed Center Logo into SVG if present
	if hasLogo {
		// Draw white background card for logo with safe margin
		safeW := float64(logoRadius*2 + 1)
		safeX := float64(centerX-logoRadius+padding) - 0.2
		safeY := float64(centerY-logoRadius+padding) - 0.2
		s.buf.WriteString(fmt.Sprintf(`  <rect x="%.2f" y="%.2f" width="%.2f" height="%.2f" rx="0.8" ry="0.8" fill="#ffffff"/>`+"\n", safeX, safeY, safeW+0.4, safeW+0.4))

		// Embed image as base64 PNG
		logoPNG := &bytes.Buffer{}
		if err := imaging.Encode(logoPNG, s.logoImg, imaging.PNG); err == nil {
			b64 := base64.StdEncoding.EncodeToString(logoPNG.Bytes())
			s.buf.WriteString(fmt.Sprintf(`  <image x="%.2f" y="%.2f" width="%.2f" height="%.2f" xlink:href="data:image/png;base64,%s"/>`+"\n", safeX+0.3, safeY+0.3, safeW-0.2, safeW-0.2, b64))
		}
	}

	s.buf.WriteString("</svg>\n")
	return nil
}

func (s *svgWriter) Close() error {
	return nil
}
