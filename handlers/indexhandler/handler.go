package indexhandler

import (
	_ "embed"
	"html/template"
	"net/http"

	"github.com/gofu/gomon/handlers"
)

type Handler struct {
	Data
}

var (
	//go:embed tpl.gohtml
	tplData string
	tpl     = template.Must(template.New("").Parse(tplData))
)

func New(data Data) Handler {
	return Handler{Data: data}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlers.ServeTemplate(w, r, tpl, h.Data)
}
