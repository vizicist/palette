import json
import glob
import sys
import os

def changeparam(paramfile,paramname,oldval,newval):
    f = open(paramfile)
    params = json.load(f)
    f.close()

    for ch in {"A","B","C","D"}:
        chname = ch + "_" + paramname
        if chname in params["params"]:
            if params["params"][chname] == oldval:
                print("Changing chname=",chname)
                params["params"][chname] = str(newval)

    if paramname in params["params"]:
        if params["params"][paramname] == oldval:
            print("Changing parmname=",paramname)
            params["params"][paramname] = str(newval)

    print("Writing ",paramfile)
    f = open(paramfile,"w")
    f.write(json.dumps(params, sort_keys=True, indent=4, separators=(',',':'))) 
    f.write("\n")
    f.close()

if len(sys.argv) < 4:
    print("usage: %s [paramdir] [paramname] [oldval] [newval]" % sys.argv[0])
    sys.exit(1)

paramdir = sys.argv[1]
paramname = sys.argv[2]
oldval = sys.argv[3]
newval = sys.argv[4]

files = glob.glob(os.path.join(paramdir,'*.json'))
for s in files:
    print("file = ",s)
    changeparam(s,paramname,oldval,newval)

