# To control your local configuration, you can create a file
# named c:\keylocal.k (ie. in the root directory of the drive you
# are running keykit from), and put a function keylocal() in it.
# That function will be executed when keykit starts, and you 
# can set things like the default MIDI input/output ports, etc.
#
# This is a sample.  You should copy this file to c:\keylocal.k
# and then edit it as needed.  The Contribdir, if set, is where things
# like GeoMaestro will be located.

function keylocal() {

	Contribdir = "c:\\contrib"

	# You can add (multiple) things to be executed at the beginning and
	# end of the boot process, by using addprerc and addpostrc, like this:
	addprerc(global(localprerc))
	addpostrc(global(localpostrc))

	# Inittask = remotecons(4444)  # would start remote console on port 4444
	# Inittask = palette_listen()
}

function localprerc() {
	# This would get executed at the beginning of keyrc()
}

function localpostrc() {
	# This would get executed at the end of keyrc()
}
