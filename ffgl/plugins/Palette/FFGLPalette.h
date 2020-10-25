#ifndef FFGLPalette_H
#define FFGLPalette_H

#include "FFGLPluginSDK.h"
#include "PaletteHost.h"

class FFGLPalette : public PaletteHost, public CFreeFrameGLPlugin
{
public:
	FFGLPalette(std::string configfile);
	~FFGLPalette() {}

	///////////////////////////////////////////////////
	// FreeFrame plugin methods
	///////////////////////////////////////////////////

	FFResult	SetFloatParameter(unsigned int dwIndex, float value) override;
	float		GetFloatParameter(unsigned int index) override;
	FFResult	ProcessOpenGL(ProcessOpenGLStruct* pGL) override;
	FFResult	InitGL(const FFGLViewportStruct* vp) override;
	FFResult	DeInitGL() override;

	///////////////////////////////////////////////////
	// Factory method
	///////////////////////////////////////////////////

	static FFResult __stdcall CreateInstance(CFreeFrameGLPlugin** ppOutInstance)
	{
		// The ffgl.json file is under %LOCALAPPDATA%
		char* p = getenv("LOCALAPPDATA");
		NosuchDebug("CreateInstance LOCALAPPDATA = %s\n", p);
		std::string jsonpath;
		if ( p != NULL ) {
			jsonpath = std::string(p) + "\\Palette\\config\\ffgl.json";
		}
		else {
			jsonpath = "c:\\windows\\temp\\ffgl.json"; // last resort
		}
		NosuchDebug("Palette.CreateInstance: PortOffset=%d jsonpath=%s", PaletteHost::PortOffset,jsonpath.c_str());
		*ppOutInstance = new FFGLPalette(jsonpath);
        if (*ppOutInstance != NULL)
            return FF_SUCCESS;
        return FF_FAIL;
    }

	float m_zdefault;
	
};


#endif
