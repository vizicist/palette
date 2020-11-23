#include "FFGLPalette.h"
#include "PaletteHost.h"
#include <math.h>//floor
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_RED1,
	PT_GREEN1,
	PT_BLUE1,
	PT_ALP1,

	PT_HUE2,
	PT_SAT2,
	PT_BRI2,
	PT_ALP2,
};

static CFFGLPluginInfo PluginInfo(
	PluginFactory< FFGLPalette >,// Create method
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

FFGLPalette::FFGLPalette() :
	rgbLeftLocation( -1 ),
	rgbRightLocation( -1 )
{
	// Input properties
	SetMinInputs( 0 );
	SetMaxInputs( 0 );

	hsba2.hue = 0.5f;
	// Parameters
	SetParamInfof( PT_RED1, "Red 1", FF_TYPE_RED );
	SetParamInfof( PT_GREEN1, "Green 1", FF_TYPE_GREEN );
	SetParamInfof( PT_BLUE1, "Blue 1", FF_TYPE_BLUE );
	SetParamInfof( PT_ALP1, "Alpha 1", FF_TYPE_ALPHA );

	SetParamInfof( PT_HUE2, "Hue 2", FF_TYPE_HUE );
	SetParamInfof( PT_SAT2, "Saturation 2", FF_TYPE_SATURATION );
	SetParamInfof( PT_BRI2, "Brightness 2", FF_TYPE_BRIGHTNESS );
	SetParamInfof( PT_ALP2, "Alpha 2", FF_TYPE_ALPHA );

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

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ScopedShaderBinding shaderBinding( shader.GetGLID() );
	rgbLeftLocation  = shader.FindUniform( "RGBALeft" );
	rgbRightLocation = shader.FindUniform( "RGBARight" );

	//Use base-class init as success result so that it retains the viewport.
	return CFFGLPlugin::InitGL( vp );
}
FFResult FFGLPalette::ProcessOpenGL( ProcessOpenGLStruct* pGL )
{
	float rgba2[ 4 ];
	float hue2 = ( hsba2.hue == 1.0f ) ? 0.0f : hsba2.hue;
	HSVtoRGB( hue2, hsba2.sat, hsba2.bri, rgba2[ 0 ], rgba2[ 1 ], rgba2[ 2 ] );
	rgba2[ 3 ] = hsba2.alpha;

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ScopedShaderBinding shaderBinding( shader.GetGLID() );
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
	quad.Draw();

	return FF_SUCCESS;
}
FFResult FFGLPalette::DeInitGL()
{
	shader.FreeGLResources();
	quad.Release();
	rgbLeftLocation  = -1;
	rgbRightLocation = -1;

	return FF_SUCCESS;
}

FFResult FFGLPalette::SetFloatParameter( unsigned int dwIndex, float value )
{
	switch( dwIndex )
	{
	case PT_RED1:
		rgba1.red = value;
		break;
	case PT_GREEN1:
		rgba1.green = value;
		break;
	case PT_BLUE1:
		rgba1.blue = value;
		break;
	case PT_ALP1:
		rgba1.alpha = value;
		break;

	case PT_HUE2:
		hsba2.hue = value;
		break;
	case PT_SAT2:
		hsba2.sat = value;
		break;
	case PT_BRI2:
		hsba2.bri = value;
		break;
	case PT_ALP2:
		hsba2.alpha = value;
		break;

	default:
		return FF_FAIL;
	}

	return FF_SUCCESS;
}

float FFGLPalette::GetFloatParameter( unsigned int index )
{
	switch( index )
	{
	case PT_RED1:
		return rgba1.red;
	case PT_GREEN1:
		return rgba1.green;
	case PT_BLUE1:
		return rgba1.blue;
	case PT_ALP1:
		return rgba1.alpha;

	case PT_HUE2:
		return hsba2.hue;
	case PT_SAT2:
		return hsba2.sat;
	case PT_BRI2:
		return hsba2.bri;
	case PT_ALP2:
		return hsba2.alpha;
	}

	return 0.0f;
}
