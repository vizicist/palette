#include "PolygonFill.h"

#include <algorithm>
#include <cmath>

namespace polygonfill {

namespace {

const float DUP_EPSILON = 1e-9f;
// Relative to the overall bounding box, so the caller's units don't matter.
const float SIMPLIFY_FRACTION = 0.0016f;
const float MIN_AREA_FRACTION = 1e-7f;

float signedArea( const Contour& p ) {
	double a = 0.0;
	size_t n = p.size();
	for( size_t i = 0; i < n; ++i ) {
		const glm::vec2& u = p[ i ];
		const glm::vec2& v = p[ ( i + 1 ) % n ];
		a += (double)u.x * v.y - (double)v.x * u.y;
	}
	return (float)( a * 0.5 );
}

bool pointInPolygon( const glm::vec2& pt, const Contour& poly ) {
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

float cross2( const glm::vec2& a, const glm::vec2& b, const glm::vec2& c ) {
	return ( b.x - a.x ) * ( c.y - a.y ) - ( b.y - a.y ) * ( c.x - a.x );
}

bool pointInTriangle( const glm::vec2& p, const glm::vec2& a, const glm::vec2& b, const glm::vec2& c ) {
	float d0 = cross2( a, b, p );
	float d1 = cross2( b, c, p );
	float d2 = cross2( c, a, p );
	bool neg = ( d0 < 0.0f ) || ( d1 < 0.0f ) || ( d2 < 0.0f );
	bool pos = ( d0 > 0.0f ) || ( d1 > 0.0f ) || ( d2 > 0.0f );
	return !( neg && pos );
}

// Drop repeated points, the duplicated closing point, and vertices that sit on
// the line between their neighbors. Flattened Beziers carry far more points
// than a fill needs, and every one of them costs O(n) in the ear clipper.
void simplifyContour( Contour& c, float tol ) {
	Contour out;
	out.reserve( c.size() );
	for( size_t i = 0; i < c.size(); ++i ) {
		if( !out.empty() && glm::length( c[ i ] - out.back() ) < DUP_EPSILON )
			continue;
		out.push_back( c[ i ] );
	}
	while( out.size() > 3 && glm::length( out.front() - out.back() ) < DUP_EPSILON )
		out.pop_back();
	if( out.size() < 5 || tol <= 0.0f ) {
		c.swap( out );
		return;
	}

	// Each pass drops points that sit within 'tol' of the line between their
	// current neighbors, never two in a row, so a point is only ever judged
	// against nearby geometry. Repeating converges on an outline whose
	// deviation from the original stays around tol; judging against the last
	// surviving point instead would let the reference drift arbitrarily far
	// back and flatten a dense circle down to a few vertices.
	for( int pass = 0; pass < 16; ++pass ) {
		size_t n = out.size();
		if( n < 5 )
			break;
		Contour kept;
		kept.reserve( n );
		bool prevRemoved = false;
		for( size_t i = 0; i < n; ++i ) {
			const glm::vec2& prev = out[ ( i + n - 1 ) % n ];
			const glm::vec2& cur  = out[ i ];
			const glm::vec2& next = out[ ( i + 1 ) % n ];
			glm::vec2 e           = next - prev;
			float len             = glm::length( e );
			float dist;
			if( len < DUP_EPSILON ) {
				dist = glm::length( cur - prev );
			} else {
				glm::vec2 d = cur - prev;
				dist        = std::fabs( d.x * e.y - d.y * e.x ) / len;
			}
			// Never simplify below a triangle.
			bool drop = !prevRemoved && dist < tol && ( kept.size() + ( n - i - 1 ) ) >= 3;
			if( drop ) {
				prevRemoved = true;
				continue;
			}
			kept.push_back( cur );
			prevRemoved = false;
		}
		if( kept.size() < 3 || kept.size() == n )
			break;
		out.swap( kept );
	}
	c.swap( out );
}

// A vertex with minimum x is always convex, so the centroid of the triangle it
// forms with its neighbors lies inside the polygon except in degenerate cases,
// which the containment check catches.
bool interiorPoint( const Contour& poly, glm::vec2& out ) {
	size_t n = poly.size();
	if( n < 3 )
		return false;
	size_t k = 0;
	for( size_t i = 1; i < n; ++i ) {
		if( poly[ i ].x < poly[ k ].x )
			k = i;
	}
	glm::vec2 c = ( poly[ ( k + n - 1 ) % n ] + poly[ k ] + poly[ ( k + 1 ) % n ] ) / 3.0f;
	if( pointInPolygon( c, poly ) ) {
		out = c;
		return true;
	}
	for( size_t i = 0; i < n; ++i ) {
		glm::vec2 m = ( poly[ i ] + poly[ ( i + 2 ) % n ] ) * 0.5f;
		if( pointInPolygon( m, poly ) ) {
			out = m;
			return true;
		}
	}
	return false;
}

void fanTriangulate( const Contour& poly, const std::vector< int >& idx, std::vector< glm::vec2 >& out ) {
	for( size_t i = 1; i + 1 < idx.size(); ++i ) {
		out.push_back( poly[ idx[ 0 ] ] );
		out.push_back( poly[ idx[ i ] ] );
		out.push_back( poly[ idx[ i + 1 ] ] );
	}
}

void earClip( Contour poly, std::vector< glm::vec2 >& out ) {
	if( poly.size() < 3 )
		return;
	if( signedArea( poly ) < 0.0f )
		std::reverse( poly.begin(), poly.end() );

	std::vector< int > idx( poly.size() );
	for( size_t i = 0; i < poly.size(); ++i )
		idx[ i ] = (int)i;

	int i     = 0;
	int fails = 0;
	while( idx.size() > 3 ) {
		int n              = (int)idx.size();
		int ia             = idx[ ( i + n - 1 ) % n ];
		int ib             = idx[ i ];
		int ic             = idx[ ( i + 1 ) % n ];
		const glm::vec2& A = poly[ ia ];
		const glm::vec2& B = poly[ ib ];
		const glm::vec2& C = poly[ ic ];

		bool isEar = cross2( A, B, C ) > 0.0f;
		for( int j = 0; isEar && j < n; ++j ) {
			int ij = idx[ j ];
			if( ij == ia || ij == ib || ij == ic )
				continue;
			// Bridging a hole leaves two vertices duplicated at each end of
			// the bridge. Comparing by index isn't enough: a duplicate sits
			// exactly on the candidate ear's corner and would veto every ear,
			// collapsing the whole contour into the fallback fan.
			const glm::vec2& P = poly[ ij ];
			if( glm::length( P - A ) < DUP_EPSILON || glm::length( P - B ) < DUP_EPSILON || glm::length( P - C ) < DUP_EPSILON )
				continue;
			if( pointInTriangle( P, A, B, C ) )
				isEar = false;
		}

		if( isEar ) {
			out.push_back( A );
			out.push_back( B );
			out.push_back( C );
			idx.erase( idx.begin() + i );
			i     = idx.empty() ? 0 : i % (int)idx.size();
			fails = 0;
		} else {
			i = ( i + 1 ) % n;
			// A full lap without finding an ear means the outline is
			// self-intersecting or otherwise not clippable; a fan still
			// covers roughly the right area, which beats drawing nothing.
			if( ++fails > n ) {
				fanTriangulate( poly, idx, out );
				return;
			}
		}
	}
	fanTriangulate( poly, idx, out );
}

// Eberly's visible-vertex search: cast a ray in +x from the hole's rightmost
// vertex M and pick a vertex of 'outer' that M can be joined to without
// crossing an edge. Returns -1 if no bridge could be found.
int findBridgeVertex( const Contour& outer, const glm::vec2& M ) {
	size_t n    = outer.size();
	float bestX = 0.0f;
	int edgeA   = -1;
	int edgeB   = -1;
	bool found  = false;
	for( size_t i = 0; i < n; ++i ) {
		const glm::vec2& a = outer[ i ];
		const glm::vec2& b = outer[ ( i + 1 ) % n ];
		if( ( a.y > M.y ) == ( b.y > M.y ) )
			continue;
		float t = ( M.y - a.y ) / ( b.y - a.y );
		float x = a.x + t * ( b.x - a.x );
		if( x <= M.x )
			continue;
		if( !found || x < bestX ) {
			bestX = x;
			edgeA = (int)i;
			edgeB = (int)( ( i + 1 ) % n );
			found = true;
		}
	}
	if( !found )
		return -1;

	glm::vec2 I( bestX, M.y );
	int iP = ( outer[ edgeA ].x > outer[ edgeB ].x ) ? edgeA : edgeB;

	// Any reflex vertex inside triangle (M,I,P) blocks the direct join; the
	// one closest to the +x ray is reachable instead.
	int best      = iP;
	float bestCos = -2.0f;
	float bestLen = 0.0f;
	for( size_t i = 0; i < n; ++i ) {
		if( (int)i == iP )
			continue;
		const glm::vec2& R    = outer[ i ];
		const glm::vec2& prev = outer[ ( i + n - 1 ) % n ];
		const glm::vec2& next = outer[ ( i + 1 ) % n ];
		if( cross2( prev, R, next ) > 0.0f )
			continue;// convex, can't block
		if( !pointInTriangle( R, M, I, outer[ iP ] ) )
			continue;
		glm::vec2 d = R - M;
		float len   = glm::length( d );
		if( len < DUP_EPSILON )
			continue;
		float cosang = d.x / len;
		if( cosang > bestCos || ( cosang == bestCos && len < bestLen ) ) {
			bestCos = cosang;
			bestLen = len;
			best    = (int)i;
		}
	}
	return best;
}

// Splice 'hole' into 'outer' along a bridge, producing one contour that walks
// in, around the hole, and back out.
void bridgeHole( Contour& outer, const Contour& hole ) {
	if( hole.size() < 3 || outer.size() < 3 )
		return;
	size_t iM = 0;
	for( size_t i = 1; i < hole.size(); ++i ) {
		if( hole[ i ].x > hole[ iM ].x )
			iM = i;
	}
	int iP = findBridgeVertex( outer, hole[ iM ] );
	if( iP < 0 )
		return;

	Contour merged;
	merged.reserve( outer.size() + hole.size() + 2 );
	for( int i = 0; i <= iP; ++i )
		merged.push_back( outer[ i ] );
	for( size_t k = 0; k < hole.size(); ++k )
		merged.push_back( hole[ ( iM + k ) % hole.size() ] );
	merged.push_back( hole[ iM ] );
	merged.push_back( outer[ iP ] );
	for( size_t i = (size_t)iP + 1; i < outer.size(); ++i )
		merged.push_back( outer[ i ] );
	outer.swap( merged );
}

}// namespace

bool triangulate( const std::vector< Contour >& subpaths, std::vector< glm::vec2 >& outTriangles ) {

	outTriangles.clear();
	if( subpaths.empty() )
		return false;

	float minx = 1e30f, miny = 1e30f, maxx = -1e30f, maxy = -1e30f;
	for( const Contour& c : subpaths ) {
		for( const glm::vec2& p : c ) {
			minx = std::min( minx, p.x );
			miny = std::min( miny, p.y );
			maxx = std::max( maxx, p.x );
			maxy = std::max( maxy, p.y );
		}
	}
	float span = std::max( maxx - minx, maxy - miny );
	if( span <= 0.0f )
		return false;

	std::vector< Contour > polys;
	for( const Contour& raw : subpaths ) {
		Contour c = raw;
		simplifyContour( c, span * SIMPLIFY_FRACTION );
		if( c.size() < 3 )
			continue;
		if( std::fabs( signedArea( c ) ) < span * span * MIN_AREA_FRACTION )
			continue;
		polys.push_back( c );
	}
	if( polys.empty() )
		return false;

	size_t total = 0;
	for( const Contour& c : polys )
		total += c.size();
	if( total > (size_t)MAX_FILL_VERTICES )
		return false;

	std::vector< glm::vec2 > reps( polys.size() );
	std::vector< bool > hasRep( polys.size(), false );
	for( size_t i = 0; i < polys.size(); ++i )
		hasRep[ i ] = interiorPoint( polys[ i ], reps[ i ] );

	// Even nesting depth is solid, odd is a hole.
	std::vector< int > depth( polys.size(), 0 );
	for( size_t i = 0; i < polys.size(); ++i ) {
		if( !hasRep[ i ] )
			continue;
		for( size_t j = 0; j < polys.size(); ++j ) {
			if( i != j && pointInPolygon( reps[ i ], polys[ j ] ) )
				depth[ i ]++;
		}
	}

	// Each hole belongs to the smallest solid contour containing it.
	std::vector< std::vector< int > > holesOf( polys.size() );
	for( size_t i = 0; i < polys.size(); ++i ) {
		if( !hasRep[ i ] || depth[ i ] % 2 == 0 )
			continue;
		int parent     = -1;
		float bestArea = 0.0f;
		for( size_t j = 0; j < polys.size(); ++j ) {
			if( i == j || depth[ j ] % 2 != 0 )
				continue;
			if( !pointInPolygon( reps[ i ], polys[ j ] ) )
				continue;
			float area = std::fabs( signedArea( polys[ j ] ) );
			if( parent < 0 || area < bestArea ) {
				parent   = (int)j;
				bestArea = area;
			}
		}
		if( parent >= 0 )
			holesOf[ parent ].push_back( (int)i );
	}

	for( size_t i = 0; i < polys.size(); ++i ) {
		if( depth[ i ] % 2 != 0 )
			continue;

		Contour outer = polys[ i ];
		if( signedArea( outer ) < 0.0f )
			std::reverse( outer.begin(), outer.end() );

		// Bridge the rightmost hole first, so an earlier bridge never sits
		// between a later hole and the outer contour.
		std::vector< int > holes = holesOf[ i ];
		std::sort( holes.begin(), holes.end(), [ &polys ]( int a, int b ) {
			float ax = -1e30f, bx = -1e30f;
			for( const glm::vec2& p : polys[ a ] )
				ax = std::max( ax, p.x );
			for( const glm::vec2& p : polys[ b ] )
				bx = std::max( bx, p.x );
			return ax > bx;
		} );
		for( int h : holes ) {
			Contour hole = polys[ h ];
			if( signedArea( hole ) > 0.0f )
				std::reverse( hole.begin(), hole.end() );
			bridgeHole( outer, hole );
		}
		earClip( outer, outTriangles );
	}

	return !outTriangles.empty();
}

}// namespace polygonfill
