import mido
import os
import sys
import palette
import time
import argparse

#########################################3

def doNote2Cursor(args,timesofar,msg):
    if msg.type == "note_off":
        cursorevent = "up"
    else:
        cursorevent = "down"
    x = timesofar * args.timefactor
    y = msg.note / 128.0
    cid = "one"
    palette.SendCursorEvent(cid,cursorevent,x,y,0.5)

def doNote2Sprite(args,timesofar,msg):
    if msg.type != "note_on":
        return

    # do piano-roll-like positioning - time in X direction, pitch in Y
    x = timesofar * args.xfactor
    midpitch = 64
    dpitch = (msg.note - midpitch)
    y = 0.5 + dpitch * args.yfactor
    # print("note=",msg.note," dpitch=",dpitch," y=",y)

    x = x % 1.0
    y = y % 1.0

    cid = "one"
    palette.SendSpriteEvent(cid,x,y,0.5)

def doNote2MIDI(args,timesofar,msg):
    palette.SendMIDIEvent(args.device,timesofar,msg)

def doNote2SpriteMIDI(args,timesofar,msg):
    doNote2MIDI(args,timesofar,msg)
    doNote2Sprite(args,timesofar,msg)

def doNote2Debug(args,timesofar,msg):
    print("%f : %s" % (timesofar,str(msg)) )

def process_midifile(args,mid,notefunc):

    timesofar = 0.0
    for msg in mid:
        if msg.type == "note_on" and msg.velocity == 0:
            msg.type == "note_off"
        timesofar += msg.time
        tosleep = msg.time * float(args.timefactor)
        time.sleep(tosleep)
    
        if msg.type == "note_on" or msg.type == "note_off":
            notefunc(args,timesofar,msg)

#########################################3

if __name__=="__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-i", "--id", help="TBD")
    parser.add_argument("-t", "--timefactor", type=float, help="Playback time factor", default=1.0)
    parser.add_argument("-x", "--xfactor", type=float, help="X factor for sprite placement", default=0.1)
    parser.add_argument("-y", "--yfactor", type=float, help="Y factor for sprite placement", default=0.01)
    parser.add_argument("-v", "--verbosity", action="count", default=0)
    parser.add_argument("-d", "--device", default="palette_midifile")
    parser.add_argument("midifile")
    parser.add_argument("generate")

    args = parser.parse_args()

    mid = mido.MidiFile(os.path.join('../default/midifiles/',args.midifile))

    if args.generate == "cursor":
        dofunc = doNote2Cursor
    elif args.generate == "midi":
        dofunc = doNote2MIDI
        palette.SendMIDITimeReset()
    elif args.generate == "sprite":
        dofunc = doNote2Sprite
    elif args.generate == "spritemidi":
        dofunc = doNote2SpriteMIDI
    elif args.generate == "debug":
        dofunc = doNote2Debug
    else:
        print("Invalid value of generate (%s)\n" % args.generate)
        sys.exit(1)

    process_midifile(args,mid,dofunc)
    sys.exit(0)
