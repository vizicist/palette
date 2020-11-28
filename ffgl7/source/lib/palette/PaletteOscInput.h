#pragma once

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
