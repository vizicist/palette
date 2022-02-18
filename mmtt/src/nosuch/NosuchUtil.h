/*
	Space Manifold - a variety of tools for Kinect and FreeFrame

	Copyright (c) 2011-2012 Tim Thompson <me@timthompson.com>

	Permission is hereby granted, free of charge, to any person obtaining
	a copy of this software and associated documentation files
	(the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge,
	publish, distribute, sublicense, and/or sell copies of the Software,
	and to permit persons to whom the Software is furnished to do so,
	subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	Any person wishing to distribute modifications to the Software is
	requested to send the modifications to the original developer so that
	they can be incorporated into the canonical version.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
	MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR
	ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
	CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
	WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

#ifndef NOSUCHUTIL_H
#define NOSUCHUTIL_H

#include <stdint.h>
#include <pthread.h>
#include "NosuchDebug.h"

#ifndef WIN32
#include <sys/errno.h>
// #define WS_VERSION_REQUIRED 0x0101
// #define WS_VERSION_REQUIRED MAKEWORD(1, 1)

#endif

////////// MEMORY LEAK DEBUGGING
// #define MEMORY_LEAK_DEBUGGING
#ifdef MEMORY_LEAK_DEBUGGING
// #define _CRTDBG_MAP_ALLOC   I put it in the properties
#include <stdlib.h>
#include <crtdbg.h>
#else
#include <stdlib.h>
#endif

#include <string>
#include <vector>

#if 0
#ifdef _DEBUG
#define DEBUG_NEW new(_NORMAL_BLOCK, __FILE__, __LINE__)
#define new DEBUG_NEW
#endif
#endif

#ifdef WIN32
#define _USE_MATH_DEFINES
#endif
#include <math.h>

#ifndef mymin
#define mymin(a,b) ((a) < (b) ? (a) : (b))
#endif

#ifndef mymax
#define mymax(a,b) ((a) > (b) ? (a) : (b))
#endif

#ifndef TRUE
#define TRUE 1
#endif
#ifndef FALSE
#define FALSE 0
#endif

#define SLIP_END 192
#define SLIP_ESC 219
#define SLIP_ESC2 221

#define IS_SLIP_END(c) (((c)&0xff)==SLIP_END)
#define IS_SLIP_ESC(c) (((c)&0xff)==SLIP_ESC)
#define IS_SLIP_ESC2(c) (((c)&0xff)==SLIP_ESC2)

#define NTHEVENTSERVER_PORT 1384
// Every so many milliseconds, we re-register with the Nth Server
#define NTHEVENTSERVER_REREGISTER_MILLISECONDS 3000

int SendToUDPServer(std::string host, int serverport, const char *data, int leng);
int SendToSLIPServer(std::string host, int serverport, const char *data, int leng);
int SlipBoundaries(char *p, int leng, char** pbegin, char** pend);

#define PARAM_DISPLAY_LEN 16

#define RAD2DEG(r) ((r)*360.0/(2.0*M_PI))
#define PI2 ((float)(2.0*M_PI))

#ifndef WIN32
#define _snprintf snprintf
#endif

// typedef const char * std::string;

#define NOSUCHMAXSTR 1024

void NosuchPrintTime(const char *prefix);

// void NosuchDebug(const char *fmt, ...);
// std::string NosuchSnprintf(const char *fmt, ...);
// void NosuchFree(std::string s);
int NosuchNetworkInit();

std::vector<std::string> NosuchSplitOnAnyChar(std::string s, std::string sepchars);
std::vector<std::string> NosuchSplitOnString(const std::string& s, const std::string& delim, const bool keep_empty);
std::string NosuchToLower(std::string s);

char *base64_encode(const uint8_t *data, size_t input_length);

std::string error_json(int code, const char *msg, const char *id = "null");
std::string ok_json(const char *id);

std::string ToNarrow( const wchar_t *s, char dfault = '?', const std::locale& loc = std::locale() );
std::wstring s2ws(const std::string& s);
std::string ws2s(const std::wstring& s);

void NosuchLockInit(pthread_mutex_t* mutex, char *tag);
void NosuchLock(pthread_mutex_t* mutex, char *tag);
void NosuchUnlock(pthread_mutex_t* mutex, char *tag);
int NosuchTryLock(pthread_mutex_t* mutex, char *tag);

class NosuchDrawInfo {
    public:
	double x;
	double y;
	double vel;
	double velang;
	double scalex;
	double scaley;
	double alpha;
	double hue;
	double handlex;
	double handley;
	double rotation;
	// double cosrot;
	// double sinrot;
	double linewidth;
};

class HLS {
    public:
	HLS(double h, double l, double s) {
		_hue = h * 360.0;
		_luminance = l;
		_saturation = s;
		_red = 0.0;
		_green = 0.0;
		_blue = 0.0;
		_ToRGB();
	}

	double red() { return _red/255.0; }
	double green() { return _green/255.0; }
	double blue() { return _blue/255.0; }

	double hue() { return _hue; }
	double luminance() { return _luminance; }
	double saturation() { return _saturation; }

	void setrgb(double r, double g, double b) {
		_red = r * 255.0;
		_green = g * 255.0;
		_blue = b * 255.0;
		_ToHLS();
	}

	void sethls(double h, double l, double s) {
		_hue = h * 360.0;
		_luminance = l;
		_saturation = s;
		_ToRGB();
	}

	void _ToHLS() {
		double minval = mymin(_red, mymin(_green, _blue));
		double maxval = mymax(_red, mymax(_green, _blue));
		double mdiff = maxval-minval;
		double msum = maxval + minval;
		_luminance = msum / 510;
		if ( maxval == minval ) {
			_saturation = 0.0;
			_hue = 0.0;
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
			// _hue %= 360;
			_hue = fmod(_hue,360.0);
		}
	}

	void _ToRGB() {
		if ( _saturation == 0 ) {
			_red = _green = _blue = _luminance * 255;
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

	double _ToRGB1(double rm1, double rm2, double rh) {
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
		return(rm1 * 255);
	}

    private:
	double _hue;
	double _luminance;
	double _saturation;
	double _red;
	double _green;
	double _blue;
};

#if 0
// typedef struct ParamDynamicDataStructTag {
// 	float value;
// 	char displayValue[PARAM_DISPLAY_LEN+1];
// } ParamDynamicDataStruct;

typedef struct VideoPixel24bitTag {
	BYTE blue;
	BYTE green;
	BYTE red;
} VideoPixel24bit;

typedef struct VideoPixel16bitTag {
	BYTE fb;
	BYTE sb;
} VideoPixel16bit;

typedef struct VideoPixel32bitTag {
	BYTE blue;
	BYTE green;
	BYTE red;
	BYTE alpha;
} VideoPixel32bit;

typedef struct VideoParamInfoTag {
	char *name;
	float defaultval;
	int type;
} VideoParamInfo;

typedef struct VideoParamDataTag {
	float value;
	char display[PARAM_DISPLAY_LEN];
} VideoParamData;

CvScalar randomRGB();

CvScalar HLStoRGB(float hue, float lum, float sat);
void RGBtoHLS(float r, float g, float b, float* hue, float* lum, float* sat);
float ToRGB1(float rm1, float rm2, float rh);
float angleNormalize(float a);
unsigned char *createimageofsize(CvSize sz);

class NSSprite {
	static int lastid;

public:
	NSSprite(CvPoint2D32f pt) {
		center = pt;
		age = 0.0;
		speedx = 0.0;
		speedy = 0.0;
		size = 1.0;
		generation = 0;
		velocity = 0.0;
		tracked = 0;
		moveang = 0.0;
		id = lastid++;
		// NosuchDebug("NEW SPRITE id=%d center=%lf,%lf this=%lld",id,center.x,center.y,(long long)this);
	}
	void increaseAge() {
		age += 1.0;
	}
	int tooOld() {
		if ( age > 100.0 ) {
			// NosuchDebug("tooOld returning TRUE, age=%lf for this=%lld",age,(long long)this);
			return true;
		}
		else
			return false;
	}
	void trackPointLost() {
		velocity = 0.0;
		tracked = 0;
	}

	CvPoint2D32f	center;
	float			speedx;
	float			speedy;
	float			velocity;
	float			moveang;
	float			size;
	float			age;
	int				id;
	int				generation;
	int				tracked;
	NSSprite *next;
};

#endif

#endif
