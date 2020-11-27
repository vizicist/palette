#pragma once
// #include "../ffgl/FFGL.h"//For OpenGL
// #include "../ffglex/FFGLShader.h"
// #include <string>
#include "ffglex/FFGLScopedVAOBinding.h"
#include "ffglex/FFGLScopedBufferBinding.h"
#include "ffglex/FFGLUtilities.h"
#include "ffglex/FFGLShader.h"

class DrawQuad
{
public:
	DrawQuad();
	DrawQuad( const DrawQuad& ) = delete;
	DrawQuad( DrawQuad&& )      = delete;
	~DrawQuad();

	bool Initialise( );//Allow this utility to load the data it requires to do it's rendering into it's buffers.
	void Draw(float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3);                          
	void Release(); //Release the gpu resources this quad has loaded into vram. Call this before destruction if you've previously initialised us.

private:
	ffglex::FFGLShader shader;
	ffglex::GlVertexTextured vertices[ 6 ];

	GLuint vaoID;
	GLuint vboID;
};
