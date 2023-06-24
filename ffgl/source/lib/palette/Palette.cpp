#include <iostream>
#include <fstream>

#include  <io.h>
#include  <stdlib.h>

#include <time.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <stdio.h>
#include <errno.h>

#include <sys/types.h>
#include <fcntl.h>
#include <errno.h>
#include <vector>
#include <string>
#include <iostream>

#include <cstdlib> // for srand, rand

#include "PaletteAll.h"

const double Palette::UNSET_DOUBLE = -999.0f;
const std::string Palette::UNSET_STRING = "UNSET";
Palette* Palette::_singleton = NULL;
const std::string Palette::configSuffix = ".plt";
const std::string Palette::configSeparator = "\\";
int Palette::lastsprite = 0;
int Palette::now = 0;

bool Palette::initialized = false;

void Palette::initialize() {
	if ( initialized )
		return;

	initialized = true;

	NosuchDebug(1,"END OF INITIALIZING globalParams and others!\n");
}

Palette::Palette(PaletteHost* b) {

	NosuchLockInit(&_palette_mutex,"palette");
	_paletteHost = b;
	_drawer      = new PaletteDrawer( &params );

	now = 0;  // Don't use Pt_Time(), it may not have been started yet

	_frames = 0;
	_frames_last = now;

	if ( _singleton != NULL ) {
		// It gets instantiated a bunch of times,
		// when Resolume is starting.
		NosuchDebug(1,"Palette is instantiated again!?");
	}
	_singleton = this;
}

Palette::~Palette() {
}

FFResult Palette::InitGL( const FFGLViewportStruct* vp)
{
	return _drawer->InitGL( vp );
}

FFResult Palette::DeInitGL()
{
	_drawer->DeInitGL();
	return FF_SUCCESS;
}


static void writestr(std::ofstream& out, std::string s) {
	const char* p = s.c_str();
	out.write(p,s.size());
}

static std::string debugJson(cJSON *j, int indent) {
	std::string s = std::string(indent,' ').c_str();
	switch (j->type) {
	case cJSON_False:
		s += NosuchSnprintf("%s = False\n",j->string);
		break;
	case cJSON_True:
		s += NosuchSnprintf("%s = True\n",j->string);
		break;
	case cJSON_NULL:
		s += NosuchSnprintf("%s = NULL\n",j->string);
		break;
	case cJSON_Number:
		s += NosuchSnprintf("%s = (number) %.3f\n",j->string,j->valuedouble);
		break;
	case cJSON_String:
		s += NosuchSnprintf("%s = (string) %s\n",j->string,j->valuestring);
		break;
	case cJSON_Array:
		s += NosuchSnprintf("%s = (array)\n",j->string);
		for ( cJSON* j2=j->child; j2!=NULL; j2=j2->next ) {
			for ( cJSON* j3=j2->child; j3!=NULL; j3=j3->next ) {
				s += debugJson(j3,indent+3);
			}
		}
		break;
	case cJSON_Object:
		s += NosuchSnprintf("%s = object\n",j->string==NULL?"NULL":j->string);
		for ( cJSON* j2=j->child; j2!=NULL; j2=j2->next ) {
			s += debugJson(j2,indent+3);
		}
		break;
	default:
		s += NosuchSnprintf("Unable to handle JSON type=%d in debugJSON?\n",j->type);
		break;
	}
	return s;
}

std::string jsonValueString(cJSON* j) {
	std::string val;

	switch (j->type) {
	case cJSON_Number:
		val = NosuchSnprintf("%f",j->valuedouble);
		break;
	case cJSON_String:
		val = j->valuestring;
		break;
	default:
		throw NosuchBadValueException();
	}
	return val;
}

std::string Palette::loadParamPushReal(cJSON* sound, cJSON* visual)
{
	cJSON* j;

	for ( j=sound->child; j!=NULL; j=j->next ) {
		std::string key = j->string;
		std::string val = jsonValueString(j) ;
		if ( NosuchDebugParam == TRUE ) {
			NosuchDebug("loadParamsPushReal sound %s %s\n", key.c_str(), val.c_str());
		}
		layer.params.Set(key,val);
	}
	for ( j=visual->child; j!=NULL; j=j->next ) {
		std::string key = j->string;
		std::string val = jsonValueString(j) ;
		if ( NosuchDebugParam == TRUE ) {
			NosuchDebug("loadParamsPushReal visual %s %s\n", key.c_str(), val.c_str());
		}
		layer.params.Set(key,val);
	}
	// ResetLayerParams();
	return "";
}

void Palette::clear() {
	NosuchDebug(1,"===================== Palette::clear");
	LockPalette();
	layer.clear();
	UnlockPalette();
}

void Palette::advanceTo(int tm) {

	NosuchDebug(3,"===================== Palette::advanceTo tm=%d setting now",tm);
	now = tm;
	LockPalette();
	layer.advanceTo(now);
	layer.deleteOldCursors(this);
	UnlockPalette();

	if (params.showfps) {
		_frames++;
		// Every second, print out FPS
		if (now > (_frames_last + 1000)) {
			NosuchDebug("FPS=%d  now=%d",_frames,now);
			_frames = 0;
			_frames_last = now;
		}
	}
}

// public float random(int n) {
// return app.random(n);
// }

int Palette::draw() {

	// pthread_t thr = pthread_self ();
	// NosuchDebug("Palette::draw start thr=%d,%d",(int)(thr.p),thr.x);

	layer.draw(_drawer);

	return 0;
}

int Palette::drawbg() {
	layer.drawbg(_drawer);
	return 0;
}


#include "NosuchColor.h"

void Palette::LoadParamPush(cJSON* sound, cJSON* visual) {

	_paletteHost->lock_paletteHost();
	std::string r = loadParamPushReal(sound, visual);
	_paletteHost->unlock_paletteHost();
	if (r != "") {
		throw NosuchUnableToLoadException();
	}
}
