package kit

import (
	"github.com/vizicist/palette/engine"
)

var tm0 int
var Firsttime int

func mdep_milliclock() int {
	return int(engine.CurrentMilli())
}

func mdep_resetclock() {
	Firsttime = int(engine.CurrentMilli())
	tm0 = Firsttime
}
