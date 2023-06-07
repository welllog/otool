package tool

import (
	"os"
	"testing"
)

func TestEncryptFile(t *testing.T) {
	f, err := os.Open("1.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	f2, err := os.Create("1.txt.enc")
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()

	err = EncryptFile(f, f2, "123456")
	if err != nil {
		t.Fatal(err)
	}

}

func TestDecryptFile(t *testing.T) {
	f, err := os.Open("1.txt.enc")
	if err != nil {
		t.Fatal(err)
	}

	defer f.Close()

	f2, err := os.Create("1.txt")
	if err != nil {
		t.Fatal(err)
	}

	defer f2.Close()

	err = DecryptFile(f, f2, "123456")
	if err != nil {
		t.Fatal(err)
	}
}
