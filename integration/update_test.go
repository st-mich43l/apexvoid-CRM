package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestSignedUpdateManifestBindsCredentialChallengeAndBytes(t *testing.T) {
	credential := strings.Repeat("c", 40)
	challenge := strings.Repeat("n", 40)
	body := []byte("{\"manifest_version\":\"v1\"}")
	signature, err := SignUpdateManifest(credential, challenge, body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(signature, "sha256=") {
		t.Fatalf("unexpected signature %q", signature)
	}
	derived := sha256.Sum256([]byte(credential))
	mac := hmac.New(sha256.New, derived[:])
	_, _ = mac.Write([]byte("apexvoid-update-manifest-v1\n" + challenge + "\n"))
	_, _ = mac.Write(body)
	if signature != "sha256="+hex.EncodeToString(mac.Sum(nil)) {
		t.Fatal("signature protocol mismatch")
	}
	changed, err := SignUpdateManifest(credential, challenge, append(body, ' '))
	if err != nil || changed == signature {
		t.Fatal("modified manifest was not rejected")
	}
	changed, err = SignUpdateManifest(credential, strings.Repeat("x", 40), body)
	if err != nil || changed == signature {
		t.Fatal("different challenge reused signature")
	}
	if _, err := SignUpdateManifest("weak", challenge, body); err == nil {
		t.Fatal("accepted short service secret")
	}
}
