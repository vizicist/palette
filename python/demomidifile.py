import mido
import sys
import os
import palette
import time

timesofar = 0.0

for msg in mido.MidiFile('../data/midifiles/prelude.mid'):
    timesofar += msg.time
    time.sleep(msg.time * 0.5)  # play it twice as fast
    # Send each note's MIDI
    palette.SendMIDIEvent("",timesofar,msg)
    # Generate a sprite in piano-roll placement for each note_on
    if msg.type == "note_on":
        # time in X direction, pitch in Y direction (centered around pitch 64)
        x = ( timesofar * 0.1 ) % 1.0
        y = ( 0.5 + (msg.note - 64) * 0.01 ) % 1.0
        palette.SendSpriteEvent("0",x,y,0.5)

