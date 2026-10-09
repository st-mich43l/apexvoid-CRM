package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBookingValidation(t *testing.T) {
	now := time.Now().UTC()
	input := BookingInput{BoothID: uuid.New(), PackageProductID: uuid.New(), GuestName: " Customer ", StartsAt: now.Add(time.Hour), EndsAt: now.Add(90 * time.Minute)}
	got, err := input.Validate(now)
	if err != nil || got.GuestName != "Customer" {
		t.Fatalf("unexpected booking: %#v %v", got, err)
	}
	input.EndsAt = input.StartsAt
	if _, err := input.Validate(now); !errors.Is(err, ErrInvalid) {
		t.Fatal("zero-duration booking accepted")
	}
	input.EndsAt = input.StartsAt.Add(3 * time.Hour)
	if _, err := input.Validate(now); !errors.Is(err, ErrInvalid) {
		t.Fatal("overlong booking accepted")
	}
}
func TestOrderValidation(t *testing.T) {
	product := uuid.New()
	if _, err := (OrderInput{Lines: []OrderLineInput{{ProductID: product, Quantity: 1}, {ProductID: product, Quantity: 1}}}).Validate(); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate product accepted")
	}
	if _, err := (OrderInput{Lines: []OrderLineInput{{ProductID: product, Quantity: 1}}}).Validate(); err != nil {
		t.Fatal(err)
	}
}
