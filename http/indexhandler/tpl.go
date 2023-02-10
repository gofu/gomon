package indexhandler

type Link struct {
	Name string
	HREF string
	Desc string
}

type Data struct {
	PProfURL string
	Links    []Link
}
