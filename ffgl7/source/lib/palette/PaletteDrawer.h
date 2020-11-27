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

typedef void (*SpriteDrawer)( PaletteDrawer* drawer, int xdir, int ydir );

class PaletteDrawer {

public:
	PaletteDrawer(PaletteParams *params);
	virtual ~PaletteDrawer();

	FFResult InitGL( const FFGLViewportStruct* vp );
	FFResult DeInitGL();

	double scale_z( double z );

	ffglex::FFGLShader* BeginDrawingWithShader(std::string shaderName);
	void EndDrawing();

	double width() { return m_width; }
	double height() { return m_height; }

	void fill(NosuchColor c, double alpha);
	void noFill();
	void stroke(NosuchColor c, double alpha);
	void strokeWeight(double w);
	void background(int);
	void pushMatrix();
	void popMatrix();
	void translate(double x, double y);
	void scale(double x, double y);
	void rotate(double degrees);

	void drawLine(double x0, double y0, double x1, double y1);
	void drawTriangle(double x0, double y0, double x1, double y1, double x2, double y2);
	void drawQuad(float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3);
	void drawEllipse(double x0, double y0, double w, double h, double fromang=0.0f, double toang=360.0f);
	void drawPolygon(PointMem* p, int npoints);

private:

	PaletteParams *m_params;
	bool m_isdrawing;
	
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
		float red;
		float green;
		float blue;
		float alpha;
	};
	struct HSBA
	{
		float hue;
		float sat;
		float bri;
		float alpha;
	};
	RGBA m_rgba1;
	HSBA m_hsba2;

	ffglex::FFGLShader m_shader_gradient;  //!< Utility to help us compile and link some shaders into a program.
	DrawQuad m_quad;//!< Utility to help us render a full screen quad.
	DrawTriangle m_triangle;//!< Utility to help us render a full screen quad.
	GLint m_rgbLeftLocation;
	GLint m_rgbRightLocation;
};
