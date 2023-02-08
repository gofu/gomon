package gomon

import (
	"html/template"
	"time"
)

type StackElem struct {
	Caller    bool          `json:"caller"`
	Package   string        `json:"package"`
	Method    string        `json:"method"`
	Args      string        `json:"args,omitempty"`
	File      string        `json:"file"`
	ShortFile string        `json:"shortFile"`
	Line      int           `json:"line"`
	Root      string        `json:"root,omitempty"`
	Extra     string        `json:"extra,omitempty"`
	Prefix    template.HTML `json:"prefix,omitempty"`
	Suffix    template.HTML `json:"suffix,omitempty"`
}

type Goroutine struct {
	ID       int           `json:"id"`
	Op       string        `json:"op"`
	Duration time.Duration `json:"duration,omitempty"`
	Stack    []StackElem   `json:"stack,omitempty"`
}

type GoroutineProvider interface {
	GetRunning() ([]Goroutine, error)
}
