#ifndef _SPRITE_H
#define _SPRITE_H

class SpriteDrawer;

#define SpriteParams LayerParams

#if NOLONGERNEEDED
// Note - this Params class is different from PaletteParams and LayerParams because
// it doesn't need the Set/Increment/Toggle methods.

class SpriteParams {

public:

	void initValues(Layer* r) {

		LayerParams& rp = r->params;

#undef INIT_PARAM
#define INIT_PARAM(name,def) name = rp.##name;
#include "LayerParams_init.h"
	}

#include "LayerParams_declare.h"
};
#endif

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
		pos = glm::vec2(0.0f,0.0f);
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
		gravityForce = glm::vec2(0.0, 0.0);
	}
	bool visible;
	float direction;
	float hue1;
	float hue2;
	glm::vec2 pos;
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
	glm::vec2 gravityForce;
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

	void initState(std::string cid, std::string cidsource, glm::vec2& pos, float movedir, float depth, float rotanginit);

	// Screen space is 2.0x2.0, while cursor space is 1.0x1.0
	void scaleCursorSpaceToScreenSpace(glm::vec2& pos) {
		state.pos.x *= 2.0f;
		state.pos.y *= 2.0f;
	}

	void draw(PaletteDrawer* app);
	void drawAt(PaletteDrawer* app, float x,float y, float w, float h, int xdir, int ydir);
	glm::vec2 deltaInDirection(float dt, float dir, float speed);
	int rotangdirOf(std::string s);
	void advanceTo(int tm, glm::vec2 force);

	SpriteParams params;
	SpriteState state;
	ffglex::FFGLShader* shader;

protected:
	glm::vec2 vertexNoise();

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
	glm::vec2 noise_0;
	glm::vec2 noise_1;
	glm::vec2 noise_2;
	glm::vec2 noise_3;
};

class SpriteTriangle : public Sprite {

public:
	SpriteTriangle();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);

private:
	bool noise_initialized;
	glm::vec2 noise_0;
	glm::vec2 noise_1;
	glm::vec2 noise_2;
	glm::vec2 rotate( glm::vec2, float, glm::vec2 );
};

class SpriteCircle : public Sprite {

public:
	SpriteCircle();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);
};

class SpriteLine : public Sprite {

public:
	SpriteLine();
	void drawShape(PaletteDrawer* app, int xdir, int ydir);

private:
	bool noise_initialized;
	glm::vec2 noise_0;
	glm::vec2 noise_1;
};

#endif
