package services

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/welllog/golib/cryptz"
	"github.com/welllog/golib/hashz"
	"github.com/welllog/golib/randz"
	"github.com/welllog/golib/strz"
)

type progressWriter struct {
	total      int64
	current    int64
	onProgress func(percent int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.current += int64(n)
	if pw.total > 0 && pw.onProgress != nil {
		pct := pw.current * 100 / pw.total
		if pct > 100 {
			pct = 100
		}
		pw.onProgress(pct)
	}
	return n, nil
}

// CryptoService provides cryptographic and hashing operations for text and files.
type CryptoService struct{}

func NewCryptoService() *CryptoService {
	return &CryptoService{}
}

func (s *CryptoService) OpenSSLAesEnc(in, secret string) (string, error) {
	b, err := cryptz.Encrypt(in, secret)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) OpenSSLAesDec(in, secret string) (string, error) {
	b, err := cryptz.Decrypt(in, secret)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) Md5(in string) string {
	return hashz.Md5ToString(in)
}

func (s *CryptoService) Sha1(in string) string {
	return hashz.Sha1ToString(in)
}

func (s *CryptoService) Sha224(in string) string {
	return hashz.Sha224ToString(in)
}

func (s *CryptoService) Sha256(in string) string {
	return hashz.Sha256ToString(in)
}

func (s *CryptoService) Sha384(in string) string {
	return hashz.Sha384ToString(in)
}

func (s *CryptoService) Sha512(in string) string {
	return hashz.Sha512ToString(in)
}

func (s *CryptoService) Sha512_224(in string) string {
	return hashz.Sha512_224ToString(in)
}

func (s *CryptoService) Sha512_256(in string) string {
	return hashz.Sha512_256ToString(in)
}

func (s *CryptoService) Hmac(in, secret, hm string) (string, error) {
	var h func() hash.Hash
	switch hm {
	case "md5":
		h = md5.New
	case "sha1":
		h = sha1.New
	case "sha224":
		h = sha256.New224
	case "sha256":
		h = sha256.New
	case "sha384":
		h = sha512.New384
	case "sha512":
		h = sha512.New
	case "sha512_224":
		h = sha512.New512_224
	case "sha512_256":
		h = sha512.New512_256
	default:
		return "", errors.New("未知哈希算法: " + hm)
	}

	return hashz.HmacToString(secret, in, h), nil
}

func (s *CryptoService) Base64Enc(in string) string {
	return strz.Base64EncodeToString(in, base64.StdEncoding)
}

func (s *CryptoService) Base64Dec(in string) string {
	return strz.Base64ParseToString(in)
}

func (s *CryptoService) UrlEnc(in string) string {
	return url.QueryEscape(in)
}

func (s *CryptoService) UrlDec(in string) (string, error) {
	return url.QueryUnescape(in)
}

func (s *CryptoService) OctEnc(in string) string {
	return strz.OctalFormatToString(in)
}

func (s *CryptoService) OctDec(in string) string {
	return strz.OctalParseToString(in)
}

func (s *CryptoService) HexEnc(in string) string {
	return strz.HexFormatToString(in)
}

func (s *CryptoService) HexDec(in string) string {
	return strz.HexParseToString(in)
}

func (s *CryptoService) UnicodeEnc(in string) string {
	return strz.UnicodeFormatToString(in)
}

func (s *CryptoService) UnicodeDec(in string) string {
	return strz.UnicodeParseToString(in)
}

func (s *CryptoService) Utf16Enc(in string) string {
	return strz.Utf16FormatToString(in)
}

func (s *CryptoService) Utf16Dec(in string) string {
	return strz.Utf16ParseToString(in)
}

func (s *CryptoService) Upper(in string) string {
	return strings.ToUpper(in)
}

func (s *CryptoService) Lower(in string) string {
	return strings.ToLower(in)
}

func makeTempPath(pathName, suffix string) string {
	dir := filepath.Dir(pathName)
	baseName := filepath.Base(pathName)
	// Try creating temp file in the same directory first to guarantee atomic rename
	tmp := filepath.Join(dir, fmt.Sprintf(".otool_tmp_%s_%d_%s%s", baseName, time.Now().UnixNano(), randz.String(8), suffix))
	if f, err := os.OpenFile(tmp, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666); err == nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return tmp
	}
	// Fallback to system temp directory
	return filepath.Join(os.TempDir(), fmt.Sprintf("otool_%s_%d_%s%s", baseName, time.Now().UnixNano(), randz.String(8), suffix))
}

func replaceFile(srcTemp, destTarget string) error {
	// 1. Try atomic rename
	if err := os.Rename(srcTemp, destTarget); err == nil {
		return nil
	}

	// 2. Fallback: Copy content if crossing filesystems / partition boundaries
	sf, err := os.Open(srcTemp)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.OpenFile(destTarget, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer df.Close()

	if _, err := io.Copy(df, sf); err != nil {
		return err
	}
	_ = sf.Close()
	_ = df.Close()
	_ = os.Remove(srcTemp)
	return nil
}

func (s *CryptoService) EncryptFile(pathName, secret string, onProgress func(percent int64)) error {
	baseName := filepath.Base(pathName)
	savePathName := makeTempPath(pathName, ".enc")

	fileInfo, err := os.Stat(pathName)
	if err != nil {
		return fmt.Errorf("获取文件 %s 信息失败: %w", baseName, err)
	}

	r, err := os.Open(pathName)
	if err != nil {
		return fmt.Errorf("打开 %s 失败: %w", baseName, err)
	}
	defer r.Close()

	w, err := os.OpenFile(savePathName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer func() {
		_ = w.Close()
		_ = os.Remove(savePathName)
	}()

	pw := &progressWriter{
		total:      fileInfo.Size() + 16,
		onProgress: onProgress,
	}
	dst := io.MultiWriter(w, pw)

	if err := cryptz.EncryptStreamTo(dst, r, secret); err != nil {
		return fmt.Errorf("加密 %s 失败: %w", baseName, err)
	}

	_ = r.Close()
	_ = w.Close()

	if err := replaceFile(savePathName, pathName); err != nil {
		return fmt.Errorf("加密替换原文件失败: %w", err)
	}

	if onProgress != nil {
		onProgress(100)
	}
	return nil
}

func (s *CryptoService) DecryptFile(pathName, secret string, onProgress func(percent int64)) error {
	baseName := filepath.Base(pathName)
	savePathName := makeTempPath(pathName, ".dec")

	fileInfo, err := os.Stat(pathName)
	if err != nil {
		return fmt.Errorf("获取文件 %s 信息失败: %w", baseName, err)
	}

	r, err := os.Open(pathName)
	if err != nil {
		return fmt.Errorf("打开 %s 失败: %w", baseName, err)
	}
	defer r.Close()

	w, err := os.OpenFile(savePathName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}
	defer func() {
		_ = w.Close()
		_ = os.Remove(savePathName)
	}()

	pw := &progressWriter{
		total:      fileInfo.Size() - 16,
		onProgress: onProgress,
	}
	dst := io.MultiWriter(w, pw)

	if err := cryptz.DecryptStreamTo(dst, r, secret); err != nil {
		return fmt.Errorf("解密 %s 失败: %w", baseName, err)
	}

	_ = r.Close()
	_ = w.Close()

	if err := replaceFile(savePathName, pathName); err != nil {
		return fmt.Errorf("解密替换原文件失败: %w", err)
	}

	if onProgress != nil {
		onProgress(100)
	}
	return nil
}

func (s *CryptoService) Md5File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", err
	}
	defer r.Close()

	b, err := hashz.Md5Stream(r)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) Sha1File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", err
	}
	defer r.Close()

	b, err := hashz.Sha1Stream(r)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) Sha224File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", err
	}
	defer r.Close()

	b, err := hashz.Sha224Stream(r)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) Sha256File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", err
	}
	defer r.Close()

	b, err := hashz.Sha256Stream(r)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) Sha384File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", err
	}
	defer r.Close()

	b, err := hashz.Sha384Stream(r)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

func (s *CryptoService) Sha512File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", err
	}
	defer r.Close()

	b, err := hashz.Sha512Stream(r)
	if err != nil {
		return "", err
	}
	return strz.UnsafeString(b), nil
}

