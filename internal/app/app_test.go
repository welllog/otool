package app

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/welllog/otool/internal/services"

	"github.com/egoist/mygo/ui"
)

func TestAppViewRender(t *testing.T) {
	app := NewApp()
	tt := ui.NewTester(app.View, 1080, 760)
	if tt == nil {
		t.Fatal("NewTester returned nil")
	}

	// Switch across all pages and verify each renders cleanly without panics
	pages := []int{PageText, PageFile, PageConversion, PageImage, PageQRCode}
	for _, page := range pages {
		app.currentPage = page
		tt.Frame()
	}
}

func TestAppTextOperations(t *testing.T) {
	app := NewApp()
	app.textIn = "hello world"
	app.textOpt = "base64Enc"

	// Trigger text execution within context
	tt := ui.NewTester(func(c *ui.Context) {
		app.executeTextTransform(c)
	}, 400, 300)
	_ = tt

	if app.textOut != "aGVsbG8gd29ybGQ=" {
		t.Fatalf("Expected base64 'aGVsbG8gd29ybGQ=', got %q", app.textOut)
	}

	// Test MD5
	app.textOpt = "md5"
	tt = ui.NewTester(func(c *ui.Context) {
		app.executeTextTransform(c)
	}, 400, 300)
	_ = tt

	if app.textOut != "5eb63bbbe01eeed093cb22bb8f5acdc3" {
		t.Fatalf("Expected MD5 hash, got %q", app.textOut)
	}

	// Test UPPER
	app.textOpt = "upper"
	tt = ui.NewTester(func(c *ui.Context) {
		app.executeTextTransform(c)
	}, 400, 300)
	_ = tt

	if app.textOut != "HELLO WORLD" {
		t.Fatalf("Expected 'HELLO WORLD', got %q", app.textOut)
	}

	// Test AES Encrypt & Decrypt via UI
	app.textIn = "Secret Message"
	app.textOpt = "opensslAesEnc"
	app.textSecretKey = "Passw0rd123!"
	tt = ui.NewTester(func(c *ui.Context) {
		app.executeTextTransform(c)
	}, 400, 300)
	_ = tt

	cipher := app.textOut
	if cipher == "" || cipher == app.textIn {
		t.Fatalf("AES encryption failed: %q", cipher)
	}

	app.textIn = cipher
	app.textOpt = "opensslAesDec"
	tt = ui.NewTester(func(c *ui.Context) {
		app.executeTextTransform(c)
	}, 400, 300)
	_ = tt

	if app.textOut != "Secret Message" {
		t.Fatalf("AES decryption failed: got %q, want 'Secret Message'", app.textOut)
	}
}

func TestAppConversionReaction(t *testing.T) {
	app := NewApp()
	app.convInText = "15"
	app.convInBase = 10
	app.convOutBase = 16

	tt := ui.NewTester(app.viewConversion, 600, 400)
	_ = tt

	if app.convOutText != "F" {
		t.Fatalf("Expected 15 in hex to be 'F', got %q", app.convOutText)
	}
	if app.convHasError {
		t.Fatal("convHasError should be false")
	}

	// Test invalid input
	app.convInText = "G"
	app.convInBase = 10
	tt.Frame()

	if !app.convHasError {
		t.Fatal("convHasError should be true for invalid base 10 digit 'G'")
	}
}

func TestAppQROperations(t *testing.T) {
	tmpDir := t.TempDir()

	app := NewApp()
	app.qrSavePath = tmpDir
	app.qrTab = 1
	app.qrEncodeText = "https://otool.app"
	app.qrSize = 128
	app.qrStyle = 1 // Dots
	app.qrFormat = "png"

	// 1. Test basic triggerQrEncode with Dots
	tt := ui.NewTester(func(c *ui.Context) {
		app.triggerQrEncode(c)
	}, 400, 300)
	_ = tt

	if app.qrPreviewBitmap == nil {
		t.Fatal("Expected qrPreviewBitmap to be generated")
	}

	savedFile := filepath.Join(tmpDir, app.qrLastSavedName)
	if _, err := os.Stat(savedFile); err != nil {
		t.Fatalf("QR file not saved: %v", err)
	}

	app.triggerQrDecode(savedFile)
	var finalDecoded string
	for i := 0; i < 50; i++ {
		text, decoding := app.GetQRDecodeResult()
		if !decoding && text != "" {
			finalDecoded = text
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if finalDecoded != "https://otool.app" {
		t.Fatalf("Decoded QR mismatch: got %q, want 'https://otool.app'", finalDecoded)
	}

	// 2. Test SVG Format Generation
	app.qrFormat = "svg"
	app.qrStyle = 2 // Rounded
	tt = ui.NewTester(func(c *ui.Context) {
		app.triggerQrEncode(c)
	}, 400, 300)
	_ = tt

	svgFile := filepath.Join(tmpDir, app.qrLastSavedName)
	if !strings.HasSuffix(svgFile, ".svg") {
		t.Fatalf("Expected .svg filename, got %s", app.qrLastSavedName)
	}
	if _, err := os.Stat(svgFile); err != nil {
		t.Fatalf("SVG file not found on disk: %v", err)
	}

	// 3. Test with Center Logo (via setQRLogo to simulate drag & drop / dialog select)
	logoImg := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			logoImg.Set(x, y, color.RGBA{R: 200, G: 10, B: 10, A: 255})
		}
	}
	logoFile := filepath.Join(tmpDir, "logo.png")
	lf, err := os.Create(logoFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(lf, logoImg); err != nil {
		lf.Close()
		t.Fatal(err)
	}
	lf.Close()

	// Test invalid file drop
	app.setQRLogo(filepath.Join(tmpDir, "invalid.txt"))
	if app.qrLogoPath != "" {
		t.Fatal("setQRLogo should reject non-image file")
	}

	// Test valid image drop
	app.setQRLogo(logoFile)
	if app.qrLogoPath != logoFile || app.qrLogoThumb == nil {
		t.Fatalf("setQRLogo failed: path=%s, thumb=%v", app.qrLogoPath, app.qrLogoThumb)
	}
	if app.qrRecover != 3 {
		t.Fatalf("setQRLogo should lock qrRecover to 3 (H), got %d", app.qrRecover)
	}

	app.qrFormat = "png"
	app.qrRecover = 0 // Should be forced to H(3)
	tt = ui.NewTester(func(c *ui.Context) {
		app.triggerQrEncode(c)
	}, 400, 300)
	_ = tt

	logoSavedFile := filepath.Join(tmpDir, app.qrLastSavedName)
	app.triggerQrDecode(logoSavedFile)
	var logoDecoded string
	for i := 0; i < 50; i++ {
		text, decoding := app.GetQRDecodeResult()
		if !decoding && text != "" {
			logoDecoded = text
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if logoDecoded != "https://otool.app" {
		t.Fatalf("Decoded QR with logo mismatch: got %q, want 'https://otool.app'", logoDecoded)
	}

	// 4. Test with Theme Color (Blue)
	app.qrFgColor = "#1e40af"
	app.qrLogoPath = "" // Remove logo
	app.qrStyle = 0     // Square
	tt = ui.NewTester(func(c *ui.Context) {
		app.triggerQrEncode(c)
	}, 400, 300)
	_ = tt

	blueSavedFile := filepath.Join(tmpDir, app.qrLastSavedName)
	app.triggerQrDecode(blueSavedFile)
	var blueDecoded string
	for i := 0; i < 50; i++ {
		text, decoding := app.GetQRDecodeResult()
		if !decoding && text != "" {
			blueDecoded = text
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if blueDecoded != "https://otool.app" {
		t.Fatalf("Decoded colored QR mismatch: got %q, want 'https://otool.app'", blueDecoded)
	}

	// 5. Test View Render in QR Code page
	app.currentPage = PageQRCode
	app.qrTab = 0 // Decode tab
	viewTester := ui.NewTester(app.View, 1080, 760)
	viewTester.Frame()

	app.qrTab = 1 // Encode tab
	viewTester.Frame()
}

func TestAppImageCropModal(t *testing.T) {
	app := NewApp()
	app.currentPage = PageImage
	app.imgOpts.Op = 8 // OpCustomCrop

	// Mock 1 image
	app.imgFiles = []string{"/tmp/test.png"}
	app.imgInfos = []*services.ImageInfo{
		{
			Name:   "test.png",
			Path:   "/tmp/test.png",
			Width:  1000,
			Height: 800,
		},
	}
	// Mock dummy bitmap
	dummyImg := ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 10, 10)))
	app.imgThumbs = []*ui.Bitmap{dummyImg}

	tt := ui.NewTester(func(c *ui.Context) {
		app.openCropModal(c)
	}, 600, 400)
	_ = tt

	if !app.imgCropModalOpen {
		t.Fatal("Crop modal should be open")
	}

	// 80% default verification
	if app.imgCropTempW != 800 || app.imgCropTempH != 640 {
		t.Fatalf("Expected 80%% default crop 800x640, got %dx%d", app.imgCropTempW, app.imgCropTempH)
	}
	if app.imgCropTempX != 100 || app.imgCropTempY != 80 {
		t.Fatalf("Expected centered offset 100,80, got %d,%d", app.imgCropTempX, app.imgCropTempY)
	}

	// Render the whole app view with modal open to verify GPU rendering without crash
	tt = ui.NewTester(app.View, 1080, 760)
	tt.Frame()
}

func TestAppImageFolderBatch(t *testing.T) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	subDir := filepath.Join(srcDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}

	createTestPNG := func(path string) {
		img := image.NewRGBA(image.Rect(0, 0, 32, 32))
		for x := 0; x < 32; x++ {
			for y := 0; y < 32; y++ {
				img.Set(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
			}
		}
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		_ = png.Encode(f, img)
	}

	createTestPNG(filepath.Join(srcDir, "pic1.png"))
	createTestPNG(filepath.Join(subDir, "pic2.png"))

	// Create an existing webp in srcDir to test direct-copy behavior
	existingWebp := filepath.Join(srcDir, "already.webp")
	_ = os.WriteFile(existingWebp, []byte("fake webp content for copy test"), 0644)

	app := NewApp()
	app.currentPage = PageImage
	app.imgTab = 1 // 文件夹全量批处理
	app.setFolderSrc(srcDir)

	expectedDefaultOut := srcDir + "_converted"
	if app.folderOutPath != expectedDefaultOut {
		t.Fatalf("Expected default out path %q, got %q", expectedDefaultOut, app.folderOutPath)
	}

	app.folderRecursive = true
	app.folderKeepTree = true
	app.folderSkipSame = true
	app.imgOpts.Encoder = "webp"

	// 1. Verify rendering folder tab
	tt := ui.NewTester(app.View, 1080, 760)
	tt.Frame()

	// 2. Trigger batch transform
	ttTrigger := ui.NewTester(func(c *ui.Context) {
		app.startFolderBatchTransform(c)
	}, 400, 300)
	_ = ttTrigger

	// Wait for background conversion completion
	var finalSuccess, finalCopied, finalFail int
	for i := 0; i < 50; i++ {
		time.Sleep(20 * time.Millisecond)
		isProc, succ, copied, fail, _ := app.GetFolderBatchResult()
		if !isProc && succ == 3 {
			finalSuccess = succ
			finalCopied = copied
			finalFail = fail
			break
		}
	}

	if finalSuccess != 3 || finalCopied != 1 {
		t.Fatalf("Expected 3 success files (1 copied), got success=%d, copied=%d, failed=%d", finalSuccess, finalCopied, finalFail)
	}

	// Verify output files exist in expectedDefaultOut
	if _, err := os.Stat(filepath.Join(expectedDefaultOut, "pic1.webp")); err != nil {
		t.Fatalf("Expected output pic1.webp to exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(expectedDefaultOut, "sub", "pic2.webp")); err != nil {
		t.Fatalf("Expected output sub/pic2.webp to exist: %v", err)
	}
	// Verify directly copied file exists
	copiedPath := filepath.Join(expectedDefaultOut, "already.webp")
	if data, err := os.ReadFile(copiedPath); err != nil || string(data) != "fake webp content for copy test" {
		t.Fatalf("Direct-copied already.webp failed: %v", err)
	}

	// 3. Test confirm modal for replaceOriginal
	app.folderReplaceOrig = true
	app.folderConfirmModal = true
	tt.Frame() // Render with modal open
	if !app.folderConfirmModal {
		t.Fatal("Expected folderConfirmModal to be true")
	}
}
