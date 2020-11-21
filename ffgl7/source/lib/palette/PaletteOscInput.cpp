#if 0

#include <string>
#include <sstream>
#include <intrin.h>
#include <float.h>

#include "PaletteAll.h"
#include "PaletteOscInput.h"
#include "NosuchOscUdpInput.h"

PaletteOscInput::PaletteOscInput(PaletteHost* server, const char* host, int port) : NosuchOscMessageProcessor() {

	NosuchDebug(2,"PaletteOscInput constructor port=%d",port);
	// _seq = -1;
	// _tcp = new NosuchOscTcpInput(host,port);
	// _tcp = NULL;
	_udp = new NosuchOscUdpInput(host,port,this);
	_server = server;
}

PaletteOscInput::~PaletteOscInput() {
#if 0
	if ( _tcp )
		delete _tcp;
#endif
	if ( _udp )
		delete _udp;
}

void
PaletteOscInput::ProcessOscMessage(std::string source, const osc::ReceivedMessage& m) {
	_server->ProcessOscMessage(source,m);
}

void
PaletteOscInput::Check() {
#if 0
	if ( _tcp )
		_tcp->Check();
#endif
	if ( _udp )
		_udp->Check();
}

void
PaletteOscInput::UnListen() {
#if 0
	if ( _tcp )
		_tcp->UnListen();
#endif
	if ( _udp )
		_udp->UnListen();
}

int
PaletteOscInput::Listen() {
	int e;
#if 0
	if ( _tcp ) {
		if ( (e=_tcp->Listen()) != 0 ) {
			if ( e == WSAEADDRINUSE ) {
				NosuchErrorOutput("TCP port/address (%d/%s) is already in use?",_tcp->Port(),_tcp->Host());
			} else {
				NosuchErrorOutput("Error in _tcp->Listen = %d\n",e);
			}
			return e;
		}
	}
#endif
	if ( _udp ) {
		if ( (e=_udp->Listen()) != 0 ) {
			if ( e == WSAEADDRINUSE ) {
				NosuchErrorOutput("UDP port/address (%d/%s) is already in use?",_udp->Port(),_udp->Host());
			} else {
				NosuchErrorOutput("Error in _udp->Listen = %d\n",e);
			}
			return e;
		}
	}
	return 0;
}
#endif
