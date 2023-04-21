package router

// Default application routes.
var Default = Router{
	Index: "/",
	HTML:  "/html",
	JSON:  "/json",
	Graph: "/graph",
	PProf: "/debug/pprof/",
}
