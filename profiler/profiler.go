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
	// Root of the calling file (project, GOROOT, GOPATH).
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

// CallStack contains a running goroutine's caller stack info.
type CallStack struct {
	// FileLine contains caller's position in file/line.
	FileLine
	// Caller is true for every first goroutine in stack.
	// The only exception is the main goroutine.
	Caller bool `json:"caller"`
	// Package name, as seen by the Go source code.
	Package string `json:"package"`
	// Method name, eg. package.(*Server).ServeHTTP.
	Method string `json:"method"`
	Args   string `json:"args,omitempty"`
	Extra  string `json:"extra,omitempty"`
	// Highlight is optionally present.
	Highlight
}

// Goroutine and call stack information.
type Goroutine struct {
	// ID of this goroutine.
	ID int `json:"id"`
	// Op that's blocking this goroutine.
	Op string `json:"op"`
	// Duration that the goroutine has been blocked for.
	Duration time.Duration `json:"duration,omitempty"`
	// CallStack information.
	CallStack []CallStack `json:"callStack,omitempty"`
}

// Profiler provides profiling information.
type Profiler interface {
	// Source identifier, eg. full /debug/pprof URL.
	Source() string
	// Goroutines that are currently running, without Highlight data.
	Goroutines() ([]Goroutine, error)
}
