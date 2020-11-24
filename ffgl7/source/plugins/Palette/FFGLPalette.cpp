#include "FFGLPalette.h"
#include "PaletteHost.h"
#include <math.h>//floor
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_OSC_PORT,
};

static CFFGLPluginInfo PluginInfo(
	// PluginFactory< FFGLPalette >,// Create method
	FFGLPalette::CreateInstance,         // Create method
	"PL03",                        // Plugin unique ID
	"Palette3",            // Plugin name
	2,                             // API major version number
	1,                             // API minor version number
	1,                             // Plugin major version number
	000,                           // Plugin minor version number
	FF_SOURCE,                     // Plugin type
	"Palette instrument plugin v0.90 #3",// Plugin description
	"by Tim Thompson"        // About
);

static const char vertexShaderCode[] = R"(#version 410 core
layout( location = 0 ) in vec4 vPosition;
layout( location = 1 ) in vec2 vUV;

uniform vec2 vScale;
uniform vec2 vTranslate;

out vec2 uv;

void main()
{
	gl_Position = vec4((vPosition.x*vScale.x)+vTranslate.x,(vPosition.y*vScale.y)+vTranslate.y,vPosition.z,vPosition.a);
	uv = vUV;
}
)";

static const char fragmentShaderCode[] = R"(#version 410 core
uniform vec4 RGBALeft;
uniform vec4 RGBARight;

in vec2 uv;

out vec4 fragColor;

void main()
{
	fragColor = mix( RGBALeft, RGBARight, uv.x );
}
)";

std::string ffgl_name()
{
	return "Palette";
}

CFFGLPluginInfo& ffgl_plugininfo()
{
	return PluginInfo;
}


FFGLPalette::FFGLPalette(std::string configfile) :
	CFFGLPlugin(),
	PaletteHost( configfile )
{
	// Input properties
	SetMinInputs( 0 );
	SetMaxInputs( 0 );

	// Parameters
	SetParamInfof( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT );

	FFGLLog::LogToHost( "Created Palette" );
}
FFResult FFGLPalette::InitGL( const FFGLViewportStruct* vp )
{
	PaletteHost::InitGL( vp );

	//Use base-class init as success result so that it retains the viewport.
	return CFFGLPlugin::InitGL( vp );
}

bool PaletteFFThreadNameSet = false;

FFResult FFGLPalette::ProcessOpenGL( ProcessOpenGLStruct* pGL )
{
	if( !PaletteFFThreadNameSet )
	{
		NosuchDebugSetThreadName( pthread_self().p, "ProcessOpenGL" );
		PaletteFFThreadNameSet = true;
	}
	return PaletteHostProcessOpenGL( pGL );
}
FFResult FFGLPalette::DeInitGL()
{
	PaletteHost::DeInitGL( );
	return FF_SUCCESS;
}

FFResult FFGLPalette::SetTextParameter( unsigned int index, const char* value )
{
	switch( index )
	{
	case PT_OSC_PORT:
		SetOscPort(std::string( value ));
		return FF_SUCCESS;
	}
	NosuchDebug( "SetTextParameter FAILS?" );
	return FF_FAIL;
}

char* FFGLPalette::GetTextParameter( unsigned int index )
{
	static std::string value;
	switch( index )
	{
	case PT_OSC_PORT:
		value = GetOscPort();
		return (char *)(value.c_str());
	}
	NosuchDebug( "GetTextParameter returns NULL?" );
	return NULL;
}

FFResult FFGLPalette::SetFloatParameter( unsigned int dwIndex, float value )
{
	return FF_FAIL;
}

float FFGLPalette::GetFloatParameter( unsigned int index )
{
	return 0.0f;
}
