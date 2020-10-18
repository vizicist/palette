import json
import glob
import sys
import os

def listport(paramfile):
    f = open(paramfile)
    params = json.load(f)
    f.close()
    p = params["params"]
    path = os.path.split(paramfile)
    fname = path[len(path)-1]
    print "%s \"%s\" %d" % (fname, p["regionport"]["value"], int(p["regionchannel"]["value"]))

if len(sys.argv) < 1:
    print "usage: %s [paramdir]" % sys.argv[0]
    sys.exit(1)

paramdir = sys.argv[1]

files = glob.glob(os.path.join(paramdir,'*.json'))
for s in files:
    listport(s)

