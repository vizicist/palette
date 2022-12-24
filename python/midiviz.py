import mido
import palette
import time
import sys

if len(sys.argv) > 1:
    fn = sys.argv[1]
else:
    fn = "midiviz.mid"
mid = mido.MidiFile('../data/midifiles/'+fn)
print("mid = ",mid)
playbacktimefactor = 0.5
mididevice = "testmidi"

def doNote(msg,msgtimesofar,layer):
    palette.SendMIDIEvent(mididevice,msgtimesofar,msg,layer)

reg = 0
layers = "abcd"
for i in range(1000):
    msgtimesofar = 0.0
    for msg in mid:
        msgtimesofar += msg.time           # note: not tm
        print("msg=",msg," sofar=",msgtimesofar)
        print("msg.bytes=",msg.bytes())
        arr = msg.bytes()
        ch = arr[0] & 0xf
        layer = layers[ch%4]
        print("arr = ",arr)
        s = ""
        for b in arr:
            s += ("%02x" % b)
        print("s = ",s)
        tosleep = msg.time * playbacktimefactor
        time.sleep(tosleep)

        if msg.type == "note_on" or msg.type == "note_off":
            doNote(msg,msgtimesofar,layer)

