/*
 *  Created by Tim Thompson on 2/6/09.
 *  Copyright 2009 __MyCompanyName__. All rights reserved.
 *
 */

#ifndef NOSUCH_EXCEPTION_H
#define NOSUCH_EXCEPTION_H

#include <exception>
#include <string>
#include <windows.h>
#include "NosuchUtil.h"

class NosuchException {
	char *_msg;
public:
	NosuchException( const char *fmt, ...);
	const char *message() { return _msg; }
};

#define SEH_STUFF_ONLY_USABLE_WHEN_COMPILING_FOR_SEH

#ifdef SEH_STUFF_ONLY_USABLE_WHEN_COMPILING_FOR_SEH

void SEH_To_Cplusplus ( unsigned int u, EXCEPTION_POINTERS *exp );

// register the translator so that all hardware exceptions
// will generate a C++ NosuchException
#define CATCH_NULL_POINTERS _set_se_translator( SEH_To_Cplusplus );

#else

#define CATCH_NULL_POINTERS

#endif



#endif
