#ifndef _PALETTEFFHOST_H
#define _PALETTEFFHOST_H

#include "FFGL.h"
#include "FFGLPluginInfo.h"
#include "PaletteHost.h"

class PaletteFFHost : public PaletteHost, public CFreeFrameGLPlugin
{

public:

	static DWORD __stdcall CreateInstance(CFreeFrameGLPlugin **ppInstance) {
		NosuchDebug(1,"PaletteFFHost CreatInstance is creating!\n");

		StaticInitialization();
		*ppInstance = new PaletteFFHost(NosuchFullPath("../config/palette.json"));
		if (*ppInstance != NULL)
			return FF_SUCCESS;
		return FF_FAIL;
	}

	PaletteFFHost(std::string defaultsfile);
	~PaletteFFHost();
	DWORD GetParameter(DWORD dwIndex);
	DWORD SetParameter(const SetParameterStruct* pParam);
	DWORD ProcessOpenGL(ProcessOpenGLStruct *pGL);

};

#endif
