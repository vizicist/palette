#include "PaletteAll.h"
#include "FFGLPaletteEffect.h"

#include <math.h> // floor
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_BRIGHTNESS_R,
	PT_BRIGHTNESS_G,
	PT_BRIGHTNESS_B,
	PT_OSC_PORT,
};

static CFFGLPluginInfo PluginInfo(
	PluginFactory< FFGLPaletteEffect >,// Create method
	// FFGLPaletteEffect::CreateInstance,
	"PLTE",                      // Plugin unique ID of maximum length 4.
	"Palette Effect",            // Plugin name
	2,                           // API major version number
	1,                           // API minor version number
	1,                           // Plugin major version number
	0,                           // Plugin minor version number
	FF_EFFECT,                   // Plugin type
	"Palette effect, see github.com/vizicist/palette",  // Plugin description
	"by Tim Thompson, me@timthompson.com"      // About
);

static const char _vertexShaderCode[] = R"(#version 410 core
uniform vec2 MaxUV;

layout( location = 0 ) in vec4 vPosition;
layout( location = 1 ) in vec2 vUV;

out vec2 uv;

void main()
{
	gl_Position = vPosition;
	uv = vUV * MaxUV;
}
)";

static const char _fragmentShaderCode[] = R"(#version 410 core
uniform sampler2D InputTexture;
uniform  vec3 Brightness;

in vec2 uv;

out vec4 fragColor;

void main()
{
	vec4 color = texture( InputTexture, uv );
	//The InputTexture contains premultiplied colors, so we need to unpremultiply first to apply our effect on straight colors.
	if( color.a > 0.0 )
		color.rgb /= color.a;

	// color.rgb += Brightness * 2. - 1.;

	//The plugin has to output premultiplied colors, this is how we're premultiplying our straight color while also
	//ensuring we aren't going out of the LDR the video engine is working in.
	color.rgb = clamp( color.rgb * color.a, vec3( 0.0 ), vec3( color.a ) );
	fragColor = color;
}
)";

FFGLPaletteEffect::FFGLPaletteEffect()
{
	std::string jsonpath;

	char* pValue;
	size_t len;
	errno_t err = _dupenv_s(&pValue, &len, "LOCALAPPDATA");

	if( err || pValue == NULL) {
		jsonpath = "c:\\windows\\temp\\ffgl.json";// last resort
		// NosuchDebug( "No value for LOCALAPPDATA? using jsonpath=%s\n", jsonpath.c_str() );
	}
	else {
		jsonpath = std::string( pValue ) + "\\Palette\\config\\ffgl.json";
		free( pValue );
	}
	// NosuchDebug( "Palette: config=%s", jsonpath.c_str() );

	paletteHost = new PaletteHost(jsonpath);

	// Input properties
	SetMinInputs( 1 );
	SetMaxInputs( 1 );

	//We declare that this plugin has a Brightness parameter which is a RGB param.
	//The name here must match the one you declared in your fragment shader.
	SetParamInfof( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT );
	AddRGBColorParam( "Brightness" );
	// AddParam(ffglqs::Param::Create("OSC Port", FF_TYPE_TEXT, "3334"));

	FFGLLog::LogToHost( "Created FFGLPaletteEffect effect" );
}
FFGLPaletteEffect::~FFGLPaletteEffect()
{
}

FFResult FFGLPaletteEffect::InitGL( const FFGLViewportStruct* vp )
{
	if( !shader.Compile( _vertexShaderCode, _fragmentShaderCode ) )
	{
		NosuchDebug("FFGLPaletteEffect: unable to compile shader\n");
		DeInitGL();
		return FF_FAIL;
	}
	if( !quad.Initialise() )
	{
		DeInitGL();
		return FF_FAIL;
	}

	paletteHost->InitGL(vp);

	//Use base-class init as success result so that it retains the viewport.
	return CFFGLPlugin::InitGL( vp );
}

bool PaletteFFThreadNameSet = false;

FFResult FFGLPaletteEffect::ProcessOpenGL( ProcessOpenGLStruct* pGL )
{

	if( !PaletteFFThreadNameSet )
	{
		NosuchDebugSetThreadName( pthread_self().p, "ProcessOpenGL" );
		PaletteFFThreadNameSet = true;
	}

	if( pGL->numInputTextures < 1 )
		return FF_FAIL;

	if( pGL->inputTextures[ 0 ] == NULL )
		return FF_FAIL;

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ScopedShaderBinding shaderBinding( shader.GetGLID() );
	//The shader's sampler is always bound to sampler index 0 so that's where we need to bind the texture.
	//Again, we're using the scoped bindings to help us keep the context in a default state.
	ScopedSamplerActivation activateSampler( 0 );
	Scoped2DTextureBinding textureBinding( pGL->inputTextures[ 0 ]->Handle );

	shader.Set( "inputTexture", 0 );

	//The input texture's dimension might change each frame and so might the content area.
	//We're adopting the texture's maxUV using a uniform because that way we dont have to update our vertex buffer each frame.
	FFGLTexCoords maxCoords = GetMaxGLTexCoords( *pGL->inputTextures[ 0 ] );
	shader.Set( "MaxUV", maxCoords.s, maxCoords.t );

	//This takes care of sending all the parameter that the plugin registered to the shader.
	ffglqs::Plugin::SendParams( shader );

	// quad.Draw();
	
	return paletteHost->PaletteHostProcessOpenGL(pGL);
	// return FF_SUCCESS;
}
FFResult FFGLPaletteEffect::DeInitGL()
{
	shader.FreeGLResources();
	quad.Release();

	paletteHost->DeInitGL();
	delete paletteHost;

	return FF_SUCCESS;
}

FFResult FFGLPaletteEffect::SetTextParameter(unsigned int index, const char* value)
{
	switch (index)
	{
	case PT_OSC_PORT:
		paletteHost->SetOscPort(std::string(value));
		return FF_SUCCESS;
	}
	NosuchDebug("SetTextParameter FAILS?");
	return FF_FAIL;
}

char* FFGLPaletteEffect::GetTextParameter(unsigned int index)
{
	static std::string value;
	switch (index)
	{
	case PT_OSC_PORT:
		value = paletteHost->GetOscPort();
		return (char*)(value.c_str());
	}
	NosuchDebug("GetTextParameter returns NULL?");
	return NULL;
}

