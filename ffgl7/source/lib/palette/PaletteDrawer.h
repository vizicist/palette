#pragma once

class Palette;
class PaletteHttp;
class TrackedCursor;
class GraphicBehaviour;
class AllMorphs;
class PaletteParams;

typedef struct PointMem {
	float x;
	float y;
	float z;
} PointMem;

#define DEFAULT_RESOLUME_PORT 7000
#define DEFAULT_RESOLUME_HOST "127.0.0.1"
#define BASE_OSC_INPUT_PORT 3333
#define DEFAULT_OSC_INPUT_HOST "127.0.0.1"

class PaletteDrawer {

public:
	PaletteDrawer(PaletteParams *params);
	virtual ~PaletteDrawer();

	FFResult InitGL( const FFGLViewportStruct* vp );
	FFResult DeInitGL();

	double scale_z( double z );

	double width() { return m_width; }
	double height() { return m_height; }

	void fill(NosuchColor c, double alpha);
	void noFill();
	void stroke(NosuchColor c, double alpha);
	void noStroke();
	void strokeWeight(double w);
	void background(int);
	void pushMatrix();
	void popMatrix();
	void translate(double x, double y);
	void scale(double x, double y);
	void rotate(double degrees);

	void drawLine(double x0, double y0, double x1, double y1);
	void drawTriangle(double x0, double y0, double x1, double y1, double x2, double y2);
	void drawQuad(double x0, double y0, double x1, double y1, double x2, double y2, double x3, double y3);
	void drawEllipse(double x0, double y0, double w, double h, double fromang=0.0f, double toang=360.0f);
	void drawPolygon(PointMem* p, int npoints);

private:

	PaletteParams *m_params;
	
	// GRAPHICS ROUTINES
	double m_width;
	double m_height;

	bool m_filled;
	NosuchColor m_fill_color;
	double m_fill_alpha;
	bool m_stroked;
	NosuchColor m_stroke_color;
	double m_stroke_alpha;

	// NEW STUFF

	struct RGBA
	{
		float red   = 1.0f;
		float green = 1.0f;
		float blue  = 0.0f;
		float alpha = 1.0f;
	};
	struct HSBA
	{
		float hue   = 0.0f;
		float sat   = 1.0f;
		float bri   = 1.0f;
		float alpha = 1.0f;
	};
	RGBA rgba1;
	HSBA hsba2;

	ffglex::FFGLShader m_shader_gradient;  //!< Utility to help us compile and link some shaders into a program.
	DrawQuad m_quad;//!< Utility to help us render a full screen quad.
	DrawTriangle m_triangle;//!< Utility to help us render a full screen quad.
	GLint m_rgbLeftLocation;
	GLint m_rgbRightLocation;
};
