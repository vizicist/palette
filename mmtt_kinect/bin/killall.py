from subprocess import call, Popen
from time import sleep

def killtask(nm):
	call(["c:/windows/system32/taskkill","/f","/im",nm])

killtask("mmtt.exe")
killtask("mmtt_depth.exe")
killtask("mmtt_pcx.exe")
killtask("mmtt_kinetic.exe")
killtask("mmtt_kinect.exe")
killtask("loopMIDI.exe")
killtask("PlogueBidule_x64.exe")
killtask("Avenue.exe")
killtask("Arena.exe")

