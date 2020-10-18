import json
import glob
import sys
import os

def makeset(paramfile):
    f = open(paramfile)
    j = json.load(f)
    f.close()
    base = os.path.basename(paramfile).replace("_A.json","")

    finalparams = { "params": {
        "patchA": { "value": (base+"_A"), "enabled": "true" },
        "patchB": { "value": (base+"_B"), "enabled": "true" },
        "patchC": { "value": (base+"_C"), "enabled": "true" },
        "patchD": { "value": (base+"_D"), "enabled": "true" },
        }
    }

    newfile = os.path.join("..","set",base + ".json")
    print("base=",base," parms=",finalparams," newfile=",newfile)
    f = open(newfile,"w")
    f.write(json.dumps(finalparams, sort_keys=True, indent=4, separators=(',',':')))
    # To avoid complaints from editors, add a final newline
    f.write("\n")
    f.close()

paramdir = "."

files = glob.glob(os.path.join(paramdir,'*_A.json'))
for s in files:
    makeset(s)

