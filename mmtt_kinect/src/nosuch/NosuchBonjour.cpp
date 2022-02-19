#include "NosuchDebug.h"

#include <stdio.h>
#include "ip/NetworkingUtils.h"

#include "dns_sd.h"

// static volatile int stopNow = 0;
// Note: the select() implementation on Windows (Winsock2) fails with any timeout much larger than this
#define LONG_TIME 100000000

typedef union {
    unsigned char b[2];
    unsigned short NotAnInteger;
} Opaque16;

static int operation;
static uint32_t opinterface = kDNSServiceInterfaceIndexAny;
static volatile int timeOut = LONG_TIME;
static DNSServiceRef tcpclient    = NULL;
static DNSServiceRef udpclient    = NULL;

static void DNSSD_API tcpreg_reply(DNSServiceRef sdref, const DNSServiceFlags flags, DNSServiceErrorType errorCode,
                                   const char *name, const char *regtype, const char *domain, void *context)
{
    (void)sdref;    // Unused
    (void)flags;    // Unused
    (void)context;  // Unused

    // printtimestamp();
    NosuchDebug("Got a reply for service %s.%s%s: ", name, regtype, domain);

    if (errorCode == kDNSServiceErr_NoError)
    {
        if (flags & kDNSServiceFlagsAdd) NosuchDebug("Name now registered and active\n");
        else NosuchDebug("Name registration removed\n");
        if (operation == 'A' || operation == 'U' || operation == 'N') timeOut = 5;
    }
    else if (errorCode == kDNSServiceErr_NameConflict)
    {
        NosuchDebug("Name in use, please choose another\n");
        exit(-1);
    }
    else
        NosuchDebug("Error %d\n", errorCode);
}

static void DNSSD_API udpreg_reply(DNSServiceRef sdref, const DNSServiceFlags flags, DNSServiceErrorType errorCode,
                                   const char *name, const char *regtype, const char *domain, void *context)
{
    (void)sdref;    // Unused
    (void)flags;    // Unused
    (void)context;  // Unused

    // printtimestamp();
    NosuchDebug("Got a reply for service %s.%s%s: ", name, regtype, domain);

    if (errorCode == kDNSServiceErr_NoError)
    {
        if (flags & kDNSServiceFlagsAdd) NosuchDebug("Name now registered and active\n");
        else NosuchDebug("Name registration removed\n");
        if (operation == 'A' || operation == 'U' || operation == 'N') timeOut = 5;
    }
    else if (errorCode == kDNSServiceErr_NameConflict)
    {
        NosuchDebug("Name in use, please choose another\n");
        exit(-1);
    }
    else
        NosuchDebug("Error %d\n", errorCode);

}

void bonjour_check(void) {

    if ( ! tcpclient ) {
        return;
    }
    int dns_sd_fd  = DNSServiceRefSockFD(tcpclient   );
    // int dns_sd_fd2 = client_pa ? DNSServiceRefSockFD(client_pa) : -1;
    int nfds = dns_sd_fd + 1;
    fd_set readfds;
    struct timeval tv;
    int result;

    // if (dns_sd_fd2 > dns_sd_fd) nfds = dns_sd_fd2 + 1;

    // while (!stopNow)
    // 	{
    // 1. Set up the fd_set as usual here.
    // This example client has no file descriptors of its own,
    // but a real application would call FD_SET to add them to the set here
    FD_ZERO(&readfds);

    // 2. Add the fd for our client(s) to the fd_set
    FD_SET(dns_sd_fd , &readfds);
    // if (client_pa) FD_SET(dns_sd_fd2, &readfds);

    // 3. Set up the timeout.
    tv.tv_sec  = timeOut;
    tv.tv_sec = 0;
    tv.tv_usec = 0;

    result = select(nfds, &readfds, (fd_set*)NULL, (fd_set*)NULL, &tv);
    if (result > 0) {
        DNSServiceErrorType err = kDNSServiceErr_NoError;
        if      (FD_ISSET(dns_sd_fd , &readfds)) err = DNSServiceProcessResult(tcpclient   );
        // else if (client_pa && FD_ISSET(dns_sd_fd2, &readfds)) err = DNSServiceProcessResult(client_pa);
        if (err) {
            NosuchDebug("DNSServiceProcessResult returned %d\n", err);
            // stopNow = 1;
        }
    }
    else if (result == 0) {
        // NosuchDebug("select() returned 0, timer expired!?\n");
    }
    else
    {
        NosuchDebug("select() returned %d errno %d %s\n", result, errno, strerror(errno));
        // if (errno != EINTR) stopNow = 1;
    }
    // }
}

void bonjour_setup() {

    NetworkInitializer networkInitializer_;

    uint16_t udpPort = 4444;
    uint16_t tcpPort = 4444;

    Opaque16 udpregisterPort = { { udpPort >> 8, udpPort & 0xFF } };

    DNSServiceErrorType err;
    Opaque16 tcpregisterPort = { { tcpPort >> 8, tcpPort & 0xFF } };

    static const char TXT[] = "\xC" "First String" "\xD" "Second String" "\xC" "Third String";

    err = DNSServiceRegister(&tcpclient, 0, opinterface,
    "TCPFFFF", "_looper._tcp", "", NULL, tcpregisterPort.NotAnInteger,
    sizeof(TXT)-1, TXT, tcpreg_reply, NULL);

    if ( err ) {
        NosuchDebug("Error in DNSServiceRegister, err=%ld\n",(long int) err);
    }

    err = DNSServiceRegister(&udpclient, 0, opinterface,
    "UDPFFFF", "_looper._udp", "", NULL, udpregisterPort.NotAnInteger,
    sizeof(TXT)-1, TXT, udpreg_reply, NULL);

    if ( err ) {
        NosuchDebug("Error in DNSServiceRegister, err=%ld\n",(long int) err);
    }

}
