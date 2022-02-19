#include "NosuchUtil.h"

int
SlipBoundaries(char *p, int leng, char** pbegin, char** pend)
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
