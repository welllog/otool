package dedup

import (
	"image"
	"math"
	"sync"

	"github.com/disintegration/imaging"
)

var grayBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 256*256)
		return &b
	},
}

// ComputeLaplacianVariance calculates the edge sharpness of an image using a 3x3 Laplacian operator.
// A higher variance indicates a sharper, crisp image; a low variance indicates blur or motion shake.
// Optimized with integer luminance weights, direct pixel strides, and Welford zero-allocation variance.
func ComputeLaplacianVariance(img image.Image) float64 {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	if w < 3 || h < 3 {
		return 0
	}

	// Downsample to max 256x256 for ultra-fast convolution without losing edge statistics
	sample := img
	if w > 256 || h > 256 {
		sample = imaging.Fit(img, 256, 256, imaging.NearestNeighbor)
	}

	sb := sample.Bounds()
	sw := sb.Dx()
	sh := sb.Dy()
	if sw < 3 || sh < 3 {
		return 0
	}

	totalPixels := sw * sh
	bufPtr := grayBufPool.Get().(*[]byte)
	buf := *bufPtr
	if len(buf) < totalPixels {
		buf = make([]byte, totalPixels)
	}
	defer grayBufPool.Put(bufPtr)

	// Fast luminance extraction using integer fixed-point (0.299*R + 0.587*G + 0.114*B)
	switch s := sample.(type) {
	case *image.NRGBA:
		idx := 0
		for y := sb.Min.Y; y < sb.Max.Y; y++ {
			offset := (y - sb.Min.Y) * s.Stride
			for x := 0; x < sw; x++ {
				p := offset + x*4
				r := uint32(s.Pix[p])
				g := uint32(s.Pix[p+1])
				b := uint32(s.Pix[p+2])
				buf[idx] = byte((r*77 + g*150 + b*29) >> 8)
				idx++
			}
		}
	case *image.RGBA:
		idx := 0
		for y := sb.Min.Y; y < sb.Max.Y; y++ {
			offset := (y - sb.Min.Y) * s.Stride
			for x := 0; x < sw; x++ {
				p := offset + x*4
				r := uint32(s.Pix[p])
				g := uint32(s.Pix[p+1])
				b := uint32(s.Pix[p+2])
				buf[idx] = byte((r*77 + g*150 + b*29) >> 8)
				idx++
			}
		}
	case *image.Gray:
		idx := 0
		for y := sb.Min.Y; y < sb.Max.Y; y++ {
			offset := (y - sb.Min.Y) * s.Stride
			copy(buf[idx:idx+sw], s.Pix[offset:offset+sw])
			idx += sw
		}
	default:
		idx := 0
		for y := sb.Min.Y; y < sb.Max.Y; y++ {
			for x := sb.Min.X; x < sb.Max.X; x++ {
				r, g, b, _ := sample.At(x, y).RGBA()
				lum := (r>>8)*77 + (g>>8)*150 + (b>>8)*29
				buf[idx] = byte(lum >> 8)
				idx++
			}
		}
	}

	// Convolve with standard 3x3 discrete Laplacian operator:
	//  0  1  0
	//  1 -4  1
	//  0  1  0
	// Online Welford algorithm to compute variance in a single pass with zero heap allocation.
	var count int64
	var mean float64
	var m2 float64

	for y := 1; y < sh-1; y++ {
		row := y * sw
		top := (y - 1) * sw
		bot := (y + 1) * sw

		for x := 1; x < sw-1; x++ {
			val := float64(int(buf[top+x]) + int(buf[row+x-1]) + int(buf[row+x+1]) + int(buf[bot+x]) - 4*int(buf[row+x]))
			count++
			delta := val - mean
			mean += delta / float64(count)
			delta2 := val - mean
			m2 += delta * delta2
		}
	}

	if count <= 1 {
		return 0
	}

	variance := m2 / float64(count-1)
	if math.IsNaN(variance) || math.IsInf(variance, 0) {
		return 0
	}
	return variance
}
