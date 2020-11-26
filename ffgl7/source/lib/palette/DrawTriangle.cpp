#include "DrawTriangle.h"
#include <assert.h>
#include "ffglex/FFGLScopedVAOBinding.h"
#include "ffglex/FFGLScopedBufferBinding.h"
#include "ffglex/FFGLUtilities.h"

#include "DrawUtil.h"
	
GlVertexTextured P_TEXTURED_TRIANGLE_VERTICES[] = {
	{ 0.0f, 1.0f, -0.5f, -0.5f, 0.0f }, //Bottom left
	{ 1.0f, 1.0f,  0.0f, 0.5f, 0.0f },  //Top
	{ 0.0f, 0.0f, 0.5f, -0.5f, 0.0f },//Bottom right
};

DrawTriangle::DrawTriangle() :
	vaoID( 0 ),
	vboID( 0 )
{
}
DrawTriangle::~DrawTriangle()
{
	//If any of these assertions hit you forgot to release this triangle's gl resources.
	assert( vaoID == 0 );
	assert( vaoID == 0 );
}

/**
 * Allow this utility to load the data it requires to do it's rendering into it's buffers.
 * This function needs to be called using an active OpenGL context, for example in your plugin's
 * InitGL function.
 *
 * @return: Whether or not initialising this triangle succeeded.
 */
bool DrawTriangle::Initialise( )
{
	glGenVertexArrays( 1, &vaoID );
	glGenBuffers( 1, &vboID );
	if( vaoID == 0 || vboID == 0 )
	{
		Release();
		return false;
	}

	//FFGL requires us to leave the context in a default state, so use these scoped bindings to
	//help us restore the state after we're done.
	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	ffglex::ScopedVBOBinding vboBinding( vboID );

	glBufferData( GL_ARRAY_BUFFER, sizeof( P_TEXTURED_TRIANGLE_VERTICES ), P_TEXTURED_TRIANGLE_VERTICES, GL_DYNAMIC_DRAW );

	glEnableVertexAttribArray( 0 );
	glVertexAttribPointer( 0, 3, GL_FLOAT, false, sizeof( P_TEXTURED_TRIANGLE_VERTICES[ 0 ] ), (char*)NULL + 2 * sizeof( float ) );

	// This is the UV
	glEnableVertexAttribArray( 1 );
	glVertexAttribPointer( 1, 2, GL_FLOAT, false, sizeof( P_TEXTURED_TRIANGLE_VERTICES[ 0 ] ), (char*)NULL + 0 * sizeof( float ) );

	//The array enablements are part of the vao binding and not the global context state so we dont have to disable those here.

	return true;
}
/**
 * Draw the triangle. Depending on your vertex shader this will apply your fragment shader in the area where the triangle ends up.
 * You need to have successfully initialised this triangle before rendering it.
 */
void DrawTriangle::Draw()
{
	if( vaoID == 0 || vboID == 0 )
		return;

	//Scoped binding to make sure we dont keep the vao bind after we're done rendering.
	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	glDrawArrays( GL_TRIANGLES, 0, 3 );
}
/**
 * Release the gpu resources this triangle has loaded into vram. Call this before destruction if you've previously initialised us.
 */
void DrawTriangle::Release()
{
	glDeleteBuffers( 1, &vboID );
	vboID = 0;
	glDeleteVertexArrays( 1, &vaoID );
	vaoID = 0;
}
