# This script generates the source code and Visual Studio project
# for a new plugin.  The PluginTemplate directory contains the 
# basis for it.

import sys
import os
import re
import hashlib
import argparse

def generate(pluginname,pluginid,pathin,pathout):

    print("Generating "+pathout)

    fin = open(pathin,"r")
    fout = open(pathout,"w")

    lines = fin.readlines()

    for ln in lines:
        ln = ln.replace("{PLUGINNAME}",pluginname)
        ln = ln.replace("{PLUGINID}",pluginid)
        fout.write(ln)

    fout.close()
    fin.close()

if __name__ != "__main__":
    print("This code needs to be invoked as a main program.")
    sys.exit(1)

parser = argparse.ArgumentParser("Generate a new FFGL plugin project")
parser.add_argument("name", help="FFGL plugin name")
parser.add_argument("-i", "--id", help="4-character FFGL plugin ID")
parser.add_argument("-f", "--force",
		help="Force overwriting of an existing project",
		action="store_true")

args = parser.parse_args()

pluginname = args.name
id = args.id
force = args.force

if not pluginname[0].isupper():
    print("The plugin name needs to start with an uppercase letter!")
    sys.exit(1)

curdir = os.getcwd();
if os.path.basename(curdir) != "scripts":
    print("This script must be invoked from the scripts directory!")
    sys.exit(1)

homedir = os.path.dirname(curdir)

os.chdir(homedir)

projdir = os.path.join("build","windows")
srcdir = os.path.join("source","plugins","PluginTemplate")
blddir = "binaries"

print("projdir=%s srcdir=%s blddir=%s" % (projdir,srcdir,blddir))

# Make sure various directories exist

if not os.path.isdir(blddir):
    print("Error: "+blddir+" directory doesn't exist!")
    sys.exit(1)

if not os.path.isdir(projdir):
    print("Error: "+projdir+" doesn't exist!")
    sys.exit(1)

if not os.path.isdir(srcdir):
    print("Error: "+srcdir+" doesn't exist!")
    sys.exit(1)

tosrcdir = os.path.join("source","plugins",pluginname)

if force == False and os.path.exists(tosrcdir):
    print("The directory "+tosrcdir+" already exists!")
    sys.exit(1)

if not id:
    id = "S%03d" % (int(hashlib.sha1(pluginname).hexdigest(), 16) % (10**3))
    print("Generated 4-character FFGL plugin ID is '"+id+"'")
else:
    id = id

if len(id) != 4:
    print("The plugin id (%s) needs to be exactly 4 characters long!"%id)
    sys.exit(1)

if not os.path.exists(tosrcdir):
    os.mkdir(tosrcdir)
    if not os.path.isdir(tosrcdir):
        print("Unable to make directory: ",tosrcdir)
        sys.exit(1)

print("========== Generating FFGL plugin:%s id:%s" % (pluginname,id))

tosrcdir = os.path.join("source","plugins",pluginname)

generate(pluginname, id,
        os.path.join(projdir,"PluginTemplate.vcxproj"),
        os.path.join(projdir,pluginname+".vcxproj"))

generate(pluginname, id,
        os.path.join(projdir,"PluginTemplate.vcxproj.filters"),
        os.path.join(projdir,pluginname+".vcxproj.filters"))

generate(pluginname, id,
        os.path.join(srcdir,"FFGLPluginTemplate"+".cpp"),
        os.path.join(tosrcdir,"FFGL"+pluginname+".cpp"))

generate(pluginname, id,
        os.path.join(srcdir,"FFGLPluginTemplate"+".h"),
        os.path.join(tosrcdir,"FFGL"+pluginname+".h"))

sys.exit(0)
