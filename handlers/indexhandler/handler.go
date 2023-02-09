// Package indexhandler serves index page HTML.
package indexhandler

import (
	_ "embed"
	"html/template"
	"net/http"

	"github.com/gofu/gomon/handlers"
)

// Handler serves static index page.
type Handler struct {
	// Data to pass to template.
	Data
}

var (
	//go:embed tpl.gohtml
	tplData string
	tpl     = template.Must(template.New("").Parse(tplData))
)

// New returns Handler that serves static data.
func New(data Data) Handler {
	return Handler{Data: data}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlers.ServeTemplate(w, r, tpl, h.Data)
}
