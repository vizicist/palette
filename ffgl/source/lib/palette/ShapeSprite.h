#ifndef _SHAPE_SPRITE_H
#define _SHAPE_SPRITE_H

#include <string>
#include <vector>

#include "glm/glm.hpp"
#include "Sprite.h"

class PaletteDrawer;

// Half-width of a sprite's local space, matching SpriteSquare's half-width and
// SpriteCircle's radius so every shape reads at the same nominal size.
#define SHAPE_EXTENT 0.125f

// Sprites whose geometry is a point list generated from visual.shapesides and
// visual.shapedetail. Subclasses only implement build(); the point list is
// computed once per sprite and reused for every frame and mirror pass.
class SpriteParametric : public Sprite {

public:
	// One outline. Closed contours join their last point back to the first and
	// honor visual.filled; open ones are always stroked.
	struct Contour {
		std::vector< glm::vec2 > pts;
		bool closed;
		Contour() : closed( true ) {}
	};

	SpriteParametric() : _built( false ), _fanFill( false ) {}

	// Returns NULL if 'shape' is not one of the parametric shape names.
	static SpriteParametric* create( const std::string& shape );

	void drawShape( PaletteDrawer* app, int xdir, int ydir ) override;

protected:
	virtual void build( std::vector< Contour >& out ) = 0;

	int sides();   // visual.shapesides, clamped to something drawable
	float detail();// visual.shapedetail, clamped to 0..1

	// build() sets this for shapes whose fill should be a centroid fan
	// rather than a real triangulation - the self-intersecting curve
	// families, where fanning from the center is the intended look.
	// Everything else is triangulated so concave outlines (crescent,
	// chevron, pacman) fill correctly.
	bool _fanFill;

	static glm::vec2 polar( float radius, float radians );
	// Center a contour and scale it so its longer axis spans the sprite
	// extent. Used by shapes whose natural size swings widely with their
	// parameters, so they stay comparable to a circle or square.
	static void fitToExtent( std::vector< glm::vec2 >& pts );

private:
	void buildOnce();

	std::vector< Contour > _contours;
	// Triangulated fill for the closed contours, computed once when the
	// sprite is filled and not _fanFill. Handles concave outlines and nested
	// contours (ring, concentric) that a centroid fan would get wrong. Empty
	// means fall back to fanning each contour.
	std::vector< glm::vec2 > _fillTriangles;
	bool _built;
};

// The built-in library of parametric shapes. New shapes can be added here, or
// as a separate SpriteParametric subclass when they need their own state.
class SpriteShape : public SpriteParametric {

public:
	enum Kind {
		POLYGON, STAR, DIAMOND, CROSS, CHEVRON, ARROW, HEART,
		CRESCENT, SQUIRCLE, CAPSULE, TEARDROP, GEAR, PACMAN,
		ROSE, SPIROGRAPH, LISSAJOUS, SPIRAL, LOGSPIRAL, BLOB,
		ZIGZAG, WAVE, BURST, CONCENTRIC, RING, ARC
	};

	SpriteShape( Kind kind ) : _kind( kind ) {}

protected:
	void build( std::vector< Contour >& out ) override;

private:
	Kind _kind;
};

#endif
