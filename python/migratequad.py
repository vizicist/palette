# utility to migrate from old huefill{initial,final,time} to hue1{initial,final,time} and hue2

import json
import glob
import sys
import os

quadfile = sys.argv[1]
defsfile = "../../../data/config/paramdefs.json"

df = open(defsfile, 'r')
qf = open(quadfile, 'r')
of = open(quadfile+".new",'w')

deflines = df.readlines()
quadlines = qf.readlines()

paramtype = {}
for line in deflines:
    if line[0] != "\"":
        continue
    di = line.find(".")
    rest = line[di+1:]
    qi = rest.find("\"")
    category = line[1:di]
    param = rest[0:qi]
    paramtype[param] = category
    if category == "effect":
        paramtype["1-"+param] = category
        paramtype["2-"+param] = category


count = 0
for line in quadlines:
    # print("line=",line)
    i = line.find("-")
    if i < 0 :
        of.write(line)
    else :
        rest = line[i+1:]
        iq = rest.find("\"")
        paramname = rest[0:iq]
        ptype = paramtype[paramname]
        newparamname = ptype + "." + paramname
        part1 = line[0:i+1]
        part2 = newparamname
        part3 = rest[iq:]
        of.write(part1+part2+part3)
        # print("Writing "+part1+part2+part3)

df.close()
qf.close()
of.close()

ffrom = open(quadfile+".new","r")
fto = open(quadfile,"w")
fromlines = ffrom.readlines()
for line in fromlines:
    fto.write(line)
ffrom.close()
fto.close()
