// Package handlers contains common functions used by HTTP handlers.
package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"
)

func ServeTemplate(w http.ResponseWriter, r *http.Request, tpl *template.Template, data any) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		ServeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func ServeJSON(w http.ResponseWriter, _ *http.Request, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

func ServeError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %s %s error: %s", r.Method, r.URL, err)
	_, _ = io.Copy(w, strings.NewReader(err.Error()))
}
