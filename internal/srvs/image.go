package srvs

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/gif"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/disintegration/imaging"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/olog"
	"github.com/welllog/otool/internal/errx"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type ImageInfo struct {
	Name        string
	Format      string
	Width       int
	Height      int
	Size        string
	ThumbWidth  int
	ThumbHeight int
	Thumbnail   string
}

const (
	Original    = iota // 原图尺寸
	FixedWH            // 固定宽高
	FixedWidth         // 固定宽度
	FixedHeight        // 固定高度
	Percentage         // 百分比
	MaxWidth           // 最大宽度
	MaxHeight          // 最大高度
	MaxWH              // 最大宽高
)

type Image struct {
	Ctx    context.Context
	cache  image.Image
	format string
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

	img, err := decode(f)
	if err != nil {
		return nil, errx.Log(err)
	}
	i.cache = img
	i.format = format

	maxWidth := 400
	maxHeight := 320
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	if width > maxWidth || height > maxHeight { // 原尺寸小于指定参数，返回原图
		// 按最大宽高裁剪
		img = imaging.Fit(img, maxWidth, maxHeight, imaging.MitchellNetravali)
	}

	olog.Debugf("resize width: %d, height: %d", img.Bounds().Dx(), img.Bounds().Dy())

	var buf bytes.Buffer
	if err := png.Encode(base64.NewEncoder(base64.StdEncoding, &buf), img); err != nil {
		return nil, errx.Log(err)
	}

	return &ImageInfo{
		Name:        filepath.Base(pathName),
		Format:      format,
		Width:       width,
		Height:      height,
		Size:        prettySize(info.Size()),
		ThumbWidth:  img.Bounds().Dx(),
		ThumbHeight: img.Bounds().Dy(),
		Thumbnail:   buf.String(),
	}, nil
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
		return "webp", nil
	}
	f, err := imaging.FormatFromExtension(ext)
	if err != nil {
		return "", errx.Log(err)
	}
	return f.String(), nil
}

func cropGif(g gif.GIF, op, width, height, percent int) {
	//for _, v := g.Image {
	//
	//}
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

func prettySize(size int64) string {
	if size < 1024 {
		return strconv.FormatInt(size, 10) + "B"
	}

	if size < (1024 << 10) {
		return strconv.FormatInt(size>>10, 10) + "KB"
	}

	if size < (1024 << 20) {
		return strconv.FormatInt(size>>20, 10) + "MB"
	}

	return strconv.FormatInt(size>>30, 10) + "GB"
}
