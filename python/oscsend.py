import re
import sys
from pythonosc import osc_message_builder
from pythonosc import udp_client

if len(sys.argv) < 3:
	print("Usage: oscsend.py {port@host} {msg_addr} {msg_args}")
	sys.exit(1)

def containsAny(str, set):
	return 1 in [c in str for c in set]

if __name__ == '__main__':


	porthost = sys.argv[1]
	print("porthost=",porthost)
	port = re.compile(".*@").search(porthost).group()[:-1]
	host = re.compile("@.*").search(porthost).group()[1:]
	
	oscaddr = sys.argv[2]
	
	n = 3
	# oscmsg = osc_message_builder.OscMessageBuilder(address=oscaddr)
	oscmsg = []
	while n < len(sys.argv):
		s = sys.argv[n]
		if len(s) > 0 and s[0].isdigit():
			if containsAny(s,"."):
				oscmsg.append(float(s))
			else:
				oscmsg.append(int(s))
		else:
			oscmsg.append(s)
		n += 1
	
	r = udp_client.SimpleUDPClient(host,int(port))
	r.send_message(oscaddr,oscmsg)
	
