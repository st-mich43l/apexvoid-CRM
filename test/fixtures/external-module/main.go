// External-module is a deliberately tiny Docker fixture used to exercise the
// ApexVoid external integration boundary. It is not a product application.
package main

import (
	"net/http"
	"os"
)

func main() {
	address := os.Getenv("FIXTURE_ADDR")
	if address == "" {
		address = ":8090"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("external module fixture"))
	})
	_ = http.ListenAndServe(address, mux)
}
