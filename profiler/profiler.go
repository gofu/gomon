// Package profiler contains goroutine profiling information.
package profiler

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

type FileLine struct {
	// Root of the calling file.
	Root RootType `json:"root"`
	// File path, relative to Root.
	File string `json:"file"`
	// Line number, starting from 1.
	Line int `json:"line"`
}

// Highlight HTML contains a source code segment.
type Highlight struct {
	// Prefix HTML contains the current line, and WrapSize lines preceding it.
	Prefix string `json:"prefix,omitempty"`
	// Suffix HTML contains WrapSize lines succeeding the current line.
	Suffix string `json:"suffix,omitempty"`
}

// StackElem contains a running goroutine's caller stack info.
type StackElem struct {
	Caller  bool   `json:"caller"`
	Package string `json:"package"`
	Method  string `json:"method"`
	Args    string `json:"args,omitempty"`
	Extra   string `json:"extra,omitempty"`
	FileLine
	Highlight
}

type Goroutine struct {
	ID       int           `json:"id"`
	Op       string        `json:"op"`
	Duration time.Duration `json:"duration,omitempty"`
	Stack    []StackElem   `json:"stack,omitempty"`
}

// Profiler provides profiling information.
type Profiler interface {
	// Goroutines that are currently running, without Highlight data.
	Goroutines() ([]Goroutine, error)
}
