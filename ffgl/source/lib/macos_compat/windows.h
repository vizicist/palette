#pragma once

#ifndef _WIN32

#include <pthread.h>
#include <stdint.h>
#include <stdlib.h>
#include <sys/time.h>
#include <unistd.h>

typedef unsigned long DWORD;
typedef int BOOL;
typedef void* HANDLE;
typedef void* HMODULE;
typedef void* LPVOID;

#ifndef TRUE
#define TRUE 1
#endif
#ifndef FALSE
#define FALSE 0
#endif

#define APIENTRY
#define DLL_PROCESS_ATTACH 1
#define DLL_PROCESS_DETACH 0
#define DLL_THREAD_ATTACH 2
#define DLL_THREAD_DETACH 3
#define WAIT_ABANDONED 0x00000080L
#define INFINITE 0xffffffff
#define MAX_PATH 1024

static inline unsigned long GetModuleFileNameA(HMODULE, char* buffer, unsigned long size) {
	if (buffer && size > 0) {
		buffer[0] = '\0';
	}
	return 0;
}

static inline DWORD timeGetTime(void) {
	struct timeval tv;
	gettimeofday(&tv, NULL);
	return (DWORD)((tv.tv_sec * 1000UL) + (tv.tv_usec / 1000UL));
}

static inline void Sleep(DWORD milliseconds) {
	usleep(milliseconds * 1000);
}

static inline HANDLE CreateMutex(void*, BOOL, const char*) {
	pthread_mutex_t* mutex = (pthread_mutex_t*)malloc(sizeof(pthread_mutex_t));
	if (mutex) {
		pthread_mutex_init(mutex, NULL);
	}
	return (HANDLE)mutex;
}

static inline DWORD WaitForSingleObject(HANDLE handle, DWORD) {
	pthread_mutex_t* mutex = (pthread_mutex_t*)handle;
	if (!mutex) {
		return WAIT_ABANDONED;
	}
	pthread_mutex_lock(mutex);
	return 0;
}

static inline BOOL ReleaseMutex(HANDLE handle) {
	pthread_mutex_t* mutex = (pthread_mutex_t*)handle;
	if (!mutex) {
		return FALSE;
	}
	pthread_mutex_unlock(mutex);
	return TRUE;
}

#endif
