package gomon

import (
	"time"
)

type RootType string

const (
	RootTypeProject RootType = "PROJECT"
	RootTypeGoRoot  RootType = "GOROOT"
	RootTypeGoPath  RootType = "GOPATH"
	RootTypeCGo     RootType = "CGO"
)

func (t RootType) FromEnv(env EnvConfig) string {
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
	Line      int      `json:"line"`
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
