#include <string>
#include <sstream>
#include <intrin.h>
#include <float.h>

#include "PaletteAll.h"

PaletteOscInput::PaletteOscInput(PaletteHost* server, const char* host, int port) : NosuchOscMessageProcessor() {

	NosuchDebug(2,"PaletteOscInput constructor port=%d",port);
	_udp = new NosuchOscUdpInput(host,port,this);
	_server = server;
}

PaletteOscInput::~PaletteOscInput() {
	if ( _udp )
		delete _udp;
}

void
PaletteOscInput::ProcessOscMessage(std::string source, const osc::ReceivedMessage& m) {
	_server->ProcessOscMessage(source,m);
}

void
PaletteOscInput::Check() {
	if ( _udp )
		_udp->Check();
}

void
PaletteOscInput::UnListen() {
	if ( _udp )
		_udp->UnListen();
}

int
PaletteOscInput::Listen() {
	int e;
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
