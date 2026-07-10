//go:build !windows

package main

import "fmt"

func main() {
	fmt.Println("Palette's executable installer runs on Windows only.")
}
