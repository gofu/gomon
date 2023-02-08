package gomon

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Parser struct {
	EnvConfig
}

func (p Parser) parseGoroutines(r io.Reader) ([]Goroutine, error) {
	s := bufio.NewScanner(r)
	s.Split(scanDoubleLines)
	var gs []Goroutine
	for s.Scan() {
		goroutine, err := p.parseGoroutine(s.Text())
		if err != nil {
			return gs, err
		}
		gs = append(gs, goroutine)
	}
	return gs, s.Err()
}

var (
	// goroutine 11847977 [chan receive, 5 minutes]:
	goroutineIDRegexp = regexp.MustCompile(`^goroutine (\d+) \[([^,\]]+)(?:, (\d+) minutes)?]:$`)
	// github.com/streadway/amqp.(*consumers).buffer(0xc01f9c6120, 0xc04b4369c0, 0xc04b436960)
	goroutineStackRegexp = regexp.MustCompile(`^(.*)\((.*?)\)$`)
	//     /home/ubuntu/workspace/pipeline_ci_cloner_worker/build/go/cloner/cloner.go:1175 +0x7eb
	//     /home/ubuntu/.gopath/pkg/mod/github.com/streadway/amqp@v1.0.0/consumers.go:61 +0x108
	goroutineFileRegexp = regexp.MustCompile(`^\t(.*):(\d+)(?: (.*?))?$`)
)

func (p Parser) parseGoroutine(data string) (Goroutine, error) {
	s := bufio.NewScanner(strings.NewReader(data))
	var gr Goroutine
	var err error
	for s.Scan() {
		if len(s.Text()) == 0 {
			continue
		}
		if gr.ID == 0 {
			matches := goroutineIDRegexp.FindStringSubmatch(s.Text())
			if len(matches) != 4 {
				return gr, fmt.Errorf("did not get expected goroutine ID: %s", s.Text())
			}
			gr.ID, err = strconv.Atoi(matches[1])
			if err != nil {
				return gr, fmt.Errorf("invalid goroutine ID: %s", matches[1])
			}
			gr.Op = matches[2]
			if len(matches[3]) != 0 {
				durationMinutes, err := strconv.Atoi(matches[3])
				if err != nil {
					return gr, fmt.Errorf("invalid goroutine block duration: %s", matches[3])
				}
				gr.Duration = time.Duration(durationMinutes) * time.Minute
			}
			continue
		}
		var caller bool
		var matches []string
		const createdByPrefix = "created by "
		if strings.HasPrefix(s.Text(), createdByPrefix) {
			caller = true
			matches = []string{"", strings.TrimPrefix(s.Text(), createdByPrefix), ""}
		} else {
			matches = goroutineStackRegexp.FindStringSubmatch(s.Text())
		}
		if len(matches) != 3 {
			return gr, fmt.Errorf("invalid goroutine stack: %s", s.Text())
		}
		pkg, method := "", matches[1]
		if cut := strings.LastIndexByte(method, '/'); cut != -1 {
			pkg, method = method[:cut], method[cut:]
		}
		if cut := strings.IndexByte(method, '.'); cut != -1 {
			pkg, method = pkg+method[:cut], method[cut+1:]
		}
		stack := StackElem{
			Caller:  caller,
			Package: pkg,
			Method:  method,
			Args:    matches[2],
		}
		if !s.Scan() {
			return gr, fmt.Errorf("could not advance scanner")
		}
		matches = goroutineFileRegexp.FindStringSubmatch(s.Text())
		if len(matches) != 4 {
			return gr, fmt.Errorf("invalid goroutine file: %s", s.Text())
		}
		if strings.HasPrefix(matches[1], p.Root) {
			stack.File = strings.TrimLeft(strings.TrimPrefix(matches[1], p.Root), "/")
			stack.ShortFile = stack.File
			stack.Root = "PROJECT"
		} else if strings.HasPrefix(matches[1], p.GoRoot) {
			stack.File = strings.TrimLeft(strings.TrimPrefix(matches[1], p.GoRoot), "/")
			stack.ShortFile = strings.TrimPrefix(stack.File, "src/")
			stack.Root = "GOROOT"
		} else if strings.HasPrefix(matches[1], p.GoPath) {
			stack.File = strings.TrimLeft(strings.TrimPrefix(matches[1], p.GoPath), "/")
			stack.ShortFile = strings.TrimPrefix(stack.File, "pkg/")
			stack.ShortFile = strings.TrimPrefix(stack.ShortFile, "mod/")
			stack.Root = "GOPATH"
		} else if strings.HasPrefix(matches[1], "_cgo_") {
			stack.File = matches[1]
			stack.Root = "CGO"
		} else {
			return gr, fmt.Errorf("unknown component path: %q", matches[1])
		}
		stack.Line, err = strconv.Atoi(matches[2])
		if err != nil {
			return gr, fmt.Errorf("invalid goroutine line: %s", matches[2])
		}
		stack.Extra = matches[3]
		gr.Stack = append([]StackElem{stack}, gr.Stack...)
	}
	if err = s.Err(); err != nil {
		return gr, err
	}
	if gr.ID == 0 || len(gr.Stack) == 0 {
		return gr, fmt.Errorf("did not find goroutine data in: %s", data)
	}
	return gr, nil
}

func scanDoubleLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		// We have a full newline-terminated line.
		return i + 1, data[0:i], nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}
