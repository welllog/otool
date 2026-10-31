package app

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/welllog/otool/internal/services"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

const (
	PageText = iota
	PageFile
	PageConversion
	PageImage
	PageQRCode
)

type App struct {
	toastMu    sync.Mutex
	stateMu    sync.RWMutex
	win        *mygo.Window
	toastQueue []string

	// Active navigation page
	currentPage int

	// Services
	cryptoSrv *services.CryptoService

	// --- Text Page State ---
	textIn            string
	textOut           string
	textOpt           string
	textSecretKey     string
	textSecretVisible bool
	textCheckHmac     bool

	// --- File Page State ---
	filePath          string
	fileOpt           string
	fileSecretKey     string
	fileSecretVisible bool
	fileOutText       string
	fileProgress      int64
	fileShowProgress  bool
	fileIsProcessing  bool
	fileConfirmModal  bool

	// --- Conversion Page State ---
	convInText       string
	convInBase       int
	convOutBase      int
	convInSelectStr  string
	convOutSelectStr string
	convOutText      string
	convHasError     bool

	// --- QR Code Page State ---
	qrTab           int
	qrDecodeFile    string
	qrDecodeText    string
	qrDecodeThumb   *ui.Bitmap
	qrIsDecoding    bool
	qrEncodeText    string
	qrStyle         int    // 0: 标准方块, 1: 精致圆点, 2: 圆角流动
	qrRecover       int    // 0: L, 1: M, 2: Q, 3: H
	qrFormat        string // "png", "jpeg", "svg"
	qrFgColor       string // 前景色 Hex (如 "#000000", "#1e40af")
	qrSize          float64
	qrLogoPath      string
	qrLogoThumb     *ui.Bitmap
	qrSavePath      string
	qrPreviewBitmap *ui.Bitmap
	qrLastSavedName string

	// --- Image Page State ---
	imgTab          int // 0: 精细处理与裁剪, 1: 文件夹全量批处理
	imgFiles        []string
	imgInfos        []*services.ImageInfo
	imgThumbs       []*ui.Bitmap
	imgOpts         services.ImageOptions
	imgOutPath      string
	imgProgress     int
	imgDoneCount    int
	imgShowProgress bool
	imgCurFile      string
	imgIsProcessing bool

	// --- Image Folder Batch State ---
	folderSrcPath      string
	folderOutPath      string
	folderOutCustom    bool
	folderReplaceOrig  bool
	folderRecursive    bool
	folderSkipSame     bool
	folderKeepTree     bool
	folderIsProcessing bool
	folderShowProgress bool
	folderProgress     int
	folderDoneCount    int
	folderTotalCount   int
	folderCurFile      string
	folderSuccessCount int
	folderCopiedCount  int
	folderFailCount    int
	folderSkippedCount int
	folderConfirmModal bool

	// --- Image Crop Modal State ---
	imgCropModalOpen  bool
	imgCropTempX      int
	imgCropTempY      int
	imgCropTempW      int
	imgCropTempH      int
	imgCropOrigW      int
	imgCropOrigH      int
	imgCropPreview    *ui.Bitmap
	imgCropTarget     string
	imgCropDragMode   int
	imgCropDragStartX float32
	imgCropDragStartY float32
	imgCropInitX      int
	imgCropInitY      int
	imgCropInitW      int
	imgCropInitH      int
}

func defaultUserPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	desk := filepath.Join(home, "Desktop")
	if f, err := os.Stat(desk); err == nil && f.IsDir() {
		return desk
	}
	return home
}

func NewApp() *App {
	defPath := defaultUserPath()

	return &App{
		currentPage: PageText,
		cryptoSrv:   services.NewCryptoService(),

		// Text defaults
		textOpt: "opensslAesEnc",

		// File defaults
		fileOpt: "encrypt",

		// Conversion defaults
		convInText:       "255",
		convInBase:       10,
		convOutBase:      16,
		convInSelectStr:  "10 进制",
		convOutSelectStr: "16 进制",

		// QR defaults
		qrStyle:    0, // 标准方块
		qrRecover:  1, // M
		qrFormat:   "png",
		qrFgColor:  "#000000",
		qrSize:     256,
		qrSavePath: defPath,

		// Image defaults
		imgOpts:    services.DefaultImageOptions(),
		imgOutPath: defPath,

		// Folder batch defaults
		folderOutPath:   "",
		folderRecursive: true,
		folderSkipSame:  true,
		folderKeepTree:  true,
	}
}

// setFolderSrc sets source directory and defaults output to [src]_converted unless user customized it
func (a *App) setFolderSrc(src string) {
	a.folderSrcPath = src
	if !a.folderOutCustom || a.folderOutPath == "" {
		a.folderOutPath = src + "_converted"
		a.folderOutCustom = false
	}
}

func (a *App) SetWindow(win *mygo.Window) {
	a.win = win
}

func (a *App) update(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	} else {
		a.stateMu.Lock()
		defer a.stateMu.Unlock()
		fn()
	}
}

// Toast queues a notification message to be displayed in the UI. Safe from any goroutine.
func (a *App) Toast(msg string) {
	a.toastMu.Lock()
	a.toastQueue = append(a.toastQueue, msg)
	a.toastMu.Unlock()

	if a.win != nil {
		a.win.Invalidate()
	}
}

// GetQRDecodeResult safely reads QR decode state across goroutines.
func (a *App) GetQRDecodeResult() (text string, isDecoding bool) {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.qrDecodeText, a.qrIsDecoding
}

// GetFolderBatchResult safely reads folder batch processing state across goroutines.
func (a *App) GetFolderBatchResult() (isProcessing bool, success, copied, fail, skipped int) {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.folderIsProcessing, a.folderSuccessCount, a.folderCopiedCount, a.folderFailCount, a.folderSkippedCount
}

// View builds the entire native UI tree.
func (a *App) View(c *ui.Context) {
	t := c.Theme()

	// Flush queued toasts from background tasks
	a.toastMu.Lock()
	for len(a.toastQueue) > 0 {
		msg := a.toastQueue[0]
		a.toastQueue = a.toastQueue[1:]
		c.Toast(msg)
	}
	a.toastMu.Unlock()

	ui.Row(c).
		Fill().
		AlignItems(ui.Stretch).
		Children(func() {
			// 1. Sidebar Navigation
			a.sidebar(c)

			// 2. Main Content Area
			ui.Column(c).
				Grow(1).
				MinWidth(0).
				Background(t.Background).
				Children(func() {
					// Scrollable page content with comfortable padding
					ui.Scroll(c).
						Grow(1).
						Padding(24, 32, 32).
						Children(func() {
							switch a.currentPage {
							case PageText:
								a.viewText(c)
							case PageFile:
								a.viewFile(c)
							case PageConversion:
								a.viewConversion(c)
							case PageImage:
								a.viewImage(c)
							case PageQRCode:
								a.viewQRCode(c)
							}
						})
				})
		})
}

