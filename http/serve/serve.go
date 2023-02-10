// Package httpserve contains common functions used by HTTP handlers.
package serve

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

func HTMLTemplate(w http.ResponseWriter, r *http.Request, tpl *template.Template, data any) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		Error(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func JSON(w http.ResponseWriter, _ *http.Request, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %s %s error: %s", r.Method, r.URL, err)
	_, _ = io.Copy(w, strings.NewReader(err.Error()))
}
