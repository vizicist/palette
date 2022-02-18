/*
 *  Created by Tim Thompson on 2/6/09.
 *  Copyright 2009 __MyCompanyName__. All rights reserved.
 *
 */

#include <math.h>
#include <string>
#include <sstream>
#include <intrin.h>
#include <float.h>

#include "NosuchUtil.h"
//  #include "NosuchOsc.h"
#include "NosuchSpace.h"
#include "NosuchException.h"
#include "NosuchAttr.h"
#include "NosuchEvent.h"

NosuchEvent::NosuchEvent(NosuchSpace* s) : NosuchAction(s) {
	_attrs = new NosuchEventAttrs(s);
	// NosuchDebug("NosuchEvent constructor called");
}

NosuchEvent::~NosuchEvent() {
	// NosuchDebug("NosuchEvent destructor called");
	delete _attrs;
}

NosuchEvent*
NosuchEvent::NewBangEvent(NosuchSpace* s, NosuchString target)
{
	NosuchEvent* e = new NosuchEvent(s);
	e->_attrs->Set("type","bang");
	e->_attrs->Set("target",target);
	return e;
}