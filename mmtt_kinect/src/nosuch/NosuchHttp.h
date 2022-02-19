/*
	Space Manifold - a variety of tools for Kinect and FreeFrame

	Copyright (c) 2011-2012 Tim Thompson <me@timthompson.com>

	Permission is hereby granted, free of charge, to any person obtaining
	a copy of this software and associated documentation files
	(the "Software"), to deal in the Software without restriction,
	including without limitation the rights to use, copy, modify, merge,
	publish, distribute, sublicense, and/or sell copies of the Software,
	and to permit persons to whom the Software is furnished to do so,
	subject to the following conditions:

	The above copyright notice and this permission notice shall be
	included in all copies or substantial portions of the Software.

	Any person wishing to distribute modifications to the Software is
	requested to send the modifications to the original developer so that
	they can be incorporated into the canonical version.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
	EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
	MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
	IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR
	ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF
	CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
	WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

#ifndef NSHTTP_H
#define NSHTTP_H

#include "cJSON.h"
#include <list>

#define REQUEST_GET 1
#define REQUEST_POST 2

class NosuchSocketConnection;
class NosuchSocket;

class NosuchHttp {
public:
	NosuchHttp(int port, std::string htmldir, int timeout);
	NosuchHttp::~NosuchHttp();
	void Check();
	void SetHtmlDir(std::string d) { _htmldir = d; }
	void RespondToGetOrPost(NosuchSocketConnection*);
	void InitializeWebSocket(NosuchSocketConnection *kd);
	void CloseWebSocket(NosuchSocketConnection *kd);
	void WebSocketMessage(NosuchSocketConnection *kdata, std::string msg);
	void SendAllWebSocketClients(std::string msg);

	virtual std::string RespondToJson(const char *method, cJSON *params, const char *id) = 0;

private:

	void _init(std::string host, int port, int timeout);
	std::list<NosuchSocketConnection *> _WebSocket_Clients;
	void AddWebSocketClient(NosuchSocketConnection* conn);

	NosuchSocket* _listening_socket;
	std::string _htmldir;
};

#endif
