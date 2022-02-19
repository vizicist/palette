#ifndef NSOSCINPUT_H
#define NSOSCINPUT_H

#define _USE_MATH_DEFINES

#include <list>

#include "osc/OscReceivedElements.h"
#include <math.h>
#include "NosuchUtil.h"

// #define NTHEVENTSERVER_PORT 1384
// Every so many milliseconds, we re-register with the Nth Server
#define NTHEVENTSERVER_REREGISTER_MILLISECONDS 3000

void DebugOscMessage(std::string prefix, const osc::ReceivedMessage& m);

#define RAD2DEG(r) ((r)*360.0/(2.0*M_PI))
#define PI2 ((float)(2.0*M_PI))

class NosuchOscMessageProcessor {
public:
	virtual void ProcessOscMessage( const char *source, const osc::ReceivedMessage& m) = 0;
};

class NosuchOscInput {

public:
	NosuchOscInput (NosuchOscMessageProcessor* p);
	virtual ~NosuchOscInput ();

	void ProcessOscBundle( const char *source, const osc::ReceivedBundle& b);

	void ProcessReceivedPacket(const char *source, osc::ReceivedPacket& rp) {
	    if( rp.IsBundle() )
	        ProcessOscBundle( source, osc::ReceivedBundle(rp) );
	    else
	        _processor->ProcessOscMessage( source, osc::ReceivedMessage(rp) );
	}

	// virtual int Listen() = 0;
	// virtual void Check() = 0;
	// virtual void UnListen() = 0;

// protected:

	// int _myport;
	// const char *_myhost;

	// int _enabled;
	NosuchOscMessageProcessor* _processor;
};

#endif
