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

PaletteEffect::PaletteEffect() : CFFGLPlugin()
{
	paletteHost = new PaletteHost();
	// savedPixels = NULL;
	readCount   = 0;

	// Input properties
	SetMinInputs( 1 );
	SetMaxInputs( 1 );

	SetParamInfo( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT, "5555" );
}

PaletteEffect::~PaletteEffect()
{
}

FFResult PaletteEffect::InitGL( const FFGLViewportStruct* vp )
{
	paletteHost->InitGL( vp );

	//Use base-class init as success result so that it retains the viewport.
	return CFFGLPlugin::InitGL( vp );
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

	// shader.Set( "inputTexture", 0 );
	// shader.Set( "tjt", 0 );

#ifdef TRYWITHOUT
	int width         = currentViewport.width;
	int height         = currentViewport.height;
	int npixels = width * height * 4;

	if (savedPixels != NULL) {
		glTexImage2D( GL_TEXTURE_2D, 0, GL_RGB, width, height, 0, GL_RGB, GL_UNSIGNED_BYTE, savedPixels );
	}
#endif

	//The input texture's dimension might change each frame and so might the content area.
	//We're adopting the texture's maxUV using a uniform because that way we dont have to update our vertex buffer each frame.
#ifdef TRYWITHOUT
	FFGLTexCoords maxCoords = GetMaxGLTexCoords( *pGL->inputTextures[ 0 ] );
	shader.Set( "MaxUV", maxCoords.s, maxCoords.t );
#endif

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

	quad.Draw();

#ifdef TRYWITHOUT
	if( saveit ) {
		if( savedPixels == NULL ) {
			savedPixels = (char*)malloc( npixels );
			memset( savedPixels, 0, npixels );
			glReadPixels(0,0,width,height,GL_RGB,GL_UNSIGNED_BYTE, savedPixels);
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
