#include "PaletteAll.h"
#include "FFGLPalette.h"

#include <math.h>//floor
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_OSC_PORT,
};

static CFFGLPluginInfo PluginInfo(
	PluginFactory< FFGLPalette >,// Create method
	"PLTX",                        // Plugin unique ID
	"Palette",                     // Plugin name
	2,                             // API major version number
	1,                             // API minor version number
	1,                             // Plugin major version number
	000,                           // Plugin minor version number
	FF_EFFECT,                     // Plugin type
	"Palette X Instrument",// Plugin description
	"by Tim Thompson"        // About
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
		NosuchDebugSetThreadName( pthread_self().p, "PALETTE_DLL" );
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

FFGLPalette::FFGLPalette() : CFFGLPlugin()
{
	paletteHost = new PaletteHost();

	// Input properties
	SetMinInputs( 1 );
	SetMaxInputs( 1 );

	// Parameters
	SetParamInfof( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT );
}

FFResult FFGLPalette::InitGL( const FFGLViewportStruct* vp )
{
	NosuchDebug( "Palette.InitGL: x,y=%d,%d w,h=%d,%d", vp->x, vp->y, vp->width, vp->height );
	paletteHost->InitGL( vp );

	//Use base-class init as success result so that it retains the viewport.
	return CFFGLPlugin::InitGL( vp );
}

bool PaletteFFThreadNameSet = false;

FFResult FFGLPalette::ProcessOpenGL( ProcessOpenGLStruct* pGL )
{
	if( pGL->numInputTextures < 1 )
		return FF_FAIL;

	if( pGL->inputTextures[ 0 ] == NULL )
		return FF_FAIL;

	if( !PaletteFFThreadNameSet )
	{
		NosuchDebugSetThreadName( pthread_self().p, "ProcessOpenGL" );
		PaletteFFThreadNameSet = true;
	}

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ScopedShaderBinding shaderBinding( shader.GetGLID() );
	//The shader's sampler is always bound to sampler index 0 so that's where we need to bind the texture.
	//Again, we're using the scoped bindings to help us keep the context in a default state.
	ScopedSamplerActivation activateSampler( 0 );
	Scoped2DTextureBinding textureBinding( pGL->inputTextures[ 0 ]->Handle );

	// quad.Draw();

	return paletteHost->PaletteHostProcessOpenGL( pGL );
}
FFResult FFGLPalette::DeInitGL()
{
	paletteHost->DeInitGL( );
	delete paletteHost;
	// quad.Release();
	return FF_SUCCESS;
}

FFResult FFGLPalette::SetTextParameter( unsigned int index, const char* value )
{
	switch( index )
	{
	case PT_OSC_PORT:
		paletteHost->SetOscPort(std::string( value ));
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
		value = paletteHost->GetOscPort();
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
