package integration

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// UpdateChallengeHeader is sent by Enterprise when requesting a fresh
// manifest from a previously enrolled application. Never send the permanent
// service credential over this request.
const UpdateChallengeHeader = "X-ApexVoid-Update-Challenge"
const UpdateSignatureHeader = "X-ApexVoid-Update-Signature"
const updateManifestContext = "apexvoid-update-manifest-v1\n"

// SignUpdateManifest authenticates the exact manifest bytes and the fresh
// challenge with a key derived from the currently active service credential.
// The platform stores only SHA256(credential), not the cleartext credential.
// The challenge must be an unpredictable, single-request random value.
func SignUpdateManifest(serviceCredential, challenge string, manifest []byte) (string, error) {
	if len(serviceCredential) < 32 || len(challenge) < 32 || len(manifest) == 0 {
		return "", errors.New("update manifest signature requires a service credential, challenge and body")
	}
	derived := sha256.Sum256([]byte(serviceCredential))
	mac := hmac.New(sha256.New, derived[:])
	_, _ = mac.Write([]byte(updateManifestContext + challenge + "\n"))
	_, _ = mac.Write(manifest)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil)), nil
}
