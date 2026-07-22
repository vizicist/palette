#include "PaletteAll.h"
#include "PolygonFill.h"
#include "ShapeSprite.h"

#include <algorithm>
#include <cmath>

// Parametric shapes. Each one turns visual.shapesides and visual.shapedetail
// into a list of contours in sprite-local space; what those two parameters
// mean is per-shape and documented at each case in SpriteShape::build.

namespace {

const float TWO_PI = 6.2831853071795864769f;

// Sample counts. Smooth curves need enough points that the polyline reads as
// a curve, but every point is a vertex upload, so they stay modest.
const int SAMPLES_SMOOTH = 96;
const int SAMPLES_CURVE  = 512;
const int SAMPLES_RING   = 64;

float lerp( float a, float b, float t ) {
	return a + ( b - a ) * t;
}

// Catmull-Rom through four scalars, used to smooth the blob's random radii.
float catmullRom( float p0, float p1, float p2, float p3, float t ) {
	float t2 = t * t;
	float t3 = t2 * t;
	return 0.5f * ( ( 2.0f * p1 ) + ( -p0 + p2 ) * t + ( 2.0f * p0 - 5.0f * p1 + 4.0f * p2 - p3 ) * t2 + ( -p0 + 3.0f * p1 - 3.0f * p2 + p3 ) * t3 );
}

void appendArc( std::vector< glm::vec2 >& pts, glm::vec2 center, float radius, float fromRad, float toRad, int samples ) {
	for( int i = 0; i < samples; ++i ) {
		float t   = float( i ) / float( samples - 1 );
		float ang = lerp( fromRad, toRad, t );
		pts.push_back( center + glm::vec2( radius * cosf( ang ), radius * sinf( ang ) ) );
	}
}

void appendCircle( std::vector< glm::vec2 >& pts, float radius, int samples ) {
	for( int i = 0; i < samples; ++i ) {
		float ang = TWO_PI * float( i ) / float( samples );
		pts.push_back( glm::vec2( radius * cosf( ang ), radius * sinf( ang ) ) );
	}
}

}// namespace

glm::vec2 SpriteParametric::polar( float radius, float radians ) {
	return glm::vec2( radius * cosf( radians ), radius * sinf( radians ) );
}

void SpriteParametric::fitToExtent( std::vector< glm::vec2 >& pts ) {
	if( pts.empty() )
		return;
	float minx = pts[ 0 ].x, maxx = pts[ 0 ].x;
	float miny = pts[ 0 ].y, maxy = pts[ 0 ].y;
	for( const glm::vec2& p : pts ) {
		minx = std::min( minx, p.x );
		maxx = std::max( maxx, p.x );
		miny = std::min( miny, p.y );
		maxy = std::max( maxy, p.y );
	}
	float w    = maxx - minx;
	float h    = maxy - miny;
	float span = std::max( w, h );
	if( span <= 0.0f )
		return;
	float scale = ( 2.0f * SHAPE_EXTENT ) / span;
	glm::vec2 center( ( minx + maxx ) * 0.5f, ( miny + maxy ) * 0.5f );
	for( glm::vec2& p : pts )
		p = ( p - center ) * scale;
}

int SpriteParametric::sides() {
	// Shapes that need more than 1 raise the floor themselves.
	int n = params.shapesides;
	if( n < 1 )
		n = 1;
	if( n > 24 )
		n = 24;
	return n;
}

float SpriteParametric::detail() {
	float d = params.shapedetail;
	if( d < 0.0f )
		d = 0.0f;
	if( d > 1.0f )
		d = 1.0f;
	return d;
}

void SpriteParametric::buildOnce() {

	build( _contours );

	int closedCount = 0;
	for( Contour& c : _contours ) {
		// visual.noisevertex[xy] jitters the outline, the same knob the square
		// and triangle use, applied once so a sprite's shape stays stable over
		// its lifetime.
		for( glm::vec2& p : c.pts )
			p += vertexNoise() * SHAPE_EXTENT;
		// Repeat the first point so a closed contour can be stroked as a
		// LINE_STRIP without copying it every frame.
		if( c.closed && c.pts.size() >= 3 ) {
			c.pts.push_back( c.pts[ 0 ] );
			closedCount++;
		}
	}

	// Filled shapes get a real triangulation: a centroid fan spills across
	// concave outlines (crescent, chevron, pacman) and draws nested contours
	// (ring, concentric) on top of each other instead of punching holes.
	// Curve-family shapes opt out via _fanFill, where the centroid fan of a
	// self-intersecting outline is the intended look. If triangulation
	// refuses the outline, _fillTriangles stays empty and drawShape falls
	// back to fanning.
	if( params.filled && !_fanFill && closedCount > 0 ) {
		std::vector< polygonfill::Contour > closed;
		closed.reserve( closedCount );
		for( Contour& c : _contours ) {
			if( c.closed && c.pts.size() >= 4 )
				closed.push_back( c.pts );
		}
		polygonfill::triangulate( closed, _fillTriangles );
	}

	_built = true;
}

void SpriteParametric::drawShape( PaletteDrawer* app, int xdir, int ydir ) {

	if( !_built )
		buildOnce();

	bool fillFromTriangles = params.filled && !_fillTriangles.empty();
	if( fillFromTriangles ) {
		app->drawTriangles( params, state, _fillTriangles.data(), (int)_fillTriangles.size(),
							app->shapeXScale( params ) );
	}

	for( Contour& c : _contours ) {
		int count = (int)c.pts.size();
		if( count < 2 )
			continue;
		if( c.closed ) {
			if( fillFromTriangles )
				continue;// already covered by the triangulated fill
			if( params.filled && count >= 4 )
				app->drawPolyFan( params, state, c.pts.data(), count );
			else
				app->drawPolyline( params, state, c.pts.data(), count );
		} else {
			app->drawPolyline( params, state, c.pts.data(), count );
		}
	}
}

SpriteParametric* SpriteParametric::create( const std::string& shape ) {

	struct Entry {
		const char* name;
		SpriteShape::Kind kind;
	};
	static const Entry table[] = {
		{ "polygon", SpriteShape::POLYGON },
		{ "star", SpriteShape::STAR },
		{ "diamond", SpriteShape::DIAMOND },
		{ "cross", SpriteShape::CROSS },
		{ "chevron", SpriteShape::CHEVRON },
		{ "arrow", SpriteShape::ARROW },
		{ "heart", SpriteShape::HEART },
		{ "crescent", SpriteShape::CRESCENT },
		{ "squircle", SpriteShape::SQUIRCLE },
		{ "capsule", SpriteShape::CAPSULE },
		{ "teardrop", SpriteShape::TEARDROP },
		{ "gear", SpriteShape::GEAR },
		{ "pacman", SpriteShape::PACMAN },
		{ "rose", SpriteShape::ROSE },
		{ "spirograph", SpriteShape::SPIROGRAPH },
		{ "lissajous", SpriteShape::LISSAJOUS },
		{ "spiral", SpriteShape::SPIRAL },
		{ "logspiral", SpriteShape::LOGSPIRAL },
		{ "blob", SpriteShape::BLOB },
		{ "zigzag", SpriteShape::ZIGZAG },
		{ "wave", SpriteShape::WAVE },
		{ "burst", SpriteShape::BURST },
		{ "concentric", SpriteShape::CONCENTRIC },
		{ "ring", SpriteShape::RING },
		{ "arc", SpriteShape::ARC },
	};

	for( const Entry& e : table ) {
		if( shape == e.name )
			return new SpriteShape( e.kind );
	}
	return NULL;
}

void SpriteShape::build( std::vector< Contour >& out ) {

	const float R = SHAPE_EXTENT;
	int n         = sides();
	float d       = detail();

	switch( _kind ) {

	// ---- closed forms ----------------------------------------------------

	case POLYGON: {
		// sides = corner count, detail = roundness (0 polygon, 1 circle).
		int corners = std::max( 3, n );
		Contour c;
		int samples = corners * 8;
		for( int i = 0; i < samples; ++i ) {
			float t   = float( i ) / float( samples );
			float u   = t * corners;
			int k     = (int)u;
			float f   = u - k;
			glm::vec2 v0 = polar( R, TWO_PI * float( k ) / corners + TWO_PI * 0.25f );
			glm::vec2 v1 = polar( R, TWO_PI * float( k + 1 ) / corners + TWO_PI * 0.25f );
			glm::vec2 flat = v0 + ( v1 - v0 ) * f;
			glm::vec2 round = polar( R, TWO_PI * t + TWO_PI * 0.25f );
			c.pts.push_back( flat + ( round - flat ) * d );
		}
		out.push_back( c );
		break;
	}

	case STAR: {
		// sides = point count, detail = how deep the notches cut.
		int points  = std::max( 3, n );
		float inner = R * ( 0.85f - d * 0.7f );
		Contour c;
		for( int i = 0; i < points * 2; ++i ) {
			float ang = TWO_PI * float( i ) / float( points * 2 ) + TWO_PI * 0.25f;
			c.pts.push_back( polar( ( i % 2 ) ? inner : R, ang ) );
		}
		out.push_back( c );
		break;
	}

	case DIAMOND: {
		// detail = width relative to height.
		float w = R * ( 0.25f + d * 1.5f );
		Contour c;
		c.pts.push_back( glm::vec2( 0.0f, R ) );
		c.pts.push_back( glm::vec2( w, 0.0f ) );
		c.pts.push_back( glm::vec2( 0.0f, -R ) );
		c.pts.push_back( glm::vec2( -w, 0.0f ) );
		out.push_back( c );
		break;
	}

	case CROSS: {
		// detail = arm thickness.
		float t = R * ( 0.1f + d * 0.55f );
		Contour c;
		c.pts.push_back( glm::vec2( t, t ) );
		c.pts.push_back( glm::vec2( t, R ) );
		c.pts.push_back( glm::vec2( -t, R ) );
		c.pts.push_back( glm::vec2( -t, t ) );
		c.pts.push_back( glm::vec2( -R, t ) );
		c.pts.push_back( glm::vec2( -R, -t ) );
		c.pts.push_back( glm::vec2( -t, -t ) );
		c.pts.push_back( glm::vec2( -t, -R ) );
		c.pts.push_back( glm::vec2( t, -R ) );
		c.pts.push_back( glm::vec2( t, -t ) );
		c.pts.push_back( glm::vec2( R, -t ) );
		c.pts.push_back( glm::vec2( R, t ) );
		out.push_back( c );
		break;
	}

	case CHEVRON: {
		// detail = band thickness.
		float a = R;
		float b = R * 0.55f;
		float t = R * ( 0.15f + d * 0.6f );
		Contour c;
		c.pts.push_back( glm::vec2( -a, 0.0f ) );
		c.pts.push_back( glm::vec2( 0.0f, -b ) );
		c.pts.push_back( glm::vec2( a, 0.0f ) );
		c.pts.push_back( glm::vec2( a, t ) );
		c.pts.push_back( glm::vec2( 0.0f, -b + t ) );
		c.pts.push_back( glm::vec2( -a, t ) );
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case ARROW: {
		// detail = shaft and head width.
		float w  = R * ( 0.1f + d * 0.4f );
		float hw = R * ( 0.35f + d * 0.45f );
		Contour c;
		c.pts.push_back( glm::vec2( -w, -R ) );
		c.pts.push_back( glm::vec2( w, -R ) );
		c.pts.push_back( glm::vec2( w, R * 0.2f ) );
		c.pts.push_back( glm::vec2( hw, R * 0.2f ) );
		c.pts.push_back( glm::vec2( 0.0f, R ) );
		c.pts.push_back( glm::vec2( -hw, R * 0.2f ) );
		c.pts.push_back( glm::vec2( -w, R * 0.2f ) );
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case HEART: {
		// detail = vertical stretch.
		float ys = 0.6f + d * 0.8f;
		Contour c;
		for( int i = 0; i < SAMPLES_SMOOTH; ++i ) {
			float t = TWO_PI * float( i ) / float( SAMPLES_SMOOTH );
			float s = sinf( t );
			float x = 16.0f * s * s * s;
			float y = 13.0f * cosf( t ) - 5.0f * cosf( 2.0f * t ) - 2.0f * cosf( 3.0f * t ) - cosf( 4.0f * t );
			c.pts.push_back( glm::vec2( x, y * ys ) );
		}
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case CRESCENT: {
		// Two equal circles, offset by 'sep'; the crescent is the part of the
		// first that lies outside the second. detail = how thin it gets.
		float sep   = R * ( 0.3f + d * 1.5f );
		float ratio = sep / ( 2.0f * R );
		if( ratio > 0.999f )
			ratio = 0.999f;
		float alpha = acosf( ratio );
		Contour c;
		appendArc( c.pts, glm::vec2( 0.0f, 0.0f ), R, alpha, TWO_PI - alpha, SAMPLES_SMOOTH );
		appendArc( c.pts, glm::vec2( sep, 0.0f ), R, TWO_PI * 0.5f + alpha, TWO_PI * 0.5f - alpha, SAMPLES_SMOOTH );
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case SQUIRCLE: {
		// Superellipse. detail sweeps from a pinched astroid through a circle
		// to a near-square.
		float e = 0.4f + d * 9.6f;
		float p = 2.0f / e;
		Contour c;
		for( int i = 0; i < SAMPLES_SMOOTH; ++i ) {
			float t  = TWO_PI * float( i ) / float( SAMPLES_SMOOTH );
			float ct = cosf( t );
			float st = sinf( t );
			float x  = R * ( ct < 0 ? -1.0f : 1.0f ) * powf( fabsf( ct ), p );
			float y  = R * ( st < 0 ? -1.0f : 1.0f ) * powf( fabsf( st ), p );
			c.pts.push_back( glm::vec2( x, y ) );
		}
		out.push_back( c );
		break;
	}

	case CAPSULE: {
		// detail = cap radius; at 1.0 the capsule closes up into a circle.
		float r  = R * ( 0.15f + d * 0.85f );
		float dx = R - r;
		Contour c;
		appendArc( c.pts, glm::vec2( dx, 0.0f ), r, -TWO_PI * 0.25f, TWO_PI * 0.25f, SAMPLES_RING / 2 );
		appendArc( c.pts, glm::vec2( -dx, 0.0f ), r, TWO_PI * 0.25f, TWO_PI * 0.75f, SAMPLES_RING / 2 );
		out.push_back( c );
		break;
	}

	case TEARDROP: {
		// detail = how sharp the point is.
		float m = 1.0f + d * 4.0f;
		Contour c;
		for( int i = 0; i < SAMPLES_SMOOTH; ++i ) {
			float t = TWO_PI * float( i ) / float( SAMPLES_SMOOTH );
			float x = cosf( t );
			float y = sinf( t ) * powf( sinf( t * 0.5f ), m );
			c.pts.push_back( glm::vec2( x, y ) );
		}
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case GEAR: {
		// sides = teeth, detail = tooth depth.
		int teeth  = std::max( 3, n );
		float root = R * ( 0.9f - d * 0.5f );
		float step = TWO_PI / float( teeth );
		Contour c;
		for( int k = 0; k < teeth; ++k ) {
			float a0 = step * float( k );
			c.pts.push_back( polar( root, a0 ) );
			c.pts.push_back( polar( R, a0 + step * 0.25f ) );
			c.pts.push_back( polar( R, a0 + step * 0.5f ) );
			c.pts.push_back( polar( root, a0 + step * 0.75f ) );
		}
		out.push_back( c );
		break;
	}

	case PACMAN: {
		// detail = mouth angle.
		float mouth = TWO_PI * ( 0.03f + d * 0.44f );
		Contour c;
		c.pts.push_back( glm::vec2( 0.0f, 0.0f ) );
		appendArc( c.pts, glm::vec2( 0.0f, 0.0f ), R, mouth * 0.5f, TWO_PI - mouth * 0.5f, SAMPLES_RING );
		out.push_back( c );
		break;
	}

	// ---- curve families --------------------------------------------------

	case ROSE: {
		// r = cos(k*theta) with an offset that opens the center up.
		// sides = petal frequency, detail = how much of a body it keeps.
		// Petals radiate from the origin, so the centroid fan fills each
		// petal exactly; real triangulation would fight the self-touching
		// outline for no visual gain.
		_fanFill  = true;
		int k     = std::max( 1, n );
		float m   = d * 0.5f;
		Contour c;
		for( int i = 0; i < SAMPLES_CURVE; ++i ) {
			float t = TWO_PI * float( i ) / float( SAMPLES_CURVE );
			float r = R * ( m + ( 1.0f - m ) * cosf( float( k ) * t ) );
			c.pts.push_back( polar( r, t ) );
		}
		out.push_back( c );
		break;
	}

	case SPIROGRAPH: {
		// Hypotrochoid with an integer ratio so it closes after one turn.
		// sides = lobe count, detail = pen offset. Self-intersecting, so the
		// fill is a deliberate centroid fan (string-art look).
		_fanFill  = true;
		float k   = 1.0f / float( std::max( 2, n ) );
		float pen = 0.2f + d * 1.1f;
		Contour c;
		for( int i = 0; i < SAMPLES_CURVE; ++i ) {
			float t = TWO_PI * float( i ) / float( SAMPLES_CURVE );
			float x = ( 1.0f - k ) * cosf( t ) + k * pen * cosf( ( ( 1.0f - k ) / k ) * t );
			float y = ( 1.0f - k ) * sinf( t ) - k * pen * sinf( ( ( 1.0f - k ) / k ) * t );
			c.pts.push_back( glm::vec2( x, y ) );
		}
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case LISSAJOUS: {
		// sides = x frequency, detail = y frequency. Self-intersecting, so
		// the fill is a deliberate centroid fan.
		_fanFill = true;
		int a = std::max( 1, n );
		int b = 1 + (int)( d * 8.0f );
		Contour c;
		for( int i = 0; i < SAMPLES_CURVE; ++i ) {
			float t = TWO_PI * float( i ) / float( SAMPLES_CURVE );
			c.pts.push_back( glm::vec2( sinf( float( a ) * t + TWO_PI * 0.25f ), sinf( float( b ) * t ) ) );
		}
		fitToExtent( c.pts );
		out.push_back( c );
		break;
	}

	case SPIRAL: {
		// sides = arms, detail = turns. Arms are open contours.
		int arms    = std::max( 1, n );
		float turns = 1.0f + d * 4.0f;
		for( int a = 0; a < arms; ++a ) {
			Contour c;
			c.closed     = false;
			float offset = TWO_PI * float( a ) / float( arms );
			for( int i = 0; i < 128; ++i ) {
				float t = float( i ) / 127.0f;
				c.pts.push_back( polar( R * t, offset + TWO_PI * turns * t ) );
			}
			out.push_back( c );
		}
		break;
	}

	case LOGSPIRAL: {
		// Same as SPIRAL but the radius grows exponentially, so the arms
		// crowd at the center and open up fast at the rim.
		int arms    = std::max( 1, n );
		float turns = 1.0f + d * 3.0f;
		for( int a = 0; a < arms; ++a ) {
			Contour c;
			c.closed     = false;
			float offset = TWO_PI * float( a ) / float( arms );
			for( int i = 0; i < 128; ++i ) {
				float t = float( i ) / 127.0f;
				c.pts.push_back( polar( R * expf( 4.0f * ( t - 1.0f ) ), offset + TWO_PI * turns * t ) );
			}
			out.push_back( c );
		}
		break;
	}

	case BLOB: {
		// A circle with random radii, smoothed so it stays organic rather
		// than spiky. sides = lumpiness, detail = amplitude.
		int ctrl   = std::max( 3, n );
		float amp  = 0.1f + d * 0.6f;
		std::vector< float > radii( ctrl );
		for( int i = 0; i < ctrl; ++i )
			radii[ i ] = R * ( 1.0f + amp * ( RANDFLOAT * 2.0f - 1.0f ) );
		Contour c;
		int samples = std::max( SAMPLES_RING, ctrl * 8 );
		for( int i = 0; i < samples; ++i ) {
			float t = float( i ) / float( samples );
			float u = t * ctrl;
			int k   = (int)u;
			float f = u - k;
			float r = catmullRom( radii[ ( k + ctrl - 1 ) % ctrl ], radii[ k % ctrl ],
								  radii[ ( k + 1 ) % ctrl ], radii[ ( k + 2 ) % ctrl ], f );
			c.pts.push_back( polar( r, TWO_PI * t ) );
		}
		out.push_back( c );
		break;
	}

	case ZIGZAG: {
		// sides = peaks, detail = amplitude. Open contour.
		int peaks  = std::max( 2, n );
		float amp  = R * ( 0.15f + d * 0.85f );
		int steps  = peaks * 2;
		Contour c;
		c.closed = false;
		for( int i = 0; i <= steps; ++i ) {
			float x = -R + 2.0f * R * float( i ) / float( steps );
			c.pts.push_back( glm::vec2( x, ( i % 2 ) ? amp : -amp ) );
		}
		out.push_back( c );
		break;
	}

	case WAVE: {
		// sides = cycles, detail = amplitude. Open contour.
		int cycles = std::max( 1, n );
		float amp  = R * ( 0.1f + d * 0.9f );
		Contour c;
		c.closed = false;
		for( int i = 0; i < 256; ++i ) {
			float t = float( i ) / 255.0f;
			c.pts.push_back( glm::vec2( -R + 2.0f * R * t, amp * sinf( TWO_PI * float( cycles ) * t ) ) );
		}
		out.push_back( c );
		break;
	}

	case BURST: {
		// sides = spokes, detail = how far the spokes start from the center.
		int spokes  = std::max( 2, n );
		float inner = R * d * 0.8f;
		for( int i = 0; i < spokes; ++i ) {
			float ang = TWO_PI * float( i ) / float( spokes );
			Contour c;
			c.closed = false;
			c.pts.push_back( polar( inner, ang ) );
			c.pts.push_back( polar( R, ang ) );
			out.push_back( c );
		}
		break;
	}

	case CONCENTRIC: {
		// sides = ring count, detail = innermost radius; rings space evenly
		// from there out to the rim. Filled, the nesting alternates, so this
		// comes out as banded rings.
		int rings   = std::max( 2, n );
		float inner = R * ( 0.1f + d * 0.6f );
		for( int i = 0; i < rings; ++i ) {
			float t = float( i ) / float( rings - 1 );
			Contour c;
			appendCircle( c.pts, inner + ( R - inner ) * t, SAMPLES_RING );
			out.push_back( c );
		}
		break;
	}

	case RING: {
		// detail = hole size. Two nested contours, so a filled ring comes out
		// as a real annulus rather than a disc with a disc drawn on it.
		Contour outer, inner;
		appendCircle( outer.pts, R, SAMPLES_RING );
		appendCircle( inner.pts, R * ( 0.15f + d * 0.7f ), SAMPLES_RING );
		out.push_back( outer );
		out.push_back( inner );
		break;
	}

	case ARC: {
		// sides = how many arcs are spaced around the circle, detail = how
		// much of each slot the arc fills. One arc at sides=1 is a plain arc;
		// more turn it into a dashed ring.
		int arcs   = std::max( 1, n );
		float slot = TWO_PI / float( arcs );
		float span = slot * ( 0.15f + d * 0.8f );
		for( int i = 0; i < arcs; ++i ) {
			float mid = slot * float( i );
			Contour c;
			c.closed = false;
			appendArc( c.pts, glm::vec2( 0.0f, 0.0f ), R, mid - span * 0.5f, mid + span * 0.5f, 32 );
			out.push_back( c );
		}
		break;
	}
	}
}
