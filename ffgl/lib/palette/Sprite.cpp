#include <cstdlib> // for srand, rand

#include "NosuchUtil.h"
#include "PaletteAll.h"

// std::vector<std::string> Sprite::spriteShapes;

bool Sprite::initialized = false;
long nsprites = 0;
int Sprite::NextSeq = 0;

#define RANDDOUBLE (((double)rand())/RAND_MAX)

double Sprite::vertexNoise()
{
	if ( params.noisevertex > 0.0f ) {
		return params.noisevertex * RANDDOUBLE * ((rand()%2)==0?1:-1);
	} else {
		return 0.0f;
	}
}

void Sprite::initialize() {
	if ( initialized )
		return;
	initialized = true;
}
	
Sprite::Sprite() {
}

void
Sprite::initState(std::string cid, std::string cidsource, NosuchVector& pos, double movedir, double depth, double rotanginit) {

	nsprites++;
	Palette::lastsprite = Palette::now;

	// most of the state has been initialized in SpriteState constructor
	state.pos = pos;
	state.direction = movedir;
	state.depth = depth;
	state.cid = cid;
	state.cidsource = cidsource;

	state.born = Palette::now;
	state.last_tm = Palette::now;
	state.hue = params.hueinitial;
	state.huefill = params.huefillinitial;
	state.alpha = params.alphainitial;
	state.size = params.sizeinitial;
	state.seq = NextSeq++;
	state.rotdir = rotangdirOf(params.rotangdir);
	state.rotanginit = rotanginit;
	state.rotangsofar = state.rotanginit;
}

Sprite::~Sprite() {
	NosuchDebug(1,"Sprite destructor! s=%d cid=%s/%s",this,state.cid.c_str(),state.cidsource.c_str());
}

double Sprite::degree2radian(double deg) {
	return 2.0f * (double)M_PI * deg / 360.0f;
}

void Sprite::draw(PaletteHost* ph) {
	if (state.depth < params.zmin ) {
		state.depth = params.zmin;
	}
	double scaled_z = scale_z(ph,state.depth);
	draw(ph,scaled_z);
}

#if 0
void Sprite::draw(PaletteHost* app) {
	draw(app,1.0);
}
#endif

void Sprite::draw(PaletteHost* app, double scaled_z) {

	if ( ! state.visible ) {
		NosuchDebug("Sprite.draw NOT DRAWING, !visible");
		return;
	}
	// double hue = state.hueoffset + params.hueinitial;
	// double huefill = state.hueoffset + params.huefill;
	
	NosuchColor color = NosuchColor(state.hue, params.luminance, params.saturation);
	NosuchColor colorfill = NosuchColor(state.huefill, params.luminance, params.saturation);
	
	if ( state.alpha <= 0.0f || state.size <= 0.0 ) {
		state.killme = true;
		return;
	}
	
	if ( params.filled ) {
		app->fill(colorfill, state.alpha);
	} else {
		app->noFill();
	}
	app->stroke(color, state.alpha);
	if ( state.size < 0.001f ) {
		state.killme = true;
		return;
	}
	double thickness = params.thickness;
	app->strokeWeight(thickness);
	double aspect = params.aspect;
	
	// double scaled_z = region->scale_z(state.depth);

	double scalex = state.size * scaled_z;
	double scaley = state.size * scaled_z;
	
	scalex *= aspect;
	// scaley *= (1.0f/aspect);
	
	// double w = app->width * scalex;
	// double h = app->height * scaley;
	
	// if (w < 0 || h < 0) {
	// 	NosuchDebug("Hey, wh < 0?  w=%f h=%f\n",w,h);
	// }

	double x;
	double y;
	// NOTE!  The x,y coming in here is scaled to ((0,0),(1,1))
	//        while the x,y computed and given to the drawAt method
	//        is scaled to ((-1,-1),(1,1))
	int xdir;
	int ydir;
	if ( params.mirrortype == "four" ) {
		x = 2.0f * state.pos.x * app->width - 1.0f;
		y = 2.0f * state.pos.y * app->height - 1.0f;
		xdir = 1;
		ydir = 1;
		drawAt(app,x,y,scalex,scaley,xdir,ydir);
		ydir = -1;
		drawAt(app,x,-y,scalex,scaley,xdir,ydir);
		xdir = -1;
		drawAt(app,-x,y,scalex,scaley,xdir,ydir);
		ydir = 1;
		drawAt(app,-x,-y,scalex,scaley,xdir,ydir);
	} else if ( params.mirrortype == "vertical" ) {
		x = 2.0f * state.pos.x * app->width - 1.0f;
		y = state.pos.y * app->height;
		xdir = 1;
		ydir = 1;
		drawAt(app,x,y,scalex,scaley,xdir,ydir);
		// y = (1.0f-state.pos.y) * app->height;
		y = (-state.pos.y) * app->height;
		ydir = -1;
		drawAt(app,x,y,scalex,scaley,xdir,ydir);
	} else if ( params.mirrortype == "horizontal" ) {
		x = state.pos.x * app->width;
		y = 2.0f * state.pos.y * app->height - 1.0f;
		xdir = 1;
		ydir = 1;
		drawAt(app,x,y,scalex,scaley,xdir,ydir);
		// x = (1.0f-state.pos.x) * app->width;
		x = (-state.pos.x) * app->width;
		xdir = -1;
		drawAt(app,x,y,scalex,scaley,xdir,ydir);
	} else {
		x = 2.0f * state.pos.x * app->width - 1.0f;
		y = 2.0f * state.pos.y * app->height - 1.0f;
		xdir = 1;
		ydir = 1;
		drawAt(app,x,y,scalex,scaley,xdir,ydir);
	}
}
	
void Sprite::drawAt(PaletteHost* app, double x,double y, double scalex, double scaley, int xdir, int ydir) {
	app->pushMatrix();
	double dx = x;
	double dy = y;

	// handle justification
	std::string j = params.justification;
	// NosuchDebug("Sprite::drawAt s=%lld drawAt j=%s xy= %.4lf %.4lf width=%lf size=%lf depth=%lf\n",
	// 	(long long)this,j.c_str(),x,y,width(),state.size,state.depth);
	if (j == "center") {
		// do nothing
	} else if ( j == "left" ) {
		dx += width()/2.0f;
	} else if ( j == "right" ) {
		dx -= width()/2.0f;
	} else if ( j == "top" ) {
		dy += height()/2.0f;
	} else if ( j == "bottom" ) {
		dy -= height()/2.0f;
	} else if ( j == "topleft" ) {
		dx += width()/2.0f;
		dy += height()/2.0f;
	} else if ( j == "topright" ) {
		dx -= width()/2.0f;
		dy += height()/2.0f;
	} else if ( j == "bottomleft" ) {
		dx += width()/2.0f;
		dy -= height()/2.0f;
	} else if ( j == "bottomright" ) {
		dx -= width()/2.0f;
		dy -= height()/2.0f;
	} else {
		NosuchDebug("Sprite::drawAt: Unknown justification value - %s\n", params.justification.c_str());
	}

	// NosuchDebug("    Sprite::drawAt left width=%f dx is now %f\n", width(), dx);

	app->translate(dx,dy);

	if ( fixedScale() ) {
		app->scale(1.0,1.0);
	} else {
		app->scale(scalex,scaley);
	}
	// double degrees = params.rotanginit + state.rotangsofar;
	double degrees = state.rotanginit + state.rotangsofar;

	// NosuchDebug("SpriteDraw seq=%d anginit=%.3f sofar=%.3f degrees=%f",
	// 	state.seq,params.rotanginit,state.rotangsofar,degrees);
	// NosuchDebug("Sprite::drawAt degrees=%.4f  w,h=%f,%f\n",degrees,w,h);
	// NosuchDebug("Sprite::drawAt s=%d degrees=%.4f",(int)this,degrees);
	app->rotate(degrees);
	drawShape(app,xdir,ydir);
	app->popMatrix();
}

NosuchVector Sprite::deltaInDirection(double dt, double dir, double speed) {
	NosuchVector delta( (double) cos(degree2radian(dir)), (double) sin(degree2radian(dir)));
	delta = delta.normalize();
	speed /= 2.0;	// slow things down
	delta = delta.mult((dt / 1000.0f) * speed);
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

double
envelopeValue(double initial, double final, double duration, double born, double now) {
	double dt = now - born;
	double dur = duration * 1000.0;
	if ( dt >= dur )
		return final;
	if ( dt <= 0 )
		return initial;
	return initial + (final-initial) * ((now-born)/(dur));
}

void Sprite::advanceTo(int now, NosuchVector force) {

	// _params->advanceTo(tm);
	state.alpha = envelopeValue(params.alphainitial,params.alphafinal,params.alphatime,state.born,now);
	state.size = envelopeValue(params.sizeinitial,params.sizefinal,params.sizetime,state.born,now);
	
	int dnow = (now - state.born);
	// NosuchDebug("Sprite::advanceTo this=%lld now=%d born=%d dnow=%d alpha=%f size=%f last_tm=%d",(long long)this,now,state.born,dnow,state.alpha,state.size,state.last_tm);
	if (params.lifetime >= 0.0 && ((now - state.born) > (1000.0 * params.lifetime))) {
		// NosuchDebug("Lifetime of Sprite %lld exceeded, setting killme",(long long)this);
		state.killme = true;
	}
	double dt = (double)(now - state.last_tm);
	state.last_tm = now;
	
	if ( ! state.visible ) {
		return;
	}
	
	state.hue = envelopeValue(params.hueinitial,params.huefinal,params.huetime,state.born,now);
	state.huefill = envelopeValue(params.huefillinitial,params.huefillfinal,params.huefilltime,state.born,now);

	// state.hueoffset = fmod((state.hueoffset + params.cyclehue), 360.0);

	if ( state.stationary ) {
		NosuchDebug(2,"Sprite %d is stationary",this);
		return;
	}

	if ( params.rotangspeed != 0.0 ) {
		state.rotangsofar = fmod((state.rotangsofar + state.rotdir * (dt/1000.0) * params.rotangspeed) , 360.0);
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
		
		double dir = state.direction;
		
		NosuchVector delta = deltaInDirection(dt,dir,params.speed);
		
		NosuchVector npos = state.pos.add(delta);
		// NosuchDebug("sprite advance dt=%f dir=%f speed=%f delta=%f,%f npos=%f,%f",
		// 	dt, dir, params.speed, delta.x, delta.y, npos.x, npos.y);
		if ( params.bounce ) { 
			
			if ( npos.x > 1.0f ) {
				dir = fmod(( dir + 180 ) , 360);
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos.add(delta);
			}
			if ( npos.x < 0.0f ) {
				dir = fmod(( dir + 180 ) , 360);
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos.add(delta);
			}
			if ( npos.y > 1.0f ) {
				dir = fmod(( dir + 180 ) , 360);
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos.add(delta);
			}
			if ( npos.y < 0.0f ) {
				dir = fmod(( dir + 180 ) , 360);
				delta = deltaInDirection(dt,dir,params.speed);
				npos = state.pos.add(delta);
			}
state.direction = dir;
		}
 else {
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
		NosuchDebug("SpriteList.add: over limit, popping cid=%s", ps->state.cid.c_str());
		sprites.pop_front();
		delete ps;
	}
	s->state.visible = true;
	unlock();
}

void
SpriteList::draw(PaletteHost* b) {
	lock_read();
	if (sprites.size() > 0) {
		// NosuchDebug("Spritelist::draw sprites.size=%d", (int)sprites.size());
	}
	for (std::list<Sprite*>::iterator i = sprites.begin(); i != sprites.end(); i++) {
		Sprite* s = *i;
		NosuchAssert(s);
		// NosuchDebug("   Spritelist::draw s=%lld  born=%d",(long long)s,s->state.born);
		s->draw(b);
	}
	unlock();
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

	double gravityFactor = gravity / 5.0;
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
			double dx = ns->state.pos.x - s->state.pos.x;
			double dy = ns->state.pos.y - s->state.pos.y;
			double dist = sqrt( (dx*dx) + (dy*dy) );
			if (dist < 0.0001) {
				dist = 0.0001; // so no divide by 0
			}
			double gf = gravityFactor / (dist * 1.0e7);

			// For real gravity, I think only the sign of dx and dy
			// should be used.  However, using dx and dy here
			// somehow produces a fluid behaviour that's very nice.
			// Not sure why, too lazy to investigate further at this point.

			double xforce = dx * gf;
			double yforce = dy * gf;
#if 0
			double xforce = gf;
			if (dx < 0.0) {
				xforce = -xforce;
			}
			double yforce = gf;
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
		NosuchVector force;
		if (gravity > 0) {
			force = s->state.gravityForce;
		}
		else {
			force = NosuchVector(0, 0);
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

void SpriteSquare::drawShape(PaletteHost* app, int xdir, int ydir) {
	double halfw = 0.25f;
	double halfh = 0.25f;

	if (!noise_initialized) {
		noise_x0 = vertexNoise();
		noise_y0 = vertexNoise();
		noise_x1 = vertexNoise();
		noise_y1 = vertexNoise();
		noise_x2 = vertexNoise();
		noise_y2 = vertexNoise();
		noise_x3 = vertexNoise();
		noise_y3 = vertexNoise();
		noise_initialized = true;
	}

	double x0 = - halfw + noise_x0 * halfw;
	double y0 = - halfh + noise_y0 * halfh;
	double x1 = -halfw + noise_x1 * halfw;
	double y1 = halfh + noise_y1 * halfh;
	double x2 = halfw + noise_x2 * halfw;
	double y2 = halfh + noise_y2 * halfh;
	double x3 = halfw + noise_x3 * halfw;
	double y3 = -halfh + noise_y3 * halfh;
	NosuchDebug(2,"drawing Square halfw=%.3f halfh=%.3f",halfw,halfh);
	app->quad( x0,y0, x1,y1, x2,y2, x3, y3);
}

SpriteTriangle::SpriteTriangle() {
	noise_initialized = false;
}
	
void SpriteTriangle::drawShape(PaletteHost* app, int xdir, int ydir) {

	if (!noise_initialized) {
		noise_x0 = vertexNoise();
		noise_y0 = vertexNoise();
		noise_x1 = vertexNoise();
		noise_y1 = vertexNoise();
		noise_x2 = vertexNoise();
		noise_y2 = vertexNoise();
		noise_initialized = true;
	}
	// double halfw = w / 2.0f;
	// double halfh = h / 2.0f;
	double sz = 0.2f;
	NosuchVector p1 = NosuchVector(sz,0.0f);
	NosuchVector p2 = p1;
	p2 = p2.rotate(Sprite::degree2radian(120));
	NosuchVector p3 = p1;
	p3 = p3.rotate(Sprite::degree2radian(-120));
	
	app->triangle(p1.x+noise_x0*sz,p1.y+noise_y0*sz,
			     p2.x+noise_x1*sz,p2.y+noise_y1*sz,
			     p3.x+noise_x2*sz,p3.y+noise_y2*sz);
}

SpriteLine::SpriteLine() {
	noise_initialized = false;
}

void SpriteLine::drawShape(PaletteHost* app, int xdir, int ydir) {
	if (!noise_initialized) {
		noise_x0 = vertexNoise();
		noise_y0 = vertexNoise();
		noise_x1 = vertexNoise();
		noise_y1 = vertexNoise();
		noise_initialized = true;
	}
	// NosuchDebug("SpriteLine::drawShape wh=%f %f\n",w,h);
	double halfw = 0.2f;
	double halfh = 0.2f;
	double x0 = -0.2f;
	double y0 =  0.0f;
	double x1 =  0.2f;
	double y1 =  0.0f;
	app->line(x0 + noise_x0, y0 + noise_y0, x1 + noise_x1, y1 + noise_y1);
}

SpriteCircle::SpriteCircle() {
}

void SpriteCircle::drawShape(PaletteHost* app, int xdir, int ydir) {
	// NosuchDebug("SpriteCircle drawing");
	app->ellipse(0, 0, 0.2f, 0.2f);
}

SpriteArc::SpriteArc() {
}

void SpriteArc::drawShape(PaletteHost* app, int xdir, int ydir) {
	// NosuchDebug("SpriteCircle drawing");
	app->ellipse(0, 0, 0.2f, 0.2f, 0.0, 180.0);
}

static void
normalize(NosuchVector* v)
{
	v->x = (v->x * 2.0) - 1.0;
	v->y = (v->y * 2.0) - 1.0;
}