package domain

type WindowOptions struct {
	Command string `json:"command,omitempty"`
	Path    string `json:"path,omitempty"`
	Layout  string `json:"layout,omitempty"`
}

type WindowInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Index  string `json:"index"`
	Layout string `json:"layout"`
}
