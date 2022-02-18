import sys
import os
import re

def despace(v):
	v = v.replace(",","")
	v = v.replace("\"","")
	v = v.replace("\t","")
	v = v.replace(" ","")
	return v

def convert(fname,outdir):

	try:
		f = open(fname)
	except:
		print("Unable to open "+fname)
		sys.exit(1)

	lines = f.readlines()
	print("There are %d lines" % len(lines))

	globals = {}
	overrides = {}
	regions = []

	state = ""
	regionid = -1
	for ln in lines:
		# print("ln=",ln)
		if re.match(".*\"global\".*",ln):
			state = "global"
			continue
		if re.match(".*\"overrides\".*",ln):
			state = "overrides"
			continue
		if re.match(".*\"id\".*\"params\".*",ln):
			state = "regions"
			regionid += 1
			regions.append({})
			continue

		if state == "global":
			if re.match(".*}",ln):
				state = ""
				continue
			m = re.match(".*\"(.*)\": (.*),*$",ln)
			# print("ln=",ln," m[0]=",m.group(0)," m[1]=",m.group(1))
			globals[m.group(1)] = despace(m.group(2))
			continue

		if state == "overrides":
			if re.match(".*}",ln):
				state = ""
				continue
			m = re.match(".*\"([a-z]*)\": (.*),*$",ln)
			if len(m.groups()) < 2:
				print("state overrides, didn't match? ln=",ln)
				continue
			v = m.group(2)
			v = despace(v)
			# print("ln=",ln," m[1]=",m.group(1)," m[2]=",v)
			overrides[m.group(1)] = v
			continue

		if state == "regions":
			if re.match(".*{,.*",ln) or re.match(".*].*",ln):
				state = ""
				continue
			if re.match(".*{.*",ln) or re.match(".*}.*",ln):
				continue
			m = re.match(".*\"([a-z]*)\": (.*),*$",ln)
			if m==None or len(m.groups()) < 2:
				print("state regions, didn't match? ln=",ln)
				continue
			v = m.group(2)
			v = despace(v)
			# print("ln=",ln," m[1]=",m.group(1)," m[2]=",v)
			regions[regionid][m.group(1)] = v

	i = fname.find(".")
	if i >= 0:
		outname = fname[0:i] + ".plt"
	else:
		outname = fname + ".plt"
	outname = outdir + "/" + outname
	print("outname=%s"%outname)
	try:
		outf = open(outname,"w")
	except:
		print("Unable to open "+outname)
		sys.exit(1)
	outf.write("{\n")
	outf.write("\"global\": {\n")
	sep = "\t"
	for v in globals:
		outf.write("%s\"%s\": \"%s\""%(sep,v,globals[v]))
		sep = ",\n\t"
	outf.write("\t},\n")
	outf.write("\"overrideparams\": {\n")
	sep = "\t"
	for v in overrides:
		outf.write("%s\"%s\": \"%s\""%(sep,v,overrides[v]))
		sep = ",\n\t"
	outf.write("\n\t},\n")
	outf.write("\"overrideflags\": {\n")
	sep = "\t"
	for v in overrides:
		val = overrides[v]
		if val != "-999.000000" and val != "UNSET":
			override = "true"
		else:
			override = "false"
		outf.write("%s\"%s\": \"%s\""%(sep,v,override))
		sep = ",\n\t"

	outf.write("\n\t},\n")
	outf.write("\"regions\": [\n")
	sep = "\t"
	regionnum = -1
	for vals in regions:
		regionnum += 1
		outf.write("%s{ \"id\": %d,\n"%(sep,regionnum))
		outf.write("\t\"regionspecificparams\": {\n")
		sep2 = "\t\t";
		for v in vals:
			outf.write("%s\"%s\": \"%s\"" % (sep2,v,vals[v]))
			sep2 = ",\n\t\t";
		outf.write("\n\t\t}\n")
		sep = "\t},\n\t"

	outf.write("\n\t}\n\t]\n}\n")
	outf.close()

if __name__ == "__main__":

	for fname in os.listdir("."):
		if re.match(".*\.plt$",fname):
			print("fname="+fname)
			outdir = "../params"
			convert(fname,outdir)

