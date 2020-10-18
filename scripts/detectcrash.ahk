; This script detects if Bidule has crashed, by looking for a window
; with a title "Bidule standalone version"
; If it finds such a window, it reboots

SetTitleMatchMode, RegEx
Loop
{
	ifWinExist, Bidule.*standalone.*version
	{
		Run msg * Bidule crash detected and we will reboot in 10 seconds
		sleep, 10000
		Run c:\local\bin\reboot.bat
	}
	ifWinExist, Microsoft.*Visual.*C.*Runtime.*Library
	{
		Run msg * tuio3d_morph crash detected and we will reboot in 10 seconds
		sleep, 10000
		Run c:\local\bin\reboot.bat
	}
	sleep, 10000
}
