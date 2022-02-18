/*
 * COMMENTS:
 *      This class is for inter-processes locking based on named mutex
 *
 */

#ifndef __UT_Mutex__
#define __UT_Mutex__


#ifdef WIN32
#include <windows.h>
#else
#include <libkern/OSAtomic.h>
#endif

#ifdef WIN32
    typedef HANDLE mutexId;
#else
    typedef int mutexId;
#endif


// Needed so people can compile this outside the Touch build environement
#ifndef DLLEXP
#define DLLEXP
#endif 
    
class DLLEXP UT_Mutex
{
public:
     UT_Mutex(const char *name);
     ~UT_Mutex();

     bool       lock(int timeout);
     bool       unlock();
     
     // This class is distributed to the users, so make sure it doesn't
     // rely on any internal Touch classes

private:
     mutexId       myMutex;
};


#endif /* __UT_Mutex__ */
