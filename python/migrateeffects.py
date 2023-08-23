# utility to migrate from old huefill{initial,final,time} to hue1{initial,final,time} and hue2
# No longer used/valid, but kept around in case it's useful

import json
import glob
import sys
import os

def changeeffects(paramfile):
    f = open(paramfile)
    params = json.load(f)
    f.close()

    p = params["params"]
    effectonoff = {
 "twisted",
"huerotate",
"kaleid",
"blur",
"mirrorquad",
"fragment",
"posterize",
"displace",
"edgedetection",
"mirror",
"trails"
    }

    for name in {
"twisted",
"twisted:twirl",
"twisted:radius",

"huerotate",
"huerotate:huescale",
"huerotate:huerotate",
"huerotate:satscale",

"kaleid",
"kaleid:angles",
"kaleid:rotation",
"kaleid:zoom",
"kaleid:inputrotation",
"kaleid:opacity",

"blur",
"blur:blurxdistance",
"blur:blurydistance",
"blur:gaussian",
"blur:quality",

"mirrorquad",
"mirrorquad:flipx",
"mirrorquad:flipy",

"fragment",
"fragment:fragments",
"fragment:distance",
"fragment:rotation",
"fragment:fragrotationx",
"fragment:fragrotationy",
"fragment:fragrotationz",
"fragment:fragscale",

"posterize",
"posterize:posterize",

"displace",
"displace:xfactor",
"displace:yfactor",
"displace:threshold",

"edgedetection",
"edgedetection:mode",
"edgedetection:multiplier",

"mirror",
"mirror:x",
"mirror:y",
"mirror:in/out",
"mirror:flipx",
"mirror:flipy",

"trails",
"trails:feedback" }:
        # print(name)

        for ch in {"A","B","C","D"}:

            # special case for kaleid, which used an obsolete method for having 2 kaleids
            if name == "kaleid":
                oldname1 = ch+"_kaleid1"
                oldname2 = ch+"_kaleid2"
                if oldname1 in p:
                    newname1 = ch+"_1-kaleid"
                    newname2 = ch+"_2-kaleid"
                    p[newname1] = p[oldname1]
                    p[newname2] = "false"
                    del p[oldname1]
                    del p[oldname2]

            elif name in effectonoff:
                oldname1 = ch+"_"+name
                if oldname1 in p:
                    newname1 = ch+"_1-"+name
                    newname2 = ch+"_2-"+name
                    p[newname1] = p[oldname1]
                    p[newname2] = "false"
                    del p[oldname1]

            elif name.startswith("kaleid:"):
                suffix = name[7:]
                oldname1 = ch+"_kaleid1:"+suffix
                oldname2 = ch+"_kaleid2:"+suffix
                if oldname1 in p:
                    newname1 = ch+"_1-kaleid:" + suffix
                    newname2 = ch+"_2-kaleid:" + suffix
                    p[newname1] = p[oldname1]
                    p[newname2] = p[oldname1]
                    del p[oldname1]
                    del p[oldname2]
            elif (ch+"_"+name) in p:
                oldname = ch+"_"+name
                if oldname in p:
                    newname1 = ch+"_1-"+name
                    newname2 = ch+"_2-"+name
                    p[newname1] = p[oldname]
                    p[newname2] = p[oldname]
                    del p[oldname]
            else:
                print("HEY! name="+name+" not handled? is it a new parameter?")

        if name == "kaleid":
            oldname1 = "kaleid1"
            oldname2 = "kaleid2"
            if oldname1 in p:
                newname1 = "1-kaleid"
                newname2 = "2-kaleid"
                p[newname1] = p[oldname1]
                p[newname2] = p[oldname2]
                del p[oldname1]
                del p[oldname2]

        elif name in effectonoff:
            oldname1 = name
            if oldname1 in p:
                newname1 = "1-"+name
                newname2 = "2-"+name
                p[newname1] = p[oldname1]
                p[newname2] = "false"
                del p[oldname1]

        elif name.startswith("kaleid:"):
            suffix = name[7:]
            oldname1 = "kaleid1:"+suffix
            oldname2 = "kaleid2:"+suffix
            if oldname1 in p:
                newname1 = "1-kaleid:" + suffix
                newname2 = "2-kaleid:" + suffix
                p[newname1] = p[oldname1]
                p[newname2] = p[oldname2]
                del p[oldname1]
                del p[oldname2]

        elif name in p:
            oldname = name
            if oldname in p:
                newname1 = "1-"+name
                newname2 = "2-"+name
                p[newname1] = p[oldname]
                p[newname2] = p[oldname]
                del p[oldname]
        else:
            print("HEY! name="+name+" not handled? is it a new parameter?")


    newparamfile = paramfile.replace("saved_original","saved_original_converted")
    print("Writing ",newparamfile)
    f = open(newparamfile,"w")
    f.write(json.dumps(params, sort_keys=True, indent=4, separators=(',',':'))) 
    f.write("\n")
    f.close()

homedir = os.getenv("PALETTE_SOURCE")

print("HEY!  THIS SCRIPT IS NO LONGER VALID")
sys.exit(1)

# This can be used on either "effect" or "patch"

files = glob.glob(os.path.join(homedir,"data","saved_original","effect",'*.json'))
for s in files:
    print("file = ",s)
    changeeffects(s)

