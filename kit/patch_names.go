package kit

var patchNames = [...]string{"A", "B", "C", "D"}

func isPatchName(patch string) bool {
	for _, name := range patchNames {
		if patch == name {
			return true
		}
	}
	return false
}
