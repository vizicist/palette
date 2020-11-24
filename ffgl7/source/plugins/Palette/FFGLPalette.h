#pragma once
#include <FFGLSDK.h>
#include "FFGLPluginSDK.h"
#include "PaletteHost.h"

class FFGLPalette : public PaletteHost, public CFFGLPlugin
{
public:
	FFGLPalette(std::string configfile);
	~FFGLPalette() { }

	// override methods in CFFGLPlugin
	FFResult InitGL( const FFGLViewportStruct* vp ) override;
	FFResult ProcessOpenGL( ProcessOpenGLStruct* pGL ) override;
	FFResult DeInitGL() override;
	FFResult SetFloatParameter( unsigned int dwIndex, float value ) override;
	float GetFloatParameter( unsigned int index ) override;
	FFResult SetTextParameter( unsigned int index, const char* value ) override;
	char* GetTextParameter( unsigned int index ) override;

	///////////////////////////////////////////////////
	// Factory method
	///////////////////////////////////////////////////

	static FFResult __stdcall CreateInstance( CFFGLPlugin** ppOutInstance )
	{
		// The ffgl.json file is under %LOCALAPPDATA%
		std::string jsonpath;

		char* pValue;
		size_t len;
		errno_t err = _dupenv_s( &pValue, &len, "LOCALAPPDATA" );
		if( err || pValue == NULL) {
			jsonpath = "c:\\windows\\temp\\ffgl.json";// last resort
			NosuchDebug( "No value for LOCALAPPDATA? using jsonpath=%s\n", jsonpath.c_str() );
		}
		else {
			jsonpath = std::string( pValue ) + "\\Palette\\config\\ffgl.json";
			free( pValue );
		}
		NosuchDebug( "Palette: config=%s", jsonpath.c_str() );
		*ppOutInstance = new FFGLPalette( jsonpath );
		if( *ppOutInstance != NULL )
			return FF_SUCCESS;
		return FF_FAIL;
	}

private:

};
