@echo off
if x%1 == x oscutil.exe listen 3333
if not x%1 == x oscutil.exe listen %1
