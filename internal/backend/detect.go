package backend

import (
	"fmt"
	"sort"
)

type DetectFunc func() (Backend, error)

var registry = map[string]DetectFunc{}

func Register(name string, f DetectFunc) {
	registry[name] = f
}

func Detect(name ...string) (Backend, error) {
	if len(name) > 0 && name[0] != "" {
		f, ok := registry[name[0]]
		if !ok {
			return nil, fmt.Errorf("unknown backend: %s", name[0])
		}
		return f()
	}
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if b, err := registry[name](); err == nil {
			return b, nil
		}
	}
	return nil, fmt.Errorf("no supported terminal multiplexer found")
}
