@echo off
if x%1 == x oscutil.exe playback 3333 fourcircles.osc
if not x%1 == x oscutil.exe playback %1 %2
