package application

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIdentityAssertionIsAudienceBoundAndExpires(t *testing.T) {
	issuer, err := NewAssertionIssuer("test-assertion-secret-with-enough-entropy")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	issuer.now = func() time.Time { return now }
	assertion, err := issuer.Issue("reports", uuid.New(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Verify(assertion, "reports")
	if err != nil {
		t.Fatalf("valid assertion failed: %#v %v", claims, err)
	}
	if _, err := issuer.Verify(assertion, "other"); err == nil {
		t.Fatal("assertion accepted by a different application audience")
	}
	now = now.Add(2 * time.Minute)
	if _, err := issuer.Verify(assertion, "reports"); err == nil {
		t.Fatal("expired assertion was accepted")
	}
}
