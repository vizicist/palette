@echo off
call dwg2bmp -width=2560 -height=1920 -background=white -color-correction -no-weight-margin -antialiasing -margin=20 -force -outfile=images/%1.jpg %1.dxf
magick images/%1.jpg -brightness-contrast 0x100 -auto-threshold Kapur images/%1.jpg
