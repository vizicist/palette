import json
import glob
import sys
import os

def changeprefix(paramfile,prefix):

    f = open(paramfile)
    params = json.load(f)
    f.close()

    for name in list(params["params"]):
        print("Changing parmname=",name)
        params["params"][prefix + "." + name] = params["params"][name]
        del params["params"][name]

    print("Writing ",paramfile)
    f = open(paramfile,"w")
    f.write(json.dumps(params, sort_keys=True, indent=4, separators=(',',':'))) 
    f.write("\n")
    f.close()

if len(sys.argv) < 2:
    print("usage: %s [paramdir] [prefix]" % sys.argv[0])
    sys.exit(1)

paramdir = sys.argv[1]
prefix = sys.argv[2]

files = glob.glob(os.path.join(paramdir,'*.json'))
for s in files:
    print("file = ",s)
    changeprefix(s,prefix)

