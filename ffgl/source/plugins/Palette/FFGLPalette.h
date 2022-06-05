#pragma once

// class FFGLPalette : public PaletteHost, public CFFGLPlugin

class FFGLPalette : public CFFGLPlugin
{
public:
	FFGLPalette();
	~FFGLPalette() { }

	//CFFGLPlugin
	FFResult InitGL(const FFGLViewportStruct* vp) override;
	FFResult ProcessOpenGL(ProcessOpenGLStruct* pGL) override;
	FFResult DeInitGL() override;
	FFResult SetFloatParameter(unsigned int dwIndex, float value) override;
	float GetFloatParameter(unsigned int index) override;
	FFResult SetTextParameter(unsigned int index, const char* value) override;
	char* GetTextParameter(unsigned int index) override;

protected:
	PaletteHost* paletteHost;

};
