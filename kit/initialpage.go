package kit

func IsBSS2InitialPage() bool {
	page, err := GetParam("global.initialpage")
	return err != nil || page == "" || page == "bss2"
}
