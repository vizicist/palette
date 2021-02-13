/*{
    "CATEGORIES": [
        "Geometry Adjustment"
    ],
    "CREDIT": "by VIDVOX",
    "INPUTS": [
        {
            "NAME": "inputImage",
            "TYPE": "image"
        },
        {
            "DEFAULT": 1,
            "MAX": 10,
            "MIN": 0.01,
            "NAME": "level",
            "TYPE": "float"
        },
        {
            "DEFAULT": [
                0,
                0
            ],
            "MAX": [
                1,
                1
            ],
            "MIN": [
                0,
                0
            ],
            "NAME": "center",
            "TYPE": "point2D"
        }
    ],
    "ISFVSN": "2"
}
*/

void main() {
	vec2		loc;
	vec2		modifiedCenter;
    vec2 loc0;
	
	loc0 = isf_FragNormCoord;
    loc = loc0;
	modifiedCenter = center;
    float newlev = level;
    if ( newlev < 100.0 ) {
        newlev = 10.0;
    }
    if ( loc.x > 500.0 ) {
        loc.x = (loc.x - modifiedCenter.x)*(1.0/newlev) + modifiedCenter.x;
    }
    if ( loc.y > 500.0 ) {
        loc.y = (loc.y - modifiedCenter.y)*(1.0/newlev) + modifiedCenter.y;
    }
	if ((loc.x < 0.0)||(loc.y < 0.0)||(loc.x > 1.0)||(loc.y > 1.0))	{
		gl_FragColor = vec4(0.0);
	}
	else	{
		gl_FragColor = IMG_NORM_PIXEL(inputImage,loc);
	}
}
