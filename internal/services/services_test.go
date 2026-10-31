package services_test

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/welllog/otool/internal/services"
)

func TestCryptoService(t *testing.T) {
	cs := services.NewCryptoService()

	// AES-256 test
	plain := "Hello Otool with MyGo Native UI"
	secret := "MyStrongSecretKey123!"

	enc, err := cs.OpenSSLAesEnc(plain, secret)
	if err != nil {
		t.Fatalf("OpenSSLAesEnc failed: %v", err)
	}
	dec, err := cs.OpenSSLAesDec(enc, secret)
	if err != nil {
		t.Fatalf("OpenSSLAesDec failed: %v", err)
	}
	if dec != plain {
		t.Fatalf("AES decrypted mismatch: got %q, want %q", dec, plain)
	}

	// Hashes
	if cs.Md5("123456") != "e10adc3949ba59abbe56e057f20f883e" {
		t.Fatalf("MD5 mismatch")
	}
	if cs.Sha1("123456") != "7c4a8d09ca3762af61e59520943dc26494f8941b" {
		t.Fatalf("Sha1 mismatch")
	}
	if cs.Sha256("123456") != "8d969eef6ecad3c29a3a629280e686cf0c3f5d5a86aff3ca12020c923adc6c92" {
		t.Fatalf("Sha256 mismatch")
	}

	// HMAC
	hmacRes, err := cs.Hmac("123456", "secret", "md5")
	if err != nil || hmacRes == "" {
		t.Fatalf("HMAC-MD5 failed: %v", err)
	}

	// Base64
	b64 := cs.Base64Enc("hello")
	if cs.Base64Dec(b64) != "hello" {
		t.Fatalf("Base64 mismatch")
	}

	// URL
	urlEnc := cs.UrlEnc("hello world & ?")
	urlDec, _ := cs.UrlDec(urlEnc)
	if urlDec != "hello world & ?" {
		t.Fatalf("URL enc/dec mismatch")
	}

	// Hex
	hexEnc := cs.HexEnc("hello")
	hexDec := cs.HexDec(hexEnc)
	if hexDec != "hello" {
		t.Fatalf("Hex mismatch")
	}

	// Upper / Lower
	if cs.Upper("abc") != "ABC" || cs.Lower("ABC") != "abc" {
		t.Fatalf("Upper/Lower mismatch")
	}
}

func TestFileCrypto(t *testing.T) {
	cs := services.NewCryptoService()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test.txt")
	content := "This is a test file for streaming encryption."
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	var progressPct int64
	secret := "FileSecret999!"

	err := cs.EncryptFile(filePath, secret, func(pct int64) {
		progressPct = pct
	})
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}
	if progressPct != 100 {
		t.Fatalf("Encrypt progress should be 100, got %d", progressPct)
	}

	// Verify file is encrypted
	raw, _ := os.ReadFile(filePath)
	if string(raw) == content {
		t.Fatalf("File content should be encrypted")
	}

	// Decrypt
	err = cs.DecryptFile(filePath, secret, func(pct int64) {
		progressPct = pct
	})
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	rawDec, _ := os.ReadFile(filePath)
	if string(rawDec) != content {
		t.Fatalf("Decrypted content mismatch: got %q, want %q", string(rawDec), content)
	}

	// File hash
	md5Str, err := cs.Md5File(filePath)
	if err != nil || md5Str == "" {
		t.Fatalf("Md5File failed: %v", err)
	}
}

func TestConvertBase(t *testing.T) {
	res, err := services.ConvertBase("255", 10, 16)
	if err != nil || res != "FF" {
		t.Fatalf("ConvertBase 255 -> 16 failed: got %q, %v", res, err)
	}

	res, err = services.ConvertBase("FF", 16, 2)
	if err != nil || res != "11111111" {
		t.Fatalf("ConvertBase FF -> 2 failed: got %q, %v", res, err)
	}

	res, err = services.ConvertBase("1010", 2, 10)
	if err != nil || res != "10" {
		t.Fatalf("ConvertBase 1010 -> 10 failed: got %q, %v", res, err)
	}

	// Invalid input
	_, err = services.ConvertBase("Z", 10, 16)
	if err == nil {
		t.Fatalf("Expected error on invalid base 10 input 'Z'")
	}
}

func TestQR(t *testing.T) {
	text := "https://github.com/egoist/mygo"
	b, err := services.QrEncodeInMemory(text, 1, 256)
	if err != nil || len(b) == 0 {
		t.Fatalf("QrEncodeInMemory failed: %v", err)
	}

	decoded, err := services.QrDecodeBytes(b)
	if err != nil {
		t.Fatalf("QrDecodeBytes failed: %v", err)
	}
	if !strings.Contains(decoded, text) {
		t.Fatalf("Decoded QR mismatch: got %q, want %q", decoded, text)
	}

	tmpDir := t.TempDir()

	// 1. Create a dummy Logo image
	logoImg := image.NewRGBA(image.Rect(0, 0, 48, 48))
	for y := 0; y < 48; y++ {
		for x := 0; x < 48; x++ {
			logoImg.Set(x, y, color.RGBA{R: 240, G: 60, B: 60, A: 255})
		}
	}
	logoPath := filepath.Join(tmpDir, "logo.png")
	lf, err := os.Create(logoPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(lf, logoImg); err != nil {
		lf.Close()
		t.Fatal(err)
	}
	lf.Close()

	// 2. Test QrGenerate across all 3 visual styles and with Logo
	styles := []struct {
		name  string
		style services.QrShapeStyle
	}{
		{name: "Square", style: services.QrShapeSquare},
		{name: "Dots", style: services.QrShapeDots},
		{name: "Rounded", style: services.QrShapeRounded},
	}

	for _, s := range styles {
		t.Run("Style_"+s.name, func(t *testing.T) {
			res, err := services.QrGenerate(services.QrGenerateOptions{
				Text:    text,
				Style:   s.style,
				Recover: 1, // M
				Size:    256,
				Format:  services.QrFormatPNG,
				SaveDir: tmpDir,
			})
			if err != nil {
				t.Fatalf("QrGenerate failed for style %s: %v", s.name, err)
			}
			if len(res.Bytes) == 0 || len(res.PreviewPNG) == 0 {
				t.Fatalf("Empty bytes returned for style %s", s.name)
			}

			// Validate with industrial gozxing decoder
			dec, err := services.QrDecodeBytes(res.Bytes)
			if err != nil {
				t.Fatalf("Failed to decode QR code for style %s: %v", s.name, err)
			}
			if dec != text {
				t.Fatalf("Decoded text mismatch: got %q, want %q", dec, text)
			}
		})
	}

	// 3. Test with Logo (automatically locks to recovery H)
	t.Run("WithLogo", func(t *testing.T) {
		res, err := services.QrGenerate(services.QrGenerateOptions{
			Text:     text,
			Style:    services.QrShapeRounded,
			Recover:  0, // Passed L, should be overridden to H
			Size:     256,
			Format:   services.QrFormatPNG,
			LogoPath: logoPath,
			SaveDir:  tmpDir,
		})
		if err != nil {
			t.Fatalf("QrGenerate with logo failed: %v", err)
		}

		dec, err := services.QrDecodeFile(res.SavedPath)
		if err != nil {
			t.Fatalf("QrDecodeFile with logo failed: %v", err)
		}
		if dec != text {
			t.Fatalf("Decoded text mismatch with logo: got %q, want %q", dec, text)
		}
	})

	// 4. Test Multiple Export Formats (JPEG and SVG)
	t.Run("Format_JPEG", func(t *testing.T) {
		res, err := services.QrGenerate(services.QrGenerateOptions{
			Text:    text,
			Style:   services.QrShapeDots,
			Recover: 1,
			Size:    256,
			Format:  services.QrFormatJPEG,
			SaveDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("QrGenerate JPEG failed: %v", err)
		}
		if !strings.HasSuffix(res.SaveName, ".jpg") {
			t.Fatalf("Expected .jpg suffix, got %s", res.SaveName)
		}
		dec, err := services.QrDecodeBytes(res.Bytes)
		if err != nil {
			t.Fatalf("Failed to decode JPEG QR: %v", err)
		}
		if dec != text {
			t.Fatalf("Decoded text mismatch: got %q, want %q", dec, text)
		}
	})

	t.Run("Format_SVG", func(t *testing.T) {
		res, err := services.QrGenerate(services.QrGenerateOptions{
			Text:     text,
			Style:    services.QrShapeDots,
			Recover:  1,
			Size:     256,
			Format:   services.QrFormatSVG,
			FgColor:  "#1e40af",
			LogoPath: logoPath,
			SaveDir:  tmpDir,
		})
		if err != nil {
			t.Fatalf("QrGenerate SVG failed: %v", err)
		}
		if !strings.HasSuffix(res.SaveName, ".svg") {
			t.Fatalf("Expected .svg suffix, got %s", res.SaveName)
		}
		svgStr := string(res.Bytes)
		if !strings.HasPrefix(svgStr, "<svg") || !strings.Contains(svgStr, "</svg>") {
			t.Fatalf("Invalid SVG content output: %s", svgStr[:min(100, len(svgStr))])
		}
		if !strings.Contains(svgStr, "fill=\"#1e40af\"") {
			t.Fatalf("Expected SVG to contain fill=\"#1e40af\"")
		}
		if len(res.PreviewPNG) == 0 {
			t.Fatal("Expected PreviewPNG to be populated even for SVG format")
		}
	})

	// 5. Test Foreground Color (Blue) and Decodability
	t.Run("Color_Theme", func(t *testing.T) {
		res, err := services.QrGenerate(services.QrGenerateOptions{
			Text:    text,
			Style:   services.QrShapeRounded,
			Recover: 1,
			Size:    256,
			Format:  services.QrFormatPNG,
			FgColor: "#1e40af",
			SaveDir: tmpDir,
		})
		if err != nil {
			t.Fatalf("QrGenerate with theme color failed: %v", err)
		}
		dec, err := services.QrDecodeBytes(res.Bytes)
		if err != nil {
			t.Fatalf("QrDecodeBytes failed on colored QR: %v", err)
		}
		if dec != text {
			t.Fatalf("Decoded text mismatch: got %q, want %q", dec, text)
		}
	})
}

func TestImageOperations(t *testing.T) {
	tmpDir := t.TempDir()

	// 1. Generate a test PNG image (100x80)
	srcImg := image.NewRGBA(image.Rect(0, 0, 100, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			srcImg.Set(x, y, color.RGBA{R: uint8(x * 2), G: uint8(y * 3), B: 128, A: 255})
		}
	}
	srcPath := filepath.Join(tmpDir, "sample.png")
	f, err := os.Create(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, srcImg); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	// 2. Test DecodeImageInfo
	info, err := services.DecodeImageInfo(srcPath)
	if err != nil {
		t.Fatalf("DecodeImageInfo failed: %v", err)
	}
	if info.Width != 100 || info.Height != 80 {
		t.Fatalf("DecodeImageInfo dimension mismatch: got %dx%d, want 100x80", info.Width, info.Height)
	}
	if info.Thumbnail == nil {
		t.Fatal("Thumbnail should not be nil")
	}

	// 3. Test TransformImage across different formats
	outDir := filepath.Join(tmpDir, "out")
	if err := os.MkdirAll(outDir, 0755); err != nil {
		t.Fatal(err)
	}

	formats := []string{"jpg", "png", "webp", "avif"}
	for _, fmtName := range formats {
		opts := services.DefaultImageOptions()
		opts.OutPath = outDir
		opts.Encoder = fmtName
		opts.Op = services.OpFixedWidth
		opts.Width = 50

		outPath, err := services.TransformImage(srcPath, opts)
		if err != nil {
			t.Fatalf("TransformImage to %s failed: %v", fmtName, err)
		}
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("Output file does not exist: %v", err)
		}
	}

	// 4. Test Crop Operations: FixedWH, Percentage, CustomCrop
	opts := services.DefaultImageOptions()
	opts.OutPath = outDir
	opts.Encoder = "png"
	opts.Op = services.OpFixedWH
	opts.Width = 40
	opts.Height = 40
	outPath, err := services.TransformImage(srcPath, opts)
	if err != nil {
		t.Fatalf("TransformImage FixedWH failed: %v", err)
	}
	cropInfo, err := services.DecodeImageInfo(outPath)
	if err != nil || cropInfo.Width != 40 || cropInfo.Height != 40 {
		t.Fatalf("Expected 40x40 cropped image, got %dx%d, %v", cropInfo.Width, cropInfo.Height, err)
	}

	// 5. Test BatchTransform
	paths := []string{srcPath, outPath}
	var progressReports int
	succ, fail := services.BatchTransform(paths, opts, func(done, total int, current string) {
		progressReports++
	})
	if succ != 2 || fail != 0 {
		t.Fatalf("BatchTransform failed: succ=%d, fail=%d", succ, fail)
	}
	if progressReports == 0 {
		t.Fatal("Expected progress reports during BatchTransform")
	}
}

func TestBatchConvertFolder(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "source")
	subDir := filepath.Join(srcDir, "subfolder")
	outDir := filepath.Join(tmpDir, "output")

	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Create a dummy PNG image helper
	createDummyPNG := func(path string) {
		img := image.NewRGBA(image.Rect(0, 0, 20, 20))
		img.Set(0, 0, color.RGBA{R: 255, A: 255})
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		if err := png.Encode(f, img); err != nil {
			t.Fatal(err)
		}
	}

	// 2. Populate source folder with images and non-image files
	img1 := filepath.Join(srcDir, "pic1.png")
	img2 := filepath.Join(subDir, "pic2.png")
	txtFile := filepath.Join(srcDir, "notes.txt")
	createDummyPNG(img1)
	createDummyPNG(img2)
	_ = os.WriteFile(txtFile, []byte("ignore me"), 0644)

	// Test ScanImageFiles (recursive vs non-recursive)
	scannedRec, err := services.ScanImageFiles(srcDir, true)
	if err != nil || len(scannedRec) != 2 {
		t.Fatalf("Expected 2 images scanned recursively, got %d, err=%v", len(scannedRec), err)
	}

	scannedFlat, err := services.ScanImageFiles(srcDir, false)
	if err != nil || len(scannedFlat) != 1 {
		t.Fatalf("Expected 1 image scanned non-recursively, got %d, err=%v", len(scannedFlat), err)
	}

	// 3. Test BatchConvertFolder with output to new folder (PNG -> WebP)
	folderOpts := services.DefaultFolderBatchOptions()
	folderOpts.SrcDir = srcDir
	folderOpts.OutDir = outDir
	folderOpts.Recursive = true
	folderOpts.KeepDirTree = true
	folderOpts.SkipSameFormat = true
	folderOpts.ImageOpts.Encoder = "webp"

	var progressCalls atomic.Int32
	res, err := services.BatchConvertFolder(folderOpts, func(done, total int, current string) {
		progressCalls.Add(1)
	})
	if err != nil {
		t.Fatalf("BatchConvertFolder failed: %v", err)
	}
	if res.TotalFiles != 2 || res.SuccessFiles != 2 || res.FailedFiles != 0 || res.SkippedFiles != 0 {
		t.Fatalf("BatchConvertFolder result unexpected: %+v", res)
	}
	if progressCalls.Load() == 0 {
		t.Fatal("Expected progress callbacks")
	}

	// Check if directory tree mirrored
	destImg1 := filepath.Join(outDir, "pic1.webp")
	destImg2 := filepath.Join(outDir, "subfolder", "pic2.webp")
	if _, err := os.Stat(destImg1); err != nil {
		t.Fatalf("Output file 1 missing: %v", err)
	}
	if _, err := os.Stat(destImg2); err != nil {
		t.Fatalf("Output file 2 missing: %v", err)
	}

	// 4. Test SkipSameFormat in New Folder mode: should direct-copy files instead of re-encoding
	skipDir := filepath.Join(tmpDir, "out_skipped")
	skipOpts := services.DefaultFolderBatchOptions()
	skipOpts.SrcDir = outDir
	skipOpts.OutDir = skipDir
	skipOpts.Recursive = true
	skipOpts.SkipSameFormat = true
	skipOpts.ImageOpts.Encoder = "webp"

	skipRes, err := services.BatchConvertFolder(skipOpts, nil)
	if err != nil {
		t.Fatalf("BatchConvertFolder skip test failed: %v", err)
	}
	if skipRes.SuccessFiles != 2 || skipRes.CopiedFiles != 2 || skipRes.SkippedFiles != 0 {
		t.Fatalf("Expected 2 direct-copied files, got %+v", skipRes)
	}
	// Verify copied files exist
	if _, err := os.Stat(filepath.Join(skipDir, "pic1.webp")); err != nil {
		t.Fatalf("Direct-copied pic1.webp missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(skipDir, "subfolder", "pic2.webp")); err != nil {
		t.Fatalf("Direct-copied pic2.webp missing: %v", err)
	}

	// 5. Test ReplaceOriginal mode (PNG -> JPG)
	replaceDir := filepath.Join(tmpDir, "replace_test")
	if err := os.MkdirAll(replaceDir, 0755); err != nil {
		t.Fatal(err)
	}
	origPng := filepath.Join(replaceDir, "photo.png")
	createDummyPNG(origPng)

	repOpts := services.DefaultFolderBatchOptions()
	repOpts.SrcDir = replaceDir
	repOpts.ReplaceOriginal = true
	repOpts.ImageOpts.Encoder = "jpg"

	repRes, err := services.BatchConvertFolder(repOpts, nil)
	if err != nil || repRes.SuccessFiles != 1 {
		t.Fatalf("ReplaceOriginal batch failed: %+v, err=%v", repRes, err)
	}

	// Old file should be deleted, new jpg should exist
	if _, err := os.Stat(origPng); !os.IsNotExist(err) {
		t.Fatal("Original PNG file should have been deleted in ReplaceOriginal mode")
	}
	targetJpg := filepath.Join(replaceDir, "photo.jpg")
	if _, err := os.Stat(targetJpg); err != nil {
		t.Fatalf("Target JPG should exist: %v", err)
	}

	// 6. Test ReplaceOriginal mode with SkipSameFormat (JPG -> JPG)
	// In ReplaceOriginal mode, same format should be completely skipped
	repSameOpts := services.DefaultFolderBatchOptions()
	repSameOpts.SrcDir = replaceDir
	repSameOpts.ReplaceOriginal = true
	repSameOpts.SkipSameFormat = true
	repSameOpts.ImageOpts.Encoder = "jpg"

	repSameRes, err := services.BatchConvertFolder(repSameOpts, nil)
	if err != nil {
		t.Fatalf("ReplaceOriginal skip same failed: %v", err)
	}
	if repSameRes.SkippedFiles != 1 || repSameRes.SuccessFiles != 0 {
		t.Fatalf("Expected 1 completely skipped file in ReplaceOriginal mode, got %+v", repSameRes)
	}

	// 7. Test Preserving Original Dimensions:
	// Verify that BatchConvertFolder strictly ignores any polluted resize/crop options and keeps original dimensions.
	dimSrcDir := filepath.Join(tmpDir, "dim_src")
	dimOutDir := filepath.Join(tmpDir, "dim_out")
	if err := os.MkdirAll(dimSrcDir, 0755); err != nil {
		t.Fatal(err)
	}
	rawPNGPath := filepath.Join(dimSrcDir, "banner.png")
	// Create a 48x32 image
	img48x32 := image.NewRGBA(image.Rect(0, 0, 48, 32))
	f48, err := os.Create(rawPNGPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = png.Encode(f48, img48x32)
	_ = f48.Close()

	pollutedOpts := services.DefaultFolderBatchOptions()
	pollutedOpts.SrcDir = dimSrcDir
	pollutedOpts.OutDir = dimOutDir
	pollutedOpts.ImageOpts.Encoder = "webp"
	// Intentionally pollute ImageOpts with resize & crop options
	pollutedOpts.ImageOpts.Op = services.OpFixedWH
	pollutedOpts.ImageOpts.Width = 10
	pollutedOpts.ImageOpts.Height = 10
	pollutedOpts.ImageOpts.CropWidth = 10
	pollutedOpts.ImageOpts.CropHeight = 10

	dimRes, err := services.BatchConvertFolder(pollutedOpts, nil)
	if err != nil || dimRes.SuccessFiles != 1 {
		t.Fatalf("BatchConvertFolder dimension test failed: %v, res=%+v", err, dimRes)
	}

	convertedWebp := filepath.Join(dimOutDir, "banner.webp")
	infoWebp, err := services.DecodeImageInfo(convertedWebp)
	if err != nil {
		t.Fatalf("DecodeImageInfo failed on converted webp: %v", err)
	}
	if infoWebp.Width != 48 || infoWebp.Height != 32 {
		t.Fatalf("Expected preserved original dimensions 48x32, but got %dx%d", infoWebp.Width, infoWebp.Height)
	}
}



