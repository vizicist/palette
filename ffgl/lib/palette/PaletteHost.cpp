
#include <iostream>
#include <sstream>
#include <fstream>
#include <strstream>
#include <cstdlib> // for srand, rand
#include <ctime>   // for time
#include <sys/stat.h>

// to get M_PI
#define _USE_MATH_DEFINES
#include <math.h>

#include "PaletteAll.h"
#include "FFGLLib.h"
#include "FFGLPalette.h"
#include "osc/OscOutboundPacketStream.h"

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Plugin information
////////////////////////////////////////////////////////////////////////////////////////////////////

bool PaletteHost::StaticInitialized = false;
void* PaletteHost::ThreadPointer = NULL;
int PaletteHost::PortOffset = 0;

extern "C"
{

bool
ffgl_setdll(std::string dllpath)
{
	NosuchDebugSetThreadName(pthread_self().p,"FFGLDLL");

	dllpath = NosuchToLower(dllpath);

	size_t lastslash = dllpath.find_last_of("/\\");
	size_t lastdot = dllpath.find_last_of(".");
	std::string suffix = (lastdot==dllpath.npos?"":dllpath.substr(lastdot));

	if ( suffix != ".dll" ) {
		NosuchDebug("Hey! dll name (%s) isn't of the form *.dll!?",dllpath.c_str());
		return FALSE;
	}

	std::string dir = dllpath.substr(0,lastslash);
	std::string prefix = dllpath.substr(lastslash+1,lastdot-lastslash-1);

	// If there's a trailing underscore and a number in the name (e.g. Palette_2.dll)
	// then use the number as the port offset
	size_t lastunder = dllpath.find_last_of("_");
	
	if (lastunder != dllpath.npos) {
		int n = atoi(dllpath.substr(lastunder + 1).c_str());
		NosuchDebug("Setting PortOffset to %d for dll=%s",n,dllpath.c_str());
		PaletteHost::PortOffset = n;
		NosuchDebugTag = n;	// used as an additional tag on debug output
	}

	NosuchCurrentDir = dir;

	char* p = getenv("LOCALAPPDATA");
	if ( p != NULL ) {
		NosuchDebugLogPath = std::string(p) + "\\Palette\\logs\\ffgl.log";
	}
	else {
		NosuchDebugLogPath = "c:\\windows\\temp\\ffgl.log"; // last resort
	}
	// NosuchDebug("LogPath = %s\n", NosuchDebugLogPath.c_str());

	struct _stat statbuff;
	int e = _stat(NosuchCurrentDir.c_str(),&statbuff);
	if ( ! (e == 0 && (statbuff.st_mode | _S_IFDIR) != 0) ) {
		NosuchDebug("Hey! No directory %s!?",NosuchCurrentDir.c_str());
		return FALSE;
	}

	return TRUE;
}
}

void *daemon_threadfunc(void *arg)
{
	PaletteDaemon* b = (PaletteDaemon*)arg;
	return b->run(arg);
}

// This depends on the actual values of the cJSON_* macros
const char* jsonType(int t) {
	switch (t) {
	case cJSON_False: return "False";
	case cJSON_True: return "True";
	case cJSON_NULL: return "NULL";
	case cJSON_Number: return "Number";
	case cJSON_String: return "String";
	case cJSON_Array: return "Array";
	case cJSON_Object: return "Object";
	}
	return "Unknown";
}

cJSON* needItem(std::string who, cJSON* j, std::string nm, int type) {
	cJSON *c = cJSON_GetObjectItem(j,nm.c_str());
	if ( ! c ) {
		throw NosuchException("Missing %s item in %s",nm.c_str(),who.c_str());
	}
	if ( c->type != type ) {
		throw NosuchException("Unexpected type for %s item in %s, expecting %s",nm.c_str(),who.c_str(),jsonType(type));
	}
	return c;
}

std::string needString(std::string who,cJSON *j,std::string nm, std::string dflt = "") {
	// return needItem(who, j, nm, cJSON_String)->valuestring;
	cJSON *c = cJSON_GetObjectItem(j,nm.c_str());
	if (c) {
		if ( c->type != cJSON_String ) {
			throw NosuchException("Unexpected type for %s item in %s, expecting string", nm.c_str(), who.c_str());
		}
		return c->valuestring;
	}
	else {
		return dflt;
	}
}

int needNumber(std::string who,cJSON *j,std::string nm, int dflt = 0) {
	cJSON *c = cJSON_GetObjectItem(j,nm.c_str());
	if (c) {
		if ( c->type != cJSON_Number ) {
			throw NosuchException("Unexpected type for %s in %s, expecting number", nm.c_str(), who.c_str());
		}
		return c->valueint;
	}
	else {
		return dflt;
	}
}

bool needBool(std::string who,cJSON *j,std::string nm) {
	cJSON *c = cJSON_GetObjectItem(j,nm.c_str());
	if (!c) {
		throw NosuchException("%s: Missing value for '%s'", who.c_str(), nm.c_str());
	}
	if ( c->type == cJSON_Number ) {
		return (c->valueint != 0);
	}
	if ( c->type == cJSON_String ) {
		std::string v = c->valuestring;
		return (v=="1" || v=="true" || v=="True" || v=="on" || v=="On");
	}
	throw NosuchException("Unexpected type for %s item in %s, expecting %s",nm.c_str(),who.c_str(),jsonType(c->type));
}

double needDouble(std::string who,cJSON *j,std::string nm) {
	return needItem(who, j, nm, cJSON_Number)->valuedouble;
}

cJSON* needArray(std::string who,cJSON *j,std::string nm) {
	return needItem(who, j, nm, cJSON_Array);
}

cJSON* needObject(std::string who,cJSON *j,std::string nm) {
	return needItem(who, j, nm, cJSON_Object);
}

void needParams(std::string meth, cJSON* params) {
	if(params==NULL) {
		throw NosuchException("No parameters on %s method?",meth.c_str());
	}
}

PaletteDaemon::PaletteDaemon(PaletteHost* mf, int osc_input_port, std::string osc_input_host)
{
	NosuchDebug(2,"PaletteDaemon CONSTRUCTOR!");

	_paletteHost = mf;
	_network_thread_created = false;
	daemon_shutting_down = false;

	if ( osc_input_port < 0 ) {
		NosuchDebug("NOT CREATING _oscinput!! because osc_input_port<0");
		_oscinput = NULL;
	} else {
		NosuchDebug(2,"CREATING _oscinput and PaletteServer!!");
		_oscinput = new PaletteOscInput(mf,osc_input_host.c_str(),osc_input_port);
		_oscinput->Listen();
	}

	NosuchDebug(2,"About to use pthread_create in PaletteDaemon");
	int err = pthread_create(&_network_thread, NULL, daemon_threadfunc, this);
	if (err) {
		NosuchDebug("pthread_create failed!? err=%d\n",err);
		NosuchErrorOutput("pthread_create failed!?");
	} else {
		_network_thread_created = true;
		// NosuchDebug("PaletteDaemon is running");
	}

	_morphs = NULL;

#ifdef EMBEDDED_MORPH_SUPPORT
	std::map<std::string, std::string> serialmap;

	float morphforce = 0.5f;
	std::string morphopt = "SM01172912315:13000,SM01172912292:14000,SM01172912306:11000,SM01172912176:12000";
	std::vector<std::string> morphspecs = NosuchSplitOnString(morphopt, ",", false);
	for (auto& x : morphspecs) {
		NosuchDebug("x=%s", x.c_str());
		std::vector<std::string> words = NosuchSplitOnString(x, ":", false);
		if (words.size() != 2) {
			NosuchDebug("Bad format of morph option: %s", x.c_str());
		}
		else {
			serialmap.insert(std::pair<std::string, std::string>(words[0], words[1]));
		}
	}

	_morphs = new AllMorphs(serialmap);
	if (_morphs->init()) {
		NosuchDebug("Morph successfully initialized");
	}
	else {
		NosuchDebug("Morph NOT initialized!");
		_morphs = NULL;
	}
#endif
}

PaletteDaemon::~PaletteDaemon()
{
	NosuchDebug(1,"PaletteDaemon DESTRUCTOR starts!");
	daemon_shutting_down = true;
	if ( _network_thread_created ) {
		// pthread_detach(_network_thread);
		pthread_join(_network_thread,NULL);
	}

	if ( _oscinput ) {
		NosuchDebug(1,"PaletteDaemon destructor, removing processor from _oscinput!");
		_oscinput->UnListen();
		delete _oscinput;
		NosuchDebug(1,"PaletteDaemon destructor, after removing processor from _oscinput!");
		_oscinput = NULL;
	}

	NosuchDebug(1,"PaletteDaemon DESTRUCTOR ends!");
}

void *PaletteDaemon::run(void *arg)
{
	NosuchDebugSetThreadName(pthread_self().p, "PaletteDaemon");
	int textcount = 0;
	while (daemon_shutting_down == false ) {
		_paletteHost->RunEveryMillisecondOrSo();
		if ( _oscinput ) {
			_oscinput->Check();
		}
#ifdef EMBEDDED_MORPH_SUPPORT
		if (_morphs) {
			_morphs->poll();
		}
#endif
		Sleep(1);
	}
	return NULL;
}

void PaletteHost::StaticInitialization()
{
	if ( StaticInitialized ) {
		return;
	}
	StaticInitialized = true;

	srand((unsigned)time(NULL));

	// Default debugging stuff
	NosuchDebugLevel = 0;   // 0=minimal messages, 1=more, 2=extreme
	NosuchDebugTimeTag = true;
	NosuchDebugToLog = true;
	NosuchAppName = "Palette";
#ifdef DEBUG_TO_BUFFER
	NosuchDebugToBuffer = true;
#endif
	NosuchDebugAutoFlush = true;
	
	NosuchDebug(1,"=== PaletteHost Static Initialization!");
}

PaletteHost::PaletteHost(std::string configfile)
{
	NosuchDebugSetThreadName(pthread_self().p,"PaletteHost");

	NosuchDebug(1,"=== PaletteHost is being constructed.");

	_configFile = configfile;
	_configJson = NULL;
	_dotest = FALSE;
	_lastActivity = -1;
	_time0 = timeGetTime();
	_activityEnabled = FALSE;

	_daemon = NULL;

	initialized = false;
	gl_shutting_down = false;
	gl_frame = 0;

	width = 1.0f;
	height = 1.0f;

	// Don't do any OpenGL calls here, it isn't initialized yet.

	NosuchLockInit(&json_mutex,"json");
	json_cond = PTHREAD_COND_INITIALIZER;
	json_pending = false;

	NosuchLockInit(&palette_mutex,"palette");

	m_filled = false;
	m_stroked = false;
	
	disabled = false;
	disable_on_exception = false;

	// Config file can override those values
	std::ifstream f;

	f.open(_configFile.c_str());
	if ( ! f.good() ) {
		std::string err = NosuchSnprintf("No config file (%s), assuming defaults\n",_configFile.c_str());
		NosuchDebug("%s",err.c_str());  // avoid re-interpreting %'s and \\'s in name
		return;
	}

	// NosuchDebug("Loading config=%s\n",_configFile.c_str());
	std::string line;
	std::string jstr;
	while ( getline(f,line) ) {
		if ( line.size()>0 && line.at(0)=='#' ) {
			NosuchDebug(1,"Ignoring comment line=%s\n",line.c_str());
			continue;
		}
		jstr += line;
	}
	f.close();

	_configJson = cJSON_Parse(jstr.c_str());
	if ( ! _configJson ) {
		std::string msg = NosuchSnprintf("Unable to parse json for config!?  json= %s\n",jstr.c_str());
		NosuchDebug(msg.c_str());
		if ( NosuchErrorPopup != NULL ) {
			NosuchErrorPopup(msg.c_str());
		}
		return;
	}


	LoadPaletteConfig(_configJson);

	_scheduler = new Scheduler(this);
}

PaletteHost::~PaletteHost()
{
	NosuchDebug(1,"PaletteHost destructor called");
	gl_shutting_down = true;
	if (_scheduler) {
		scheduler()->Stop();
		delete _scheduler;
		_scheduler = NULL;
	}

	if ( _daemon != NULL ) {
		delete _daemon;
		_daemon = NULL;
	}
	NosuchDebug(1,"PaletteHost destructor end");
}

void PaletteHost::ErrorPopup(const char* msg) {
		MessageBoxA(NULL,msg,"Palette",MB_OK);
}

static cJSON *
getNumber(cJSON *json,char *name)
{
	cJSON *j = cJSON_GetObjectItem(json,name);
	if ( j && j->type == cJSON_Number )
		return j;
	return NULL;
}

static cJSON *
getString(cJSON *json,char *name)
{
	cJSON *j = cJSON_GetObjectItem(json,name);
	if ( j && j->type == cJSON_String && j->valuestring != NULL )
		return j;
	return NULL;
}

static bool
istrue(std::string s)
{
	return(s == "true" || s == "True" || s == "1");
}

void
PaletteHost::LoadPaletteConfig(cJSON* c)
{
	cJSON *j;

	if ( (j=getString(c,"debugcursor")) != NULL ) {
		NosuchDebugCursor = istrue(j->valuestring);
	}
	if ( (j=getNumber(c,"debugsprite")) != NULL ) {
		NosuchDebugSprite = istrue(j->valuestring);
	}
	if ( (j=getNumber(c,"debugsprite")) != NULL ) {
		NosuchDebugSprite = istrue(j->valuestring);
	}
	if ( (j=getNumber(c,"debugautoflush")) != NULL ) {
		NosuchDebugAutoFlush = istrue(j->valuestring);
	}
}

int PaletteHost::SendToResolume(osc::OutboundPacketStream& p) {
	NosuchDebug(1,"SendToResolume host=%s port=%d",DEFAULT_RESOLUME_HOST,DEFAULT_RESOLUME_PORT);
    return SendToUDPServer(DEFAULT_RESOLUME_HOST,DEFAULT_RESOLUME_PORT,p.Data(),(int)p.Size());
}

void
PaletteHost::RunEveryMillisecondOrSo() {
	Timestamp tm = timeGetTime() - _time0;
	if (_scheduler) {
		_scheduler->RunEveryMillisecondOrSo(tm);
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Methods
////////////////////////////////////////////////////////////////////////////////////////////////////

void PaletteHost::rect(double x, double y, double w, double h) {
	// if ( w != 2.0f || h != 2.0f ) {
	// 	NosuchDebug("Drawing rect xy = %.3f %.3f  wh = %.3f %.3f",x,y,w,h);
	// }
	quad(x,y, x+w,y,  x+w,y+h,  x,y+h);
}
void PaletteHost::fill(NosuchColor c, double alpha) {
	m_filled = true;
	m_fill_color = c;
	m_fill_alpha = alpha;
}
void PaletteHost::stroke(NosuchColor c, double alpha) {
	// glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, alpha);
	m_stroked = true;
	m_stroke_color = c;
	m_stroke_alpha = alpha;
}
void PaletteHost::noStroke() {
	m_stroked = false;
}
void PaletteHost::noFill() {
	m_filled = false;
}
void PaletteHost::background(int b) {
	NosuchDebug("PaletteHost::background!");
}
void PaletteHost::strokeWeight(double w) {
	glLineWidth((GLfloat)w);
}
void PaletteHost::rotate(double degrees) {
	glRotated(-degrees,0.0f,0.0f,1.0f);
}
void PaletteHost::translate(double x, double y) {
	glTranslated(x,y,0.0f);
}
void PaletteHost::scale(double x, double y) {
	glScaled(x,y,1.0f);
	// NosuchDebug("SCALE xy= %f %f",x,y);
}
void PaletteHost::quad(double x0, double y0, double x1, double y1, double x2, double y2, double x3, double y3) {
	NosuchDebug(2,"   Drawing quad = %.3f %.3f, %.3f %.3f, %.3f %.3f, %.3f %.3f",x0,y0,x1,y1,x2,y2,x3,y3);
	if ( m_filled ) {
		glBegin(GL_QUADS);
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1); 
		glVertex2d( x2, y2); 
		glVertex2d( x3, y3); 
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		glBegin(GL_LINE_LOOP); 
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1); 
		glVertex2d( x2, y2); 
		glVertex2d( x3, y3); 
		glEnd();
	}
	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, quad() called when both m_filled and m_stroked are off!?");
	}
}
void PaletteHost::triangle(double x0, double y0, double x1, double y1, double x2, double y2) {
	NosuchDebug(2,"Drawing triangle xy0=%.3f,%.3f xy1=%.3f,%.3f xy2=%.3f,%.3f",x0,y0,x1,y1,x2,y2);
	if ( m_filled ) {
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		NosuchDebug(2,"   fill_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_fill_alpha);
		glBegin(GL_TRIANGLE_STRIP); 
		glVertex3d( x0, y0, 0.0f );
		glVertex3d( x1, y1, 0.0f );
		glVertex3d( x2, y2, 0.0f );
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		NosuchDebug(2,"   stroke_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_stroke_alpha);
		glBegin(GL_LINE_LOOP); 
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1);
		glVertex2d( x2, y2);
		glEnd();
	}
	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, triangle() called when both m_filled and m_stroked are off!?");
	}
}

void PaletteHost::line(double x0, double y0, double x1, double y1) {
	// NosuchDebug("Drawing line xy0=%.3f,%.3f xy1=%.3f,%.3f",x0,y0,x1,y1);
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		// NosuchDebug(2,"   stroke_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_stroke_alpha);
		glBegin(GL_LINES); 
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1);
		glEnd();
	} else {
		NosuchDebug("Hey, line() called when m_stroked is off!?");
	}
}

static double degree2radian(double deg) {
	return 2.0f * (double)M_PI * deg / 360.0f;
}

void PaletteHost::ellipse(double x0, double y0, double w, double h, double fromang, double toang) {
	NosuchDebug(2,"Drawing ellipse xy0=%.3f,%.3f wh=%.3f,%.3f",x0,y0,w,h);
	if ( m_filled ) {
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		NosuchDebug(2,"   fill_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_fill_alpha);
		glBegin(GL_TRIANGLE_FAN);
		double radius = w;
		glVertex2d(x0, y0);
		for ( double degree=fromang; degree <= toang; degree+=5.0f ) {
			glVertex2d(x0 + sin(degree2radian(degree)) * radius, y0 + cos(degree2radian(degree)) * radius);
		}
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		NosuchDebug(2,"   stroke_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_stroke_alpha);
		if (fromang == 0.0 && toang == 360.0) {
			glBegin(GL_LINE_LOOP);
		} else {
			glBegin(GL_LINES);
		}
		double radius = w;
		for ( double degree=fromang; degree <= toang; degree+=5.0f ) {
			glVertex2d(x0 + sin(degree2radian(degree)) * radius, y0 + cos(degree2radian(degree)) * radius);
		}
		glEnd();
	}

	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, ellipse() called when both m_filled and m_stroked are off!?");
	}
}

void PaletteHost::polygon(PointMem* points, int npoints) {
	if ( m_filled ) {
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		glBegin(GL_TRIANGLE_FAN);
		glVertex2d(0.0, 0.0);
		for ( int pn=0; pn<npoints; pn++ ) {
			PointMem* p = &points[pn];
			glVertex2d(p->x,p->y);
		}
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		glBegin(GL_LINE_LOOP);
		for ( int pn=0; pn<npoints; pn++ ) {
			PointMem* p = &points[pn];
			glVertex2d(p->x,p->y);
		}
		glEnd();
	}

	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, ellipse() called when both m_filled and m_stroked are off!?");
	}
}

void PaletteHost::popMatrix() {
	glPopMatrix();
}

void PaletteHost::pushMatrix() {
	glPushMatrix();
}

#define RANDONE (((double)rand())/RAND_MAX)
#define RANDB ((((double)rand())/RAND_MAX)*2.0f-1.0f)

void
PaletteHost::test_draw()
{
	for ( int i=0; i<1000; i++ ) {
		glColor4d(RANDONE,RANDONE,RANDONE,RANDONE);
		glBegin(GL_QUADS);
		glVertex2d(RANDB,RANDB);
		glVertex2d(RANDB,RANDB);
		glVertex2d(RANDB,RANDB);
		glVertex2d(RANDB,RANDB);
		glVertex2d(RANDB,RANDB);
		glEnd();
	}
}

// Return everything after the '=' (and whitespace)
std::string
everything_after_char(std::string line, char lookfor = '=')
{
	const char *p = line.c_str();
	const char *q = strchr(p,lookfor);
	if ( q == NULL ) {
		NosuchDebug("Invalid format (no =): %s",p);
		return "";
	}
	q++;
	while ( *q != 0 && isspace(*q) ) {
		q++;
	}
	size_t len = strlen(q);
	if ( q[len-1] == '\r' ) {
		len--;
	}
	return std::string(q,len);
}

bool PaletteHost::initStuff() {

	bool r = true;
	try {
		_palette = new Palette(this);

		// static initializations
		RegionParams::Initialize();
		TrackedCursor::initialize();
		Sprite::initialize();
		Palette::initialize();

		if (_scheduler) {
			_scheduler->SetRunning(true);
		}

		_palette->now = MillisecondsSoFar();

		int osc_input_port = BASE_OSC_INPUT_PORT + PortOffset;
		NosuchDebug("Palette: listening for OSC on port %d",osc_input_port);

		_daemon = new PaletteDaemon(this, osc_input_port, DEFAULT_OSC_INPUT_HOST);

	} catch (NosuchException& e) {
		NosuchDebug("NosuchException: %s",e.message());
		r = false;
	} catch (...) {
		// Does this really work?  Not sure
		NosuchDebug("Some other kind of exception occured!?");
		r = false;
	}
	NosuchDebug(2,"PaletteHost::initStuff returns %s\n",r?"true":"false");
	return r;
}

DWORD PaletteHost::PaletteHostPoke()
{
	NosuchDebug("PalettePoke!\n");
	return 0;
}

DWORD PaletteHost::PaletteHostProcessOpenGL(ProcessOpenGLStruct *pGL)
{
	gl_frame++;
	// NosuchDebug("PaletteHostProcessOpenGL");
	if ( gl_shutting_down ) {
		return FF_SUCCESS;
	}
	if ( disabled ) {
		return FF_SUCCESS;
	}

	if ( ! initialized ) {
		// NosuchDebug("PaletteHost calling initStuff()");
		if ( ! initStuff() ) {
			NosuchDebug("PaletteHost::initStuff failed, disabling plugin!");
			disabled = true;
			return FF_FAIL;
		}
		initialized = true;
	}

	NosuchLock(&json_mutex,"json");
	if (json_pending) {
		// Execute json stuff and generate response
		// NosuchDebug("####### EXECUTING json method=%s now=%d",json_method.c_str(),Palette::now);
		json_result = ExecuteJsonAndCatchExceptions(json_method, json_params, json_id);
		json_pending = false;

		// NosuchDebug("####### Signaling json_cond! now=%d",Palette::now);
		int e = pthread_cond_signal(&json_cond);
		if ( e ) {
			NosuchDebug("ERROR from pthread_cond_signal e=%d\n",e);
		}
		// NosuchDebug("####### After signaling now=%d",Palette::now);
	}
	NosuchUnlock(&json_mutex,"json");

	bool passthru = FALSE;
	if ( passthru == TRUE && pGL != NULL ) {
		if (pGL->numInputTextures<1)
			return FF_FAIL;

		if (pGL->inputTextures[0]==NULL)
			return FF_FAIL;
  
		FFGLTextureStruct &Texture = *(pGL->inputTextures[0]);

		//bind the texture handle
		glBindTexture(GL_TEXTURE_2D, Texture.Handle);

		 //enable texturemapping
		glEnable(GL_TEXTURE_2D);
		
		//get the max s,t that correspond to the 
		//width,height of the used portion of the allocated texture space
		FFGLTexCoords maxCoords = GetMaxGLTexCoords(Texture);
		glColor4f(1.0, 1.0, 1.0, 1.0);
		glBegin(GL_QUADS);
		glTexCoord2d(0.0, 0.0);					glVertex2f(-1,-1);
		glTexCoord2d(0.0, maxCoords.t);			glVertex2f(-1,1);
		glTexCoord2d(maxCoords.s, maxCoords.t); glVertex2f(1,1);
		glTexCoord2d(maxCoords.s, 0.0);			glVertex2f(1,-1);
		glEnd();
		
		//unbind the texture
		glBindTexture(GL_TEXTURE_2D, 0);
	}

#if 0
	// Draw line just to show we're alive
	glColor4d(0.0, 0.0, 1.0, 1.0);
	glBegin(GL_LINES); 
	glVertex2d( -0.5, -0.5); 
	glVertex2d( 0.5, 0.5); 
	glEnd(); 
#endif

	lock_paletteHost();

	bool gotexception = false;
	try {
		CATCH_NULL_POINTERS;

		int tm = _palette->now;
		int begintm = _palette->now;
		int endtm = MillisecondsSoFar();
		NosuchDebug(2,"ProcessOpenGL tm=%d endtm=%d dt=%d",tm,endtm,(endtm-tm));

		glDisable(GL_TEXTURE_2D); 
		glEnable(GL_BLEND); 
		glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA); 
		glLineWidth((GLfloat)3.0f);

		int ndt = 1;
		int n;
		for ( n=1; n<=ndt; n++ ) {
			tm = (int)(begintm + 0.5 + n * ((double)(endtm-begintm)/(double)ndt));
			if ( tm > endtm ) {
				tm = endtm;
			}
			int r = _palette->draw();

			if ( _dotest ) {
				test_draw();
			}
			_palette->advanceTo(tm);
			if ( r > 0 ) {
				NosuchDebug("Palette::draw returned failure? (r=%d)\n",r);
				gotexception = true;
				break;
			}
		}
	} catch (NosuchException& e ) {
		NosuchDebug("NosuchException in Palette::draw : %s",e.message());
		gotexception = true;
	} catch (...) {
		NosuchDebug("UNKNOWN Exception in Palette::draw!");
		gotexception = true;
	}

	if ( gotexception && disable_on_exception ) {
		NosuchDebug("DISABLING PaletteHost due to exception!!!!!");
		disabled = true;
	}

	unlock_paletteHost();

	glDisable(GL_BLEND); 
	glEnable(GL_TEXTURE_2D); 
	// END NEW CODE

	//disable texturemapping
	glDisable(GL_TEXTURE_2D);
	
	//restore default color
	glColor4d(1.f,1.f,1.f,1.f);
	
	return FF_SUCCESS;
}

void PaletteHost::lock_paletteHost() {
	// NosuchLock(&palette_mutex,"paletteHost");
}

void PaletteHost::unlock_paletteHost() {
	// NosuchUnlock(&palette_mutex,"paletteHost");
}

bool has_invalid_char(const char *nm)
{
	for ( const char *p=nm; *p!='\0'; p++ ) {
		if ( ! isalnum(*p) )
			return true;
	}
	return false;
}

std::string PaletteHost::jsonDoubleResult(double r, const char *id) {
	return NosuchSnprintf("{ \"jsonrpc\": \"2.0\", \"result\": %f, \"id\": \"%s\" }",r,id);
}

std::string PaletteHost::jsonIntResult(int r, const char *id) {
	return NosuchSnprintf("{ \"jsonrpc\": \"2.0\", \"result\": %d, \"id\": \"%s\" }\r\n",r,id);
}

std::string PaletteHost::jsonStringResult(std::string r, const char *id) {
	return NosuchSnprintf("{ \"jsonrpc\": \"2.0\", \"result\": \"%s\", \"id\": \"%s\" }\r\n",r.c_str(),id);
}

std::string PaletteHost::jsonMethError(std::string e, const char *id) {
	return jsonError(-32602, e,id);
}

std::string PaletteHost::jsonError(int code, std::string e, const char* id) {
	return NosuchSnprintf("{ \"jsonrpc\": \"2.0\", \"error\": {\"code\": %d, \"message\": \"%s\" }, \"id\":\"%s\" }\r\n",code,e.c_str(),id);
}

std::string PaletteHost::jsonConfigResult(std::string name, const char *id) {

	// Remove the filename suffix on the config name
	int suffindex = (int)name.length() - (int)Palette::configSuffix.length();
	if ( suffindex > 0 && name.substr(suffindex) == Palette::configSuffix ) {
		name = name.substr(0,name.length()-Palette::configSuffix.length());
	}
	return jsonStringResult(name,id);
}

std::string PaletteHost::ExecuteJsonAndCatchExceptions(std::string meth, cJSON *params, const char *id) {
	std::string r;
	try {
		CATCH_NULL_POINTERS;

		r = ExecuteJson(meth,params,id);
	} catch (NosuchException& e) {
		std::string s = NosuchSnprintf("NosuchException in ProcessJson!! - %s",e.message());
		r = error_json(-32000,s.c_str(),id);
	} catch (...) {
		// This doesn't seem to work - it doesn't seem to catch other exceptions...
		std::string s = NosuchSnprintf("Some other kind of exception occured in ProcessJson!?");
		r = error_json(-32000,s.c_str(),id);
	}
	return r;
}

std::string PaletteHost::ExecuteJson(std::string meth, cJSON *params, const char *id) {

	static std::string errstr;  // So errstr.c_str() stays around, but I'm not sure that's now needed

	if ( meth == "debug_tail" ) {
#if 0
		cJSON *c_amount = cJSON_GetObjectItem(params,"amount");
		if ( ! c_amount ) {
			return error_json(-32000,"Missing amount argument",id);
		}
		if ( c_amount->type != cJSON_String ) {
			return error_json(-32000,"Expecting string type in amount argument to get",id);
		}
#endif
#ifdef DEBUG_DUMP_BUFFER
		std::string s = NosuchDebugDumpBuffer();
		std::string s2 = NosuchEscapeHtml(s);
		std::string result = 
			"{\"jsonrpc\": \"2.0\", \"result\": \""
			+ s2
			+ "\", \"id\": \""
			+ id
			+ "\"}";
		return(result);
#else
		return error_json(-32000,"DEBUG_DUMP_BUFFER not defined",id);
#endif
	}
	if ( meth == "_echo" || meth == "echo" ) {
		cJSON *c_value = cJSON_GetObjectItem(params,"value");
		if ( ! c_value ) {
			return error_json(-32000,"Missing value argument",id);
		}
		if ( c_value->type != cJSON_String ) {
			return error_json(-32000,"Expecting string type in value argument to echo",id);
		}
		return jsonStringResult(c_value->valuestring,id);
	}
	if (meth == "activity_set") {
		bool onoff = needBool(meth, params, "onoff");
		_activityEnabled = onoff;
		return jsonIntResult(0,id);
	}
	if (meth == "tempo_default") {
		Scheduler::ChangeClicksPerSecond(DEFAULT_CLICKS_PER_SECOND);
		return jsonIntResult(0,id);
	}
	if (meth == "tempo_set") {
		double cps = needDouble(meth, params, "clickspersecond");
		Scheduler::ChangeClicksPerSecond((click_t)(cps));
		return jsonIntResult(0,id);
	}
	if (meth == "tempo_get") {
		return jsonIntResult((int)(Scheduler::ClicksPerSecond),id);
	}
	if (meth == "tempo_adjust") {
		double factor = needDouble(meth, params, "factor");
		Scheduler::ChangeClicksPerSecond((int)(click_t)(factor*Scheduler::ClicksPerSecond));
		return jsonIntResult(0,id);
	}
	if (meth == "push_config") {
		cJSON* sound = needObject(meth,params,"sound");
		cJSON* visual = needObject(meth,params,"visual");
		std::string soundstr = cJSON_PrintUnformatted(sound);
		std::string visualstr = cJSON_PrintUnformatted(visual);
		size_t sz = soundstr.length() + visualstr.length();
		_palette->LoadParamPush(sound, visual);
		return jsonIntResult((int)sz,id);
	}
	if (meth == "set_param") {
		std::string param = needString(meth,params,"param");
		std::string value = needString(meth,params,"value");
		_palette->region.params.Set(param, value);
		return jsonIntResult(0,id);
	}
	if (meth == "set_params") {
		for (cJSON* item = params->child; item != NULL; item = item->next) {
			if (item->type == cJSON_String) {
				std::string nm = item->string;
				std::string val = item->valuestring;
				// NosuchDebug("set %s %s\n", nm.c_str(), val.c_str());
				_palette->region.params.Set(nm, val);
			}
		}
		return jsonIntResult(0, id);
	}
	if (meth == "push_effects") {
		std::string effects = needString(meth,params,"effects");
		return jsonIntResult(0,id);
	}
	if (meth == "debug") {
		needParams(meth, params);
		std::string action = needString(meth, params, "action", "");
		if (action == "") {
			throw NosuchException("debug method - needs action parameter");
		}
		else if (action == "scheduler_on") {
			Scheduler::Debug = true;
			NosuchDebug("Schedule debugging is ON");
		} 
		else if (action == "scheduler_off") {
			Scheduler::Debug = false;
			NosuchDebug("Schedule debugging is OFF");
		}
		else {
			throw NosuchException("debug method - unrecognize action: %s",action.c_str());
		}
		return jsonIntResult(0, id);
	}

	errstr = NosuchSnprintf("Unrecognized method name - %s",meth.c_str());
	return error_json(-32000,errstr.c_str(),id);
}

bool
PaletteHost::checkAddrPattern(const char *addr, char *patt)
{
	return ( strncmp(addr,patt,strlen(patt)) == 0 );
}

int
ArgAsInt32(const osc::ReceivedMessage& m, unsigned int n)
{
    osc::ReceivedMessage::const_iterator arg = m.ArgumentsBegin();
	const char *types = m.TypeTags();
	if ( n >= strlen(types) )  {
		DebugOscMessage("ArgAsInt32 ",m);
		throw NosuchException("Attempt to get argument n=%d, but not that many arguments on addr=%s\n",n,m.AddressPattern());
	}
	if ( types[n] != 'i' ) {
		DebugOscMessage("ArgAsInt32 ",m);
		throw NosuchException("Expected argument n=%d to be an int(i), but it is (%c)\n",n,types[n]);
	}
	for ( unsigned i=0; i<n; i++ )
		arg++;
    return arg->AsInt32();
}

float
ArgAsFloat(const osc::ReceivedMessage& m, unsigned int n)
{
    osc::ReceivedMessage::const_iterator arg = m.ArgumentsBegin();
	const char *types = m.TypeTags();
	if ( n >= strlen(types) )  {
		DebugOscMessage("ArgAsFloat ",m);
		throw NosuchException("Attempt to get argument n=%d, but not that many arguments on addr=%s\n",n,m.AddressPattern());
	}
	if ( types[n] != 'f' ) {
		DebugOscMessage("ArgAsFloat ",m);
		throw NosuchException("Expected argument n=%d to be a double(f), but it is (%c)\n",n,types[n]);
	}
	for ( unsigned i=0; i<n; i++ )
		arg++;
    return arg->AsFloat();
}

std::string
ArgAsString(const osc::ReceivedMessage& m, unsigned n)
{
    osc::ReceivedMessage::const_iterator arg = m.ArgumentsBegin();
	const char *types = m.TypeTags();
	if ( n < 0 || n >= strlen(types) )  {
		DebugOscMessage("ArgAsString ",m);
		throw NosuchException("Attempt to get argument n=%d, but not that many arguments on addr=%s\n",n,m.AddressPattern());
	}
	if ( types[n] != 's' ) {
		DebugOscMessage("ArgAsString ",m);
		throw NosuchException("Expected argument n=%d to be a string(s), but it is (%c)\n",n,types[n]);
	}
	for ( unsigned i=0; i<n; i++ )
		arg++;
	return std::string(arg->AsString());
}

static void
xyz_adjust(double expand, bool switchyz, double& x, double& y, double& z) {
	if ( switchyz ) {
		double t = y;
		y = z;
		z = t;
		z = 1.0 - z;
	}
	// The values we get from the Palette don't go all the way to
	// 0.0 or 1.0, so we expand
	// the range a bit so people can draw all the way to the edges.
	x = ((x - 0.5f) * expand) + 0.5f;
	y = ((y - 0.5f) * expand) + 0.5f;
	if (x < 0.0)
		x = 0.0f;
	else if (x > 1.0)
		x = 1.0f;
	if (y < 0.0)
		y = 0.0f;
	else if (y > 1.0)
		y = 1.0f;
}

void
PaletteHost::SetCursorCid(std::string cid, std::string cidsource, NosuchVector point, double z, bool recordable )
{
	_palette->region.setTrackedCursor(_palette,cid, cidsource, point, z);
}

void PaletteHost::ProcessOscMessage( std::string source, const osc::ReceivedMessage& m) {
	static int Nprocessed = 0;
	try{
	    const char *types = m.TypeTags();
		const char *addr = m.AddressPattern();
		Nprocessed++;
		NosuchDebug(1,"ProcessOscMessage source=%s currentclick=%d addr=%s",
			source.c_str(),Scheduler::CurrentClick,addr);

		if (checkAddrPattern(addr, "/cursor")) {
			std::string cmd = ArgAsString(m,0);
			std::string cid = ArgAsString(m,1); // it's a long string, globally unique
			double x = ArgAsFloat(m,2);
			double y = ArgAsFloat(m,3);
			double z = ArgAsFloat(m,4);

			if (cmd == "down" || cmd == "drag") {
				if (NosuchDebugCursor) {
					NosuchDebug("GOT /cursor %s cid=%s x,y=%.4f,%.4f  z=%f", cmd.c_str(), cid.c_str(), x, y, z);
				}
				SetCursorCid(cid, source, NosuchVector(x, y), z);
			}
			else if (cmd == "up") {
				if (NosuchDebugCursor) {
					NosuchDebug("GOT /cursor %s cid=%s x,y=%.4f,%.4f", cmd.c_str(), cid.c_str(), x, y);
				}
				_palette->region.doCursorUp(_palette, cid);
			}
			return;
		}
		if (checkAddrPattern(addr, "/spriteon") || checkAddrPattern(addr,"/sprite") ) {
			double x = ArgAsFloat(m,0);
			double y = ArgAsFloat(m,1);
			y = 1.0 - y;
			double z = ArgAsFloat(m,2);
			std::string cid = ArgAsString(m,3);
			// NosuchDebug("GOT /spriteon x,y,z=%.4f,%.4f,%.4f id=%s\n",x,y,z,id.c_str());
			palette()->region.instantiateSpriteAt(cid,NosuchVector(x, y), z);

			return;
		}
		if (checkAddrPattern(addr, "/clear")) {
			// NosuchDebug("GOT /clear\n");
			_palette->clear();
			return;
		}
		if (checkAddrPattern(addr, "/api")) {
			std::string meth = ArgAsString(m,0);
			std::string params = ArgAsString(m,1);
			if (NosuchDebugAPI) {
				NosuchDebug("/api !! meth=%s params=%s\n", meth.c_str(), params.c_str());
			}
			cJSON *c_params = cJSON_Parse(params.c_str());
			if (c_params == NULL) {
				NosuchDebug("ProcessOscMessage can't parse params=%s\n", params.c_str());
				return;
			}
			std::string ret = RespondToJson(meth.c_str(),c_params,"54321");
			cJSON_Delete(c_params);
			return;
		}

		// First do things that have no arguments
		if ( strcmp(addr,"/clear") == 0 ) {
			_palette->clear();
		} else if ( strcmp(addr,"/list") == 0 ) {
		} else if ( strcmp(addr,"/run") == 0 ) {
		} else if ( strcmp(addr,"/stop") == 0 ) {
		}

		NosuchDebug("PaletteOscInput - NO HANDLER FOR addr=%s",m.AddressPattern());
	} catch( osc::Exception& e ){
		// any parsing errors such as unexpected argument types, or 
		// missing arguments get thrown as exceptions.
		NosuchDebug("ProcessOscMessage error while parsing message: %s : %s",m.AddressPattern(),e.what());
	} catch (NosuchException& e) {
		NosuchDebug("ProcessOscMessage, NosuchException: %s",e.message());
	} catch (...) {
		// This doesn't seem to work - it doesn't seem to catch other exceptions...
		NosuchDebug("ProcessOscMessage, some other kind of exception occured during !?");
	}
}

std::string PaletteHost::RespondToJson(std::string method, cJSON *params, const char *id) {

	// We want JSON requests to be interpreted in the main thread of the FFGL plugin,
	// so we stuff the request into json_* variables and wait for the main thread to
	// pick it up (in ProcessOpenGL)
	// NosuchDebug("About to Lock json B");
	NosuchLock(&json_mutex,"json");
	// NosuchDebug("After Lock json B");

	json_pending = true;
	json_method = std::string(method);
	json_params = params;
	json_id = id;

	bool err = false;
	while ( json_pending ) {
		NosuchDebug(2, "####### Waiting for json_cond!");
		int e = pthread_cond_wait(&json_cond, &json_mutex);
		if ( e ) {
			NosuchDebug(2,"####### ERROR from pthread_cond_wait e=%d",e);
			err = true;
			break;
		}
	}
	std::string result;
	if ( err ) {
		result = error_json(-32000,"Error waiting for json!?");
	} else {
		result = json_result;
	}

	// NosuchDebug("About to UnLock json B");
	NosuchUnlock(&json_mutex,"json");

	return result;
}

