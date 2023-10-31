package srvs

import (
	"context"
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
	"time"

	"github.com/welllog/golib/cryptz"
	"github.com/welllog/golib/goz"
	"github.com/welllog/golib/hashz"
	"github.com/welllog/golib/randz"
	"github.com/welllog/golib/strz"
	"github.com/welllog/olog"
	"github.com/welllog/otool/internal/errx"
)

type Encrypt struct {
	Ctx context.Context
	Gow *goz.Limiter
}

func (e *Encrypt) OpenSSLAesEnc(in, secret string) (string, error) {
	b, err := cryptz.Encrypt(in, secret)
	if err != nil {
		return "", errx.Log(err)
	}
	return strz.UnsafeString(b), nil
}

func (e *Encrypt) OpenSSLAesDec(in, secret string) (string, error) {
	b, err := cryptz.Decrypt(in, secret)
	if err != nil {
		return "", errx.Log(err)
	}
	return strz.UnsafeString(b), nil
}

func (e *Encrypt) Md5(in string) string {
	return hashz.Md5ToString(in)
}

func (e *Encrypt) Sha1(in string) string {
	return hashz.Sha1ToString(in)
}

func (e *Encrypt) Sha224(in string) string {
	return hashz.Sha224ToString(in)
}

func (e *Encrypt) Sha256(in string) string {
	return hashz.Sha256ToString(in)
}

func (e *Encrypt) Sha384(in string) string {
	return hashz.Sha384ToString(in)
}

func (e *Encrypt) Sha512(in string) string {
	return hashz.Sha512ToString(in)
}

func (e *Encrypt) Sha512_224(in string) string {
	return hashz.Sha512_224ToString(in)
}

func (e *Encrypt) Sha512_256(in string) string {
	return hashz.Sha512_256ToString(in)
}

func (e *Encrypt) Hmac(in, secret, hm string) (string, error) {
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
		return "", errors.New("Unknown hash method")
	}

	return hashz.HmacToString(secret, in, h), nil
}

func (e *Encrypt) Base64Enc(in string) string {
	return strz.Base64EncodeToString(in, base64.StdEncoding)
}

func (e *Encrypt) Base64Dec(in string) (string, error) {
	return errx.LogStr(strz.Base64DecodeToString(in, base64.StdEncoding))
}

func (e *Encrypt) UrlEnc(in string) string {
	return url.QueryEscape(in)
}

func (e *Encrypt) UrlDec(in string) (string, error) {
	return errx.LogStr(url.QueryUnescape(in))
}

func (e *Encrypt) OctEnc(in string) string {
	return strz.OctalFormatToString(in)
}

func (e *Encrypt) OctDec(in string) string {
	return strz.OctalParseToString(in)
}

func (e *Encrypt) HexEnc(in string) string {
	return strz.HexFormatToString(in)
}

func (e *Encrypt) HexDec(in string) string {
	return strz.HexParseToString(in)
}

func (e *Encrypt) UnicodeEnc(in string) string {
	return strz.UnicodeFormatToString(in)
}

func (e *Encrypt) UnicodeDec(in string) string {
	return strz.UnicodeParseToString(in)
}

func (e *Encrypt) Utf16Enc(in string) string {
	return strz.Utf16FormatToString(in)
}

func (e *Encrypt) Utf16Dec(in string) string {
	return strz.Utf16ParseToString(in)
}

func (e *Encrypt) EncryptFile(pathName, secret string) (string, error) {
	baseName := filepath.Base(pathName)
	savePathName := filepath.Join(os.TempDir(),
		fmt.Sprintf("otool_%s_%d_%d.enc", baseName, time.Now().Unix(), randz.String(10)),
	)

	fileInfo, err := os.Stat(pathName)
	if err != nil {
		return "", errx.Logf("获取文件%s信息失败: %s", baseName, err.Error())
	}

	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("打开%s失败: %s", baseName, err.Error())
	}

	w, err := os.OpenFile(savePathName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		_ = r.Close()
		return "", errx.Logf("创建临时文件失败: %s", err.Error())
	}

	progress := streamProgress{size: fileInfo.Size() + 16, event: savePathName, ctx: e.Ctx}
	e.Gow.Go(func() {
		defer func() {
			_ = w.Close()
			_ = r.Close()
			progress.close()
		}()

		dst := io.MultiWriter(w, &progress)
		err = cryptz.EncryptStreamTo(dst, r, secret)
		if err != nil {
			_ = os.Remove(savePathName)
			olog.Errorf("加密%s失败: %s", baseName, err.Error())

			notify(e.Ctx, NotifyEvent{
				Info: fmt.Sprintf("加密%s失败: %s", baseName, err.Error()),
				Type: "danger",
			})
			return
		}

		_ = r.Close()
		_ = w.Close()

		err = os.Rename(savePathName, pathName)
		if err != nil {
			_ = os.Remove(savePathName)
			olog.Errorf("重命名%s失败: %s", baseName, err.Error())

			notify(e.Ctx, NotifyEvent{
				Info: fmt.Sprintf("加密替换%s失败: %s", baseName, err.Error()),
				Type: "danger",
			})
			return
		}

		notify(e.Ctx, NotifyEvent{
			Info: fmt.Sprintf("加密%s成功", baseName),
			Type: "success",
		})
	})

	return savePathName, nil
}

func (e *Encrypt) DecryptFile(pathName, secret string) (string, error) {
	baseName := filepath.Base(pathName)
	savePathName := filepath.Join(os.TempDir(),
		fmt.Sprintf("otool_%s_%d_%d.dec", baseName, time.Now().Unix(), randz.String(10)),
	)

	fileInfo, err := os.Stat(pathName)
	if err != nil {
		return "", errx.Logf("获取文件%s信息失败: %s", baseName, err.Error())
	}

	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("打开%s失败: %s", baseName, err.Error())
	}

	w, err := os.OpenFile(savePathName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		_ = r.Close()
		return "", errx.Logf("创建临时文件失败: %s", err.Error())
	}

	progress := streamProgress{size: fileInfo.Size() - 16, event: savePathName, ctx: e.Ctx}
	e.Gow.Go(func() {
		defer func() {
			_ = w.Close()
			_ = r.Close()
			progress.close()
		}()

		dst := io.MultiWriter(w, &progress)
		err = cryptz.DecryptStreamTo(dst, r, secret)
		if err != nil {
			_ = os.Remove(savePathName)
			olog.Errorf("解密%s失败: %s", baseName, err.Error())

			notify(e.Ctx, NotifyEvent{
				Info: fmt.Sprintf("解密%s失败: %s", baseName, err.Error()),
				Type: "danger",
			})
			return
		}

		_ = r.Close()
		_ = w.Close()

		err = os.Rename(savePathName, pathName)
		if err != nil {
			_ = os.Remove(savePathName)
			olog.Errorf("重命名%s失败: %s", baseName, err.Error())

			notify(e.Ctx, NotifyEvent{
				Info: fmt.Sprintf("解密替换%s失败: %s", baseName, err.Error()),
				Type: "danger",
			})
			return
		}

		notify(e.Ctx, NotifyEvent{
			Info: fmt.Sprintf("解密%s成功", baseName),
			Type: "success",
		})
	})

	return savePathName, nil
}

func (e *Encrypt) Md5File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	b, err := hashz.Md5Stream(r)
	if err != nil {
		return "", errx.Log(err)
	}

	return strz.UnsafeString(b), nil
}

func (e *Encrypt) Sha1File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	b, err := hashz.Sha1Stream(r)
	if err != nil {
		return "", errx.Log(err)
	}

	return strz.UnsafeString(b), nil
}

func (e *Encrypt) Sha224File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	b, err := hashz.Sha224Stream(r)
	if err != nil {
		return "", errx.Log(err)
	}

	return strz.UnsafeString(b), nil
}

func (e *Encrypt) Sha256File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	b, err := hashz.Sha256Stream(r)
	if err != nil {
		return "", errx.Log(err)
	}

	return strz.UnsafeString(b), nil
}

func (e *Encrypt) Sha384File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	b, err := hashz.Sha384Stream(r)
	if err != nil {
		return "", errx.Log(err)
	}

	return strz.UnsafeString(b), nil
}

func (e *Encrypt) Sha512File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	b, err := hashz.Sha512Stream(r)
	if err != nil {
		return "", errx.Log(err)
	}

	return strz.UnsafeString(b), nil
}
