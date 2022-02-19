#include <FFGL.h>
#include <FFGLLib.h>
#include "FFGLPluginSDK.h"
#include "FFGLPluginInfo.h"

#define FFGL_ALREADY_DEFINED
#include "PaletteFFHost.h"

#include <pthread.h>
#include <iostream>
#include <fstream>
#include <strstream>
#include <cstdlib> // for srand, rand
#include <ctime>   // for time

#include "NosuchUtil.h"

// #define FFPARAM_BRIGHTNESS (0)

static CFFGLPluginInfo PluginInfo ( 
	PaletteFFHost::CreateInstance,	// Create method
	"NSPL",		// Plugin unique ID
	"Palette",	// Plugin name	
	1,		// API major version number
	000,		// API minor version number	
	1,		// Plugin major version number
	000,		// Plugin minor version number
	FF_EFFECT,	// Plugin type
	"Space Palette: TUIO-controlled graphics and music",	// description
	"by Tim Thompson - me@timthompson.com" 			// About
);

PaletteFFHost::PaletteFFHost(std::string defaultsfile) : CFreeFrameGLPlugin(), PaletteHost(defaultsfile)
{
	NosuchDebug(1,"PaletteFFHost is being constructed.");
	SetMinInputs(1);
	SetMaxInputs(1);
}

PaletteFFHost::~PaletteFFHost() {
	NosuchDebug(1,"PaletteFFHost is being destroyed!");
}

DWORD PaletteFFHost::GetParameter(DWORD dwIndex)
{
	return FF_FAIL;  // no parameters
}

DWORD PaletteFFHost::SetParameter(const SetParameterStruct* pParam)
{
	return FF_FAIL;  // no parameters
}

DWORD PaletteFFHost::ProcessOpenGL(ProcessOpenGLStruct *pGL) {
	return PaletteHostProcessOpenGL(pGL);
}

