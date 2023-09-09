package hal

type Object struct {
	ID    string
	Links LinkMap
}

type LinkMap map[string]Link

type Link struct {
	Href      string
	Name      *string
	Type      *string
	Templated bool
}
