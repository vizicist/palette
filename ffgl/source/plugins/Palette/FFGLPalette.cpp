#include "PaletteAll.h"
#include "FFGLPalette.h"

#include <math.h>//floor
using namespace ffglex;

enum ParamType : FFUInt32
{
	PT_OSC_PORT,
};

static CFFGLPluginInfo PluginInfo(
	// PluginFactory< FFGLPalette >,// Create method
	FFGLPalette::CreateInstance,         // Create method
	"PLTA",                        // Plugin unique ID
	"Palette",                     // Plugin name
	2,                             // API major version number
	1,                             // API minor version number
	1,                             // Plugin major version number
	000,                           // Plugin minor version number
	FF_SOURCE,                     // Plugin type
	"Palette instrument plugin v0.90 #3",// Plugin description
	"by Tim Thompson"        // About
);

extern "C" {
extern bool ffgl_setdll( std::string dllpath );
}

//////////////////////////////////////////////////////////////////
// Plugin dll entry point
//////////////////////////////////////////////////////////////////
BOOL APIENTRY DllMain( HANDLE hModule, DWORD ul_reason_for_call, LPVOID lpReserved )
{
	char dllpath[ MAX_PATH ];
	GetModuleFileNameA( (HMODULE)hModule, dllpath, MAX_PATH );

	if( ul_reason_for_call == DLL_PROCESS_ATTACH )
	{
		if( !ffgl_setdll( std::string( dllpath ) ) )
		{
			NosuchDebug( "DllMain: ffgl_setdll failed" );
			return FALSE;
		}

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

std::string ffgl_name()
{
	return "Palette";
}

CFFGLPluginInfo& ffgl_plugininfo()
{
	return PluginInfo;
}


FFGLPalette::FFGLPalette(std::string configfile) :
	CFFGLPlugin()
{
	paletteHost = new PaletteHost( configfile );

	// Input properties
	SetMinInputs( 0 );
	SetMaxInputs( 0 );

	// Parameters
	SetParamInfof( PT_OSC_PORT, "OSC Port", FF_TYPE_TEXT );

	FFGLLog::LogToHost( "Created Palette" );
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
	if( !PaletteFFThreadNameSet )
	{
		NosuchDebugSetThreadName( pthread_self().p, "ProcessOpenGL" );
		PaletteFFThreadNameSet = true;
	}
	return paletteHost->PaletteHostProcessOpenGL( pGL );
}
FFResult FFGLPalette::DeInitGL()
{
	paletteHost->DeInitGL( );
	delete paletteHost;
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
