package srvs

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/png"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/chai2010/webp"
	"github.com/disintegration/gift"
	"github.com/disintegration/imaging"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/golib/slicez"
	"github.com/welllog/olog"
	"github.com/welllog/otool/internal/errx"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type ImageInfo struct {
	Name         string
	Format       string
	Width        int
	Height       int
	Size         string
	Frames       int
	ThumbWidth   int
	ThumbHeight  int
	Thumbnail    string
	Path         string
	NoSuffixName string
}

const (
	Original    = iota // 原图尺寸
	FixedWidth         // 固定宽度
	MaxWidth           // 最大宽度
	FixedWH            // 固定宽高
	MaxWH              // 最大宽高
	FixedHeight        // 固定高度
	MaxHeight          // 最大高度
	Percentage         // 百分比
)

type Image struct {
	Ctx      context.Context
	cache    image.Image
	gifCache *gif.GIF
	format   string
	pathName string
}

func (i *Image) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(i.Ctx, runtime.OpenDialogOptions{
		Title:           "Please select image file",
		ShowHiddenFiles: true,
		Filters: []runtime.FileFilter{
			{
				DisplayName: "Image Files (*.jpg, *.png, *.gif, *.bmp, *.tiff, *.webp)",
				Pattern:     "*.jpg;*.jpeg;*.png;*.gif;*.bmp;*.tiff;*.tif;*.webp",
			},
		},
	})
}

func (i *Image) Decode(pathName string) (*ImageInfo, error) {
	format, err := formatFromFilename(pathName)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(pathName)
	if err != nil {
		return nil, errx.Log(err)
	}

	f, err := os.Open(pathName)
	if err != nil {
		return nil, errx.Log(err)
	}

	defer f.Close()

	frames := 1
	var img image.Image
	if format == imaging.GIF.String() {
		gifImg, err := gif.DecodeAll(f)
		if err != nil {
			return nil, errx.Log(err)
		}
		img = gifImg.Image[0]
		i.gifCache = gifImg
		frames = len(gifImg.Image)
	} else {
		img, err = decode(f)
		if err != nil {
			return nil, errx.Log(err)
		}
	}

	i.cache = img
	i.format = format
	i.pathName = pathName

	maxWidth := 400
	maxHeight := 320
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	if width > maxWidth || height > maxHeight { // 原尺寸小于指定参数，返回原图
		// 按最大宽高裁剪
		img = imaging.Fit(img, maxWidth, maxHeight, imaging.MitchellNetravali)
	}

	var buf bytes.Buffer
	if err := png.Encode(base64.NewEncoder(base64.StdEncoding, &buf), img); err != nil {
		return nil, errx.Log(err)
	}
	olog.Debugf("resize width: %d, height: %d, size: %s", img.Bounds().Dx(), img.Bounds().Dy(), prettySize(int64(buf.Len())))

	return &ImageInfo{
		Name:         filepath.Base(pathName),
		Format:       format,
		Width:        width,
		Height:       height,
		Size:         prettySize(info.Size()),
		Frames:       frames,
		ThumbWidth:   img.Bounds().Dx(),
		ThumbHeight:  img.Bounds().Dy(),
		Thumbnail:    buf.String(),
		Path:         filepath.Dir(pathName),
		NoSuffixName: filepath.Base(pathName[:len(pathName)-len(filepath.Ext(pathName))]),
	}, nil
}

func (i *Image) Crop(op, width, height, percent, encodeOption int, savePath, saveName string) error {
	savePathName := filepath.Join(savePath, saveName)

	_, err := os.Stat(savePathName)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return errx.Logf("file already exists: %s", savePathName)
	}

	olog.Debugf("savePathName: %s, width: %d height: %d percent: %d encodeOption: %d",
		savePathName, width, height, percent, encodeOption)

	img := crop(i.cache, op, width, height, percent)
	var opts []imaging.EncodeOption
	format, _ := formatFromFilename(saveName)
	switch format {
	case imaging.JPEG.String():
		opts = append(opts, imaging.JPEGQuality(encodeOption))
	case imaging.PNG.String():
		opts = append(opts, imaging.PNGCompressionLevel(png.CompressionLevel(encodeOption)))
	case imaging.GIF.String():
		opts = append(opts, imaging.GIFNumColors(encodeOption))
	case "WEBP":
		return webp.Save(savePathName, img, &webp.Options{Lossless: false, Quality: float32(encodeOption), Exact: false})
	}
	return imaging.Save(img, savePathName, opts...)
}

func (i *Image) CropGif(op, width, height, percent, encodeOption, drop int, savePath, saveName string, drawOnBefore bool) error {
	if len(i.gifCache.Image) == 1 {
		return i.Crop(op, width, height, percent, encodeOption, savePath, saveName)
	}

	savePathName := filepath.Join(savePath, saveName)

	_, err := os.Stat(savePathName)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return errx.Logf("file already exists: %s", savePathName)
	}

	olog.Debugf("savePathName: %s, width: %d height: %d percent: %d encodeOption: %d drop: %d, drawOnBefore: %v",
		savePathName, width, height, percent, encodeOption, drop, drawOnBefore)

	cropGif(i.gifCache, op, width, height, percent, drop, drawOnBefore)

	of, err := os.Create(savePathName)
	if err != nil {
		return errx.Log(err)
	}

	defer of.Close()

	return gif.EncodeAll(of, i.gifCache)
}

func decode(r io.Reader) (image.Image, error) {
	return imaging.Decode(r, imaging.AutoOrientation(true))
}

func decodeGif(r io.Reader) (*gif.GIF, error) {
	return gif.DecodeAll(r)
}

func formatFromFilename(filename string) (string, error) {
	ext := filepath.Ext(filename)
	if ext == ".webp" {
		return "WEBP", nil
	}
	f, err := imaging.FormatFromExtension(ext)
	if err != nil {
		return "", errx.Log(err)
	}
	return f.String(), nil
}

func cropGif(g *gif.GIF, op, width, height, percent, drop int, drawOnBefore bool) *gif.GIF {
	g = &gif.GIF{
		Image:           slicez.Copy(g.Image, 0, -1),
		Delay:           slicez.Copy(g.Delay, 0, -1),
		LoopCount:       g.LoopCount,
		Disposal:        slicez.Copy(g.Disposal, 0, -1),
		Config:          g.Config,
		BackgroundIndex: g.BackgroundIndex,
	}

	var filters []gift.Filter
	switch op {
	case Original:
	case FixedWH:
		filters = append(filters, gift.ResizeToFill(width, height, gift.NearestNeighborResampling, gift.CenterAnchor))
	case FixedWidth:
		filters = append(filters, gift.Resize(width, 0, gift.NearestNeighborResampling))
	case FixedHeight:
		filters = append(filters, gift.Resize(0, height, gift.NearestNeighborResampling))
	case Percentage:
		w := math.Ceil(float64(g.Config.Width) * float64(percent) / 100)
		h := math.Ceil(float64(g.Config.Height) * float64(percent) / 100)
		filters = append(filters, gift.ResizeToFill(int(w), int(h), gift.NearestNeighborResampling, gift.CenterAnchor))
	case MaxWidth:
		if g.Config.Width > width {
			filters = append(filters, gift.Resize(width, 0, gift.NearestNeighborResampling))
		}
	case MaxHeight:
		if g.Config.Height > height {
			filters = append(filters, gift.Resize(0, height, gift.NearestNeighborResampling))
		}
	case MaxWH:
		if g.Config.Width > width || g.Config.Height > height {
			filters = append(filters, gift.ResizeToFill(width, height, gift.NearestNeighborResampling, gift.CenterAnchor))
		}
	}

	filter := gift.New(filters...)

	if drawOnBefore {
		tmp := image.NewNRGBA(g.Image[0].Bounds())
		for i := range g.Image {
			// draw current frame over previous:
			gift.New().DrawAt(tmp, g.Image[i], g.Image[i].Bounds().Min, gift.OverOperator)
			dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), g.Image[i].Palette)
			filter.Draw(dst, tmp)
			if i == 0 {
				g.Config.Width = dst.Bounds().Dx()
				g.Config.Height = dst.Bounds().Dy()
			}

			if drop == 0 || (i+1)%drop != 0 {
				g.Image[i] = dst
			}
		}
	} else {
		for i := range g.Image {
			if drop > 0 && (i+1)%drop == 0 {
				continue
			}
			tmp := image.NewNRGBA(g.Image[i].Bounds())
			gift.New().DrawAt(tmp, g.Image[i], g.Image[i].Bounds().Min, gift.CopyOperator)
			dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), g.Image[i].Palette)
			filter.Draw(dst, tmp)
			g.Image[i] = dst
			if i == 0 {
				g.Config.Width = dst.Bounds().Dx()
				g.Config.Height = dst.Bounds().Dy()
			}
		}
	}
	return g
}

func crop(img image.Image, op, width, height, percent int) image.Image {
	switch op {
	case Original:
		return img
	case FixedWH:
		return imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	case FixedWidth:
		return imaging.Resize(img, width, 0, imaging.Lanczos)
	case FixedHeight:
		return imaging.Resize(img, 0, height, imaging.Lanczos)
	case Percentage:
		w := math.Ceil(float64(img.Bounds().Dx()) * float64(percent) / 100)
		h := math.Ceil(float64(img.Bounds().Dy()) * float64(percent) / 100)
		return imaging.Fill(img, int(w), int(h), imaging.Center, imaging.Lanczos)
	case MaxWidth:
		if img.Bounds().Dx() <= width {
			return img
		}
		return imaging.Resize(img, width, 0, imaging.Lanczos)
	case MaxHeight:
		if img.Bounds().Dy() <= height {
			return img
		}
		return imaging.Resize(img, 0, height, imaging.Lanczos)
	case MaxWH:
		if img.Bounds().Dx() <= width && img.Bounds().Dy() <= height {
			return img
		}
		return imaging.Fit(img, width, height, imaging.Lanczos)
	}
	return img
}

var (
	_kb int64 = 1 << 10
	_mb       = _kb << 10
	_gb       = _mb << 10
)

func prettySize(size int64) string {
	if size < _kb {
		return strconv.FormatInt(size, 10) + "B"
	}

	if size < _mb {
		return fmt.Sprintf("%.2f", float64(size)/float64(_kb)) + "KB"
	}

	if size < (_gb) {
		return fmt.Sprintf("%.2f", float64(size)/float64(_mb)) + "MB"
	}

	return fmt.Sprintf("%.2f", float64(size)/float64(_gb)) + "GB"
}
