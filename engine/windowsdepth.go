//go:build windowsdepth
// +build windowsdepth

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
	LogInfo("GO DepthCallback!", "subj", gosubj, "msg", gomsg)
}

func DepthRunForever() {
	LogInfo("Calling C.DepthRun")
	show := 1
	i := C.DepthRun((C.DepthCallbackFunc)(unsafe.Pointer(C.DepthCallback)), C.int(show))
	LogInfo("C.DepthRun returned?", "i", i)
}
