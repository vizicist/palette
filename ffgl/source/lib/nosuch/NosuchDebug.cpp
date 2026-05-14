#include <windows.h>
#include <stdarg.h>
#include <iostream>
#include <fstream>
#ifndef _WIN32
#include <libgen.h>
#include <limits.h>
#include <stdio.h>
#include <string.h>
#endif

#include "NosuchUtil.h"

#include <list>
#include <map>

using namespace std;

int NosuchDebugLevel = 0;
bool NosuchDebugCursor = false;
bool NosuchDebugAPI = false;
bool NosuchDebugParam = false;
bool NosuchDebugSprite = false;
bool NosuchDebugTimeTag = true;
bool NosuchDebugToLog = true;
bool NosuchDebugToLogWarned = false;
bool NosuchDebugAutoFlush = true;
std::string NosuchAppName = "Nosuch App";

int NosuchDebugTag = 0;
std::string NosuchDebugPrefix = "";
std::string NosuchDebugLogPath;

#ifdef DEBUG_TO_BUFFER
bool NosuchDebugToBuffer = true;
size_t NosuchDebugBufferSize = 8;
static std::list<std::string> DebugBuffer;
#endif

std::list<std::string> DebugLog;
bool DebugInitialized = FALSE;

HANDLE dMutex;

std::map<void*, std::string> DebugThreadMap;

static Milliseconds Time0 = 0;

Milliseconds MillisecondsSoFar() {
	if (Time0 == 0) {
		Time0 = timeGetTime();
	}
	return timeGetTime() - Time0;
}

void NosuchPrintTime(const char *prefix) {
	Milliseconds milli = MillisecondsSoFar();
	long secs = milli / 1000;
	milli -= secs * 1000;
	NosuchDebug("%s: time= %ld.%03u\n",prefix,secs,milli);
}

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
#ifdef _WIN32
		int written = vsnprintf_s(msg,msglen,_TRUNCATE,fmt,args);
#else
		int written = vsnprintf(msg,msglen,fmt,args);
#endif
		va_end(args);
		if ( written >= 0 && written < msglen ) {
			return std::string(msg);
		}
		free(msg);
		msglen *= 2;
		msg = (char*)malloc(msglen);
	}
}

void
NosuchDebugSetThreadName(void* p, std::string name) {
	DebugThreadMap.insert_or_assign(p, name);
}

void
RealDebugDumpLog() {
	ofstream f(NosuchDebugLogPath.c_str(),ios::app);
	if ( ! f.is_open() ) {
		NosuchDebugLogPath = "c:/windows/temp/ffgl.log";
		f.open(NosuchDebugLogPath.c_str(),ios::app);
		if ( ! f.is_open() ) {
			return;
		}
	}

	while (!DebugLog.empty()) {
		std::string s = DebugLog.front();
	    f << s.c_str();
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

std::string
PaletteDataPath()
{

#ifdef _WIN32
	errno_t err;
	char* palette = NULL;
	char* palette_data_path = NULL;
	char* source = NULL;
	char *dataval = NULL;
	char *value = NULL;
	char *dataRoot = NULL;
	char *localAppData = NULL;
	std::string sourcepath = "";
	std::string datapath;
	size_t len;

	err = _dupenv_s( &dataval, &len, "PALETTE_DATA" );
	if ( err != 0 || dataval == NULL ) {
		dataval = "default"; // default value
	}
	std::string datadir = "data_" + std::string( dataval );	

	err = _dupenv_s( &dataRoot, &len, "PALETTE_DATAROOT" );
	if ( err == 0 && dataRoot != NULL && dataRoot[0] != '\0' ) {
		datapath = std::string(dataRoot) + "\\" + datadir;
	} else {
		err = _dupenv_s( &localAppData, &len, "LOCALAPPDATA" );
		if ( err == 0 && localAppData != NULL && localAppData[0] != '\0' ) {
			datapath = std::string(localAppData) + "\\Palette\\" + datadir;
		} else {
			err = _dupenv_s( &palette, &len, "PALETTE" );
			if( err || palette == NULL )
			{
				NosuchDebug( "No value for PALETTE environment variable!?\n" );
				return NULL;
			}
			datapath = std::string(palette) + "\\" + datadir;
		}
	}

	NosuchDebug( "NosuchDebugInit: Level=%d Cursor=%d Param=%d API=%d\n", NosuchDebugLevel, NosuchDebugCursor , NosuchDebugParam, NosuchDebugAPI);
	return datapath;
#else
	const char* palette = getenv("PALETTE");
	const char* dataval = getenv("PALETTE_DATA");
	if (palette == NULL || palette[0] == '\0') {
		palette = getenv("HOME");
	}
	if (dataval == NULL || dataval[0] == '\0') {
		dataval = "default";
	}
	if (palette == NULL || palette[0] == '\0') {
		return "/tmp";
	}
	return std::string(palette) + "/data_" + std::string(dataval);
#endif
}

void
RealNosuchDebugInit() {
	if ( DebugInitialized ) {
		return;
	}

	NosuchDebugLogPath = "c:\\windows\\temp\\ffgl.log";// last resort

#ifdef _WIN32
	errno_t err;
	char* palette = NULL;
	char *value = NULL;
	size_t len;

	err = _dupenv_s( &palette, &len, "PALETTE" );
	if( err || palette == NULL )
	{
		NosuchDebug( "No value for PALETTE environment variable!?\n" );
		return;
	}

	dMutex = CreateMutex(NULL, FALSE, NULL);
	DebugInitialized = TRUE;

	std::string datapath = PaletteDataPath();
	NosuchDebugLogPath = datapath + "\\logs\\ffgl.log";

	NosuchDebug( "NosuchDebugInit: Level=%d Cursor=%d Param=%d API=%d\n", NosuchDebugLevel, NosuchDebugCursor , NosuchDebugParam, NosuchDebugAPI);

	if (value != NULL) {
			free( value );
	}
	if (palette != NULL) {
			free( palette );
	}
#else
	std::string datapath = PaletteDataPath();
	NosuchDebugLogPath = datapath + "/logs/ffgl.log";
	dMutex = CreateMutex(NULL, FALSE, NULL);
	DebugInitialized = TRUE;
#endif
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
		int nchars = snprintf(pmsg,msgsize,"%s",NosuchDebugPrefix.c_str());
		pmsg += nchars;
		msgsize -= nchars;
	}
	if ( NosuchDebugTimeTag ) {
		int nchars;
		long tm;
		tm = MillisecondsSoFar();
		nchars = snprintf(pmsg, msgsize, "[%.3f,%d] ", tm / 1000.0f, NosuchDebugTag);
		pmsg += nchars;
		msgsize -= nchars;
	}

    // va_start(args, fmt);
    vsnprintf(pmsg,msgsize,fmt,args);

	char *p = strchr(msg,'\0');
	if ( p != NULL && p != msg && *(p-1) != '\n' ) {
		strncat(msg,"\n",msgsize - strlen(msg) - 1);
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
    vsnprintf(msg,sizeof(msg)-2,fmt,args);
    va_end(args);

	char *p = strchr(msg,'\0');
	if ( p != NULL && p != msg && *(p-1) != '\n' ) {
		strncat(msg,"\n",sizeof(msg) - strlen(msg) - 1);
	}

#ifdef _WIN32
	OutputDebugStringA(msg);
#else
	fputs(msg, stderr);
#endif

#ifdef THIS_DOES_NOT_WORK
	// Why doesn't this work?
	// Trying to force it into the debug output
    va_list args2;
    va_start(args2, fmt);
	RealNosuchDebug(-1,"%s",args2);
    va_end(args2);
#endif
}

std::string
NosuchForwardSlash(std::string filepath) {
	size_t i;
	while ( (i=filepath.find("\\")) != filepath.npos ) {
		filepath.replace(i,1,"/");
	}
	return filepath;
}
