/*{
    "DESCRIPTION": "Empty - does nothing ",
    "CATEGORIES": [
        "Geometry Adjustment"
    ],
    "CREDIT": "by Tim Thompson",
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
        }
    ],
    "ISFVSN": "2"
}
*/

void main() {
    gl_FragColor = IMG_THIS_PIXEL(inputImage);
}
