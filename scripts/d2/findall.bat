@echo off
bash -c "/usr/bin/find %1 -type f -print0 | xargs -0 grep %2 > /tmp/find$$.out ; vi /tmp/find$$.out ; rm -f /tmp/find$$.*"
