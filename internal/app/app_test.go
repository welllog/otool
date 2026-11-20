package app

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/welllog/otool/internal/services"
	"github.com/welllog/otool/internal/services/dedup"
	"github.com/welllog/otool/internal/storage"

	"github.com/egoist/mygo/ui"
)

func TestMain(m *testing.M) {
	oldHome, hadHome := os.LookupEnv("HOME")
	oldCache, hadCache := os.LookupEnv("XDG_CACHE_HOME")
	home, err := os.MkdirTemp("", "otool-test-home-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("HOME", home)
	_ = os.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	code := m.Run()
	if hadHome {
		_ = os.Setenv("HOME", oldHome)
	} else {
		_ = os.Unsetenv("HOME")
	}
	if hadCache {
		_ = os.Setenv("XDG_CACHE_HOME", oldCache)
	} else {
		_ = os.Unsetenv("XDG_CACHE_HOME")
	}
	_ = os.RemoveAll(home)
	os.Exit(code)
}

func TestAppViewRender(t *testing.T) {
	app := NewApp()
	tt := ui.NewTester(app.View, 1080, 760)
	if tt == nil {
		t.Fatal("NewTester returned nil")
	}

	// Switch across all pages and verify each renders cleanly without panics
	pages := []int{PageText, PageFile, PageConversion, PageImage, PageQRCode, PageDedup}
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

func TestAppDedupEmptyProfilesFallback(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".otool"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".otool", "storage_profiles.json"), []byte(`{"version":1,"last_used_id":"","profiles":[]}`), 0600); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.currentPage = PageDedup
	tt := ui.NewTester(app.View, 1080, 760)
	tt.Frame()

	if app.dedupProfiles == nil || len(app.dedupProfiles.Profiles) == 0 {
		t.Fatal("expected an empty profile config to fall back to the default local profile")
	}
}

func TestAppDedupDeletionPreservesUserSelection(t *testing.T) {
	dir := t.TempDir()
	bestPath := filepath.Join(dir, "best.jpg")
	keptPath := filepath.Join(dir, "kept.jpg")
	pendingPath := filepath.Join(dir, "pending.jpg")
	content := []byte("photo")
	for _, p := range []string{bestPath, keptPath, pendingPath} {
		if err := os.WriteFile(p, content, 0644); err != nil {
			t.Fatal(err)
		}
	}
	contentHash, err := dedup.ComputeFullContentHash(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.dedupProfiles = &storage.ProfilesFile{
		Version:    1,
		LastUsedID: "local_test",
		Profiles: []storage.Profile{
			{ID: "local_test", Name: "测试本地", Type: storage.DriverLocal, Local: storage.LocalConfig{DefaultPath: dir}},
		},
	}
	app.dedupActiveProfileIdx = 0
	app.dedupScanStore = storage.NewLocalStorage("local_test")
	app.dedupScanProfileID = "local_test"
	app.dedupConfirmTrashOnly = false

	bestInfo, _ := os.Stat(bestPath)
	keptInfo, _ := os.Stat(keptPath)
	pendingInfo, _ := os.Stat(pendingPath)
	best := &dedup.PhotoMeta{ID: 1, StorageID: "local_test", Path: bestPath, Size: 5, ModTime: bestInfo.ModTime(), ContentHash: contentHash, Selected: false}
	kept := &dedup.PhotoMeta{ID: 2, StorageID: "local_test", Path: keptPath, Size: 5, ModTime: keptInfo.ModTime(), ContentHash: contentHash, Selected: false}
	pending := &dedup.PhotoMeta{ID: 3, StorageID: "local_test", Path: pendingPath, Size: 5, ModTime: pendingInfo.ModTime(), ContentHash: contentHash, Selected: true}
	app.dedupGroups = []*dedup.DuplicateGroup{{
		GroupID:    "grp",
		Mode:       dedup.ModeSimilar,
		BestPickID: 1,
		Photos:     []*dedup.PhotoMeta{best, kept, pending},
	}}

	app.executeDedupDeletion(true)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		app.stateMu.RLock()
		deleting := app.dedupIsDeleting
		app.stateMu.RUnlock()
		if !deleting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	app.stateMu.RLock()
	deleting := app.dedupIsDeleting
	groups := app.dedupGroups
	app.stateMu.RUnlock()
	if deleting {
		t.Fatal("timed out waiting for dedup deletion")
	}
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("expected pending photo to be deleted, stat err=%v", err)
	}
	if len(groups) != 1 || len(groups[0].Photos) != 2 {
		t.Fatalf("expected one remaining two-photo group, got %#v", groups)
	}
	for _, p := range groups[0].Photos {
		if p.ID == kept.ID && p.Selected {
			t.Fatal("user-kept photo was re-selected after deletion")
		}
	}
}

func TestAppDedupDeletionVisibleOnlyScope(t *testing.T) {
	dir := t.TempDir()
	content := []byte("photo")
	contentHash, err := dedup.ComputeFullContentHash(bytes.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.dedupProfiles = &storage.ProfilesFile{
		Version:    1,
		LastUsedID: "local_scope",
		Profiles: []storage.Profile{
			{ID: "local_scope", Name: "测试本地", Type: storage.DriverLocal, Local: storage.LocalConfig{DefaultPath: dir}},
		},
	}
	app.dedupActiveProfileIdx = 0
	app.dedupScanStore = storage.NewLocalStorage("local_scope")
	app.dedupScanProfileID = "local_scope"
	app.dedupConfirmTrashOnly = false
	// Only the first two groups are expanded for review; the third is a
	// paginated-away group preselected by the recommendation.
	app.dedupVisibleGroupLimit = 2

	newMeta := func(id uint64, name string, selected bool) *dedup.PhotoMeta {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return &dedup.PhotoMeta{
			ID:          id,
			StorageID:   "local_scope",
			Path:        path,
			Name:        name,
			Size:        fi.Size(),
			ModTime:     fi.ModTime(),
			ContentHash: contentHash,
			Selected:    selected,
		}
	}

	app.dedupGroups = []*dedup.DuplicateGroup{
		{GroupID: "g1", Mode: dedup.ModeSimilar, BestPickID: 1, Photos: []*dedup.PhotoMeta{newMeta(1, "best1.jpg", false), newMeta(2, "dup1.jpg", true)}},
		{GroupID: "g2", Mode: dedup.ModeSimilar, BestPickID: 3, Photos: []*dedup.PhotoMeta{newMeta(3, "best2.jpg", false), newMeta(4, "dup2.jpg", false)}},
		{GroupID: "g3", Mode: dedup.ModeSimilar, BestPickID: 5, Photos: []*dedup.PhotoMeta{newMeta(5, "best3.jpg", false), newMeta(6, "dup3.jpg", true)}},
	}

	app.executeDedupDeletion(false)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		app.stateMu.RLock()
		deleting := app.dedupIsDeleting
		app.stateMu.RUnlock()
		if !deleting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	app.stateMu.RLock()
	deleting := app.dedupIsDeleting
	groups := app.dedupGroups
	app.stateMu.RUnlock()
	if deleting {
		t.Fatal("timed out waiting for scoped dedup deletion")
	}

	// The selection inside the expanded window was honored.
	if _, err := os.Stat(filepath.Join(dir, "dup1.jpg")); !os.IsNotExist(err) {
		t.Fatalf("expected visible selected photo to be deleted, stat err=%v", err)
	}
	// The paginated-away group kept both its photos and its selection.
	if _, err := os.Stat(filepath.Join(dir, "dup3.jpg")); err != nil {
		t.Fatalf("expected deferred group photo to survive visible-only cleanup: %v", err)
	}
	// The unselected visible group was left untouched.
	if _, err := os.Stat(filepath.Join(dir, "dup2.jpg")); err != nil {
		t.Fatalf("expected unselected visible photo to survive: %v", err)
	}

	if len(groups) != 2 {
		t.Fatalf("expected the two remaining two-photo groups to survive, got %d", len(groups))
	}
	var deferredGroup *dedup.DuplicateGroup
	for _, g := range groups {
		if g.GroupID == "g3" {
			deferredGroup = g
		}
	}
	if deferredGroup == nil {
		t.Fatal("deferred group was dropped from the result list")
	}
	foundSelected := false
	for _, p := range deferredGroup.Photos {
		if p.ID == 6 && p.Selected {
			foundSelected = true
		}
	}
	if !foundSelected {
		t.Fatal("deferred group lost its preselection; a later batch cannot continue where review left off")
	}

	// A visible-only round with nothing selected inside the window must be a
	// no-op even though hidden groups still carry selections.
	app.dedupVisibleGroupLimit = 1
	for _, p := range groups[0].Photos {
		p.Selected = false
	}
	app.executeDedupDeletion(false)
	app.stateMu.RLock()
	deleting = app.dedupIsDeleting
	app.stateMu.RUnlock()
	if deleting {
		t.Fatal("visible-only cleanup without a selection must not start a worker")
	}
	if _, err := os.Stat(filepath.Join(dir, "dup3.jpg")); err != nil {
		t.Fatalf("deferred photo must survive the empty visible selection guard: %v", err)
	}
}

func TestDedupWebdavPasswordSaveError(t *testing.T) {
	profiles := []storage.Profile{
		{ID: "has_pass", WebDAV: storage.WebDAVConfig{Password: "enc:AAAA"}},
		{ID: "no_pass", WebDAV: storage.WebDAVConfig{Password: ""}},
	}
	cases := []struct {
		name       string
		editingIdx int
		typed      string
		wantErr    bool
	}{
		{"editing profile with a stored password and empty input", 0, "", true},
		{"editing profile with a stored password and new input", 0, "newpass", false},
		{"editing password-less profile with empty input", 1, "", false},
		{"creating a new profile with empty input", -1, "", false},
		{"out-of-range editing index", 7, "", false},
	}
	for _, tc := range cases {
		got := dedupWebdavPasswordSaveError(tc.editingIdx, profiles, tc.typed)
		if (got != "") != tc.wantErr {
			t.Errorf("%s: got error %q, wantErr=%v", tc.name, got, tc.wantErr)
		}
	}
}

func TestAppDedupOperations(t *testing.T) {
	app := NewApp()
	app.currentPage = PageDedup
	tt := ui.NewTester(app.View, 1080, 760)

	// 1. Initial Frame render
	tt.Frame()

	// 2. Ensure profiles loaded
	app.ensureDedupProfilesLoaded()
	if app.dedupProfiles == nil || len(app.dedupProfiles.Profiles) == 0 {
		t.Fatal("Expected dedupProfiles to be loaded")
	}

	// 3. Simulate groups & recommend keep all
	app.dedupGroups = []*dedup.DuplicateGroup{
		{
			GroupID:    "grp_1",
			Mode:       dedup.ModeSimilar,
			BestPickID: 1,
			Photos: []*dedup.PhotoMeta{
				{ID: 1, Name: "best.jpg", Size: 2048, Selected: false},
				{ID: 2, Name: "copy.jpg", Size: 1024, Selected: true},
			},
			TotalWastedBytes: 1024,
		},
	}
	app.recalculateDedupStats()
	if app.dedupSelectedDeleteCount != 1 || app.dedupSelectedDeleteBytes != 1024 {
		t.Fatalf("expected 1 delete item (1024 bytes), got %d (%d)", app.dedupSelectedDeleteCount, app.dedupSelectedDeleteBytes)
	}

	// 4. Test applyRecommendedKeepAll
	app.applyRecommendedKeepAll()
	if app.dedupGroups[0].Photos[0].Selected != false {
		t.Fatal("expected best pick to not be selected for delete")
	}
	if app.dedupGroups[0].Photos[1].Selected != true {
		t.Fatal("expected copy to be selected for delete")
	}

	// 5. Render frame with groups and modal
	app.dedupShowConfigModal = true
	tt.Frame()
	app.dedupShowConfigModal = false
	app.dedupShowConfirmModal = true
	tt.Frame()
	app.dedupShowConfirmModal = false
	app.dedupCacheStatsLoaded = true
	app.dedupCacheStatsBytes = 1024
	app.dedupCacheStatsCount = 2
	app.dedupCacheStatsFpCount = 2
	app.dedupShowCacheClearModal = true
	tt.Frame()
}

func TestAppDedupThumbnailLoadsAsynchronously(t *testing.T) {
	app := NewApp()
	key := storage.GenerateCacheKey("async_thumb_test", filepath.Join(t.TempDir(), "thumb.png"), 1, time.Now())

	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for y := 0; y < 30; y++ {
		for x := 0; x < 40; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 120, B: 220, A: 255})
		}
	}
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	cm := storage.GetCacheManager()
	if err := cm.SaveThumbnail(key, buf.Bytes()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cm.RemoveThumbnail(key) })

	photo := &dedup.PhotoMeta{CacheKey: key, Path: filepath.Join(t.TempDir(), "thumb.png")}
	bmp, loading := app.getDedupThumbBitmap(photo)
	if bmp != nil {
		t.Fatal("first lookup must not decode a thumbnail on the UI thread")
	}
	if !loading {
		t.Fatal("expected the first lookup to report an in-flight load")
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		if bmp, _ = app.getDedupThumbBitmap(photo); bmp != nil {
			return
		}
	}
	t.Fatal("thumbnail was not loaded asynchronously")
}

func TestAppDedupBrokenProfileConfigIsNotOverwritten(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfgDir := filepath.Join(home, ".otool")
	if err := os.MkdirAll(cfgDir, 0700); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(cfgDir, "storage_profiles.json")
	broken := []byte(`{"version":1,"profiles":`)
	if err := os.WriteFile(cfgPath, broken, 0600); err != nil {
		t.Fatal(err)
	}

	app := NewApp()
	app.currentPage = PageDedup
	tt := ui.NewTester(app.View, 1080, 760)
	tt.Frame()

	if app.dedupProfilesLoadErr == "" {
		t.Fatal("expected a visible profile load error")
	}
	if app.dedupProfiles == nil || len(app.dedupProfiles.Profiles) == 0 {
		t.Fatal("expected an in-memory fallback profile")
	}

	app.dedupProfiles.Profiles[0].Local.DefaultPath = filepath.Join(home, "photos")
	app.saveDedupProfiles()

	after, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(broken) {
		t.Fatalf("broken profile config was overwritten: %s", string(after))
	}
}

func TestDedupFileStillMatchesRejectsDirectoryAndChangedContent(t *testing.T) {
	dir := t.TempDir()
	store := storage.NewLocalStorage("match_test")

	dirPath := filepath.Join(dir, "folder.jpg")
	if err := os.Mkdir(dirPath, 0755); err != nil {
		t.Fatal(err)
	}
	if dedupFileStillMatches(context.Background(), store, &dedup.PhotoMeta{Path: dirPath, Size: 1}) {
		t.Fatal("directory path must not match a photo")
	}

	filePath := filepath.Join(dir, "photo.jpg")
	data := []byte("photo")
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		t.Fatal(err)
	}
	fi, err := store.Stat(context.Background(), filePath)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := dedup.ComputeFullContentHash(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	p := &dedup.PhotoMeta{Path: filePath, Size: fi.Size, ContentHash: hash}
	if !dedupFileStillMatches(context.Background(), store, p) {
		t.Fatal("expected matching file to pass")
	}

	if err := os.WriteFile(filePath, []byte("other"), 0644); err != nil {
		t.Fatal(err)
	}
	if dedupFileStillMatches(context.Background(), store, p) {
		t.Fatal("expected changed same-size content to be rejected")
	}
}

func TestDedupFileStillMatchesRejectsMissingContentHash(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "photo.jpg")
	data := []byte("photo")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	store := storage.NewLocalStorage("match_missing_hash")
	p := &dedup.PhotoMeta{Path: path, Size: fi.Size(), ModTime: fi.ModTime()}
	// Even if size and mtime are unchanged, a missing scan-time content hash
	// must fail closed rather than allow a metadata-only deletion.
	if dedupFileStillMatches(context.Background(), store, p) {
		t.Fatal("expected missing ContentHash to be rejected")
	}
}
