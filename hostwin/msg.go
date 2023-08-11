package hostwin

import (
	"fmt"
	"image"
	"image/color"
	"strconv"
)

var PointZero image.Point = image.Point{0, 0}

type Cmd struct {
	Subj   string
	Values map[string]string
}

// ErrorResult xxx
func ErrorResult(err string) string {
	return fmt.Sprintf("\"error\":\"%s\"", err)
}

// ErrorResult xxx
func OkResult() string {
	return ""
}

func (cmd Cmd) ValuesToString() string {
	sep := ""
	s := ""
	for _, v := range cmd.Values {
		s = s + sep + v
		sep = ","
	}
	return s
}

func (cmd Cmd) VerifyValues(names ...string) bool {
	values := cmd.Values
	for _, name := range names {
		_, ok := values[name]
		if !ok {
			return false
		}
	}
	return true
}

func (cmd Cmd) ValuesBool(name string, dflt bool) bool {
	values := cmd.Values
	v, ok := values[name]
	if !ok {
		return dflt
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		LogIfError(err)
		b = dflt
	}
	return b
}

func (cmd Cmd) ValuesFloat(name string, dflt float32) float32 {
	values := cmd.Values
	v, ok := values[name]
	if !ok {
		return dflt
	}
	f, err := strconv.ParseFloat(v, 32)
	if err == nil {
		return dflt
	}
	return float32(f)
}

func (cmd Cmd) ValuesSetXY(key string, pos image.Point) {
	cmd.Values[key] = fmt.Sprintf("%d,%d", pos.X, pos.Y)
}

func (cmd Cmd) ValuesSetPos(pos image.Point) {
	cmd.ValuesSetXY("pos", pos)
}

func (cmd Cmd) ValuesSetXY0(pos image.Point) {
	cmd.ValuesSetXY("xy0", pos)
}

func (cmd Cmd) ValuesSetXY1(pos image.Point) {
	cmd.ValuesSetXY("xy1", pos)
}

func (cmd Cmd) ValuesPos(dflt image.Point) image.Point {
	return cmd.ValuesXY("pos", dflt)
}

func (cmd Cmd) ValuesSize(dflt image.Point) image.Point {
	return cmd.ValuesXY("size", dflt)
}

func (cmd Cmd) ValuesXY0(dflt image.Point) image.Point {
	return cmd.ValuesXY("xy0", dflt)
}

func (cmd Cmd) ValuesXY1(dflt image.Point) image.Point {
	return cmd.ValuesXY("xy1", dflt)
}

func (cmd Cmd) ValuesRect(dflt image.Rectangle) image.Rectangle {
	xy0 := cmd.ValuesXY("xy0", PointZero)
	xy1 := cmd.ValuesXY("xy1", PointZero)
	return image.Rectangle{Min: xy0, Max: xy1}
}

// ValuesXY is used to get any 2-valued int value (ie. pos or size)
func (cmd Cmd) ValuesXY(xyname string, dflt image.Point) image.Point {
	xystr, ok := cmd.Values[xyname]
	if !ok {
		return dflt
	}
	var x, y int
	n, err := fmt.Sscanf(xystr, "%d,%d", &x, &y)
	if err != nil {
		LogIfError(err)
		return dflt
	}
	if n != 2 {
		LogWarn("ValuesXY didn't parse", "xystr", xystr)
		return dflt
	}
	return image.Point{x, y}
}

func (cmd Cmd) ValuesColor(dflt color.RGBA) color.RGBA {
	r := cmd.ValuesUint8("r", dflt.R)
	g := cmd.ValuesUint8("g", dflt.G)
	b := cmd.ValuesUint8("b", dflt.B)
	a := cmd.ValuesUint8("a", dflt.A)
	return color.RGBA{
		R: r,
		G: g,
		B: b,
		A: a,
	}
}

func (cmd Cmd) ValuesUint8(name string, dflt uint8) uint8 {
	return uint8(cmd.ValuesInt(name, int(dflt)))
}

func (cmd Cmd) ValuesInt(name string, dflt int) int {
	v, ok := cmd.Values[name]
	if !ok {
		return dflt
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		LogIfError(err)
		return dflt
	}
	return i
}

func (cmd Cmd) ValuesString(name string, dflt string) string {
	s, ok := cmd.Values[name]
	if !ok {
		return dflt
	}
	return s
}
