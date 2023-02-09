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

// EnvRoot returns the corresponding root path from env.
func EnvRoot(env EnvConfig, t RootType) string {
	switch t {
	case RootTypeProject:
		return env.Root
	case RootTypeGoRoot:
		return env.GoRoot
	case RootTypeGoPath:
		return env.GoPath
	default:
		return ""
	}
}

type StackElem struct {
	Caller    bool     `json:"caller"`
	Package   string   `json:"package"`
	Method    string   `json:"method"`
	Args      string   `json:"args,omitempty"`
	File      string   `json:"file"`
	ShortFile string   `json:"shortFile"`
	Root      RootType `json:"root,omitempty"`
	Extra     string   `json:"extra,omitempty"`
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
