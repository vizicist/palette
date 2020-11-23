#ifndef NSOSCINPUT_H
#define NSOSCINPUT_H

#include "osc/OscReceivedElements.h"
#include "NosuchUtil.h"

// #define NTHEVENTSERVER_PORT 1384
// Every so many milliseconds, we re-register with the Nth Server
#define NTHEVENTSERVER_REREGISTER_MILLISECONDS 3000

void DebugOscMessage(std::string prefix, const osc::ReceivedMessage& m);

#define RAD2DEG(r) ((r)*360.0/(2.0*M_PI))
#define PI2 ((float)(2.0*M_PI))

class NosuchOscMessageProcessor {
public:
	virtual void ProcessOscMessage( std::string source, const osc::ReceivedMessage& m) = 0;
};

class NosuchOscInput {

public:
	NosuchOscInput (NosuchOscMessageProcessor* p);
	virtual ~NosuchOscInput ();

	void ProcessOscBundle( std::string source, const osc::ReceivedBundle& b);

	void ProcessReceivedPacket(std::string source, osc::ReceivedPacket& rp) {
	    if( rp.IsBundle() )
	        ProcessOscBundle( source, osc::ReceivedBundle(rp) );
	    else
	        _processor->ProcessOscMessage( source, osc::ReceivedMessage(rp) );
	}

	NosuchOscMessageProcessor* _processor;
};

#endif
