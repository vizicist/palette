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
func RegisterResponder(name string, responder engine.Responder) {
	engine.AddResponder(name, responder)
	AllResponders[name] = responder
}

func init() {
}
