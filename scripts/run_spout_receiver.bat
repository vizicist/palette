@echo off
py -3.13 "%~dp0..\python\spout_receiver.py" --sender "Avenue - Screen 2" --x -500 --y 1000 --width 320 --height 180 --always-on-top %*
