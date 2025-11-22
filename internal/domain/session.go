package domain

type SessionOptions struct {
	Command string `json:"command,omitempty"`
	Path    string `json:"path,omitempty"`
}
