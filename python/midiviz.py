import mido
import palette
import time

mid = mido.MidiFile('../default/midifiles/midiviz.mid')
playbacktimefactor = 0.5
mididevice = "testmidi"

def doNote(msg,msgtimesofar,region):
    palette.SendMIDIEvent(mididevice,msgtimesofar,msg,region)

reg = 0
regions = "ABCD"
for i in range(1000):
    msgtimesofar = 0.0
    for msg in mid:
        msgtimesofar += msg.time           # note: not tm
        print("msg=",msg," sofar=",msgtimesofar)
        print("msg.bytes=",msg.bytes())
        arr = msg.bytes()
        ch = arr[0] & 0xf
        region = "ABCD"[ch%4]
        print("arr = ",arr)
        s = ""
        for b in arr:
            s += ("%02x" % b)
        print("s = ",s)
        tosleep = msg.time * playbacktimefactor
        time.sleep(tosleep)

        if msg.type == "note_on" or msg.type == "note_off":
            doNote(msg,msgtimesofar,region)

