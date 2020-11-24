#include "FFGLPalette.h"
#include "PaletteHost.h"
#include <math.h>//floor
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_TJT,
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
	static std::string nm;
	if( PaletteHost::PortOffset == 0 )
	{
		nm = "Palette";
	}
	else
	{
		nm = NosuchSnprintf( "Palette_%d", PaletteHost::PortOffset );
	}
	return nm.c_str();
}

CFFGLPluginInfo& ffgl_plugininfo()
{
	return PluginInfo;
}


FFGLPalette::FFGLPalette(std::string configfile) :
	CFFGLPlugin(),
	PaletteHost( configfile ),
	rgbLeftLocation( -1 ),
	rgbRightLocation( -1 )
{
	// Input properties
	SetMinInputs( 0 );
	SetMaxInputs( 0 );

	hsba2.hue = 0.5f;

	// Parameters
	SetParamInfof( PT_TJT, "OSC Port", FF_TYPE_TEXT );

	FFGLLog::LogToHost( "Created Palette" );
}
FFResult FFGLPalette::InitGL( const FFGLViewportStruct* vp )
{
	NosuchDebug( "HI From FFGLPalette.cpp");
	if( !shader.Compile( vertexShaderCode, fragmentShaderCode ) )
	{
		DeInitGL();
		return FF_FAIL;
	}
	if( !quad.Initialise() )
	{
		DeInitGL();
		return FF_FAIL;
	}
	if( !triangle.Initialise() )
	{
		DeInitGL();
		return FF_FAIL;
	}

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ScopedShaderBinding shaderBinding( shader.GetGLID() );
	rgbLeftLocation  = shader.FindUniform( "RGBALeft" );
	rgbRightLocation = shader.FindUniform( "RGBARight" );

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

	DWORD f = PaletteHostProcessOpenGL( pGL );
	if( f == FF_FAIL )
	{
		return f;
	}

	const char* port = GetTextParameter( PT_TJT );
	if( port != NULL && *port != 0 )
	{
		// NosuchDebug( "port=%s\n", port );
	}

	float rgba2[ 4 ];
	float hue2 = 0.5f;
	hsba2.sat  = 1.0f;
	hsba2.bri  = 1.0f;
	hsba2.alpha  = 1.0f;
	HSVtoRGB( hue2, hsba2.sat, hsba2.bri, rgba2[ 0 ], rgba2[ 1 ], rgba2[ 2 ] );
	rgba2[ 3 ] = hsba2.alpha;

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ScopedShaderBinding shaderBinding( shader.GetGLID() );
	rgba1.red = 1.0f;
	rgba1.green = 1.0f;
	rgba1.blue = 0.0f;
	rgba1.alpha = 1.0f;
	glUniform4f( rgbLeftLocation, rgba1.red, rgba1.green, rgba1.blue, rgba1.alpha );
	glUniform4f( rgbRightLocation, rgba2[ 0 ], rgba2[ 1 ], rgba2[ 2 ], rgba2[ 3 ] );

	GLfloat xscale = random( 0.2f, 0.5f );
	GLfloat yscale = random( 0.2f, 0.5f );
	shader.Set( "vScale", xscale, yscale );
	GLfloat xtranslate = 0.8f;
	GLfloat ytranslate = 0.8f;
	shader.Set( "vTranslate", xtranslate, ytranslate );

	quad.Draw();

	xtranslate = 0.0f;
	ytranslate = 0.0f;
	shader.Set( "vTranslate", xtranslate, ytranslate );
	triangle.Draw();

	return FF_SUCCESS;
}
FFResult FFGLPalette::DeInitGL()
{
	shader.FreeGLResources();
	quad.Release();
	triangle.Release();
	rgbLeftLocation  = -1;
	rgbRightLocation = -1;

	PaletteHost::DeInitGL( );
	return FF_SUCCESS;
}

FFResult FFGLPalette::SetTextParameter( unsigned int index, const char* value )
{
	NosuchDebug( "SetTextParameter index=%d value=%s\n", index, value );
	// There's only one, this will eventually be a switch
	oscport = std::string( value );
	return FF_SUCCESS;
}

char* FFGLPalette::GetTextParameter( unsigned int index )
{
	// There's only one, this will eventually be a switch
	return (char*)(oscport.c_str());
}

FFResult FFGLPalette::SetFloatParameter( unsigned int dwIndex, float value )
{
	switch( dwIndex )
	{
	// There would be some cases here, if we had float parameters
	default:
		return FF_FAIL;
	}

	return FF_SUCCESS;
}

float FFGLPalette::GetFloatParameter( unsigned int index )
{
	switch( index )
	{
	// There would be some cases here, if we had float parameters
	}

	return 0.0f;
}
