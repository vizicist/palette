import json
import glob
import sys
import os

def changeparam(paramfile,paramname,newval):
    f = open(paramfile)
    params = json.load(f)
    f.close()

    if not paramname in params["params"]:
        print "Hey, param ",paramname," isn't in the params!?"
        sys.exit(1)

    params["params"][paramname]["value"] = str(newval)

    print "Writing ",paramfile
    f = open(paramfile,"w")
    f.write(json.dumps(params, sort_keys=True, indent=4, separators=(',',':'))) 
    f.close()

if len(sys.argv) < 3:
    print "usage: %s [paramdir] [paramname] [newval]" % sys.argv[0]
    sys.exit(1)

paramdir = sys.argv[1]
paramname = sys.argv[2]
newval = sys.argv[3]

files = glob.glob(os.path.join(paramdir,'*.json'))
for s in files:
    print "file = ",s
    changeparam(s,paramname,newval)

