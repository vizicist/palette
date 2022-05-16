#include "NosuchUtil.h"
#include "NosuchOscInput.h"

#include "winerror.h"
#include "winsock.h"
#include "osc/OscOutboundPacketStream.h"
#include "osc/OscReceivedElements.h"

#include <iostream>
#include <fstream>
using namespace std;

NosuchOscInput::NosuchOscInput(NosuchOscMessageProcessor* p) {

    // _enabled = 0;
	NosuchDebug(2,"NosuchOscInput constructor");
	_processor = p;
}

NosuchOscInput::~NosuchOscInput() {

    NosuchDebug("NosuchOscInput destructor");
}

void
DebugOscMessage(std::string prefix, const osc::ReceivedMessage& m)
{
    const char *s = m.AddressPattern();
    const char *types = m.TypeTags();
	if ( types == NULL ) {
		types = "";
	}

	NosuchDebug("%s: %s ",prefix==""?"OSC message":prefix.c_str(),s);
    osc::ReceivedMessage::const_iterator arg = m.ArgumentsBegin();
    int arg_i;
    float arg_f;
    const char *arg_s;
    for ( const char *p=types; *p!='\0'; p++ ) {
        switch (*p) {
        case 'i':
            arg_i = (arg++)->AsInt32();
            NosuchDebug("   Arg (int) = %d",arg_i);
            break;
        case 'f':
            arg_f = (arg++)->AsFloat();
            NosuchDebug("   Arg (float) = %lf",arg_f);
            break;
        case 's':
            arg_s = (arg++)->AsString();
            NosuchDebug("   Arg (string) = %s",arg_s);
            break;
        case 'b':
            arg++;
            NosuchDebug("   Arg (blob) = ??");
            break;
        }
    }
}

void
NosuchOscInput::ProcessOscBundle( const char *source, const osc::ReceivedBundle& b )
{
    // ignore bundle time tag for now

    for( osc::ReceivedBundle::const_iterator i = b.ElementsBegin();
		i != b.ElementsEnd();
		++i ) {

		if( i->IsBundle() ) {
            ProcessOscBundle( source, osc::ReceivedBundle(*i) );
		} else {
	        _processor->ProcessOscMessage( source, osc::ReceivedMessage(*i) );
		}
    }
}

int
SendToUDPServer(char *serverhost, int serverport, const char *data, int leng)
{
    SOCKET s;
    struct sockaddr_in sin;
    int sin_len = sizeof(sin);
    int i;
    DWORD nbio = 1;
    PHOSTENT phe;

    phe = gethostbyname(serverhost);
    if (phe == NULL) {
        NosuchDebug("SendToUDPServer: gethostbyname(localhost) fails?");
        return 1;
    }
    s = socket(PF_INET, SOCK_DGRAM, 0);
    if ( s < 0 ) {
        NosuchDebug("SendToUDPServer: unable to create socket!?");
        return 1;
    }
    sin.sin_family = AF_INET;
    memcpy((struct sockaddr FAR *) &(sin.sin_addr),
           *(char **)phe->h_addr_list, phe->h_length);
    sin.sin_port = htons(serverport);

    i = sendto(s,data,leng,0,(LPSOCKADDR)&sin,sin_len);

    closesocket(s);
    return 0;
}

int
RegisterWithAServer(char *serverhost, int serverport, char *myhost, int myport)
{
    char buffer[1024];
    // NosuchDebug("RegisterWithServer, serverport=%d myport=%d",serverport, myport);
    osc::OutboundPacketStream p( buffer, sizeof(buffer) );
    p << osc::BeginMessage( "/registerclient" )
      << "localhost" << myport << osc::EndMessage;
    return SendToUDPServer(serverhost,serverport,p.Data(),(int)p.Size());
}

int
UnRegisterWithAServer(char *serverhost, int serverport, char *myhost, int myport)
{
    char buffer[1024];
    osc::OutboundPacketStream p( buffer, sizeof(buffer) );
    p << osc::BeginMessage( "/unregisterclient" )
      << "localhost" << myport << osc::EndMessage;
    return SendToUDPServer(serverhost,serverport,p.Data(),(int)p.Size());
}


