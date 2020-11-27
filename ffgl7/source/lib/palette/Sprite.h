#ifndef _SPRITE_H
#define _SPRITE_H

// Note - this Params class is different from PaletteParams and RegionParams because
// it doesn't need the Set/Increment/Toggle methods.

class SpriteParams {

public:

	void initValues(Region* r) {

		RegionParams& rp = r->params;

#define INIT_PARAM(name) name = rp.##name;

#include "SpriteParams_init.h"
	}

#include "SpriteParams_declare.h"
};

class SpriteList {

public:
	SpriteList();
	void lock_read();
	void lock_write();
	void unlock();
	void draw(PaletteDrawer* b);
	void clear();
	void advanceTo(int tm, int gravity);
	void computeForce(std::list<Sprite*> &sprites, int gravity);
	void add(Sprite* s, int limit);
	int size() {
		return (int)(sprites.size());
	}

private:
	std::list<Sprite*> sprites;
	pthread_rwlock_t rwlock;

};

class SpriteState {
public:
	SpriteState() {
		visible = false;
		direction = 0.0;
		hue1 = 0.0f;
		hue2 = 0.0f;
		pos = NosuchVector(0.0f,0.0f);
		depth = 0.0;
		size = 0.5;
		alpha = 1.0;
		born = 0;
		last_tm = 0;
		killme = false;
		rotangsofar = 0.0f;
		stationary = false;
		cid = "nosuchcursor";
		rotanginit = 0.0;
		gravityForce = NosuchVector(0.0, 0.0);
	}
	bool visible;
	float direction;
	float hue1;
	float hue2;
	NosuchVector pos;
	float depth;
	float size;
	float alpha;
	int born;
	int last_tm;
	bool killme;
	float rotangsofar;
	bool stationary;
	std::string cid;
	int seq;     // sprite sequence # (mostly for debugging)
	int rotdir;  // -1, 0, 1
	float rotanginit;
	NosuchVector gravityForce;
};

class Sprite {
	
public:

	Sprite();
	virtual ~Sprite();

	virtual void drawShape(PaletteDrawer* app, int xdir, int ydir) = 0;
	virtual void startAccumulate(TrackedCursor* c) { };
	virtual void accumulate(TrackedCursor* c) { }

	virtual float width() { return float(state.size * state.depth); }
	virtual float height() { return float(state.size * state.depth); }

	static bool initialized;
	static void initialize();
	// static std::vector<std::string> spriteShapes;
	static float degree2radian(float deg);

	void initState(std::string cid, std::string cidsource, NosuchVector& pos, float movedir, float depth, float rotanginit);

	// Screen space is 2.0x2.0, while cursor space is 1.0x1.0
	void scaleCursorSpaceToScreenSpace(NosuchVector& pos) {
		state.pos.x *= 2.0f;
		state.pos.y *= 2.0f;
	}

	void draw(PaletteDrawer* app);
	void drawAt(PaletteDrawer* app, float x,float y, float w, float h, int xdir, int ydir);
	NosuchVector deltaInDirection(float dt, float dir, float speed);
	int rotangdirOf(std::string s);
	void advanceTo(int tm, NosuchVector force);

	SpriteParams params;
	SpriteState state;
	SpriteDrawer* drawFunc;
	ffglex::FFGLShader* shader;

protected:
	float vertexNoise();

	static int NextSeq;

private:
	void draw(PaletteDrawer* app, float scaled_z);
};

class SpriteSquare : public Sprite {

public:
	SpriteSquare();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);

private:
	bool noise_initialized;
	float noise_x0;
	float noise_y0;
	float noise_x1;
	float noise_y1;
	float noise_x2;
	float noise_y2;
	float noise_x3;
	float noise_y3;
};

class SpriteTriangle : public Sprite {

public:
	SpriteTriangle();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);

private:
	bool noise_initialized;
	float noise_x0;
	float noise_y0;
	float noise_x1;
	float noise_y1;
	float noise_x2;
	float noise_y2;
};

class SpriteCircle : public Sprite {

public:
	SpriteCircle();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);
};

class SpriteArc : public Sprite {

public:
	SpriteArc();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);
};

class SpriteLine : public Sprite {

public:
	SpriteLine();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);

private:
	bool noise_initialized;
	float noise_x0;
	float noise_y0;
	float noise_x1;
	float noise_y1;
};

#endif
