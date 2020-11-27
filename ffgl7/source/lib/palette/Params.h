#ifndef _PARAMS_H
#define _PARAMS_H

#define IntString(x) NosuchSnprintf("%d",x)
#define DoubleString(x) NosuchSnprintf("%f",x)
#define FloatString(x) NosuchSnprintf("%f",x)
#define BoolString(x) NosuchSnprintf("%s",x?"on":"off")

class Params {
public:
	float adjust(float v, float amount, float vmin, float vmax) {
		v += amount*(vmax-vmin);
		if ( v < vmin )
			v = vmin;
		else if ( v > vmax )
			v = vmax;
		return v;
	}
	int adjust(int v, float amount, int vmin, int vmax) {
		int incamount = (int)(amount*(vmax-vmin));
		if ( incamount != 0 ) {
			incamount = (amount>0.0) ? 1 : -1;
		}
		v = (int)(v + incamount);
		if ( v < vmin )
			v = vmin;
		else if ( v > vmax )
			v = vmax;
		return v;
	}
	bool adjust(bool v, float amount) {
		if ( amount > 0.0 ) {
			return true;
		}
		if ( amount < 0.0 ) {
			return false;
		}
		// if amount is 0.0, no change.
		return v;
	}
	std::string adjust(std::string v, float amount, std::vector<std::string>& vals) {
		// Find the existing value
		size_t existing = 0;
		size_t sz = vals.size();
		if ( sz == 0 ) {
			throw NosuchException("vals array is empty!?");
		}
		// Even if amount is 0.0, we want to make sure v is one of the valid values
		for ( size_t ei=0; ei<sz; ei++ ) {
			if ( v == vals[ei] ) {
				existing = ei;
				break;
			}
		}
		if (amount == 0.0) {
			return vals[existing];
		}
		// Return the next or previous value in the list
		size_t i = existing + ((amount>0.0)?1:-1);
		if ( i < 0 ) {
			i = (int)sz - 1;
		}
		if ( i >= sz ) {
			i = 0;
		}
		return vals[i];
	}
	virtual std::string Get(std::string) = 0;
	std::string JsonList(char* names[], std::string pre, std::string indent, std::string post) {
		std::string s = pre;
		std::string sep = indent;
		for ( char** nm=names; *nm; nm++ ) {
			s += (sep + "\"" + *nm + "\": \"" + Get(*nm) + "\"");
			sep = ",\n"+indent;
		}
		s += post;
		return s;
	}
};

#endif