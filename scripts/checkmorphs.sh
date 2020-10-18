
tjt=/c/Users/tjt

nmorphs=`$tjt/go/src/github.com/vizicist/morph/vs2019/build/morph/x64/Debug/morph.exe -l | grep "serial.*=" | wc -l | sed -e 's/\r//' `

desiredmorphs=4
echo "nmorphs=$nmorphs"

if [ "$nmorphs" != "$desiredmorphs" ]
then
	echo "ERROR, DID NOT FIND $desiredmorphs MORPHS !!  REBOOTING IN 20 SECONDS!!!"
	sleep 20
	c:/windows/system32/shutdown.exe -t 0 -r -f
else
	echo "HURRAY !! Found $desiredmorphs Morphs !!"
fi
