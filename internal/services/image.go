package services

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	webp "github.com/SeriousBug/webp-go-pure/std"
	"github.com/disintegration/gift"
	"github.com/disintegration/imaging"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gen2brain/avif"
	gav1davif "github.com/gen2brain/gav1d/avif"
	"github.com/welllog/golib/randz"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

const (
	OpOriginal    = 0 // 保持原尺寸
	OpFixedWidth  = 1 // 固定宽度
	OpMaxWidth    = 2 // 最大宽度
	OpFixedWH     = 3 // 固定宽高
	OpMaxWH       = 4 // 最大宽高
	OpFixedHeight = 5 // 固定高度
	OpMaxHeight   = 6 // 最大高度
	OpPercentage  = 7 // 百分比
	OpCustomCrop  = 8 // 自定义裁剪
)

type ImageInfo struct {
	Name         string
	Path         string
	Format       string
	Width        int
	Height       int
	Size         string
	SizeBytes    int64
	Frames       int
	Thumbnail    image.Image
	NoSuffixName string
}

type ImageOptions struct {
	Op                   int
	Encoder              string
	OutPath              string
	Width                int
	Height               int
	Percent              int
	JpgQuality           int
	PngCompression       int
	GifNumColors         int
	GifDropRate          int
	GifDrawOnBefore      bool
	WebpLossless         bool
	WebpQuality          int
	WebpRgbInTransparent bool
	AvifQuality          int
	AvifQualityAlpha     int
	AvifSpeed            int
	AvifEncoder          string
	CropX                int
	CropY                int
	CropWidth            int
	CropHeight           int
}

func DefaultImageOptions() ImageOptions {
	return ImageOptions{
		Op:                   OpOriginal,
		Encoder:              "png",
		Percent:              100,
		JpgQuality:           95,
		PngCompression:       0,
		GifNumColors:         256,
		GifDropRate:          0,
		GifDrawOnBefore:      false,
		WebpLossless:         false,
		WebpQuality:          90,
		WebpRgbInTransparent: false,
		AvifQuality:          60,
		AvifQualityAlpha:     60,
		AvifSpeed:            10,
		AvifEncoder:          "auto",
	}
}

type FolderBatchOptions struct {
	SrcDir          string
	OutDir          string
	ReplaceOriginal bool
	Recursive       bool
	SkipSameFormat  bool
	KeepDirTree     bool
	ImageOpts       ImageOptions
}

type FolderBatchResult struct {
	TotalFiles   int
	SuccessFiles int
	FailedFiles  int
	SkippedFiles int
	CopiedFiles  int
}

func DefaultFolderBatchOptions() FolderBatchOptions {
	return FolderBatchOptions{
		ReplaceOriginal: false,
		Recursive:       true,
		SkipSameFormat:  true,
		KeepDirTree:     true,
		ImageOpts:       DefaultImageOptions(),
	}
}

// DecodeImageInfo decodes an image file and returns its metadata and a thumbnail image.
func DecodeImageInfo(pathName string) (*ImageInfo, error) {
	info, err := os.Stat(pathName)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(pathName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	mimetype.SetLimit(16)
	mTyp, err := mimetype.DetectReader(f)
	if err != nil {
		return nil, err
	}
	_, _ = f.Seek(0, io.SeekStart)

	frames := 1
	var img image.Image
	if mTyp.String() == "image/gif" {
		gifImg, err := gif.DecodeAll(f)
		if err != nil {
			return nil, err
		}
		if len(gifImg.Image) == 0 {
			return nil, errors.New("无效的 GIF 文件: 帧数为 0")
		}
		img = gifImg.Image[0]
		frames = len(gifImg.Image)
	} else if mTyp.String() == "image/avif" {
		img, err = gav1davif.Decode(f)
		if err != nil {
			// Fallback
			_, _ = f.Seek(0, io.SeekStart)
			img, err = imaging.Decode(f, imaging.AutoOrientation(true))
		}
	} else {
		img, err = imaging.Decode(f, imaging.AutoOrientation(true))
	}
	if err != nil {
		return nil, err
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()

	// Create thumbnail (max 300x200)
	var thumb image.Image = img
	if w > 300 || h > 200 {
		thumb = imaging.Fit(img, 300, 200, imaging.MitchellNetravali)
	}

	ext := filepath.Ext(pathName)
	noSuffix := filepath.Base(pathName)
	if len(ext) > 0 {
		noSuffix = strings.TrimSuffix(noSuffix, ext)
	}

	return &ImageInfo{
		Name:         filepath.Base(pathName),
		Path:         pathName,
		Format:       strings.ToUpper(strings.TrimPrefix(mTyp.Extension(), ".")),
		Width:        w,
		Height:       h,
		Size:         FormatFileSize(info.Size()),
		SizeBytes:    info.Size(),
		Frames:       frames,
		Thumbnail:    thumb,
		NoSuffixName: noSuffix,
	}, nil
}

// TransformImage transforms a single image file according to ImageOptions and saves it to OutPath.
func TransformImage(srcPath string, opts ImageOptions) (string, error) {
	if opts.OutPath == "" {
		return "", errors.New("未指定输出路径")
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("打开图片失败: %w", err)
	}
	defer f.Close()

	mimetype.SetLimit(16)
	mTyp, err := mimetype.DetectReader(f)
	if err != nil {
		return "", fmt.Errorf("检测格式失败: %w", err)
	}
	_, _ = f.Seek(0, io.SeekStart)

	outputName := OutImageName(srcPath, opts)
	of, err := os.Create(outputName)
	if err != nil {
		return "", fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer of.Close()

	// Special handling for animated GIF when target is GIF
	if mTyp.String() == "image/gif" && opts.Encoder == "gif" {
		gifImg, err := gif.DecodeAll(f)
		if err == nil && len(gifImg.Image) > 1 {
			gifImg = cropGif(gifImg, &opts)
			if err := gif.EncodeAll(of, gifImg); err != nil {
				_ = os.Remove(outputName)
				return "", fmt.Errorf("编码 GIF 失败: %w", err)
			}
			return outputName, nil
		}
		_, _ = f.Seek(0, io.SeekStart)
	}

	var img image.Image
	if mTyp.String() == "image/avif" {
		img, err = gav1davif.Decode(f)
		if err != nil {
			_, _ = f.Seek(0, io.SeekStart)
			img, err = imaging.Decode(f, imaging.AutoOrientation(true))
		}
	} else {
		img, err = imaging.Decode(f, imaging.AutoOrientation(true))
	}
	if err != nil {
		_ = os.Remove(outputName)
		return "", fmt.Errorf("解码图片失败: %w", err)
	}
	_ = f.Close()

	// Apply Crop / Resize
	img = applyCrop(img, &opts)

	// Save with specific encoder
	switch opts.Encoder {
	case "jpg", "jpeg":
		err = imaging.Encode(of, img, imaging.JPEG, imaging.JPEGQuality(opts.JpgQuality))
	case "png":
		err = imaging.Encode(of, img, imaging.PNG, imaging.PNGCompressionLevel(png.CompressionLevel(opts.PngCompression)))
	case "gif":
		err = imaging.Encode(of, img, imaging.GIF, imaging.GIFNumColors(opts.GifNumColors))
	case "webp":
		err = webp.Encode(of, img, &webp.Options{Lossless: opts.WebpLossless, Quality: opts.WebpQuality})
	case "avif":
		err = encodeAVIF(of, img, &opts)
	case "bmp":
		err = imaging.Encode(of, img, imaging.BMP)
	case "tiff", "tif":
		err = imaging.Encode(of, img, imaging.TIFF)
	default:
		err = imaging.Encode(of, img, imaging.PNG)
	}

	if err != nil {
		_ = os.Remove(outputName)
		return "", fmt.Errorf("编码文件失败: %w", err)
	}

	return outputName, nil
}

// BatchTransform processes a list of image paths concurrently and reports progress.
func BatchTransform(
	paths []string,
	opts ImageOptions,
	onProgress func(done int, total int, current string),
) (successCount int, failCount int) {
	total := len(paths)
	if total == 0 {
		return 0, 0
	}

	sem := make(chan struct{}, 4) // Max 4 concurrent workers
	var mu sync.Mutex
	var wg sync.WaitGroup
	done := 0

	wg.Add(total)
	for _, p := range paths {
		sem <- struct{}{}
		go func(pathName string) {
			defer wg.Done()
			defer func() { <-sem }()

			_, err := TransformImage(pathName, opts)

			mu.Lock()
			done++
			curDone := done
			if err != nil {
				failCount++
			} else {
				successCount++
			}
			mu.Unlock()

			if onProgress != nil {
				onProgress(curDone, total, filepath.Base(pathName))
			}
		}(p)
	}

	// Wait for all workers to finish cleanly
	wg.Wait()
	return successCount, failCount
}

var supportedImageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
	".tiff": true,
	".tif":  true,
	".avif": true,
}

func IsSupportedImageExt(ext string) bool {
	return supportedImageExtensions[strings.ToLower(ext)]
}

// ScanImageFiles returns all supported image file paths in srcDir.
func ScanImageFiles(srcDir string, recursive bool) ([]string, error) {
	srcDir = filepath.Clean(srcDir)
	fi, err := os.Stat(srcDir)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s 不是有效的目录", srcDir)
	}

	var files []string
	err = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // Skip items with permission errors
		}
		if d.IsDir() {
			if !recursive && path != srcDir {
				return fs.SkipDir
			}
			return nil
		}
		if IsSupportedImageExt(filepath.Ext(d.Name())) {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// ConvertImageTo converts an image from srcPath directly to destPath using opts.
func ConvertImageTo(srcPath, destPath string, opts ImageOptions) error {
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	actualDest := destPath
	isSame := false
	if cleanSrc, cleanDst := filepath.Clean(srcPath), filepath.Clean(destPath); cleanSrc == cleanDst {
		isSame = true
		actualDest = filepath.Join(destDir, fmt.Sprintf(".otool_tmp_%d_%s", time.Now().UnixNano(), filepath.Base(destPath)))
	}

	f, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("打开图片失败: %w", err)
	}
	defer f.Close()

	mimetype.SetLimit(16)
	mTyp, err := mimetype.DetectReader(f)
	if err != nil {
		return fmt.Errorf("检测格式失败: %w", err)
	}
	_, _ = f.Seek(0, io.SeekStart)

	of, err := os.Create(actualDest)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer of.Close()

	// Special handling for animated GIF when target is GIF
	if mTyp.String() == "image/gif" && opts.Encoder == "gif" {
		gifImg, err := gif.DecodeAll(f)
		if err == nil && len(gifImg.Image) > 1 {
			gifImg = cropGif(gifImg, &opts)
			if err := gif.EncodeAll(of, gifImg); err != nil {
				_ = of.Close()
				_ = os.Remove(actualDest)
				return fmt.Errorf("编码 GIF 失败: %w", err)
			}
			_ = of.Close()
			if isSame {
				return os.Rename(actualDest, destPath)
			}
			return nil
		}
		_, _ = f.Seek(0, io.SeekStart)
	}

	var img image.Image
	if mTyp.String() == "image/avif" {
		img, err = gav1davif.Decode(f)
		if err != nil {
			_, _ = f.Seek(0, io.SeekStart)
			img, err = imaging.Decode(f, imaging.AutoOrientation(true))
		}
	} else {
		img, err = imaging.Decode(f, imaging.AutoOrientation(true))
	}
	if err != nil {
		_ = of.Close()
		_ = os.Remove(actualDest)
		return fmt.Errorf("解码图片失败: %w", err)
	}
	_ = f.Close()

	// Apply Crop / Resize
	img = applyCrop(img, &opts)

	// Save with specific encoder
	switch opts.Encoder {
	case "jpg", "jpeg":
		err = imaging.Encode(of, img, imaging.JPEG, imaging.JPEGQuality(opts.JpgQuality))
	case "png":
		err = imaging.Encode(of, img, imaging.PNG, imaging.PNGCompressionLevel(png.CompressionLevel(opts.PngCompression)))
	case "gif":
		err = imaging.Encode(of, img, imaging.GIF, imaging.GIFNumColors(opts.GifNumColors))
	case "webp":
		err = webp.Encode(of, img, &webp.Options{Lossless: opts.WebpLossless, Quality: opts.WebpQuality})
	case "avif":
		err = encodeAVIF(of, img, &opts)
	case "bmp":
		err = imaging.Encode(of, img, imaging.BMP)
	case "tiff", "tif":
		err = imaging.Encode(of, img, imaging.TIFF)
	default:
		err = imaging.Encode(of, img, imaging.PNG)
	}

	if err != nil {
		_ = of.Close()
		_ = os.Remove(actualDest)
		return fmt.Errorf("编码文件失败: %w", err)
	}

	_ = of.Close()
	if isSame {
		return os.Rename(actualDest, destPath)
	}
	return nil
}

func normalizeExt(enc string) string {
	switch strings.ToLower(enc) {
	case "jpeg":
		return ".jpg"
	case "tif":
		return ".tiff"
	default:
		return "." + strings.ToLower(enc)
	}
}

func copyFile(src, dest string) error {
	if filepath.Clean(src) == filepath.Clean(dest) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return err
	}
	return df.Sync()
}

// BatchConvertFolder converts all supported images in srcDir per FolderBatchOptions.
func BatchConvertFolder(
	opts FolderBatchOptions,
	onProgress func(done, total int, current string),
) (FolderBatchResult, error) {
	var res FolderBatchResult
	files, err := ScanImageFiles(opts.SrcDir, opts.Recursive)
	if err != nil {
		return res, err
	}
	res.TotalFiles = len(files)
	if res.TotalFiles == 0 {
		return res, nil
	}

	// 文件夹批量格式转换必须保留原图原始尺寸，不执行缩放或裁剪
	opts.ImageOpts.Op = OpOriginal
	opts.ImageOpts.Width = 0
	opts.ImageOpts.Height = 0
	opts.ImageOpts.CropWidth = 0
	opts.ImageOpts.CropHeight = 0
	opts.ImageOpts.Percent = 100

	targetExt := normalizeExt(opts.ImageOpts.Encoder)

	type taskItem struct {
		src     string
		dest    string
		isSkip  bool
		isCopy  bool
		delOrig bool
	}

	var tasks []taskItem
	for _, f := range files {
		srcExt := strings.ToLower(filepath.Ext(f))
		isSameFormat := (srcExt == targetExt) || (targetExt == ".jpg" && srcExt == ".jpeg") || (targetExt == ".tiff" && srcExt == ".tif")

		baseName := strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		var destPath string
		delOrig := false

		if opts.ReplaceOriginal {
			destPath = filepath.Join(filepath.Dir(f), baseName+targetExt)
			if filepath.Clean(f) != filepath.Clean(destPath) {
				delOrig = true
			}
		} else {
			if opts.KeepDirTree {
				rel, _ := filepath.Rel(opts.SrcDir, f)
				relDir := filepath.Dir(rel)
				destPath = filepath.Join(opts.OutDir, relDir, baseName+targetExt)
			} else {
				destPath = filepath.Join(opts.OutDir, baseName+targetExt)
			}
		}

		if opts.SkipSameFormat && isSameFormat {
			if opts.ReplaceOriginal {
				// 替换模式下，同格式文件完全忽略，无需任何处理
				res.SkippedFiles++
				tasks = append(tasks, taskItem{src: f, isSkip: true})
				continue
			} else {
				// 另存新目录模式下，免重新转码压缩，直接复制文件到新目录对应层级
				tasks = append(tasks, taskItem{src: f, dest: destPath, isCopy: true})
				continue
			}
		}

		tasks = append(tasks, taskItem{src: f, dest: destPath, isSkip: false, delOrig: delOrig})
	}

	sem := make(chan struct{}, 4)
	var mu sync.Mutex
	var wg sync.WaitGroup
	done := 0

	for _, item := range tasks {
		t := item
		if t.isSkip {
			mu.Lock()
			done++
			curDone := done
			mu.Unlock()
			if onProgress != nil {
				onProgress(curDone, res.TotalFiles, filepath.Base(t.src)+" (跳过)")
			}
			continue
		}

		sem <- struct{}{}
		wg.Add(1)
		go func(task taskItem) {
			defer wg.Done()
			defer func() { <-sem }()

			var taskErr error
			if task.isCopy {
				taskErr = copyFile(task.src, task.dest)
			} else {
				taskErr = ConvertImageTo(task.src, task.dest, opts.ImageOpts)
			}

			mu.Lock()
			done++
			curDone := done
			if taskErr != nil {
				res.FailedFiles++
			} else {
				res.SuccessFiles++
				if task.isCopy {
					res.CopiedFiles++
				}
				if task.delOrig {
					_ = os.Remove(task.src)
				}
			}
			mu.Unlock()

			if onProgress != nil {
				suffix := ""
				if task.isCopy {
					suffix = " (免压缩直通)"
				}
				onProgress(curDone, res.TotalFiles, filepath.Base(task.src)+suffix)
			}
		}(t)
	}

	wg.Wait()
	return res, nil
}

func OutImageName(name string, opts ImageOptions) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	if len(ext) > 0 {
		base = strings.TrimSuffix(base, ext)
	}
	return filepath.Join(opts.OutPath, fmt.Sprintf("%s_%s.%s", base, randz.String(4), opts.Encoder))
}

func FormatFileSize(size int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	if size < kb {
		return strconv.FormatInt(size, 10) + " B"
	}
	if size < mb {
		return fmt.Sprintf("%.2f KB", float64(size)/float64(kb))
	}
	if size < gb {
		return fmt.Sprintf("%.2f MB", float64(size)/float64(mb))
	}
	return fmt.Sprintf("%.2f GB", float64(size)/float64(gb))
}

func customCropRect(b image.Rectangle, opts *ImageOptions) (image.Rectangle, bool) {
	if opts.CropWidth <= 0 || opts.CropHeight <= 0 {
		return image.Rectangle{}, false
	}
	x0 := max(b.Min.X, opts.CropX)
	y0 := max(b.Min.Y, opts.CropY)
	x1 := min(b.Max.X, opts.CropX+opts.CropWidth)
	y1 := min(b.Max.Y, opts.CropY+opts.CropHeight)
	if x1 <= x0 || y1 <= y0 {
		return image.Rectangle{}, false
	}
	return image.Rect(x0, y0, x1, y1), true
}

func applyCrop(img image.Image, opts *ImageOptions) image.Image {
	switch opts.Op {
	case OpOriginal:
		return img
	case OpFixedWH:
		return imaging.Fill(img, opts.Width, opts.Height, imaging.Center, imaging.Lanczos)
	case OpFixedWidth:
		return imaging.Resize(img, opts.Width, 0, imaging.Lanczos)
	case OpFixedHeight:
		return imaging.Resize(img, 0, opts.Height, imaging.Lanczos)
	case OpPercentage:
		w := math.Ceil(float64(img.Bounds().Dx()) * float64(opts.Percent) / 100)
		h := math.Ceil(float64(img.Bounds().Dy()) * float64(opts.Percent) / 100)
		return imaging.Fill(img, int(w), int(h), imaging.Center, imaging.Lanczos)
	case OpCustomCrop:
		rect, ok := customCropRect(img.Bounds(), opts)
		if !ok {
			return img
		}
		return imaging.Crop(img, rect)
	case OpMaxWidth:
		if img.Bounds().Dx() <= opts.Width {
			return img
		}
		return imaging.Resize(img, opts.Width, 0, imaging.Lanczos)
	case OpMaxHeight:
		if img.Bounds().Dy() <= opts.Height {
			return img
		}
		return imaging.Resize(img, 0, opts.Height, imaging.Lanczos)
	case OpMaxWH:
		if img.Bounds().Dx() <= opts.Width && img.Bounds().Dy() <= opts.Height {
			return img
		}
		return imaging.Fit(img, opts.Width, opts.Height, imaging.Lanczos)
	}
	return img
}

func cropGif(g *gif.GIF, opts *ImageOptions) *gif.GIF {
	if len(g.Image) == 0 {
		return g
	}
	c := &gif.GIF{
		Image:           make([]*image.Paletted, 0, len(g.Image)),
		Delay:           make([]int, 0, len(g.Delay)),
		Disposal:        make([]byte, 0, len(g.Disposal)),
		Config:          g.Config,
		BackgroundIndex: g.BackgroundIndex,
	}

	var filters []gift.Filter
	switch opts.Op {
	case OpOriginal:
	case OpFixedWH:
		filters = append(filters, gift.ResizeToFill(opts.Width, opts.Height, gift.NearestNeighborResampling, gift.CenterAnchor))
	case OpFixedWidth:
		filters = append(filters, gift.Resize(opts.Width, 0, gift.NearestNeighborResampling))
	case OpFixedHeight:
		filters = append(filters, gift.Resize(0, opts.Height, gift.NearestNeighborResampling))
	case OpPercentage:
		w := math.Ceil(float64(g.Config.Width) * float64(opts.Percent) / 100)
		h := math.Ceil(float64(g.Config.Height) * float64(opts.Percent) / 100)
		filters = append(filters, gift.ResizeToFill(int(w), int(h), gift.NearestNeighborResampling, gift.CenterAnchor))
	case OpCustomCrop:
		if rect, ok := customCropRect(image.Rect(0, 0, g.Config.Width, g.Config.Height), opts); ok {
			filters = append(filters, gift.Crop(rect))
		}
	case OpMaxWidth:
		if g.Config.Width > opts.Width {
			filters = append(filters, gift.Resize(opts.Width, 0, gift.NearestNeighborResampling))
		}
	case OpMaxHeight:
		if g.Config.Height > opts.Height {
			filters = append(filters, gift.Resize(0, opts.Height, gift.NearestNeighborResampling))
		}
	case OpMaxWH:
		if g.Config.Width > opts.Width || g.Config.Height > opts.Height {
			filters = append(filters, gift.ResizeToFit(opts.Width, opts.Height, gift.NearestNeighborResampling))
		}
	}

	filter := gift.New(filters...)
	delay := 0
	firstFrame := g.Image[0]
	g.Config.Width = filter.Bounds(firstFrame.Bounds()).Dx()
	g.Config.Height = filter.Bounds(firstFrame.Bounds()).Dy()

	if opts.GifDrawOnBefore {
		tmp := image.NewNRGBA(firstFrame.Bounds())
		for i := range g.Image {
			gift.New().DrawAt(tmp, g.Image[i], g.Image[i].Bounds().Min, gift.OverOperator)
			dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), g.Image[i].Palette)
			filter.Draw(dst, tmp)
			delay += g.Delay[i]

			if opts.GifDropRate == 0 || (i+1)%opts.GifDropRate != 0 {
				c.Image = append(c.Image, dst)
				c.Delay = append(c.Delay, delay)
				c.Disposal = append(c.Disposal, g.Disposal[i])
				delay = 0
			}
		}
	} else {
		for i := range g.Image {
			delay += g.Delay[i]
			if opts.GifDropRate > 0 && (i+1)%opts.GifDropRate == 0 {
				continue
			}
			tmp := image.NewNRGBA(firstFrame.Bounds())
			gift.New().DrawAt(tmp, g.Image[i], g.Image[i].Bounds().Min, gift.CopyOperator)
			dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), g.Image[i].Palette)
			filter.Draw(dst, tmp)

			c.Image = append(c.Image, dst)
			c.Delay = append(c.Delay, delay)
			c.Disposal = append(c.Disposal, g.Disposal[i])
			delay = 0
		}
	}
	return c
}

func gav1dSafeSize(w, h int) bool {
	return w > 0 && h > 0 && w <= 4096 && h <= 4096 && w*h <= 9_000_000
}

func encodeAVIF(w io.Writer, m image.Image, opts *ImageOptions) error {
	useGav1d := false
	switch opts.AvifEncoder {
	case "gav1d":
		useGav1d = true
	case "libavif":
		useGav1d = false
	default: // "auto"
		b := m.Bounds()
		useGav1d = isOpaque(m) && gav1dSafeSize(b.Dx(), b.Dy())
	}
	if useGav1d {
		var buf bytes.Buffer
		if err := gav1davif.Encode(&buf, m, gav1davif.EncodeOptions{
			Quality:      opts.AvifQuality,
			QualityAlpha: opts.AvifQualityAlpha,
			Speed:        opts.AvifSpeed,
		}); err == nil {
			if _, derr := gav1davif.Decode(bytes.NewReader(buf.Bytes())); derr == nil {
				_, werr := w.Write(buf.Bytes())
				return werr
			}
		}
	}
	return avif.Encode(w, m, avif.Options{
		Quality:      opts.AvifQuality,
		QualityAlpha: opts.AvifQualityAlpha,
		Speed:        opts.AvifSpeed,
	})
}

func isOpaque(m image.Image) bool {
	if o, ok := m.(interface{ Opaque() bool }); ok {
		return o.Opaque()
	}
	b := m.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := m.At(x, y).RGBA(); a != 0xffff {
				return false
			}
		}
	}
	return true
}

func extractPalette(img image.Image) color.Palette {
	bounds := img.Bounds()
	palette := make(color.Palette, 0, 256)
	colorMap := make(map[color.Color]struct{})

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := img.At(x, y)
			if _, exists := colorMap[c]; !exists {
				colorMap[c] = struct{}{}
				palette = append(palette, c)
				if len(palette) >= 256 {
					return palette
				}
			}
		}
	}
	return palette
}

