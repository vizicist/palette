#ifndef _NosuchOscTcpInput
#define _NosuchOscTcpInput

#include "NosuchOscInput.h"

class NosuchSocket;
class NosuchSocketMemory;

#define SLIP_END 192
#define SLIP_ESC 219
#define SLIP_ESC2 221

#define IS_SLIP_END(c) (((c)&0xff)==SLIP_END)
#define IS_SLIP_ESC(c) (((c)&0xff)==SLIP_ESC)
#define IS_SLIP_ESC2(c) (((c)&0xff)==SLIP_ESC2)

class NosuchOscTcpInput : public NosuchOscInput {

public:
	NosuchOscTcpInput(const char *host, int port, NosuchOscMessageProcessor* processor);
	~NosuchOscTcpInput();
	int Listen();
	void Check();
	void UnListen();
	const char *Host() { return _myhost; }
	int Port() { return _myport; }

private:
	void ProcessBytes(const char *source, NosuchSocketMemory* buff);
	int ProcessOneOscMessage( const char *source, NosuchSocketMemory* buff);
	int SlipBoundaries(char *p, int leng, char** pbegin, char** pend);
	NosuchSocketMemory* _oscmsg;
	NosuchSocket* mi_Socket;
	int _myport;
	const char *_myhost;
};

#endif