import sys
import palette

if len(sys.argv) != 3:
    print("Usage: debug {debug-type} {onoff}")
    sys.exit(1)

dtype = sys.argv[1]
onoff = sys.argv[2]
palette.palette_api("global.debug","\"debug\": \""+dtype+"\", \"onoff\": \""+onoff+"\"" )
