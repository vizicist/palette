#Requires AutoHotkey v2.0

monitorCount := SysGet(80)

MsgBox("Number of monitors: " monitorCount)

if ( monitorCount < 2 ) {

    MsgBox("Trying to connect Miracast display")
    Send '#k'
    Sleep 1000
    Send "{Tab}"
    Sleep 1000
    Send "{Enter}"
    Sleep 1000
    Send "{Esc}"
}
