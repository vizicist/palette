package winsys

import (
	"fmt"
)

// NewSetParamMsg xxx
func NewSetParamMsg(name string, value string) string {
	return fmt.Sprintf("\"name\":\"%s\",\"value\":\"%s\"", name, value)
}

var WindowList map[string]Window

func WindowNamed(wname string) Window {
	w, ok := WindowList[wname]
	if !ok {
		return nil
	}
	return w
}
