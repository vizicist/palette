#ifndef _NSOSCUDP
#define _NSOSCUDP

#include "NosuchOscInput.h"
#include "winsock.h"

class NosuchOscUdpInput : public NosuchOscInput {

public:
	NosuchOscUdpInput(const char *host, int port, NosuchOscMessageProcessor* processor);
	virtual ~NosuchOscUdpInput();
	int Listen();
	void Check();
	void UnListen();
	const char *Host() { return _myhost; }
	int Port() { return _myport; }

private:
	SOCKET _s;
	int _myport;
	const char *_myhost;
	NosuchOscMessageProcessor* _processor;
};

#endif