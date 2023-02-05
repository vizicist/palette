package rtmidi

// #cgo LDFLAGS: -luuid -lksuser -lwinmm -lole32 -L${SRCDIR}/lib -lrtmidi
/*

#include <stdlib.h>
#include <stdint.h>
#include "rtmidilib.h"

static inline int cgoTjt() {
	return 444;
}

*/
import "C"
import (
	// "errors"
	"fmt"
	// "sync"
	// "unsafe"
)

func Tjt() int {
	fmt.Printf("Hi from TJT()\n")
	return int(C.cgoTjt())
}

/*
func Tjt2() {
	fmt.Printf("Hi from TJT2()\n")
	apis := CompiledAPI()
	fmt.Printf("apis = %v\n", apis)
}
*/
