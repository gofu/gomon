package indexhandler

// Link shown on the index page.
type Link struct {
	// Text of the link.
	Text string
	// HREF of the link.
	HREF string
	// Description of the page the link navigates to.
	Description string
}

// Data for the index page.
type Data struct {
	// ProfilerSource contains the (profiler.Profiler).Source() value.
	ProfilerSource string
	// Links to list on the index page.
	Links []Link
}
