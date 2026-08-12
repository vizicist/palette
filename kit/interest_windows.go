//go:build windows

package kit

// Screen capture for the interestingness evaluation: grabs one monitor,
// downscaled to a small luma buffer via GDI. Runs at a couple of Hz at
// most, so plain BitBlt/StretchBlt is plenty.

import (
	"fmt"
	"sync"
	"syscall"
	"unsafe"
)

var (
	gdi32                  = syscall.NewLazyDLL("gdi32.dll")
	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procStretchBlt         = gdi32.NewProc("StretchBlt")
	procSetStretchBltMode  = gdi32.NewProc("SetStretchBltMode")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procGdiFlush           = gdi32.NewProc("GdiFlush")
	procGetDC              = user32.NewProc("GetDC")
	procReleaseDC          = user32.NewProc("ReleaseDC")
)

const (
	srccopy      = 0x00CC0020
	captureblt   = 0x40000000
	halftone     = 4
	biRGB        = 0
	dibRGBColors = 0
)

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

// Captures are serialized; GDI handles here are not thread-safe.
var captureMutex sync.Mutex

// autoCaptureIndex picks the monitor an interestmonitor of -1 captures:
// the largest non-primary monitor by area. The Resolume output is
// typically the biggest secondary display, and this skips smaller control
// surfaces like the Space Palette Pro's portrait GUI touchscreen. Falls
// back to the primary when there is no secondary monitor.
func autoCaptureIndex(bounds []rect) int {
	best := -1
	bestArea := int32(0)
	for i, r := range bounds {
		if r.Left == 0 && r.Top == 0 {
			continue // primary
		}
		area := (r.Right - r.Left) * (r.Bottom - r.Top)
		if area > bestArea {
			bestArea = area
			best = i
		}
	}
	if best < 0 {
		return 0
	}
	return best
}

// interestMonitorRect picks which monitor to capture. index >= 0 selects
// that monitor from the enumeration; -1 means auto (see autoCaptureIndex).
func interestMonitorRect(index int) (rect, error) {
	bounds, err := enumerateMonitors()
	if err != nil {
		return rect{}, err
	}
	if len(bounds) == 0 {
		return rect{}, fmt.Errorf("no monitors found")
	}
	if index >= 0 {
		if index >= len(bounds) {
			return rect{}, fmt.Errorf("monitor %d not found (%d monitors)", index, len(bounds))
		}
		return bounds[index], nil
	}
	return bounds[autoCaptureIndex(bounds)], nil
}

// ListCaptureMonitors returns a JSON array describing the monitors in
// capture-index order, so a global.interestmonitor value can be chosen:
// [{"index":0,"left":0,"top":0,"width":2560,"height":1440,"primary":true,"auto":false}, ...]
// "auto" marks the one an interestmonitor of -1 would capture.
func ListCaptureMonitors() (string, error) {
	bounds, err := enumerateMonitors()
	if err != nil {
		return "", err
	}
	autoIndex := autoCaptureIndex(bounds)
	out := "["
	for i, r := range bounds {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf(`{"index":%d,"left":%d,"top":%d,"width":%d,"height":%d,"primary":%v,"auto":%v}`,
			i, r.Left, r.Top, r.Right-r.Left, r.Bottom-r.Top,
			r.Left == 0 && r.Top == 0, i == autoIndex)
	}
	return out + "]", nil
}

// CaptureMonitorLuma captures the given monitor (-1 = auto) downscaled to
// w x h and returns row-major luma values in 0..1.
func CaptureMonitorLuma(monitor int, w int, h int) ([]float64, error) {

	captureMutex.Lock()
	defer captureMutex.Unlock()

	mr, err := interestMonitorRect(monitor)
	if err != nil {
		return nil, err
	}
	srcW := int(mr.Right - mr.Left)
	srcH := int(mr.Bottom - mr.Top)
	if srcW <= 0 || srcH <= 0 {
		return nil, fmt.Errorf("bad monitor size %dx%d", srcW, srcH)
	}

	screenDC, _, _ := procGetDC.Call(0)
	if screenDC == 0 {
		return nil, fmt.Errorf("GetDC failed")
	}
	defer procReleaseDC.Call(0, screenDC)

	memDC, _, _ := procCreateCompatibleDC.Call(screenDC)
	if memDC == 0 {
		return nil, fmt.Errorf("CreateCompatibleDC failed")
	}
	defer procDeleteDC.Call(memDC)

	// Top-down 32bpp DIB so the bits are directly addressable.
	bi := bitmapInfoHeader{
		Width:    int32(w),
		Height:   -int32(h),
		Planes:   1,
		BitCount: 32,
	}
	bi.Size = uint32(unsafe.Sizeof(bi))
	var bitsPtr unsafe.Pointer
	bitmap, _, _ := procCreateDIBSection.Call(memDC, uintptr(unsafe.Pointer(&bi)),
		dibRGBColors, uintptr(unsafe.Pointer(&bitsPtr)), 0, 0)
	if bitmap == 0 || bitsPtr == nil {
		return nil, fmt.Errorf("CreateDIBSection failed")
	}
	defer procDeleteObject.Call(bitmap)

	oldObj, _, _ := procSelectObject.Call(memDC, bitmap)
	defer procSelectObject.Call(memDC, oldObj)

	// HALFTONE averages source pixels during the downscale, so the luma
	// statistics reflect the whole screen rather than a sparse sampling.
	procSetStretchBltMode.Call(memDC, halftone)
	ret, _, err := procStretchBlt.Call(memDC, 0, 0, uintptr(w), uintptr(h),
		screenDC, uintptr(int32(mr.Left)), uintptr(int32(mr.Top)),
		uintptr(srcW), uintptr(srcH), srccopy|captureblt)
	if ret == 0 {
		return nil, fmt.Errorf("StretchBlt failed: %v", err)
	}
	procGdiFlush.Call()

	pixels := unsafe.Slice((*byte)(bitsPtr), w*h*4)
	luma := make([]float64, w*h)
	for i := 0; i < w*h; i++ {
		b := float64(pixels[i*4+0])
		g := float64(pixels[i*4+1])
		r := float64(pixels[i*4+2])
		luma[i] = (0.2126*r + 0.7152*g + 0.0722*b) / 255.0
	}
	return luma, nil
}
