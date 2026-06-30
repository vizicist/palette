import sys
import palette

if len(sys.argv) != 2:
    print("Usage: log {log-types}")
    sys.exit(1)

val = sys.argv[1]

palette.palette_api("global.set","\"name\":\"global.log\", \"value\": \""+val+"\"" )
