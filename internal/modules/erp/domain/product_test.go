package domain

import "testing"

func TestNormalizeProduct(t *testing.T) {
	input := Input{SKU: " item-01 ", Name: " Office chair ", Kind: "good", Unit: "unit", Currency: "usd", UnitPrice: "12.3456"}
	got, err := input.Normalize()
	if err != nil || got.SKU != "ITEM-01" || got.Name != "Office chair" || got.Currency != "USD" {
		t.Fatalf("unexpected normalized product: %#v %v", got, err)
	}
	for _, price := range []string{"-1", "1.23456", "NaN", "1e5", "99999999999999999", ""} {
		input.UnitPrice = price
		if _, err := input.Normalize(); err == nil {
			t.Fatalf("invalid price %q accepted", price)
		}
	}
}
