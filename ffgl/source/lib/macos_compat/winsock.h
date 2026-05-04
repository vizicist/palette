#pragma once

#ifndef _WIN32

#include <errno.h>
#include <fcntl.h>
#include <netdb.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <stdint.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <unistd.h>

typedef int SOCKET;
typedef struct sockaddr SOCKADDR;
typedef struct sockaddr_in SOCKADDR_IN;
typedef struct hostent* PHOSTENT;
typedef struct sockaddr* LPSOCKADDR;
typedef unsigned long DWORD;

#ifndef FAR
#define FAR
#endif

#define INVALID_SOCKET (-1)
#define SOCKET_ERROR (-1)
#define WSAEADDRINUSE EADDRINUSE
#define WSAENOTSOCK ENOTSOCK
#define WSAEWOULDBLOCK EWOULDBLOCK

static inline int WSAGetLastError(void) {
	return errno;
}

static inline int closesocket(SOCKET s) {
	return close(s);
}

static inline int ioctlsocket(SOCKET s, long cmd, unsigned long* argp) {
	return ioctl(s, (unsigned long)cmd, argp);
}

#endif
