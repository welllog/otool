package srvs

import (
	"encoding/base64"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/welllog/golib/cryptz"
	"github.com/welllog/golib/hashz"
	"github.com/welllog/golib/strz"
	"github.com/welllog/otool/internal/errx"
)

type Encrypt struct{}

func (a *Encrypt) OpenSSLAesEnc(in, secret string) (string, error) {
	b, err := cryptz.Encrypt(in, secret)
	if err != nil {
		return "", errx.Log(err)
	}
	return strz.UnsafeString(b), nil
}

func (a *Encrypt) OpenSSLAesDec(in, secret string) (string, error) {
	b, err := cryptz.Decrypt(in, secret)
	if err != nil {
		return "", errx.Log(err)
	}
	return strz.UnsafeString(b), nil
}

func (a *Encrypt) Md5(in string) string {
	return hashz.Md5ToString(in)
}

func (a *Encrypt) Sha1(in string) string {
	return hashz.Sha1ToString(in)
}

func (a *Encrypt) Sha256(in string) string {
	return hashz.Sha256ToString(in)
}

func (a *Encrypt) Base64Enc(in string) string {
	return strz.Base64EncodeToString(in, base64.StdEncoding)
}

func (a *Encrypt) Base64Dec(in string) (string, error) {
	return errx.LogStr(strz.Base64DecodeToString(in, base64.StdEncoding))
}

func (a *Encrypt) UrlEnc(in string) string {
	return url.QueryEscape(in)
}

func (a *Encrypt) UrlDec(in string) (string, error) {
	return errx.LogStr(url.QueryUnescape(in))
}

func (a *Encrypt) OctEnc(in string) string {
	return strz.OctalEncodeToString(in)
}

func (a *Encrypt) OctDec(in string) string {
	return strz.OctalDecodeToString(in)
}

func (a *Encrypt) HexEnc(in string) string {
	return strz.HexEncodeToString(in)
}

func (a *Encrypt) HexDec(in string) (string, error) {
	return errx.LogStr(strz.HexDecodeToString(in))
}

func (a *Encrypt) EncryptFile(pathName, savePath, saveName, secret string) error {
	r, err := os.Open(pathName)
	if err != nil {
		return errx.Logf("open file error: %s", err.Error())
	}
	defer r.Close()

	savePathName := filepath.Join(savePath, saveName)
	_, err = os.Stat(savePathName)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return errx.Logf("file already exists: %s", savePathName)
	}

	w, err := os.OpenFile(savePathName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return errx.Logf("create file error: %s", err)
	}
	defer w.Close()

	return errx.Log(cryptz.EncryptStreamTo(w, r, secret))
}

func (a *Encrypt) DecryptFile(pathName, savePath, saveName, secret string) error {
	r, err := os.Open(pathName)
	if err != nil {
		return errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	savePathName := filepath.Join(savePath, saveName)
	_, err = os.Stat(savePathName)
	if err == nil || errors.Is(err, fs.ErrExist) {
		return errx.Logf("file already exists: %s", savePathName)
	}

	w, err := os.OpenFile(savePathName, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return errx.Logf("create file error: %s", err)
	}
	defer w.Close()

	return errx.Log(cryptz.DecryptStreamTo(w, r, secret))
}

func (a *Encrypt) Md5File(pathName string) (string, error) {
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

func (a *Encrypt) Sha1File(pathName string) (string, error) {
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

func (a *Encrypt) Sha256File(pathName string) (string, error) {
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

func (a *Encrypt) DefaultEncryptFilePath(pathName string) []string {
	pathName = pathName + ".enc"
	return []string{filepath.Dir(pathName), filepath.Base(pathName)}
}

func (a *Encrypt) DefaultDecryptFilePath(pathName string) []string {
	name := filepath.Base(pathName)
	if strings.HasSuffix(name, ".enc") {
		name = strings.TrimSuffix(name, ".enc")
	} else {
		index := strings.LastIndex(name, ".")
		if index > 0 {
			name = name[:index] + ".dec" + name[index:]
		} else {
			name += ".dec"
		}
	}

	return []string{filepath.Dir(pathName), name}
}
