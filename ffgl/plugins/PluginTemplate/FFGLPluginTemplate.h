#ifndef FFGL{PLUGINNAME}_H
#define FFGL{PLUGINNAME}_H

#include "FFGLPluginSDK.h"

class FFGL{PLUGINNAME} : public CFreeFrameGLPlugin
{
public:
	FFGL{PLUGINNAME}();
    ~FFGL{PLUGINNAME}() {}

	///////////////////////////////////////////////////
	// FreeFrame plugin methods
	///////////////////////////////////////////////////
	
	FFResult	SetFloatParameter(unsigned int dwIndex, float value) override;
	float		GetFloatParameter(unsigned int index) override;
	FFResult	ProcessOpenGL(ProcessOpenGLStruct* pGL) override;
	FFResult	InitGL(const FFGLViewportStruct *vp) override;
	FFResult	DeInitGL() override;

	///////////////////////////////////////////////////
	// Factory method
	///////////////////////////////////////////////////

	static FFResult __stdcall CreateInstance(CFreeFrameGLPlugin **ppOutInstance)
    {
        *ppOutInstance = new FFGL{PLUGINNAME}();
        if (*ppOutInstance != NULL)
            return FF_SUCCESS;
        return FF_FAIL;
    }
    



protected:	

	float m_Hue1;
	float m_Hue2;
	float m_Saturation;
	float m_Brightness;
	
};


#endif
