@echo off
rem Resolve paths relative to this script's own location (%~dp0) so the
rem prebuild step works regardless of the current working directory or
rem whether the current dir is on the executable search path.
python "%~dp0..\..\..\python\generateparams.py"
