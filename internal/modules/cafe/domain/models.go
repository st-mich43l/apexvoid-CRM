package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalid = errors.New("invalid cafe request")
	ErrNotFound = errors.New("cafe record not found")
	ErrConflict = errors.New("cafe conflict")
)

type Booth struct {
	ID uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name string `json:"name"`
	Active bool `json:"active"`
	CreatedAt time.Time `json:"created_at"`
}

type Booking struct {
	ID uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	BoothID uuid.UUID `json:"booth_id"`
	BoothName string `json:"booth_name"`
	PackageProductID uuid.UUID `json:"package_product_id"`
	GuestName string `json:"guest_name"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt time.Time `json:"ends_at"`
	Status string `json:"status"`
	Price string `json:"price"`
	Currency string `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
}

type BookingInput struct {
	BoothID uuid.UUID `json:"booth_id"`
	PackageProductID uuid.UUID `json:"package_product_id"`
	GuestName string `json:"guest_name"`
	StartsAt time.Time `json:"starts_at"`
	EndsAt time.Time `json:"ends_at"`
}

func (in BookingInput) Validate(now time.Time) (BookingInput,error) {
	in.GuestName = strings.TrimSpace(in.GuestName)
	duration := in.EndsAt.Sub(in.StartsAt)
	if in.BoothID == uuid.Nil || in.PackageProductID == uuid.Nil || len(in.GuestName) == 0 ||
		len(in.GuestName) > 120 || in.StartsAt.IsZero() || in.StartsAt.Before(now.Add(-5*time.Minute)) ||
		in.StartsAt.After(now.AddDate(0,3,0)) || duration < 10*time.Minute || duration > 2*time.Hour {
		return BookingInput{}, ErrInvalid
	}
	return in,nil
}

type OrderLineInput struct {
	ProductID uuid.UUID `json:"product_id"`
	Quantity int `json:"quantity"`
}
type OrderInput struct {
	Note string `json:"note"`
	Lines []OrderLineInput `json:"lines"`
}
func (in OrderInput) Validate() (OrderInput,error) {
	in.Note = strings.TrimSpace(in.Note)
	if len(in.Note) > 300 || len(in.Lines) == 0 || len(in.Lines) > 20 { return OrderInput{},ErrInvalid }
	seen := map[uuid.UUID]bool{}
	for _,line := range in.Lines {
		if line.ProductID == uuid.Nil || line.Quantity < 1 || line.Quantity > 99 || seen[line.ProductID] {return OrderInput{},ErrInvalid}
		seen[line.ProductID] = true
	}
	return in,nil
}
type Order struct {
	ID uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Status string `json:"status"`
	Total string `json:"total"`
	Currency string `json:"currency"`
	Note string `json:"note"`
	LineCount int `json:"line_count"`
	CreatedAt time.Time `json:"created_at"`
}
