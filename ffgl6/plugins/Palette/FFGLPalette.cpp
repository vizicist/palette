#include <FFGL.h>
#include <FFGLLib.h>
#include "FFGLPalette.h"

#include "../../lib/ffgl/utilities/utilities.h"

#include <math.h> //floor

#define FFPARAM_ZDEFAULT  (0)
#define FFPARAM_RANDOM_TOUCH  (1)

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Plugin information
////////////////////////////////////////////////////////////////////////////////////////////////////

static CFFGLPluginInfo PluginInfo (
	FFGLPalette::CreateInstance,	// Create method
	"SPMF",				// Plugin unique ID
	"Palette",			// Plugin name
	1,				// API major version number
	000,				// API minor version number
	1,				// Plugin major version number
	000,				// Plugin minor version number
	FF_SOURCE,			// Plugin type
	"Palette - a visual instrument",	// Plugin description
	"by Tim Thompson - timthompson.com" 	// About
);

std::string ffgl_name() {
	static std::string nm;
	if (PaletteHost::PortOffset == 1) {
		nm = "Palette";
	} else {
		nm = NosuchSnprintf("Palette_%d", PaletteHost::PortOffset);
	}
	return nm.c_str();
}
CFFGLPluginInfo& ffgl_plugininfo() {
	return PluginInfo;
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Constructor and destructor
////////////////////////////////////////////////////////////////////////////////////////////////////

FFGLPalette::FFGLPalette(std::string configfile) : CFreeFrameGLPlugin(), PaletteHost(configfile)
{
	// Input properties
	SetMinInputs(0);
	SetMaxInputs(0);

	SetParamInfo(FFPARAM_ZDEFAULT, "Z Default", FF_TYPE_STANDARD, 0.5f);
	m_zdefault = 0.3f;

    SetParamInfo( FFPARAM_RANDOM_TOUCH, "Poke", FF_TYPE_EVENT, false );
}

FFResult FFGLPalette::InitGL(const FFGLViewportStruct *vp)
{
	NosuchDebug("Palette.InitGL: width,height=%d %d",vp->width,vp->height);
	return FF_SUCCESS;
}

FFResult FFGLPalette::DeInitGL()
{
    return FF_SUCCESS;
}


////////////////////////////////////////////////////////////////////////////////////////////////////
//  Methods
////////////////////////////////////////////////////////////////////////////////////////////////////

bool PaletteFFThreadNameSet = false;

FFResult FFGLPalette::ProcessOpenGL(ProcessOpenGLStruct *pGL)
{
	if (!PaletteFFThreadNameSet) {
		NosuchDebugSetThreadName(pthread_self().p, "ProcessOpenGL");
		PaletteFFThreadNameSet = true;
	}

	return PaletteHostProcessOpenGL(pGL);
}

float FFGLPalette::GetFloatParameter(unsigned int index)
{
	float retValue = 0.0;
	
	switch (index)
	{
		case FFPARAM_ZDEFAULT:
			retValue = m_zdefault;
			break;
		default:
			break;
	}
	
	return retValue;
}

FFResult FFGLPalette::SetFloatParameter(unsigned int dwIndex, float value)
{
	switch (dwIndex)
	{
		case FFPARAM_ZDEFAULT:
			m_zdefault = value;
			break;
		case FFPARAM_RANDOM_TOUCH:
			if ( value > 0.5 ) {
				PaletteHostPoke();
			}
			break;
		default:
			return FF_FAIL;
	}
	
	return FF_SUCCESS;
}



