package application

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
)

func protectProvisioningPassword(secret, password string) ([]byte, error) {
	if len(secret) < 32 {
		return nil, errors.New("database provisioning encryption key is required")
	}
	key := sha256.Sum256([]byte("apexvoid-provisioning-password-v1:" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, []byte(password), []byte("apexvoid-provisioned-database-v1")), nil
}

func recoverProvisioningPassword(secret string, ciphertext []byte) (string, error) {
	if len(secret) < 32 {
		return "", errors.New("database provisioning encryption key is required")
	}
	key := sha256.Sum256([]byte("apexvoid-provisioning-password-v1:" + secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < gcm.NonceSize() {
		return "", errors.New("corrupt database credential")
	}
	clear, err := gcm.Open(nil, ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():], []byte("apexvoid-provisioned-database-v1"))
	if err != nil {
		return "", errors.New("database credential cannot be decrypted with current provisioning key")
	}
	return string(clear), nil
}
