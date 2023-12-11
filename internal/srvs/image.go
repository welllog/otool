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
	"strings"

	"github.com/chai2010/webp"
	"github.com/disintegration/gift"
	"github.com/disintegration/imaging"
	"github.com/liyue201/goqr"
	"github.com/skip2/go-qrcode"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/welllog/golib/goz"
	"github.com/welllog/golib/mapz"
	"github.com/welllog/golib/randz"
	"github.com/welllog/golib/strz"
	"github.com/welllog/olog"
	"github.com/welllog/otool/internal/errx"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

type ImageInfo struct {
	Name         string `json:"name"`
	Format       string `json:"format"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	Size         string `json:"size"`
	Frames       int    `json:"frames"`
	ThumbWidth   int    `json:"thumbWidth"`
	ThumbHeight  int    `json:"thumbHeight"`
	Thumbnail    string `json:"thumbnail"`
	Path         string `json:"path"`
	NoSuffixName string `json:"noSuffixName"`
}

type ImageFile struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Body []byte `json:"body"`
}

type ImageOptions struct {
	Op                   int    `json:"op"`
	Encoder              string `json:"encoder"`
	OutPath              string `json:"outPath"`
	Width                int    `json:"width"`
	Height               int    `json:"height"`
	Percent              int    `json:"percent"`
	JpgQuality           int    `json:"jpgQuality"`
	PngCompression       int    `json:"pngCompression"`
	GifNumColors         int    `json:"gifNumColors"`
	GifDropRate          int    `json:"gifDropRate"`
	GifDrawOnBefore      bool   `json:"gifDrawOnBefore"`
	WebpLossless         bool   `json:"webpLossless"`
	WebpQuality          int    `json:"webpQuality"`
	WebpRgbInTransparent bool   `json:"webpRgbInTransparent"`
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
	Ctx context.Context
	Gow *goz.Limiter
	Kv  *mapz.SafeKV[string, any]
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
		frames = len(gifImg.Image)
	} else {
		img, err = decode(f)
		if err != nil {
			return nil, errx.Log(err)
		}
	}

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

func (i *Image) CropAndSave(file ImageFile, opts ImageOptions, filesNum int, eventName string) (bool, error) {
	var record *taskRecord
	var first bool
	i.Kv.Map(func(m map[string]any) {
		r, ok := m[eventName]
		if ok {
			record = r.(*taskRecord)
		} else {
			record = &taskRecord{
				progress: countProgress{count: 3 * filesNum, event: eventName, ctx: i.Ctx},
				errCh:    make(chan error, filesNum),
				name:     file.Name + "...",
			}
			record.done.Add(filesNum)
			m[eventName] = record
			first = true
		}
	})

	i.Gow.Go(func() {
		defer record.done.Done()

		outputName := outImageName(file.Name, opts)
		of, err := os.Create(outputName)
		if err != nil {
			record.errCh <- err
			notify(i.Ctx, NotifyEvent{
				Info: fmt.Sprintf("创建文件%s失败: %s", outputName, err.Error()),
				Type: "danger",
			})
			return
		}

		defer func() {
			_ = of.Close()
			if err != nil {
				record.errCh <- err
				_ = os.Remove(outputName)
			}
		}()

		var img image.Image
		if file.Type == "image/gif" && opts.Encoder == "gif" {
			var gifImg *gif.GIF
			gifImg, err = gif.DecodeAll(bytes.NewReader(file.Body))
			if err != nil {
				notify(i.Ctx, NotifyEvent{
					Info: fmt.Sprintf("解码%s失败: %s", file.Name, err.Error()),
					Type: "danger",
				})
				return
			}
			record.progress.Incr()

			if len(gifImg.Image) > 1 {
				gifImg = cropGif(gifImg, &opts)
				record.progress.Incr()

				err = gif.EncodeAll(of, gifImg)
				if err != nil {
					notify(i.Ctx, NotifyEvent{
						Info: fmt.Sprintf("编码%s失败: %s", file.Name, err.Error()),
						Type: "danger",
					})
					return
				}

				record.progress.Incr()
				return
			}
			img = gifImg.Image[0]
		} else {
			img, err = decode(bytes.NewReader(file.Body))
			if err != nil {
				notify(i.Ctx, NotifyEvent{
					Info: fmt.Sprintf("解码%s失败: %s", file.Name, err.Error()),
					Type: "danger",
				})
				return
			}

			record.progress.Incr()
		}

		img = crop(img, &opts)
		record.progress.Incr()

		switch opts.Encoder {
		case "jpg":
			err = imaging.Save(img, outputName, imaging.JPEGQuality(opts.JpgQuality))
		case "png":
			err = imaging.Save(img, outputName, imaging.PNGCompressionLevel(png.CompressionLevel(opts.PngCompression)))
		case "gif":
			err = imaging.Save(img, outputName, imaging.GIFNumColors(opts.GifNumColors))
		case "webp":
			err = webp.Save(outputName, img, &webp.Options{Lossless: opts.WebpLossless, Quality: float32(opts.WebpQuality), Exact: opts.WebpRgbInTransparent})
		default:
			err = imaging.Save(img, outputName)
		}
		if err != nil {
			notify(i.Ctx, NotifyEvent{
				Info: fmt.Sprintf("编码%s失败: %s", file.Name, err.Error()),
				Type: "danger",
			})
			return
		}

		record.progress.Incr()
	})

	if first {
		i.Gow.Go(func() {
			record.done.Wait()
			i.Kv.Delete(record.progress.event)
			if len(record.errCh) == 0 {
				notify(i.Ctx, NotifyEvent{
					Info: fmt.Sprintf("%s处理全部完成", record.name),
					Type: "success",
				})
			} else if len(record.errCh) == filesNum {
				notify(i.Ctx, NotifyEvent{
					Info: fmt.Sprintf("%s处理全部失败", record.name),
					Type: "danger",
				})
			} else {
				notify(i.Ctx, NotifyEvent{
					Info: fmt.Sprintf("%s处理部分失败", record.name),
					Type: "warning",
				})
			}
			record.progress.close()
		})
	}

	return first, nil
}

func (i *Image) QrEncode(text, savePath string, recover, size int) (string, error) {
	saveName := fmt.Sprintf("%s_%s_%d_%d.png", fileName(strz.Sub(text, 0, 10)), randz.String(5), recover, size)
	savePathName := filepath.Join(savePath, saveName)

	_, err := os.Stat(savePathName)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return "", errx.Logf("file already exists: %s", savePathName)
	}

	b, err := qrcode.Encode(text, qrcode.RecoveryLevel(recover), size)
	if err != nil {
		return "", errx.Log(err)
	}

	return saveName, errx.Log(os.WriteFile(savePathName, b, 0644))
}

func (i *Image) QrDecode(b []byte) (string, error) {
	img, err := imaging.Decode(bytes.NewReader(b))
	if err != nil {
		return "", errx.Log(err)
	}

	qrCodes, err := goqr.Recognize(img)
	if err != nil {
		return "", errx.Log(err)
	}

	if len(qrCodes) == 1 {
		return string(qrCodes[0].Payload), nil
	}

	var buf strings.Builder
	for _, qrCode := range qrCodes {
		buf.Write(qrCode.Payload)
		buf.WriteString("\n")
	}
	return buf.String(), nil
}

func decode(r io.Reader) (image.Image, error) {
	return imaging.Decode(r, imaging.AutoOrientation(true))
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

func cropGif(g *gif.GIF, opts *ImageOptions) *gif.GIF {
	c := &gif.GIF{
		Image:           make([]*image.Paletted, 0, len(g.Image)),
		Delay:           make([]int, 0, len(g.Delay)),
		Disposal:        make([]byte, 0, len(g.Disposal)),
		Config:          g.Config,
		BackgroundIndex: g.BackgroundIndex,
	}

	var filters []gift.Filter
	switch opts.Op {
	case Original:
	case FixedWH:
		filters = append(filters, gift.ResizeToFill(opts.Width, opts.Height, gift.NearestNeighborResampling, gift.CenterAnchor))
	case FixedWidth:
		filters = append(filters, gift.Resize(opts.Width, 0, gift.NearestNeighborResampling))
	case FixedHeight:
		filters = append(filters, gift.Resize(0, opts.Height, gift.NearestNeighborResampling))
	case Percentage:
		w := math.Ceil(float64(g.Config.Width) * float64(opts.Percent) / 100)
		h := math.Ceil(float64(g.Config.Height) * float64(opts.Percent) / 100)
		filters = append(filters, gift.ResizeToFill(int(w), int(h), gift.NearestNeighborResampling, gift.CenterAnchor))
	case MaxWidth:
		if g.Config.Width > opts.Width {
			filters = append(filters, gift.Resize(opts.Width, 0, gift.NearestNeighborResampling))
		}
	case MaxHeight:
		if g.Config.Height > opts.Height {
			filters = append(filters, gift.Resize(0, opts.Height, gift.NearestNeighborResampling))
		}
	case MaxWH:
		if g.Config.Width > opts.Width || g.Config.Height > opts.Height {
			filters = append(filters, gift.ResizeToFit(opts.Width, opts.Height, gift.NearestNeighborResampling))
		}
	}

	filter := gift.New(filters...)
	delay := 0

	firstFrame := g.Image[0]
	c.Config.Width = filter.Bounds(firstFrame.Bounds()).Max.X
	c.Config.Height = filter.Bounds(firstFrame.Bounds()).Max.Y
	if opts.GifDrawOnBefore {
		tmp := image.NewNRGBA(firstFrame.Bounds())
		for i := range g.Image {
			// draw current frame over previous:
			gift.New().DrawAt(tmp, g.Image[i], g.Image[i].Bounds().Min, gift.OverOperator)
			dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), g.Image[i].Palette)
			filter.Draw(dst, tmp)
			delay = delay + g.Delay[i]
			if opts.GifDropRate == 0 || (i+1)%opts.GifDropRate != 0 {
				c.Image = append(c.Image, dst)
				c.Delay = append(c.Delay, delay)
				c.Disposal = append(c.Disposal, g.Disposal[i])
				delay = 0
			}
		}
	} else {
		for i := range g.Image {
			delay = delay + g.Delay[i]
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

func crop(img image.Image, opts *ImageOptions) image.Image {
	switch opts.Op {
	case Original:
		return img
	case FixedWH:
		return imaging.Fill(img, opts.Width, opts.Height, imaging.Center, imaging.Lanczos)
	case FixedWidth:
		return imaging.Resize(img, opts.Width, 0, imaging.Lanczos)
	case FixedHeight:
		return imaging.Resize(img, 0, opts.Height, imaging.Lanczos)
	case Percentage:
		w := math.Ceil(float64(img.Bounds().Dx()) * float64(opts.Percent) / 100)
		h := math.Ceil(float64(img.Bounds().Dy()) * float64(opts.Percent) / 100)
		return imaging.Fill(img, int(w), int(h), imaging.Center, imaging.Lanczos)
	case MaxWidth:
		if img.Bounds().Dx() <= opts.Width {
			return img
		}
		return imaging.Resize(img, opts.Width, 0, imaging.Lanczos)
	case MaxHeight:
		if img.Bounds().Dy() <= opts.Height {
			return img
		}
		return imaging.Resize(img, 0, opts.Height, imaging.Lanczos)
	case MaxWH:
		if img.Bounds().Dx() <= opts.Width && img.Bounds().Dy() <= opts.Height {
			return img
		}
		return imaging.Fit(img, opts.Width, opts.Height, imaging.Lanczos)
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

func outImageName(name string, opts ImageOptions) string {
	return filepath.Join(opts.OutPath, fmt.Sprintf("%s_%d_%s.%s", name, opts.Op, randz.String(4), opts.Encoder))
}
