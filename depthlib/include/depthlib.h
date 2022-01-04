// The following ifdef block is the standard way of creating macros which make exporting
// from a DLL simpler. All files within this DLL are compiled with the DEPTHLIB_EXPORTS
// symbol defined on the command line. This symbol should not be defined on any project
// that uses this DLL. This way any other project whose source files include this file see
// DEPTHLIB_API functions as being imported from a DLL, whereas this DLL sees symbols
// defined with this macro as being exported.
#ifdef DEPTHLIB_EXPORTS
#define DEPTHLIB_API __declspec(dllexport)
#else
#define DEPTHLIB_API __declspec(dllimport)
#endif

typedef void(*DepthCallbackFunc)(char *subj, char *msg);

extern DEPTHLIB_API int DepthIsRunning();
extern DEPTHLIB_API int DepthRun(DepthCallbackFunc f, int show);
extern DEPTHLIB_API int DepthSet(char *name, char *value);
extern DEPTHLIB_API void DepthStop();

