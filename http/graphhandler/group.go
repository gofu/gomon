package graphhandler

import (
	"sort"

	"github.com/gofu/gomon/profiler"
	"golang.org/x/exp/slices"
)

type Group struct {
	Stack []profiler.LineInfo   `json:"stack"`
	Calls [][]profiler.CallInfo `json:"calls"`
}

func GroupGoroutines(gs []profiler.Goroutine) []Group {
	gs = slices.Clone(gs)
	sort.Slice(gs, func(i, j int) bool {
		return profiler.CompareStack(gs[i].CallStack, gs[j].CallStack) < 0
	})
	var grouped []Group
	for _, g := range gs {
		if l := len(grouped); l > 0 && profiler.CompareStack(grouped[l-1].Stack, g.CallStack) == 0 {
			callInfo := make([]profiler.CallInfo, len(g.CallStack))
			for i, c := range g.CallStack {
				callInfo[i] = c.CallInfo
			}
			grouped[l-1].Calls = append(grouped[l-1].Calls, callInfo)
			continue
		}
		stack := make([]profiler.LineInfo, len(g.CallStack))
		callInfo := make([]profiler.CallInfo, len(g.CallStack))
		for i, c := range g.CallStack {
			stack[i] = c.FileLine
			callInfo[i] = c.CallInfo
		}
		grouped = append(grouped, Group{Stack: stack, Calls: [][]profiler.CallInfo{callInfo}})
	}
	return grouped
}
