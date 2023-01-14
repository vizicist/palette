import json
import glob
import sys
import os

def changeparam(paramfile,paramname,oldval,newval):
    f = open(paramfile)
    params = json.load(f)
    f.close()

    for pad in {"A","B","C","D"}:

        # In a quad, sound and visual parameters are {pad}-{paramname}
        fullname = pad + "-" + paramname
        if fullname in params["params"]:
            if params["params"][fullname] == oldval:
                print(paramfile,": Changing param=",fullname)
                params["params"][fullname] = str(newval)

        # ... except for effect parameters which are {pad}-{number}-{paramname}
        for num in {"1","2"}:
            fullname = pad + "-" + num + "-" + paramname
            if fullname in params["params"]:
                if params["params"][fullname] == oldval:
                    print(paramfile,": Changing param=",fullname)
                    params["params"][fullname] = str(newval)

    # In other saved things, the parameter is just the {paramname}
    if paramname in params["params"]:
        if params["params"][paramname] == oldval:
            print("Changing parmname=",paramname)
            params["params"][paramname] = str(newval)

    # ... except for effect parameters which are {number}-{paramname}
    for num in {"1","2"}:
        fullname = num + "-" + paramname
        if fullname in params["params"]:
            if params["params"][fullname] == oldval:
                print(paramfile,": Changing param=",fullname)
                params["params"][fullname] = str(newval)

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

