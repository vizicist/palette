#include <cstdlib> // for srand, rand

#include "PaletteAll.h"

bool Sprite::initialized = false;
long nsprites = 0;
int Sprite::NextSeq = 0;

glm::vec2 Sprite::vertexNoise()
{
	float x = 0.0f;
	float y = 0.0f;
	if ( params.noisevertexx > 0.0f ) {
		x = (float)(params.noisevertexx * RANDFLOAT * ((rand()%2)==0?1:-1));
		NosuchDebug( "noise x=%f\n", x );
	}
	if ( params.noisevertexy > 0.0f ) {
		y = (float)(params.noisevertexy * RANDFLOAT * ((rand()%2)==0?1:-1));
		NosuchDebug( "noise y=%f\n", y );
	}
	return glm::vec2(x,y);
}

void Sprite::initialize() {
	if ( initialized )
		return;
	initialized = true;
}
	
Sprite::Sprite() {
}

void
Sprite::initState(std::string cid, std::string cidsource, glm::vec2& pos, float movedir, float depth, float rotanginit) {

	nsprites++;
	Palette::lastsprite = Palette::now;

	// most of the state has been initialized in SpriteState constructor
	state.pos = pos;
	state.direction = movedir;
	state.depth = depth;
	state.cid = cid;

	state.born = Palette::now;
	state.last_tm = Palette::now;
	state.hue1 = params.hue1initial;
	state.hue2 = params.hue2initial;
	state.alpha = params.alphainitial;
	state.size = params.sizeinitial;
	state.seq = NextSeq++;
	state.rotdir = rotangdirOf(params.rotangdir);
	state.rotanginit = rotanginit;
	state.rotangsofar = state.rotanginit;
}

Sprite::~Sprite() {
	NosuchDebug(1, "Sprite destructor! s=%d cid=%s", this, state.cid.c_str());
}

float Sprite::degree2radian(float deg) {
	return 2.0f * float(M_PI) * deg / 360.0f;
}

void Sprite::draw(PaletteDrawer* drawer) {

	if ( ! state.visible ) {
		NosuchDebug("Sprite.draw NOT DRAWING, !visible");
		return;
	}
	
	if ( state.alpha <= 0.0f || state.size < 0.001f ) {
		state.killme = true;
		return;
	}

	shader = drawer->BeginDrawingWithShader("gradient");
	if( shader == NULL )
	{
		NosuchDebug( "No gradient shader?  Unable to draw Sprite.");
		return;
	}

	if (state.depth < params.zmin ) {
		state.depth = params.zmin;
	}
	float scaled_z = drawer->scale_z(state.depth);
	
	// XXX - this should move into PaletteDrawer, I suspect
	float thickness = params.thickness;
	drawer->strokeWeight(thickness);

	float scalex = state.size * scaled_z;
	float scaley = state.size * scaled_z;

	// These control flipping of the drawing orientation
	int xdir = 1;
	int ydir = 1;

	float x = state.pos.x;
	float y = state.pos.y;

	if ( params.mirrortype == "four" ) {
		drawAt(drawer,x,y,scalex,scaley,xdir,ydir);
		ydir = -1;
		drawAt(drawer,x,1.0f-y,scalex,scaley,xdir,ydir);
		xdir = -1;
		drawAt(drawer,1.0f-x,y,scalex,scaley,xdir,ydir);
		ydir = 1;
		drawAt(drawer,1.0f-x,1.0f-y,scalex,scaley,xdir,ydir);
	} else if ( params.mirrortype == "vertical" ) {
		drawAt(drawer,x,y,scalex,scaley,xdir,ydir);
		ydir = -1;
		drawAt(drawer,x,1.0f-y,scalex,scaley,xdir,ydir);
	} else if ( params.mirrortype == "horizontal" ) {
		drawAt(drawer,x,y,scalex,scaley,xdir,ydir);
		xdir = -1;
		drawAt(drawer,1.0f-x,y,scalex,scaley,xdir,ydir);
	} else {
		drawAt(drawer,x,y,scalex,scaley,xdir,ydir);
	}

	drawer->EndDrawing();
}
	
void Sprite::drawAt(PaletteDrawer* drawer, float x, float y, float scalex, float scaley, int xdir, int ydir) {
	drawer->resetMatrix();

	// handle justification
	std::string j = params.justification;

	glm::vec2 pos( x, y );

	NosuchDebug(1,"Sprite::drawAt s=%lld drawAt j=%s xy= %f %f width=%f size=%f depth=%f\n",
		(long long)this,j.c_str(),pos.x,pos.y,width(),state.size,state.depth);

	float halfWidth = width() / 2.0f;
	float halfHeight = height() / 2.0f;

	if (j == "center") {
		// do nothing
	} else if ( j == "left" ) {
		pos += glm::vec2( halfWidth,0.0f );
	} else if ( j == "right" ) {
		pos += glm::vec2( -halfWidth,0.0f );
	} else if ( j == "top" ) {
		pos += glm::vec2( 0.0f, -halfHeight );
	} else if ( j == "bottom" ) {
		pos += glm::vec2( 0.0f, halfHeight );
	} else if ( j == "topleft" ) {
		pos += glm::vec2( halfWidth, -halfHeight );
	} else if ( j == "topright" ) {
		pos += glm::vec2( -halfWidth, -halfHeight );
	} else if ( j == "bottomleft" ) {
		pos += glm::vec2( halfWidth, halfHeight );
	} else if ( j == "bottomright" ) {
		pos += glm::vec2( -halfWidth, halfHeight );
	} else {
		NosuchDebug("Sprite::drawAt: Unknown justification value - %s\n", params.justification.c_str());
	}

	float degrees = state.rotanginit + state.rotangsofar;

	// XXX - I'm not sure why I have to do this - there's an extra translate/scale somewhere
	// XXX - in the rendering or assumptions, and this is needed to negate it.
	// XXX - Eventually both should be removed.
	drawer->translate(-1.0,-1.0);
	drawer->scale(2.0,2.0);

	drawer->translate(pos.x,pos.y);
	drawer->scale(scalex,scaley);
	drawer->rotate(degree2radian(degrees));

	drawShape( drawer, xdir, ydir );
}

glm::vec2 Sprite::deltaInDirection(float dt, float dir, float speed) {
	glm::vec2 delta( cos(degree2radian(dir)), sin(degree2radian(dir)));
	delta = glm::normalize( delta );
	speed /= 2.0;	// slow things down
	delta = delta * ((dt / 1000.0f) * speed);
	return delta;
}

int Sprite::rotangdirOf(std::string s) {
	int dir = 1;
	if ( s == "right" ) {
		dir = 1;
	} else if ( s == "left" ) {
		dir = -1;
	} else if ( s == "random" ) {
		dir = ((rand()%2) == 0) ? 1 : -1;
	} else {
		NosuchDebug("Sprite.advanceto, bad value for rotangdir!? = %s, assuming random",s.c_str());
		dir = ((rand()%2) == 0) ? 1 : -1;
	}
	return dir;
}

float
envelopeValue(float initial, float final, float duration, float born, float now) {
	float dt = now - born;
	float dur = duration * 1000.0f;
	if ( dt >= dur )
		return final;
	if ( dt <= 0 )
		return initial;
	return initial + (final-initial) * ((now-born)/(dur));
}

void Sprite::advanceTo(int now, glm::vec2 force) {

	// _params->advanceTo(tm);
	state.alpha = envelopeValue(params.alphainitial,params.alphafinal,params.alphatime,float(state.born),float(now));
	state.size = envelopeValue(params.sizeinitial,params.sizefinal,params.sizetime,float(state.born),float(now));
	
	int dnow = (now - state.born);
	// NosuchDebug("Sprite::advanceTo this=%lld now=%d born=%d dnow=%d alpha=%f size=%f last_tm=%d",(long long)this,now,state.born,dnow,state.alpha,state.size,state.last_tm);
	if (params.lifetime >= 0.0 && ((now - state.born) > (1000.0 * params.lifetime))) {
		// NosuchDebug("Lifetime of Sprite %lld exceeded, setting killme",(long long)this);
		state.killme = true;
	}
	float dt = float(now - state.last_tm);
	state.last_tm = now;
	
	if ( ! state.visible ) {
		return;
	}
	
	state.hue1 = envelopeValue(params.hue1initial,params.hue1final,params.hue1time,float(state.born),float(now));
	state.hue2 = envelopeValue(params.hue2initial,params.hue2final,params.hue2time,float(state.born),float(now));

	// state.hueoffset = fmod((state.hueoffset + params.cyclehue), 360.0);

	if ( state.stationary ) {
		NosuchDebug(2,"Sprite %d is stationary",this);
		return;
	}

	if ( params.rotangspeed != 0.0 ) {
		state.rotangsofar = float(fmod((state.rotangsofar + state.rotdir * (dt/1000.0) * params.rotangspeed) , 360.0));
	}
	// if (params.rotauto) {
		// state.rotangsofar = curr_degrees;
	// state.rotangsofar += state.rotanginit;
	// }

	if (force.x != 0.0) {
		state.pos.x += dt * force.x;
	}
	if (force.y != 0.0) {
		state.pos.y += dt * force.y;
	}
	
	if ( params.speed != 0.0 ) {
		
		float dir = state.direction;
		
		glm::vec2 delta = deltaInDirection(dt,dir,params.speed);
		
		glm::vec2 npos = state.pos + delta;
		// NosuchDebug("sprite advance dt=%f dir=%f speed=%f delta=%f,%f npos=%f,%f",
		// 	dt, dir, params.speed, delta.x, delta.y, npos.x, npos.y);
		if ( params.bounce ) { 
			
			if ( npos.x > 1.0f ) {
				dir = float(fmod(( dir + 180 ) , 360));
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos + delta;
			}
			if ( npos.x < 0.0f ) {
				dir = float(fmod(( dir + 180 ) , 360));
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos + delta;
			}
			if ( npos.y > 1.0f ) {
				dir = float(fmod(( dir + 180 ) , 360));
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos + delta;
			}
			if ( npos.y < 0.0f ) {
				dir = float(fmod(( dir + 180 ) , 360));
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos + delta;
			}
			state.direction = dir;
		} else {
			 if (npos.x > 1.0f || npos.x < 0.0f || npos.y > 1.0f || npos.y < 0.0f) {
				 state.killme = true;
			 }
		}

		state.pos = npos;
	}
}

SpriteList::SpriteList() {
	rwlock = PTHREAD_RWLOCK_INITIALIZER;
	int rc1 = pthread_rwlock_init(&rwlock, NULL);
	if (rc1) {
		NosuchDebug("Failure on pthread_rwlock_init!? rc=%d", rc1);
	}
	NosuchDebug(2, "rwlock has been initialized");
}

void
SpriteList::lock_read() {
	int e = pthread_rwlock_rdlock(&rwlock);
	if (e != 0) {
		NosuchDebug("rwlock for read failed!? e=%d", e);
	}
}

void
SpriteList::lock_write() {
	int e = pthread_rwlock_wrlock(&rwlock);
	if (e != 0) {
		NosuchDebug("rwlock for write failed!? e=%d", e);
	}
}

void
SpriteList::unlock() {
	int e = pthread_rwlock_unlock(&rwlock);
	if (e != 0) {
		NosuchDebug("rwlock unlock failed!? e=%d", e);
	}
}

void
SpriteList::add(Sprite* s, int limit)
{
	lock_write();
	sprites.push_back(s);
	NosuchAssert(limit >= 1);
	if (NosuchDebugSprite) {
		NosuchDebug("SpriteList.add: cid=%s", s->state.cid.c_str());
	}
	while ((int)sprites.size() > limit) {
		Sprite* ps = sprites.front();
		if (NosuchDebugSprite) {
			NosuchDebug("SpriteList.add: over limit, popping cid=%s", ps->state.cid.c_str());
		}
		sprites.pop_front();
		delete ps;
	}
	s->state.visible = true;
	unlock();
}

void
SpriteList::draw(PaletteDrawer* drawer) {
	if (sprites.size() > 0) {
		// NosuchDebug("Spritelist::draw sprites.size=%d", (int)sprites.size());
	}
	for (std::list<Sprite*>::iterator i = sprites.begin(); i != sprites.end(); i++) {
		Sprite* s = *i;
		NosuchAssert(s);
		// NosuchDebug("   Spritelist::draw s=%lld  born=%d",(long long)s,s->state.born);
		s->draw(drawer);
	}
}

void
SpriteList::clear() {
	lock_write();
	for (std::list<Sprite*>::iterator i = sprites.begin(); i != sprites.end(); ) {
		Sprite* s = *i;
		NosuchAssert(s);
		if (NosuchDebugSprite) {
			NosuchDebug("SpriteList.clear: deleting Sprite cid=%s",s->state.cid.c_str());
		}
		i = sprites.erase(i);
		delete s;
	}
	unlock();
}

void
SpriteList::computeForce(std::list<Sprite*>& sprites, int gravity) {

	float gravityFactor = gravity / 5.0f;
	if (sprites.size() > 0) {
		// NosuchDebug("computerForce nsprites = %d\n", sprites.size());
	}
	for (std::list<Sprite*>::iterator i = sprites.begin(); i != sprites.end(); ) {
		Sprite* s = *i;
		NosuchAssert(s);
		// NosuchDebug("advanceTo First S=%lld pos=%f, %f\n",(long long)s, s->state.pos.x,s->state.pos.y);
		std::list<Sprite*>::iterator ni = ++i;
		float force = 0.0;
		for (; ni != sprites.end(); ni++) {
			Sprite* ns = *ni;
			float dx = ns->state.pos.x - s->state.pos.x;
			float dy = ns->state.pos.y - s->state.pos.y;
			float dist = sqrt( (dx*dx) + (dy*dy) );
			if (dist < 0.0001) {
				dist = 0.0001f; // so no divide by 0
			}
			float gf = float(gravityFactor / (dist * 1.0e7));

			// For real gravity, I think only the sign of dx and dy
			// should be used.  However, using dx and dy here
			// somehow produces a fluid behaviour that's very nice.
			// Not sure why, too lazy to investigate further at this point.

			float xforce = dx * gf;
			float yforce = dy * gf;
#if 0
			float xforce = gf;
			if (dx < 0.0) {
				xforce = -xforce;
			}
			float yforce = gf;
			if (dy < 0.0) {
				yforce = -yforce;
			}
#endif
			s->state.gravityForce.x += xforce;
			s->state.gravityForce.y += yforce;
			// NosuchDebug("  dist=%f xyforce = %f %f\n", dist, xforce, yforce);
		}
	}
	return;
}

void
SpriteList::advanceTo(int tm, int gravity) {
	lock_write();
	if (gravity > 0) {
		computeForce(sprites,gravity);
	}
	for ( std::list<Sprite*>::iterator i = sprites.begin(); i!=sprites.end(); ) {
		Sprite* s = *i;
		NosuchAssert(s);
		glm::vec2 force;
		if (gravity > 0) {
			force = s->state.gravityForce;
		}
		else {
			force = glm::vec2(0, 0);
		}
		s->advanceTo(tm,force);
		if ( s->state.killme ) {
			if (NosuchDebugSprite) {
				NosuchDebug("SpriteList.clear: killme, deleting Sprite cid=%s",s->state.cid.c_str());
			}
			i = sprites.erase(i);
			// NosuchDebug("Should be deleting Sprite s=%d",(int)s);
			delete s;
		} else {
			i++;
		}
	}
	unlock();
}

SpriteSquare::SpriteSquare() {
	noise_initialized = false;
}

void SpriteSquare::drawShape(PaletteDrawer* drawer, int xdir, int ydir) {
	float halfw = 0.125;
	float halfh = 0.125;

	if (!noise_initialized) {
		noise_0 = vertexNoise();
		noise_1 = vertexNoise();
		noise_2 = vertexNoise();
		noise_3 = vertexNoise();
		noise_initialized = true;
	}

	float x0 = -halfw + noise_0.x * halfw;
	float y0 = -halfh + noise_0.y * halfh;
	float x1 = -halfw + noise_1.x * halfw;
	float y1 = halfh + noise_1.y * halfh;
	float x2 = halfw + noise_2.x * halfw;
	float y2 = halfh + noise_2.y * halfh;
	float x3 = halfw + noise_3.x * halfw;
	float y3 = -halfh + noise_3.y * halfh;
	NosuchDebug(2,"drawing Square halfw=%.3f halfh=%.3f",halfw,halfh);
	drawer->drawQuad( params, state, x0,y0, x1,y1, x2,y2, x3, y3);
}

SpriteTriangle::SpriteTriangle() {
	noise_initialized = false;
}

glm::vec2 SpriteTriangle::rotate(glm::vec2 point, float radians, glm::vec2 about = glm::vec2(0.0f,0.0f) ) {
	float c, s;
	c = cos(radians);
	s = sin(radians);
	point -= about;
	glm::vec2 newpoint = glm::vec2{
		point[ 0 ] * c - point[ 1 ] * s,
		point[ 0 ] * s + point[ 1 ] * c
	};
	glm::vec2 finalpoint = newpoint + about;
	return finalpoint;
}

void SpriteTriangle::drawShape(PaletteDrawer* drawer, int xdir, int ydir) {

	if (!noise_initialized) {
		noise_0 = vertexNoise();
		noise_1 = vertexNoise();
		noise_2 = vertexNoise();
		noise_initialized = true;
	}
	float sz = 0.2f;
	glm::vec2 p1 = glm::vec2(0.0f,sz);
	glm::vec2 p2 = rotate(p1, Sprite::degree2radian( 120), glm::vec2(0.0,0.0));
	glm::vec2 p3 = rotate(p1, Sprite::degree2radian(-120), glm::vec2(0.0,0.0));
	
	drawer->drawTriangle(params, state, p1.x+noise_0.x*sz,p1.y+noise_0.y*sz,
			     p2.x+noise_2.x*sz,p2.y+noise_1.y*sz,
			     p3.x+noise_2.x*sz,p3.y+noise_2.y*sz);
}

SpriteLine::SpriteLine() {
	noise_initialized = false;
}

void SpriteLine::drawShape(PaletteDrawer* app, int xdir, int ydir) {
	if (!noise_initialized) {
		noise_0 = vertexNoise();
		noise_1 = vertexNoise();
		noise_initialized = true;
	}
	// NosuchDebug("SpriteLine::drawShape wh=%f %f\n",w,h);
	float halfw = 0.25f;
	float halfh = 0.25f;
	float x0 = -0.2f;
	float y0 =  0.0f;
	float x1 =  0.2f;
	float y1 =  0.0f;
	app->drawLine(params, state, x0 + noise_0.x, y0 + noise_0.y, x1 + noise_1.x, y1 + noise_1.y);
}

SpriteCircle::SpriteCircle() {
}

void SpriteCircle::drawShape(PaletteDrawer* drawer, int xdir, int ydir) {
	drawer->drawEllipse(params, state, 0, 0, 0.125, 0.0, 360.0);
}
