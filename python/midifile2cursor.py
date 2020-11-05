import mido
import palette
import time

mid = mido.MidiFile('../default/midifiles/prelude.mid')
msgtimesofar = 0.0
playbacktimefactor = 0.5
xtimefactor = 0.1
cid = "one"

def doNote(msg,msgtimesofar):
    if msg.velocity == 0 or msg.type == "note_off":
        cursorevent = "up"
    else:
        cursorevent = "down"
    x = msgtimesofar * xtimefactor
    y = msg.note / 128.0
    print("cursorevent=",cursorevent," x=",x," y=",y)
    palette.SendCursorEvent(cid,cursorevent,x,y,0.5)

for msg in mid:
    msgtimesofar += msg.time           # note: not tm
    print("msg=",msg," sofar=",msgtimesofar)
    tosleep = msg.time * playbacktimefactor
    time.sleep(tosleep)

    if msg.type == "note_on" or msg.type == "note_off":
        doNote(msg,msgtimesofar)

