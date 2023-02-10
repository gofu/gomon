// Package statichandler serves static files.
package statichandler

import (
	_ "embed"
	"io"
	"net/http"
	"strings"
)

// FaviconURL is the default favicon URL requested by browsers.
const FaviconURL = "/favicon.ico"

// Handler serves static files.
type Handler struct{}

//go:embed favicon.png
var favicon string

func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case FaviconURL:
		w.Header().Set("Content-Type", "image/png")
		_, _ = io.Copy(w, strings.NewReader(favicon))
	default:
		http.NotFound(w, r)
	}
}
