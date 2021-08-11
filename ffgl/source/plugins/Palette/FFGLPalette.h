#pragma once

// class FFGLPalette : public PaletteHost, public CFFGLPlugin

class FFGLPalette : public CFFGLPlugin
{
public:
	FFGLPalette(std::string configfile);
	~FFGLPalette() { }

	// override methods in CFFGLPlugin
	FFResult InitGL(const FFGLViewportStruct* vp) override;
	FFResult ProcessOpenGL(ProcessOpenGLStruct* pGL) override;
	FFResult DeInitGL() override;
	FFResult SetFloatParameter(unsigned int dwIndex, float value) override;
	float GetFloatParameter(unsigned int index) override;
	FFResult SetTextParameter(unsigned int index, const char* value) override;
	char* GetTextParameter(unsigned int index) override;

	PaletteHost* paletteHost;

	///////////////////////////////////////////////////
	// Factory method
	///////////////////////////////////////////////////

	static FFResult __stdcall CreateInstance(CFFGLPlugin** ppOutInstance)
	{
		// The search for ffgl.json is as follows:
		// - look in %LOCALAPPDATA%
		// - last resort is temp dir

		std::string jsonpath;

		char* localValue;
		size_t locallen;
		errno_t localerr = _dupenv_s(&localValue, &locallen, "LOCALAPPDATA");
		if (!localerr && localValue != NULL) {
			jsonpath = std::string( localValue ) + "\\Palette\\config\\ffgl.json";
			free( localValue );
		} else {
			jsonpath = "c:\\windows\\temp\\ffgl.json";// last resort
			NosuchDebug("No value for LOCALAPPDATA? using jsonpath=%s\n", jsonpath.c_str());
		}

		*ppOutInstance = new FFGLPalette( jsonpath );
		if( *ppOutInstance != NULL )
			return FF_SUCCESS;
		return FF_FAIL;
	}

private:

};
