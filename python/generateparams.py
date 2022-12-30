#
# This script takes a Params.list file and generates all the .h files
# for Layer and Sprite parameters.  This allows new parameters to be added
# just by editing the Params.list file and re-running this script.
# Originally the parameters were represented in generalized lists,
# but looking them up at execution time was expensive (as determined
# by profilng), so now the parameters are all explicit members
# of the structures. 

import sys
import os
import re
import json

def openFile(dir,name):
	return open(os.path.join(dir,name),"w")

def generate(homedir, force, sourcedir, floatType):

	c = os.path.join(homedir,"data_omnisphere/config")

	#########################################
	## First process paramdefs.json
	#########################################

	jsonfile = os.path.join(c,"paramdefs.json")
	enumfile = os.path.join(c,"paramenums.json")

	# if generated file is more recent than the definitions, do nothing
	jsonmtime = os.path.getmtime(jsonfile)
	enummtime = os.path.getmtime(enumfile)

	try:
		outputmtime = os.path.getmtime(os.path.join(sourcedir,"LayerParams_declare.h"))
		jsonupdated = outputmtime < jsonmtime 
		enumupdated = outputmtime < enummtime
	except:
		jsonupdated = True
		enumupdated = True

	if not force and not jsonupdated and not enumupdated:
		return

	if force:
		print("Generateparams: update forced due to -f argument")
	else:
		if enumupdated:
			print("Generateparams: update due to change in %s" % (enumfile))
		if jsonupdated:
			print("Generateparams: update due to change in %s" % (jsonfile))

	try:
		f = open(jsonfile)
	except:
		print("Unable to open "+jsonfile)
		sys.exit(1)

	out_rp_declare = openFile(sourcedir,"LayerParams_declare.h")
	out_rp_get = openFile(sourcedir,"LayerParams_get.h")
	out_rp_increment = openFile(sourcedir,"LayerParams_increment.h")
	out_rp_init = openFile(sourcedir,"LayerParams_init.h")
	out_rp_issprite = openFile(sourcedir,"LayerParams_issprite.h")
	out_rp_list = openFile(sourcedir,"LayerParams_list.h")
	out_rp_set = openFile(sourcedir,"LayerParams_set.h")
	out_rp_toggle = openFile(sourcedir,"LayerParams_toggle.h")

	j = json.load(f)

	for name in j:
		namewords = name.split(".")
		if len(namewords) != 2:
			print("Unable to handle paramdefs.json name=",name)
			continue
		paramtype = namewords[0]
		basename = namewords[1]
		typ = j[name]["valuetype"]
		mn = j[name]["min"]
		mx = j[name]["max"]
		paramtype = paramtype
		init = j[name]["init"]
		# comment = j[name]["comment"]
		if typ == "string":
			init = "\"" + init + "\""
		
		types={"bool":"BOOL","int":"INT",floatType:"DBL","string":"STR"}
		rtypes={"bool":"bool","int":"int",floatType:floatType,"string":"std::string"}
		captype = types[typ]
		realtype = rtypes[typ]

		is_layer_param = (paramtype == "sprite" or paramtype == "visual" or paramtype == "NO_MORE_SOUND_PARAMS_sound")
		
		if realtype == "float" and floatType == "float":
			fsuffix = "f"
		else:
			fsuffix = ""

		if is_layer_param:

			out_rp_declare.write("%s %s;\n"%(realtype,basename))
			out_rp_get.write("GET_%s_PARAM(%s);\n"%(captype,basename))

			out_rp_init.write("INIT_PARAM ( %s , %s%s );\n"%(basename,init,fsuffix))
			out_rp_list.write("\"%s\",\n"%basename)
			out_rp_set.write("SET_%s_PARAM(%s);\n"%(captype,basename))

			if typ == "bool":
				out_rp_increment.write("INC_%s_PARAM(%s);\n"%(captype,basename))
				out_rp_toggle.write("TOGGLE_PARAM(%s);\n"%basename)
			elif typ == "int" or typ == floatType:
				out_rp_increment.write("INC_%s_PARAM(%s,%s%s,%s%s);\n"%(captype,basename,mn,fsuffix,mx,fsuffix))
			elif typ == "string":
				if mn == "None":
					out_rp_increment.write("INC_NO_PARAM(%s);\n"%(basename))
				else:
					# The mn value is the Types array
					out_rp_increment.write("INC_%s_PARAM(%s,%s);\n"%(captype,basename,mn))
			else:
				print("Unrecognized paramtype: %s" % typ)

		if paramtype == "sprite" or paramtype == "visual":
			out_rp_issprite.write("IS_SPRITE_PARAM(%s);\n"%basename)

	out_rp_declare.close()
	out_rp_get.close()
	out_rp_increment.close()
	out_rp_init.close()
	out_rp_issprite.close()
	out_rp_list.close()
	out_rp_set.close()
	out_rp_toggle.close()

	########################################
	## Now process paramenums.json
	########################################

	try:
		enumf = open(enumfile)
	except:
		print("Unable to open "+enumfile)
		sys.exit(1)

	out_rp_types = openFile(sourcedir,"LayerParams_types.h")
	out_rp_typesdeclare = openFile(sourcedir,"LayerParams_typesdeclare.h")
	enumj = json.load(enumf)

	for name in enumj:
		out_rp_typesdeclare.write("DECLARE_TYPES(%s);\n"%(name))
		out_rp_types.write("DEFINE_TYPES(%s);\n"%(name))

	# define the LayerParams_Initializeypes() function
	out_rp_types.write("\n")
	out_rp_types.write("void\n")
	out_rp_types.write("LayerParams_InitializeTypes() {\n")

	for name in enumj:
		arr = enumj[name]
		out_rp_types.write("\n")
		for a in arr:
			out_rp_types.write("\tLayerParams_%sTypes.push_back(\"%s\");\n"%(name,a))

	out_rp_types.write("};\n")

	out_rp_types.close()
	out_rp_typesdeclare.close()


if __name__ == "__main__":

    homedir = os.getenv("PALETTESOURCE")

    force = False
    sourcedir = os.path.join(homedir,"ffgl/source/lib/palette")
    ftype = "float"
    if len(sys.argv) > 1:
        for a in sys.argv[1:]:
            if a == "-f":
                force = True

    print("Generateparams: checking source in: "+sourcedir)
    generate(homedir,force,sourcedir,ftype)

    sys.exit(0)
