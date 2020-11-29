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

// The coordinate space used is (0,0) to (1,1), lower-left to upper-right

class PaletteDrawer {

public:
	PaletteDrawer(PaletteParams *params);
	virtual ~PaletteDrawer();

	FFResult InitGL( const FFGLViewportStruct* vp );
	FFResult DeInitGL();
	void initBuffers();
	void releaseBuffers();

	float scale_z( float z );

	ffglex::FFGLShader* BeginDrawingWithShader(std::string shaderName);
	bool prepareToDraw( SpriteParams& params, SpriteState& state );
	void EndDrawing();

	void strokeWeight(float w);
	void background(int);
	void resetMatrix();
	void translate(float x, float y);
	void scale(float x, float y);
	void rotate(float radians);

	void drawLine(SpriteParams& params, SpriteState& state, float x0, float y0, float x1, float y1);
	void drawTriangle(SpriteParams& params, SpriteState& state, float x0, float y0, float x1, float y1, float x2, float y2);
	void drawQuad(SpriteParams& params, SpriteState& state, float x0, float y0, float x1, float y1, float x2, float y2, float x3, float y3);
	void drawEllipse(SpriteParams& params, SpriteState& state, float x0, float y0, float w, float h, float fromang=0.0f, float toang=360.0f);
	void drawPolygon(PointMem* p, int npoints);

private:

	FFGLViewportStruct m_vp;
	PaletteParams *m_params;
	bool m_isdrawing;
	
	glm::mat4 m_matrix;
	glm::mat4 m_matrix_identity;

#define MAX_VERTICES 72
	ffglex::GlVertexTextured vertices[ MAX_VERTICES ];

	GLuint vaoID;
	GLuint vboID;

	ffglex::FFGLShader m_shader_gradient;  //!< Utility to help us compile and link some shaders into a program.

	GLint m_rgbLeftLocation;
	GLint m_rgbRightLocation;
	GLint m_matrixLocation;
};
