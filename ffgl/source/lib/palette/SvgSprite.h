#ifndef _SVG_SPRITE_H
#define _SVG_SPRITE_H

#include <map>
#include <string>
#include <vector>

#include "glm/glm.hpp"
#include "Sprite.h"

class PaletteDrawer;

struct ParsedSvg {
	std::vector< std::vector< glm::vec2 > > subpaths;
};

class SpriteSVG : public Sprite {

public:
	static SpriteSVG* tryLoad( const std::string& shapeName );

	void drawShape( PaletteDrawer* app, int xdir, int ydir ) override;

private:
	SpriteSVG( const ParsedSvg* data );

	const ParsedSvg* _data;

	static std::map< std::string, ParsedSvg > _cache;
	static bool parseFile( const std::string& path, ParsedSvg& out );
};

#endif
