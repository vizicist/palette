#ifndef NOSUCHGRAPHICS_H
#define NOSUCHGRAPHICS_H

#include <math.h>
#include <float.h>

#define RADIAN2DEGREE(r) ((r) * 360.0 / (2.0 * (double)M_PI))
#define DEGREE2RADIAN(d) (((d) * 2.0 * (double)M_PI) / 360.0 )

// #define CHECK_VECTOR
#ifdef CHECK_VECTOR
void checkVector(NosuchVector v) {
	if ( _isnan(v.x) || _isnan(v.y) ) {
		NosuchDebug("checkVector found NaN!");
	}
	if ( v.x > 10.0 || v.y > 10.0 ) {
		NosuchDebug("checkVector found > 10.0!");
	}
}
#else
#define checkVector(v)
#endif

class NosuchVector {
public:
	NosuchVector() {
		// set(FLT_MAX,FLT_MAX,FLT_MAX);
		set(0.0,0.0);
	}
	NosuchVector(double xx, double yy) {
		set(xx,yy);
	};
	void set(double xx, double yy) {
		x = xx;
		y = yy;
	}
	bool isnull() {
		return ( x == FLT_MAX && y == FLT_MAX );
	}
	bool isequal(NosuchVector p) {
		return ( x == p.x && y == p.y );
	}
	NosuchVector sub(NosuchVector v) {
		return NosuchVector(x-v.x,y-v.y);
	}
	NosuchVector add(NosuchVector v) {
		return NosuchVector(x+v.x,y+v.y);
	}
	double mag() {
		return sqrt( (x*x) + (y*y) );
	}
	NosuchVector normalize() {
		double leng = mag();
		return NosuchVector(x/leng, y/leng);
	}
	NosuchVector mult(double m) {
		return NosuchVector(x*m,y*m);
	}
	NosuchVector rotate(double radians, NosuchVector about = NosuchVector(0.0f,0.0f) ) {
		double c, s;
		c = cos(radians);
		s = sin(radians);
		x -= about.x;
		y -= about.y;
		double newx = x * c - y * s;
		double newy = x * s + y * c;
		newx += about.x;
		newy += about.y;
		return NosuchVector(newx,newy);
	}
	double heading() {
        return -atan2(-y, x);
	}

	double x;
	double y;
};

#endif
