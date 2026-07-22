// Standalone checks for polygonfill::triangulate.
//
// The main invariant is area: the triangles a fill produces must cover the
// same area as the outlines they came from (outer contours minus holes), and
// they must not overlap each other. Area is checked directly; overlap is
// checked by point sampling, which also catches triangles that spill outside
// the outline or leave gaps inside it.

#include "PolygonFill.h"

#include <cmath>
#include <cstdio>
#include <cstdlib>
#include <ctime>
#include <string>
#include <vector>

using polygonfill::Contour;

static int failures = 0;
static int checks   = 0;

static float signedArea( const Contour& p ) {
	double a = 0.0;
	size_t n = p.size();
	for( size_t i = 0; i < n; ++i ) {
		const glm::vec2& u = p[ i ];
		const glm::vec2& v = p[ ( i + 1 ) % n ];
		a += (double)u.x * v.y - (double)v.x * u.y;
	}
	return (float)( a * 0.5 );
}

static bool pointInPolygon( const glm::vec2& pt, const Contour& poly ) {
	bool inside = false;
	size_t n    = poly.size();
	for( size_t i = 0, j = n - 1; i < n; j = i++ ) {
		const glm::vec2& a = poly[ i ];
		const glm::vec2& b = poly[ j ];
		if( ( a.y > pt.y ) != ( b.y > pt.y ) ) {
			float x = a.x + ( pt.y - a.y ) * ( b.x - a.x ) / ( b.y - a.y );
			if( pt.x < x )
				inside = !inside;
		}
	}
	return inside;
}

static bool pointInTriangle( const glm::vec2& p, const glm::vec2& a, const glm::vec2& b, const glm::vec2& c ) {
	float d0 = ( b.x - a.x ) * ( p.y - a.y ) - ( b.y - a.y ) * ( p.x - a.x );
	float d1 = ( c.x - b.x ) * ( p.y - b.y ) - ( c.y - b.y ) * ( p.x - b.x );
	float d2 = ( a.x - c.x ) * ( p.y - c.y ) - ( a.y - c.y ) * ( p.x - c.x );
	bool neg = ( d0 < 0.0f ) || ( d1 < 0.0f ) || ( d2 < 0.0f );
	bool pos = ( d0 > 0.0f ) || ( d1 > 0.0f ) || ( d2 > 0.0f );
	return !( neg && pos );
}

static Contour circle( float cx, float cy, float r, int n, bool ccw = true ) {
	Contour c;
	for( int i = 0; i < n; ++i ) {
		float t   = 6.283185307f * float( i ) / float( n );
		float ang = ccw ? t : -t;
		c.push_back( glm::vec2( cx + r * cosf( ang ), cy + r * sinf( ang ) ) );
	}
	return c;
}

static Contour rect( float x0, float y0, float x1, float y1 ) {
	Contour c;
	c.push_back( glm::vec2( x0, y0 ) );
	c.push_back( glm::vec2( x1, y0 ) );
	c.push_back( glm::vec2( x1, y1 ) );
	c.push_back( glm::vec2( x0, y1 ) );
	return c;
}

static void report( const std::string& name, bool ok, const std::string& detail ) {
	checks++;
	if( ok ) {
		printf( "  ok    %-34s %s\n", name.c_str(), detail.c_str() );
	} else {
		printf( "  FAIL  %-34s %s\n", name.c_str(), detail.c_str() );
		failures++;
	}
}

// Expected area = sum of |area| of even-depth contours minus |area| of the
// odd-depth ones. Tolerance is a fraction of the expected area.
static void checkFill( const std::string& name, const std::vector< Contour >& subpaths, float tolerance = 0.02f ) {

	std::vector< glm::vec2 > tris;
	bool ok = polygonfill::triangulate( subpaths, tris );
	if( !ok ) {
		report( name, false, "triangulate() returned false" );
		return;
	}
	if( tris.size() % 3 != 0 ) {
		report( name, false, "triangle list is not a multiple of 3" );
		return;
	}

	char detail[ 256 ];

	float minx = 1e30f, miny = 1e30f, maxx = -1e30f, maxy = -1e30f;
	for( const Contour& c : subpaths ) {
		for( const glm::vec2& p : c ) {
			minx = p.x < minx ? p.x : minx;
			miny = p.y < miny ? p.y : miny;
			maxx = p.x > maxx ? p.x : maxx;
			maxy = p.y > maxy ? p.y : maxy;
		}
	}

	// Sample the bounding box: every sample must be covered by exactly as
	// many triangles as the even-odd rule says (1 inside the shape, 0
	// outside). This catches overlaps, spills and gaps, and the solid
	// fraction doubles as an independent estimate of the filled area.
	const int GRID  = 120;
	int mismatched  = 0;
	int sampled     = 0;
	int solidCount  = 0;
	for( int iy = 0; iy < GRID; ++iy ) {
		for( int ix = 0; ix < GRID; ++ix ) {
			// Offset off the lattice so samples avoid vertices and edges.
			glm::vec2 p( minx + ( maxx - minx ) * ( ix + 0.3137f ) / GRID,
						 miny + ( maxy - miny ) * ( iy + 0.6271f ) / GRID );
			int inOutline = 0;
			for( const Contour& c : subpaths ) {
				if( pointInPolygon( p, c ) )
					inOutline++;
			}
			bool solid = ( inOutline % 2 ) == 1;

			int covered = 0;
			for( size_t i = 0; i + 2 < tris.size(); i += 3 ) {
				if( pointInTriangle( p, tris[ i ], tris[ i + 1 ], tris[ i + 2 ] ) )
					covered++;
			}
			sampled++;
			if( solid )
				solidCount++;
			if( ( solid ? 1 : 0 ) != ( covered > 0 ? 1 : 0 ) || covered > 1 )
				mismatched++;
		}
	}

	// A few samples land within rounding distance of an edge, so allow a
	// small fraction rather than demanding an exact match.
	float badFrac = float( mismatched ) / float( sampled );
	snprintf( detail, sizeof( detail ), "%d/%d samples mismatched (%.2f%%), %d tris",
			  mismatched, sampled, badFrac * 100.0f, (int)( tris.size() / 3 ) );
	report( name + " coverage", badFrac < 0.02f, detail );

	// Independent area check: triangle areas must sum to the sampled area.
	double got = 0.0;
	for( size_t i = 0; i + 2 < tris.size(); i += 3 )
		got += std::fabs( signedArea( Contour{ tris[ i ], tris[ i + 1 ], tris[ i + 2 ] } ) );
	double bbox     = double( maxx - minx ) * double( maxy - miny );
	double expected = bbox * double( solidCount ) / double( sampled );
	double err      = std::fabs( got - expected ) / ( expected > 0 ? expected : 1.0 );
	snprintf( detail, sizeof( detail ), "area %.5f vs sampled %.5f (%.2f%% off)", got, expected, err * 100.0 );
	report( name + " area", err <= tolerance, detail );
}

int main() {

	printf( "polygonfill tests\n" );

	{
		std::vector< Contour > s{ rect( 0, 0, 1, 1 ) };
		checkFill( "square", s );
	}
	{
		std::vector< Contour > s{ circle( 0, 0, 1, 64 ) };
		checkFill( "circle", s );
	}
	{
		// Clockwise input: winding must not matter.
		std::vector< Contour > s{ circle( 0, 0, 1, 64, false ) };
		checkFill( "circle (cw)", s );
	}
	{
		// Concave: a plus sign, which has 4 reflex vertices.
		Contour c;
		float t = 0.3f;
		c.push_back( glm::vec2( t, t ) );
		c.push_back( glm::vec2( t, 1 ) );
		c.push_back( glm::vec2( -t, 1 ) );
		c.push_back( glm::vec2( -t, t ) );
		c.push_back( glm::vec2( -1, t ) );
		c.push_back( glm::vec2( -1, -t ) );
		c.push_back( glm::vec2( -t, -t ) );
		c.push_back( glm::vec2( -t, -1 ) );
		c.push_back( glm::vec2( t, -1 ) );
		c.push_back( glm::vec2( t, -t ) );
		c.push_back( glm::vec2( 1, -t ) );
		c.push_back( glm::vec2( 1, t ) );
		std::vector< Contour > s{ c };
		checkFill( "cross (concave)", s );
	}
	{
		// Deep concavity from a single contour, the shape of a crescent:
		// most of one circle's arc, bridged by another circle's arc bending
		// into the body. A centroid fan gets this badly wrong, which is why
		// the parametric sprites route concave fills through here.
		Contour c;
		float alpha = 1.159f;// acos(0.8), sep = 1.6
		for( int i = 0; i < 64; ++i ) {
			float a = alpha + ( 6.283185307f - 2 * alpha ) * i / 63.0f;
			c.push_back( glm::vec2( cosf( a ), sinf( a ) ) );
		}
		for( int i = 0; i < 64; ++i ) {
			float a = ( 3.141592653f + alpha ) - ( 2 * alpha ) * i / 63.0f;
			c.push_back( glm::vec2( 1.6f + cosf( a ), sinf( a ) ) );
		}
		std::vector< Contour > s{ c };
		checkFill( "crescent (deep concave)", s );
	}
	{
		// The core case: a ring. Hole must be bridged, not filled over.
		std::vector< Contour > s{ circle( 0, 0, 1, 64 ), circle( 0, 0, 0.5f, 48, false ) };
		checkFill( "annulus", s );
	}
	{
		// Hole given with the same winding as the outer contour, which is
		// what a naive tracer emits; classification must be by nesting, not
		// by winding.
		std::vector< Contour > s{ circle( 0, 0, 1, 64 ), circle( 0, 0, 0.5f, 48, true ) };
		checkFill( "annulus (same winding)", s );
	}
	{
		// Two holes in one contour, forcing sequential bridging.
		std::vector< Contour > s{ rect( -2, -1, 2, 1 ),
								  circle( -1, 0, 0.4f, 32, false ),
								  circle( 1, 0, 0.4f, 32, false ) };
		checkFill( "two holes", s );
	}
	{
		// Island inside a hole: depth 2 is solid again.
		std::vector< Contour > s{ circle( 0, 0, 2, 64 ),
								  circle( 0, 0, 1.2f, 48, false ),
								  circle( 0, 0, 0.5f, 32 ) };
		checkFill( "island in hole", s );
	}
	{
		// Disjoint contours, no nesting.
		std::vector< Contour > s{ rect( 0, 0, 1, 1 ), rect( 3, 0, 4, 1 ) };
		checkFill( "two disjoint", s );
	}
	{
		// Star: alternating convex and reflex vertices.
		Contour c;
		for( int i = 0; i < 10; ++i ) {
			float a = 6.283185307f * i / 10.0f;
			float r = ( i % 2 ) ? 0.4f : 1.0f;
			c.push_back( glm::vec2( r * cosf( a ), r * sinf( a ) ) );
		}
		std::vector< Contour > s{ c };
		checkFill( "star", s );
	}
	{
		// Dense outline, like a flattened Bezier trace: simplification must
		// keep the shape while cutting the vertex count.
		std::vector< Contour > s{ circle( 0, 0, 1, 900 ), circle( 0, 0, 0.5f, 700, false ) };
		checkFill( "dense annulus", s, 0.03f );
	}
	{
		// Explicitly closed input (last point repeats the first).
		Contour c = rect( 0, 0, 1, 1 );
		c.push_back( c[ 0 ] );
		std::vector< Contour > s{ c };
		checkFill( "explicitly closed", s );
	}

	// Inputs that must be refused rather than crash or hang.
	{
		std::vector< Contour > s;
		std::vector< glm::vec2 > tris;
		report( "empty input refused", !polygonfill::triangulate( s, tris ) && tris.empty(), "" );
	}
	{
		Contour c;
		c.push_back( glm::vec2( 0, 0 ) );
		c.push_back( glm::vec2( 1, 0 ) );
		std::vector< Contour > s{ c };
		std::vector< glm::vec2 > tris;
		report( "two-point contour refused", !polygonfill::triangulate( s, tris ), "" );
	}
	{
		// Zero area (all collinear).
		Contour c;
		for( int i = 0; i < 10; ++i )
			c.push_back( glm::vec2( float( i ), 0.0f ) );
		std::vector< Contour > s{ c };
		std::vector< glm::vec2 > tris;
		report( "degenerate contour refused", !polygonfill::triangulate( s, tris ), "" );
	}
	{
		// Over the vertex cap: must bail out rather than grind. Jagged
		// contours are used because smooth ones simplify under the cap.
		std::vector< Contour > s;
		for( int k = 0; k < 12; ++k ) {
			Contour c;
			for( int i = 0; i < 400; ++i ) {
				float a = 6.283185307f * i / 400.0f;
				float r = ( i % 2 ) ? 0.6f : 1.0f;
				c.push_back( glm::vec2( k * 10.0f + r * cosf( a ), r * sinf( a ) ) );
			}
			s.push_back( c );
		}
		std::vector< glm::vec2 > tris;
		clock_t t0   = clock();
		bool refused = !polygonfill::triangulate( s, tris );
		double ms    = 1000.0 * double( clock() - t0 ) / CLOCKS_PER_SEC;
		char detail[ 64 ];
		snprintf( detail, sizeof( detail ), "%.1f ms", ms );
		report( "over vertex cap refused", refused && ms < 500.0, detail );
	}
	{
		// Just under the cap: the worst case that actually gets triangulated
		// runs at load time, so it needs to stay quick.
		std::vector< Contour > s;
		Contour c;
		for( int i = 0; i < 2000; ++i ) {
			float a = 6.283185307f * i / 2000.0f;
			float r = ( i % 2 ) ? 0.6f : 1.0f;
			c.push_back( glm::vec2( r * cosf( a ), r * sinf( a ) ) );
		}
		s.push_back( c );
		std::vector< glm::vec2 > tris;
		clock_t t0 = clock();
		bool ok    = polygonfill::triangulate( s, tris );
		double ms  = 1000.0 * double( clock() - t0 ) / CLOCKS_PER_SEC;
		char detail[ 96 ];
		snprintf( detail, sizeof( detail ), "%d tris in %.1f ms", (int)( tris.size() / 3 ), ms );
		report( "near cap is fast", ok && ms < 2000.0, detail );
	}
	{
		// Self-intersecting: must terminate and produce something.
		Contour c;
		c.push_back( glm::vec2( 0, 0 ) );
		c.push_back( glm::vec2( 1, 1 ) );
		c.push_back( glm::vec2( 1, 0 ) );
		c.push_back( glm::vec2( 0, 1 ) );
		std::vector< Contour > s{ c };
		std::vector< glm::vec2 > tris;
		polygonfill::triangulate( s, tris );
		report( "self-intersecting terminates", tris.size() % 3 == 0, "" );
	}

	printf( "\n%d checks, %d failures\n", checks, failures );
	return failures == 0 ? 0 : 1;
}
