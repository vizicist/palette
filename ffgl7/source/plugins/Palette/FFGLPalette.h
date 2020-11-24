#pragma once
#include <FFGLSDK.h>
#include "FFGLPluginSDK.h"
#include "PaletteHost.h"

class FFGLPalette : public PaletteHost, public CFFGLPlugin
{
public:
	FFGLPalette(std::string configfile);
	~FFGLPalette() { }

	//CFFGLPlugin
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
		char* p = getenv( "LOCALAPPDATA" );
		std::string jsonpath;
		if( p != NULL )
		{
			jsonpath = std::string( p ) + "\\Palette\\config\\ffgl.json";
		}
		else
		{
			jsonpath = "c:\\windows\\temp\\ffgl.json";// last resort
		}
		NosuchDebug( "Palette: PortOffset=%d config=%s", PaletteHost::PortOffset, jsonpath.c_str() );
		*ppOutInstance = new FFGLPalette( jsonpath );
		if( *ppOutInstance != NULL )
			return FF_SUCCESS;
		return FF_FAIL;
	}

private:
	struct RGBA
	{
		float red   = 1.0f;
		float green = 1.0f;
		float blue  = 0.0f;
		float alpha = 1.0f;
	};
	struct HSBA
	{
		float hue   = 0.0f;
		float sat   = 1.0f;
		float bri   = 1.0f;
		float alpha = 1.0f;
	};
	RGBA rgba1;
	HSBA hsba2;

	ffglex::FFGLShader shader;  //!< Utility to help us compile and link some shaders into a program.
	ffglex::FFGLScreenQuad quad;//!< Utility to help us render a full screen quad.
	ffglex::FFGLScreenTriangle triangle;//!< Utility to help us render a full screen quad.
	GLint rgbLeftLocation;
	GLint rgbRightLocation;
};
