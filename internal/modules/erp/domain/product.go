package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalid = errors.New("invalid ERP product")
	ErrNotFound = errors.New("ERP product not found")
	ErrConflict = errors.New("ERP product conflict")
)
var (
	skuPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{0,63}$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
	pricePattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,4})?$`)
)

type Product struct {
	ID uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	SKU string `json:"sku"`
	Name string `json:"name"`
	Description string `json:"description"`
	Kind string `json:"kind"`
	Unit string `json:"unit"`
	UnitPrice string `json:"unit_price"`
	Currency string `json:"currency"`
	Status string `json:"status"`
	Version int `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Input struct {
	SKU string `json:"sku"`
	Name string `json:"name"`
	Description string `json:"description"`
	Kind string `json:"kind"`
	Unit string `json:"unit"`
	UnitPrice string `json:"unit_price"`
	Currency string `json:"currency"`
}

func (input Input) Normalize() (Input, error) {
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Unit = strings.TrimSpace(input.Unit)
	input.Kind = strings.TrimSpace(input.Kind)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.UnitPrice = strings.TrimSpace(input.UnitPrice)
	if !skuPattern.MatchString(input.SKU) || len(input.Name) == 0 || len(input.Name) > 200 ||
		len(input.Description) > 2000 || (input.Kind != "good" && input.Kind != "service") ||
		(input.Unit != "unit" && input.Unit != "hour" && input.Unit != "kg") ||
		!currencyPattern.MatchString(input.Currency) || !pricePattern.MatchString(input.UnitPrice) {
		return Input{}, ErrInvalid
	}
	return input, nil
}
