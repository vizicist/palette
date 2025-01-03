#include <iostream>
#include <sstream>
#include <fstream>
#include <strstream>
#include <cstdlib>// for srand, rand
#include <ctime>  // for time
#include <sys/stat.h>

// to get M_PI
#define _USE_MATH_DEFINES
#include <math.h>

#include "PaletteAll.h"
#include <glm/gtc/type_ptr.hpp>

using namespace ffglex;

static const char vertexShaderPalette[] = R"(#version 410 core
layout( location = 0 ) in vec4 vPosition;
layout( location = 1 ) in vec2 vUV;

// uniform vec2 vTranslate;
uniform mat4 vMatrix;

out vec2 uv;

void main()
{
	gl_Position = vMatrix * vPosition;
	uv = vUV;
}
)";

static const char fragmentShaderPalette[] = R"(#version 410 core
uniform vec4 RGBALeft;
uniform vec4 RGBARight;
uniform sampler2D InputTexture;
uniform int tjt;
uniform int style;

in vec2 uv;

out vec4 fragColor;

void main()
{
	int usetexture;
	usetexture = 1;
	if ( style > 0 ) {
		fragColor = texture( InputTexture, uv );
	} else {
		fragColor = mix( RGBALeft, RGBARight, uv.x );
	}
}
)";

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Plugin information
////////////////////////////////////////////////////////////////////////////////////////////////////

PaletteDrawer::PaletteDrawer( PaletteParams* params ) :
	m_matrix( 1.0f ),
	m_matrix_identity( 1.0f ),
	vaoID( 0 ),
	vboID( 0 )
{
	srand( (unsigned)time( NULL ) );

	m_params = params;

	m_isdrawing = false;

	m_rgbLeftLocation  = -1;
	m_rgbRightLocation = -1;
	m_matrixLocation   = -1;

	resetMatrix();
}

void PaletteDrawer::initBuffers()
{
	glGenVertexArrays( 1, &vaoID );
	glGenBuffers( 1, &vboID );
	if( vaoID == 0 || vboID == 0 )
	{
		NosuchDebug( "PaletteDrawer.initBuffers: unable to create!?" );
		releaseBuffers();
		return;
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
}

void PaletteDrawer::resetMatrix()
{
	m_matrix = m_matrix_identity;
}

PaletteDrawer::~PaletteDrawer()
{
	NosuchDebug( 1, "PaletteDrawer destructor called" );
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Methods
////////////////////////////////////////////////////////////////////////////////////////////////////

void PaletteDrawer::background( int b )
{
	NosuchDebug( "PaletteDrawer::background!" );
}
void PaletteDrawer::strokeWeight( float w )
{
	glLineWidth( (GLfloat)w );
}
void PaletteDrawer::rotate( float radians )
{
	m_matrix = glm::rotate( m_matrix, radians, glm::vec3( 0.0f, 0.0f, 1.0f ) );
}
void PaletteDrawer::translate( float x, float y )
{
	m_matrix = glm::translate( m_matrix, glm::vec3( x, y, 0.0f ) );
}
void PaletteDrawer::scale( float x, float y )
{
	m_matrix = glm::scale( m_matrix, glm::vec3( x, y, 1.0f ) );
}

float PaletteDrawer::scale_z( float z )
{
	// We want the z value to be scaled exponentially toward 1.0,
	// i.e. raw z of .5 should result in a scale_z value of .75
	double expz = 1.0f - pow( ( 1.0 - z ), m_params->zexponential );
	return float( expz * m_params->zmultiply );
}

ffglex::FFGLShader* PaletteDrawer::BeginDrawingWithShader( std::string shaderName )
{
	ffglex::FFGLShader* shader;
	if( m_isdrawing )
	{
		NosuchDebug( "Warning, BeginDrawingWithShader called when isDrawing" );
		return NULL;
	}
	if( shaderName == "gradient" )
	{
		shader = &m_shader_gradient;
	}
	else
	{
		shader = &m_shader_gradient;
	}
	glUseProgram( shader->GetGLID() );
	m_isdrawing = true;
	return shader;
}

void PaletteDrawer::EndDrawing()
{
	if( !m_isdrawing )
	{
		NosuchDebug( "Warning, EndDrawing called when !isDrawing" );
		return;
	}
	glUseProgram( 0 );
	m_isdrawing = false;
}

bool PaletteDrawer::prepareToDraw( SpriteParams& params, SpriteState& state )
{
	if( vaoID == 0 || vboID == 0 )
	{
		NosuchDebug( "prepareToDraw: vaoID or vboID not set?" );
		return false;
	}

	NosuchColor c1( state.hue1, params.luminance, params.saturation );
	NosuchColor c2( state.hue2, params.luminance, params.saturation );

	glUniform4f( m_rgbLeftLocation, c1.R(), c1.G(), c1.B(), state.alpha );
	glUniform4f( m_rgbRightLocation, c2.R(), c2.G(), c2.B(), state.alpha );
	glUniformMatrix4fv( m_matrixLocation, 1, GL_FALSE, glm::value_ptr( m_matrix ) );
	int style = 0;
	if( params.spritestyle == "texture" )
	{
		style = 1;
	}
	glUniform1i( m_styleLocation, style );

	return true;
}

float PaletteDrawer::finalAspect( float aspect )
{
	// 0.0 aspect is finalaspect 0.1
	// 0.5 aspect is finalaspect 0.5
	// 1.0 aspect is finalaspect 10.0
	float finalaspect = 1.0;
	if( aspect <= 0.5f )
	{
		finalaspect = 0.05f + aspect * 0.9f;
	}
	else
	{
		finalaspect = 0.5f + ( aspect - 0.5f ) * 19.0f;
	}
	return finalaspect;
}

void PaletteDrawer::drawQuad( SpriteParams& params, SpriteState& state, float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3 )
{
	if( !prepareToDraw( params, state ) )
	{
		return;
	}

	//Scoped binding to make sure we dont keep the vao bind after we're done rendering.
	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	ffglex::ScopedVBOBinding vboBinding( vboID );

	float screenAspect = float( viewportHeight() ) / float( viewportWidth() );
	m_matrix           = glm::scale( m_matrix, glm::vec3( screenAspect, 1.0f, 1.0f ) );

	float finalaspect = finalAspect( params.aspect );

	if( finalaspect != 1.0f )
	{
		x0 *= finalaspect;
		x1 *= finalaspect;
		x2 *= finalaspect;
		x3 *= finalaspect;
	}

	if( params.filled )
	{
		int nvertices = 6;

		vertices[ 0 ] = { 0.0f, 1.0f, x0, y0, 0.0f };//Top-left
		vertices[ 1 ] = { 1.0f, 1.0f, x1, y1, 0.0f };//Top-right
		vertices[ 2 ] = { 0.0f, 0.0f, x3, y3, 0.0f };//Bottom left

		vertices[ 3 ] = { 0.0f, 0.0f, x3, y3, 0.0f };//Bottom left
		vertices[ 4 ] = { 1.0f, 1.0f, x1, y1, 0.0f };//Top right
		vertices[ 5 ] = { 1.0f, 0.0f, x2, y2, 0.0f };//Bottom right

		glBufferSubData( GL_ARRAY_BUFFER, 0, nvertices * sizeof( vertices[ 0 ] ), vertices );
		glDrawArrays( GL_TRIANGLES, 0, nvertices );
	}
	else
	{
		int nvertices = 4;
		vertices[ 0 ] = { 0.0f, 1.0f, x0, y0, 0.0f };//Top-left
		vertices[ 1 ] = { 1.0f, 1.0f, x1, y1, 0.0f };//Top-right
		vertices[ 2 ] = { 0.0f, 0.0f, x2, y2, 0.0f };//Bottom-right
		vertices[ 3 ] = { 0.0f, 0.0f, x3, y3, 0.0f };//Bottom-left

		glBufferSubData( GL_ARRAY_BUFFER, 0, nvertices * sizeof( vertices[ 0 ] ), vertices );
		glDrawArrays( GL_LINE_LOOP, 0, nvertices );
	}
}

void PaletteDrawer::drawTriangle( SpriteParams& params, SpriteState& state, float x0, float y0, float x1, float y1, float x2, float y2 )
{
	if( !prepareToDraw( params, state ) )
	{
		return;
	}

	//Scoped binding to make sure we dont keep the vao bind after we're done rendering.
	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	ffglex::ScopedVBOBinding vboBinding( vboID );

	float screenAspect = float( viewportHeight() ) / float( viewportWidth() );
	m_matrix           = glm::scale( m_matrix, glm::vec3( screenAspect, 1.0f, 1.0f ) );

	float finalaspect = finalAspect( params.aspect );
	if( finalaspect != 1.0f )
	{
		x0 *= finalaspect;
		x1 *= finalaspect;
		x2 *= finalaspect;
	}

	int nvertices = 3;
	vertices[ 0 ] = { 0.0f, 1.0f, x0, y0, 0.0f };
	vertices[ 1 ] = { 1.0f, 1.0f, x1, y1, 0.0f };
	vertices[ 2 ] = { 0.0f, 0.0f, x2, y2, 0.0f };

	glBufferSubData( GL_ARRAY_BUFFER, 0, nvertices * sizeof( vertices[ 0 ] ), vertices );
	glDrawArrays( params.filled ? GL_TRIANGLES : GL_LINE_LOOP, 0, nvertices );
}

void PaletteDrawer::drawLine( SpriteParams& params, SpriteState& state, float x0, float y0, float x1, float y1 )
{
	if( !prepareToDraw( params, state ) )
	{
		return;
	}

	//Scoped binding to make sure we dont keep the vao bind after we're done rendering.
	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	ffglex::ScopedVBOBinding vboBinding( vboID );

	int nvertices = 2;
	vertices[ 0 ] = { 0.0f, 1.0f, x0, y0, 0.0f };
	vertices[ 1 ] = { 1.0f, 1.0f, x1, y1, 0.0f };

	glBufferSubData( GL_ARRAY_BUFFER, 0, nvertices * sizeof( vertices[ 0 ] ), vertices );
	glDrawArrays( GL_LINES, 0, 2 );
}

static float degree2radian( float deg )
{
	return 2.0f * (float)M_PI * deg / 360.0f;
}

void PaletteDrawer::drawEllipse( SpriteParams& params, SpriteState& state, float x0, float y0, float radius, float fromang, float toang )
{
	if( !prepareToDraw( params, state ) )
	{
		return;
	}
	//Scoped binding to make sure we dont keep the vao bind after we're done rendering.
	ffglex::ScopedVAOBinding vaoBinding( vaoID );
	ffglex::ScopedVBOBinding vboBinding( vboID );

	float screenAspect = float( viewportHeight() ) / float( viewportWidth() );
	m_matrix           = glm::scale( m_matrix, glm::vec3( screenAspect, 1.0f, 1.0f ) );

	float finalaspect = finalAspect( params.aspect );

	int nvertices = MAX_VERTICES;

	for( int n = 0; n < nvertices; n++ )
	{
		float delta  = float( n ) / ( nvertices - 1 );
		float degree = fromang + delta * toang;
		float x      = x0 + sin( degree2radian( degree ) ) * radius;
		float y      = y0 + cos( degree2radian( degree ) ) * radius;
		if( finalaspect != 1.0f )
		{
			x *= finalaspect;
		}
		vertices[ n ] = { 0.0f, 1.0f, x, y, 0.0f };
	}

	if( params.filled )
	{
		glBufferSubData( GL_ARRAY_BUFFER, 0, nvertices * sizeof( vertices[ 0 ] ), vertices );
		glDrawArrays( GL_TRIANGLE_FAN, 0, nvertices );
	}
	else
	{
		glBufferSubData( GL_ARRAY_BUFFER, 0, nvertices * sizeof( vertices[ 0 ] ), vertices );
		glDrawArrays( GL_LINE_LOOP, 0, nvertices );
	}
}

void PaletteDrawer::drawPolygon( PointMem* points, int npoints )
{
	NosuchDebug( 2, "Drawing polygon" );
#ifdef OLD_GRAPHICS
	if( m_filled )
	{
		glBegin( GL_TRIANGLE_FAN );
		glVertex2d( 0.0, 0.0 );
		for( int pn = 0; pn < npoints; pn++ )
		{
			PointMem* p = &points[ pn ];
			glVertex2d( p->x, p->y );
		}
		glEnd();
	}
	if( m_stroked )
	{
		glBegin( GL_LINE_LOOP );
		for( int pn = 0; pn < npoints; pn++ )
		{
			PointMem* p = &points[ pn ];
			glVertex2d( p->x, p->y );
		}
		glEnd();
	}

	if( !m_filled && !m_stroked )
	{
		NosuchDebug( "Hey, ellipse() called when both m_filled and m_stroked are off!?" );
	}
#endif
}

FFResult PaletteDrawer::InitGL( const FFGLViewportStruct* vp )
{
	m_vp = *vp;

	// NosuchDebug( "PaletteDrawer::InitGL: m_vp = w,h=%d,%d  xy=%d,%d\n", m_vp.height, m_vp.width, m_vp.x, m_vp.y );

	if( !m_shader_gradient.Compile( vertexShaderPalette, fragmentShaderPalette ) )
	{
		NosuchDebug( "Error in compiling m_shader_gradient!" );
		DeInitGL();
		return FF_FAIL;
	}
	initBuffers();

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ffglex::ScopedShaderBinding shaderBinding( m_shader_gradient.GetGLID() );
	m_rgbLeftLocation  = m_shader_gradient.FindUniform( "RGBALeft" );
	m_rgbRightLocation = m_shader_gradient.FindUniform( "RGBARight" );
	m_matrixLocation   = m_shader_gradient.FindUniform( "vMatrix" );
	m_styleLocation    = m_shader_gradient.FindUniform( "style" );

	return FF_SUCCESS;
}

FFResult PaletteDrawer::DeInitGL()
{
	m_shader_gradient.FreeGLResources();

	releaseBuffers();

	m_rgbLeftLocation  = -1;
	m_rgbRightLocation = -1;

	return FF_SUCCESS;
}

void PaletteDrawer::releaseBuffers()
{
	glDeleteBuffers( 1, &vboID );
	vboID = 0;
	glDeleteVertexArrays( 1, &vaoID );
	vaoID = 0;
}
