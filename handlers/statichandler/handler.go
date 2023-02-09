package statichandler

import (
	_ "embed"
	"io"
	"net/http"
	"strings"
)

type Handler struct{}

//go:embed favicon.png
var favicon string

func (Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/favicon.ico":
		w.Header().Set("Content-Type", "image/png")
		_, _ = io.Copy(w, strings.NewReader(favicon))
	default:
		http.NotFound(w, r)
	}
}
