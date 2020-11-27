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

	float scale_z( float z );

	ffglex::FFGLShader* BeginDrawingWithShader(std::string shaderName);
	void EndDrawing();

	float width() { return m_width; }
	float height() { return m_height; }

	void fill(NosuchColor c, float alpha);
	void noFill();
	void stroke(NosuchColor c, float alpha);
	void strokeWeight(float w);
	void background(int);
	void resetMatrix();
	// void setMatrix(GLfloat matrix[16]);
	void translate(float x, float y);
	void scale(float x, float y);
	void rotate(float radians);

	void drawLine(float x0, float y0, float x1, float y1);
	void drawTriangle(float x0, float y0, float x1, float y1, float x2, float y2);
	void drawQuad(float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3);
	void drawEllipse(float x0, float y0, float w, float h, float fromang=0.0f, float toang=360.0f);
	void drawPolygon(PointMem* p, int npoints);

private:

	PaletteParams *m_params;
	bool m_isdrawing;
	
	float m_width;
	float m_height;
	bool m_filled;
	NosuchColor m_fill_color;
	float m_fill_alpha;
	bool m_stroked;
	NosuchColor m_stroke_color;
	float m_stroke_alpha;

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
	// GLfloat m_matrix[16];
	glm::mat4 m_matrix;
	glm::mat4 m_matrix_identity;

	ffglex::FFGLShader m_shader_gradient;  //!< Utility to help us compile and link some shaders into a program.
	DrawQuad m_quad;//!< Utility to help us render a full screen quad.
	DrawTriangle m_triangle;//!< Utility to help us render a full screen quad.
	GLint m_rgbLeftLocation;
	GLint m_rgbRightLocation;
	GLint m_matrixLocation;
};
