package tool

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const (
	CBC_SALT_LEN = 8
	CBC_KEY_LEN  = 32
	CBC_CRED_LEN = 48 // CBC_BLOCK_LEN(16)+CBC_KEY_LEN(32)
)

// 预先生成PrePadPatterns
var prePadPatterns [aes.BlockSize + 1][]byte

// fix header
var cbcfixedSaltHeader = []byte("Salted__")

func init() {
	for i := 0; i < len(prePadPatterns); i++ {
		prePadPatterns[i] = bytes.Repeat([]byte{byte(i)}, i)
	}
	/*
		[]
		[1]
		[2 2]
		[3 3 3]
		[4 4 4 4]
		[5 5 5 5 5]
		[6 6 6 6 6 6]
		[7 7 7 7 7 7 7]
		[8 8 8 8 8 8 8 8]
		[9 9 9 9 9 9 9 9 9]
		[10 10 10 10 10 10 10 10 10 10]
		[11 11 11 11 11 11 11 11 11 11 11]
		[12 12 12 12 12 12 12 12 12 12 12 12]
		[13 13 13 13 13 13 13 13 13 13 13 13 13]
		[14 14 14 14 14 14 14 14 14 14 14 14 14 14]
		[15 15 15 15 15 15 15 15 15 15 15 15 15 15 15]
		[16 16 16 16 16 16 16 16 16 16 16 16 16 16 16 16]
	*/
}

type creds [CBC_CRED_LEN]byte

func (c *creds) Fill(pass, salt []byte) {
	m := c[:]
	buf := make([]byte, 0, 16+len(pass)+len(salt))
	var prevSum [16]byte
	for i := 0; i < 3; i++ { // salted 48byte, md5 16byte, three times could fill
		n := 0 // first prevSum length is zero,so n must be zero
		if i > 0 {
			n = 16
		}
		buf = buf[:n+len(pass)+len(salt)]
		copy(buf, prevSum[:])
		copy(buf[n:], pass)
		copy(buf[n+len(pass):], salt)
		prevSum = md5.Sum(buf)     // md5(prevSum + pass + salt)
		copy(m[i*16:], prevSum[:]) // concat every md5
	}
}

func OpenSSLAesEncToStr(plainText, pass string) (string, error) {
	dec, err := OpenSSLAesEnc(StringToBytes(plainText), StringToBytes(pass))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(dec), nil
}

func OpenSSlAesDecToStr(encryptText, pass string) (string, error) {
	buf := make([]byte, base64.StdEncoding.DecodedLen(len(encryptText)))
	n, err := base64.StdEncoding.Decode(buf, StringToBytes(encryptText))
	if err != nil {
		return "", err
	}
	b, err := OpenSSLAesDec(buf[:n], StringToBytes(pass))
	if err != nil {
		return "", err
	}
	return BytesToString(b), nil
}

func OpenSSLAesEnc(plainText, pass []byte) ([]byte, error) {
	var salt [CBC_SALT_LEN]byte
	_, err := io.ReadFull(rand.Reader, salt[:])
	if err != nil {
		return nil, fmt.Errorf("generate random salt error: %w", err)
	}

	var cred creds
	cred.Fill(pass, salt[:])
	key := cred[:CBC_KEY_LEN] // 32 bytes, 256 / 8
	iv := cred[CBC_KEY_LEN:]  // 16 bytes, same as block size

	/*
		|Salted__(8 byte)|salt(8 byte)|plaintext|
	*/
	data := make([]byte, len(plainText)+aes.BlockSize /*16*/)
	copy(data[0:], cbcfixedSaltHeader)
	copy(data[8:], salt[:])
	copy(data[aes.BlockSize:], plainText)

	padded := pkcs7Padding(data)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("NewCipher error: %w", err)
	}
	cbc := cipher.NewCBCEncrypter(block, iv)

	// 只从plaintext位置开始加密（上图）
	cbc.CryptBlocks(padded[aes.BlockSize:], padded[aes.BlockSize:])
	return padded, nil
}

func OpenSSLAesDec(encryptText, pass []byte) ([]byte, error) {
	/*
		|Salted__(8 byte)|salt(8 byte)|encrypt_text|
	*/
	if len(encryptText) < aes.BlockSize {
		return nil, errors.New("length illegal")
	}
	saltHeader := encryptText[:aes.BlockSize]
	if !bytes.Equal(saltHeader[:8], cbcfixedSaltHeader) {
		return nil, errors.New("check cbc fixed header error")
	}
	var cred creds
	cred.Fill(pass, saltHeader[8:])
	key := cred[:CBC_KEY_LEN] // 32 bytes, 256 / 8
	iv := cred[CBC_KEY_LEN:]  // 16 bytes, same as block size

	if len(encryptText)&15 != 0 {
		return nil, fmt.Errorf("encrypt text length illegal: len=%d", len(encryptText))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("NewCipher error: %w", err)
	}
	cbc := cipher.NewCBCDecrypter(block, iv)
	cbc.CryptBlocks(encryptText[aes.BlockSize:], encryptText[aes.BlockSize:])

	// 删除加密时候填充的padding
	return pkcs7UnPadding(encryptText[aes.BlockSize:])
}

func pkcs7Padding(data []byte) []byte {
	length := len(data) & 15 // len(data) % 16
	if length == 0 {
		return data
	}
	padlen := 16 - length
	return append(data, prePadPatterns[padlen]...)
}

func pkcs7UnPadding(data []byte) ([]byte, error) {
	if len(data)&15 != 0 || len(data) == 0 {
		return nil, fmt.Errorf("invalid data len %d", len(data))
	}
	padlen := int(data[len(data)-1])
	if padlen > aes.BlockSize || padlen == 0 {
		return nil, errors.New("param illegal")
	}
	if !bytes.Equal(prePadPatterns[padlen], data[len(data)-padlen:]) {
		return nil, errors.New("param illegal")
	}
	return data[:len(data)-padlen], nil
}

func PKCS5Padding(ciphertext []byte) []byte {
	return PKCS7Padding(ciphertext, 8)
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	// padText := bytes.Repeat([]byte{byte(padding)}, padding)
	// return append(ciphertext, padText...)
	return append(ciphertext, prePadPatterns[padding]...)
}

func PKCSUnPadding(data []byte) []byte {
	length := len(data)
	paddingNum := int(data[length-1])
	return data[:(length - paddingNum)]
}
