import mido
import palette
import time

mid = mido.MidiFile('../default/midifiles/cdef.mid')
msgtimesofar = 0.0
playbacktimefactor = 0.5
xtimefactor = 0.1
cursorid = "one"
mididevice = "testmidi"

def doNote(msg,msgtimesofar):
    if msg.velocity == 0 or msg.type == "note_off":
        cursorevent = "up"
    else:
        cursorevent = "down"
    x = msgtimesofar * xtimefactor
    y = msg.note / 128.0
    print("cursorevent=",cursorevent," x=",x," y=",y)
    palette.SendCursorEvent(cursorid,cursorevent,x,y,0.5)
    palette.SendMIDIEvent(mididevice,msg)

for msg in mid:
    msgtimesofar += msg.time           # note: not tm
    print("msg=",msg," sofar=",msgtimesofar)
    print("msg.bytes=",msg.bytes())
    arr = msg.bytes()
    print("arr = ",arr)
    s = ""
    for b in arr:
        s += ("%02x" % b)
    print("s = ",s)
    tosleep = msg.time * playbacktimefactor
    time.sleep(tosleep)

    if msg.type == "note_on" or msg.type == "note_off":
        doNote(msg,msgtimesofar)

