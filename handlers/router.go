package handlers

// Router defines available HTTP routes
// for use across different handlers.
type Router struct {
	// Index page (homepage).
	Index string
	// HTML list of running goroutines.
	HTML string
	// JSON list of running goroutines.
	JSON string
	// PProf debug info (by default /debug/pprof)
	PProf string
}
