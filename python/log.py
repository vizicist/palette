import sys
import palette

if len(sys.argv) != 2:
    print("Usage: log {log-types}")
    sys.exit(1)

val = sys.argv[1]

palette.palette_api("engine.set","\"name\":\"engine.log\", \"value\": \""+val+"\"" )
