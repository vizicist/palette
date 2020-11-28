#ifndef _PALETTEHOST_H
#define _PALETTEHOST_H

#ifdef _WIN32
#include <windows.h>
// #include <gl/gl.h>
#endif

class PaletteHost;
class Palette;
class PaletteHttp;
class TrackedCursor;
class GraphicBehaviour;
class AllMorphs;
class PaletteOscInput;

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

	void SetCursorCid( std::string cid, std::string source, glm::vec2 point, float z, bool recordable = true );

	static bool checkAddrPattern(const char *addr, char *patt);

	std::string jsonDoubleResult(double r, const char* id);
	std::string jsonIntResult(int r, const char* id);
	std::string jsonStringResult(std::string r, const char* id);
	std::string jsonMethError(std::string e, const char* id);
	std::string jsonError(int code, std::string e, const char* id);
	std::string jsonConfigResult(std::string name, const char *id);

	// GRAPHICS ROUTINES
	float width;
	float height;

	PaletteDrawer* _drawer;

	///////////////////////////////////////////////////
	// Factory method
	///////////////////////////////////////////////////

	PaletteDaemon* _daemon;

	Scheduler* scheduler() {
		return _scheduler;
	}

	Palette* palette() { return _palette; }
	void SetOscPort( std::string oscport );
	std::string GetOscPort( );

	void RunEveryMillisecondOrSo();

	// NEW STUFF

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

	std::string m_oscport;

private:
	Timestamp _time0;
	bool _dotest;
	std::map<std::string, cJSON*> _patchJson;
	std::string _configFile;
	cJSON* _configJson;

	int SendToResolume(osc::OutboundPacketStream& p);

};

#endif
