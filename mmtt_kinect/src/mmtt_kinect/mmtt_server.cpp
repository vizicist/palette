/*
	Space Manifold - a variety of tools for depth cameras and FreeFrame

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

#include <winsock2.h>
#pragma comment(lib, "ws2_32")

#include <list>

#include "stdafx.h"
#include "stdint.h"
#include "sha1.h"

#include <GL/gl.h>
#include <GL/glu.h>
 
#include <pthread.h>
#include <opencv/cv.h>
#include <opencv/highgui.h>

#include "NosuchDebug.h"
#include "NosuchSocket.h"
#include "NosuchHttp.h"
#include "mmsystem.h"
#include "mmtt.h"
#include "mmtt_depth.h"
#include "Event.h"
#include "Cursor.h"

#include "BlobResult.h"

#include "OscSender.h"
#include "OscMessage.h"

#include <iostream>
#include <sstream>
#include <fstream>

#include "UT_SharedMem.h"
#include "UT_Mutex.h"
#include "SharedMemHeader.h"

#include "NuiApi.h"

MmttServer* ThisServer;

using namespace std;

extern "C" {

int MaskDilateAmount = 2;

typedef struct RGBcolor {
	int r;
	int g;
	int b;
} RGBcolor;

static RGBcolor region2color[] = {
	{ 0,    0,  0 }, //region = 0   is this used?
	{ 255,  0,  0 }, //region = 1   background
	{ 0  ,255,  0 }, //region = 2
	{ 0  ,  0,255 }, //region = 3
	{ 255,255,  0 }, //region = 4
	{ 0  ,255,255 }, //region = 5
	{ 255,  0,255 }, //region = 6
	{ 255,128,  0 }, //region = 7
	{ 128,255,  0 }, //region = 8
	{ 128,  0,255 }, //region = 9
	{ 255,  0,128 }, //region = 10
	{ 0  ,255,128 }, //region = 11
	{ 0  ,128,255 }, //region = 12
	{ 255,128,128 }, //region = 13
	{ 128,255,128 }, //region = 14
	{ 128,128,255 }, //region = 15
	{ 255,255,128 }, //region = 16
	{ 64, 128,128 }, //region = 17
	{ 128, 64,128 }, //region = 18
	{ 128,128,64  }, //region = 19
	{ 64,  64,128 }, //region = 20
	{ 128, 64,64  }, //region = 21
	{ 255,255,255 }, //region = mask
};

#define MAX_REGION_ID 22
#define MASK_REGION_ID 22

static CvScalar region2cvscalar[MAX_REGION_ID+1];

};  // extern "C"

static float raw_depth_to_meters(int raw_depth)
{
	if ( raw_depth < 2047 ) {
		return 1.0f / (raw_depth * -0.0030711016f + 3.3309495161f);
	}
	return 0.0f;
}
 
std::string PaletteDataPath(std::string fname)
{
	errno_t err;
	char* palette = NULL;
	char* source = NULL;
	char *dataval = NULL;
	char *value = NULL;
	std::string sourcepath = "";
	std::string datapath;
	size_t len;

	err = _dupenv_s(&palette, &len, "PALETTE");
	if (err || palette == NULL)
	{
		// NosuchDebug("No value for PALETTE environment variable!?\n");
		return NULL;
	}

	err = _dupenv_s(&dataval, &len, "PALETTE_DATA");
	if (err == 0 && dataval != NULL) {
		 dataval = "default";
	}

	err = _dupenv_s(&value, &len, "PALETTE_SOURCE");
	std::string parent;
	if (err == 0 && value != NULL) {
		parent = std::string(value);
	}
	else {
		// This weird name is to get the 64-bit Common Files directory
		// (where the Palette stuff is kept) from a 32-bit program.
		char *p = getenv("CommonProgramW6432");
		if ( p == NULL ) {
			p = ".";
		}
		parent = std::string(p) + "\\Palette";
	}
	datapath = std::string(parent) + "\\data_" + std::string(dataval) + "\\" + fname;
	return datapath.c_str();
}

MmttServer::MmttServer()
{
	NosuchErrorPopup = MmttServer::ErrorPopup;

	ThisServer = this;  // should try to remove this eventually 

	_status = "Uninitialized";
	_regionsfilled = false;
	_regionsDefinedByPatch = false;
	_showrawdepth = false;
	_showregionrects = true;

	registrationState = 0;
	continuousAlign = false;

	_OscClientList = "127.0.0.1:3333";

	_newblobresult = NULL;
	_oldblobresult = NULL;
	shutting_down = FALSE;
	mFirstDraw = TRUE;
	_httpserver = NULL;
	lastFpsTime = 0.0;
	doBlob = TRUE;
	doBW = TRUE;
	doSmooth = FALSE;
	autoDepth = FALSE;
	currentRegionValue = 1;
	do_setnextregion = FALSE;

	json_mutex = PTHREAD_MUTEX_INITIALIZER;
	json_cond = PTHREAD_COND_INITIALIZER;
	json_pending = FALSE;

	_jsonport = 4444;
	_do_initialalign = true;
	_patchFile = PaletteDataPath("config/mmtt_kinect.json");
	_tempDir = "c:/windows/temp";

	camera = DepthCamera::makeDepthCamera(this,"kinect");
	if ( ! camera ) {
		std::string msg = NosuchSnprintf("Unable to make kinect depth camera");
		NosuchDebug(msg.c_str());
		FatalAppExit( NULL, s2ws(msg).c_str());
	}

	init_values();

	_camWidth = camera->width();
	_camHeight = camera->height();
	_camBytesPerPixel = 3;
	_fpscount = 0;
	_framenum = 0;

	depth_mid = (uint8_t*)malloc(_camWidth*_camHeight*_camBytesPerPixel);
	depth_front = (uint8_t*)malloc(_camWidth*_camHeight*_camBytesPerPixel);

	rawdepth_back = (uint16_t*)malloc(_camWidth*_camHeight*sizeof(uint16_t));
	rawdepth_mid = (uint16_t*)malloc(_camWidth*_camHeight*sizeof(uint16_t));
	rawdepth_front = (uint16_t*)malloc(_camWidth*_camHeight*sizeof(uint16_t));

	depthmm_mid = (uint16_t*)malloc(_camWidth*_camHeight*sizeof(uint16_t));
	depthmm_front = (uint16_t*)malloc(_camWidth*_camHeight*sizeof(uint16_t));

	thresh_mid = (uint8_t*)malloc(_camWidth*_camHeight);
	thresh_front = (uint8_t*)malloc(_camWidth*_camHeight);

	mmtt_values["debug"] = &val_debug;
	mmtt_values["showrawdepth"] = &val_showrawdepth;
	mmtt_values["showfps"] = &val_showfps;
	mmtt_values["showregionhits"] = &val_showregionhits;
	mmtt_values["showmask"] = &val_showmask;
	mmtt_values["showregionmap"] = &val_showregionmap;
	mmtt_values["showregionrects"] = &val_showregionrects;
	mmtt_values["usemask"] = &val_usemask;
	mmtt_values["tilt"] = &val_tilt;
	mmtt_values["left"] = &val_left;
	mmtt_values["right"] = &val_right;
	mmtt_values["top"] = &val_top;
	mmtt_values["bottom"] = &val_bottom;
	mmtt_values["front"] = &val_front;
	mmtt_values["backtop"] = &val_backtop;
	mmtt_values["backbottom"] = &val_backbottom;
	mmtt_values["blob_filter"] = &val_blob_filter;
	mmtt_values["blob_param1"] = &val_blob_param1;
	mmtt_values["blob_maxsize"] = &val_blob_maxsize;
	mmtt_values["blob_minsize"] = &val_blob_minsize;
	mmtt_values["confidence"] = &val_confidence;
	mmtt_values["autowindow"] = &val_auto_window;
	mmtt_values["shiftx"] = &val_shiftx;
	mmtt_values["shifty"] = &val_shifty;


	_camSize = cvSize(_camWidth,_camHeight);
	_tmpGray = cvCreateImage( _camSize, 8, 1 ); // allocate a 1 channel byte image
	_maskImage = cvCreateImage( _camSize, IPL_DEPTH_8U, 1 ); // allocate a 1 channel byte image

	clearImage(_maskImage);

	_regionsImage = cvCreateImage( _camSize, IPL_DEPTH_8U, 1 ); // allocate a 1 channel byte image
	_tmpRegionsColor = cvCreateImage( _camSize, IPL_DEPTH_8U, 3 );

	_tmpThresh = cvCreateImageHeader( _camSize, IPL_DEPTH_8U, 1 );

	_ffImage = cvCreateImageHeader( _camSize, IPL_DEPTH_8U, 3 );

	std::string htmldir = PaletteDataPath("html");
	_httpserver = new MmttHttp(this,_jsonport,htmldir,60);

	SetOscClientList(_OscClientList,_Clients);

	std::string ss;

	_ffpixels = NULL;
	_ffpixelsz = 0;

	if ( _patchFile != "" ) {
		std::string err = LoadPatch(_patchFile);
		if ( err != "" ) {
			NosuchDebug("LoadPatch of %s failed!?  err=%s",_patchFile.c_str(),err.c_str());
			std::string msg = NosuchSnprintf("*** Error ***\n\nLoadPatch of %s failed\n%s",_patchFile.c_str(),err.c_str());
			ErrorPopup(msg.c_str());
			exit(1);
		}
		// Do it again?  There's a bug where it doesn't get completely initialized unless you do it twice!?  HACK!!
		// LoadPatch(_patchFile);
	}

	if ( _do_initialalign ) {
		startAlign();
	}
	_status = "";
}

void
MmttServer::init_values() {

	// pre-compute cvScalar values, optimization
	for ( int i=0; i<=MAX_REGION_ID; i++ ) {
		RGBcolor color = region2color[i];
		// NOTE: swapping the R and B values, since OpenCV defaults to BGR
		region2cvscalar[i] = CV_RGB(color.b,color.g,color.r);
	}

	// These should NOT be persistent in a saved patch
	val_debug = MmttValue(0,0,1,false);
	val_showrawdepth = MmttValue(_showrawdepth,0,1,false);
	val_showregionrects = MmttValue(_showregionrects,0,1,false);
	val_showfps = MmttValue(0,0,1,false);
	val_showregionhits = MmttValue(0,0,1,false);
	val_showregionmap = MmttValue(0,0,1,false);
	val_showmask = MmttValue(0,0,1,false);
	val_usemask = MmttValue(1,0,1,false);

	// These should be persistent in a saved patch
	val_tilt = MmttValue(0,-12.9,30.0);
	val_left = MmttValue(0,0,639);
	val_right = MmttValue(639,0,639);
	val_top = MmttValue(0,0,479);
	val_bottom = MmttValue(479,0,479);
	val_front = MmttValue(0,0,3000);         // mm
	val_backtop = MmttValue(camera->default_backtop(),0,3000);    // mm
	val_backbottom = MmttValue(camera->default_backtop(),0,3000); // mm
	val_auto_window = MmttValue(80,0,400); // mm
	val_blob_filter = MmttValue(1,0,1);
	val_blob_param1 = MmttValue(100,0,250.0);
	val_blob_maxsize = MmttValue(10000.0,0,15000.0);
	val_blob_minsize = MmttValue(/* 65.0 */ 350,0,5000.0);
	val_confidence = MmttValue(200,0,4000.0);
	val_shiftx = MmttValue(0,-639,639);
	val_shifty = MmttValue(0,-479,479);
	NosuchDebug(1,"TEMPORARY blob_minsize HACK - used to be 65.0");
}

MmttServer::~MmttServer() {

	shutting_down = TRUE;

	NosuchDebug(1,"MmttServer destructor!\n");
	delete _httpserver;

	camera->Shutdown();
}

#ifdef USE_FREENECT
void MmttServer::real_kinect_depth_cb(freenect_device *dev, void *v_depth, uint32_t timestamp)
{
	// NosuchDebug("depth_cb !!!!! timestamp=%ld\n",timestamp);
	uint16_t *depth = (uint16_t*)v_depth;

	pthread_mutex_lock(&gl_backbuf_mutex);

	// swap buffers
	if (rawdepth_back != depth) {
		NosuchDebug("Hey, rawdepth_back != depth!?\n");
	}
	
	if ( got_depth == 0 ) {
		rawdepth_back = rawdepth_mid;
		freenect_set_depth_buffer(dev, rawdepth_back);
		rawdepth_mid = depth;
	} else {
		// NosuchDebug("real_depth_cb got_depth wasn't 0?\n");
	}

	if ( val_debug.internal_value ) NosuchDebug("depthFrames++\n");
	_depthFrames++;

	got_depth++;
	pthread_cond_signal(&gl_frame_cond);
	pthread_mutex_unlock(&gl_backbuf_mutex);

}
#endif



void *MmttServer::mmtt_json_threadfunc(void *arg)
{
	NosuchDebug("mmtt_json_threadfunc is starting\n");
	while (shutting_down == FALSE ) {
		if ( _httpserver ) {
			NosuchDebug(1,"mmtt_json_threadfunc is calling Check\n");
			_httpserver->Check();
		}
		Sleep(1);
	}
	NosuchDebug("mmtt_json_threadfunc is exiting\n");
	return NULL;
}

void MmttServer::ErrorPopup(wchar_t const* msg) {
	MessageBoxW(NULL,msg,L"MultiMultiTouchTouch",MB_OK);
}

void MmttServer::ErrorPopup(const char* msg) {
	MessageBoxA(NULL,msg,"MultiMultiTouchTouch",MB_OK);
}

static vector<string>
tokenize(const string& str,const string& delimiters) {

	vector<string> tokens;
    	
	// skip delimiters at beginning.
	string::size_type lastPos = str.find_first_not_of(delimiters, 0);
    	
	// find first "non-delimiter".
	string::size_type pos = str.find_first_of(delimiters, lastPos);

	while (string::npos != pos || string::npos != lastPos) {
    	// found a token, add it to the vector.
    	tokens.push_back(str.substr(lastPos, pos - lastPos));
		
    	// skip delimiters.  Note the "not_of"
    	lastPos = str.find_first_not_of(delimiters, pos);
		
    	// find next "non-delimiter"
    	pos = str.find_first_of(delimiters, lastPos);
	}

	return tokens;
}

void MmttServer::SetOscClientList(std::string& clientlist,std::vector<OscSender*>& clientvector)
{
	vector<string> tokens = tokenize(clientlist,";:");
	for ( vector<string>::iterator it=tokens.begin(); it != tokens.end(); ) {
		string ahost = *it++;
		if ( it == tokens.end() )
			break;
		string aport = *it++;

		OscSender *o = new OscSender();
		o->setup(ahost,atoi(aport.c_str()));
		clientvector.push_back(o);
	}

}
void *ptr_mmtt_json_threadfunc(void *arg)
{
	MmttServer* server = (MmttServer*)arg;
	if ( server != NULL ) {
		return server->mmtt_json_threadfunc(arg);
	}
	NosuchErrorOutput("Hey! server==NULL in ptr_mmtt_json_threadfunc!?");
	return NULL;
}

void startHttpThread(MmttServer* server)
{
	pthread_t json_thread;

	int res = pthread_create(&json_thread, NULL, ptr_mmtt_json_threadfunc, server);
	if (res) {
		NosuchDebug("pthread_create for json_thread failed!?\n");
	}

}

// This normalization results in the range (0,0)->(1,1), relative to the region space
void
normalize_region_xy(float& x, float& y, CvRect& rect)
{
	x = x - rect.x;
	y = y - rect.y;
	x = x / rect.width;
	y = 1.0f - (y / rect.height);

	if ( x > 1.0 ) {
		x = 1.0;
	} else if ( x < 0.0 ) {
		x = 0.0;
	}

	if ( y > 1.0 ) {
		y = 1.0;
	} else if ( y < 0.0 ) {
		y = 0.0;
	}
}

// This normalization results in the range (-1,-1)->(1,1), relative to the outline center
void
normalize_outline_xy(float& x, float& y, float& blobcenterx, float& blobcentery, CvRect& rect)
{
	normalize_region_xy(x,y,rect);
	x -= blobcenterx;
	y -= blobcentery;
	// At this point we're centered on the blobcenter, with range (-1,-1) to (1,1)
	x = (x * 2.0f);
	y = (y * 2.0f);
}

void
MmttServer::SendAllOscClients(OscBundle& bundle, std::vector<OscSender *> &oscClients)
{
	vector<OscSender*>::iterator it;
	for ( it=oscClients.begin(); it != oscClients.end(); it++ ) {
		OscSender *sender = *it;
		sender->sendBundle(bundle);
	}
}



void MmttServer::check_json_and_execute()
{
	pthread_mutex_lock(&json_mutex);
	if (json_pending) {
		// Execute json stuff and generate response
		json_result = ExecuteJson(json_method, json_params, json_id);
		NosuchDebug(1,"AFTER ExecuteJson, result=%s\n",json_result.c_str());
		json_pending = FALSE;
		pthread_cond_signal(&json_cond);
	}
	pthread_mutex_unlock(&json_mutex);
}

void MmttServer::analyze_depth_images()
{
	// pthread_mutex_lock(&gl_backbuf_mutex);

	uint8_t *tmp;
	uint16_t *tmp16;

	// This buffering was needed when using the freenect library, but
	// with the Microsoft SDK, I don't thing it's actually needed (as much).
	// However, it's not really expensive (just twiddling some pointers),
	// and I don't want to break anything, so I'm leaving it.

	tmp = depth_front;
	depth_front = depth_mid;
	depth_mid = tmp;

	tmp16 = depthmm_front;
	depthmm_front = depthmm_mid;
	depthmm_mid = tmp16;

	tmp16 = rawdepth_mid;
	rawdepth_mid = rawdepth_front;
	rawdepth_front = tmp16;

	tmp = thresh_front;
	thresh_front = thresh_mid;
	thresh_mid = tmp;

	// processRawDepth(rawdepth_front);

	size_t camsz = _camWidth*_camHeight*_camBytesPerPixel;
	unsigned char *surfdata = NULL;

	if ( _ffpixels == NULL || _ffpixelsz < camsz ) {
		_ffpixels = (unsigned char *)malloc(camsz);
		_ffpixelsz = camsz;
	}
	if ( depth_front != NULL ) {
		surfdata = depth_front;
	}

	if ( _ffpixels != NULL && surfdata != NULL ) {

		_ffImage->origin = 1;
		_ffImage->imageData = (char *)_ffpixels;

		_tmpThresh->origin = 1;
		_tmpThresh->imageData = (char *)thresh_front;

		if ( surfdata ) {
			if ( val_debug.internal_value ) NosuchDebug("  Copying surfdata to ffpixels");
			memcpy(_ffpixels,surfdata,camsz);
			if ( ! val_showrawdepth.internal_value ) {
				analyzePixels();
			}
		} else {
#if 0
			memset(_ffpixels,0,camsz);
#endif
		}

		// Now put whatever we want to show into the _ffImage/_ffpixels image

		if ( surfdata && val_showrawdepth.internal_value ) {
			if ( val_debug.internal_value ) NosuchDebug("  Copying surfdata to ffpixels (again?)");
			// Doesn't seem to be needed...
			// memcpy(_ffpixels,surfdata,camsz);
		}

		if ( val_showregionrects.internal_value ) {
			// When showing the region rectangles, nothing else is shown.
			if ( val_debug.internal_value ) NosuchDebug("  Showing regionrects");
			showRegionRects();
		}

		if ( val_showregionmap.internal_value ) {
			if ( _regionsDefinedByPatch ) {
				// NosuchDebug("Unable to show region map when _regionsDefinedByPatch");
			} else {
				// When showing the colored regions, nothing else is shown.
				if ( val_debug.internal_value ) NosuchDebug("  Copying regionsImage to ffpixels");
				copyRegionsToColorImage(_regionsImage,_ffpixels,TRUE,FALSE,FALSE);
			}
		} else {
			if ( val_showregionhits.internal_value ) {
				showRegionHits();
				showBlobSessions();
			}

			if ( val_showmask.internal_value ) {
				showMask();
			} else if ( val_showregionmap.internal_value ) {
				copyRegionsToColorImage(_regionsImage,_ffpixels,TRUE,FALSE,FALSE);
			}
		}
	}

	long tm = timeGetTime();
	double curFrameTime = tm / 1000.0;

	if ( curFrameTime > (lastFpsTime+1.0) ) {
		if ( val_showfps.internal_value ) {
			NosuchDebug("Analyzed FPS = %d\n",_fpscount);
		}
		lastFpsTime = curFrameTime;
		_fpscount = 0;
	}

	if ( val_debug.internal_value ) NosuchDebug("MmttServer::update() end\n");
}

void MmttServer::draw_depth_image() {

	glMatrixMode(GL_MODELVIEW);	// Select The Modelview Matrix
	glLoadIdentity();			// Reset The Modelview Matrix

	glPushMatrix();

	// this is tweaked so kinect image
	// fills the window.
	gluLookAt(  0, 0, 2.43,
		0, 0, 0,
		0, 1, 0); 
	glClear( GL_COLOR_BUFFER_BIT | GL_DEPTH_BUFFER_BIT );

#ifdef DRAW_BOX_TO_DEBUG_THINGS
	glColor4f(0.0,1.0,0.0,0.5);
	glLineWidth((GLfloat)10.0f);
	glBegin(GL_LINE_LOOP);
	glVertex3f(-0.8f, 0.8f, 0.0f);	// Top Left
	glVertex3f( 0.8f, 0.8f, 0.0f);	// Top Right
	glVertex3f( 0.8f,-0.8f, 0.0f);	// Bottom Right
	glVertex3f(-0.8f,-0.8f, 0.0f);	// Bottom Left
	glEnd();

	glColor4f(1.0,1.0,1.0,1.0);
#endif

	glPixelStorei (GL_UNPACK_ROW_LENGTH, _camWidth);
 
	unsigned char *pix = ffpixels();
	// unsigned char *pix = depth_mid;

	static bool initialized = false;
	static GLuint texture;
	if ( ! initialized ) {
		initialized = true;
	    glGenTextures( 1, &texture );
	}

	glEnable( GL_TEXTURE_2D );

	glEnable(GL_BLEND); 

	// glBlendFunc(GL_SRC_ALPHA, GL_ONE_MINUS_SRC_ALPHA);  // original

	// We want the black in the texture image
	// to NOT wipe out other things.
	glBlendFunc(GL_ONE, GL_ONE);

	if ( pix != NULL ) {

	    glBindTexture( GL_TEXTURE_2D, texture );

		glTexImage2D(GL_TEXTURE_2D, 0,GL_RGB,
			_camWidth, _camHeight,
			0, GL_RGB, GL_UNSIGNED_BYTE, pix);

		glTexParameteri(GL_TEXTURE_2D,GL_TEXTURE_MIN_FILTER,GL_LINEAR);
		glTexParameteri(GL_TEXTURE_2D,GL_TEXTURE_MAG_FILTER,GL_LINEAR);

		glPushMatrix();

		glColor4f(1.0,1.0,1.0,0.5);
		glBegin(GL_QUADS);
		glTexCoord2d(1.0,0.0); glVertex3f( 1.0f, 1.0f, 0.0f);	// Top Left
		glTexCoord2d(0.0,0.0); glVertex3f(-1.0f, 1.0f, 0.0f);	// Top Right
		glTexCoord2d(0.0,1.0); glVertex3f(-1.0f,-1.0f, 0.0f);	// Bottom Right
		glTexCoord2d(1.0,1.0); glVertex3f( 1.0f,-1.0f, 0.0f);	// Bottom Left
		glEnd();			

		glPopMatrix();
	}

	glDisable( GL_TEXTURE_2D );

	// Should I disable GL_BLEND here?

#ifdef DRAW_BOX_TO_DEBUG_THINGS
	glColor4f(1.0,0.0,0.0,0.5);
	glLineWidth((GLfloat)10.0f);
	glBegin(GL_LINE_LOOP);
	glVertex3f(-0.5f, 0.5f, 0.0f);	// Top Left
	glVertex3f( 0.5f, 0.5f, 0.0f);	// Top Right
	glVertex3f( 0.5f,-0.5f, 0.0f);	// Bottom Right
	glVertex3f(-0.5f,-0.5f, 0.0f);	// Bottom Left
	glEnd();
#endif

	glPopMatrix();
	glColor4f(1.0,1.0,1.0,1.0);

	if ( continuousAlign == true && (
		registrationState == 300
		|| registrationState == 310
		|| registrationState == 311
		|| registrationState == 320
		|| registrationState == 330
		) ) {
		NosuchDebug(1,"registrationState = %d",registrationState);
		// do nothing - this avoids screen blinking/etc with in continuousAlign registration
	} else {
		SwapBuffers(g.hdc);
	}
}

static std::string
OscMessageToJson(OscMessage& msg) {
	std::string s = NosuchSnprintf("{ \"address\" : \"%s\", \"args\" : [ ", msg.getAddress().c_str());
	std::string sep = "";
	int nargs = msg.getNumArgs();
	for ( int n=0; n<nargs; n++ ) {
		s += sep;
		ArgType t = msg.getArgType(n);
		switch (t) {
		case TYPE_INT32:
			s += NosuchSnprintf("%d",msg.getArgAsInt32(n));
			break;
		case TYPE_FLOAT:
			s += NosuchSnprintf("%f",msg.getArgAsFloat(n));
			break;
		case TYPE_STRING:
			s += NosuchSnprintf("\"%s\"",msg.getArgAsString(n).c_str());
			break;
		default:
			NosuchDebug("Unable to handle type=%d in OscMessageToJson!",t);
			break;
		}
		sep = ",";
	}
	s += " ] }";
	return s;
}

static std::string
OscBundleToJson(OscBundle& bundle) {
	std::string s = "{ \"messages\" : [";
	std::string sep = "";
	int nmessages = bundle.getMessageCount();
	for ( int n=0; n<nmessages; n++ ) {
		s += sep;
		s += OscMessageToJson(bundle.getMessageAt(n));
		sep = ", ";
	}
	s += " ] }";
	return s;
}

void
MmttServer::SendOscToAllWebSocketClients(OscBundle& bundle)
{
	std::string msg = OscBundleToJson(bundle);
	if ( _httpserver ) {
		_httpserver->SendAllWebSocketClients(msg);
	}
}


std::string
MmttServer::RespondToJson(const char *method, cJSON *params, const char *id) {

	pthread_mutex_lock(&json_mutex);

	std::string result;

	json_pending = TRUE;
	json_method = method;
	json_params = params;
	json_id = id;
	while ( json_pending ) {
		pthread_cond_wait(&json_cond, &json_mutex);
	}
	result = json_result;

	pthread_mutex_unlock(&json_mutex);

	return result;
}

static u_long LookupAddress(const char* pcHost)
{
    u_long nRemoteAddr = inet_addr(pcHost);
    if (nRemoteAddr == INADDR_NONE) {
        // pcHost isn't a dotted IP, so resolve it through DNS
        hostent* pHE = gethostbyname(pcHost);
        if (pHE == 0) {
            return INADDR_NONE;
        }
        nRemoteAddr = *((u_long*)pHE->h_addr_list[0]);
    }

    return nRemoteAddr;
}

#if !defined(_WINSOCK2API_) 
// Winsock 2 header defines this, but Winsock 1.1 header doesn't.  In
// the interest of not requiring the Winsock 2 SDK which we don't really
// need, we'll just define this one constant ourselves.
#define SD_SEND 1
#endif

// direction = -1  (decrease)
// direction =  0  (toggle)
// direction =  1  (increase)
std::string
MmttServer::AdjustValue(cJSON *params, const char *id, int direction) {

	static std::string errstr;  // So errstr.c_str() stays around

	cJSON *c_name = cJSON_GetObjectItem(params,"name");
	if ( ! c_name ) {
		return error_json(-32000,"Missing name argument",id);
	}
	if ( c_name->type != cJSON_String ) {
		return error_json(-32000,"Expecting string type in name argument to mmtt_set",id);
	}
	std::string nm = std::string(c_name->valuestring);

	map<std::string, MmttValue*>::iterator it = mmtt_values.find(nm);
	if ( it == mmtt_values.end() ) {
		errstr = NosuchSnprintf("No kinect parameter with that name - %s",nm.c_str());
		return error_json(-32000,errstr.c_str(),id);
	}
	MmttValue* kv = it->second;

	if ( direction != 0 ) {
		cJSON *c_amount = cJSON_GetObjectItem(params,"amount");
		if ( ! c_amount ) {
			return error_json(-32000,"Missing amount argument",id);
		}
		if ( c_amount->type != cJSON_Number ) {
			return error_json(-32000,"Expecting number type in amount argument to mmtt_set",id);
		}
		kv->set_external_value(kv->external_value + direction * c_amount->valuedouble);
	} else {
		kv->set_external_value(! kv->external_value);
	}

	_updateValue(nm,kv);
	NosuchDebug(1,"mmtt_SET name=%s external=%lf internal=%lf\n",nm.c_str(),kv->external_value,kv->internal_value);
	return NosuchSnprintf(
			"{\"jsonrpc\": \"2.0\", \"result\": %lf, \"id\": \"%s\"}", kv->external_value, id);
}

void
MmttServer::_updateValue(std::string nm, MmttValue* v) {
	if ( nm == "debug" ) {
		if ( v->internal_value )
			NosuchDebugLevel = 1;
		else
			NosuchDebugLevel = 0;
#ifdef MMTT_KINECT
	} else if ( nm == "tilt" ) {
		NosuchDebug("Tilt value = %f",v->internal_value);
	    HRESULT tiltr;
		long degrees;
		tiltr = m_pNuiSensor[m_currentSensor]->NuiCameraElevationGetAngle(&degrees);
		if ( SUCCEEDED(tiltr) ) {
			NosuchDebug("Get tilt of %ld succeeded, is %ld",degrees);
		}
		degrees = (long)(v->internal_value);
		tiltr = m_pNuiSensor[m_currentSensor]->NuiCameraElevationSetAngle(degrees);
		if ( SUCCEEDED(tiltr) ) {
			NosuchDebug("Set tilt of %ld succeeded!",degrees);
		}
#endif
	}
}

void
MmttServer::_toggleValue(MmttValue* v) {
		v->internal_value = !v->internal_value;
		v->external_value = !v->external_value;
}

void
MmttServer::_stop_registration() {
	registrationState = 0;
	finishNewRegions();
	val_showregionmap.set_internal_value(0.0);
}

static bool
has_invalid_char(const char *nm)
{
	for ( const char *p=nm; *p!='\0'; p++ ) {
		if ( ! isalnum(*p) )
			return TRUE;
	}
	return FALSE;
}

std::string
MmttServer::startAlign() {
	// std::string err = LoadPatch(_patchFile);
	// if ( err != "" ) {
	// 	return NosuchSnprintf("LoadPatch failed!?  err=%s",err.c_str());
	// }
	NosuchDebug("startAlign is NOT calling LoadPatch!\n");
	if ( _curr_regions.size() == 0 ) {
		return "No regions yet - do you need to load a patch?";
	}
	registrationState = 300;
	continuousAlign = true;
	continuousAlignOkayCount = 0;
	continuousAlignOkayEnough = 10;
	return "";
}

std::string
MmttServer::ExecuteJson(const char *method, cJSON *params, const char *id) {

	static std::string errstr;  // So errstr.c_str() stays around

	if ( strncmp(method,"mmtt_get",8) != 0 ) {
		NosuchDebug("ExecuteJson, method=%s\n",method);
	}
	std::string m = std::string(method);

	// NosuchPrintTime(NosuchSnprintf("Begin method=%s id=%s",method,id).c_str());
	if ( strcmp(method,"echo") == 0 ) {
		cJSON *c_nm = cJSON_GetObjectItem(params,"value");
		if ( ! c_nm ) {
			return error_json(-32000,"Missing value argument",id);
		}
		char *nm = c_nm->valuestring;
		return NosuchSnprintf(
			"{\"jsonrpc\": \"2.0\", \"result\": { \"value\": \"%s\" }, \"id\": \"%s\"}",nm,id);
	}
	if ( strcmp(method,"do_reset") == 0 ) {
		// Obsolete method.
		return ok_json(id);
	}
	if ( strcmp(method,"toggle_showmask") == 0 ) {
		_toggleValue(&val_showmask);
		return ok_json(id);
	}
	if ( strcmp(method,"toggle_autodepth") == 0 ) {
		autoDepth = !autoDepth;
		return ok_json(id);
	}
	if ( strcmp(method,"toggle_showregionmap") == 0 ) {
		_toggleValue(&val_showregionmap);
		return ok_json(id);
	}
	if ( strcmp(method,"do_setnextregion") == 0 ) {
		do_setnextregion = TRUE;
		return ok_json(id);
	}
	if ( strcmp(method,"toggle_bw") == 0 ) {
		doBW = !doBW;
		return ok_json(id);
	}
	if ( strcmp(method,"toggle_smooth") == 0 ) {
		doSmooth = !doSmooth;
		return ok_json(id);
	}
	if ( strcmp(method,"start") == 0 ) {
		val_showrawdepth.set_internal_value(false);
		val_showregionrects.set_internal_value(false);
		return ok_json(id);
	}
	if ( strcmp(method,"stop") == 0 ) {
		val_showrawdepth.set_internal_value(true);
		val_showregionrects.set_internal_value(true);
		return ok_json(id);
	}
	if ( strcmp(method,"start_registration") == 0 ) {
		registrationState = 100;
		return ok_json(id);
	}
	if ( strcmp(method,"autopoke") == 0 ) {
		std::string err = LoadPatch(_patchFile);
		if ( err != "" ) {
			std::string msg = NosuchSnprintf("LoadPatch failed!?  err=%s",err.c_str());
			return error_json(-32000,msg.c_str(),id);
		}
		if ( _curr_regions.size() == 0 ) {
			return error_json(-32000,"No regions yet - do you need to load a patch?",id);
		}
		registrationState = 300;
		return ok_json(id);
	}
	if ( strcmp(method,"align_start") == 0 ) {
		std::string err = startAlign();
		if ( err != "" ) {
			std::string msg = NosuchSnprintf("startAlign failed!?  err=%s",err.c_str());
			return error_json(-32000,msg.c_str(),id);
		}
		return ok_json(id);
	}
	if ( strcmp(method,"align_stop") == 0 ) {
		continuousAlign = false;
		return ok_json(id);
	}
	if ( strcmp(method,"align_isdone") == 0 ) {
		int r = continuousAlign ? 0 : 1;
		return NosuchSnprintf(
			"{\"jsonrpc\": \"2.0\", \"result\": %d, \"id\": \"%s\"}", r, id);
	}
	if ( strcmp(method,"autotiltpoke") == 0 ) {
		std::string err = LoadPatch(_patchFile);
		if ( err != "" ) {
			std::string msg = NosuchSnprintf("LoadPatch failed!?  err=%s",err.c_str());
			return error_json(-32000,msg.c_str(),id);
		}
		// if ( _curr_regions.size() == 0 ) {
		// 	return error_json(-32000,"No regions yet - do you need to load a patch?",id);
		// }
		registrationState = 1300;
		return ok_json(id);
	}
	if ( strcmp(method,"stop_registration") == 0 ) {
		_stop_registration();
		return ok_json(id);
	}
	if ( strcmp(method,"config_save")==0 || strcmp(method,"patch_save")==0 ) {
		return SavePatch(id);
	}
	if ( strcmp(method,"config_load")==0 || strcmp(method,"patch_load")==0 ) {
		cJSON *c_nm = cJSON_GetObjectItem(params,"name");
		if ( ! c_nm ) {
			return error_json(-32000,"Missing name argument",id);
		}
		char *nm = c_nm->valuestring;
		if ( has_invalid_char(nm) ) {
			return error_json(-32000,"Invalid characters in name",id);
		}
		_patchFile = nm;
		std::string err = LoadPatch(_patchFile);
		if ( err == "" ) {
			// Not sure this is still needed - it was a bug workaround
			// NosuchDebug("Loading Patch a second time"); // HACK!!
			// (void) LoadPatch(_patchFile);
			return ok_json(id);
		} else {
			return error_json(-32000,err.c_str(),id);
		}
	}
	if ( strcmp(method,"mmtt_increment") == 0 ) {
		return AdjustValue(params,id,1);
	}
	if ( strcmp(method,"mmtt_toggle") == 0 ) {
		return AdjustValue(params,id,0);
	}
	if ( strcmp(method,"mmtt_decrement") == 0 ) {
		return AdjustValue(params,id,-1);
	}
	if ( strcmp(method,"mmtt_set") == 0 ) {
		cJSON *c_value = cJSON_GetObjectItem(params,"value");
		if ( ! c_value ) {
			return error_json(-32000,"Missing value argument",id);
		}
		if ( c_value->type != cJSON_Number ) {
			return error_json(-32000,"Expecting number type in value argument to mmtt_set",id);
		}
		cJSON *c_name = cJSON_GetObjectItem(params,"name");
		if ( ! c_name ) {
			return error_json(-32000,"Missing name argument",id);
		}
		if ( c_name->type != cJSON_String ) {
			return error_json(-32000,"Expecting string type in name argument to mmtt_set",id);
		}
		std::string nm = std::string(c_name->valuestring);

		map<std::string, MmttValue*>::iterator it = mmtt_values.find(nm);
		if ( it == mmtt_values.end() ) {
			errstr = NosuchSnprintf("No Mmtt parameter with that name - %s",nm.c_str());
			return error_json(-32000,errstr.c_str(),id);
		}
		MmttValue* kv = it->second;
		double f = c_value->valuedouble;
		if ( f < 0.0 || f > 1.0 ) {
			errstr = NosuchSnprintf("Invalid Mmtt parameter - name=%s value=%lf - must be between 0 and 1",nm.c_str(),f);
			return error_json(-32000,errstr.c_str(),id);
		}
		kv->set_external_value(f);
		_updateValue(nm,kv);
		NosuchDebug("mmtt_SET name=%s external=%lf internal=%lf\n",nm.c_str(),kv->external_value,kv->internal_value);
		return NosuchSnprintf(
			"{\"jsonrpc\": \"2.0\", \"result\": %lf, \"id\": \"%s\"}", kv->external_value, id);
	}
	if ( strcmp(method,"mmtt_get") == 0 ) {
		cJSON *c_name = cJSON_GetObjectItem(params,"name");
		if ( ! c_name ) {
			return error_json(-32000,"Missing name argument",id);
		}
		if ( c_name->type != cJSON_String ) {
			return error_json(-32000,"Expecting string type in name argument to mmtt_get",id);
		}
		std::string nm = std::string(c_name->valuestring);

		map<std::string, MmttValue*>::iterator it = mmtt_values.find(nm);
		if ( it == mmtt_values.end() ) {
			errstr = NosuchSnprintf("No Mmtt parameter with that name - %s",nm.c_str());
			return error_json(-32000,errstr.c_str(),id);
		}
		MmttValue* kv = it->second;
		return NosuchSnprintf(
			"{\"jsonrpc\": \"2.0\", \"result\": %lf, \"id\": \"%s\"}", kv->external_value, id);
	}

	errstr = NosuchSnprintf("Unrecognized method name - %s",method);
	return error_json(-32000,errstr.c_str(),id);
}

void
MmttServer::doDepthRegistration()
{
	if ( autoDepth )
		doAutoDepthRegistration();
	else
		doManualDepthRegistration();

	NosuchDebug(1,"After DepthRegistration, val_front=%lf  backtop_mm=%lf  backbottom_mm=%lf\n",
		val_front.internal_value,val_backtop.internal_value,val_backbottom.internal_value);
}

void
MmttServer::doManualDepthRegistration()
{
	// NosuchDebug("Start ManualDepthRegistration");
	// NosuchDebug("End ManualDepthRegistration");
}

void
MmttServer::doAutoDepthRegistration()
{
	NosuchDebug("Starting AutoDepth registration!\n");

	// Using depth_mm (millimeter depth values for the image), scan the image with a distance-window (100 mm?)
	// starting from the front, toward the back, looking for the first peak and then the first valley.
	int dwindow = (int)val_auto_window.internal_value;
	int dinc = 10;
	int max_trigger = (int)( (camera->height() * camera->width() ) / 15);
	int min_trigger = (int)( (camera->height() * camera->width() ) / 60);
	int max_tot_sofar = 0;
	int mm_of_max_tot_sofar = 0;
	for (int mm = 1; mm < 3000; mm += dinc) {
		int tot = 0;
		int i = 0;
		int h = camera->height();
		int w = camera->width();
		for (int y=0; y<h; y++) {
			for (int x=0; x<w; x++) {
				int thismm = depthmm_front[i];
				if ( thismm >= mm && thismm < (mm+dwindow) ) {
					tot++;
				}
				i++;
			}
		}
		if ( tot > max_trigger && tot > max_tot_sofar ) {
				max_tot_sofar = tot;
				mm_of_max_tot_sofar = mm;
		}
		if ( max_tot_sofar > 0 && tot < min_trigger ) {
				break;
		}
	}
	val_front.set_internal_value(mm_of_max_tot_sofar);
	val_backtop.set_internal_value(mm_of_max_tot_sofar + dwindow);   // + dwindow?
	val_backbottom.set_internal_value(val_backtop.internal_value);

	return;
}

std::string
MmttServer::LoadPatchJson(std::string jstr)
{
	_regionsDefinedByPatch = false;

	NosuchDebug(1,"LoadPatchJson start");
	cJSON *json = cJSON_Parse(jstr.c_str());
	if ( ! json ) {
		NosuchDebug("Unable to parse json in patch file!?  json= %s\n",jstr.c_str());
		return "Unable to parse json in patch file";
	}
	for ( map<std::string, MmttValue*>::iterator vit=mmtt_values.begin(); vit != mmtt_values.end(); vit++ ) {
		std::string nm = vit->first;
		MmttValue* v = vit->second;
		NosuchDebug(1,"Looking for nm=%s in json",nm.c_str());
		cJSON *jval = cJSON_GetObjectItem(json,nm.c_str());
		if ( jval == NULL ) {
			if ( nm.find_first_of("show") != 0
				&& nm.find_first_of("blob_filter") != 0
				&& nm.find_first_of("confidence") != 0
				) {
				NosuchDebug("No value for '%s' in patch file!?\n",nm.c_str());
			}
			continue;
		}
		if ( jval->type != cJSON_Number ) {
			NosuchDebug("The type of '%s' in patch file is wrong!?\n",nm.c_str());
			continue;
		}
		if ( nm == "tilt" ) {
			NosuchDebug(1,"ignoring tilt value in patch");
			continue;
		}
		NosuchDebug(1,"Patch file value for '%s' is '%lf'\n",nm.c_str(),jval->valuedouble);
		v->set_internal_value(jval->valuedouble);
		_updateValue(nm,v);
	}

	cJSON *regionsval = cJSON_GetObjectItem(json,"regions");
	if ( regionsval ) {
		_regionsDefinedByPatch = true;
		// Use regions as defined in the patch
		if ( regionsval->type != cJSON_Array ) {
			return("The type of regions in patch file is wrong!?\n");
		}
		int nregions = cJSON_GetArraySize(regionsval);
		NosuchDebug(1,"Mmtt nregions=%d",nregions);
		int regionid = _new_regions.size();
		for ( int n=0; n<nregions; n++ ) {
			cJSON *rv = cJSON_GetArrayItem(regionsval,n);

			cJSON *c_name = cJSON_GetObjectItem(rv,"name");
			if ( c_name == NULL ) { return("Missing name value in patch file!?"); }
			if ( c_name->type != cJSON_String ) { return("The type of name in patch file is wrong!?\n"); }

			cJSON *c_first_sid = cJSON_GetObjectItem(rv,"first_sid");
			if ( c_first_sid == NULL ) { return("Missing first_sid value in patch file!?"); }
			if ( c_first_sid->type != cJSON_Number ) { return("The type of first_sid in patch file is wrong!?\n"); }

			cJSON *c_x = cJSON_GetObjectItem(rv,"x");
			if ( c_x == NULL ) { return("Missing x value in patch file!?"); }
			if ( c_x->type != cJSON_Number ) { return("The type of x in patch file is wrong!?\n"); }

			cJSON *c_y = cJSON_GetObjectItem(rv,"y");
			if ( c_y == NULL ) { return("Missing y value in patch file!?"); }
			if ( c_y->type != cJSON_Number ) { return("The type of y in patch file is wrong!?\n"); }

			cJSON *c_width = cJSON_GetObjectItem(rv,"width");
			if ( c_width == NULL ) { return("Missing width value in patch file!?"); }
			if ( c_width->type != cJSON_Number ) { return("The type of width in patch file is wrong!?\n"); }

			cJSON *c_height = cJSON_GetObjectItem(rv,"height");
			if ( c_height == NULL ) { return("Missing height value in patch file!?"); }
			if ( c_height->type != cJSON_Number ) { return("The type of height in patch file is wrong!?\n"); }

			if ( c_width->valueint > _camWidth ) {
				return("The width of a region is larger than the camera width!?");
			}

			int x = _camWidth - c_x->valueint - c_width->valueint;
			int y = c_y->valueint;
			x += (int)val_shiftx.internal_value;
			y += (int)val_shifty.internal_value;
			if (x < 0) {
				x = 0;
			}
			else if (x > (_camWidth - 1 - c_width->valueint)) {
				x = _camWidth - 1 - c_width->valueint;
			}
			if (y < 0) {
				y = 0;
			}
			else if (y > (_camHeight - 1 - c_height->valueint)) {
				y = _camHeight - 1 - c_height->valueint;
			}
			CvRect rect = cvRect(x, y, c_width->valueint, c_height->valueint);
			int first_sid = c_first_sid->valueint;
			char *name = c_name->valuestring;
			_new_regions.push_back(new MmttRegion(name,regionid,first_sid,rect));
			NosuchDebug(1,"LoadPatchJson regionid=%d first_sid=%d x,y=%d,%d width,height=%d,%d",
				regionid,first_sid,c_x->valueint,c_y->valueint,c_width->valueint,c_height->valueint);
			regionid++;
		}
	}
	return("");
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
	if ( j && j->type == cJSON_String )
		return j;
	return NULL;
}

std::string
MmttServer::LoadPatch(std::string fname)
{
	startNewRegions();

	ifstream f;
	f.open(fname.c_str());
	if ( ! f.good() ) {
		std::string err = NosuchSnprintf("Warning - unable to open patch file: %s\n",fname.c_str());
		NosuchDebug("%s",err.c_str());  // avoid re-interpreting %'s and \\'s in name
		return err;
	}
	NosuchDebug("Loading patch=%s\n",fname.c_str());
	std::string line;
	std::string jstr;
	while ( getline(f,line) ) {
		NosuchDebug(1,"patch line=%s\n",line.c_str());
		if ( line.size()>0 && line.at(0)=='#' ) {
			NosuchDebug(1,"Ignoring comment line=%s\n",line.c_str());
			continue;
		}
		jstr += line;
	}
	f.close();
	std::string err = LoadPatchJson(jstr);
	if ( err != "" ) {
		return err;
	}

	if ( ! _regionsDefinedByPatch ) {
		std::string fn_patch_image = PaletteDataPath("config/mmtt.ppm");
		NosuchDebug("Reading mask image from %s",fn_patch_image.c_str());
		IplImage* img = cvLoadImage( fn_patch_image.c_str(), CV_LOAD_IMAGE_COLOR );
		if ( ! img ) {
			NosuchDebug("Unable to open or read %s!?",fn_patch_image.c_str());
			return NosuchSnprintf("Unable to read image file: %s",fn_patch_image.c_str());
		}
		copyColorImageToRegionsAndMask((unsigned char *)(img->imageData), _regionsImage, _maskImage, TRUE, TRUE);
		deriveRegionsFromImage();
	} else {
		NosuchDebug(1,"Not reading mask image, since regions defined in patch");
		copyRegionRectsToRegionsImage( _regionsImage, TRUE, TRUE);
	}

	finishNewRegions();

	return "";
}

void MmttServer::startNewRegions()
{
	NosuchDebug("startNewRegions!!");
	_new_regions.clear();
	CvRect all = cvRect(0,0,_camWidth,_camHeight);
	_new_regions.push_back(new MmttRegion("Region0",0,0,all));   // region 0 - the whole image?  Not really used, I think
	_new_regions.push_back(new MmttRegion("Region1",1,0,all));   // region 1 - the background
}

void MmttServer::finishNewRegions()
{
	int nrsize = _new_regions.size();
	int crsize = _curr_regions.size();
	if ( nrsize == crsize ) {
		NosuchDebug(1,"finishNewRegions!! copying first_sid and _name values");
		for ( int i=0; i<nrsize; i++ ) {
			MmttRegion* cr = _curr_regions[i];
			MmttRegion* nr = _new_regions[i];
			nr->_first_sid = cr->_first_sid;
			nr->_name = cr->_name;
		}
	} else {
		if ( nrsize!=0 && crsize!=0 ) {
			NosuchDebug("WARNING!! _curr_regions=%d _new_regions=%d size doesn't match!",crsize,nrsize);
		}
	}
	NosuchDebug("finishNewRegions!! _curr_regions follows");
	_curr_regions = _new_regions;
	for ( int i=0; i<nrsize; i++ ) {
		MmttRegion* cr = _curr_regions[i];
		NosuchDebug("REGION i=%d xy=%d,%d wh=%d,%d",i,cr->_rect.x,cr->_rect.y,cr->_rect.width,cr->_rect.height);
	}
}

void
MmttServer::deriveRegionsFromImage()
{
	int region_id_minx[MAX_REGION_ID+1];
	int region_id_miny[MAX_REGION_ID+1];
	int region_id_maxx[MAX_REGION_ID+1];
	int region_id_maxy[MAX_REGION_ID+1];

	int id;
	for ( id=0; id<=MAX_REGION_ID; id++ ) {
		region_id_minx[id] = _camWidth;
		region_id_miny[id] = _camHeight;
		region_id_maxx[id] = -1;
		region_id_maxy[id] = -1;
	}

	for (int x=0; x<_camWidth; x++ ) {
		for (int y=0; y<_camHeight; y++ ) {
			id = _regionsImage->imageData[x+y*_camWidth];
			if ( id < 0 || id > MAX_REGION_ID ) {
				NosuchDebug("Hey, regionsImage has byte > MAX_REGION_ID? (%d) at %d,%d\n",id,x,y);
				continue;
			}
			if ( x < region_id_minx[id] ) {
				region_id_minx[id] = x;
			}
			if ( y < region_id_miny[id] ) {
				region_id_miny[id] = y;
			}
			if ( x > region_id_maxx[id] ) {
				region_id_maxx[id] = x;
			}
			if ( y > region_id_maxy[id] ) {
				region_id_maxy[id] = y;
			}
		}
	}
	// The loop here doesn't include MAX_REGION_ID, because that's the mask id
	for ( id=2; id<MAX_REGION_ID; id++ ) {
		if ( region_id_maxx[id] == -1 || region_id_maxy[id] == -1 ) {
				continue;
		}
		NosuchDebug(1,"Adding Region=%d  minxy=%d,%d maxxy=%d,%d\n",id-1,
			region_id_minx[id],
			region_id_miny[id],
			region_id_maxx[id],
			region_id_maxy[id]);
		int w = region_id_maxx[id] - region_id_minx[id] + 1;
		int h = region_id_maxy[id] - region_id_miny[id] + 1;
		int first_sid = (id-1) * 1000;
		std::string name = NosuchSnprintf("Region_fromimage_%d", id);
		_new_regions.push_back(new MmttRegion(name,id,first_sid,cvRect(region_id_minx[id],region_id_miny[id],w,h)));
		NosuchDebug(1,"derived region id=%d x,y=%d,%d width,height=%d,%d",
				id,region_id_minx[id],region_id_miny[id],w,h);
	}
}

std::string
MmttServer::SavePatch(const char* id)
{
	if ( ! _regionsfilled ) {
		NosuchDebug("Unable to save patch, regions are not filled yet");
		return error_json(-32700,"Unable to save patch, regions are not filled yet",id);
	}
	std::string fname = PaletteDataPath("config/mmtt_patch_save.json");
	ofstream f_json(fname.c_str());
	if ( ! f_json.is_open() ) {
		NosuchDebug("Unable to open %s!?",fname.c_str());
		return error_json(-32700,"Unable to open patch file",id);
	}
	f_json << "{";

	std::string sep = "\n";
	for ( map<std::string, MmttValue*>::iterator vit=mmtt_values.begin(); vit != mmtt_values.end(); vit++ ) {
		std::string nm = vit->first;
		MmttValue* v = vit->second;
		// These values are only used when reading,
		// ie. the other coords will already be shifted
		if (nm == "shiftx" || nm == "shifty") {
			continue;
		}
		f_json << sep << "  \"" << nm << "\": " << v->internal_value;
		sep = ",\n";
	}
	f_json << sep << "  \"regions\": [";
	std::string sep2 = "\n";
	for ( vector<MmttRegion*>::iterator ait=_curr_regions.begin(); ait != _curr_regions.end(); ait++ ) {
		MmttRegion* r = *ait;
		// internal id 0 isn't exposed
		if ( r->_first_sid == 0 ) {
			continue;
		}
		f_json << sep2
			<< "    { \"name\": \"" << (r->_name) << "\""
			<< ", \"first_sid\": " << (r->_first_sid)
			<< ", \"x\": " << r->_rect.x
			<< ", \"y\": " << r->_rect.y
			<< ", \"width\": " << r->_rect.width
			<< ", \"height\": " << r->_rect.height
			<< "}";
		sep2 = ",\n";
	}
	f_json << "\n  ]\n";
	f_json << "\n}\n";
	f_json.close();
	NosuchDebug("Saved patch: %s",fname.c_str());

	std::string fn_patch_image = PaletteDataPath("config/mmtt_patch_save.ppm");

	copyRegionsToColorImage(_regionsImage,(unsigned char *)(_tmpRegionsColor->imageData),TRUE,TRUE,TRUE);
	if ( !cvSaveImage(fn_patch_image.c_str(),_tmpRegionsColor) ) {
		NosuchDebug("Could not save ppm: %s\n",fn_patch_image.c_str());
	} else {
		NosuchDebug("Saved ppm: %s",fn_patch_image.c_str());
	}
	return ok_json(id);
}

void
MmttServer::registrationStart()
{
	CvConnectedComp connected;

	// Grab the Mask
	cvCvtColor( _ffImage, _tmpGray, CV_BGR2GRAY );
	if ( doSmooth ) {
		cvSmooth( _tmpGray, _tmpGray, CV_GAUSSIAN, 9, 9);
	}
	cvThreshold( _tmpGray, _maskImage, val_blob_param1.internal_value, 255, CV_THRESH_BINARY );
	cvDilate( _maskImage, _maskImage, NULL, MaskDilateAmount );
	NosuchDebug(1,"GRABMASK has been done!");
	cvCopy(_maskImage,_regionsImage);
	NosuchDebug(1,"GRABMASK copied to Regions");

	// Go along all the outer edges of the image and fill any black (i.e. background)
	// pixels with the first region color. 
	int x;
	int y = 0;
	for (x=0; x<_camWidth; x++ ) {
		if ( _regionsImage->imageData[x+y*_camWidth] == 0 ) {
			cvFloodFill(_regionsImage, cvPoint(x,y), cvScalar(currentRegionValue), cvScalarAll(0), cvScalarAll(0), &connected);
		}
	}
	y = _camHeight-1;
	for (x=0; x<_camWidth; x++ ) {
		if ( _regionsImage->imageData[x+y*_camWidth] == 0 ) {
			cvFloodFill(_regionsImage, cvPoint(x,y), cvScalar(currentRegionValue), cvScalarAll(0), cvScalarAll(0), &connected);
		}
	}
	x = 0;
	for (int y=0; y<_camHeight; y++ ) {
		if ( _regionsImage->imageData[x+y*_camWidth] == 0 ) {
			cvFloodFill(_regionsImage, cvPoint(x,y), cvScalar(currentRegionValue), cvScalarAll(0), cvScalarAll(0), &connected);
		}
	}
	x = _camWidth-1;
	for (int y=0; y<_camHeight; y++ ) {
		if ( _regionsImage->imageData[x+y*_camWidth] == 0 ) {
			cvFloodFill(_regionsImage, cvPoint(x,y), cvScalar(currentRegionValue), cvScalarAll(0), cvScalarAll(0), &connected);
		}
	}

	startNewRegions();
	currentRegionValue = 2;
}

void
MmttServer::doRegistration()
{
	NosuchDebug(1,"doRegistration state=%d",registrationState);

	if ( registrationState == 100 ) {
		currentRegionValue = 1;
		val_showregionmap.set_internal_value(1.0);
		// Figure out the depth of the Mask
		doDepthRegistration();
		registrationState = 110;  // Start registering right away, but wait a couple frames
		return;
	}

	if ( registrationState >= 110 && registrationState < 120 ) {
		registrationState++;
		if ( registrationState == 113 ) {
			// Continue on to the manual registration process
			registrationState = 120;
		}
		return;
	}

	if ( registrationState == 120 ) {
		NosuchDebug(1,"Starting registration!\n");
		registrationStart();
		// Continue the manual registration process (wait for someone
		// to poke their hand in each region
		registrationState = 199;
		return;
	}

	if ( registrationState == 199 ) {
		registrationContinueManual();
		return;
	}

	if ( registrationState == 300 ) {
		NosuchDebug(1,"State 300");
		// Start auto-poke.  First re-do the depth registration,
		// then start poking the center of each region
		currentRegionValue = 1;
		doDepthRegistration();   // try without, for new file-based autopoke
		// NosuchDebug("State 310");
		if ( continuousAlign ) {
			copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
		}
		registrationState = 310;
		return;
	}

	if ( registrationState >= 310 && registrationState < 320) {
		registrationState++;
		if ( registrationState == 312 ) {
			// NosuchDebug("State 312");
			registrationState = 320;
		}
		if ( continuousAlign ) {
			copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
		}
		return;
	}

	if ( registrationState == 320 ) {
		NosuchDebug(1,"State 320");
		val_showregionmap.set_internal_value(1.0);

		_savedpokes.clear();

		ifstream autopokefile;
		autopokefile.open("autopoke.txt");
		if ( autopokefile.good() ) {
			std::string line;
			// while ( autopokefile.getline(line,sizeof(line)) ) {
			while ( getline(autopokefile,line) ) {
				NosuchDebug("autopoke input line=%s\n",line.c_str());
				std::stringstream ss;
				CvPoint pt;
				ss << line;
				ss >> pt.x;
				ss >> pt.y;
				_savedpokes.push_back(pt);
				NosuchDebug("    pt.x=%d y=%d\n",pt.x,pt.y);
			}
			autopokefile.close();
		} else {
			std::vector<MmttRegion*>::const_iterator it;
			for ( it=_curr_regions.begin(); it!=_curr_regions.end(); it++ ) {
					MmttRegion* r = (MmttRegion*)(*it);
					if ( r->_first_sid <= 1 ) {
						// The first two regions are the background and mask
						continue;
					}
					CvRect rect = r->_rect;
					CvPoint pt = cvPoint(rect.x+rect.width/2,rect.y+rect.height/2);
					NosuchDebug(1,"Region id=%d pt=%d,%d\n",r->_first_sid - 1,pt.x,pt.y);
					_savedpokes.push_back(pt);
			}
		}
		registrationStart();
		NosuchDebug(1,"after registrationStart, going to 330");
		registrationState = 330;
		if ( continuousAlign ) {
			copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
		}
		return;
	}

	if ( registrationState == 330 ) {
		NosuchDebug(1,"State 330, calling doPokeRegistration");
		doPokeRegistration();
		// NosuchDebug("State 340");
		registrationState = 340;
		if ( continuousAlign ) {
			copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
		}
		return;
	}

	if ( registrationState >= 340 && registrationState < 360 ) {
		registrationState++;
		if ( registrationState == 359 || (continuousAlign && registrationState == 343) ) {
			// NosuchDebug("State 359/343");
			bool stopit = true;
			if ( continuousAlign ) {
				stopit = false;
#ifdef THIS_SHOULD_NOT_BE_DONE
				std::string err = LoadPatch(_patchFile);
				if ( err != "" ) {
					NosuchDebug("LoadPatch in continuousAlign failed!?  err=%s",err.c_str());
				}
#endif
				// int needed_regions = currentRegionValue;
				int needed_regions = _curr_regions.size();
				if ( _new_regions.size() == needed_regions ) {
					NosuchDebug("Continuous registration, OKAY (_new_regions.size=%d needed_regions=%d continuousAlignOkayCount=%d)",_new_regions.size(),needed_regions,continuousAlignOkayCount);
					continuousAlignOkayCount++;
					if ( continuousAlignOkayCount > continuousAlignOkayEnough ) {
						// We've had enough Okay registrations, so stop the continuousAlign registration
						NosuchDebug("STOPPING continuousAlign registration, continuousAlignOkayCount=%d",continuousAlignOkayCount);
						continuousAlign = false;
						stopit = true;
					}
				} else {
					NosuchDebug("Continuous registration, NOT OKAY (missing %d regions)",needed_regions - _new_regions.size());
					continuousAlignOkayCount = 0;
				}
			}
			if ( stopit ) {
				_stop_registration();
			} else {
				registrationState = 300;  // restart registration
			}
		}
		copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
		return;
	}

	NosuchDebug("Hey!  Unexpected registrationState: %d\n",registrationState);
	return;

}

void
MmttServer::doPokeRegistration()
{
	NosuchDebug(1,"doPokeRegistration");
	// check _savedpokes
	std::vector<CvPoint>::const_iterator it;
	for ( it=_savedpokes.begin(); it!=_savedpokes.end(); it++ ) {
			CvPoint pt = (CvPoint)(*it);
			// NosuchDebug("doPokeRegistration, saved point = %d,%d\n",pt.x,pt.y);
			registrationPoke(pt);
			// NosuchDebug("doPokeRegistration B, saved point = %d,%d\n",pt.x,pt.y);
	}
	copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
}

void
MmttServer::registrationContinueManual()
{
	removeMaskFrom(thresh_front);

	_tmpThresh->origin = 1;
	_tmpThresh->imageData = (char *)thresh_front;

	CBlobResult blobs = CBlobResult(_tmpThresh, NULL, 0);
	NosuchDebug("Registration continue blobs = %d\n",blobs.GetNumBlobs());

	blobs.Filter( blobs, B_EXCLUDE, CBlobGetArea(), B_GREATER, val_blob_maxsize.internal_value );
	blobs.Filter( blobs, B_EXCLUDE, CBlobGetArea(), B_LESS, val_blob_minsize.internal_value );

	CvRect bigRect = cvRect(0,0,0,0);
	int bigarea = 0;
	for ( int i=0; i<blobs.GetNumBlobs(); i++ ) {
		CBlob *b = blobs.GetBlob(i);
		CvRect r = b->GetBoundingBox();

		CvPoint pt = cvPoint(r.x+r.width/2,r.y+r.height/2);
		// If the middle of the blob is already an assigned region (or the frame), ignore it
		unsigned char v = _regionsImage->imageData[pt.x+pt.y*_camWidth];

		// This hack for auto-stopping the registration process only works if there's more than one region to set
		if ( v == 0 || v < (currentRegionValue-1) ) {
			int area = r.width*r.height;
			// For crosshair detection, min blob size is greater than normal
			if ( area > bigarea && area > (5*val_blob_minsize.internal_value) ) {
				bigRect = r;
				bigarea = area;
			}
		}
	}
	if ( bigarea != 0 ) {
		registrationPoke(cvPoint(bigRect.x + bigRect.width/2, bigRect.y + bigRect.height/2));
	}

	// SHOW ANY EXISTING AREAS
	copyRegionsToColorImage(_regionsImage,_ffpixels,FALSE,FALSE,FALSE);
	return;
}

void
MmttServer::registrationPoke(CvPoint pt)
{
	// If this point isn't already set to an existing region (and isn't the frame),
	// make it a new region
	unsigned char v = _regionsImage->imageData[pt.x+pt.y*_camWidth];
	if ( v == 0 ) {
		// If the point is black, we've got a new region
		CvConnectedComp connected;
		cvFloodFill(_regionsImage, pt, cvScalar(currentRegionValue), cvScalarAll(0), cvScalarAll(0), &connected);
		int first_sid = (currentRegionValue-1) * 1000;

		std::string name = NosuchSnprintf("Region%d", currentRegionValue);
		_new_regions.push_back(new MmttRegion(name,currentRegionValue,first_sid,connected.rect));
		CvRect r = connected.rect;
		NosuchDebug(1,"Creating new region %d at minxy=%d,%d maxxy=%d,%d\n",currentRegionValue-1,r.x,r.y,r.x+r.width,r.y+r.height);
		currentRegionValue++;
	} else if ( v == 2 ) {
		NosuchDebug(1,"Repeated first region - registration is terminated");
		_stop_registration();
	} else if ( v == 1 ) {
		// If the point is red (the background), we haven't got a new region
		NosuchDebug(1,"v==1 for registrationPoke pt=%d,%d",pt.x,pt.y);
	}
}

int NextAvailableSid = 99;

int
MmttRegion::getAvailableSid() {
	// The session ID has to be globally unique, since I'm sending it to the Palette software
	// that wants globally unique values that aren't reused.
	return NextAvailableSid++;
}

void
MmttServer::removeMaskFrom(uint8_t* pixels)
{
	unsigned char *maskdata = (unsigned char *)_maskImage->imageData;
	int i = 0;
	for (int x=0; x<_camWidth; x++ ) {
		for (int y=0; y<_camHeight; y++ ) {
			unsigned char g = maskdata[i];
			if ( g != 0 ) {
				pixels[i] = 0;
			}
			i++;
		}
	}
}

typedef struct FloatPoint {
	float x;
	float y;
} FloatPoint;


FloatPoint
relativeToRegion(MmttRegion* r, CvPoint xy) {
	FloatPoint rxy = FloatPoint{ (float)(xy.x - r->_rect.x), (float)(xy.y - r->_rect.y) };
	// normalize to 0-1
	rxy.x = rxy.x / r->_rect.width;
	rxy.y = rxy.y / r->_rect.height;
	// Y is reversed?
	rxy.y = 1.0f - rxy.y;
	return rxy;
}

void
MmttServer::analyzePixels()
{
	if ( val_debug.internal_value ) NosuchDebug("MmttServer::analyzePixels\n");
	if ( registrationState > 0 ) {
		doRegistration();
		return;
	}

	// Osc messages don't get sent when we're doing registration

	OscBundle bundle;
	bundle.clear();

	if ( doSmooth ) {
		cvSmooth( _tmpGray, _tmpGray, CV_GAUSSIAN, 9, 9);
	}

	if ( val_usemask.internal_value ) {
		removeMaskFrom(thresh_front);
	}

	_tmpThresh->origin = 1;
	_tmpThresh->imageData = (char *)thresh_front;
	_newblobresult = new CBlobResult(_tmpThresh, NULL, 0);

	// NosuchDebug("blobs = %d\n",newblobs->GetNumBlobs());

	if ( val_blob_filter.internal_value != 0.0 ) {
		// If the value of blob_maxsize is 1.0 (the maximum external value), turn off max filtering
		if ( val_blob_maxsize.external_value != 1.0 ) {
			_newblobresult->Filter( *_newblobresult, B_EXCLUDE, CBlobGetArea(), B_GREATER, val_blob_maxsize.internal_value );
		}
		_newblobresult->Filter( *_newblobresult, B_EXCLUDE, CBlobGetArea(), B_LESS, val_blob_minsize.internal_value );
	}

	int numblobs = _newblobresult->GetNumBlobs();

	// XXX - should really make these static/global so they don't get re-allocated every time, just clear them
	// XXX - BUT, make sure they get re-initialized (e.g. blob_sid's should start out as -1).
	std::vector<MmttRegion*> blob_region(numblobs,NULL);
	std::vector<CvPoint> blob_center(numblobs);
	std::vector<int> blob_sid(numblobs,-1);
	std::vector<CvRect> blob_rect(numblobs);

	// NosuchDebug("ANALYZE START=============================================================\n");
	// Go through the blobs and identify the region each is in

	_framenum++;
	_fpscount++;

	int nregions = _curr_regions.size();
	for ( int i=0; i<numblobs; i++ ) {
		CBlob *blob = _newblobresult->GetBlob(i);
		CvRect r = blob->GetBoundingBox();   // this is expensive, I think
		blob_rect[i] = r;
		CvPoint center = cvPoint(r.x + r.width/2, r.y + r.height/2);
		blob_center[i] = center;
		unsigned char g = _regionsImage->imageData[center.x+center.y*_camWidth];
		// 0 is the background color
		// 1 is the "filled in" background (starting from the edges)
		if ( g != 0 ) {
			NosuchDebug(1,"blob num = %d  g=%d  area=%.3f",i,g,blob->Area());
		}
		if ( g > 1 && g < MASK_REGION_ID ) {
			if ( g >= nregions ) {
				NosuchDebug("Hey, g (%d) is greater than number of regions (%d)\n",g,nregions);
			} else {
				blob_region[i] = _curr_regions[g];
			}
		}
	}

	// For each region...
	for ( vector<MmttRegion*>::iterator ait=_curr_regions.begin(); ait != _curr_regions.end(); ait++ ) {
		MmttRegion* r = *ait;

		// Scan existing sessions for the region
		for ( map<int,MmttSession*>::iterator it = r->_sessions.begin(); it != r->_sessions.end(); ) {

			int sid = (*it).first;
			MmttSession* sess = (*it).second;

			double mindist = 999999.0;
			int mindist_i = -1;

			// Find the closest blob to the session's center
			for ( int i=0; i<numblobs; i++ ) {
				if ( blob_region[i] != r ) {
					// blob isn't in the region we're looking at
					continue;
				}
				if ( blob_sid[i] >= 0 ) {
					// blob has already been assigned to a session
					continue;
				}
				CvPoint blobcenter = blob_center[i];
				int dx = abs(blobcenter.x - sess->_center.x);
				int dy = abs(blobcenter.y - sess->_center.y);
				double dist = sqrt(double(dx*dx+dy*dy));
				if ( dist < mindist ) {
					mindist = dist;
					mindist_i = i;
				}
			}
			if ( mindist_i >= 0 ) {
				// Update the session with the new blob
				// NosuchDebug("   Updating session sid=%d with blob i=%d\n",sid,mindist_i);
				CBlob *blob = _newblobresult->GetBlob(mindist_i);

				sess->_blob = blob;
				sess->_center = blob_center[mindist_i];

				// r->_sessions[sid] = new MmttSession(blob,sess->_frame_born);

				CvPoint xy = sess->_center;
				FloatPoint rxy = relativeToRegion(r, xy);
				float z = float(sess->_depth_mm);

				addCursorEvent(bundle, "drag", r->_name, sid, rxy.x, rxy.y, z, (float)blob->Area());

				blob_sid[mindist_i] = sid;
				it++;
			} else {
				// No blob found for this session, remove session
				// NosuchDebug("   No blob found, Erasing Region=%d Session=%d\n",r->_id - 1,sid);
				map<int,MmttSession*>::iterator erase_it = it;
				it++;

				CvPoint xy = sess->_center;
				FloatPoint rxy = relativeToRegion(r, xy);
				float z = sess->_depth_normalized;
				addCursorEvent(bundle, "up", r->_name, sid, rxy.x, rxy.y, z, 0.0);

				delete sess;
				r->_sessions.erase(erase_it);

			}
		}
	}

	// Go back through the blobs. Any that are not attached to an existing session id
	// will be attached to a new session.  This is also where we compute
	// the depth of each blob/session.

	int nactive = 0;
	bool didtitle = FALSE;
	for ( int i=0; i<numblobs; i++ ) {
		MmttRegion* r = blob_region[i];
		if ( r == NULL ) {
			continue;
		}
		CBlob *blob = _newblobresult->GetBlob(i);

		// Go through the blob and get average depth
		float depthtotal = 0.0f;
		float depth_adjusted_total = 0.0f;
		int depthcount = 0;
		CvRect blobrect = blob->GetBoundingBox();
		int endy = blobrect.y + blobrect.height;

		// XXX - THIS CODE NEEDS Optimization (probably)

		for ( int y=blobrect.y; y<endy; y++ ) {
			int yi = y * _camWidth + blobrect.x;

			float backval = (float)(val_backtop.internal_value
				+ (val_backbottom.internal_value - val_backtop.internal_value)
				* (float(y)/_camHeight));

			for ( int dx=0; dx<blobrect.width; dx++ ) {
				int mm = depthmm_front[yi+dx];
				if ( mm == 0 || mm < val_front.internal_value || mm > backval )
					continue;
				depthtotal += mm;
				depth_adjusted_total += (backval-mm);
				depthcount++;
			}
		}
		if ( depthcount == 0 ) {
			continue;
		}

		float depthavg = depthtotal / depthcount;
		float depth_adjusted_avg = depth_adjusted_total / depthcount;

		bool isnew = false;
		int sid = blob_sid[i];
		if ( sid < 0 ) {
			// New session!
			int new_sid = r->getAvailableSid();
			NosuchDebug(1,"New Session new_sid=%d!",new_sid);
			MmttSession* sess = new MmttSession(blob,blob_center[i],_framenum);
			r->_sessions[new_sid] = sess;
			blob_sid[i] = new_sid;
			sid = new_sid;
			isnew = true;

			CvPoint xy = sess->_center;
			FloatPoint rxy = relativeToRegion(r, xy);
			// float z = float(sess->_depth_mm);
			float z = float(depthavg + 0.5f);
			addCursorEvent(bundle, "down", r->_name, sid, rxy.x, rxy.y, z, (float)blob->Area());
		}

		r->_sessions[sid]->_depth_mm = (int)(depthavg + 0.5f);
		r->_sessions[sid]->_depth_normalized = depth_adjusted_avg / 1000.0f;

		// NosuchDebug("r->sessions sid=%d depth_normalized=%f",sid,r->_sessions[sid]._depth_normalized);

		if ( depthavg > 0 ) {
			double ar = blobrect.width*blobrect.height;
			NosuchDebug(1,"BLOB sid=%d area=%d  depth count=%d avg=%d adjusted_avg=%d\n",sid,ar,depthcount,depthavg,depth_adjusted_avg);
			if ( ar > 0.0 ) {
				CBlobContour *contour = blob->GetExternalContour();
				CvSeq* points = contour->GetContourPoints();
				NosuchDebug(2,"Blob i=%d contour=%d points->total=%d",i,(int)contour,points->total);
				CvPoint pt0;
				for(int i = 0; i < points->total; i++)
				{
					pt0 = *CV_GET_SEQ_ELEM( CvPoint, points, i );
					NosuchDebug(2,"i=%d pt=%d,%d",i,pt0.x,pt0.y);
				}
			}
		}
		nactive++;
	}

	if ( _oldblobresult ) {
		delete _oldblobresult;
	}
	_oldblobresult = _newblobresult;
	_newblobresult = NULL;

	if ( bundle.getMessageCount() > 0 ) {
		sendCursorEvents(bundle);
	}
}

void
MmttServer::showBlobSessions()
{
	for ( vector<MmttRegion*>::iterator ait=_curr_regions.begin(); ait != _curr_regions.end(); ait++ ) {
		MmttRegion* r = *ait;
		for ( map<int,MmttSession*>::iterator it = r->_sessions.begin(); it != r->_sessions.end(); it++ ) {
			int sid = (*it).first;
			MmttSession* sess = r->_sessions[sid];

			// Sessions for which blob is NULL are created by the Leap
			if ( sess->_blob ) {
				CvRect blobrect = sess->_blob->GetBoundingBox();
				CvScalar c = colorOfSession(sid);
				int thick = 2;
				cvRectangle(_ffImage, cvPoint(blobrect.x,blobrect.y),
					cvPoint(blobrect.x+blobrect.width-1,blobrect.y+blobrect.height-1),
					c,thick,8,0);
			}
		}
	}
}

void
MmttServer::showRegionHits()
{
	std::vector<MmttRegion*>::const_iterator it;
	for ( it=_curr_regions.begin(); it != _curr_regions.end(); it++ ) {
		MmttRegion* r = (MmttRegion*) *it;
		if ( r->_sessions.size() > 0 ) {
			CvRect arect = r->_rect;
			CvScalar c = CV_RGB(128,128,128);
			c = region2cvscalar[r->id];
			NosuchDebug(1,"showRegionHits id=%d c=0x%x",r->id,c);
			int thick = 1;
			cvRectangle(_ffImage, cvPoint(arect.x,arect.y), cvPoint(arect.x+arect.width-1,arect.y+arect.height-1), c,thick,8,0);
		}
	}
}

void
MmttServer::showRegionRects()
{
	std::vector<MmttRegion*>::const_iterator it;
	int nregions = _curr_regions.size();
	for ( int n=2; n<nregions; n++ ) {
		MmttRegion* r = _curr_regions[n];
		CvRect arect = r->_rect;
		CvScalar c = region2cvscalar[r->id];
		int thick = 1;
		cvRectangle(_ffImage, cvPoint(arect.x,arect.y),
cvPoint(arect.x + arect.width - 1, arect.y + arect.height - 1), c, thick, 8, 0);
// NosuchDebug("Showing region %d x,y=%d,%d  width,height=%d,%d",r->_id,arect.x,arect.y,arect.width,arect.height);
	}
}

void
MmttServer::showMask()
{
	for (int x = 0; x < _camWidth; x++) {
		for (int y = 0; y < _camHeight; y++) {
			int i = x + y * _camWidth;
			unsigned char g = _maskImage->imageData[i];
			_ffpixels[i * 3 + 0] = g;
			_ffpixels[i * 3 + 1] = g;
			_ffpixels[i * 3 + 2] = g;
		}
	}
}

static void
write_span(CvPoint spanpt0, CvPoint spanpt1, OscMessage& msg, CvRect regionrect)
{
	// NosuchDebug(2,"Writing span pt0=%d,%d  pt1=%d,%d  region=xy=%d,%d wh=%d,%d",spanpt0.x,spanpt0.y,spanpt1.x,spanpt1.y,regionrect.x,regionrect.y,regionrect.width,regionrect.height);
	if (spanpt0.x > spanpt1.x) {
		// NosuchDebug(2,"Swapping x values");
		int tmp = spanpt0.x;
		spanpt0.x = spanpt1.x;
		spanpt1.x = tmp;
	}
	int spandx = spanpt1.x - spanpt0.x;
	int spandy = spanpt1.y - spanpt0.y;
	int dx = spanpt0.x - regionrect.x;
	int dy = spanpt0.y - regionrect.y;
	if (spandy != 0) {
		NosuchDebug("Hey, something is wrong, y of spanpt0 and spanpt1 should be the same!");
		return;
	}
	float x = float(dx);
	float y = float(dy);
	normalize_region_xy(x, y, regionrect);

	float scaledlength = float(spandx + 1) / (regionrect.width);
	if (scaledlength > 1.0) {
		scaledlength = 1.0;
	}

	msg.addFloatArg(x);
	msg.addFloatArg(y);
	msg.addFloatArg(scaledlength);
}

double
bound_it(double v) {
	if (v < 0.0) {
		return 0.0;
	}
	if (v > 1.0) {
		return 1.0;
	}
	return v;
}

bool
point_is_in(CvPoint pt, CvRect& rect) {
	if (pt.x < rect.x) return false;
	if (pt.x > (rect.x + rect.width)) return false;
	if (pt.y < rect.y) return false;
	if (pt.y > (rect.y + rect.height)) return false;
	return true;
}

int
make_hfid(int hid, int fid)
{
	return hid * 100000 + fid;
}

void
MmttServer::addCursorEvent(OscBundle &bundle, std::string downdragup, std::string region, int sid, float x, float y, float z, float area) {
	OscMessage msg;

	NosuchDebug(1,"addCursorEvent ddu=%s x=%f y=%f z=%f area=%f", downdragup.c_str(), x, y, z, area);

	float backmiddle = float(val_backbottom.internal_value + val_backtop.internal_value) / 2.0f;
	if (z < 1.0 || downdragup == "up") {
		z = 0.0;
	}
	else {
		// If it's beyond the front value, clip it
		// NosuchDebug("CLIPPING z=%f to front=%f\n", z, val_front.internal_value);
		if (z < val_front.internal_value) {
			z = float(val_front.internal_value);
		}

		// The z value coming is in the distance to the camera,
		// and we want the distance from the frame.
		z = backmiddle - z;
		float range = float(backmiddle - val_front.internal_value);
		z = z / range;
		if (z < 0.0) {
			z = 0.0;
		} else if ( z > 1.0 ) {
			z = 1.0;
		}
	}
	msg.clear();
	msg.setAddress("/cursor");
	msg.addStringArg(downdragup);

	// std::string cid = NosuchSnprintf("%s#%d",region.c_str(),sid);
	// msg.addStringArg(cid);
	msg.addStringArg(region.c_str());  // this is the source name, i.e. A,B,C,D
	msg.addIntArg(sid);  // gets used as gid in the Palette software

	msg.addFloatArg(x);      // x (position)
	msg.addFloatArg(y);      // y (position)
	msg.addFloatArg(z);      // z (position)

	// NosuchDebug("backmiddle=%f  backbottom=%f  backtop=%f  front=%f\n",
	//	backmiddle, float(val_backbottom.internal_value), float(val_backtop.internal_value), val_front.internal_value);
	// NosuchDebug("CursorEvent cid=%s ddu=%s xyz=%f %f %f\n", cid.c_str(), downdragup.c_str(), x, y, z);

	// NosuchDebug("Sending /cursor source=%s gid=%d ddu=%s xyz=%f %f %f\n", region.c_str(), sid, downdragup.c_str(), x, y, z);

	// msg.addFloatArg((float)blobrect.width);   // w (width)
	// msg.addFloatArg((float)blobrect.height);  // h (height)
	// msg.addFloatArg(f);			   // f (area)

	bundle.addMessage(msg);
}

void
MmttServer::sendCursorEvents(OscBundle &bundle) {
	SendAllOscClients(bundle,_Clients);
	SendOscToAllWebSocketClients(bundle);
}

void
MmttServer::copyRegionsToColorImage(IplImage* regions, unsigned char* pixels, bool overwriteBackground, bool reverseColor, bool reverseX)
{
	CvScalar c = cvScalar(0);
	for (int x=0; x<_camWidth; x++ ) {
		for (int y=0; y<_camHeight; y++ ) {
			int i = x + y*_camWidth;
			unsigned char g = regions->imageData[i];
			if ( ! overwriteBackground && g == 0 ) {
				continue;
			}
			// bit of a hack - sometimes the mask region id is MASK_REGION_ID, sometimes it's 255
			if ( g == 255 ) {
				g = MASK_REGION_ID;
			} else if ( g > MAX_REGION_ID ) {
				NosuchDebug("Hey! invalid region value (%d)\n",g);
				g = MASK_REGION_ID;
			}
			c = region2cvscalar[g];
			if ( reverseX ) {
				i = (_camWidth-1-x) + y*_camWidth;
			}
			if ( reverseColor ) {
				pixels[i*3 + 0] = (unsigned char)c.val[2];
				pixels[i*3 + 1] = (unsigned char)c.val[1];
				pixels[i*3 + 2] = (unsigned char)c.val[0];
			} else {
				pixels[i*3 + 0] = (unsigned char)c.val[0];
				pixels[i*3 + 1] = (unsigned char)c.val[1];
				pixels[i*3 + 2] = (unsigned char)c.val[2];
			}
		}
	}
}

void
MmttServer::clearImage(IplImage* image)
{
	for (int x=0; x<_camWidth; x++ ) {
		for (int y=0; y<_camHeight; y++ ) {
			int i = x + y*_camWidth;
			image->imageData[i] = 0;   // black
		}
	}
}

void
MmttServer::copyColorImageToRegionsAndMask(unsigned char *pixels, IplImage* regions, IplImage* mask, bool reverseColor, bool reverseX)
{
	_regionsfilled = true;
	CvScalar c = cvScalar(0);
	for (int x=0; x<_camWidth; x++ ) {
		for (int y=0; y<_camHeight; y++ ) {
			int i = x + y*_camWidth;

			unsigned char r;
			unsigned char g;
			unsigned char b;

			if ( reverseColor ) {
				r = pixels[i*3 + 0];
				g = pixels[i*3 + 1];
				b = pixels[i*3 + 2];
			} else {
				r = pixels[i*3 + 2];
				g = pixels[i*3 + 1];
				b = pixels[i*3 + 0];
			}

			int region_id = regionOfColor(r,g,b);

			if ( reverseX ) {
				i = (_camWidth-1-x) + y*_camWidth;
			}
			regions->imageData[i] = region_id;

			if ( region_id == MASK_REGION_ID ) {
				mask->imageData[i] = (char)255;   // white
			} else {
				mask->imageData[i] = 0;   // black
			}
		}
	}
}

void
MmttServer::copyRegionRectsToRegionsImage(IplImage* regions, bool reverseColor, bool reverseX)
{
	_regionsfilled = true;

	int thick = CV_FILLED;

	for (int x=0; x<_camWidth; x++ ) {
		for (int y=0; y<_camHeight; y++ ) {
				int i = x + y*_camWidth;
				regions->imageData[i] = 0;
		}
	}

	int nregions = _curr_regions.size();
	for (int region_id=2; region_id<nregions; region_id++ ) {
		MmttRegion* r = _curr_regions[region_id];

		CvScalar c = region2cvscalar[region_id];

		CvRect rect = r->_rect;

		int x0 = rect.x;
		int x1 = rect.x + rect.width;
		int y0 = rect.y;
		int y1 = rect.y + rect.height;
		for (int x=x0; x<x1; x++ ) {
			for (int y=y0; y<y1; y++ ) {
				int i = x + y*_camWidth;
#if 0
				if ( reverseX ) {
					i = (_camWidth-1-x) + y*_camWidth;
				}
#endif
				regions->imageData[i] = region_id;
			}
		}
	}
}

int
MmttServer::regionOfColor(int r, int g, int b)
{
	for ( int n=0; n<sizeof(region2color); n++ ) {
		RGBcolor color = region2color[n];
		// NOTE: swapping the R and B values, since OpenCV defaults to BGR
		if ( (r == color.b) && (g == color.g) && (b == color.r) ) {
			return n;
		}
	}
	return 0;
}



CvScalar
MmttServer::colorOfSession(int g)
{
	CvScalar c;
	switch (g) {
	case 0: c = CV_RGB(255,0,0); break;
	case 1: c = CV_RGB(0,255,0); break;
	case 2: c = CV_RGB(0,0,255); break;
	case 3: c = CV_RGB(255,255,0); break;
	case 4: c = CV_RGB(0,255,255); break;
	case 5: c = CV_RGB(255,0,255); break;
	default: c = CV_RGB(128,128,128); break;
	}
	return c;
}

MmttServer*
MmttServer::makeMmttServer()
{
	// Default debugging stuff
	NosuchDebugLevel = 0;   // minimal messages
	NosuchDebugToConsole = TRUE;
	NosuchDebugToLog = TRUE;
	NosuchDebugAutoFlush = TRUE;
	NosuchAppName = "Space Manifold";

	std::string logdir = PaletteDataPath("logs");
	NosuchDebugSetLogDirFile(logdir,"mmtt_kinect.log");

	NosuchDebug("Hello mmtt_kinect logdir=%s\n",logdir.c_str());

	MmttServer* server = new MmttServer();

	NosuchDebug("after new MmttServer logdir=%s\n",logdir.c_str());

	std::string stat = server->status();
	if ( stat != "" ) {
		NosuchDebug("Failure in creating MmttServer? status=%s",stat.c_str());
		return NULL;
	}

	NosuchDebug("after status\n");

	if ( NosuchNetworkInit() ) {
		NosuchDebug("Unable to initialize networking?");
		return NULL;
	}

	NosuchDebug("before startHttpThread in makeMmttServer\n");

	startHttpThread(server);

	NosuchDebug("after startHttpThread in makeMmttServer\n");

	return server;
}

std::string
MmttForwardSlash(std::string filepath) {
	size_t i;
	while ( (i=filepath.find("\\")) != filepath.npos ) {
		filepath.replace(i,1,"/");
	}
	return filepath;
}

bool isFullPath(std::string path) {
	if ( path.find(":") == 2 ) {
		return true;
	}
	return false;
}
