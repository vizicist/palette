#include <iostream>
#include <sstream>
#include <fstream>
#include <strstream>
#include <cstdlib> // for srand, rand
#include <ctime>   // for time
#include <sys/stat.h>

// to get M_PI
#define _USE_MATH_DEFINES
#include <math.h>

#include "PaletteAll.h"
#include <glm/gtc/type_ptr.hpp>

using namespace ffglex;

static const char vertexShaderGradient[] = R"(#version 410 core
layout( location = 0 ) in vec4 vPosition;
layout( location = 1 ) in vec2 vUV;

uniform vec2 vScale;
uniform vec2 vTranslate;
uniform mat4 vMatrix;

out vec2 uv;

void main()
{
	gl_Position = vMatrix * vec4((vPosition.x*vScale.x)+vTranslate.x,(vPosition.y*vScale.y)+vTranslate.y,vPosition.z,vPosition.a);
	uv = vUV;
}
)";

static const char fragmentShaderGradient[] = R"(#version 410 core
uniform vec4 RGBALeft;
uniform vec4 RGBARight;

in vec2 uv;

out vec4 fragColor;

void main()
{
	fragColor = mix( RGBALeft, RGBARight, uv.x );
}
)";

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Plugin information
////////////////////////////////////////////////////////////////////////////////////////////////////

PaletteDrawer::PaletteDrawer(PaletteParams *params) :
	m_matrix(1.0f),
	m_matrix_identity(1.0f)
{
	srand((unsigned)time(NULL));

	NosuchDebug("PaletteDrawer constructor!");

	m_params = params;

	m_isdrawing = false;
	m_filled = false;
	m_stroked = false;
	m_fill_alpha = 1.0;
	m_stroke_alpha = 1.0;
	m_stroke_color = NosuchColor( 255, 255, 0 );

	m_rgbLeftLocation = -1;
	m_rgbRightLocation = -1;
	m_matrixLocation   = -1;

	m_width              = 1.0;
	m_height              = 1.0;

	m_rgba1 = { 1.0f, 1.0f, 0.0f, 1.0f };
	m_hsba2 = { 0.0f, 1.0f, 1.0f, 1.0f };

	resetMatrix();
}

void
PaletteDrawer::resetMatrix()
{
#if 0
	GLfloat matrix[16] = {
		1.0, 0.0, 0.0, 0.0,
		0.0, 1.0, 0.0, 0.0,
		0.0, 0.0, 1.0, 0.0,
		0.0, 0.0, 0.0, 1.0
	};
	setMatrix( matrix );
#endif
	m_matrix = m_matrix_identity;
}

#if 0
void
PaletteDrawer::setMatrix(GLfloat matrix[16]) {
	// XXX - there's got to be a better way of doing this
	for( int i = 0; i < 16; i++ )
	{
		m_matrix[ i ] = matrix[ i ];
	}
}
#endif

PaletteDrawer::~PaletteDrawer()
{
	NosuchDebug(1,"PaletteDrawer destructor called");
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Methods
////////////////////////////////////////////////////////////////////////////////////////////////////

void PaletteDrawer::fill(NosuchColor c, float alpha) {
	m_filled = true;
	m_fill_color = c;
	m_fill_alpha = alpha;
}
void PaletteDrawer::stroke(NosuchColor c, float alpha) {
	// glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, alpha);
	m_stroked = true;
	m_stroke_color = c;
	m_stroke_alpha = alpha;
}
void PaletteDrawer::noFill() {
	m_filled = false;
}
void PaletteDrawer::background(int b) {
	NosuchDebug("PaletteDrawer::background!");
}
void PaletteDrawer::strokeWeight(float w) {
	glLineWidth((GLfloat)w);
}
void PaletteDrawer::rotate(float radians) {
	m_matrix = glm::rotate( m_matrix, radians, glm::vec3(0.0f,0.0f,1.0f));
}
void PaletteDrawer::translate(float x, float y) {
	m_matrix = glm::translate( m_matrix, glm::vec3(x,y,0.0f));
}
void PaletteDrawer::scale(float x, float y) {
}

float PaletteDrawer::scale_z(float z) {
	// We want the z value to be scaled exponentially toward 1.0,
	// i.e. raw z of .5 should result in a scale_z value of .75
	double expz = 1.0f - pow((1.0-z),m_params->zexponential);
	return float(expz * m_params->zmultiply);
}

ffglex::FFGLShader* PaletteDrawer::BeginDrawingWithShader(std::string shaderName)
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
	glUseProgram(shader->GetGLID());
	m_isdrawing = true;
	return shader;
}

void PaletteDrawer::EndDrawing()
{
	if( ! m_isdrawing )
	{
		NosuchDebug( "Warning, EndDrawing called when !isDrawing" );
		return;
	}
	glUseProgram( 0 );
	m_isdrawing = false;
}

void PaletteDrawer::drawQuad(float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3) {
	NosuchDebug(1,"PaletteDrawer.drawQuad: %.3f %.3f, %.3f %.3f, %.3f %.3f, %.3f %.3f",x0,y0,x1,y1,x2,y2,x3,y3);

	NosuchColor c1 = m_fill_color;
	RGBA rgba1{
		c1.r() / 255.0f,
		c1.g() / 255.0f,
		c1.b() / 255.0f,
		(float) m_fill_alpha };

	NosuchColor c2 = m_stroke_color;
	RGBA rgba2{
		c2.r() / 255.0f,
		c2.g() / 255.0f,
		c2.b() / 255.0f,
		(float) m_stroke_alpha };

	glUniform4f( m_rgbLeftLocation, rgba1.red, rgba1.green, rgba1.blue, rgba1.alpha );
	glUniform4f( m_rgbRightLocation, rgba2.red, rgba2.green, rgba2.blue, rgba2.alpha );
	const GLfloat* p = glm::value_ptr( m_matrix );
	glUniformMatrix4fv( m_matrixLocation, 1, GL_FALSE, p );

	m_quad.Draw(x0,y0,x1,y1,x2,y2,x3,y3);
}
void PaletteDrawer::drawTriangle(float x0, float y0, float x1, float y1, float x2, float y2) {
	NosuchDebug("Drawing triangle xy0=%.3f,%.3f xy1=%.3f,%.3f xy2=%.3f,%.3f",x0,y0,x1,y1,x2,y2);
}

void PaletteDrawer::drawLine(float x0, float y0, float x1, float y1) {
	NosuchDebug("Drawing line xy0=%.3f,%.3f xy1=%.3f,%.3f",x0,y0,x1,y1);
}

static float degree2radian(float deg) {
	return 2.0f * (float)M_PI * deg / 360.0f;
}

void PaletteDrawer::drawEllipse(float x0, float y0, float w, float h, float fromang, float toang) {
	NosuchDebug(2,"Drawing ellipse xy0=%.3f,%.3f wh=%.3f,%.3f",x0,y0,w,h);
#ifdef OLD_GRAPHICS
	if ( m_filled ) {
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		NosuchDebug(2,"   fill_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_fill_alpha);
		glBegin(GL_TRIANGLE_FAN);
		double radius = w;
		glVertex2d(x0, y0);
		for ( double degree=fromang; degree <= toang; degree+=5.0f ) {
			glVertex2d(x0 + sin(degree2radian(degree)) * radius, y0 + cos(degree2radian(degree)) * radius);
		}
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		NosuchDebug(2,"   stroke_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_stroke_alpha);
		if (fromang == 0.0 && toang == 360.0) {
			glBegin(GL_LINE_LOOP);
		} else {
			glBegin(GL_LINES);
		}
		double radius = w;
		for ( double degree=fromang; degree <= toang; degree+=5.0f ) {
			glVertex2d(x0 + sin(degree2radian(degree)) * radius, y0 + cos(degree2radian(degree)) * radius);
		}
		glEnd();
	}

	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, ellipse() called when both m_filled and m_stroked are off!?");
	}
#endif
}

void PaletteDrawer::drawPolygon(PointMem* points, int npoints) {
	NosuchDebug( 2, "Drawing polygon" );
#ifdef OLD_GRAPHICS
	if ( m_filled ) {
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		glBegin(GL_TRIANGLE_FAN);
		glVertex2d(0.0, 0.0);
		for ( int pn=0; pn<npoints; pn++ ) {
			PointMem* p = &points[pn];
			glVertex2d(p->x,p->y);
		}
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		glBegin(GL_LINE_LOOP);
		for ( int pn=0; pn<npoints; pn++ ) {
			PointMem* p = &points[pn];
			glVertex2d(p->x,p->y);
		}
		glEnd();
	}

	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, ellipse() called when both m_filled and m_stroked are off!?");
	}
#endif
}

#define RANDONE (((float)rand())/RAND_MAX)
#define RANDB ((((float)rand())/RAND_MAX)*2.0f-1.0f)

FFResult PaletteDrawer::InitGL( const FFGLViewportStruct* vp)
{
	if( !m_shader_gradient.Compile( vertexShaderGradient, fragmentShaderGradient ) )
	{
		NosuchDebug( "Error in compiling m_shader_gradient!" );
		DeInitGL();
		return FF_FAIL;
	}
	NosuchDebug( "HI From PaletteDrawer::InitGL, shader compiled okay" );
	if( !m_quad.Initialise() )
	{
		DeInitGL();
		return FF_FAIL;
	}
	if( !m_triangle.Initialise() )
	{
		DeInitGL();
		return FF_FAIL;
	}

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.
	ffglex::ScopedShaderBinding shaderBinding( m_shader_gradient.GetGLID() );
	m_rgbLeftLocation  = m_shader_gradient.FindUniform( "RGBALeft" );
	m_rgbRightLocation = m_shader_gradient.FindUniform( "RGBARight" );
	m_matrixLocation = m_shader_gradient.FindUniform( "vMatrix" );

	return FF_SUCCESS;
}

FFResult PaletteDrawer::DeInitGL()
{
	NosuchDebug( "HI From PaletteDrawer::DeInitGL" );
	m_shader_gradient.FreeGLResources();
	m_quad.Release();
	m_triangle.Release();
	m_rgbLeftLocation  = -1;
	m_rgbRightLocation = -1;

	return FF_SUCCESS;
}
