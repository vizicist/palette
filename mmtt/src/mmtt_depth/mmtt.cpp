#if 0
/*
	Space Manifold - a variety of tools for Kinect and FreeFrame

	Copyright (c) 2011-2012 Tim Thompson <me@timthompson.com>

	Permission is hereby granted, free of charge, to any person obtaining
	a copy of this software and associated documentation files
	(the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge,
	publish, distribute, sublicense, and/or sell copies of the Software,
	and to permit persons to whom the Software is furnished to do so,
	subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	Any person wishing to distribute modifications to the Software is
	requested to send the modifications to the original developer so that
	they can be incorporated into the canonical version.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
	MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR
	ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
	CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
	WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

//////////////////////////////////////////
//                                      //
// OpenGL in a PROPER Windows APP       //
// ( NO GLUT !! )                       //
//                                      //
// You found this at bobobobo's weblog, //
// http://bobobobo.wordpress.com        //
//                                      //
// Creation date:  Feb 9/08             //
// Last modified:  Feb 10/08            //
//                                      //
//////////////////////////////////////////

#include <windows.h>
#include <stdio.h>
#include <stdlib.h>
#include <math.h>
#include <gl/gl.h>
#include <gl/glu.h>
#pragma comment(lib, "opengl32.lib")
#pragma comment(lib, "glu32.lib")
#include "mmsystem.h"
#include "pthread.h"

#include "resource.h"

#include "NuiApi.h"

#include "NosuchDebug.h"
#include "stdint.h"
#include "mmtt_creative.h"

////////////////////////
// OK, here's the deal.
//
// OpenGL is typically so hard to
// get started on Windows (what, with the
// HDC and the HGLRC and the PIXELFORMATDESCRIPTOR)
// most get turned off very quickly
// and stick with GLUT for a very long time,
// because GLUT hides away A LOT of the
// complexities of getting a window up
// and drawing into it.

// If you're still reading, you're
// probably tired of doing things the
// GLUT way, and you want to cross over
// to doing things in OpenGL using a pure
// Windows application.

// Good choice my friend!  If you want
// full control over your app, you'll 
// want to do things in pure Win32.

// Once you get used to programming in Win32,
// its not ALL THAT BAD, I promise (evil grin).

// This demo shows you how to get a basic
// window up that OpenGL can draw into,
// and explains each step in detail.

// I give some additional references at the
// bottom of the page.

//////////////////////////
// OVERVIEW:
//
// The OUTPUT of OpenGL is a two dimensional
// PICTURE of a 3D scene.

// That's right.  I said 2D.

// People often say that the process OpenGL
// goes through to render a 3D scene is just like
// a CAMERA TAKING A PHOTOGRAPH OF A REAL WORLD SCENE.
//
// So, you know obviously that a PHOTOGRAPH
// is PURELY 2D after its been developed,
// and it DEPICTS the 3D scene that it
// "took a picture" of accurately.

// That's exactly what OpenGL does.  OpenGL's JOB
// is to take all your instructions, all your
// glColor3f()'s and your glVertex3f()'s, and
// to ultimately end up DRAWING A 2D PICTURE
// from those instructions that can be displayed
// on a computer screen.

// In THIS program, our MAIN GOAL is to
// CONNECT UP the OUTPUT of OpenGL (that 2D
// image OpenGL produces)
// with YOUR APPLICATION'S WINDOW.

// Does this look familiar?

/*
     HDC hdc = GetDC( hwnd );
*/

// You should already know that the way
// you as a Windows programmer draw to
// your application's window is using
// your window's HDC.

// If this doesn't sound familiar,
// then I strongly recommend you go read up
// on Win GDI before continuing!

////////////////////////////////!
//
// So if our way to draw to our application window
// is the HDC, and OpenGL produces some 2D image
// HOW IN THE WORLD DO YOU CONNECT UP the OUTPUT
// of the OpenGL program (that 2D picture)
// WITH the HDC of a WINDOW?
//
// That's the main subject we're tackling here
// in this tutorial.
//
// And its EASY.
//
// WE'RE lucky.  Microsoft created a bunch
// of functions (all beginning with "wgl"),
// that make this job of "connecting up" the output
// of OpenGL with the HDC of our window quite easy!

/////////////////////////
// BIG PICTURE:
//
// Here's the big picture of what
// we're going to be doing here:

/*

|---------|  draws to   |-------|  copied out   |---------|  shows in  |-----------|
|         | ==========> |       | ============> |         | =========> |           |
|---------|             |-------|               |---------|            |-----------|
  OPENGL                  HGLRC                     HDC                 application
 FUNCTION                                                                 window
  CALLS                                                                     

*/

//////////////////////
// In code:  this is the steps
// we'll follow.
//
// 1.  CREATE WINDOW AS USUAL.
//
// 2.  GET DC OF WINDOW.
//     Get the HDC of our window using a line like:
//          hdc = GetDC( hwnd );
//
// 3.  SET PIXEL FORMAT OF HDC.
//     You do 3 things here:
//          Create a PFD ('pixel format descriptor')
//          ChoosePixelFormat()
//          SetPixelFormat()
//
// 4.  CREATE RENDERING CONTEXT (HGLRC).
//          wglCreateContext()
//     Create the surface to which OpenGL
//     shall draw.  It is created such that
//     it shall be completely compatible
//     with the DC of our window, (it will
//     use the same pixel format!)
//
// 5.  CONNECT THE RENDER CONTEXT (HGLRC)
//     WITH THE DEVICE CONTEXT (HDC) OF WINDOW.
//          wglMakeCurrent()
//
// 6.  DRAW USING OPENGL.
//          glVertex3d(); glColor3d(); // ETC!
//     You call OpenGL functions to perform
//     your drawing!  OpenGL will spit out
//     its result picture to the HGLRC, which is
//     connected to the backbuffer of your HDC.
//
// 7.  SWAP BUFFERS.
//          SwapBuffers( hdc );	
//     Assuming you've picked a DOUBLE BUFFERED
//     pixel format all the way back in step 2, 
//     you'll need to SWAP the buffers so that
//     the image you've created using OpenGL on
//     the backbuffer of your hdc is shown
//     in your application window.

// And that's all!

// Ready for the code??? Let's go!


// Function prototypes.
LRESULT CALLBACK WndProc( HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam );
int WINAPI WinMain( HINSTANCE hInstance, HINSTANCE hPrevInstance, LPSTR szCmdLine, int iCmdShow );
void ReSizeGLScene();

void MmttServer::ReSizeGLScene()		// Resize And Initialize The GL Window
{
	// NosuchDebug("ReSizeGLScene %d %d",width,height);
	glViewport(0,0,g.width,g.height);						// Reset The Current Viewport
	glMatrixMode(GL_PROJECTION);						// Select The Projection Matrix
	glLoadIdentity();									// Reset The Projection Matrix
	// Calculate The Aspect Ratio Of The Window
    GLdouble aspect = 1.0;
	GLdouble dnear = 1.0;
	GLdouble dfar = 500.0;
	GLdouble field_of_view_angle = 45.0;
	GLdouble fH = tan( float(field_of_view_angle / 360.0f * 3.14159f) ) * dnear;
	GLdouble fW = fH * aspect;
	glFrustum( -fW, fW, -fH, fH, dnear, dfar );
#if 0
	glMatrixMode(GL_MODELVIEW);							// Select The Modelview Matrix
	glLoadIdentity();									// Reset The Modelview Matrix
#endif
}

////////////////////////
// WNDPROC
// Notice that WndProc is very very neglected.
// We hardly do anything with it!  That's because
// we do all of our processing in the draw()
// function.
LRESULT CALLBACK MmttServer::WndProc(   HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam ) 
{
	// NosuchDebug("MmttServer::WndProc message=0x%x",message);
    switch( message ) {

	case WM_ACTIVATE:
		NosuchDebug(1,"Got WM_ACTIVATE");
		break;

	case WM_SYSCOMMAND:
		switch (wparam) {
			case SC_SCREENSAVE:
			case SC_MONITORPOWER:
				return 0;
		}
		break;

    case WM_CREATE:
        Beep( 50, 10 );
        return 0;
        break;

    case WM_PAINT:
        {
            HDC hdc;
            PAINTSTRUCT ps;
            hdc = BeginPaint( hwnd, &ps );
                // don't draw here.  would be waaay too slow.
                // draw in the draw() function instead.
            EndPaint( hwnd, &ps );
        }
        return 0;
        break;

    case WM_KEYDOWN:
        switch( wparam )
        {
        case VK_ESCAPE:
            PostQuitMessage( 0 );
            break;
        default:
            break;
        }
        return 0;

    case WM_CLOSE:
        PostQuitMessage( 0 ) ;
        return 0;
        break;

    case WM_DESTROY:
        PostQuitMessage( 0 ) ;
        return 0;
        break;

	case WM_SIZE:	// Resize The OpenGL Window
		g.width = LOWORD(lparam);
		g.height = HIWORD(lparam);
		ReSizeGLScene();
		return 0;								// Jump Back
		break;
    }
 
    return DefWindowProc( hwnd, message, wparam, lparam );
}


////////////////
// END NOTES:
//
// Some good references:
// WGL function ref on MSDN:
// http://msdn2.microsoft.com/en-us/library/ms673957%28VS.85%29.aspx

// MSDN example from 1994 (but still good!)
// "OpenGL I: Quick Start"
// http://msdn2.microsoft.com/en-us/library/ms970745.aspx


//////////////////
// QUICK Q&A (make sure you know what's going on):
//
// QUESTION:  What's the means by which we can draw
//            to our window itself?
//
// ANSWER:  The HDC (HANDLE TO DEVICE CONTEXT).

// QUESTION:  What's the means by which OpenGL can
//            draw to the window?
//
// ANSWER:  USING that SAME HDC WE woulda used
//          to draw to it!! (more to come on this now).

/////////////////////
// It IS possible to access the bits
// of the output of OpenGL in 2 ways:
//      1)  Use glReadPixels() to obtain
//          the arrayful of pixels on the
//          screen.  You can then save this
//          to a .TGA or .BMP file easily.

//      2)  Render to a BITMAP, then
//          blit that bitmap to your HDC.
//          There's a 1995 msdn article on this:
//          "OpenGL VI: Rendering on DIBs with PFD_DRAW_TO_BITMAP"
//          http://msdn2.microsoft.com/en-us/library/ms970768.aspx



/* 
     ____   __   __      __   __  ___
    / _  \ /  / /  /    /  /  \ \/  /
   / _/ / /  / /  /    /  /    \   /
  / _/ \ /  / /  /__  /  /__   /  /
 /_____//__/ /______//______/ /__/

*/
#endif
