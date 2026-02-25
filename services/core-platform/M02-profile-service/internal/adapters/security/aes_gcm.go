package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type AESGCMEncryption struct {
	seed string
}

func NewAESGCMEncryption(seed string) *AESGCMEncryption {
	return &AESGCMEncryption{seed: seed}
}

func (e *AESGCMEncryption) Encrypt(userID string, value string) ([]byte, error) {
	key := deriveKey(userID, e.seed)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := key[:gcm.NonceSize()]
	cipherText := gcm.Seal(nil, nonce, []byte(value), nil)
	return []byte(base64.StdEncoding.EncodeToString(cipherText)), nil
}

func (e *AESGCMEncryption) Decrypt(userID string, payload []byte) (string, error) {
	key := deriveKey(userID, e.seed)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := key[:gcm.NonceSize()]
	decoded, err := base64.StdEncoding.DecodeString(string(payload))
	if err != nil {
		return "", fmt.Errorf("decode payload: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, decoded, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deriveKey(userID, seed string) []byte {
	sum := sha256.Sum256([]byte(seed + ":" + userID))
	return sum[:]
}
