package srvs

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/welllog/olog"
	"github.com/welllog/otool/internal/errx"
	"github.com/welllog/otool/internal/tool"
)

type Encrypt struct{}

func (a *Encrypt) OpenSSLAesEnc(in, secret string) (string, error) {
	return errx.LogStr(tool.OpenSSLAesEncToStr(in, secret))
}

func (a *Encrypt) OpenSSLAesDec(in, secret string) (string, error) {
	return errx.LogStr(tool.OpenSSlAesDecToStr(in, secret))
}

func (a *Encrypt) Md5(in string) string {
	h := md5.Sum(tool.StringToBytes(in))
	return hex.EncodeToString(h[:])
}

func (a *Encrypt) Sha1(in string) string {
	h := sha1.Sum(tool.StringToBytes(in))
	return hex.EncodeToString(h[:])
}

func (a *Encrypt) Sha256(in string) string {
	h := sha256.Sum256(tool.StringToBytes(in))
	return hex.EncodeToString(h[:])
}

func (a *Encrypt) Base64Enc(in string) string {
	return base64.StdEncoding.EncodeToString(tool.StringToBytes(in))
}

func (a *Encrypt) Base64Dec(in string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(in)
	if err != nil {
		return "", errx.Log(err)
	}
	return tool.BytesToString(b), nil
}

func (a *Encrypt) UrlEnc(in string) string {
	return url.QueryEscape(in)
}

func (a *Encrypt) UrlDec(in string) (string, error) {
	return errx.LogStr(url.QueryUnescape(in))
}

func (a *Encrypt) OctEnc(in string) string {
	var buf strings.Builder
	buf.Grow(len(in) * 4)
	for _, char := range in {
		if char < 128 {
			buf.WriteString(fmt.Sprintf("\\%o", char))
		} else {
			bs := []byte(string(char))
			for _, b := range bs {
				buf.WriteString(fmt.Sprintf("\\%o", b))
			}
		}
	}
	return buf.String()
}

func (a *Encrypt) OctDec(in string) string {
	arr := strings.Split(in, "\\")
	var buf strings.Builder
	buf.Grow(len(in))
	for _, v := range arr {
		n, _ := strconv.ParseInt(v, 8, 64)
		buf.WriteByte(byte(n))
	}
	return buf.String()
}

func (a *Encrypt) HexEnc(in string) string {
	return hex.EncodeToString(tool.StringToBytes(in))
}

func (a *Encrypt) HexDec(in string) (string, error) {
	b, err := hex.DecodeString(in)
	if err != nil {
		return "", errx.Log(err)
	}
	return tool.BytesToString(b), nil
}

func (a *Encrypt) EncryptFile(pathName, savePath, secret string) error {
	olog.Debugf("pathName: %s, savePath: %s", pathName, savePath)

	saveName := filepath.Base(pathName) + ".enc"
	saveName = filepath.Join(savePath, saveName)

	olog.Debugf("EncryptFile save file name: %s", saveName)

	r, err := os.Open(pathName)
	if err != nil {
		return errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	w, err := os.Create(saveName)
	if err != nil {
		return errx.Logf("create file error: %s", err)
	}
	defer w.Close()

	return errx.Log(tool.EncryptFile(r, w, secret))
}

func (a *Encrypt) DecryptFile(pathName, savePath, secret string) error {
	olog.Debugf("pathName: %s, savePath: %s", pathName, savePath)

	saveName := filepath.Base(pathName)
	if strings.HasSuffix(saveName, ".enc") {
		saveName = strings.TrimSuffix(saveName, ".enc")
	} else {
		index := strings.Index(saveName, ".")
		if index > 0 {
			saveName = saveName[:index] + ".dec" + saveName[index:]
		} else {
			saveName += ".dec"
		}
	}

	saveName = filepath.Join(savePath, saveName)

	olog.Debugf("EncryptFile save file name: %s", saveName)

	r, err := os.Open(pathName)
	if err != nil {
		return errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	w, err := os.Create(saveName)
	if err != nil {
		return errx.Logf("create file error: %s", err)
	}
	defer w.Close()

	return errx.Log(tool.DecryptFile(r, w, secret))
}

func (a *Encrypt) Md5File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	h := md5.New()
	_, err = io.Copy(h, r)
	if err != nil {
		return "", errx.Log(err)
	}

	b := h.Sum(nil)
	return hex.EncodeToString(b), nil
}

func (a *Encrypt) Sha1File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	h := sha1.New()
	_, err = io.Copy(h, r)
	if err != nil {
		return "", errx.Log(err)
	}

	b := h.Sum(nil)
	return hex.EncodeToString(b), nil
}

func (a *Encrypt) Sha256File(pathName string) (string, error) {
	r, err := os.Open(pathName)
	if err != nil {
		return "", errx.Logf("open file error: %s", err)
	}
	defer r.Close()

	h := sha256.New()
	_, err = io.Copy(h, r)
	if err != nil {
		return "", errx.Log(err)
	}

	b := h.Sum(nil)
	return hex.EncodeToString(b), nil
}
