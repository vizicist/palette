#include <exception>
#include <stdint.h>
#include <stdarg.h>
#include <string>

#include "NosuchUtil.h"
#include "NosuchException.h"

#define EXCEPTION_BUFFER_SIZE 8096
static char NosuchExceptionBuffer[EXCEPTION_BUFFER_SIZE];

NosuchException::NosuchException( const char *fmt, ...) {
	_msg = NosuchExceptionBuffer;
	va_list args;

	NosuchErrorOutput("NosuchException::NosuchException, message follows\n");
	va_start(args, fmt);
	vsnprintf_s(_msg,EXCEPTION_BUFFER_SIZE,EXCEPTION_BUFFER_SIZE,fmt,args);

	size_t lng = strlen(_msg);
	if ( lng > 0 && _msg[lng-1] == '\n' )
		_msg[lng-1] = '\0';

	NosuchErrorOutput(_msg);

	va_end(args);
}

#ifdef SEH_STUFF_ONLY_USABLE_WHEN_COMPILING_FOR_SEH
// the translator function 
void SEH_To_Cplusplus ( unsigned int u, EXCEPTION_POINTERS *exp ) { 
		int code = exp->ExceptionRecord->ExceptionCode;
		if ( code == EXCEPTION_ACCESS_VIOLATION ) {
			NosuchErrorOutput("NULL POINTER DEREFERENCE!! throwing NosuchException\n");
			throw NosuchException("NULL POINTER DEREFERENCE!! (NosuchException translated from SEH exception)");
		} else {
			throw NosuchException("NosuchException translated from SEH exception, code=%d",code);
		}

} 
#endif
