#include "NosuchUtil.h"
#ifdef WIN32
#else
   #include <string.h>
   #define _strdup(s) strdup(s) 
#endif 
#include <assert.h>
#include <stdlib.h>
#include <time.h>
#include "UT_SharedMem.h"

bool
UT_SharedMem::open(const char *name,  unsigned int size, bool supportInfo)
{
	mySize = size;
	myMemory = 0;
	myMapping = 0;
    myName = 0;
	mySharedMemInfo = NULL;
    memset(myNamePostFix, 0, UT_SHM_MAX_POST_FIX_SIZE);
    myShortName = _strdup(name);

	mySupportInfo = supportInfo;

    int len = strlen(myShortName);

    createName();
    char *m = new char[strlen(myName) + 5 + 1];
    strcpy(m, myName);
    strcat(m, "Mutex");
    myMutex = new UT_Mutex(m);
    delete m;

    if (size > 0)
		myAmOwner = true;
    else
		myAmOwner = false;

    if (supportInfo)
    {
		if (!createInfo())
		{
			return false;
		}
    }
    else
    {
		mySharedMemInfo = NULL;
    }

    if (size > 0)
    {
		if (!createSharedMem())
		{
		    myErrorState = UT_SHM_ERR_ALREADY_EXIST;
		    return false;
		}
    }
    else
    {
		if (!openSharedMem())
		{
		    myErrorState = UT_SHM_ERR_DOESNT_EXIST;
		    return false;
		}
    }
    myErrorState = UT_SHM_ERR_NONE;
	return true;
}

UT_SharedMem::UT_SharedMem(const char *name)
{
	open(name);
}

UT_SharedMem::UT_SharedMem(const char *name, unsigned int size)
{
	open(name, size);
}

UT_SharedMem::UT_SharedMem(const char *name,  unsigned int size, bool supportInfo)
{
	open(name, size, supportInfo);
}

UT_SharedMem::~UT_SharedMem()
{
    detach();
    delete myShortName;
    delete myName;
    delete mySharedMemInfo;
    delete myMutex;
}

bool
UT_SharedMem::checkInfo()
{
	if (mySupportInfo)
	{
		// If we are looking for an info and can't find it,
		// then release the segment also
		if (!createInfo())
		{
			detach();
			myErrorState = UT_SHM_ERR_INFO_DOESNT_EXIST;
			return false;
		}
	}

    if (mySharedMemInfo && mySharedMemInfo->getErrorState() == UT_SHM_ERR_NONE && !myAmOwner)
    {
		mySharedMemInfo->lock();
		UT_SharedMemInfo *info = (UT_SharedMemInfo*)mySharedMemInfo->getMemory();

		if (info->version > 1)
		{
			if (info->detach)
			{
				mySharedMemInfo->unlock();
				detach();
				myErrorState = UT_SHM_ERR_INFO_DOESNT_EXIST;
				return false;
			}
		}

		char pn[UT_SHM_MAX_POST_FIX_SIZE];
		memcpy(pn, info->namePostFix, UT_SHM_MAX_POST_FIX_SIZE);
		
		if (strcmp(pn, myNamePostFix) != 0)
		{
		    memcpy(myNamePostFix, pn, UT_SHM_MAX_POST_FIX_SIZE);
		    detachInternal();
		}
		mySharedMemInfo->unlock();

    }
	return true;
}

void
UT_SharedMem::resize(unsigned int s)
{

    // This can't be called by someone that didn't create it in the first place
    // Also you can't resize it if you arn't using the info feature
    // Finally, don't set the size to 0, just delete this object if you want to clean it
    if (mySize > 0 && mySharedMemInfo && myAmOwner)
    {
		mySharedMemInfo->lock();
		UT_SharedMemInfo *info = (UT_SharedMemInfo*)mySharedMemInfo->getMemory();
		if (info && info->supported)
		{
		    detachInternal();
		    mySize = s;
		    // Keep trying until we find a name that works
		    do 
		    {
				randomizePostFix();
				createName();
		    } while(!createSharedMem());
		    memcpy(info->namePostFix, myNamePostFix, UT_SHM_MAX_POST_FIX_SIZE);
		}
		else // Otherwise, just try and detach and resize, if it fails give up
		{
		    detachInternal();
		    mySize = s;
		    if (!createSharedMem())
		    {
				myErrorState = UT_SHM_ERR_ALREADY_EXIST;
		    }

		}
		mySharedMemInfo->unlock();
    }
}

void
UT_SharedMem::randomizePostFix()
{
    for (int i = 0; i < UT_SHM_MAX_POST_FIX_SIZE - 1; i++)
    {
		int r = rand() % 26;
		char ch = 'a' + r;
		myNamePostFix[i] = ch;
    }
}

void
UT_SharedMem::createName()
{
    if (!myName)
		myName = new char[strlen(myShortName) + 10 + UT_SHM_MAX_POST_FIX_SIZE];

    strcpy(myName, "TouchSHM");
    strcat(myName, myShortName);
    strcat(myName, myNamePostFix);
}

bool
UT_SharedMem::createSharedMem()
{
    if (myMapping)
		return true;

#ifdef WIN32 
    myMapping = CreateFileMapping(INVALID_HANDLE_VALUE, 
		    						  NULL,
		    						  PAGE_READWRITE,
								  0,
								  mySize,
								  s2ws(myName).c_str());

    if (GetLastError() == ERROR_ALREADY_EXISTS)
    {
		detach();
		return false;
    }
#else
    assert(false);
#endif 

    if (myMapping)
		return true;
    else
		return false;
}

bool
UT_SharedMem::openSharedMem()
{
    if (myMapping)
		return true;
    createName();
#ifdef WIN32 
    myMapping = OpenFileMapping( FILE_MAP_ALL_ACCESS, FALSE, s2ws(myName).c_str());
#else
    assert(false);
#endif 

    if (!myMapping)
		return false;


   return true;
}

bool
UT_SharedMem::detachInternal()
{
    if (myMemory)
    {
#ifdef WIN32 
		UnmapViewOfFile(myMemory);
#else
        assert(false);
#endif 
		myMemory = 0;
    }
    if (myMapping)
    {
#ifdef WIN32 
		CloseHandle(myMapping);
#else
        assert(false);
#endif 
		myMapping = 0;
    }


    // Try to open the file again, if it works then someone else is still holding onto the file
    if (openSharedMem())
    {
#ifdef WIN32
		CloseHandle(myMapping);
#else
        assert(false);
#endif 
		myMapping = 0;
		return false;
    }
    
		    
    return true;
}


bool
UT_SharedMem::detach()
{
    if (mySharedMemInfo)
    {
		if (mySharedMemInfo->getErrorState() == UT_SHM_ERR_NONE)
		{
			mySharedMemInfo->lock();
			UT_SharedMemInfo *info = (UT_SharedMemInfo*)mySharedMemInfo->getMemory();
			if (info && myAmOwner)
			{
				info->detach = true;
			}
			mySharedMemInfo->unlock();
		}
		delete mySharedMemInfo;
		mySharedMemInfo = NULL;
	}
	memset(myNamePostFix, 0, sizeof(myNamePostFix));
	return detachInternal();
}

bool
UT_SharedMem::createInfo()
{
	if (!mySupportInfo)
		return true;
	if (mySharedMemInfo)
	{
		return mySharedMemInfo->getErrorState() == UT_SHM_ERR_NONE;
	}

	srand((unsigned int)time(NULL));
	char *infoName = new char[strlen(myName) + strlen(UT_SHM_INFO_DECORATION) + 1];
	strcpy(infoName, myName);
	strcat(infoName, UT_SHM_INFO_DECORATION);
	mySharedMemInfo = new UT_SharedMem(infoName, 
									   myAmOwner ? sizeof(UT_SharedMemInfo) : 0, false);
	delete infoName;
	if (myAmOwner)
	{
		if (mySharedMemInfo->getErrorState() != UT_SHM_ERR_NONE)
		{
			myErrorState = UT_SHM_ERR_INFO_ALREADY_EXIST;
			return false;
		}
		mySharedMemInfo->lock();
		UT_SharedMemInfo *info = (UT_SharedMemInfo*)mySharedMemInfo->getMemory();
		if (!info)
		{
			myErrorState = UT_SHM_ERR_UNABLE_TO_MAP;
			mySharedMemInfo->unlock();
			return false;
		}
		info->magicNumber = UT_SHM_INFO_MAGIC_NUMBER;
		info->version = 2;
		info->supported = false;
		info->detach = false;
		memset(info->namePostFix, 0, UT_SHM_MAX_POST_FIX_SIZE);
		mySharedMemInfo->unlock();
	}
	else
	{
		if (mySharedMemInfo->getErrorState() != UT_SHM_ERR_NONE)
		{
			myErrorState = UT_SHM_ERR_INFO_DOESNT_EXIST;
			return false;
		}
		mySharedMemInfo->lock();
		UT_SharedMemInfo *info = (UT_SharedMemInfo*)mySharedMemInfo->getMemory();
		if (!info)
		{
			myErrorState = UT_SHM_ERR_UNABLE_TO_MAP;
			mySharedMemInfo->unlock();
			return false;
		}
		if (info->magicNumber != UT_SHM_INFO_MAGIC_NUMBER)
		{
			myErrorState = UT_SHM_ERR_INFO_DOESNT_EXIST;
			mySharedMemInfo->unlock();
			return false;
		}
		// Let the other process know that we support the info
		info->supported = true;
		mySharedMemInfo->unlock();
	}

	return true;
}

void *
UT_SharedMem::getMemory()
{
	if (!checkInfo())
	{
		return NULL;
	}

    if( myMemory == 0 )
    {
		if ((myAmOwner && createSharedMem()) || (!myAmOwner && openSharedMem()))
		{
#ifdef WIN32 
		    myMemory = MapViewOfFile(myMapping, FILE_MAP_ALL_ACCESS, 0, 0, 0);
#else
		    assert(false);
		    myMemory = NULL;
#endif 
		    if (!myMemory)
				myErrorState = UT_SHM_ERR_UNABLE_TO_MAP;
		}
    }
    if (myMemory)
    {
		myErrorState = UT_SHM_ERR_NONE;
    }
    return myMemory;
}

bool
UT_SharedMem::lock()
{
    return myMutex->lock(5000);
}

bool
UT_SharedMem::unlock()
{
    return myMutex->unlock();
}
