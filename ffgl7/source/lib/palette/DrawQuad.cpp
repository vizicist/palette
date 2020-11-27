#include "DrawQuad.h"
#include <assert.h>
#include "ffglex/FFGLScopedVAOBinding.h"
#include "ffglex/FFGLScopedBufferBinding.h"
#include "ffglex/FFGLUtilities.h"

DrawQuad::DrawQuad() :
	vaoID( 0 ),
	vboID( 0 )
{
}

DrawQuad::~DrawQuad()
{
	//If any of these assertions hit you forgot to release this quad's gl resources.
	assert( vaoID == 0 );
	assert( vaoID == 0 );
}

/**
 * Allow this utility to load the data it requires to do it's rendering into it's buffers.
 * This function needs to be called using an active OpenGL context, for example in your plugin's
 * InitGL function.
 *
 * @return: Whether or not initialising this quad succeeded.
 */
bool DrawQuad::Initialise( )
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

	glBufferData( GL_ARRAY_BUFFER, sizeof( vertices ), vertices, GL_DYNAMIC_DRAW );

	glEnableVertexAttribArray( 0 );
	glVertexAttribPointer( 0, 3, GL_FLOAT, false, sizeof( vertices[ 0 ] ), (char*)NULL + 2 * sizeof( float ) );
	glEnableVertexAttribArray( 1 );
	glVertexAttribPointer( 1, 2, GL_FLOAT, false, sizeof( vertices[ 0 ] ), (char*)NULL + 0 * sizeof( float ) );

	//The array enablements are part of the vao binding and not the global context state so we dont have to disable those here.

	return true;
}
/**
 * Draw the quad. Depending on your vertex shader this will apply your fragment shader in the area where the quad ends up.
 * You need to have successfully initialised this quad before rendering it.
 */
void DrawQuad::Draw(float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3)
{
	if( vaoID == 0 || vboID == 0 )
		return;

	//Scoped binding to make sure we dont keep the vao bind after we're done rendering.

	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	ffglex::ScopedVBOBinding vboBinding( vboID );

	vertices[0] = { 0.0f, 1.0f, x0, y0, 0.0f };//Top-left
	vertices[1] = { 1.0f, 1.0f, x1, y1, 0.0f };//Top-right
	vertices[2] = { 0.0f, 0.0f, x3, y3, 0.0f };//Bottom left

	vertices[3] = { 0.0f, 0.0f, x3, y3, 0.0f };//Bottom left
	vertices[4] = { 1.0f, 1.0f, x1, y1, 0.0f };//Top right
	vertices[5] = { 1.0f, 0.0f, x2, y2, 0.0f };//Bottom right

	glBufferData( GL_ARRAY_BUFFER, sizeof( vertices ), vertices, GL_DYNAMIC_DRAW );

	glDrawArrays( GL_TRIANGLES, 0, 6 );
}
/**
 * Release the gpu resources this quad has loaded into vram. Call this before destruction if you've previously initialised us.
 */
void DrawQuad::Release()
{
	glDeleteBuffers( 1, &vboID );
	vboID = 0;
	glDeleteVertexArrays( 1, &vaoID );
	vaoID = 0;
}
