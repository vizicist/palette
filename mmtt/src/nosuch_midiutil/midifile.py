"""
midifile.py -- MIDI classes and parser in Python
Originally placed into the public domain in December 2001 by Will Ware.
Much modified and enhanced by Tim Thompson.
"""

import sys, string, types, exceptions

debugflag = 0

def showstr(str, n=16):
    for x in str[:n]:
        print ('%02x' % ord(x)),
    print

def getNumber(str, length):
    # MIDI uses big-endian for everything
    sum = 0
    for i in range(length):
        sum = (sum << 8) + ord(str[i])
    return sum, str[length:]

def getVariableLengthNumber(str):
    sum = 0
    i = 0
    while 1:
        x = ord(str[i])
        i = i + 1
        sum = (sum << 7) + (x & 0x7F)
        if not (x & 0x80):
            return sum, str[i:]

def putNumber(num, length):
    # MIDI uses big-endian for everything
    lst = [ ]
    for i in range(length):
        n = 8 * (length - 1 - i)
        lst.append(chr((num >> n) & 0xFF))
    return string.join(lst, "")

def putVariableLengthNumber(x):
    lst = [ ]
    while 1:
        y, x = x & 0x7F, x >> 7
        lst.append(chr(y + 0x80))
        if x == 0:
            break
    lst.reverse()
    lst[-1] = chr(ord(lst[-1]) & 0x7f)
    return string.join(lst, "")


class EnumException(exceptions.Exception):
    pass

class Enumeration:
    def __init__(self, enumList):
        lookup = { }
        reverseLookup = { }
        i = 0
        uniqueNames = [ ]
        uniqueValues = [ ]
        for x in enumList:
            if type(x) == types.TupleType:
                x, i = x
            if type(x) != types.StringType:
                raise EnumException, "enum name is not a string: " + x
            if type(i) != types.IntType:
                raise EnumException, "enum value is not an integer: " + i
            if x in uniqueNames:
                raise EnumException, "enum name is not unique: " + x
            if i in uniqueValues:
                raise EnumException, "enum value is not unique for " + x
            uniqueNames.append(x)
            uniqueValues.append(i)
            lookup[x] = i
            reverseLookup[i] = x
            i = i + 1
        self.lookup = lookup
        self.reverseLookup = reverseLookup
    def __add__(self, other):
        lst = [ ]
        for k in self.lookup.keys():
            lst.append((k, self.lookup[k]))
        for k in other.lookup.keys():
            lst.append((k, other.lookup[k]))
        return Enumeration(lst)
    def hasattr(self, attr):
        return self.lookup.has_key(attr)
    def has_value(self, attr):
        return self.reverseLookup.has_key(attr)
    def __getattr__(self, attr):
        if not self.lookup.has_key(attr):
            raise AttributeError
        return self.lookup[attr]
    def whatis(self, value):
        return self.reverseLookup[value]


channelVoiceMessages = Enumeration([("NOTE_OFF", 0x80),
                                    ("NOTE_ON", 0x90),
                                    ("POLYPHONIC_KEY_PRESSURE", 0xA0),
                                    ("CONTROLLER_CHANGE", 0xB0),
                                    ("PROGRAM_CHANGE", 0xC0),
                                    ("CHANNEL_KEY_PRESSURE", 0xD0),
                                    ("PITCH_BEND", 0xE0)])

channelModeMessages = Enumeration([("ALL_SOUND_OFF", 0x78),
                                   ("RESET_ALL_CONTROLLERS", 0x79),
                                   ("LOCAL_CONTROL", 0x7A),
                                   ("ALL_NOTES_OFF", 0x7B),
                                   ("OMNI_MODE_OFF", 0x7C),
                                   ("OMNI_MODE_ON", 0x7D),
                                   ("MONO_MODE_ON", 0x7E),
                                   ("POLY_MODE_ON", 0x7F)])

metaEvents = Enumeration([("SEQUENCE_NUMBER", 0x00),
                          ("TEXT_EVENT", 0x01),
                          ("COPYRIGHT_NOTICE", 0x02),
                          ("SEQUENCE_TRACK_NAME", 0x03),
                          ("INSTRUMENT_NAME", 0x04),
                          ("LYRIC", 0x05),
                          ("MARKER", 0x06),
                          ("CUE_POINT", 0x07),
                          ("MIDI_CHANNEL_PREFIX", 0x20),
                          ("MIDI_PORT", 0x21),
                          ("END_OF_TRACK", 0x2F),
                          ("SET_TEMPO", 0x51),
                          ("SMTPE_OFFSET", 0x54),
                          ("TIME_SIGNATURE", 0x58),
                          ("KEY_SIGNATURE", 0x59),
                          ("SEQUENCER_SPECIFIC_META_EVENT", 0x7F)])


# runningStatus appears to want to be an attribute of a MidiFileTrack. But
# it doesn't seem to do any harm to implement it as a global.
runningStatus = None

class MidiFileEvent:

    def __init__(self, track):
        self.track = track
        self.clocks = None
        self.channel = self.pitch = self.velocity = self.data = None

    # def __cmp__(self, other):
    #    # assert self.clocks != None and other.clocks != None
    #   return cmp(self.clocks, other.clocks)

    def read(self, clocks, str):
        global runningStatus
        self.clocks = clocks
        # do we need to use running status?
        if not (ord(str[0]) & 0x80):
            str = runningStatus + str
        runningStatus = x = str[0]
        x = ord(x)
        y = x & 0xF0
        z = ord(str[1])

        if channelVoiceMessages.has_value(y):
            self.channel = (x & 0x0F) + 1
            channel = self.track.channels[self.channel - 1]
            self.type = channelVoiceMessages.whatis(y)
	    if (self.type == "PROGRAM_CHANGE"):
                self.data = z
                channel.program(self.clocks, z)
                return str[2:]
            elif (self.type == "CHANNEL_KEY_PRESSURE"):
                self.data = z
                channel.chanpressure(self.clocks, z)
                return str[2:]
	    elif self.type == "PITCH_BEND":
                self.data = z
                v1 = ord(str[1]) & 0x3f
                v2 = ord(str[2]) & 0x3f
	        self.track.midifile.callback.pitchbend(self.clocks,self.track.index,self.channel,v1 + (v2<<6))
                return str[3:]
	    elif self.type == "CONTROLLER_CHANGE":
                self.data = z
                val = ord(str[2])
	        self.track.midifile.callback.controller(self.clocks,self.track.index,self.channel,z,val)
                return str[3:]
	    elif (self.type=="NOTE_ON" or self.type=="NOTE_OFF"):
                self.pitch = z
                self.velocity = ord(str[2])
                if (self.type == "NOTE_OFF" or
                    (self.velocity == 0 and self.type == "NOTE_ON")):
                    channel.noteOff(self.clocks, self.pitch, self.velocity)
                elif self.type == "NOTE_ON":
                    channel.noteOn(self.clocks, self.pitch, self.velocity)
                return str[3:]
	    else:
	        raise "Unhandled self.type=",self.type

        elif y == 0xB0 and channelModeMessages.has_value(z):
            self.channel = (x & 0x0F) + 1
            self.type = channelModeMessages.whatis(z)
            if self.type == "LOCAL_CONTROL":
                self.data = (ord(str[2]) == 0x7F)
            elif self.type == "MONO_MODE_ON":
                self.data = ord(str[2])
            return str[3:]

        elif x == 0xF0 or x == 0xF7:
            self.type = {0xF0: "F0_SYSEX_EVENT",
                         0xF7: "F7_SYSEX_EVENT"}[x]
            length, str = getVariableLengthNumber(str[1:])
            self.data = str[:length]
            return str[length:]

        elif x == 0xFF:
            if not metaEvents.has_value(z):
                print "Unknown meta event: FF %02X" % z
                sys.stdout.flush()
                raise "Unknown midi event type"
            self.type = metaEvents.whatis(z)
            length, str = getVariableLengthNumber(str[2:])
            self.data = str[:length]
            return str[length:]

        raise "Unknown midi event type"

    def write(self):
        sysex_event_dict = {"F0_SYSEX_EVENT": 0xF0,
                            "F7_SYSEX_EVENT": 0xF7}
        if channelVoiceMessages.hasattr(self.type):
            x = chr((self.channel - 1) +
                    getattr(channelVoiceMessages, self.type))
            if (self.type != "PROGRAM_CHANGE" and
                self.type != "CHANNEL_KEY_PRESSURE"):
                data = chr(self.pitch) + chr(self.velocity)
            else:
                data = chr(self.data)
            return x + data

        elif channelModeMessages.hasattr(self.type):
            x = getattr(channelModeMessages, self.type)
            x = (chr(0xB0 + (self.channel - 1)) +
                 chr(x) +
                 chr(self.data))
            return x

        elif sysex_event_dict.has_key(self.type):
            str = chr(sysex_event_dict[self.type])
            str = str + putVariableLengthNumber(len(self.data))
            return str + self.data

        elif metaEvents.hasattr(self.type):
            str = chr(0xFF) + chr(getattr(metaEvents, self.type))
            str = str + putVariableLengthNumber(len(self.data))
            return str + self.data

        else:
            raise "unknown midi event type: " + self.type

class MidiFileChannel:

    """A channel (together with a track) provides the continuity
	connecting a NOTE_ON event with its corresponding NOTE_OFF event.
	Together, those define the beginning and ending clocks for a Note.
	"""

    def __init__(self, track, index):
        self.index = index
        self.track = track

    def noteOn(self, clocks, pitch, velocity):
        self.track.midifile.callback.noteon(clocks,
			self.track.index, self.index, pitch, velocity)

    def noteOff(self, clocks, pitch, velocity):
        self.track.midifile.callback.noteoff(clocks,
			self.track.index, self.index, pitch, velocity)

    def program(self, clocks, v):
        self.track.midifile.callback.program(clocks,
			self.track.index, self.index, v)

    def chanpressure(self, clocks, v):
        self.track.midifile.callback.chanpressure(clocks,
			self.track.index, self.index, v)

class DeltaTime(MidiFileEvent):

    type = "DeltaTime"

    def read(self, oldstr):
        self.clocks, newstr = getVariableLengthNumber(oldstr)
        return self.clocks, newstr

    def write(self):
        str = putVariableLengthNumber(self.clocks)
        return str



class MidiFileTrack:

    def __init__(self, midifile, index):
        self.index = index
        self.events = [ ]
        self.channels = [ ]
        self.length = 0
	self.midifile = midifile
        for i in range(16):
            self.channels.append(MidiFileChannel(self, i+1))

    def read(self, str):
        clocks = 0
        assert str[:4] == "MTrk"
        length, str = getNumber(str[4:], 4)
        self.length = length
        mystr = str[:length]
        remainder = str[length:]
        while mystr:
            delta_t = DeltaTime(self)
            dt, mystr = delta_t.read(mystr)
            clocks = clocks + dt
            self.events.append(delta_t)
            e = MidiFileEvent(self)
            mystr = e.read(clocks, mystr)
            self.events.append(e)
        return remainder

    def write(self):
        clocks = self.events[0].clocks
        # build str using MidiFileEvent
        str = ""
        for e in self.events:
            str = str + e.write()
        return "MTrk" + putNumber(len(str), 4) + str

    def __repr__(self):
        r = "<MidiFileTrack %d -- %d events\n" % (self.index,
len(self.events))
        for e in self.events:
            r = r + "    " + `e` + "\n"
        return r + "  >"


class MidiFileCallback:

	def noteon(self, clocks, track, channel, pitch, velocity):
		pass

	def noteoff(self, clocks, track, channel, pitch, velocity):
		pass

	def program(self, clocks, track, channel, program):
		pass

	def chanpressure(self, clocks, track, channel, pressure):
		pass

class MidiFile:

    """A class for manipulating MIDI Files"""

    def __init__(self, callback = None):
        self.file = None
        self.format = 1
        self.tracks = [ ]
        self.ticksPerQuarterNote = None
        self.ticksPerSecond = None
	if callback == None:
		callback = MidiFileCallback()
	self.callback = callback

    def open(self, filename, attrib="rb"):
	"""Open the file for reading or writing"""
        if filename == None:
            if attrib in ["r", "rb"]:
                self.file = sys.stdin
            else:
                self.file = sys.stdout
        else:
            self.file = open(filename, attrib)

    def __repr__(self):
        r = "<MidiFile %d tracks\n" % len(self.tracks)
        for t in self.tracks:
            r = r + "  " + `t` + "\n"
        return r + ">"

    def close(self):
        self.file.close()

    def read(self):
        self.readstr(self.file.read())

    def readstr(self, str):
        assert str[:4] == "MThd"
        length, str = getNumber(str[4:], 4)
        assert length == 6
        format, str = getNumber(str, 2)
        self.format = format
        assert format == 0 or format == 1   # dunno how to handle 2
        numTracks, str = getNumber(str, 2)
        division, str = getNumber(str, 2)
        if division & 0x8000:
            framesPerSecond = -((division >> 8) | -128)
            ticksPerFrame = division & 0xFF
            assert ticksPerFrame == 24 or ticksPerFrame == 25 or \
                   ticksPerFrame == 29 or ticksPerFrame == 30
            if ticksPerFrame == 29: ticksPerFrame = 30  # drop frame
            self.ticksPerSecond = ticksPerFrame * framesPerSecond
        else:
            self.ticksPerQuarterNote = division & 0x7FFF
        for i in range(numTracks):
            trk = MidiFileTrack(self,i)
            str = trk.read(str)
            self.tracks.append(trk)

    def write(self):
        self.file.write(self.writestr())

    def writestr(self):
        division = self.ticksPerQuarterNote
        # Don't handle ticksPerSecond yet, too confusing
        assert (division & 0x8000) == 0
        str = "MThd" + putNumber(6, 4) + putNumber(self.format, 2)
        str = str + putNumber(len(self.tracks), 2)
        str = str + putNumber(division, 2)
        for trk in self.tracks:
            str = str + trk.write()
        return str


if __name__ == "__main__":
    print "No main here"
