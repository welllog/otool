package dedup

import (
	"image"
	"image/color"
	"testing"
)

// genericImage hides the concrete image type so ComputeLaplacianVariance
// takes the definitionally-correct default branch (absolute At(x, y)
// addressing) instead of the fast paths.
type genericImage struct{ img image.Image }

func (g genericImage) ColorModel() color.Model { return g.img.ColorModel() }
func (g genericImage) Bounds() image.Rectangle { return g.img.Bounds() }
func (g genericImage) At(x, y int) color.Color { return g.img.At(x, y) }

// pattern returns a position-dependent color so any addressing offset error
// changes the sampled pixels and therefore the computed variance.
func pattern(x, y int) (r, g, b uint8) {
	v := uint8((x*7 + y*13 + (x*y)%251) & 0xff)
	return v, uint8(255 - int(v)), uint8((x ^ y) & 0xff)
}

func fillNRGBA(m *image.NRGBA, ox, oy int) {
	for y := 0; y < m.Rect.Dy(); y++ {
		off := y * m.Stride
		for x := 0; x < m.Rect.Dx(); x++ {
			p := off + x*4
			r, g, b := pattern(ox+x, oy+y)
			m.Pix[p], m.Pix[p+1], m.Pix[p+2], m.Pix[p+3] = r, g, b, 255
		}
	}
}

func fillRGBA(m *image.RGBA, ox, oy int) {
	for y := 0; y < m.Rect.Dy(); y++ {
		off := y * m.Stride
		for x := 0; x < m.Rect.Dx(); x++ {
			p := off + x*4
			r, g, b := pattern(ox+x, oy+y)
			m.Pix[p], m.Pix[p+1], m.Pix[p+2], m.Pix[p+3] = r, g, b, 255
		}
	}
}

func fillGray(m *image.Gray, ox, oy int) {
	for y := 0; y < m.Rect.Dy(); y++ {
		off := y * m.Stride
		for x := 0; x < m.Rect.Dx(); x++ {
			r, g, b := pattern(ox+x, oy+y)
			m.Pix[off+x] = uint8((uint32(r)*77 + uint32(g)*150 + uint32(b)*29) >> 8)
		}
	}
}

// nrgbaCopyFrom materializes the same pixels a SubImage view covers into a
// fresh zero-origin image, using only the stdlib At/Set accessors.
func nrgbaCopyFrom(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x-b.Min.X, y-b.Min.Y, src.At(x, y))
		}
	}
	return dst
}

// TestComputeLaplacianVarianceSubImageNonZeroOrigin locks in the fast-path
// pixel addressing for images whose Bounds().Min is non-zero (SubImage
// views). The fast paths iterate columns relatively (x from 0 to w) and
// compute row starts as (y-bounds.Min.Y)*Stride, which is exactly the stdlib
// PixOffset(bounds.Min.X, y). A refactor that switches to absolute-x
// addressing would read the wrong pixels or go out of bounds, and this test
// would fail or panic.
func TestComputeLaplacianVarianceSubImageNonZeroOrigin(t *testing.T) {
	const parentDim = 300

	buildNRGBA := func() *image.NRGBA {
		m := image.NewNRGBA(image.Rect(0, 0, parentDim, parentDim))
		fillNRGBA(m, 0, 0)
		return m
	}
	buildRGBA := func() *image.RGBA {
		m := image.NewRGBA(image.Rect(0, 0, parentDim, parentDim))
		fillRGBA(m, 0, 0)
		return m
	}
	buildGray := func() *image.Gray {
		m := image.NewGray(image.Rect(0, 0, parentDim, parentDim))
		fillGray(m, 0, 0)
		return m
	}

	// Sub-rects stay <= 256 so ComputeLaplacianVariance does not downsample
	// and the fast paths are exercised directly. The corners and the
	// single-axis-offset rects probe the maximum row/column addresses.
	rects := []image.Rectangle{
		image.Rect(13, 17, 113, 117),   // both axes offset
		image.Rect(13, 0, 113, 100),    // only X offset
		image.Rect(0, 17, 100, 117),    // only Y offset
		image.Rect(280, 280, 300, 300), // bottom-right corner
	}

	for _, r := range rects {
		w, h := r.Dx(), r.Dy()
		if w < 3 || h < 3 {
			t.Fatalf("sub rect %v too small", r)
		}

		t.Run(("NRGBA " + r.String()), func(t *testing.T) {
			sub := buildNRGBA().SubImage(r).(*image.NRGBA)
			if sub.Bounds().Min != r.Min {
				t.Fatalf("unexpected sub bounds %v", sub.Bounds())
			}
			want := ComputeLaplacianVariance(nrgbaCopyFrom(sub))
			if want <= 0 {
				t.Fatal("expected non-degenerate reference variance")
			}
			got := ComputeLaplacianVariance(sub)
			if got != want {
				t.Fatalf("non-zero origin changed NRGBA variance: got %v want %v", got, want)
			}
		})

		t.Run("RGBA "+r.String(), func(t *testing.T) {
			sub := buildRGBA().SubImage(r).(*image.RGBA)
			// RGBA At() returns premultiplied values and the fast path reads
			// the same premultiplied bytes, so the generic branch (absolute
			// At addressing) must agree exactly on the very same view.
			generic := ComputeLaplacianVariance(genericImage{sub})
			if generic <= 0 {
				t.Fatal("expected non-degenerate reference variance")
			}
			fast := ComputeLaplacianVariance(sub)
			if fast != generic {
				t.Fatalf("non-zero origin changed RGBA variance: got %v want %v", fast, generic)
			}
		})

		t.Run("Gray "+r.String(), func(t *testing.T) {
			sub := buildGray().SubImage(r).(*image.Gray)
			generic := ComputeLaplacianVariance(genericImage{sub})
			if generic <= 0 {
				t.Fatal("expected non-degenerate reference variance")
			}
			fast := ComputeLaplacianVariance(sub)
			if fast != generic {
				t.Fatalf("non-zero origin changed Gray variance: got %v want %v", fast, generic)
			}
		})
	}
}
