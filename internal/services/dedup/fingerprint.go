package dedup

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"math/bits"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/SeriousBug/webp-go-pure/std"
	"github.com/cespare/xxhash/v2"
	"github.com/corona10/goimagehash"
	"github.com/disintegration/imaging"
	_ "github.com/gen2brain/avif"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
	"github.com/welllog/otool/internal/storage"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
)

type AnalysisResult struct {
	Width          int
	Height         int
	Format         string
	ContentHash    uint64
	PHash          uint64
	DHash          uint64
	BlurScore      float64
	ThumbnailBytes []byte
	HasEXIF        bool
	CameraModel    string
	DateTimeOrig   time.Time
}

// ImageProbe contains the lightweight header information needed to choose an
// appropriate decode-memory budget before a full image is decoded.
type ImageProbe struct {
	Width         int
	Height        int
	Format        string
	BytesPerPixel int
}

// ProbeImage reads only the image header. It seeks back to the beginning so the
// same reader can be used for EXIF parsing and decoding afterwards.
func ProbeImage(r io.ReadSeeker) (*ImageProbe, error) {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	cfg, format, err := image.DecodeConfig(r)
	if err != nil {
		return nil, fmt.Errorf("读取图像信息失败: %w", err)
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, fmt.Errorf("无效的图像尺寸: %dx%d", cfg.Width, cfg.Height)
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return &ImageProbe{
		Width:         cfg.Width,
		Height:        cfg.Height,
		Format:        strings.ToUpper(format),
		BytesPerPixel: bytesPerPixelForModel(cfg.ColorModel),
	}, nil
}

// bytesPerPixelForModel returns a conservative decoded-image memory cost for
// the color model reported by image.DecodeConfig. PNG/TIFF 16-bit images use
// 8 bytes per pixel even though the pixel count is the same as an 8-bit image.
func bytesPerPixelForModel(model color.Model) int {
	switch model {
	case color.GrayModel:
		return 2 // decoders commonly expand gray to an 8-bit work image
	case color.Gray16Model, color.Alpha16Model:
		return 8
	case color.NRGBAModel, color.RGBAModel, color.CMYKModel:
		return 4
	case color.NRGBA64Model, color.RGBA64Model:
		return 8
	default:
		return 4
	}
}

// EstimateDecodeMemory returns a conservative estimate of the memory needed by
// a full-resolution RGBA image decode. The calculation is overflow-safe and
// callers must reject MaxInt64-sized results.
func EstimateDecodeMemory(width, height int) int64 {
	return estimateDecodeMemory(width, height, 4)
}

func estimateDecodeMemory(width, height, bytesPerPixel int) int64 {
	if width <= 0 || height <= 0 {
		return 64 << 20
	}
	if bytesPerPixel <= 0 {
		bytesPerPixel = 4
	}
	// Reject absurd headers before multiplying: a 200MP image already needs
	// far more memory than any sane decode budget.
	const maxPixels = int64(200_000_000)
	if int64(width) > maxPixels/int64(height) {
		return math.MaxInt64
	}
	pixels := int64(width) * int64(height)
	if pixels > math.MaxInt64/int64(bytesPerPixel) {
		return math.MaxInt64
	}
	return pixels*int64(bytesPerPixel) + 8<<20
}

// HammingDistance calculates the Hamming distance between two 64-bit perceptual hashes.
func HammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// ComputePartialHash quickly hashes up to 3 blocks of a file (head, middle, tail) to filter non-duplicates.
// Seekable readers use seeks; non-seekable readers use a streaming skip path
// with exactly the same block layout so the two paths are interchangeable.
func ComputePartialHash(r io.Reader, size int64) (uint64, error) {
	const blockSize = int64(4096) // 4KB

	if size > blockSize*4 {
		h := xxhash.New()
		buf := make([]byte, blockSize)

		if seeker, ok := r.(io.ReadSeeker); ok {
			// 1. Head
			if _, err := io.ReadFull(seeker, buf); err != nil {
				return 0, err
			}
			_, _ = h.Write(buf)

			// 2. Middle
			midOffset := size / 2
			if _, err := seeker.Seek(midOffset, io.SeekStart); err != nil {
				return 0, err
			}
			if _, err := io.ReadFull(seeker, buf); err != nil {
				return 0, err
			}
			_, _ = h.Write(buf)

			// 3. Tail
			tailOffset := size - blockSize
			if _, err := seeker.Seek(tailOffset, io.SeekStart); err != nil {
				return 0, err
			}
			if _, err := io.ReadFull(seeker, buf); err != nil {
				return 0, err
			}
			_, _ = h.Write(buf)

			return h.Sum64(), nil
		}

		// Non-seekable stream: read the same head/middle/tail blocks while
		// discarding the intervening bytes instead of hashing only the first
		// 16KB. This keeps Range and non-Range WebDAV samples identical.
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		_, _ = h.Write(buf)

		midOffset := size / 2
		if _, err := io.CopyN(io.Discard, r, midOffset-blockSize); err != nil {
			return 0, err
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		_, _ = h.Write(buf)

		tailOffset := size - blockSize
		if _, err := io.CopyN(io.Discard, r, tailOffset-(midOffset+blockSize)); err != nil {
			return 0, err
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			return 0, err
		}
		_, _ = h.Write(buf)

		return h.Sum64(), nil
	}

	// Fallback for small files: read up to 16KB
	limit := int64(16384)
	if size > 0 && size < limit {
		limit = size
	}
	buf := make([]byte, limit)
	n, err := io.ReadFull(r, buf)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return 0, err
	}
	return xxhash.Sum64(buf[:n]), nil
}

// ComputePartialHashFromStore hashes head/middle/tail blocks using range
// requests. This is important for WebDAV, where opening a non-seekable stream
// used to read only the first 16KB for every candidate.
func ComputePartialHashFromStore(ctx context.Context, store storage.Storage, path string, size int64) (uint64, error) {
	const blockSize = int64(4096)

	if size <= blockSize*4 {
		rc, err := store.Open(ctx, path)
		if err != nil {
			return 0, err
		}
		defer rc.Close()
		return ComputePartialHash(rc, size)
	}

	// Local files are seekable: one open plus three seeks is cheaper than
	// three separate range opens.
	if store.Driver() == storage.DriverLocal {
		rc, err := store.Open(ctx, path)
		if err != nil {
			return 0, err
		}
		defer rc.Close()
		return ComputePartialHash(rc, size)
	}

	offsets := []int64{0, size / 2, size - blockSize}
	h := xxhash.New()
	buf := make([]byte, blockSize)
	for _, offset := range offsets {
		rc, err := store.OpenRange(ctx, path, offset, blockSize)
		if err != nil {
			// Some servers/proxies reject Range requests. Falling back to a
			// normal stream keeps correctness; the full hash stage will still
			// eliminate false candidates.
			return computePartialHashViaOpen(ctx, store, path, size)
		}
		n, readErr := io.ReadFull(rc, buf)
		closeErr := rc.Close()
		if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
			return 0, readErr
		}
		if closeErr != nil {
			return 0, closeErr
		}
		if n == 0 {
			return 0, fmt.Errorf("空的分段读取结果")
		}
		_, _ = h.Write(buf[:n])
	}
	return h.Sum64(), nil
}

func computePartialHashViaOpen(ctx context.Context, store storage.Storage, path string, size int64) (uint64, error) {
	rc, err := store.Open(ctx, path)
	if err != nil {
		return 0, err
	}
	defer rc.Close()
	return ComputePartialHash(rc, size)
}

// ComputeFullContentHash streams the entire file into xxhash without loading all into memory.
func ComputeFullContentHash(r io.Reader) (uint64, error) {
	return ComputeFullContentHashContext(context.Background(), r)
}

// ComputeFullContentHashContext is the context-aware variant used by scans.
// The reader is checked before every chunk so a Stop request can interrupt a
// long local file hash without waiting for the whole file.
func ComputeFullContentHashContext(ctx context.Context, r io.Reader) (uint64, error) {
	h := xxhash.New()
	buf := make([]byte, 64*1024) // 64KB chunk stream
	if _, err := io.CopyBuffer(h, &contextReader{ctx: ctx, r: r}, buf); err != nil {
		return 0, err
	}
	return h.Sum64(), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}

type contextReadSeeker struct {
	ctx context.Context
	rs  io.ReadSeeker
}

func (r *contextReadSeeker) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.rs.Read(p)
}

func (r *contextReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.rs.Seek(offset, whence)
}

var (
	thumbnailGateOnce sync.Once
	thumbnailGate     *decodeMemoryGate
)

func thumbnailDecodeGate() *decodeMemoryGate {
	thumbnailGateOnce.Do(func() {
		thumbnailGate = newDecodeMemoryGate(512 << 20)
	})
	return thumbnailGate
}

// GenerateThumbnailFromStore decodes one image from storage and returns a
// 200px JPEG thumbnail. It is used by the UI when a fingerprint survived cache
// pruning but its thumbnail was evicted.
func GenerateThumbnailFromStore(ctx context.Context, store storage.Storage, path, filename string) ([]byte, error) {
	rc, err := store.Open(ctx, path)
	if err != nil {
		return nil, err
	}

	var reader io.ReadSeeker
	var closers []io.Closer
	if rs, ok := rc.(io.ReadSeeker); ok {
		reader = rs
		closers = append(closers, rc)
		defer rc.Close()
	} else {
		spool, err := os.CreateTemp("", "otool-thumb-*")
		if err != nil {
			_ = rc.Close()
			return nil, err
		}
		spoolName := spool.Name()
		defer func() {
			_ = spool.Close()
			_ = os.Remove(spoolName)
		}()
		if _, err := io.Copy(spool, rc); err != nil {
			_ = rc.Close()
			return nil, err
		}
		_ = rc.Close()
		if _, err := spool.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
		reader = spool
		closers = append(closers, spool)
	}
	// Closing the underlying file unblocks a long image.Decode when the user
	// cancels the scan or resets the thumbnail generation.
	stopClose := context.AfterFunc(ctx, func() {
		for _, c := range closers {
			_ = c.Close()
		}
	})
	defer stopClose()

	probe, err := ProbeImage(&contextReadSeeker{ctx: ctx, rs: reader})
	if err != nil {
		return nil, err
	}
	estimated := estimateDecodeMemory(probe.Width, probe.Height, probe.BytesPerPixel)
	if estimated > 512<<20 {
		return nil, fmt.Errorf("图片尺寸过大，跳过缩略图生成: %dx%d", probe.Width, probe.Height)
	}
	reserved, ok := thumbnailDecodeGate().acquire(ctx, estimated)
	if !ok {
		return nil, ctx.Err()
	}
	defer thumbnailDecodeGate().release(reserved)

	res, err := analyzeImage(ctx, reader, filename, true, false)
	if err != nil {
		return nil, err
	}
	if len(res.ThumbnailBytes) == 0 {
		return nil, fmt.Errorf("生成缩略图失败: 空结果")
	}
	return res.ThumbnailBytes, nil
}

// AnalyzeImage analyzes an in-memory image. It is kept for tests and small
// callers; scanning code uses AnalyzeImageReader to avoid retaining file bytes.
func AnalyzeImage(data []byte, filename string) (*AnalysisResult, error) {
	res, err := analyzeImage(context.Background(), bytes.NewReader(data), filename, true, true)
	if err != nil {
		return nil, err
	}
	res.ContentHash = xxhash.Sum64(data)
	return res, nil
}

// AnalyzeImageReader streams EXIF parsing and image decoding from a seekable
// reader. The reader is not retained after the call, so the decoded image can
// be collected as soon as the small work image and thumbnail are built.
func AnalyzeImageReader(r io.ReadSeeker, filename string) (*AnalysisResult, error) {
	return AnalyzeImageReaderContext(context.Background(), r, filename)
}

// AnalyzeImageReaderContext is the context-aware variant used by scans.
func AnalyzeImageReaderContext(ctx context.Context, r io.ReadSeeker, filename string) (*AnalysisResult, error) {
	return analyzeImage(ctx, r, filename, true, true)
}

// AnalyzeImageReaderContextNoThumb skips the JPEG thumbnail step. Similar-mode
// scans use it for the full library: only photos that end up in duplicate
// groups need previews, and those are generated lazily by the UI.
func AnalyzeImageReaderContextNoThumb(ctx context.Context, r io.ReadSeeker, filename string) (*AnalysisResult, error) {
	return analyzeImage(ctx, r, filename, false, true)
}

func analyzeImage(ctx context.Context, r io.ReadSeeker, filename string, generateThumb, computeFingerprint bool) (*AnalysisResult, error) {
	cr := &contextReadSeeker{ctx: ctx, rs: r}
	probe, err := ProbeImage(cr)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 1. Parse EXIF (best effort; PNG/WebP and stripped photos have none).
	var ex *exif.Exif
	if _, err := cr.Seek(0, io.SeekStart); err == nil {
		ex, _ = exif.Decode(cr)
	}
	orientation := exifOrientation(ex)

	var hasEXIF bool
	var cameraModel string
	var dtOrig time.Time
	if ex != nil {
		hasEXIF = true
		if dt, err := ex.DateTime(); err == nil {
			dtOrig = dt
		}
		camMake, _ := ex.Get(exif.Make)
		camMod, _ := ex.Get(exif.Model)
		makeStr := cleanExifString(camMake)
		modStr := cleanExifString(camMod)
		if makeStr != "" && modStr != "" {
			cameraModel = fmt.Sprintf("%s %s", makeStr, modStr)
		} else if modStr != "" {
			cameraModel = modStr
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// 2. Decode only once. Image decoders must expand the full image, but we
	// immediately shrink it to a 256px work image and release the original.
	if _, err := cr.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(cr)
	if err != nil {
		return nil, fmt.Errorf("解码图像失败: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	bounds := img.Bounds()
	origW := bounds.Dx()
	origH := bounds.Dy()
	if orientation >= 5 && orientation <= 8 {
		// EXIF orientations 5-8 rotate by 90 degrees; reported dimensions
		// should reflect the displayed orientation.
		origW, origH = origH, origW
	}

	// 3. Downsample to lightweight work image (max 256x256) before applying
	// rotation. Rotating the small image is much cheaper than rotating a
	// full-resolution 50MP photo.
	workImg := img
	if origW > 256 || origH > 256 {
		workImg = imaging.Fit(img, 256, 256, imaging.Box)
	}
	img = nil

	if orientation >= 2 && orientation <= 8 {
		workImg = applyOrientation(workImg, orientation)
	}

	// 4/5. Compute perceptual hashes and sharpness only when the caller needs
	// them. Lazy thumbnail generation can skip this CPU work entirely.
	var pHash, dHash uint64
	var blurScore float64
	if computeFingerprint {
		pHashObj, err := goimagehash.PerceptionHash(workImg)
		if err != nil {
			return nil, fmt.Errorf("计算 pHash 失败: %w", err)
		}
		dHashObj, err := goimagehash.DifferenceHash(workImg)
		if err != nil {
			return nil, fmt.Errorf("计算 dHash 失败: %w", err)
		}
		pHash = pHashObj.GetHash()
		dHash = dHashObj.GetHash()
		blurScore = ComputeLaplacianVariance(workImg)
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}

	var thumbBytes []byte
	if generateThumb {
		// 6. Generate 200px fast JPEG thumbnail
		var thumb image.Image = workImg
		wb := workImg.Bounds()
		if wb.Dx() > 200 || wb.Dy() > 200 {
			thumb = imaging.Fit(workImg, 200, 200, imaging.Linear)
		}
		var thumbBuf bytes.Buffer
		if err := jpeg.Encode(&thumbBuf, thumb, &jpeg.Options{Quality: 80}); err != nil {
			return nil, fmt.Errorf("生成缩略图失败: %w", err)
		}
		thumbBytes = thumbBuf.Bytes()
	}

	return &AnalysisResult{
		Width:          origW,
		Height:         origH,
		Format:         probe.Format,
		PHash:          pHash,
		DHash:          dHash,
		BlurScore:      blurScore,
		ThumbnailBytes: thumbBytes,
		HasEXIF:        hasEXIF,
		CameraModel:    cameraModel,
		DateTimeOrig:   dtOrig,
	}, nil
}

func exifOrientation(ex *exif.Exif) int {
	if ex == nil {
		return 1
	}
	tag, err := ex.Get(exif.Orientation)
	if err != nil {
		return 1
	}
	v, err := tag.Int(0)
	if err != nil || v < 1 || v > 8 {
		return 1
	}
	return v
}

func applyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.Transpose(img)
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.Transverse(img)
	case 8:
		return imaging.Rotate90(img)
	default:
		return img
	}
}

func cleanExifString(tag *tiff.Tag) string {
	if tag == nil {
		return ""
	}
	s, err := tag.StringVal()
	if err != nil {
		return ""
	}
	return strings.Trim(s, "\" \r\n\t")
}
