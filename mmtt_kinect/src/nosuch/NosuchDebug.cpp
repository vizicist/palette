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

#include <windows.h>
#include <stdarg.h>
#include <iostream>
#include <fstream>
#include "tstring.h"

#include "pthread.h"

#include <list>

using namespace std;

int NosuchDebugLevel = 0;
bool NosuchDebugToConsole = true;
bool NosuchDebugTimeTag = true;
bool NosuchDebugThread = true;
bool NosuchDebugToLog = true;
bool NosuchDebugToLogWarned = false;
bool NosuchDebugAutoFlush = true;
std::string NosuchAppName = "Nosuch App";

typedef void (*ErrorPopupFuncType)(const char* msg); 
ErrorPopupFuncType NosuchErrorPopup = NULL;

std::string NosuchDebugPrefix = "";
std::string NosuchDebugLogFile = "";
std::string NosuchDebugLogDir = ".";
std::string NosuchDebugLogPath;
std::string NosuchLocalDir = ".";
std::string NosuchPaletteDir = ".";

#ifdef DEBUG_TO_BUFFER
bool NosuchDebugToBuffer = true;
size_t NosuchDebugBufferSize = 8;
static std::list<std::string> DebugBuffer;
#endif

std::list<std::string> DebugLog;
bool DebugInitialized = FALSE;

HANDLE dMutex;

void
NosuchDebugSetLogDirFile(std::string logdir, std::string logfile)
{
	NosuchDebugLogFile = logfile;
	NosuchDebugLogDir = logdir;
	NosuchDebugLogPath = logdir + "/" + logfile;
}

void
RealDebugDumpLog() {
	ofstream f(NosuchDebugLogPath.c_str(),ios::app);
	if ( ! f.is_open() ) {
		NosuchDebugLogPath = "c:/windows/temp/debug.txt";
		f.open(NosuchDebugLogPath.c_str(),ios::app);
		if ( ! f.is_open() ) {
			return;
		}
	}

	while (!DebugLog.empty()) {
		std::string s = DebugLog.front();
	    f << s;
		DebugLog.pop_front();
	}
	f.close();
}

void
NosuchDebugDumpLog()
{
	DWORD wait = WaitForSingleObject( dMutex, INFINITE);
	if ( wait == WAIT_ABANDONED )
		return;

	RealDebugDumpLog();

	ReleaseMutex(dMutex);
}

void
RealNosuchDebugInit() {
	if ( ! DebugInitialized ) {
		dMutex = CreateMutex(NULL, FALSE, NULL);
		DebugInitialized = TRUE;
	}
}

void
RealNosuchDebug(int level, char const *fmt, va_list args)
{
	RealNosuchDebugInit();
	if ( level > NosuchDebugLevel )
		return;

	DWORD wait = WaitForSingleObject( dMutex, INFINITE);
	if ( wait == WAIT_ABANDONED )
		return;

    // va_list args;
    char msg[10000];
	char* pmsg = msg;
	int msgsize = sizeof(msg)-2;

	if ( NosuchDebugPrefix != "" ) {
		int nchars = _snprintf_s(pmsg,msgsize,_TRUNCATE,"%s",NosuchDebugPrefix.c_str());
		pmsg += nchars;
		msgsize -= nchars;
	}
	if ( NosuchDebugTimeTag ) {
		int nchars;
		long tm;
		tm = 0;
		if ( NosuchDebugThread ) {
			nchars = _snprintf_s(pmsg,msgsize,_TRUNCATE,"[%.3f,T%ld] ",tm/1000.0f,(int)pthread_self().p);
		} else {
			nchars = _snprintf_s(msg,msgsize,_TRUNCATE,"[%.3f] ",tm/1000.0f);
		}
		pmsg += nchars;
		msgsize -= nchars;
	}

    // va_start(args, fmt);
    vsprintf_s(pmsg,msgsize,fmt,args);

	char *p = strchr(msg,'\0');
	if ( p != NULL && p != msg && *(p-1) != '\n' ) {
		strcat_s(msg,msgsize,"\n");
	}

	if ( NosuchDebugToConsole ) {
		OutputDebugStringA(msg);
	}
	if ( NosuchDebugToLog ) {
		DebugLog.push_back(msg);
		if ( NosuchDebugAutoFlush )
			RealDebugDumpLog();
	}

#ifdef DEBUG_TO_BUFFER
	if ( NosuchDebugToBuffer ) {
		// We want the entries in the DebugBuffer to be single lines,
		// so that someone can request a specific number of lines.
		std::istringstream iss(msg);
		std::string line;
		while (std::getline(iss, line)) {
			DebugBuffer.push_back(line+"\n");
		}
		while ( DebugBuffer.size() >= NosuchDebugBufferSize ) {
			DebugBuffer.pop_front();
		}
	}
#endif

    // va_end(args);

	ReleaseMutex(dMutex);
}

void
NosuchDebug(char const *fmt, ...)
{
    va_list args;
    va_start(args, fmt);
	RealNosuchDebug(0,fmt,args);
    va_end(args);
}

void
NosuchDebug(int level, char const *fmt, ...)
{
    va_list args;
    va_start(args, fmt);
	RealNosuchDebug(level,fmt,args);
    va_end(args);
}

void
NosuchErrorOutput(const char *fmt, ...)
{
	RealNosuchDebugInit();

	if ( fmt == NULL ) {
		// Yes, this is recursive, but we're passing in a non-NULL fmt...
		NosuchErrorOutput("fmt==NULL in NosuchErrorOutput!?\n");
		return;
	}

    va_list args;
    va_start(args, fmt);

    char msg[10000];
    vsprintf_s(msg,sizeof(msg)-2,fmt,args);
    va_end(args);

	char *p = strchr(msg,'\0');
	if ( p != NULL && p != msg && *(p-1) != '\n' ) {
		strcat_s(msg,sizeof(msg),"\n");
	}

	if ( NosuchErrorPopup != NULL ) {
		NosuchErrorPopup(msg);
	}

	OutputDebugStringA(msg);

#if 0
	// Why doesn't this work?
	// Trying to force it into the debug output
    va_list args2;
    va_start(args2, fmt);
	RealNosuchDebug(-1,"%s",args2);
    va_end(args2);
#endif
}

std::string
NosuchDataPath(std::string filepath)
{
	char *data_path = getenv("PALETTE_DATA_PATH");
	if ( data_path == NULL ) {
		if ( filepath == "." ) {
			return NosuchLocalDir;
		}
		else {
			return NosuchLocalDir + "/" + filepath;
		}
	}
	return std::string(data_path) + "/" + filepath;
}

#ifdef OLDSTUFF
std::string
NosuchPalettePath(std::string filepath)
{
	if ( filepath == "." ) {
		return NosuchPaletteDir;
	} else {
		return NosuchPaletteDir + "/" + filepath;
	}
}
#endif

std::string
NosuchSnprintf(const char *fmt, ...)
{
	static char *msg = NULL;
	static int msglen = 4096;
	va_list args;

	if ( msg == NULL )
		msg = (char*)malloc(msglen);

	while (1) {
		va_start(args, fmt);
		int written = vsnprintf_s(msg,msglen,_TRUNCATE,fmt,args);
		va_end(args);
		if ( written < msglen ) {
			return std::string(msg);
		}
		free(msg);
		msglen *= 2;
		msg = (char*)malloc(msglen);
	}
}

std::string
NosuchForwardSlash(std::string filepath) {
	int i;
	while ( (i=filepath.find("\\")) != filepath.npos ) {
		filepath.replace(i,1,"/");
	}
	return filepath;
}
