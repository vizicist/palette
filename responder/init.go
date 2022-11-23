package responder

import "github.com/vizicist/palette/engine"

var AllResponders = map[string]engine.Responder{}

func GetResponder(name string) engine.Responder {
	r, ok := AllResponders[name]
	if !ok {
		return nil
	}
	return r
}

func init() {
}
