#include "PaletteAll.h"
#include "PaletteEffect.h"
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_OSC_PORT,
};

static CFFGLPluginInfo PluginInfo(
	PluginFactory< PaletteEffect >,// Create method
	"PLTE",                      // Plugin unique ID of maximum length 4.
	"PaletteEffect",            // Plugin name
	2,                           // API major version number
	1,                           // API minor version number
	1,                           // Plugin major version number
	0,                           // Plugin minor version number
	FF_EFFECT,                   // Plugin type
	"Palette Effect",  // Plugin description
	"by Tim Thompson"      // About
);

//////////////////////////////////////////////////////////////////
// Plugin dll entry point
//////////////////////////////////////////////////////////////////
BOOL APIENTRY DllMain( HANDLE hModule, DWORD ul_reason_for_call, LPVOID lpReserved )
{
	char dllpath[ MAX_PATH ];
	GetModuleFileNameA( (HMODULE)hModule, dllpath, MAX_PATH );

	if( ul_reason_for_call == DLL_PROCESS_ATTACH )
	{
		NosuchDebugSetThreadName( pthread_self().p, "PALETTEEFFECT_DLL" );
		NosuchDebug( 1, "DllMain: DLLPROCESS_ATTACH dll=%s", dllpath );
	}
	if( ul_reason_for_call == DLL_PROCESS_DETACH )
	{
		NosuchDebug( 1, "DllMain: DLLPROCESS_DETACH dll=%s", dllpath );
	}
	if( ul_reason_for_call == DLL_THREAD_ATTACH )
	{
		NosuchDebug( 1, "DllMain: DLLTHREAD_ATTACH dll=%s", dllpath );
	}
	if( ul_reason_for_call == DLL_THREAD_DETACH )
	{
		NosuchDebug( 1, "DllMain: DLLTHREAD_DETACH dll=%s", dllpath );
	}
	return TRUE;
}

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
uniform int tjt;

in vec2 uv;

out vec4 fragColor;

void main()
{
	vec4 color;
	vec3 bright;
	bright = vec3(0.5, 0.5, 0.5);

	if ( tjt > 0 ) {
		vec2 uv2 = uv / 2.0;
		color = texture( InputTexture, uv2 );
	} else {
		color = texture( InputTexture, uv );
	}
	//The InputTexture contains premultiplied colors, so we need to unpremultiply first to apply our effect on straight colors.
	if( color.a > 0.0 )
		color.rgb /= color.a;

	color.rgb += bright * 2. - 1.;

	//The plugin has to output premultiplied colors, this is how we're premultiplying our straight color while also
	//ensuring we aren't going out of the LDR the video engine is working in.
	color.rgb = clamp( color.rgb * color.a, vec3( 0.0 ), vec3( color.a ) );
	fragColor = color;
}
)";


// class PaletteEffect : public ffglqs::Plugin

PaletteEffect::PaletteEffect() : CFFGLPlugin()
{
	paletteHost = new PaletteHost();
	savedPixels = NULL;
	readCount   = 0;

	// Input properties
	SetMinInputs( 1 );
	SetMaxInputs( 1 );

	//We declare that this plugin has a Brightness parameter which is a RGB param.
	//The name here must match the one you declared in your fragment shader.
	// AddRGBColorParam( "Brightness" );

	// My Parameters
	// AddParam( Param::Create( "OSC Port" ) );
	SetParamInfo( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT, "5555" );
	// SetParamInfof( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT );

	FFGLLog::LogToHost( "Created PaletteEffect effect" );
}
PaletteEffect::~PaletteEffect()
{
}

FFResult PaletteEffect::InitGL( const FFGLViewportStruct* vp )
{
	paletteHost->InitGL( vp ); // NEW

	// x = vp->x;
	// y = vp->y;
	// width = vp->width;
	// height = vp->height;

	NosuchDebug( "PaletteEffect.InitGL: x,y=%d,%d w,h=%d,%d", vp->x, vp->y, vp->width, vp->height );
	if( !shader.Compile( _vertexShaderCode, _fragmentShaderCode ) )
	{
		DeInitGL();
		return FF_FAIL;
	}
	if( !quad.Initialise() )
	{
		DeInitGL();
		return FF_FAIL;
	}

	//Use base-class init as success result so that it retains the viewport.
	return CFFGLPlugin::InitGL( vp );
}

int pixelhash( char* pixels, int npixels )
{
	int h = 0;
	int n;
	for( n=0; n<npixels; n++ ) {
		h += pixels[ n ];
	}
	return h;
}

FFResult PaletteEffect::ProcessOpenGL( ProcessOpenGLStruct* pGL )
{
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
	shader.Set( "tjt", 0 );

	PaletteHost* ph = paletteHost;
	PaletteDrawer* pd = ph->palette()->Drawer();

	int width         = currentViewport.width;
	int height         = currentViewport.height;
	int npixels = width * height * 4;

	if (savedPixels != NULL) {

		// NosuchDebug( "ProcessOpenGL inpixels wh=%d,%d  hash=%d\n", width, height, inh1 );
		// NosuchDebug( "ProcessOpenGL readPixels wh=%d,%d  pix0,1=%d,%d hash=%d\n", width, height, pixels[0],pixels[1], h1);

		// glTexImage2D( GL_TEXTURE_2D, 0, GL_RGB, width, height, 0, GL_RGB, GL_UNSIGNED_BYTE, savedPixels );
		glTexImage2D( GL_TEXTURE_2D, 0, GL_RGB, width, height, 0, GL_RGB, GL_UNSIGNED_BYTE, savedPixels );
	}

	//The input texture's dimension might change each frame and so might the content area.
	//We're adopting the texture's maxUV using a uniform because that way we dont have to update our vertex buffer each frame.
	FFGLTexCoords maxCoords = GetMaxGLTexCoords( *pGL->inputTextures[ 0 ] );
	shader.Set( "MaxUV", maxCoords.s, maxCoords.t );

#ifdef TRYWITHOUT
	bool saveit = false;

	if( readCount++ > 100 ) {
		shader.Set( "tjt", 1 );
		readCount = 0;
		saveit  = true;
	}
#endif

	//This takes care of sending all the parameter that the plugin registered to the shader.
	// SendParams( shader );

	// quad.Draw();

#ifdef TRYWITHOUT
	if( saveit ) {
		if( savedPixels == NULL ) {
			savedPixels = (char*)malloc( npixels );
			memset( savedPixels, 0, npixels );
			NosuchDebug( "Saving Pixels!\n" );
			glReadPixels(0,0,width,height,GL_RGB,GL_UNSIGNED_BYTE, savedPixels);
			int h2       = pixelhash( savedPixels, npixels );
			NosuchDebug( "ProcessOpenGL readPixels wh=%d,%d hash=%d\n", width, height, h2 );
			}
	}
#endif

	return paletteHost->PaletteHostProcessOpenGL( pGL );

	// return FF_SUCCESS;
}

FFResult PaletteEffect::DeInitGL()
{
	paletteHost->DeInitGL();
	delete paletteHost;

	shader.FreeGLResources();
	quad.Release();

	return FF_SUCCESS;
}

FFResult PaletteEffect::SetTextParameter( unsigned int index, const char* value )
{
	switch( index )
	{
	case PT_OSC_PORT:
		paletteHost->SetOscPort( std::string( value ) );
		return FF_SUCCESS;
	}
	NosuchDebug( "SetTextParameter FAILS?" );
	return FF_FAIL;
}

char* PaletteEffect::GetTextParameter( unsigned int index )
{
	static std::string value;
	switch( index )
	{
	case PT_OSC_PORT:
		value = paletteHost->GetOscPort();
		return (char*)( value.c_str() );
	}
	NosuchDebug( "GetTextParameter returns NULL?" );
	return NULL;
}

// FFResult PaletteEffect::SetFloatParameter( unsigned int dwIndex, float value )
// {
// 	return FF_FAIL;
// }

// float PaletteEffect::GetFloatParameter( unsigned int index )
// {
// 	return 0.0f;
// }
