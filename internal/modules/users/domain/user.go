package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusActive   Status = "active"
	StatusInactive Status = "inactive"
	StatusLocked   Status = "locked"
)

type User struct {
	ID           uuid.UUID
	Email        string
	Username     string
	DisplayName  string
	PasswordHash string
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  *time.Time
}

func NormalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

func (u User) Validate() error {
	u.Email = NormalizeEmail(u.Email)
	if u.Email == "" || !strings.Contains(u.Email, "@") {
		return fmt.Errorf("email is invalid")
	}
	if strings.TrimSpace(u.DisplayName) == "" {
		return fmt.Errorf("display name is required")
	}
	if u.Status != StatusActive && u.Status != StatusInactive && u.Status != StatusLocked {
		return fmt.Errorf("invalid user status %q", u.Status)
	}
	return nil
}

func (u User) IsUsable() bool { return u.Status == StatusActive }
