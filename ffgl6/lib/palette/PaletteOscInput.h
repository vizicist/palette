/*
 *  Created by Tim Thompson on 1/31/09.
 *  Copyright 2009 __MyCompanyName__. All rights reserved.
 *
 */

#ifndef PALETTE_OSC_INPUT
#define PALETTE_OSC_INPUT

// class NosuchOscTcpInput;
class NosuchOscUdpInput;
class PaletteHost;

class PaletteOscInput : public NosuchOscMessageProcessor {

public:
	PaletteOscInput(PaletteHost* server, const char *host, int port);
	~PaletteOscInput();
	void Check();
	int Listen();
	void UnListen();
	void ProcessOscMessage(std::string source, const osc::ReceivedMessage& m);

private:
	PaletteHost* _server;
	// NosuchOscTcpInput* _tcp;
	NosuchOscUdpInput* _udp;
};

#endif
