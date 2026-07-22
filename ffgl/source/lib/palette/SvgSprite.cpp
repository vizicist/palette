#include "PaletteAll.h"
#include "PolygonFill.h"
#include "SvgSprite.h"

#include <algorithm>
#include <cctype>
#include <cmath>
#include <cstdlib>
#include <fstream>
#include <sstream>

#ifndef _WIN32
#include <sys/stat.h>
#endif

// Minimal SVG-path loader for shapes produced by potrace, icon sets, and
// similar simple sources. Parses the <svg viewBox>, the outer
// <g transform> (translate/scale only), and the "d" attribute of every
// <path> element; other elements (<rect>, <circle>, <line>, <polygon>)
// and per-path transforms are NOT handled. Path commands supported are
// M L H V C S Q T A Z, with curves and arcs flattened into line
// segments. The result is normalized into the unit square used by the
// other Sprite subclasses.

std::map< std::string, SpriteSVG::CacheEntry > SpriteSVG::_cache;
std::vector< std::unique_ptr< ParsedSvg > > SpriteSVG::_retired;

// How often tryLoad is willing to stat a shape file to see if it changed.
// Below this interval a cached shape is returned with no filesystem access
// at all, so editing an SVG shows up within about a second without adding
// syscalls to the per-sprite path.
static const int SVG_RECHECK_MS = 1000;

// Target radius in sprite-local space. The built-in SpriteCircle draws a
// 0.125-radius ellipse (Sprite.cpp:546) and SpriteSquare uses 0.125 half-
// width; match that so sigils render at the same nominal size.
static const float SIGIL_HALF_EXTENT = 0.125f;
static const float SIGIL_BOUNDS_PADDING = 0.10f;

static const int BEZIER_STEPS = 16;

namespace {

bool extractAttribute( const std::string& tag, const std::string& name, std::string& out ) {
	size_t p = 0;
	while( ( p = tag.find( name, p ) ) != std::string::npos ) {
		bool attrStart = p == 0 || std::isspace( (unsigned char)tag[ p - 1 ] ) || tag[ p - 1 ] == '<' || tag[ p - 1 ] == '/';
		size_t q       = p + name.size();
		while( q < tag.size() && std::isspace( (unsigned char)tag[ q ] ) )
			++q;
		if( attrStart && q < tag.size() && tag[ q ] == '=' ) {
			++q;
			while( q < tag.size() && std::isspace( (unsigned char)tag[ q ] ) )
				++q;
			if( q >= tag.size() || ( tag[ q ] != '"' && tag[ q ] != '\'' ) )
				return false;
			char quote   = tag[ q++ ];
			size_t value = q;
			size_t end   = tag.find( quote, value );
			if( end == std::string::npos )
				return false;
			out = tag.substr( value, end - value );
			return true;
		}
		p += name.size();
	}
	return false;
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

	// Arc flags are a single 0/1 digit and may be run together with the
	// following number ("a1 1 0 011 0" = flags 0,1 then x=1), so they can't
	// go through readNumber.
	bool readFlag( bool& out ) {
		skipSep();
		if( i >= s.size() )
			return false;
		char c = s[ i ];
		if( c != '0' && c != '1' )
			return false;
		++i;
		out = ( c == '1' );
		return true;
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

// Elliptical arc, using the W3C endpoint-to-center conversion (SVG spec
// appendix B.2.4), then sampled at a rate proportional to the swept angle.
void flattenArc( const glm::vec2& p0, float rx, float ry, float xrotDeg, bool largeArc, bool sweep, const glm::vec2& p1, std::vector< glm::vec2 >& out ) {

	rx = std::fabs( rx );
	ry = std::fabs( ry );
	// Degenerate radii mean a straight line, per the spec.
	if( rx < 1e-6f || ry < 1e-6f || ( p0.x == p1.x && p0.y == p1.y ) ) {
		out.push_back( p1 );
		return;
	}

	float phi  = xrotDeg * (float)M_PI / 180.0f;
	float cphi = cosf( phi );
	float sphi = sinf( phi );

	// Midpoint form in the ellipse's rotated frame.
	glm::vec2 d( ( p0.x - p1.x ) * 0.5f, ( p0.y - p1.y ) * 0.5f );
	glm::vec2 pp( cphi * d.x + sphi * d.y, -sphi * d.x + cphi * d.y );

	// Scale radii up if the endpoints are too far apart for them.
	float lam = ( pp.x * pp.x ) / ( rx * rx ) + ( pp.y * pp.y ) / ( ry * ry );
	if( lam > 1.0f ) {
		float s = sqrtf( lam );
		rx *= s;
		ry *= s;
	}

	float rx2 = rx * rx, ry2 = ry * ry;
	float num = rx2 * ry2 - rx2 * pp.y * pp.y - ry2 * pp.x * pp.x;
	float den = rx2 * pp.y * pp.y + ry2 * pp.x * pp.x;
	float rad = ( den > 0.0f && num > 0.0f ) ? num / den : 0.0f;
	float coef = sqrtf( rad );
	if( largeArc == sweep )
		coef = -coef;
	glm::vec2 cp( coef * rx * pp.y / ry, -coef * ry * pp.x / rx );
	glm::vec2 center( cphi * cp.x - sphi * cp.y + ( p0.x + p1.x ) * 0.5f,
					  sphi * cp.x + cphi * cp.y + ( p0.y + p1.y ) * 0.5f );

	// Signed angle from u to v.
	auto ang = []( float ux, float uy, float vx, float vy ) {
		float dot = ux * vx + uy * vy;
		float len = sqrtf( ( ux * ux + uy * uy ) * ( vx * vx + vy * vy ) );
		float cosv = len > 0.0f ? dot / len : 1.0f;
		if( cosv > 1.0f ) cosv = 1.0f;
		if( cosv < -1.0f ) cosv = -1.0f;
		float a = acosf( cosv );
		return ( ux * vy - uy * vx < 0.0f ) ? -a : a;
	};

	float ux     = ( pp.x - cp.x ) / rx;
	float uy     = ( pp.y - cp.y ) / ry;
	float theta1 = ang( 1.0f, 0.0f, ux, uy );
	float dtheta = ang( ux, uy, ( -pp.x - cp.x ) / rx, ( -pp.y - cp.y ) / ry );
	float twoPi  = 2.0f * (float)M_PI;
	if( !sweep && dtheta > 0.0f )
		dtheta -= twoPi;
	else if( sweep && dtheta < 0.0f )
		dtheta += twoPi;

	// A full circle gets 2*BEZIER_STEPS samples, shorter arcs fewer.
	int steps = (int)ceilf( std::fabs( dtheta ) / twoPi * ( 2.0f * BEZIER_STEPS ) );
	if( steps < 2 )
		steps = 2;
	for( int k = 1; k <= steps; ++k ) {
		float t = theta1 + dtheta * float( k ) / float( steps );
		float x = rx * cosf( t );
		float y = ry * sinf( t );
		out.push_back( glm::vec2( center.x + cphi * x - sphi * y,
								  center.y + sphi * x + cphi * y ) );
	}
	// Land exactly on the endpoint regardless of rounding.
	out.back() = p1;
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
		else if( up == 'A' ) {
			float rx, ry, xrot, x, y;
			bool laf, sf;
			if( !c.readNumber( rx ) || !c.readNumber( ry ) || !c.readNumber( xrot ) )
				break;
			if( !c.readFlag( laf ) || !c.readFlag( sf ) )
				break;
			if( !c.readNumber( x ) || !c.readNumber( y ) )
				break;
			glm::vec2 p = rel ? cur + glm::vec2( x, y ) : glm::vec2( x, y );
			if( path )
				flattenArc( cur, rx, ry, xrot, laf, sf, p, *path );
			cur           = p;
			haveLastCubic = haveLastQuad = false;
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

	float padX = bw * SIGIL_BOUNDS_PADDING;
	float padY = bh * SIGIL_BOUNDS_PADDING;
	minX -= padX;
	maxX += padX;
	minY -= padY;
	maxY += padY;
	bw = maxX - minX;
	bh = maxY - minY;

	float scale   = ( 2.0f * SIGIL_HALF_EXTENT ) / std::max( bw, bh );
	float offsetX = ( minX + maxX ) * 0.5f;
	float offsetY = ( minY + maxY ) * 0.5f;

	// SVG user space is y-down; Sprite/GL space is y-up. Flip y during
	// normalization so the sigil renders right-side-up.
	out.subpaths.clear();
	out.lineSegments.clear();
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
	for( const auto& sub : out.subpaths ) {
		for( size_t i = 1; i < sub.size(); ++i ) {
			out.lineSegments.push_back( sub[ i - 1 ] );
			out.lineSegments.push_back( sub[ i ] );
		}
	}
	// Triangles for visual.filled. Dense or degenerate outlines come back
	// empty, and drawShape falls back to the outline for those.
	if( !polygonfill::triangulate( out.subpaths, out.triangles ) ) {
		NosuchDebug( "SpriteSVG: %s cannot be filled, will draw its outline instead", path.c_str() );
	}
	return true;
}

SpriteSVG::SpriteSVG( const ParsedSvg* data ) : _data( data ) {
}

static std::string shapeFilePath( const std::string& shapeName ) {
	return PaletteDataPath() + "\\shapes\\" + shapeName + ".svg";
}

// Last-write time, or 0 if the file can't be examined. The Windows branch
// has 100ns resolution; plain stat's whole seconds could miss two quick
// saves landing inside the same second.
static long long shapeFileMtime( const std::string& path ) {
#ifdef _WIN32
	WIN32_FILE_ATTRIBUTE_DATA fa;
	if( !GetFileAttributesExA( path.c_str(), GetFileExInfoStandard, &fa ) )
		return 0;
	ULARGE_INTEGER u;
	u.LowPart  = fa.ftLastWriteTime.dwLowDateTime;
	u.HighPart = fa.ftLastWriteTime.dwHighDateTime;
	return (long long)u.QuadPart;
#else
	struct stat st;
	if( stat( path.c_str(), &st ) != 0 )
		return 0;
	return (long long)st.st_mtime;
#endif
}

// Note on threading: like the original cache, this assumes tryLoad is only
// called from the sprite-instantiation path; there is no lock here.
SpriteSVG* SpriteSVG::tryLoad( const std::string& shapeName ) {

	int now = Palette::now;

	auto it = _cache.find( shapeName );
	if( it != _cache.end() ) {
		CacheEntry& e = it->second;
		if( now - e.lastCheckMs >= SVG_RECHECK_MS ) {
			e.lastCheckMs = now;
			std::string file = shapeFilePath( shapeName );
			long long m      = shapeFileMtime( file );
			if( m != 0 && m != e.mtime ) {
				std::unique_ptr< ParsedSvg > fresh( new ParsedSvg );
				if( parseFile( file, *fresh ) ) {
					// Sprites created before this edit still point at the
					// old ParsedSvg; park it rather than freeing it.
					_retired.push_back( std::move( e.parsed ) );
					e.parsed = std::move( fresh );
					e.mtime  = m;
					NosuchDebug( "SpriteSVG: reloaded %s", file.c_str() );
				}
				// On parse failure (often a file caught mid-save) keep
				// serving the old shape; leaving e.mtime stale makes the
				// next interval retry until the file parses again.
			}
		}
		return new SpriteSVG( e.parsed.get() );
	}

	std::string file = shapeFilePath( shapeName );
	std::unique_ptr< ParsedSvg > parsed( new ParsedSvg );
	if( !parseFile( file, *parsed ) )
		return nullptr;

	CacheEntry e;
	e.parsed      = std::move( parsed );
	e.mtime       = shapeFileMtime( file );
	e.lastCheckMs = now;
	auto inserted = _cache.emplace( shapeName, std::move( e ) );
	return new SpriteSVG( inserted.first->second.parsed.get() );
}

void SpriteSVG::drawShape( PaletteDrawer* app, int xdir, int ydir ) {
	if( !_data )
		return;
	// triangles is empty when the outline was too complex to triangulate, in
	// which case a filled sprite falls back to its outline.
	if( params.filled && !_data->triangles.empty() ) {
		app->drawTriangles( params, state, _data->triangles.data(), (int)_data->triangles.size() );
		return;
	}
	if( !_data->lineSegments.empty() )
		app->drawLineSegments( params, state, _data->lineSegments.data(), (int)_data->lineSegments.size() );
}
