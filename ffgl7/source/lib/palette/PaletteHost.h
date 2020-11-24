#ifndef _PALETTEHOST_H
#define _PALETTEHOST_H

#ifdef _WIN32
#include <windows.h>
// #include <gl/gl.h>
#endif

#define FFGL_ALREADY_DEFINED
#ifndef FFGL_ALREADY_DEFINED

#define FF_SUCCESS					0
#define FF_FAIL					0xFFFFFFFF

// The following typedefs are in FFGL.h, but I don't want to pull that entire file in... 

//FFGLViewportStruct (for InstantiateGL)
typedef struct FFGLViewportStructTag
{
  GLuint x,y,width,height;
} FFGLViewportStruct;

//FFGLTextureStruct (for ProcessOpenGLStruct)
typedef struct FFGLTextureStructTag
{
  DWORD Width, Height;
  DWORD HardwareWidth, HardwareHeight;
  GLuint Handle; //the actual texture handle, from glGenTextures()
} FFGLTextureStruct;

// ProcessOpenGLStruct
// ProcessOpenGLStruct
typedef struct ProcessOpenGLStructTag {
  DWORD numInputTextures;
  FFGLTextureStruct **inputTextures;
  
  //if the host calls ProcessOpenGL with a framebuffer object actively bound
  //(as is the case when the host is capturing the plugins output to an offscreen texture)
  //the host must provide the GL handle to its EXT_framebuffer_object
  //so that the plugin can restore that binding if the plugin
  //makes use of its own FBO's for intermediate rendering
  GLuint HostFBO; 
} ProcessOpenGLStruct;

#endif

#include "osc/OscOutboundPacketStream.h"
#include "NosuchOscInput.h"
#include "PaletteOscInput.h"
#include "NosuchOscInput.h"
#include "NosuchColor.h"
#include "NosuchGraphics.h"
#include "cJSON.h"
#include "Scheduler.h"

#include <FFGLSDK.h>
#include "FFGLPluginSDK.h"
#include "FFGL.h"

class PaletteHost;
class Palette;
class PaletteHttp;
class TrackedCursor;
class GraphicBehaviour;
class AllMorphs;

typedef struct PointMem {
	float x;
	float y;
	float z;
} PointMem;

#define DEFAULT_RESOLUME_PORT 7000
#define DEFAULT_RESOLUME_HOST "127.0.0.1"
#define BASE_OSC_INPUT_PORT 3333
#define DEFAULT_OSC_INPUT_HOST "127.0.0.1"

class PaletteDaemon {
public:
	PaletteDaemon(PaletteHost* mf, int osc_input_port, std::string osc_input_host);
	~PaletteDaemon();
	void *run(void *arg);
private:
	bool _network_thread_created;
	bool daemon_shutting_down;
	pthread_t _network_thread;
	PaletteHost* _paletteHost;
	PaletteOscInput* _oscinput;
	AllMorphs* _morphs;
};

class Effect {
public:
	Effect(std::string nm, cJSON* j) {
		name = nm;
		json = j;
	}
	std::string name;
	cJSON* json;
};

static int transpose_vals[] = { 0,3,-2,5 };

class PaletteHost : public NosuchOscMessageProcessor
{
public:
	PaletteHost(std::string configfile);
	virtual ~PaletteHost();

	///////////////////////////////////////////////////
	// FreeFrame plugin methods
	///////////////////////////////////////////////////
	
	DWORD PaletteHostProcessOpenGL(ProcessOpenGLStruct *pGL);
	DWORD PaletteHostPoke();

	FFResult InitGL( const FFGLViewportStruct* vp );
	FFResult DeInitGL();

	bool initStuff();
	void lock_paletteHost();
	void unlock_paletteHost();

	bool disable_on_exception;
	bool disabled;

	void LoadPaletteConfig(cJSON* c);

	std::string RespondToJson(std::string method, cJSON *params, const char *id);
	std::string ExecuteJson(std::string meth, cJSON *params, const char *id);
	std::string ExecuteJsonAndCatchExceptions(std::string meth, cJSON *params, const char *id);

	void ProcessOscMessage(std::string , const osc::ReceivedMessage& m);

	void SetCursorCid(std::string cid, std::string source, NosuchVector point, double z, bool recordable = true );

	static bool checkAddrPattern(const char *addr, char *patt);

	std::string jsonDoubleResult(double r, const char* id);
	std::string jsonIntResult(int r, const char* id);
	std::string jsonStringResult(std::string r, const char* id);
	std::string jsonMethError(std::string e, const char* id);
	std::string jsonError(int code, std::string e, const char* id);
	std::string jsonConfigResult(std::string name, const char *id);

	// GRAPHICS ROUTINES
	double width;
	double height;

	bool m_filled;
	NosuchColor m_fill_color;
	double m_fill_alpha;
	bool m_stroked;
	NosuchColor m_stroke_color;
	double m_stroke_alpha;

	void fill(NosuchColor c, double alpha);
	void noFill();
	void stroke(NosuchColor c, double alpha);
	void noStroke();
	void strokeWeight(double w);
	void background(int);
	void rect(double x, double y, double width, double height);
	void pushMatrix();
	void popMatrix();
	void translate(double x, double y);
	void scale(double x, double y);
	void rotate(double degrees);
	void line(double x0, double y0, double x1, double y1);
	void triangle(double x0, double y0, double x1, double y1, double x2, double y2);
	void quad(double x0, double y0, double x1, double y1, double x2, double y2, double x3, double y3);
	void ellipse(double x0, double y0, double w, double h, double fromang=0.0f, double toang=360.0f);
	void polygon(PointMem* p, int npoints);

	///////////////////////////////////////////////////
	// Factory method
	///////////////////////////////////////////////////

	PaletteDaemon* _daemon;

	Scheduler* scheduler() {
		return _scheduler;
	}

	Palette* palette() { return _palette; }

	int gl_frame;

	static void ErrorPopup(const char* msg);
	void RunEveryMillisecondOrSo();

	static int PortOffset;  // applied to http and osc ports

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

	ffglex::FFGLShader m_shader;  //!< Utility to help us compile and link some shaders into a program.
	ffglex::FFGLScreenQuad m_quad;//!< Utility to help us render a full screen quad.
	ffglex::FFGLScreenTriangle m_triangle;//!< Utility to help us render a full screen quad.
	GLint m_rgbLeftLocation;
	GLint m_rgbRightLocation;

protected:	

	Palette* _palette;
	Scheduler* _scheduler;
	
	pthread_mutex_t json_mutex;
	pthread_cond_t json_cond;

	pthread_mutex_t palette_mutex;

	bool json_pending;
	std::string json_method;
	cJSON* json_params;
	const char *json_id;
	std::string json_result;

	bool gl_shutting_down;
	bool initialized;

	static void* ThreadPointer;
	static bool StaticInitialized;
	static void StaticInitialization();

	void test_draw();

private:
	Timestamp _time0;
	bool _dotest;
	std::map<std::string, cJSON*> _patchJson;
	std::string _configFile;
	cJSON* _configJson;

	int SendToResolume(osc::OutboundPacketStream& p);

};

#endif
