
#include "PaletteUtil.h"
#include "NosuchDebug.h"

int
string2int(std::string s) {
	return atoi(s.c_str());
}

float
string2double(std::string s) {
	return (float) atof(s.c_str());
}

bool
string2bool(std::string s) {
	if ( s == "" ) {
		NosuchDebug("Unexpected empty value in string2bool!?");
		return false;
	}
	if ( s == "true" || s == "True" || s[0] == '1' || s == "on" ) {
		return true;
	} else {
		return false;
	}
}

