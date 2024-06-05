# keylocal is a function that gets called automatically when Keykit starts up.
# 

function palette_init() {
	print("Starting listener for Space Palette events...")
	task palette_listen()
}

function keylocal() {
	addpostrc(palette_init)
}
