NOSUCH MIDIUTIL PACKAGE
-----------------------

This is a crudely packaged but functioning Python interface that
lets you handle midi files and do scheduling of MIDI I/O.
Essentially, it's a higher-level interface to the low-level
pyportmidi package, which provides a raw MIDI interface using
the portmidi libraries.  See the bottom of this file for
contributors and licensing.

See the notes below about the location of a bug-fixed version of pyportmidi.
Although the nosuch.midiutil package works in both Python 2.4 and 2.5,
you'll need to get and install the appropriate (2.4 or 2.5)
bug-fixed version of pyportmidi.

TO INSTALL:
-----------
Create lib\site-packages\nosuch under your python root directory.
Copy midifile.py and midiutil.py into that directory.
If there's not already a __init__.py file in the lib\site-packages\nosuch
directory, create an empty file named __init__.py file in that directory.

TO RUN THE DEMOS:
----------------

"python midimon.py" will monitor and print MIDI input.
"python midiprint.py {midifile-filename}" will read a midifile
as a single "phrase" and print it as a "schedule list".
"python midiprint.py {midifile-filename} {midi-output-name}" will
read a midifile and play it on the specified MIDI output.

DOCUMENTATION:
--------------
There is no documentation on using the classes other than
the source code.

PYPORTMIDI Version v0.0.4
-------------------------
It is assumed that you already have the pyportmidi package installed.
I have fixed some bugs in pyportmidi v0.0.3 - if possible, you should
get/install pyportmidi version v0.0.4 or later.  You can find v0.0.4
here: http://nosuch.com/pyportmidi . There are versions there for both
Python 2.4 and 2.5.

Any problems/questions, send email to tjt@nosuch.com

---------------------------------------------------------------

Contributors:

	Tim Thompson (tjt@nosuch.com)
	Julianne Sharer (jsharer@tarletonltd.com)

License:

	Images embedded in midiseq.py are from the Tango Desktop Project at
	http://tango.freedesktop.org/Tango_Desktop_Project, which licenses
	them under the Creative Commons Attribution Share-Alike license.

	Everything else here is licensed under the
	Creative Commons Attribution Share-Alike license, found here:
	http://creativecommons.org/licenses/by-sa/3.0/

