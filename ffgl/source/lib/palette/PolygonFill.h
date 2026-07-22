#ifndef _POLYGON_FILL_H
#define _POLYGON_FILL_H

#include <vector>

#include "glm/glm.hpp"

// Turning a set of closed outlines into fillable triangles.
//
// Outlines that come from tracing artwork nest: a traced letter "O" is an
// outer ring plus a hole. Subpaths are classified by nesting depth (even is
// solid, odd is a hole), every hole is bridged into the contour that contains
// it, and the resulting simple polygons are ear-clipped.
//
// This deliberately depends on nothing but glm and the standard library, so it
// can be exercised on its own.

namespace polygonfill {

typedef std::vector< glm::vec2 > Contour;

// Ear clipping is O(n^2), so very dense outlines are refused rather than
// stalling whoever asked for them.
const int MAX_FILL_VERTICES = 3000;

// Fills 'outTriangles' with point triples. Returns false (and leaves the
// output empty) if the input had nothing fillable in it or was too dense.
bool triangulate( const std::vector< Contour >& subpaths, std::vector< glm::vec2 >& outTriangles );

}// namespace polygonfill

#endif
