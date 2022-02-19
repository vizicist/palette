#include "NosuchUtil.h"
#include <assert.h>

#include "UT_Mutex.h"

// We don't include the leakwatch here
// since this file is used by users

UT_Mutex::UT_Mutex(const char *name)
{
#ifdef WIN32
    myMutex = CreateMutex(NULL, FALSE, s2ws(name).c_str());
#else
    assert(false);
#endif

}


UT_Mutex::~UT_Mutex()
{
#ifdef WIN32
    CloseHandle(myMutex);
#else
#endif 
}

bool
UT_Mutex::lock(int timeout)
{
#ifdef WIN32
    DWORD result = WaitForSingleObject(myMutex, timeout);
    if (result != WAIT_OBJECT_0)
        return false;
    else
        return true;
#else
    return false;
#endif 
}

bool
UT_Mutex::unlock()
{
#ifdef WIN32
	// Disable some warning about BOOL versus bool
#pragma warning(disable:4800)
    return (bool) ReleaseMutex(myMutex);
#else
    return false;
#endif 
}
