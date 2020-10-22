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
		// I've tried unsucessfully to retrieve the PALETTE environment variable with getenv(),
		// but the FFGL plugin seems to crash at that point, perhaps it's missing a needed .dll.
		// So, the location of the ffgl.json file is hardcoded to ../config/ffgl.json.
		// I.e. the config and ffgl directories should have the same parent.
		const char* palette = "..";
		// char *e = _getenv("PALETTE");
		// if ( e != NULL ) {
		// 	palette = e;
		// }
		std::string rpath = NosuchSnprintf("%s\\config\\ffgl.json",palette);
		std::string jsonpath = NosuchFullPath(rpath);
		NosuchDebug("Palette.CreateInstance: PortOffset=%d jsonpath=%s", PaletteHost::PortOffset,jsonpath.c_str());
		*ppOutInstance = new FFGLPalette(jsonpath);
        if (*ppOutInstance != NULL)
            return FF_SUCCESS;
        return FF_FAIL;
    }

	float m_zdefault;
	
};


#endif
