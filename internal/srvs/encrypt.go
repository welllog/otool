package srvs

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/welllog/otool/internal/tool"
)

type Encrypt struct{}

func (a *Encrypt) OpenSSLAesEnc(in, secret string) (string, error) {
	return tool.OpenSSLAesEncToStr(in, secret)
}

func (a *Encrypt) OpenSSLAesDec(in, secret string) (string, error) {
	return tool.OpenSSlAesDecToStr(in, secret)
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
		return "", err
	}
	return tool.BytesToString(b), nil
}

func (a *Encrypt) UrlEnc(in string) string {
	return url.QueryEscape(in)
}

func (a *Encrypt) UrlDec(in string) (string, error) {
	return url.QueryUnescape(in)
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
		return "", err
	}
	return tool.BytesToString(b), nil
}
