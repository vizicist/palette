#include "winsock.h"
#include "mmsystem.h"
#include "NosuchUtil.h"
#include "NosuchOscInput.h"
#include "NosuchOscUdpInput.h"
#include <stdexcept>

int
OscSocketError(const char *s)
{
    int e = WSAGetLastError();
    NosuchDebug("OscSocketError: %s e=%d",s,e);
    return e;
}

NosuchOscUdpInput::NosuchOscUdpInput(const char *host, int port, NosuchOscMessageProcessor* processor) : NosuchOscInput(processor) {
	NosuchDebug(2,"NosuchOscUdpInput constructor");
	_s = INVALID_SOCKET;
	_myhost = host;
	_myport = port;
}

NosuchOscUdpInput::~NosuchOscUdpInput() {
	NosuchDebug(1,"NosuchOscUdpInput destructor");
	if ( _s != INVALID_SOCKET ) {
		NosuchDebug("HEY!  _info._s is still set in NSosc destructor!?");
	}
}

int
NosuchOscUdpInput::Listen() {

    struct sockaddr_in sin;
    struct sockaddr_in sin2;
#ifdef _WIN32
    int sin2_len = sizeof(sin2);
#else
    socklen_t sin2_len = sizeof(sin2);
#endif

    DWORD nbio = 1;
    PHOSTENT phe;

	SOCKET s = socket(PF_INET, SOCK_DGRAM, 0);
    if ( s < 0 ) {
        NosuchDebug("_openListener error 1");
        return OscSocketError("unable to create socket");
    }
    sin.sin_family = AF_INET;
    // sin.sin_addr.s_addr = INADDR_ANY;

	if ( _myhost != NULL && strcmp(_myhost,"*") != 0 ) {
	    phe = gethostbyname(_myhost);
	    if (phe == NULL) {
	        // Every failure from here on closes the socket before returning.
	        // They all used to leak it, so a plugin retrying a listen that keeps
	        // failing leaked one descriptor per attempt.
	        int e = OscSocketError("unable to get hostname");
	        closesocket(s);
	        return e;
	    }
	    memcpy((struct sockaddr FAR *) &(sin.sin_addr),
	           *(char **)phe->h_addr_list, phe->h_length);
	    sin.sin_port = htons(_myport);
	} else {
		// Listen on all ip addresses
	    sin.sin_port = htons(_myport);
		sin.sin_addr.s_addr = INADDR_ANY;
	}

    if (  ioctlsocket(s,FIONBIO,&nbio) < 0 ) {
        NosuchDebug("_openListener error 2");
        int e = OscSocketError("unable to set socket to non-blocking");
        closesocket(s);
        return e;
    }
    if (bind(s, (LPSOCKADDR)&sin, sizeof (sin)) < 0) {
        int e = WSAGetLastError();
		if( e == WSAEADDRINUSE )
		{
			// Reported, not thrown. This used to throw std::runtime_error,
			// which went straight past the caller's error-return branch - and
			// Listen is called from a constructor on the plugin's own thread,
			// so a port already in use took the host down instead of being
			// handled. Every other failure here returns an error code.
			NosuchDebug("Palette: host=%s port=%d is already in use.",_myhost,_myport);
		}
		else
		{
			NosuchDebug("Palette: socket bind error: host=%s port=%d e=%d",_myhost,_myport,e);
        }
        closesocket(s);
        return e;
        // return OscSocketError("unable to bind socket");
    }
    if ( getsockname(s,(LPSOCKADDR)&sin2, &sin2_len) != 0 ) {
        int e = OscSocketError("unable to getsockname after bind");
        closesocket(s);
        return e;
    }
    // *myport = ntohs(sin2.sin_port);
    NosuchDebug(1,"Listening for OSC on UDP port %d@%s",_myport,_myhost);
    _s = s;
    return 0;
}

void
NosuchOscUdpInput::Check()
{
	if ( _s == INVALID_SOCKET )
		return;

    struct sockaddr_in sin;
#ifdef _WIN32
    int sin_len = sizeof(sin);
#else
    socklen_t sin_len = sizeof(sin);
#endif
    char buf[8096];

    // NosuchDebug("OscCheck!");
	long tm0 = timeGetTime();
	int toomany = 20;
	unsigned long toolong = tm0 + 1000;   // Stop processing if it takes longer than this
    for ( int cnt=0; cnt<toomany; cnt++ ) {
		if ( timeGetTime() >= toolong ) {
			NosuchDebug("OSC processing taking too long, Check returning early, cnt=%d  tm0=%ld now=%ld\n",cnt,tm0,timeGetTime());
			break;
		}
        int i = recvfrom(_s,buf,sizeof(buf),0,(LPSOCKADDR)&sin, &sin_len);
        if ( i <= 0 ) {
            int e = WSAGetLastError();
			switch (e) {
			case WSAENOTSOCK:
				NosuchDebug("NosuchOscUdpInput::Check e==WSAENOTSOCK");
				_s = INVALID_SOCKET;
				break;
			case WSAEWOULDBLOCK:
				break;
			default:
                NosuchDebug("Hmmm, B e=%d isn't EWOULDBLOCK or WSAENOTSOCK!?",e);
				break;
            }
            return;
        }
        // NosuchDebug("%ld: GOT recvfrom _myport=%d i=%d  cnt=%d",timeGetTime(),_myport,i,cnt);

		// One datagram must not be able to kill the host.
		//
		// osc::ReceivedPacket's constructor throws for a malformed packet, and
		// both it and the dispatch below used to sit outside any catch. Check
		// runs on the plugin's own pthread, and an exception escaping a pthread
		// entry function terminates the process - so a single stray or hostile
		// UDP packet on this port took Resolume down. Anything thrown for one
		// packet is now logged and that packet dropped; the rest keep flowing.
		try {
			osc::ReceivedPacket p( buf, i );
			ProcessReceivedPacket(inet_ntoa(sin.sin_addr),p);
		}
		catch (osc::Exception& e) {
			NosuchDebug("NosuchOscUdpInput::Check - ignoring malformed OSC packet from %s: %s",
				inet_ntoa(sin.sin_addr), e.what());
		}
		catch (std::exception& e) {
			NosuchDebug("NosuchOscUdpInput::Check - exception handling a packet from %s: %s",
				inet_ntoa(sin.sin_addr), e.what());
		}
		catch (...) {
			NosuchDebug("NosuchOscUdpInput::Check - unknown exception handling a packet from %s",
				inet_ntoa(sin.sin_addr));
		}
    }
    NosuchDebug(1,"NosuchOscUdpInput.Check: quiting early, too many packets at once (%d)\n", toomany);
    // The rest of the packets will be picked up by the next call to Check()
}

void
NosuchOscUdpInput::UnListen()
{
    NosuchDebug(1,"_oscUnlisten( _myport=%d)", _myport);
    closesocket(_s);
    _s = INVALID_SOCKET;
}
