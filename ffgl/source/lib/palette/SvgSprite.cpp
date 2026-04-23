#include "PaletteAll.h"
#include "SvgSprite.h"

#include <cctype>
#include <cmath>
#include <cstdlib>
#include <fstream>
#include <sstream>

// Minimal SVG-path loader for shapes produced by potrace (and similar
// simple tools). Parses the <svg viewBox>, the outer <g transform>, and
// the "d" attribute of every <path>, then flattens cubic/quadratic
// Beziers into line segments. The result is normalized into the unit
// square used by the other Sprite subclasses and drawn via
// PaletteDrawer::drawLine.

std::map< std::string, ParsedSvg > SpriteSVG::_cache;

// Target radius in sprite-local space. The built-in SpriteCircle draws a
// 0.125-radius ellipse (Sprite.cpp:546) and SpriteSquare uses 0.125 half-
// width; match that so sigils render at the same nominal size.
static const float SIGIL_HALF_EXTENT = 0.125f;

static const int BEZIER_STEPS = 16;

namespace {

bool extractAttribute( const std::string& tag, const std::string& name, std::string& out ) {
	std::string key = name + "=\"";
	size_t p      = tag.find( key );
	if( p == std::string::npos )
		return false;
	p += key.size();
	size_t q = tag.find( '"', p );
	if( q == std::string::npos )
		return false;
	out = tag.substr( p, q - p );
	return true;
}

bool findElement( const std::string& doc, const std::string& elemName, size_t fromPos, size_t& tagStart, size_t& tagEnd ) {
	std::string marker = "<" + elemName;
	size_t p           = doc.find( marker, fromPos );
	if( p == std::string::npos )
		return false;
	char next = doc[ p + marker.size() ];
	if( next != ' ' && next != '\t' && next != '\n' && next != '\r' && next != '>' && next != '/' )
		return false;
	size_t q = doc.find( '>', p );
	if( q == std::string::npos )
		return false;
	tagStart = p;
	tagEnd   = q + 1;
	return true;
}

struct PathCursor {
	const std::string& s;
	size_t i;
	PathCursor( const std::string& str ) : s( str ), i( 0 ) {}

	void skipSep() {
		while( i < s.size() ) {
			char c = s[ i ];
			if( c == ' ' || c == ',' || c == '\t' || c == '\n' || c == '\r' )
				++i;
			else
				break;
		}
	}

	bool atEnd() {
		skipSep();
		return i >= s.size();
	}

	bool peekCommand() {
		skipSep();
		if( i >= s.size() )
			return false;
		char c = s[ i ];
		return std::isalpha( (unsigned char)c ) != 0;
	}

	char readCommand() {
		skipSep();
		return s[ i++ ];
	}

	bool readNumber( float& out ) {
		skipSep();
		if( i >= s.size() )
			return false;
		size_t start = i;
		char c       = s[ i ];
		if( c == '+' || c == '-' )
			++i;
		bool anyDigit = false;
		while( i < s.size() && std::isdigit( (unsigned char)s[ i ] ) ) {
			++i;
			anyDigit = true;
		}
		if( i < s.size() && s[ i ] == '.' ) {
			++i;
			while( i < s.size() && std::isdigit( (unsigned char)s[ i ] ) ) {
				++i;
				anyDigit = true;
			}
		}
		if( i < s.size() && ( s[ i ] == 'e' || s[ i ] == 'E' ) ) {
			++i;
			if( i < s.size() && ( s[ i ] == '+' || s[ i ] == '-' ) )
				++i;
			while( i < s.size() && std::isdigit( (unsigned char)s[ i ] ) )
				++i;
		}
		if( !anyDigit )
			return false;
		out = std::strtof( s.c_str() + start, nullptr );
		return true;
	}
};

void flattenCubic( const glm::vec2& p0, const glm::vec2& p1, const glm::vec2& p2, const glm::vec2& p3, std::vector< glm::vec2 >& out ) {
	for( int k = 1; k <= BEZIER_STEPS; ++k ) {
		float t  = float( k ) / float( BEZIER_STEPS );
		float u  = 1.0f - t;
		float b0 = u * u * u;
		float b1 = 3.0f * u * u * t;
		float b2 = 3.0f * u * t * t;
		float b3 = t * t * t;
		out.push_back( p0 * b0 + p1 * b1 + p2 * b2 + p3 * b3 );
	}
}

void flattenQuadratic( const glm::vec2& p0, const glm::vec2& p1, const glm::vec2& p2, std::vector< glm::vec2 >& out ) {
	for( int k = 1; k <= BEZIER_STEPS; ++k ) {
		float t  = float( k ) / float( BEZIER_STEPS );
		float u  = 1.0f - t;
		float b0 = u * u;
		float b1 = 2.0f * u * t;
		float b2 = t * t;
		out.push_back( p0 * b0 + p1 * b1 + p2 * b2 );
	}
}

void parsePathData( const std::string& d, std::vector< std::vector< glm::vec2 > >& subpaths ) {
	PathCursor c( d );
	glm::vec2 cur( 0.0f, 0.0f );
	glm::vec2 start( 0.0f, 0.0f );
	glm::vec2 lastCubicCtl( 0.0f, 0.0f );
	glm::vec2 lastQuadCtl( 0.0f, 0.0f );
	bool haveLastCubic  = false;
	bool haveLastQuad   = false;
	char cmd            = 0;
	std::vector< glm::vec2 >* path = nullptr;

	auto beginSub = []( std::vector< std::vector< glm::vec2 > >& subs, const glm::vec2& p ) -> std::vector< glm::vec2 >* {
		subs.push_back( std::vector< glm::vec2 >() );
		subs.back().push_back( p );
		return &subs.back();
	};

	while( !c.atEnd() ) {
		if( c.peekCommand() )
			cmd = c.readCommand();

		bool rel = std::islower( (unsigned char)cmd ) != 0;
		char up  = std::toupper( (unsigned char)cmd );

		if( up == 'M' ) {
			float x, y;
			if( !c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 p = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			cur         = p;
			start       = p;
			path        = beginSub( subpaths, p );
			haveLastCubic = haveLastQuad = false;
			// Subsequent coord pairs after M become implicit L/l.
			cmd = rel ? 'l' : 'L';
			while( c.peekCommand() == false && !c.atEnd() ) {
				float nx, ny;
				if( !c.readNumber( nx ) )
					break;
				if( !c.readNumber( ny ) )
					break;
				glm::vec2 np = rel ? cur + glm::vec2( nx, ny ) : glm::vec2( nx, ny );
				if( path )
					path->push_back( np );
				cur = np;
			}
			haveLastCubic = haveLastQuad = false;
		}
		else if( up == 'L' ) {
			float x, y;
			if( !c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 p = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			if( path )
				path->push_back( p );
			cur           = p;
			haveLastCubic = haveLastQuad = false;
		}
		else if( up == 'H' ) {
			float x;
			if( !c.readNumber( x ) )
				break;
			glm::vec2 p = rel ? glm::vec2( cur.x + x, cur.y ) : glm::vec2( x, cur.y );
			if( path )
				path->push_back( p );
			cur           = p;
			haveLastCubic = haveLastQuad = false;
		}
		else if( up == 'V' ) {
			float y;
			if( !c.readNumber( y ) )
				break;
			glm::vec2 p = rel ? glm::vec2( cur.x, cur.y + y ) : glm::vec2( cur.x, y );
			if( path )
				path->push_back( p );
			cur           = p;
			haveLastCubic = haveLastQuad = false;
		}
		else if( up == 'C' ) {
			float x1, y1, x2, y2, x, y;
			if( !c.readNumber( x1 ) || !c.readNumber( y1 ) ||
				!c.readNumber( x2 ) || !c.readNumber( y2 ) ||
				!c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 c1 = rel ? cur + glm::vec2( x1, y1 ) : glm::vec2( x1, y1 );
			glm::vec2 c2 = rel ? cur + glm::vec2( x2, y2 ) : glm::vec2( x2, y2 );
			glm::vec2 p  = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			if( path )
				flattenCubic( cur, c1, c2, p, *path );
			cur           = p;
			lastCubicCtl  = c2;
			haveLastCubic = true;
			haveLastQuad  = false;
		}
		else if( up == 'S' ) {
			float x2, y2, x, y;
			if( !c.readNumber( x2 ) || !c.readNumber( y2 ) ||
				!c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 c1 = haveLastCubic ? ( cur + ( cur - lastCubicCtl ) ) : cur;
			glm::vec2 c2 = rel ? cur + glm::vec2( x2, y2 ) : glm::vec2( x2, y2 );
			glm::vec2 p  = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			if( path )
				flattenCubic( cur, c1, c2, p, *path );
			cur           = p;
			lastCubicCtl  = c2;
			haveLastCubic = true;
			haveLastQuad  = false;
		}
		else if( up == 'Q' ) {
			float x1, y1, x, y;
			if( !c.readNumber( x1 ) || !c.readNumber( y1 ) ||
				!c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 q = rel ? cur + glm::vec2( x1, y1 ) : glm::vec2( x1, y1 );
			glm::vec2 p = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			if( path )
				flattenQuadratic( cur, q, p, *path );
			cur           = p;
			lastQuadCtl   = q;
			haveLastQuad  = true;
			haveLastCubic = false;
		}
		else if( up == 'T' ) {
			float x, y;
			if( !c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 q = haveLastQuad ? ( cur + ( cur - lastQuadCtl ) ) : cur;
			glm::vec2 p = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			if( path )
				flattenQuadratic( cur, q, p, *path );
			cur           = p;
			lastQuadCtl   = q;
			haveLastQuad  = true;
			haveLastCubic = false;
		}
		else if( up == 'Z' ) {
			if( path && !path->empty() )
				path->push_back( start );
			cur           = start;
			path          = nullptr;
			haveLastCubic = haveLastQuad = false;
		}
		else {
			// Unknown command — bail out on this path rather than loop forever.
			break;
		}
	}
}

void parseTransform( const std::string& expr, glm::vec2& translate, glm::vec2& scale ) {
	translate = glm::vec2( 0.0f, 0.0f );
	scale     = glm::vec2( 1.0f, 1.0f );

	size_t p = expr.find( "translate(" );
	if( p != std::string::npos ) {
		p += std::string( "translate(" ).size();
		size_t q = expr.find( ')', p );
		if( q != std::string::npos ) {
			std::string args = expr.substr( p, q - p );
			PathCursor c( args );
			float tx = 0, ty = 0;
			c.readNumber( tx );
			c.readNumber( ty );
			translate = glm::vec2( tx, ty );
		}
	}

	p = expr.find( "scale(" );
	if( p != std::string::npos ) {
		p += std::string( "scale(" ).size();
		size_t q = expr.find( ')', p );
		if( q != std::string::npos ) {
			std::string args = expr.substr( p, q - p );
			PathCursor c( args );
			float sx = 1, sy = 1;
			c.readNumber( sx );
			if( !c.readNumber( sy ) )
				sy = sx;
			scale = glm::vec2( sx, sy );
		}
	}
}

}// namespace

bool SpriteSVG::parseFile( const std::string& path, ParsedSvg& out ) {
	std::ifstream f( path.c_str() );
	if( !f.good() )
		return false;
	std::stringstream ss;
	ss << f.rdbuf();
	std::string doc = ss.str();

	// Grab <svg> viewBox (or width/height) for normalization.
	size_t svgStart = 0, svgEnd = 0;
	if( !findElement( doc, "svg", 0, svgStart, svgEnd ) )
		return false;
	std::string svgTag = doc.substr( svgStart, svgEnd - svgStart );

	float vbMinX = 0, vbMinY = 0, vbW = 0, vbH = 0;
	std::string vb;
	if( extractAttribute( svgTag, "viewBox", vb ) ) {
		PathCursor c( vb );
		c.readNumber( vbMinX );
		c.readNumber( vbMinY );
		c.readNumber( vbW );
		c.readNumber( vbH );
	}
	if( vbW <= 0 || vbH <= 0 ) {
		std::string wStr, hStr;
		if( extractAttribute( svgTag, "width", wStr ) )
			vbW = (float)std::atof( wStr.c_str() );
		if( extractAttribute( svgTag, "height", hStr ) )
			vbH = (float)std::atof( hStr.c_str() );
	}
	if( vbW <= 0 || vbH <= 0 )
		return false;

	// Outer <g transform="..."> (potrace always emits one).
	glm::vec2 gTrans( 0.0f, 0.0f );
	glm::vec2 gScale( 1.0f, 1.0f );
	size_t gStart = 0, gEnd = 0;
	if( findElement( doc, "g", svgEnd, gStart, gEnd ) ) {
		std::string gTag = doc.substr( gStart, gEnd - gStart );
		std::string tr;
		if( extractAttribute( gTag, "transform", tr ) )
			parseTransform( tr, gTrans, gScale );
	}

	// Accumulate all <path d="..."> into raw subpaths (in SVG user space,
	// post outer group transform).
	std::vector< std::vector< glm::vec2 > > raw;
	size_t cursor = gEnd;
	while( true ) {
		size_t pStart = 0, pEnd = 0;
		if( !findElement( doc, "path", cursor, pStart, pEnd ) )
			break;
		cursor           = pEnd;
		std::string pTag = doc.substr( pStart, pEnd - pStart );
		std::string d;
		if( !extractAttribute( pTag, "d", d ) )
			continue;
		std::vector< std::vector< glm::vec2 > > subs;
		parsePathData( d, subs );
		for( auto& s : subs ) {
			for( auto& pt : s )
				pt = gTrans + pt * gScale;
			raw.push_back( std::move( s ) );
		}
	}

	if( raw.empty() )
		return false;

	// Compute overall bbox, then normalize into [-HALF, +HALF] preserving
	// aspect ratio (centered).
	float minX = 1e30f, minY = 1e30f, maxX = -1e30f, maxY = -1e30f;
	for( auto& s : raw ) {
		for( auto& p : s ) {
			if( p.x < minX ) minX = p.x;
			if( p.y < minY ) minY = p.y;
			if( p.x > maxX ) maxX = p.x;
			if( p.y > maxY ) maxY = p.y;
		}
	}
	float bw = maxX - minX;
	float bh = maxY - minY;
	if( bw <= 0 || bh <= 0 )
		return false;
	float scale   = ( 2.0f * SIGIL_HALF_EXTENT ) / std::max( bw, bh );
	float offsetX = ( minX + maxX ) * 0.5f;
	float offsetY = ( minY + maxY ) * 0.5f;

	// SVG user space is y-down; Sprite/GL space is y-up. Flip y during
	// normalization so the sigil renders right-side-up.
	out.subpaths.clear();
	for( auto& s : raw ) {
		std::vector< glm::vec2 > norm;
		norm.reserve( s.size() );
		for( auto& p : s ) {
			glm::vec2 q = ( p - glm::vec2( offsetX, offsetY ) ) * scale;
			q.y         = -q.y;
			norm.push_back( q );
		}
		out.subpaths.push_back( std::move( norm ) );
	}
	return true;
}

SpriteSVG::SpriteSVG( const ParsedSvg* data ) : _data( data ) {
}

SpriteSVG* SpriteSVG::tryLoad( const std::string& shapeName ) {
	auto it = _cache.find( shapeName );
	if( it == _cache.end() ) {
		std::string file = PaletteDataPath() + "\\shapes\\" + shapeName + ".svg";
		ParsedSvg parsed;
		if( !parseFile( file, parsed ) )
			return nullptr;
		auto inserted = _cache.emplace( shapeName, std::move( parsed ) );
		it            = inserted.first;
	}
	return new SpriteSVG( &it->second );
}

void SpriteSVG::drawShape( PaletteDrawer* app, int xdir, int ydir ) {
	if( !_data )
		return;
	for( const auto& sub : _data->subpaths ) {
		if( sub.size() < 2 )
			continue;
		app->drawPolyline( params, state, sub.data(), (int)sub.size() );
	}
}
