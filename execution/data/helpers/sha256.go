package helpers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
)

func SHA256(input string) string {
	return SHA256Bytes([]byte(input))
}

func SHA256Bytes(input []byte) string {
	hash := sha256.Sum256(input)
	return hex.EncodeToString(hash[:])
}

func SHA256Reader(reader io.Reader) (string, error) {
	hash := sha256.New()
	_, err := io.Copy(hash, reader)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
