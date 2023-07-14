package srvs

import (
	"fmt"
	"image"
	"image/gif"
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
	// f, err := os.Open("/Users/chentairen/Desktop/54604325_p0.webp")
	// f, err := os.Open("/Users/chentairen/Desktop/Untitled.png")
	f, err := os.Open("/Users/chentairen/Desktop/btree.gif")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	img, name, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	// gif.DecodeAll(f)

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
	f, err := imaging.FormatFromFilename("/Users/chentairen/Desktop/54604325_p0.webp")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(f.String())
	// imaging.Resize()
	// imaging.Fill()
	// imaging.Fit()
	// imaging.FlipV()
	// imaging.Crop()
	// imaging.Thumbnail()
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

func TestGifCrop(t *testing.T) {
	f, err := os.Open("/Users/chentairen/Downloads/资源/pic/f351d594-d7dd-11e4-89a2-c568840683fd.gif")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	gifImg, err := gif.DecodeAll(f)
	filter := gift.New(
		gift.Resize(400, 400, gift.NearestNeighborResampling),
	)
	// for i, img := range gifImg.Image {
	// 	tmp := image.NewNRGBA(img.Bounds())
	// 	gift.New().DrawAt(tmp, img, img.Bounds().Min, gift.CopyOperator)
	// 	dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), img.Palette)
	// 	filter.Draw(dst, tmp)
	// 	gifImg.Image[i] = dst
	// }

	tmp := image.NewNRGBA(gifImg.Image[0].Bounds())
	for i := range gifImg.Image {
		// draw current frame over previous:
		gift.New().DrawAt(tmp, gifImg.Image[i], gifImg.Image[i].Bounds().Min, gift.OverOperator)
		dst := image.NewPaletted(filter.Bounds(tmp.Bounds()), gifImg.Image[i].Palette)
		filter.Draw(dst, tmp)
		gifImg.Image[i] = dst
	}
	gifImg.Config.Width = 400
	gifImg.Config.Height = 400

	of, err := os.Create("out2.gif")
	if err != nil {
		t.Fatal(err)
	}
	defer of.Close()

	if err := gif.EncodeAll(of, gifImg); err != nil {
		t.Fatal(err)
	}
}
