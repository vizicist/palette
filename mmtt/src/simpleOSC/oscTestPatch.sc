// bucherhausen, innsbruck. April 9, 2004.
// ixi software OSC communication client made in supercollider 3
// made for test purposes in communicating with python, pd, max and java

(
var w, text, textwin, sendbutt, remoteIP, iPbutt, remoteText, otherIP, otherNetAddr;
var portField, portButt, port, sendslider;

// default port and IP (change in textfields)
port = 9000;
otherIP = "127.0.0.1";
otherNetAddr = NetAddr(otherIP, port);

// send synth def to server
SynthDef(\tpulse, { arg out=0,freq=700,sawFreq=440.0; 
	Out.ar(out, SyncSaw.ar(freq,  sawFreq,0.1) * EnvGen.kr(Env.perc,1.0,doneAction: 2)) 
}).send(s);

w = SCWindow("ixi sc chat", Rect(38, 364, 340, 300)).front;
remoteIP = SCTextField(w, Rect(20, 40, 120, 20) );
remoteIP.value_("127.0.0.1"); // default
iPbutt = SCButton(w, Rect(180, 40, 60, 20));
iPbutt.states = [["set IP",Color.black,Color.clear]];
iPbutt.action_({
	remoteIP.defaultKeyDownAction(3.asAscii);
	otherIP = remoteIP.value;
	otherNetAddr = NetAddr(otherIP, port.asInteger);
	});

portField = SCTextField(w, Rect(20, 80, 120, 20));
portField.value_(7000); // default
portButt = SCButton(w, Rect(180, 80, 60, 20));
portButt.states = [["set port",Color.black,Color.clear]];
portButt.action_({
	portField.defaultKeyDownAction(3.asAscii);
	port = portField.value;
	otherNetAddr = NetAddr(otherIP, port.asInteger);
	});
	

text = SCTextField(w, Rect(20, 120, 220, 20) );
sendbutt = SCButton(w, Rect(250, 120, 60, 20));
sendbutt.states = [["send",Color.black,Color.clear]];
sendbutt.action_({
	text.defaultKeyDownAction(3.asAscii);
	otherNetAddr.sendMsg('/print', text.value); 

	});
	
sendslider = SCSlider(w, Rect(20, 160, 220, 20));
sendslider.action_({arg sl; var spec, out;
	spec = [200,2000].asSpec;
	out = spec.map(sl.value); 
	otherNetAddr.sendMsg('/print', out);
	});

remoteText = SCTextField(w, Rect(20, 220, 220, 20) );
SCStaticText(w, Rect(250, 220, 40, 20)).string_("other");

g = {arg time, responder, message;
	[time, responder, message].postln;
	y = Synth(\tpulse, [\freq, message.at(1).asInteger]);
	{remoteText.value = message.at(1).asString}.defer;
	};

OSCresponder(nil, '/test', g).add;

)



