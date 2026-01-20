set PATH=c:\Program Files\QCAD;c:\Program Files\ImageMagick;%PATH%
@echo off
for %%i in (SPP_*.dxf) do convertone %%~ni
