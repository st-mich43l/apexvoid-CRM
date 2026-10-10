package application

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	manifestSignatureHeader = "X-ApexVoid-Manifest-Signature"
	enrollmentContext       = "apexvoid-enrollment-v1"
	manifestContext         = "apexvoid-manifest-v1\n"
	minEnrollmentCodeLength = 32
)

// Fetch the bootstrap contract without exposing the enrollment secret to the
// network. The service signs the exact response bytes using the setup code.
func authenticatedManifest(ctx context.Context, client *http.Client, endpoint, code string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: manifest endpoint returned status %d", ErrInvalidManifest, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxManifestBytes+1))
	if err != nil || len(body) > maxManifestBytes {
		return nil, fmt.Errorf("%w: manifest exceeds size limit or could not be read", ErrInvalidManifest)
	}
	sig := strings.TrimPrefix(response.Header.Get(manifestSignatureHeader), "sha256=")
	decoded, err := hex.DecodeString(sig)
	if err != nil || len(decoded) != sha256.Size {
		return nil, fmt.Errorf("%w: missing or invalid manifest authenticity proof", ErrInvalidManifest)
	}
	mac := hmac.New(sha256.New, []byte(code))
	_, _ = mac.Write([]byte(manifestContext))
	_, _ = mac.Write(body)
	if !hmac.Equal(decoded, mac.Sum(nil)) {
		return nil, fmt.Errorf("%w: manifest signature does not match enrollment secret", ErrInvalidManifest)
	}
	return body, nil
}

type sealedEnrollment struct {
	Version    string `json:"version"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// A high-entropy enrollment code is the out-of-band shared secret. The code is
// never transmitted as an HTTP header. AES-GCM protects credentials during the
// authenticated internal HTTP enrollment exchange.
func sealEnrollment(code string, clear []byte) ([]byte, error) {
	if len(code) < minEnrollmentCodeLength {
		return nil, errors.New("enrollment secret must contain at least 32 characters")
	}
	key := sha256.Sum256([]byte(enrollmentContext + ":" + code))
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
	ciphertext := gcm.Seal(nil, nonce, clear, []byte(enrollmentContext))
	return json.Marshal(sealedEnrollment{
		Version: "v1", Nonce: base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(ciphertext),
	})
}

// Used by the service-side reference fixture and by external applications
// implementing the enrollment protocol. Rejects altered or replayed messages
// at the service layer using its one-time code state.
func unsealEnrollment(code string, raw []byte) ([]byte, error) {
	var payload sealedEnrollment
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || payload.Version != "v1" {
		return nil, errors.New("invalid encrypted enrollment envelope")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(payload.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return nil, err
	}
	key := sha256.Sum256([]byte(enrollmentContext + ":" + code))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid enrollment nonce")
	}
	return gcm.Open(nil, nonce, ciphertext, []byte(enrollmentContext))
}
