package srvs

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"hash"
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
	return strz.OctalEncodeToString(in)
}

func (e *Encrypt) OctDec(in string) string {
	return strz.OctalDecodeToString(in)
}

func (e *Encrypt) HexEnc(in string) string {
	return strz.HexEncodeToString(in)
}

func (e *Encrypt) HexDec(in string) (string, error) {
	return errx.LogStr(strz.HexDecodeToString(in))
}

func (e *Encrypt) EncryptFile(pathName, savePath, saveName, secret string) error {
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

func (e *Encrypt) DecryptFile(pathName, savePath, saveName, secret string) error {
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

func (e *Encrypt) DefaultEncryptFilePath(pathName string) []string {
	pathName = pathName + ".enc"
	return []string{filepath.Dir(pathName), filepath.Base(pathName)}
}

func (e *Encrypt) DefaultDecryptFilePath(pathName string) []string {
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
