package kit

var tm0 int
var Firsttime int

func mdep_milliclock() int {
	return (timeGetTime()) - tm0
}

func mdep_resetclock() {
	Firsttime = timeGetTime()
	tm0 = Firsttime
}
