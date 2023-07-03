package srvs

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"testing"

	"github.com/disintegration/gift"
	"github.com/disintegration/imaging"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
)

func TestDecodeImage(t *testing.T) {
	//f, err := os.Open("/Users/chentairen/Desktop/54604325_p0.webp")
	//f, err := os.Open("/Users/chentairen/Desktop/Untitled.png")
	f, err := os.Open("/Users/chentairen/Desktop/btree.gif")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	img, name, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	//gif.DecodeAll(f)

	dst, err := os.Create("/Users/chentairen/Desktop/tt.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	fmt.Println(name)
	jpeg.Encode(dst, img, &jpeg.Options{Quality: 100})
}

func TestImagine(t *testing.T) {
	_, err := imaging.Open("/Users/chentairen/Desktop/54604325_p0.webp")
	if err != nil {
		t.Fatal(err)
	}
	//imaging.Resize()
	//imaging.Fill()
	//imaging.Fit()
	//imaging.FlipV()
	//imaging.Crop()
	//imaging.Thumbnail()
}

func TestGift(t *testing.T) {
	f, err := os.Open("/Users/chentairen/Desktop/54604325_p0.webp")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}

	g := gift.New(gift.Resize(400, 400, gift.LanczosResampling))
	dst := image.NewNRGBA(g.Bounds(src.Bounds()))
	g.Draw(dst, src)
}
