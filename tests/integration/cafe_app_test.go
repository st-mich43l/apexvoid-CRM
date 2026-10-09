//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
)

func TestCafePhotoBoothFirstApplication(t *testing.T) {
	databaseURL, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	platform, err := app.Bootstrap(ctx, config.Config{
		App:       config.AppConfig{Name: "cafe-test", Environment: "test"},
		Server:    config.ServerConfig{Address: ":0"},
		Database:  config.DatabaseConfig{URL: databaseURL, MaxConns: 10, MinConns: 1},
		Auth:      config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128},
		Bootstrap: config.BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"},
		Logging:   config.LoggingConfig{Level: "ERROR"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer platform.Close(context.Background())
	router := chi.NewRouter()
	if err := platform.RegisterRoutes(router); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	guest := newAPIClient(t, server.URL)
	guest.must(http.MethodGet, "/api/v1/cafe/orders", "", nil, nil, http.StatusUnauthorized)
	guest.must(http.MethodPost, "/api/v1/cafe/bookings", "", nil, nil, http.StatusUnauthorized)
	admin := newAPIClient(t, server.URL)
	admin.must(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "admin@localhost", "password": "admin"}, nil, http.StatusOK)
	admin.must(http.MethodPost, "/api/v1/auth/change-password", "", map[string]string{"current_password": "admin", "new_password": "admin-password-123"}, nil, http.StatusOK)
	var setup setupResponse
	admin.must(http.MethodPost, "/api/v1/setup/organization", "", map[string]string{"organization_name": "Coffee and photos", "workspace_name": "Main café", "timezone": "Asia/Ho_Chi_Minh"}, &setup, http.StatusCreated)
	ws := setup.Workspace.ID

	var applications []struct {
		ID string `json:"id"`
	}
	admin.must(http.MethodGet, "/api/v1/framework/applications", ws, nil, &applications, http.StatusOK)
	found := map[string]bool{}
	for _, a := range applications {
		found[a.ID] = true
	}
	for _, name := range []string{"crm", "erp", "cafe"} {
		if !found[name] {
			t.Fatalf("application %s not registered", name)
		}
	}

	newProduct := func(sku, name, kind, price, currency string) string {
		var product struct {
			ID string `json:"id"`
		}
		admin.must(http.MethodPost, "/api/v1/erp/products", ws, map[string]any{
			"sku": sku, "name": name, "description": "", "kind": kind, "unit": "unit", "unit_price": price, "currency": currency,
		}, &product, http.StatusCreated)
		return product.ID
	}
	latte := newProduct("CAFE-LATTE", "Iced Latte", "good", "49000", "VND")
	coffee := newProduct("CAFE-COFFEE", "Black Coffee", "good", "39000", "VND")
	photo := newProduct("PHOTO-20M", "Photo booth 20 minutes", "service", "80000", "VND")
	usd := newProduct("OTHER-USD", "USD drink", "good", "10", "USD")

	var order struct {
		ID        string `json:"id"`
		Total     string `json:"total"`
		Status    string `json:"status"`
		LineCount int    `json:"line_count"`
	}
	admin.must(http.MethodPost, "/api/v1/cafe/orders", ws, map[string]any{
		"note":  "Takeaway",
		"lines": []map[string]any{{"product_id": latte, "quantity": 2}, {"product_id": coffee, "quantity": 1}},
	}, &order, http.StatusCreated)
	if order.Status != "open" || order.LineCount != 2 || order.Total != "137000.0000" {
		t.Fatalf("unexpected exact order %#v", order)
	}
	admin.must(http.MethodPost, "/api/v1/cafe/orders", ws, map[string]any{
		"lines": []map[string]any{{"product_id": latte, "quantity": 1}, {"product_id": usd, "quantity": 1}},
	}, nil, http.StatusBadRequest)
	admin.must(http.MethodPost, "/api/v1/cafe/orders", ws, map[string]any{"lines": []map[string]any{{"product_id": photo, "quantity": 1}}}, nil, http.StatusBadRequest)

	var updated struct {
		Status string `json:"status"`
	}
	admin.must(http.MethodPost, "/api/v1/cafe/orders/"+order.ID+"/complete", ws, nil, &updated, http.StatusOK)
	if updated.Status != "completed" {
		t.Fatalf("order transition incorrect %#v", updated)
	}
	admin.must(http.MethodPost, "/api/v1/cafe/orders/"+order.ID+"/cancel", ws, nil, nil, http.StatusConflict)

	var booth struct {
		ID string `json:"id"`
	}
	admin.must(http.MethodPost, "/api/v1/cafe/booths", ws, map[string]string{"name": "Booth A"}, &booth, http.StatusCreated)
	admin.must(http.MethodPost, "/api/v1/cafe/booths", ws, map[string]string{"name": "Booth A"}, nil, http.StatusConflict)
	start := time.Now().UTC().Truncate(time.Minute).Add(5 * time.Minute)
	end := start.Add(20 * time.Minute)
	request := map[string]any{"booth_id": booth.ID, "package_product_id": photo, "guest_name": "Guest Group", "starts_at": start, "ends_at": end}
	var booking struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Price  string `json:"price"`
	}
	admin.must(http.MethodPost, "/api/v1/cafe/bookings", ws, request, &booking, http.StatusCreated)
	if booking.Status != "reserved" || booking.Price != "80000.0000" {
		t.Fatalf("booking snapshot incorrect %#v", booking)
	}
	admin.must(http.MethodPost, "/api/v1/cafe/bookings", ws, request, nil, http.StatusConflict)
	adjacent := map[string]any{"booth_id": booth.ID, "package_product_id": photo, "guest_name": "Next Group", "starts_at": end, "ends_at": end.Add(20 * time.Minute)}
	admin.must(http.MethodPost, "/api/v1/cafe/bookings", ws, adjacent, nil, http.StatusCreated)
	admin.must(http.MethodPost, "/api/v1/cafe/bookings/"+booking.ID+"/complete", ws, nil, nil, http.StatusConflict)
	admin.must(http.MethodPost, "/api/v1/cafe/bookings/"+booking.ID+"/check-in", ws, nil, nil, http.StatusOK)
	admin.must(http.MethodPost, "/api/v1/cafe/bookings/"+booking.ID+"/complete", ws, nil, nil, http.StatusOK)
	admin.must(http.MethodPost, "/api/v1/cafe/bookings/"+booking.ID+"/cancel", ws, nil, nil, http.StatusConflict)

	var second struct {
		ID string `json:"id"`
	}
	admin.must(http.MethodPost, "/api/v1/workspaces", ws, map[string]any{"name": "Second café", "timezone": "Asia/Ho_Chi_Minh"}, &second, http.StatusCreated)
	var orders []struct {
		ID string `json:"id"`
	}
	admin.must(http.MethodGet, "/api/v1/cafe/orders", second.ID, nil, &orders, http.StatusOK)
	if len(orders) != 0 {
		t.Fatal("café orders leaked between workspaces")
	}
	var bookings []struct {
		ID string `json:"id"`
	}
	admin.must(http.MethodGet, "/api/v1/cafe/bookings", second.ID, nil, &bookings, http.StatusOK)
	if len(bookings) != 0 {
		t.Fatal("bookings leaked between workspaces")
	}
	admin.must(http.MethodPost, "/api/v1/cafe/orders", second.ID, map[string]any{"lines": []map[string]any{{"product_id": latte, "quantity": 1}}}, nil, http.StatusBadRequest)
	admin.must(http.MethodPost, "/api/v1/cafe/bookings", second.ID, request, nil, http.StatusBadRequest)
	admin.must(http.MethodGet, "/api/v1/cafe/orders", uuid.NewString(), nil, nil, http.StatusForbidden)
}
