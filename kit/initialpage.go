package kit

func IsBSSInitialPage() bool {
	page, err := GetParam("global.initialpage")
	return err != nil || page != "pro"
}
