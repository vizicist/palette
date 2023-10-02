# utility to migrate from old huefill{initial,final,time} to hue1{initial,final,time} and hue2

import json
import glob
import sys
import os

def changehue(paramfile):
    f = open(paramfile)
    params = json.load(f)
    f.close()

    p = params["params"]

    if "huefillfinal" in p:

        filled = p["filled"]

        if filled == "true":
            huefinal = p["huefillfinal"]
            hueinitial = p["huefillinitial"]
            huetime = p["huefilltime"]
        else:
            huefinal = p["huefinal"]
            hueinitial = p["hueinitial"]
            huetime = p["huetime"]

        p["hue1final"] = huefinal
        p["hue1initial"] = hueinitial
        p["hue1time"] = huetime
        p["hue2final"] = huefinal
        p["hue2initial"] = hueinitial
        p["hue2time"] = huetime

        del p["huefillfinal"]
        del p["huefillinitial"]
        del p["huefilltime"]
        del p["huefinal"]
        del p["hueinitial"]
        del p["huetime"]

    else:
        for c in {"A_","B_","C_","D_"}:
            filled = p[c+"filled"]
            if filled == "true":
                huefinal = p[c+"huefillfinal"]
                hueinitial = p[c+"huefillinitial"]
                huetime = p[c+"huefilltime"]
            else:
                huefinal = p[c+"huefinal"]
                hueinitial = p[c+"hueinitial"]
                huetime = p[c+"huetime"]

            p[c+"hue1final"] = huefinal
            p[c+"hue1initial"] = hueinitial
            p[c+"hue1time"] = huetime
            p[c+"hue2final"] = huefinal
            p[c+"hue2initial"] = hueinitial
            p[c+"hue2time"] = huetime
    
            del p[c+"huefillfinal"]
            del p[c+"huefillinitial"]
            del p[c+"huefilltime"]
            del p[c+"huefinal"]
            del p[c+"hueinitial"]
            del p[c+"huetime"]
 
    print("Writing ",paramfile)
    f = open(paramfile,"w")
    f.write(json.dumps(params, sort_keys=True, indent=4, separators=(',',':'))) 
    f.write("\n")
    f.close()

homedir = os.getenv("PALETTE_SOURCE")

files = glob.glob(os.path.join(homedir,"default","presets","visual",'*.json'))
for s in files:
    print("file = ",s)
    changehue(s)

files = glob.glob(os.path.join(homedir,"default","presets","snap",'*.json'))
for s in files:
    print("file = ",s)
    changehue(s)

