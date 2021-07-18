import json
import glob
import sys
import os

def changeparam(paramfile,paramname,paramnewname):
    f = open(paramfile)
    params = json.load(f)
    f.close()

    for ch in {"A","B","C","D"}:
        chname = ch + "-" + paramname
        if chname in params["params"]:
            params["params"][paramnewname] = params["params"][chname]
            del params["params"][chname]

    if paramname in params["params"]:
        print("Changing parmname=",paramname)
        params["params"][paramnewname] = params["params"][paramname]
        del params["params"][paramname]

    print("Writing ",paramfile)
    f = open(paramfile,"w")
    f.write(json.dumps(params, sort_keys=True, indent=4, separators=(',',':'))) 
    f.write("\n")
    f.close()

if len(sys.argv) < 4:
    print("usage: %s [paramdir] [paramname] [oldval] [newval]" % sys.argv[0])
    sys.exit(1)

paramdir = sys.argv[1]
oldname = sys.argv[2]
newname = sys.argv[3]

files = glob.glob(os.path.join(paramdir,'*.json'))
for s in files:
    print("file = ",s)
    changeparam(s,oldname,newname)

