#ifndef _SVG_SPRITE_H
#define _SVG_SPRITE_H

#include <map>
#include <memory>
#include <string>
#include <vector>

#include "glm/glm.hpp"
#include "Sprite.h"

class PaletteDrawer;

struct ParsedSvg {
	std::vector< std::vector< glm::vec2 > > subpaths;
	// Consecutive point pairs ready for one batched GL_LINES draw.
	std::vector< glm::vec2 > lineSegments;
	// Point triples ready for one batched GL_TRIANGLES draw, used when
	// visual.filled is on. Empty if the outline was too complex to fill,
	// in which case the outline is drawn instead.
	std::vector< glm::vec2 > triangles;
};

class SpriteSVG : public Sprite {

public:
	static SpriteSVG* tryLoad( const std::string& shapeName );

	void drawShape( PaletteDrawer* app, int xdir, int ydir ) override;

private:
	SpriteSVG( const ParsedSvg* data );

	const ParsedSvg* _data;

	// Cache of parsed files, with enough bookkeeping to notice edits: a
	// freshness check runs at most once per SVG_RECHECK_MS per shape, so
	// the steady-state cost of tryLoad stays one integer compare. ParsedSvg
	// lives behind a unique_ptr so its address survives map operations -
	// live sprites keep raw pointers to it for seconds after creation.
	struct CacheEntry {
		std::unique_ptr< ParsedSvg > parsed;
		long long mtime;    // file mtime at last successful parse
		int lastCheckMs;    // Palette::now at last freshness check
	};

	static std::map< std::string, CacheEntry > _cache;
	// Replaced entries are parked here until shutdown rather than freed:
	// sprites created before an edit still draw from the old data. Bounded
	// by edits-per-session, so effectively nothing.
	static std::vector< std::unique_ptr< ParsedSvg > > _retired;

	static bool parseFile( const std::string& path, ParsedSvg& out );
};

#endif
