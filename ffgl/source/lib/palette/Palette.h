#pragma once

class PaletteParams {
public:
	PaletteParams() {
		showfps = false;
		zexponential = 2.0;
		zmultiply = 1.4f;
		switchyz = false;
		area2d = 0.03;
		depth2d = 0.03;
	}
	bool showfps;
	double zexponential;	// -3 to 3
	double zmultiply;		// 0.1 to 11.0
	bool switchyz;
	double area2d;
	double depth2d;
};

class Palette {

public:
	Palette(PaletteHost* b);
	~Palette();

	// STATIC STUFF
	static Palette* _singleton;
	static Palette* palette() { return _singleton; }
	static const double UNSET_DOUBLE;
	static const std::string UNSET_STRING;

	static const std::string configSuffix;
	static const std::string configSeparator;
	static bool initialized;
	static int lastsprite;

	static void initialize();

	static int now;   // milliseconds
	static const int idleattract = 0;

	// NON-STATIC STUFF

	PaletteParams params;
	PaletteDrawer* p;

	PaletteHost* paletteHost() { return _paletteHost; }
	
	FFResult InitGL( const FFGLViewportStruct* vp );
	FFResult DeInitGL();

	// Scheduler* scheduler() { return _paletteHost->scheduler(); }

	void LockPalette() {
		NosuchLock(&_palette_mutex,"palette");
	}
	void UnlockPalette() {
		NosuchUnlock(&_palette_mutex,"palette");
	}

	void clear();
	int draw();
	int drawbg();
	void advanceTo(int tm);

	void LoadParamPush(cJSON* sound, cJSON* visual);
	std::string loadParamPushReal(cJSON* sound, cJSON* visual);

	Region region;

	PaletteDrawer* Drawer() { return _drawer; }

private:

	PaletteHost* _paletteHost;
	PaletteDrawer* _drawer;
	pthread_mutex_t _palette_mutex;

	int _frames;
	int _frames_last;

};
