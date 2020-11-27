#ifndef NOSUCHGRAPHICS_H
#define NOSUCHGRAPHICS_H

class NosuchVector {
public:
	NosuchVector() {
		// set(FLT_MAX,FLT_MAX,FLT_MAX);
		set(0.0,0.0);
	}
	NosuchVector(float xx, float yy) {
		set(xx,yy);
	};
	void set(float xx, float yy) {
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
	float mag() {
		return sqrt( (x*x) + (y*y) );
	}
	NosuchVector normalize() {
		float leng = mag();
		return NosuchVector(x/leng, y/leng);
	}
	NosuchVector mult(float m) {
		return NosuchVector(x*m,y*m);
	}
	NosuchVector rotate(float radians, NosuchVector about = NosuchVector(0.0f,0.0f) ) {
		float c, s;
		c = cos(radians);
		s = sin(radians);
		x -= about.x;
		y -= about.y;
		float newx = x * c - y * s;
		float newy = x * s + y * c;
		newx += about.x;
		newy += about.y;
		return NosuchVector(newx,newy);
	}
	float heading() {
        return -atan2(-y, x);
	}

	float x;
	float y;
};

#endif
