package integration

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestSignManifestNeverExposesSetupCode(t *testing.T) {
	code := "secure-one-time-application-pairing-secret-12345"
	signature, err := SignManifest(code, []byte(`{"manifest_version":"v1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(signature, "sha256=") || strings.Contains(signature, code) {
		t.Fatal("invalid signed manifest")
	}
	if _, err = SignManifest("short", []byte("{}")); err == nil {
		t.Fatal("weak code signed a manifest")
	}
}
func TestOpenEnrollmentAuthenticatedEnvelope(t *testing.T) {
	code := "secure-one-time-application-pairing-secret-12345"
	credentials := EnrollmentCredentials{
		ApplicationID: "photobooth", ServiceCredential: strings.Repeat("x", 40), APIContractVersion: "v1",
		Database: EnrollmentDatabase{Name: "apexvoid_photobooth", Schema: "photobooth", Role: "apexvoid_photobooth", Password: strings.Repeat("p", 40)},
	}
	clear, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	key := sha256.Sum256([]byte(enrollmentAEADContext + ":" + code))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	encrypted := gcm.Seal(nil, nonce, clear, []byte(enrollmentAEADContext))
	payload, err := json.Marshal(enrollmentEnvelope{Version: "v1", Nonce: base64.RawURLEncoding.EncodeToString(nonce), Ciphertext: base64.RawURLEncoding.EncodeToString(encrypted)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := OpenEnrollment(code, payload)
	if err != nil {
		t.Fatal(err)
	}
	if got != credentials {
		t.Fatalf("unexpected service bootstrap %#v", got)
	}
	if _, err = OpenEnrollment("incorrect-different-setup-secret-123456", payload); err == nil {
		t.Fatal("wrong setup secret decrypted credentials")
	}
	if _, err = OpenEnrollment(code, append(payload, []byte(`{"unexpected":"second"}`)...)); err == nil {
		t.Fatal("extra enrollment payload accepted")
	}
}
