package main

import (
	"fmt"
)

// NewSetParamMsg xxx
func NewSetParamMsg(name string, value string) string {
	return fmt.Sprintf("\"name\":\"%s\",\"value\":\"%s\"", name, value)
}
