jq . %1 1>/dev/null 2>/dev/null
if test $? -ne 0 ; then echo BAD ; else echo BAD ; fi
