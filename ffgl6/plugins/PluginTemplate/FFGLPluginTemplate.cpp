#include <FFGL.h>
#include <FFGLLib.h>
#include "FFGL{PLUGINNAME}.h"

#include "../../lib/ffgl/utilities/utilities.h"

#include <math.h> //floor

#define FFPARAM_Hue1  (0)
#define FFPARAM_Hue2  (1)
#define FFPARAM_Saturation  (2)
#define FFPARAM_Brightness  (3)


////////////////////////////////////////////////////////////////////////////////////////////////////
//  Plugin information
////////////////////////////////////////////////////////////////////////////////////////////////////

static CFFGLPluginInfo PluginInfo (
	FFGL{PLUGINNAME}::CreateInstance,	// Create method
	"{PLUGINID}",				// Plugin unique ID
	"{PLUGINNAME}",		        	// Plugin name
	1,					// API major version number
	000,					// API minor version number
	1,					// Plugin major version number
	000,					// Plugin minor version number
	FF_SOURCE,				// Plugin type
	"{PLUGINNAME} plugin",			// Plugin description
	"by Tim Thompson - timthompson.com" 	// About
);


////////////////////////////////////////////////////////////////////////////////////////////////////
//  Constructor and destructor
////////////////////////////////////////////////////////////////////////////////////////////////////

FFGL{PLUGINNAME}::FFGL{PLUGINNAME}()
:CFreeFrameGLPlugin()
{
	// Input properties
	SetMinInputs(0);
	SetMaxInputs(0);

	// Parameters
	SetParamInfo(FFPARAM_Hue1, "Hue 1", FF_TYPE_STANDARD, 0.0f);
	m_Hue1 = 0.0f;

	SetParamInfo(FFPARAM_Hue2, "Hue 2", FF_TYPE_STANDARD, 0.5f);
	m_Hue2 = 0.5f;

	SetParamInfo(FFPARAM_Saturation, "Saturation", FF_TYPE_STANDARD, 1.0f);
	m_Saturation = 1.0f;

	SetParamInfo(FFPARAM_Brightness, "Brightness", FF_TYPE_STANDARD, 1.0f);
	m_Brightness = 1.0f;
}

FFResult FFGL{PLUGINNAME}::InitGL(const FFGLViewportStruct *vp)
{
	return FF_SUCCESS;
}

FFResult FFGL{PLUGINNAME}::DeInitGL()
{
    return FF_SUCCESS;
}


////////////////////////////////////////////////////////////////////////////////////////////////////
//  Methods
////////////////////////////////////////////////////////////////////////////////////////////////////



FFResult FFGL{PLUGINNAME}::ProcessOpenGL(ProcessOpenGLStruct *pGL)
{

	double rgb1[3];
    //we need to make sure the hue doesn't reach 1.0f, otherwise the result will be pink and not red how it should be
	double hue1 = (m_Hue1 == 1.0) ? 0.0 : m_Hue1;
	HSVtoRGB( hue1, m_Saturation, m_Brightness, &rgb1[0], &rgb1[1], &rgb1[2]);

	double rgb2[3];
	double hue2 = (m_Hue2 == 1.0) ? 0.0 : m_Hue2;
	HSVtoRGB( hue2, m_Saturation, m_Brightness, &rgb2[0], &rgb2[1], &rgb2[2]);


	glShadeModel(GL_SMOOTH);
	glBegin(GL_POLYGON);
		glColor3d( rgb1[0], rgb1[1], rgb1[2] );
		glVertex2f(-1.0, -1.0);	// Bottom Left Of The Texture and Quad

		glColor3d( rgb2[0], rgb2[1], rgb2[2] );
		glVertex2f( 1.0, -1.0);	// Bottom Right Of The Texture and Quad

		glColor3d( rgb2[0], rgb2[1], rgb2[2] );
		glVertex2f( 1.0,  1.0);	// Top Right Of The Texture and Quad

		glColor3d( rgb1[0], rgb1[1], rgb1[2] );
		glVertex2f(-1.0,  1.0);	// Top Left Of The Texture and Quad
	glEnd();


	return FF_SUCCESS;
}

float FFGL{PLUGINNAME}::GetFloatParameter(unsigned int index)
{
	float retValue = 0.0;
	
	switch (index)
	{
		case FFPARAM_Hue1:
			retValue = m_Hue1;
			break;
		case FFPARAM_Hue2:
			retValue = m_Hue2;
			break;
		case FFPARAM_Saturation:
			retValue = m_Saturation;
			break;
		case FFPARAM_Brightness:
			retValue = m_Brightness;
			break;
		default:
			break;
	}
	
	return retValue;
}

FFResult FFGL{PLUGINNAME}::SetFloatParameter(unsigned int dwIndex, float value)
{
	switch (dwIndex)
	{
		case FFPARAM_Hue1:
			m_Hue1 = value;
			break;
		case FFPARAM_Hue2:
			m_Hue2 = value;
			break;
		case FFPARAM_Saturation:
			m_Saturation = value;
			break;
		case FFPARAM_Brightness:
			m_Brightness = value;
			break;
		default:
			return FF_FAIL;
	}
	
	return FF_SUCCESS;
}



