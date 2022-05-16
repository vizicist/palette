// This is to disable the "deprecated" warnings from some of the include files
#pragma warning( disable : 4995 )

#include "stdafx.h"
#include <strsafe.h>
#include <direct.h>
#include <math.h>
#include "mmsystem.h"
#include "pthread.h"

#include "resource.h"

#include "NosuchUtil.h"
#include "mmtt.h"

#include <GL/gl.h>
#include <GL/glu.h>

#pragma comment(lib, "opengl32.lib")
#pragma comment(lib, "glu32.lib")

void ReSizeGLScene();		// Resize And Initialize The GL Window

LRESULT CALLBACK ThisWndProc( HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam );

int APIENTRY wWinMain(HINSTANCE hInstance, HINSTANCE hPrevInstance, LPWSTR lpCmdLine, int nCmdShow)
{
	// Change directory to wherever the binary is, so that all file paths
	// can be relative to it.
	char mypath[MAX_PATH];
	GetModuleFileNameA((HMODULE)hInstance, mypath, MAX_PATH);
	std::string path = mypath;
	size_t lastslash = path.find_last_of("/\\");
	if ( lastslash != path.npos ) {
		std::string dir = path.substr(0,lastslash);
		if ( _chdir(dir.c_str()) != 0 ) {
			std::string msg = NosuchSnprintf("*** Error ***\n\nUnable to chdir to %s, errno=%d",dir.c_str(),errno);
			MessageBoxA(NULL,msg.c_str(),"Mmtt_kinect",MB_OK);
			exit(1);
		}
	}

	MmttServer *server = MmttServer::makeMmttServer();
	if ( server == NULL ) {
		NosuchDebug("Unable to create MmttServer!!?");
		std::string msg = NosuchSnprintf("*** Error ***\n\nUnable to create MmttServer");
		MessageBoxA(NULL,msg.c_str(),"Mmtt_kinect",MB_OK);
		exit(1);
	}
    return server->Run(hInstance, nCmdShow);
}

LRESULT CALLBACK ThisWndProc(   HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam ) 
{
	return ThisServer->WndProc(hwnd,message,wparam,lparam);
}

/*
	Space Manifold - a variety of tools for depth cameras and FreeFrame

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

int MmttServer::Run(HINSTANCE hInstance, int nCmdShow)
{
    MSG       msg = {0};

    g.hInstance = hInstance;

    WNDCLASSEX wcx;
	wcx.cbSize = sizeof(WNDCLASSEX);
    wcx.cbClsExtra = 0; 
    wcx.cbWndExtra = 0; 
    wcx.hInstance = hInstance;         
	wcx.hIcon = LoadIcon(GetModuleHandle(NULL), MAKEINTRESOURCE(IDI_APP));
	wcx.hCursor	= LoadCursor(NULL, IDC_ARROW);
	wcx.hbrBackground = (HBRUSH)(COLOR_WINDOW+1);
    wcx.lpfnWndProc = ThisWndProc;         
    wcx.lpszClassName = TEXT("MMTT");
    wcx.lpszMenuName = 0; 
    wcx.style = CS_HREDRAW | CS_VREDRAW | CS_OWNDC;
    wcx.hIconSm = 0;

    // Register that class with the Windows O/S..
	if ( !RegisterClassEx(&wcx) ) {
		MessageBox(NULL, TEXT("Window Registration Failed!"), TEXT("Error!"),
            MB_ICONEXCLAMATION | MB_OK);
        return 0;
	}
    
    RECT rect;
    SetRect( &rect, 50,  // left
                    50,  // top
                    450, // right
                    350 ); // bottom
    
    g.width = rect.right - rect.left;
    g.height = rect.bottom - rect.top;
    
    AdjustWindowRect( &rect, WS_OVERLAPPEDWINDOW, false );

    g.hwnd = CreateWindow(TEXT("MMTT"),
                          TEXT("MMTT"),
                          WS_OVERLAPPEDWINDOW,
                          rect.left, rect.top,  // adjusted x, y positions
                          rect.right - rect.left, rect.bottom - rect.top,  // adjusted width and height
                          NULL, NULL,
                          hInstance, NULL);

    if( g.hwnd == NULL ) {
        FatalAppExit( NULL, TEXT("CreateWindow() failed!") );
    }

    // and show.
    ShowWindow( g.hwnd, nCmdShow );

    g.hdc = GetDC( g.hwnd );

    PIXELFORMATDESCRIPTOR pfd = { 0 };

    pfd.nSize = sizeof( PIXELFORMATDESCRIPTOR );    // just its size
    pfd.nVersion = 1;   // always 1

    pfd.dwFlags = PFD_SUPPORT_OPENGL |  // OpenGL support - not DirectDraw
                  PFD_DOUBLEBUFFER   |  // double buffering support
                  PFD_DRAW_TO_WINDOW;   // draw to the app window, not to a bitmap image

    pfd.iPixelType = PFD_TYPE_RGBA ;    // red, green, blue, alpha for each pixel
    pfd.cColorBits = 24;                // 24 bit == 8 bits for red, 8 for green, 8 for blue.
                                        // This count of color bits EXCLUDES alpha.

    pfd.cDepthBits = 32;                // 32 bits to measure pixel depth.  That's accurate!

    int chosenPixelFormat = ChoosePixelFormat( g.hdc, &pfd );
    if( chosenPixelFormat == 0 ) {
        FatalAppExit( NULL, TEXT("ChoosePixelFormat() failed!") );
    }

    int result = SetPixelFormat( g.hdc, chosenPixelFormat, &pfd );

    if (result == NULL) {
        FatalAppExit( NULL, TEXT("SetPixelFormat() failed!") );
    }

    g.hglrc = wglCreateContext( g.hdc );

    wglMakeCurrent( g.hdc, g.hglrc );

	ReSizeGLScene();		// Resize And Initialize The GL Window

	if ( ! camera->InitializeCamera() ) {
        FatalAppExit( NULL, TEXT("No depth camera detected!") );
		exit(1);
	}

    // Main message loop
    while (WM_QUIT != msg.message) {

		camera->Update();

		check_json_and_execute();
		analyze_depth_images();
		draw_depth_image();

        if (PeekMessageW(&msg, NULL, 0, 0, PM_REMOVE)) {
            // If a dialog message will be taken care of by the dialog proc
            if ((g.hwnd != NULL) && IsDialogMessageW(g.hwnd, &msg)) {
                continue;
            }

            TranslateMessage(&msg);
            DispatchMessageW(&msg);
        }
    }

    return static_cast<int>(msg.wParam);
}

LRESULT CALLBACK MmttServer::DlgProc(HWND hWnd, UINT message, WPARAM wParam, LPARAM lParam)
{
    switch (message) {
        case WM_INITDIALOG:
	        break;

        // If the titlebar X is clicked, destroy app
        case WM_CLOSE:
            DestroyWindow(hWnd);
            break;

        case WM_DESTROY:
            // Quit the main message pump
            PostQuitMessage(0);
            break;

		case WM_SIZE:	// Resize The OpenGL Window
			g.width = LOWORD(lParam);
			g.height = HIWORD(lParam);
			ReSizeGLScene();
			break;

        // Handle button press
        case WM_COMMAND:
            break;
    }

    return FALSE;
}
