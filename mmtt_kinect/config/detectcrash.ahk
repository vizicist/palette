; This script detects if Bidule has crashed, by looking for a window
; with a title "Bidule standalone version"
; If it finds such a window, it reboots

SetTitleMatchMode, RegEx
Loop
{
	ifWinExist, Bidule.*standalone.*version
	{
		Run c:\local\manifold\bin\reboot.bat
	}
	sleep, 10000
}
