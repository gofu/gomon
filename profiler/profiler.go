// Package profiler contains goroutine profiling information.
package profiler

import (
	"sort"
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

type LineInfo interface {
	RootType() RootType
	FilePath() string
	LineNumber() int
}

type FileLine struct {
	// Root of the calling file (project, GOROOT, GOPATH).
	Root RootType `json:"root"`
	// File path, relative to Root.
	File string `json:"file"`
	// Line number, starting from 1.
	Line int `json:"line"`
}

func (f FileLine) RootType() RootType { return f.Root }
func (f FileLine) FilePath() string   { return f.File }
func (f FileLine) LineNumber() int    { return f.Line }

func CompareLineInfo(f, stack LineInfo) int {
	if f.RootType() != stack.RootType() {
		if f.RootType() < stack.RootType() {
			return -1
		}
		return 1
	}
	if f.FilePath() != stack.FilePath() {
		if f.FilePath() < stack.FilePath() {
			return -1
		}
		return 1
	}
	if f.LineNumber() != stack.LineNumber() {
		if f.LineNumber() < stack.LineNumber() {
			return -1
		}
		return 1
	}
	return 0
}

// CallInfo provides goroutine call stack information.
type CallInfo struct {
	// Called is true for every first goroutine in stack.
	// The only exception is the main goroutine.
	Called bool `json:"called"`
	// Package name, as seen by the Go source code.
	Package string `json:"package"`
	// Method name, eg. (*Server).ServeHTTP.
	Method string `json:"method"`
	Args   string `json:"args,omitempty"`
	Extra  string `json:"extra,omitempty"`
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
	// CallInfo provides call-specific information.
	CallInfo
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

// Main is true if this is the main goroutine.
func (g Goroutine) Main() bool {
	return len(g.CallStack) > 0 && !g.CallStack[0].Called
}

// CompareStack returns 0 if s1 and s2 point to identical
// files/lines, -1 if s1<s2, or 1 if s1>s2.
func CompareStack[T1, T2 LineInfo](s1 []T1, s2 []T2) int {
	l1, l2 := len(s1), len(s2)
	for i := 0; i < l1 && i < l2; i++ {
		comp := CompareLineInfo(s1[i], s2[i])
		if comp == 0 {
			continue
		}
		return comp
	}
	if l1 < l2 {
		return -1
	}
	if l1 > l2 {
		return 1
	}
	return 0
}

// Sort goroutines by descending duration, then ascending
// goroutine ID. The main goroutine is always first.
func Sort(gs []Goroutine) {
	sort.Slice(gs, func(i, j int) bool {
		gi, gj := gs[i], gs[j]
		if gi.Main() {
			return true
		}
		if gj.Main() {
			return false
		}
		if gi.Duration == gj.Duration {
			return gi.ID < gj.ID
		}
		return gi.Duration > gj.Duration
	})
}

// Profiler provides profiling information.
type Profiler interface {
	// Source identifier, eg. full /debug/pprof URL.
	Source() string
	// Goroutines that are currently running, without Highlight data.
	Goroutines() ([]Goroutine, error)
}
