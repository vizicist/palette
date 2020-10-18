//
// FFGLPluginInfoData.cpp
//
// Usually you do not need to edit this file!
//

#include "FFGLPluginInfo.h"
#include <string>

extern std::string ffgl_name();
CFFGLPluginInfo& ffgl_plugininfo();

void ffgl_setid(CFFGLPluginInfo& plugininfo, std::string name)
{
	char id[5];
	// Compute a hash of the plugin name and use two 4-bit values
	// from it to produce the last 2 characters of the unique ID.
	// It's possible there will be a collision.
	int hash = 0;
	for (const char* p = name.c_str(); *p != '\0'; p++) {
		hash += *p;
	}
	id[0] = 'V';
	id[1] = 'Z';
	id[2] = 'A' + (hash & 0xf);
	id[3] = 'A' + ((hash >> 4) & 0xf);
	id[4] = '\0';
	plugininfo.SetPluginIdAndName(id, name.c_str());
}

//////////////////////////////////////////////////////////////////
// Information about the plugin
//////////////////////////////////////////////////////////////////

CFFGLPluginInfo* g_CurrPluginInfo = NULL;

// If your plugin wants to know the pathname of the DLL (e.g. for locating
// other data files associated with the plugin), it should be compiled with
// FFGL_OVERRIDE_SETDLL defined, and then provide its own ffgl_setdll() function.
extern "C" {
#ifndef FFGL_OVERRIDE_SETDLL
	bool ffgl_setdll(std::string dllpath) {
		return true;
	}
#endif
	extern bool ffgl_setdll(std::string dllpath);
}

//////////////////////////////////////////////////////////////////
// Plugin dll entry point
//////////////////////////////////////////////////////////////////
#ifdef _WIN32
BOOL APIENTRY DllMain(HANDLE hModule, DWORD  ul_reason_for_call, LPVOID lpReserved)
{
	char dllpath[MAX_PATH];
	GetModuleFileNameA((HMODULE)hModule, dllpath, MAX_PATH);

	if (ul_reason_for_call == DLL_PROCESS_ATTACH) {
		// Initialize once for each new process.
		// Return FALSE if we fail to load DLL.
		if (!ffgl_setdll(std::string(dllpath))) {
			printf("ffgl_setdll failed");
			return FALSE;
		}
		std::string s = ffgl_name();
		ffgl_setid(ffgl_plugininfo(), s);
		printf("DLLPROCESS_ATTACH dll=%s", dllpath);
	}
	if (ul_reason_for_call == DLL_PROCESS_DETACH) {
		printf("DLLPROCESS_DETACH dll=%s", dllpath);
	}
	if (ul_reason_for_call == DLL_THREAD_ATTACH) {
		printf("DLLTHREAD_ATTACH dll=%s", dllpath);
	}
	if (ul_reason_for_call == DLL_THREAD_DETACH) {
		printf("DLLTHREAD_DETACH dll=%s", dllpath);
	}
	return TRUE;
}
#endif
