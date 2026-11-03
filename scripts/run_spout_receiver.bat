@echo off
rem Center two previews below their labels in the 250px Performance preview area.
start "" /b py -3.13 "%~dp0..\python\spout_receiver.py" --sender "Avenue - Screen 3" --x -792 --y 841 --width 384 --height 216 --always-on-top --performance-only %*
start "" /b py -3.13 "%~dp0..\python\spout_receiver.py" --sender "Avenue - Screen 2" --x -392 --y 841 --width 384 --height 216 --always-on-top --preview-only --performance-only %*
