from pythonosc import dispatcher
from pythonosc import osc_server
from traceback import format_exc
from time import sleep
import sys
import re
import time
from typing import List, Any

showalive = False
time0 = time.time()

def mycallback(address: str, *args: List[Any]) -> None:
	print("received ",address, "args=",args)

if __name__ == '__main__':

	if len(sys.argv) < 2:
		print("Usage: osclisten [-a] {port@addr}")
		sys.exit(1)

	if sys.argv[1] == "-a":
		showalive = True
		input_name = sys.argv[2]
	else:
		input_name = sys.argv[1]

	port = re.compile(".*@").search(input_name).group()[:-1]
	host = re.compile("@.*").search(input_name).group()[1:]

	print("host=",host," port=",port)

	dispatcher = dispatcher.Dispatcher()
	dispatcher.map("/*",mycallback)

	server = osc_server.ThreadingOSCUDPServer( (host,int(port)), dispatcher)

	server.serve_forever()
