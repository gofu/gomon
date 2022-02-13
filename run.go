package gomon

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers/g"
	"github.com/alecthomas/chroma/styles"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

type Config struct {
	Addr         string
	URL          string
	LocalRoot    string
	LocalGoRoot  string
	LocalGoPath  string
	RemoteRoot   string
	RemoteGoRoot string
	RemoteGoPath string
}

func Run(ctx context.Context, conf Config) error {
	ln, err := net.Listen("tcp", conf.Addr)
	if err != nil {
		return err
	}
	log.Printf("Listening on http://%s", ln.Addr())
	group, ctx := errgroup.WithContext(ctx)
	if len(conf.LocalGoRoot) == 0 {
		conf.LocalGoRoot = runtime.GOROOT()
	}
	if len(conf.LocalGoPath) == 0 {
		conf.LocalGoPath = os.Getenv("GOPATH")
	}
	svc := NewService(conf.URL, conf.LocalRoot, conf.LocalGoRoot, conf.LocalGoPath, conf.RemoteRoot, conf.RemoteGoRoot, conf.RemoteGoPath)
	h := NewHandler(conf.LocalRoot, conf.LocalGoRoot, conf.LocalGoPath, svc)
	srv := &http.Server{
		Addr:              ln.Addr().String(),
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}
	group.Go(func() error { return srv.Serve(ln) })
	group.Go(func() error {
		<-ctx.Done()
		_ = srv.Close()
		return ctx.Err()
	})
	return group.Wait()
}

type goroutineParser struct {
	root   string
	goroot string
	gopath string
}

func newGoroutineParser(root, goroot, gopath string) *goroutineParser {
	return &goroutineParser{root: root, goroot: goroot, gopath: gopath}
}

type Service struct {
	url    string
	parser *goroutineParser
}

func NewService(url, localRoot, localGoRoot, localGoPath, remoteRoot, remoteGoRoot, remoteGoPath string) *Service {
	if len(url) != 0 && !strings.Contains(url, "://") {
		url = "http://" + url
	}
	root, goroot, gopath := remoteRoot, remoteGoRoot, remoteGoPath
	if len(root) == 0 {
		root = localRoot
	}
	if len(goroot) == 0 {
		goroot = localGoRoot
	}
	if len(gopath) == 0 {
		gopath = localGoPath
	}
	return &Service{
		url:    url,
		parser: newGoroutineParser(root, goroot, gopath),
	}
}

type StackElem struct {
	Caller  bool          `json:"caller"`
	Package string        `json:"package"`
	Method  string        `json:"method"`
	Args    string        `json:"args,omitempty"`
	File    string        `json:"file"`
	Line    int           `json:"line"`
	Root    string        `json:"root,omitempty"`
	Extra   string        `json:"extra,omitempty"`
	Prefix  template.HTML `json:"prefix,omitempty"`
	Suffix  template.HTML `json:"suffix,omitempty"`
}

type Goroutine struct {
	ID       int           `json:"id"`
	Op       string        `json:"op"`
	Duration time.Duration `json:"duration,omitempty"`
	Stack    []StackElem   `json:"stack,omitempty"`
}

type GoroutineContent struct {
	Highlight
	Goroutine
}

func (s *Service) GetRunning() ([]Goroutine, error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	running, err := s.parser.parseGoroutines(res.Body)
	_ = res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read %s response: %w", req.URL, err)
	}
	sort.Slice(running, func(i, j int) bool {
		return running[i].Duration > running[j].Duration
	})
	return running, nil
}

func (p *goroutineParser) parseGoroutines(r io.Reader) ([]Goroutine, error) {
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

func (p *goroutineParser) parseGoroutine(data string) (Goroutine, error) {
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
		if strings.HasPrefix(matches[1], p.root) {
			stack.File = strings.TrimLeft(strings.TrimPrefix(matches[1], p.root), "/")
			stack.Root = "PROJECT"
		} else if strings.HasPrefix(matches[1], p.goroot) {
			stack.File = strings.TrimLeft(strings.TrimPrefix(matches[1], p.goroot), "/")
			stack.Root = "GOROOT"
		} else if strings.HasPrefix(matches[1], p.gopath) {
			stack.File = strings.TrimLeft(strings.TrimPrefix(matches[1], p.gopath), "/")
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
		gr.Stack = append(gr.Stack, stack)
	}
	if err = s.Err(); err != nil {
		return gr, err
	}
	if gr.ID == 0 || len(gr.Stack) == 0 {
		return gr, fmt.Errorf("did not find goroutine data in: %s", data)
	}
	reverse(gr.Stack)
	return gr, nil
}

func copySlice[T any](s []T) []T {
	ss := make([]T, len(s))
	for i, v := range s {
		ss[i] = v
	}
	return ss
}

func reverse[T any](s []T) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
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

type Handler struct {
	root        string
	goroot      string
	gopath      string
	service     *Service
	highlighter Highlighter
}

func NewHandler(root, goroot, gopath string, service *Service) *Handler {
	return &Handler{root: root, goroot: goroot, gopath: gopath, service: service}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		h.serveIndex(w, r)
	case "/json":
		h.serveJSON(w, r)
	default:
		http.NotFound(w, r)
	}
}

type IndexData struct {
	Total       int
	Lines       int
	Skipped     int
	Min         string
	MarkupLimit int
	Running     []Goroutine
	Reversed    []Goroutine
}

var (
	//go:embed index.gohtml
	indexTplData string
	indexTpl     = template.Must(template.New("").Funcs(template.FuncMap{
		"reverseStack": func(s []StackElem) []StackElem {
			ss := copySlice(s)
			reverse(ss)
			return ss
		},
		"sub": func(a, b int) int { return a - b },
	}).Parse(indexTplData))
)

func (h *Handler) serveIndex(w http.ResponseWriter, r *http.Request) {
	running, err := h.service.GetRunning()
	if err != nil {
		h.serveError(w, r, err)
		return
	}
	data := IndexData{
		Total:   len(running),
		Running: running,
	}
	if min := r.URL.Query().Get("min"); len(min) != 0 {
		minDur, err := time.ParseDuration(min)
		if err != nil {
			h.serveError(w, r, err)
			return
		}
		var filtered []Goroutine
		for _, gr := range running {
			if gr.Duration < minDur {
				data.Skipped++
				continue
			}
			filtered = append(filtered, gr)
		}
		data.Min = min
		data.Running = filtered
	}
	if markupLimit := r.URL.Query().Get("markup"); len(markupLimit) != 0 {
		markup, err := strconv.Atoi(markupLimit)
		if err != nil {
			h.serveError(w, r, err)
			return
		}
		data.MarkupLimit = markup
	}
	if data.MarkupLimit < 0 {
		data.MarkupLimit = 0
	}
	if linesStr := r.URL.Query().Get("lines"); len(linesStr) != 0 {
		data.Lines, err = strconv.Atoi(linesStr)
		if err != nil {
			h.serveError(w, r, err)
			return
		}
	}
	if data.Lines >= 0 {
		for i, goroutine := range data.Running {
			if data.MarkupLimit != 0 && i > data.MarkupLimit {
				break
			}
			for j, s := range goroutine.Stack {
				hh, err := h.highlightStack(s, data.Lines)
				if err != nil {
					h.serveError(w, r, err)
					return
				}
				goroutine.Stack[j].Prefix = template.HTML(hh.Prefix)
				goroutine.Stack[j].Suffix = template.HTML(hh.Suffix)
			}
		}
	}
	var buf bytes.Buffer
	if err = indexTpl.Execute(&buf, data); err != nil {
		h.serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func (h *Handler) serveJSON(w http.ResponseWriter, r *http.Request) {
	running, err := h.service.GetRunning()
	if err != nil {
		h.serveError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(running)
}

func (h *Handler) serveError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("HTTP %s %s error: %s", r.Method, r.URL, err)
	_, _ = io.Copy(w, strings.NewReader(err.Error()))
}

func (h *Handler) highlightStack(s StackElem, wrap int) (Highlight, error) {
	if wrap < 0 {
		return Highlight{}, nil
	}
	var root string
	style := styles.Registry["vulcan"]
	switch s.Root {
	case "PROJECT":
		root = h.root
	case "GOROOT":
		root = h.goroot
	case "GOPATH":
		root = h.gopath
	default:
		return Highlight{}, nil
	}
	hh, err := h.highlighter.Highlight(path.Join(root, s.File), s.Line, wrap, style)
	if err != nil {
		return hh, err
	}
	return hh, nil
}

type Highlighter struct {
	mu    sync.RWMutex
	sf    singleflight.Group
	cache map[string][]chroma.Token
}

type Highlight struct {
	Prefix string
	Suffix string
}

func (h *Highlighter) Highlight(file string, line, wrapSize int, style *chroma.Style) (Highlight, error) {
	var hh Highlight
	if wrapSize < 0 {
		return hh, nil
	}
	if style == nil {
		style = styles.Registry["dracula"]
	}
	allTokens, err := h.getTokens(file)
	if err != nil {
		return hh, err
	}
	var tokens []chroma.Token
	var buf bytes.Buffer
	base := 1
	if wrapDiff := line - wrapSize; wrapDiff > 0 {
		base = wrapDiff
	}
	makeFormatter := func(baseLine, mark int) chroma.Formatter {
		opts := []html.Option{
			html.WithLineNumbers(true),
			html.LineNumbersInTable(true),
			html.Standalone(false),
			html.BaseLineNumber(baseLine),
			//html.WrapLongLines(true),
			html.TabWidth(3),
		}
		if mark != 0 {
			opts = append(opts, html.HighlightLines([][2]int{{mark, mark}}))
		}
		return html.New(opts...)
	}
	skipLines := line - 1 - wrapSize
	if skipLines < 0 {
		skipLines = 0
	}
	stopAtLine := line - 1 + wrapSize
	formatter := makeFormatter(base, line)
	var gotPrefix bool
	var haveLines int
	setSuffix := func() error {
		if len(tokens) == 0 {
			return nil
		}
		buf.Reset()
		err = formatter.Format(&buf, style, chroma.Literator(tokens...))
		if err != nil {
			return err
		}
		hh.Suffix = buf.String()
		tokens = tokens[:0]
		return nil
	}
	for _, tok := range allTokens {
		if file == "../cloner/cmd/cloner/main.go" && line == 405 && haveLines == 400 {
			log.Printf("got file")
		}
		lfCount := strings.Count(tok.Value, "\n")
		if haveLines+lfCount < skipLines {
			haveLines += lfCount
			continue
		}
		if haveLines+lfCount >= line && !gotPrefix {
			if haveLines+lfCount > line {
				log.Printf("overflow")
			}
			haveLines += lfCount
			tokens = appendToken(tokens, tok, 0)
			buf.Reset()
			err = formatter.Format(&buf, style, chroma.Literator(tokens...))
			if err != nil {
				return hh, err
			}
			gotPrefix = true
			tokens = tokens[:0]
			hh.Prefix = buf.String()
			if wrapSize <= 0 {
				break
			}
			formatter = makeFormatter(line+1, 0)
			continue
		}
		if haveLines+lfCount > stopAtLine {
			tokens = appendToken(tokens, tok, 0)
			if err = setSuffix(); err != nil {
				return hh, err
			}
			break
		}
		tokens = appendToken(tokens, tok, skipLines-haveLines)
		haveLines += lfCount
	}
	return hh, setSuffix()
}

func appendToken(tokens []chroma.Token, tok chroma.Token, overflow int) []chroma.Token {
	for i := 0; i < overflow; i++ {
		_, tok.Value, _ = strings.Cut(tok.Value, "\n")
	}
	return append(tokens, tok)
}

func (h *Highlighter) getTokens(file string) ([]chroma.Token, error) {
	h.mu.RLock()
	cached, ok := h.cache[file]
	h.mu.RUnlock()
	if ok {
		return cached, nil
	}
	v, err, _ := h.sf.Do(file, func() (interface{}, error) {
		h.mu.RLock()
		cached, ok := h.cache[file]
		h.mu.RUnlock()
		if ok {
			return cached, nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		iter, err := g.Go.Tokenise(nil, string(data))
		if err != nil {
			return nil, err
		}
		tokens := iter.Tokens()
		h.mu.Lock()
		if h.cache == nil {
			h.cache = map[string][]chroma.Token{}
		}
		h.cache[file] = tokens
		h.mu.Unlock()
		return tokens, nil
	})
	if err != nil {
		return nil, err
	}
	if cached, ok = v.([]chroma.Token); !ok {
		return nil, fmt.Errorf("expected cache type %T, got %T", cached, v)
	}
	return cached, nil
}
