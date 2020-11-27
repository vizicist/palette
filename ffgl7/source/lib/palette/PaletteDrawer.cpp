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
#include "FFGLSDK.h"
#include "FFGLLib.h"
#include "osc/OscOutboundPacketStream.h"
#include "cJSON.h"

using namespace ffglex;

static const char vertexShaderGradient[] = R"(#version 410 core
layout( location = 0 ) in vec4 vPosition;
layout( location = 1 ) in vec2 vUV;

uniform vec2 vScale;
uniform vec2 vTranslate;

out vec2 uv;

void main()
{
	gl_Position = vec4((vPosition.x*vScale.x)+vTranslate.x,(vPosition.y*vScale.y)+vTranslate.y,vPosition.z,vPosition.a);
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

PaletteDrawer::PaletteDrawer(PaletteParams *params)
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

	m_width              = 1.0;
	m_height              = 1.0;

	m_rgba1 = { 1.0f, 1.0f, 0.0f, 1.0f };
	m_hsba2 = { 0.0f, 1.0f, 1.0f, 1.0f };
}

PaletteDrawer::~PaletteDrawer()
{
	NosuchDebug(1,"PaletteDrawer destructor called");
}

////////////////////////////////////////////////////////////////////////////////////////////////////
//  Methods
////////////////////////////////////////////////////////////////////////////////////////////////////

void PaletteDrawer::fill(NosuchColor c, double alpha) {
	m_filled = true;
	m_fill_color = c;
	m_fill_alpha = alpha;
}
void PaletteDrawer::stroke(NosuchColor c, double alpha) {
	// glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, alpha);
	m_stroked = true;
	m_stroke_color = c;
	m_stroke_alpha = alpha;
}
void PaletteDrawer::noStroke() {
	m_stroked = false;
}
void PaletteDrawer::noFill() {
	m_filled = false;
}
void PaletteDrawer::background(int b) {
	NosuchDebug("PaletteDrawer::background!");
}
void PaletteDrawer::strokeWeight(double w) {
	glLineWidth((GLfloat)w);
}
void PaletteDrawer::rotate(double degrees) {
#ifdef OLD_GRAPHICS
	glRotated(-degrees,0.0f,0.0f,1.0f);
#endif
}
void PaletteDrawer::translate(double x, double y) {
#ifdef OLD_GRAPHICS
	glTranslated(x,y,0.0f);
#endif
}
void PaletteDrawer::scale(double x, double y) {
#ifdef OLD_GRAPHICS
	glScaled(x,y,1.0f);
	// NosuchDebug("SCALE xy= %f %f",x,y);
#endif
}

double PaletteDrawer::scale_z(double z) {
	// We want the z value to be scaled exponentially toward 1.0,
	// i.e. raw z of .5 should result in a scale_z value of .75
	double expz = 1.0f - pow((1.0-z),m_params->zexponential);
	return expz * m_params->zmultiply;
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
	NosuchDebug("PaletteDrawer.drawQuad: %.3f %.3f, %.3f %.3f, %.3f %.3f, %.3f %.3f",x0,y0,x1,y1,x2,y2,x3,y3);

	//FFGL requires us to leave the context in a default state on return, so use this scoped binding to help us do that.

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

	GLfloat xscale = 0.5f;
	GLfloat yscale = 0.5f;
	m_shader_gradient.Set( "vScale", xscale, yscale );
	GLfloat xtranslate = 0.0f;
	GLfloat ytranslate = 0.0f;
	m_shader_gradient.Set( "vTranslate", xtranslate, ytranslate );

	m_quad.Draw(x0,y0,x1,y1,x2,y2,x3,y3);

#ifdef OLD_GRAPHICS
	if ( m_filled ) {
		glBegin(GL_QUADS);
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1); 
		glVertex2d( x2, y2); 
		glVertex2d( x3, y3); 
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		glBegin(GL_LINE_LOOP); 
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1); 
		glVertex2d( x2, y2); 
		glVertex2d( x3, y3); 
		glEnd();
	}
	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, quad() called when both m_filled and m_stroked are off!?");
	}
#endif
}
void PaletteDrawer::drawTriangle(double x0, double y0, double x1, double y1, double x2, double y2) {
	NosuchDebug(2,"Drawing triangle xy0=%.3f,%.3f xy1=%.3f,%.3f xy2=%.3f,%.3f",x0,y0,x1,y1,x2,y2);
#ifdef OLD_GRAPHICS
	if ( m_filled ) {
		NosuchColor c = m_fill_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_fill_alpha);
		NosuchDebug(2,"   fill_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_fill_alpha);
		glBegin(GL_TRIANGLE_STRIP); 
		glVertex3d( x0, y0, 0.0f );
		glVertex3d( x1, y1, 0.0f );
		glVertex3d( x2, y2, 0.0f );
		glEnd();
	}
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		NosuchDebug(2,"   stroke_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_stroke_alpha);
		glBegin(GL_LINE_LOOP); 
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1);
		glVertex2d( x2, y2);
		glEnd();
	}
	if ( ! m_filled && ! m_stroked ) {
		NosuchDebug("Hey, triangle() called when both m_filled and m_stroked are off!?");
	}
#endif
}

void PaletteDrawer::drawLine(double x0, double y0, double x1, double y1) {
	NosuchDebug(2,"Drawing line xy0=%.3f,%.3f xy1=%.3f,%.3f",x0,y0,x1,y1);
#ifdef OLD_GRAPHICS
	if ( m_stroked ) {
		NosuchColor c = m_stroke_color;
		glColor4d(c.r()/255.0f, c.g()/255.0f, c.b()/255.0f, m_stroke_alpha);
		// NosuchDebug(2,"   stroke_color=%d %d %d alpha=%.3f",c.r(),c.g(),c.b(),m_stroke_alpha);
		glBegin(GL_LINES); 
		glVertex2d( x0, y0); 
		glVertex2d( x1, y1);
		glEnd();
	} else {
		NosuchDebug("Hey, line() called when m_stroked is off!?");
	}
#endif
}

static double degree2radian(double deg) {
	return 2.0f * (double)M_PI * deg / 360.0f;
}

void PaletteDrawer::drawEllipse(double x0, double y0, double w, double h, double fromang, double toang) {
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

void PaletteDrawer::popMatrix() {
#ifdef OLD_GRAPHICS
	glPopMatrix();
#endif
}

void PaletteDrawer::pushMatrix() {
#ifdef OLD_GRAPHICS
	glPushMatrix();
#endif
}

#define RANDONE (((double)rand())/RAND_MAX)
#define RANDB ((((double)rand())/RAND_MAX)*2.0f-1.0f)

FFResult PaletteDrawer::InitGL( const FFGLViewportStruct* vp)
{
	NosuchDebug( "HI From PaletteDrawer::InitGL" );
	if( !m_shader_gradient.Compile( vertexShaderGradient, fragmentShaderGradient ) )
	{
		DeInitGL();
		return FF_FAIL;
	}
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
