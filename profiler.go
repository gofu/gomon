package gomon

import (
	"time"
)

// RootType distinguishes Go source code roots.
type RootType string

const (
	// RootTypeProject represents project root.
	RootTypeProject RootType = "PROJECT"
	// RootTypeGoRoot represents GOROOT.
	RootTypeGoRoot RootType = "GOROOT"
	// RootTypeGoPath represents GOPATH.
	RootTypeGoPath RootType = "GOPATH"
	// RootTypeCGo represents linked CGO.
	RootTypeCGo RootType = "CGO"
)

// StackElem contains a running goroutine's caller stack info.
type StackElem struct {
	Caller    bool   `json:"caller"`
	Package   string `json:"package"`
	Method    string `json:"method"`
	Args      string `json:"args,omitempty"`
	ShortFile string `json:"shortFile"`
	Extra     string `json:"extra,omitempty"`
	FileInfo
	Highlight
}

type Goroutine struct {
	ID       int           `json:"id"`
	Op       string        `json:"op"`
	Duration time.Duration `json:"duration,omitempty"`
	Stack    []StackElem   `json:"stack,omitempty"`
}

type Profiler interface {
	Goroutines() ([]Goroutine, error)
}
