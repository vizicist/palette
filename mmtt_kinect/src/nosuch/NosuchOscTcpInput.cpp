#include "winsock2.h"
#include "osc/OscReceivedElements.h"
#include "NosuchUtil.h"
#include "NosuchSocket.h"
#include "NosuchOscInput.h"
#include "NosuchOscTCPInput.h"

NosuchOscTcpInput::NosuchOscTcpInput(const char *host, int port, NosuchOscMessageProcessor* processor) : NosuchOscInput(processor) {
	NosuchDebug(2,"NosuchOscTcpInput constructor");
	_oscmsg = new NosuchSocketMemory(128);
	mi_Socket = new NosuchSocket();
	_myhost = host;
	_myport = port;
}

NosuchOscTcpInput::~NosuchOscTcpInput() {
	delete _oscmsg;
	delete mi_Socket;
}

int
NosuchOscTcpInput::Listen() {
    mi_Socket->Listen(0, _myport, 0, 0);
    return 0;
}

void
NosuchOscTcpInput::Check()
{
    NosuchSocketMemory* pi_RecvMem;
    SOCKET  h_Socket;
	NosuchSocketConnection *h_connection;
    DWORD u32_Event, u32_IP, u32_Read, u32_Sent;
	USHORT u16_Port_source;

    DWORD u32_Err = mi_Socket->ProcessEvents(&u32_Event, &u32_IP,
		&u16_Port_source, &h_Socket, &h_connection,
		&pi_RecvMem, &u32_Read,  &u32_Sent);

    if (u32_Err == ERROR_TIMEOUT) // 50 ms interval has elapsed
        return;

    if (u32_Event) // ATTENTION: u32_Event may be == 0 -> do nothing.
    {
        // NosuchDebug("u32_Event = %d\n",u32_Event);
        if (u32_Event & FD_READ && pi_RecvMem) // pi_RecvMem may be NULL if an error occurred!!
        {
            // NosuchDebug("SHOULD BE PROCESSING received data!  u32_Read=%d\n",u32_Read);
			struct in_addr a;
			a.S_un.S_addr = u32_IP;
			std::string source = NosuchSnprintf("%d@%s",u16_Port_source,inet_ntoa(a));
            ProcessBytes(source.c_str(),pi_RecvMem);
        }
    }

    if (u32_Err)
    {
        NosuchDebug("u32_Err = %d\n",u32_Err);
        // mi_Socket->Close() has been called -> don't print this error message
        if (u32_Err == WSAENOTCONN)
            return;

        // An error normally means that the socket has a problem -> abort the loop.
        // A few errors should not abort the processing:
        if (u32_Err != WSAECONNABORTED && // e.g. after the other side was killed in TaskManager
                u32_Err != WSAECONNRESET   && // Connection reset by peer.
                u32_Err != WSAECONNREFUSED && // FD_ACCEPT with already 62 clients connected
                u32_Err != WSAESHUTDOWN)      // Sending data to a socket just in the short timespan
            return;                        //   between shutdown() and closesocket()
    }
}

int
NosuchOscTcpInput::SlipBoundaries(char *p, int leng, char** pbegin, char** pend)
{
    int bytesleft = leng;
    int found = 0;

    *pbegin = 0;
    *pend = 0;
    // int pn = (*p & 0xff);
    // NosuchDebug("SLIPBOUNDARIES pn=%d SLIP_END=%d\n",pn,SLIP_END);
    if ( IS_SLIP_END(*p) ) {
        *pbegin = p++;
        bytesleft--;
        found = 1;
    } else {
        // Scan for next unescaped SLIP_END
        p++;
        bytesleft--;
        while ( !found && bytesleft > 0 ) {
            if ( IS_SLIP_END(*p) && ! IS_SLIP_ESC(*(p-1)) ) {
                *pbegin = p;
                found = 1;
            }
            p++;
            bytesleft--;
        }
    }
    if ( ! found ) {
        return 0;
    }
    // We've got the beginning of a message, now look for
    // the end.
    found = 0;
    while ( !found && bytesleft > 0 ) {
        if ( IS_SLIP_END(*p) && ! IS_SLIP_ESC(*(p-1)) ) {
            *pend = p;
            found = 1;
        }
        p++;
        bytesleft--;
    }
    return found;
}

void
NosuchOscTcpInput::ProcessBytes(const char *source,	NosuchSocketMemory* buff)
{
    while ( ProcessOneOscMessage(source,buff) ) {
    }
}

int
NosuchOscTcpInput::ProcessOneOscMessage( const char *source, NosuchSocketMemory* buff)
{
    char *p = buff->GetBuffer();
    int nbytes = buff->GetLength();
    // NosuchDebug("ProcessBytes, buffer is %d bytes long\n",nbytes);
    // for ( int n=0; n<nbytes; n++ ) {
    // 	NosuchDebug("  p[%d] = 0x%02x\n",n,p[n]);
    // }
    char* pbegin;
    char* pend;
    if ( SlipBoundaries(p,nbytes,&pbegin,&pend) == 0 ) {
        return 0;
    }
    int oscsize = (int)(pend - pbegin - 1);
    // NosuchDebug("SLIPBOUNDARIES found it! leng = %d\n",oscsize);
    char *oscp = _oscmsg->GetBuffer();
    if ( _oscmsg->GetLength() != 0 ) {
        NosuchDebug("HEY, _oscmsg isn't empty!?");
        _oscmsg->DeleteLeft(_oscmsg->GetLength());
    }
    if ( ! IS_SLIP_END(*pbegin) || ! IS_SLIP_END(*pend) ) {
        // This indicates SlipBoundaries isn't doing its job.
        NosuchDebug("HEY! pbegin/pend don't have SLIP_END !??");
        return 0;
    }
    p = pbegin+1;
    int bytesleft = oscsize;
    while ( bytesleft > 0 ) {
        if ( IS_SLIP_ESC(*p) && bytesleft>1 && IS_SLIP_ESC2(*(p+1)) ) {
            _oscmsg->Append(p,1);
            p += 2;
            bytesleft -= 2;
        } else if ( IS_SLIP_ESC(*p) && bytesleft>1 && IS_SLIP_END(*(p+1)) ) {
            _oscmsg->Append(p+1,1);
            p += 2;
            bytesleft -= 2;
        } else {
            _oscmsg->Append(p,1);
            p += 1;
            bytesleft -= 1;
        }
    }
    buff->DeleteLeft(oscsize+2);
    // NosuchDebug("Final size of _oscmsg = %d, oscsize=%d\n",_oscmsg->GetLength(),oscsize);

    osc::ReceivedPacket rp( _oscmsg->GetBuffer(), _oscmsg->GetLength() );
	ProcessReceivedPacket(source,rp);

    _oscmsg->DeleteLeft(_oscmsg->GetLength());
    return 1;
}


void
NosuchOscTcpInput::UnListen()
{
    mi_Socket->Close();
}

