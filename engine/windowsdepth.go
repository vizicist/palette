//go:build windows
// +build windows

package engine

// #cgo LDFLAGS: -L. "${SRCDIR}/../depthlib/build/x64/Release/depthlib.dll"
/*
#include <stdlib.h>
#include <stdio.h>
#include "../depthlib/include/depthlib.h"

extern void DepthCallback(char *subj, char *msg);
*/
import "C"
import (
	"unsafe"
)

//export DepthCallback
func DepthCallback(subj *C.char, msg *C.char) {
	gosubj := C.GoString(subj)
	gomsg := C.GoString(msg)
	Log.Debugf("GO DepthCallback! subj=%s msg=%s\n", gosubj, gomsg)
}

func DepthRunForever() {
	Log.Debugf("Calling C.DepthRun\n")
	show := 1
	i := C.DepthRun((C.DepthCallbackFunc)(unsafe.Pointer(C.DepthCallback)), C.int(show))
	Log.Debugf("C.DepthRun returned? i = %d\n", i)
}
