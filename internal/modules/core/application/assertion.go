package application

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidAssertion = errors.New("invalid external identity assertion")

// IdentityAssertion is encrypted and authenticated with AES-GCM. It is an
// opaque envelope to the external service: the contained session access token
// is never exposed to browser JavaScript or forwarded as an HTTP credential.
type IdentityAssertion struct {
	ApplicationID string    `json:"aud"`
	UserID        uuid.UUID `json:"sub"`
	SessionID     uuid.UUID `json:"sid"`
	WorkspaceID   uuid.UUID `json:"wid"`
	AccessToken   string    `json:"token"`
	IssuedAt      time.Time `json:"iat"`
	ExpiresAt     time.Time `json:"exp"`
	ID            uuid.UUID `json:"jti"`
}

type AssertionIssuer struct {
	aead cipher.AEAD
	now  func() time.Time
}

func NewAssertionIssuer(secret string) (*AssertionIssuer, error) {
	if secret == "" {
		// External registrations are disabled without an allowlisted service host,
		// but retain a non-predictable process-local key for that safe default.
		// Configuration validation requires an explicit secret before any service
		// host can be enabled.
		randomSecret := make([]byte, 32)
		if _, err := rand.Read(randomSecret); err != nil {
			return nil, err
		}
		secret = base64.RawStdEncoding.EncodeToString(randomSecret)
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &AssertionIssuer{aead: aead, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (i *AssertionIssuer) Issue(applicationID string, userID, sessionID, workspaceID uuid.UUID, accessToken string) (string, error) {
	if i == nil || applicationID == "" || userID == uuid.Nil || sessionID == uuid.Nil || workspaceID == uuid.Nil || accessToken == "" {
		return "", ErrInvalidAssertion
	}
	now := i.now()
	claims := IdentityAssertion{ApplicationID: applicationID, UserID: userID, SessionID: sessionID, WorkspaceID: workspaceID, AccessToken: accessToken, IssuedAt: now, ExpiresAt: now.Add(time.Minute), ID: uuid.New()}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, i.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := i.aead.Seal(nil, nonce, payload, []byte(applicationID))
	return base64.RawURLEncoding.EncodeToString(append(nonce, ciphertext...)), nil
}

func (i *AssertionIssuer) Verify(raw, applicationID string) (IdentityAssertion, error) {
	if i == nil || raw == "" || applicationID == "" {
		return IdentityAssertion{}, ErrInvalidAssertion
	}
	encoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(encoded) <= i.aead.NonceSize() {
		return IdentityAssertion{}, ErrInvalidAssertion
	}
	payload, err := i.aead.Open(nil, encoded[:i.aead.NonceSize()], encoded[i.aead.NonceSize():], []byte(applicationID))
	if err != nil {
		return IdentityAssertion{}, ErrInvalidAssertion
	}
	var claims IdentityAssertion
	if err := json.Unmarshal(payload, &claims); err != nil || claims.ApplicationID != applicationID || claims.UserID == uuid.Nil || claims.SessionID == uuid.Nil || claims.WorkspaceID == uuid.Nil || claims.AccessToken == "" || claims.ID == uuid.Nil {
		return IdentityAssertion{}, ErrInvalidAssertion
	}
	now := i.now()
	if claims.IssuedAt.After(now.Add(5*time.Second)) || !claims.ExpiresAt.After(now) || claims.ExpiresAt.After(claims.IssuedAt.Add(2*time.Minute)) {
		return IdentityAssertion{}, fmt.Errorf("%w: expired or malformed timing", ErrInvalidAssertion)
	}
	return claims, nil
}
