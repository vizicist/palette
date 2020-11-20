#ifndef NOSUCHCOLOR_H
#define NOSUCHCOLOR_H

class NosuchColor {
	
public:
	NosuchColor() {
		setrgb(0,0,0);
	}

	NosuchColor(int r_, int g_, int b_) {
		setrgb(r_,g_,b_);
	}
	
	NosuchColor(double h, double l, double s) {
		h = fmod( h , 360.0);
		sethls(h,l,s);
	}

	void setrgb(int r,int g,int b) {
		_red = r;
		_green = g;
		_blue = b;
		_ToHLS();
	}

	void sethls(double h,double l,double s) {
		if ( h > 360.0 ) {
			NosuchDebug("Hey, hue in NosuchColor > 360?");
		}
		_hue = h;
		_luminance = l;
		_saturation = s;
		_ToRGB();
	}
	
	int r() { return _red; }
	int g() { return _green; }
	int b() { return _blue; }

	int _min(int a, int b) {
		if ( a < b )
			return a;
		else
			return b;
	};

	int _max(int a, int b) {
		if ( a > b )
			return a;
		else
			return b;
	};

	void _ToHLS() {
		double minval = _min(_red, _min(_green, _blue));
		double maxval = _max(_red, _max(_green, _blue));
		double mdiff = maxval-minval;
		double msum = maxval + minval;
		_luminance = msum / 510;
		if ( maxval == minval ) {
			_saturation = 0;
			_hue = 0;
		} else {
			double rnorm = (maxval - _red) / mdiff;
			double gnorm = (maxval - _green) / mdiff;
			double bnorm = (maxval - _blue) / mdiff;
			if ( _luminance <= .5 ) {
				_saturation = mdiff/msum;
			} else {
				_saturation = mdiff / (510 - msum);
			}
			// _saturation = (_luminance <= .5) ? (mdiff/msum) : (mdiff / (510 - msum));
			if ( _red == maxval ) {
				_hue = 60 * (6 + bnorm - gnorm);
			} else if ( _green == maxval ) {
				_hue = 60 * (2 + rnorm - bnorm);
			} else if ( _blue == maxval ) {
				_hue = 60 * (4 + gnorm - rnorm);
			}
			_hue = fmod(_hue, 360.0);
		}
	}

	void _ToRGB() {
		if ( _saturation == 0 ) {
			_red = _green = _blue = (int)(_luminance * 255);
		} else {
			double rm2;
			if ( _luminance <= 0.5 ) {
				rm2 = _luminance + _luminance * _saturation;
			} else {
				rm2 = _luminance + _saturation - _luminance * _saturation;
			}
			double rm1 = 2 * _luminance - rm2;
			_red = _ToRGB1(rm1, rm2, _hue + 120);
			_green = _ToRGB1(rm1, rm2, _hue);
			_blue = _ToRGB1(rm1, rm2, _hue - 120);
		}
	}

	int _ToRGB1(double rm1, double rm2, double rh) {
		if ( rh > 360 ) {
			rh -= 360;
		} else if ( rh < 0 ) {
			rh += 360;
		}

		if ( rh < 60 ) {
			rm1 = rm1 + (rm2 - rm1) * rh / 60;
		} else if ( rh < 180 ) {
			rm1 = rm2;
		} else if ( rh < 240 ) {
			rm1 = rm1 + (rm2 - rm1) * (240 - rh) / 60;
		}
		return (int)(rm1 * 255);
		
	}

private:
	int _red;  // 0 to 255
	int _green;  // 0 to 255
	int _blue;  // 0 to 255
	double _hue;   // 0.0 to 360.0
	double _luminance;  // 0.0 to 1.0
	double _saturation;  // 0.0 to 1.0

};

#endif
