#pragma once
#include "../ffgl/FFGL.h"//For OpenGL
#include "../ffglex/FFGLShader.h"
#include <string>

class DrawTriangle
{
public:
	DrawTriangle();
	DrawTriangle( const DrawTriangle& ) = delete;
	DrawTriangle( DrawTriangle&& )      = delete;
	~DrawTriangle();

	bool Initialise( );		//Allow this utility to load the data it requires to do it's rendering into it's buffers.
	void Draw();            //Draw the quad. Depending on your vertex shader this will apply your fragment shader in the area where the quad ends up.
	void Release();         //Release the gpu resources this quad has loaded into vram. Call this before destruction if you've previously initialised us.

private:
	ffglex::FFGLShader shader;

	GLuint vaoID;
	GLuint vboID;
};
