package graphhandler

type Data struct {
	Total  int     `json:"total"`
	Unique int     `json:"unique"`
	Group  []Group `json:"group"`
}
