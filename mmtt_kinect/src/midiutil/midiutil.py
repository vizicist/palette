"""
This module provides a high-level interface to MIDI things.
"""

import sys
import time
import traceback
import thread
import threading
import copy
import string
import nosuch.midifile

from threading import Thread,Lock
from math import sqrt
from ctypes import *
from time import sleep
from traceback import format_exc
from array import array

from nosuch.midifile import *

EOX = 0xf7

DEFAULT_CHANNEL = 1
DEFAULT_PITCH = 64
DEFAULT_VELOCITY = 64
DEFAULT_DURATION = 1000

class BaseEvent:
	def __init__(self):
		self.time = 0.0

class MidiBaseHardware:

	def __init__(self):
		pass

	def time_now(self):
		return time.time()

	def input_devices(self):
		arr = []
		return arr

	def output_devices(self):
		arr = []
		return arr

	def get_input(self,input_name=None):
		return MidiBaseHardwareInput(input_name)

	def get_output(self,output_name=None):
		return MidiBaseHardwareOutput(output_name)


class MidiEvent(BaseEvent):

	def __init__(self,midimsg,tm=0.0):
		BaseEvent.__init__(self)
		self.midimsg = midimsg
		self.time = tm

	def __str__(self):
		return self.to_xml()

	def to_xml(self):
		return '<event time="%.3f">%s</event>' % (self.time,self.midimsg.to_xml())

	def to_osc(self):
		# I suppose this should construct a bundle with a timetag
		return self.midimsg.to_osc()
		
class TimerEvent(BaseEvent):
	def __init__(self, tm, func, *args, **kwargs):
		BaseEvent.__init__(self)
		self.time = tm
		self.func = func
		self.args = args
		self.kwargs = kwargs
	
	def invoke(self, tm):
		result = self.func(tm, self.time, *self.args, **self.kwargs)
		if isinstance(result, tuple):
			self.time = result[0]
			self.args = result[1]
			self.kwargs = len(result) > 2 and result[2] or {}
		else:
			self.time = result
			self.args = []
			self.kwargs = {}
		return self.time

# Base classes

class MidiMsg:
	"""
	A single MIDI message or note
	"""

	def __init__(self,name):
		self.name = name

	def __str__(self):
		return self.to_xml();

	def common_xml(self):
		if hasattr(self,"device"):
			return 'midi_%s devindex="%d"' % (self.name,self.device.index)
		else:
			return 'midi_%s' % (self.name)

	@staticmethod
	def from_xml(node):
		attrs = node.attributes
		if node.nodeName == "midi_controller":
			c = int(attrs.get("controller").nodeValue)
			v = int(attrs.get("value").nodeValue)
			ch = int(attrs.get("channel").nodeValue)
			return Controller(controller=c,value=v,channel=ch)
		if node.nodeName == "midi_program":
			ch = int(attrs.get("channel").nodeValue)
			p = int(attrs.get("program").nodeValue)
			return Program(program=p,channel=ch)
		if node.nodeName == "midi_noteon":
			p = int(attrs.get("pitch").nodeValue)
			ch = int(attrs.get("channel").nodeValue)
			v = int(attrs.get("velocity").nodeValue)
			return NoteOn(pitch=p,channel=ch,velocity=v)
		if node.nodeName == "midi_noteoff":
			p = int(attrs.get("pitch").nodeValue)
			ch = int(attrs.get("channel").nodeValue)
			v = int(attrs.get("velocity").nodeValue)
			return NoteOff(pitch=p,channel=ch,velocity=v)
		if node.nodeName == "midi_pressure":
			p = int(attrs.get("pitch").nodeValue)
			ch = int(attrs.get("channel").nodeValue)
			v = int(attrs.get("pressure").nodeValue)
			return Pressure(pitch=p,pressure=v,channel=ch)
		if node.nodeName == "midi_channelpressure":
			v = int(attrs.get("pressure").nodeValue)
			ch = int(attrs.get("channel").nodeValue)
			return ChannelPressure(pressure=v,channel=ch)
		if node.nodeName == "midi_pitchbend":
			v = int(attrs.get("value").nodeValue)
			ch = int(attrs.get("channel").nodeValue)
			return PitchBend(value=v,channel=ch)
		if node.nodeName == "midi_realtime":
			b = int(attrs.get("value").nodeValue)
			return RealTime(b)
		if node.nodeName == "midi_sysex":
			lng = int(attrs.get("length").nodeValue)
			b = attrs.get("bytes").nodeValue
			print("from_xml needs to implement midi_sysex")
			return None
		raise Exception, "Unrecognized node in MidiMsg.from_xml: "+node.nodeName

class ChanMsg(MidiMsg):

	def __init__(self,name,channel=1):
		MidiMsg.__init__(self,name)
		self.channel = channel

	def __eq__(self,other):
		if not isinstance(other,ChanMsg):
			return False
		return self.channel == other.channel

# Concrete MIDI message classes

class RealTime(MidiMsg):

	def __init__(self,b):
		MidiMsg.__init__(self,"realtime")
		self.onebyte = b

	def __eq__(self,other):
		if not isinstance(other,RealTime):
			return False
		return self.onebyte == other.onebyte

	def to_xml(self):
		return '<%s value="%d"/>' % (self.common_xml(),self.onebyte)

	def to_osc(self):
		return ("/midi/realtime",[self.onebyte])

	def write(self,out):
		out.write_short(self.onebyte)

class SysEx(MidiMsg):

	def __init__(self,b = None):
		MidiMsg.__init__(self,"sysex")
		if b == None:
			self.bytes = []
		else:
			self.bytes = [b]

	def append(self,b):
		self.bytes.append(b)

	def __eq__(self,other):
		if not isinstance(other,SysEx):
			return False
		print "Warning: __eq__ for SysEx needs implementing, always returns true"
		return True

	def to_hex(self,c):
		return "%02x" % c

	def to_xml(self):
		s = ''.join(map(self.to_hex,self.bytes))
		return '<%s length="%d" bytes="0x%s"/>' % (
			self.common_xml(),len(self.bytes),s)

	def to_osc(self):
		# return ("/midi/sysex",[0])
		return None

	def write(self,out):
		out.write_sysex(self.bytes)

class NoteOn(ChanMsg):

	def __init__(self,pitch,velocity=DEFAULT_VELOCITY,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"noteon",channel=channel)
		self.pitch = Midi.bound_value(int(pitch))
		self.velocity = Midi.bound_value(int(velocity))

	def __eq__(self,other):
		if not isinstance(other,NoteOn):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.pitch == other.pitch and self.velocity == other.velocity

	def to_xml(self):
		return '<%s channel="%d" pitch="%d" velocity="%d"/>' % (
			self.common_xml(),self.channel,self.pitch,self.velocity)

	def to_osc(self):
		return ("/midi/noteon",[self.channel,self.pitch,self.velocity])

	def write(self,out):
		# print "NoteOn written pitch=%d vel=%d" % (self.pitch,self.velocity)
		out.write_short(0x90 + (self.channel-1),self.pitch,self.velocity)

class NoteOff(ChanMsg):

	def __init__(self,pitch,velocity=DEFAULT_VELOCITY,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"noteoff",channel=channel)
		self.pitch = Midi.bound_value(int(pitch))
		self.velocity = Midi.bound_value(int(velocity))

	def __eq__(self,other):
		if not isinstance(other,NoteOff):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.pitch == other.pitch and self.velocity == other.velocity

	def to_xml(self):
		return '<%s channel="%d" pitch="%d" velocity="%d"/>' % (
			self.common_xml(),self.channel,self.pitch,self.velocity)

	def to_osc(self):
		return ("/midi/noteoff",[self.channel,self.pitch,self.velocity])

	def write(self,out):
		# print "NoteOff written pitch=%d vel=%d" % (self.pitch,self.velocity)
		out.write_short(0x80 + (self.channel-1),self.pitch,self.velocity)

class Pressure(ChanMsg):

	def __init__(self,pitch,pressure,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"pressure",channel=channel)
		self.pitch = Midi.bound_value(int(pitch))
		self.pressure = int(pressure)

	def __eq__(self,other):
		if not isinstance(other,Pressure):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.pitch == other.pitch and \
				self.pressure == other.pressure

	def to_xml(self):
		return '<%s channel="%d" pitch="%d" pressure="%d"/>' % (
			self.common_xml(),self.channel,self.pitch,self.pressure)

	def to_osc(self):
		return ("/midi/pressure",[self.channel,self.pitch,self.pressure])

	def write(self,out):
		out.write_short(0xa0 + (self.channel-1),self.pitch,self.pressure)

class Controller(ChanMsg):

	def __init__(self,controller,value,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"controller",channel=channel)
		self.controller = int(controller)
		self.value = int(value)

	def __eq__(self,other):
		if not isinstance(other,Controller):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.controller == other.controller and \
				self.value == other.value

	def to_xml(self):
		return '<%s channel="%d" controller="%d" value="%d"/>' % (
			self.common_xml(),self.channel,self.controller,self.value)

	def to_osc(self):
		return ("/midi/controller",[self.channel,self.controller,self.value])

	def write(self,out):
		out.write_short(0xb0 + (self.channel-1),self.controller,self.value)

class PitchBend(ChanMsg):

	def __init__(self,value,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"pitchbend",channel=channel)
		self.value = int(value)

	def __eq__(self,other):
		if not isinstance(other,PitchBend):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.value == other.value

	def to_xml(self):
		return '<%s channel="%d" value="%d"/>' % (
			self.common_xml(),self.channel,self.value)

	def to_osc(self):
		return ("/midi/pitchbend",[self.channel,self.value])

	def write(self,out):
		b0 = self.value & 0x7f
		b1 = (self.value>>7) & 0x7f
		out.write_short(0xe0 + (self.channel-1),b0,b1)

class Program(ChanMsg):

	def __init__(self,program,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"program",channel=channel)
		self.program = int(program)

	def __eq__(self,other):
		if not isinstance(other,Program):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.program == other.program

	def to_xml(self):
		return '<%s channel="%d" program="%d" />' % (
			self.common_xml(),self.channel,self.program)

	def to_osc(self):
		return ("/midi/program",[self.channel,self.program])

	def write(self,out):
		out.write_short(0xc0 + (self.channel-1),self.program-1)

class ChannelPressure(ChanMsg):

	def __init__(self,pressure,channel=DEFAULT_CHANNEL):
		ChanMsg.__init__(self,"channelpressure",channel=channel)
		self.pressure = int(pressure)

	def __eq__(self,other):
		if not isinstance(other,ChannelPressure):
			return False
		if not ChanMsg.__eq__(self,other):
			return False
		return self.pressure == other.pressure

	def to_xml(self):
		return '<%s channel="%d" pressure="%d"/>' % (
			self.common_xml(),self.channel,self.pressure)

	def to_osc(self):
		return ("/midi/channelpressure",[self.channel,self.pressure])

	def write(self,out):
		out.write_short(0xd0 + (self.channel-1),self.pressure)

# Sequenced things have a clocks value

class SequencedEvent:

	def __init__(self,clocks):
		self.clocks = float(clocks)

class SequencedNote(SequencedEvent):
	"""
	A note with a duration and release velocity.
	"""

	def __init__(self,pitch,velocity=DEFAULT_VELOCITY,channel=DEFAULT_CHANNEL,clocks=0,duration=DEFAULT_DURATION,releasevelocity=0):
		SequencedEvent.__init__(self,clocks)
		self.pitch = Midi.bound_value(int(pitch))
		self.velocity = Midi.bound_value(int(velocity))
		self.channel = channel
		self.duration = float(duration)
		self.releasevelocity = Midi.bound_value(int(releasevelocity))

	def __str__(self):
		return "SequencedNote(clocks=%f pitch=%d channel=%d velocity=%d releasevelocity=%d duration=%f)" % (self.clocks,self.pitch,self.channel,self.velocity,self.releasevelocity,self.duration)

class SequencedMidiMsg(SequencedEvent):

	def __init__(self,msg,clocks=0):
		SequencedEvent.__init__(self,clocks)
		self.msg = msg

	def __str__(self):
		return "SequencedMidiMsg(clocks=%f msg=%s)" % (self.clocks,str(self.msg))

# Scheduled things have a time value and an output device

class ScheduledMidiMsg:

	def __init__(self,time,msg,output = None):
		self.output = output
		self.time = float(time)
		self.msg = msg

	def __str__(self):
		if self.output == None:
			return "ScheduledMidiMsg(time=%f msg=%s)" % (self.time,str(self.msg))
		else:
			return "ScheduledMidiMsg(time=%f output=%s msg=%s)" % (self.time,self.output.name,str(self.msg))

# Phrase things
class PhraseMidiFileCallback:

	# Currently, this combines all the tracks into one phrase.
	# At some point, we should make the tracks separate

	def __init__(self,p):
		self.p = p

	def noteon(self, clocks, trackindex, c, p, v):
		m = NoteOn(pitch=p, channel=c, velocity=v)
		self.p.append(SequencedMidiMsg(m,clocks=clocks))

	def noteoff(self, clocks, trackindex, c, p, v):
		m = NoteOff(pitch=p, channel=c, velocity=v)
		self.p.append(SequencedMidiMsg(m,clocks=clocks))

	def program(self, clocks, trackindex, c, p):
		m = Program(channel=c, program=p)
		self.p.append(SequencedMidiMsg(m,clocks=clocks))

	def chanpressure(self, clocks, trackindex, c, p):
		m = ChannelPressure(channel=c, pressure=p)
		self.p.append(SequencedMidiMsg(m,clocks=clocks))

	def controller(self, clocks, trackindex, c, ct, cv):
		m = Controller(channel=c, controller=ct, value=cv)
		self.p.append(SequencedMidiMsg(m,clocks=clocks))

	def pitchbend(self, clocks, trackindex, c, v):
		m = PitchBend(channel=c, value=v)
		self.p.append(SequencedMidiMsg(m,clocks=clocks))

class Phrase(list):
	"""
	A time-ordered list containing SequencedEvents
	"""

	def __init__(self):
		array.__init__(self,)

	def append(self,e):
		if not issubclass(e.__class__,SequencedEvent):
			raise Exception,"Phrases can only take SequencedEvent objects!"
		list.append(self,e);

	# def length(self):
	# 	return len(self.events);

	@staticmethod
	def fromMidiFile(path):
		p = Phrase()
		f = MidiFile(PhraseMidiFileCallback(p))
		f.open(path)
		f.read()
		f.close()
		return p

	@staticmethod
	def merged(phraseIterable):
		"""
		Iterate events from one or more phrases, ordered by time.
		
		@param phraseIterable: a sequence or iterable of Phrase objects
		"""
		import heapq
		from itertools import imap
		def _merge(*subsequences):
			# Python Cookbook Recipe 19.14
			
			# prepare a priority queue whose items are pairs of the form
			# (current-value, iterator), one each per (non-empty) subsequence
			heap = [  ]
			for subseq in subsequences:
				iterator = iter(subseq)
				for current_value in iterator:
					# subseq is not empty, therefore add this subseq's pair
					# (current-value, iterator) to the list
					heap.append((current_value, iterator))
					break
			# make the priority queue into a heap
			heapq.heapify(heap)
			while heap:
				# get and yield lowest current value (and corresponding iterator)
				current_value, iterator = heap[0]
				yield current_value
				for current_value in iterator:
					# subseq is not finished, therefore add this subseq's pair
					# (current-value, iterator) back into the priority queue
					heapq.heapreplace(heap, (current_value, iterator))
					break
				else:
					# subseq has been exhausted, therefore remove it from
					# the queue
					heapq.heappop(heap)
		
		decoratedEventIterator = imap(lambda phrase: \
			imap(lambda e: (e.clocks, e), phrase), phraseIterable)
		for clocks, e in _merge(*decoratedEventIterator):
			yield e


class Note:

	def __init__(self,pitch=DEFAULT_PITCH,channel=DEFAULT_CHANNEL,duration=DEFAULT_DURATION,velocity=DEFAULT_VELOCITY):
		self.channel = channel
		self.pitch = Midi.bound_value(int(pitch))
		self.duration = duration
		self.velocity = Midi.bound_value(int(velocity))

	def __str__(self):
		return "Note(channel=%d pitch=%d velocity=%d duration=%d)" % (
			self.channel,self.pitch,self.velocity,self.duration)


class Midi:

	oneThread = None
	debug = False
	device_index = 0
	clocks_per_second = 192.0   # 96/quarter, 120 bpm

	@staticmethod
	def next_device_index():
		Midi.device_index += 1
		return Midi.device_index

	@staticmethod
	def startup():
		# Perhaps we shouldn't really throw this exception,
		# but it's probably good that people have only one place
		# where it's started
		if Midi.oneThread != None:
			raise Exception,"Midi has already been started"
		Midi.oneThread = MidiThread()
		Midi.oneThread.start()

	@staticmethod
	def shutdown():
		if Midi.oneThread:
			Midi.oneThread.keepgoing = False

	@staticmethod
	def schedule(output,msg,time=None):
		if not Midi.oneThread:
			raise Exception,"Midi hasn't been started"
		if time == None:
			time = Midi.time_now()
		Midi.oneThread.schedule(output,msg,time)

	@staticmethod
	def num_scheduled():
		if not Midi.oneThread:
			raise Exception,"Midi hasn't been started"
		return Midi.oneThread.num_scheduled()

	@staticmethod
	def time_now():
		return time.time()  # time in seconds

	@staticmethod
	def callback(f,data):
		if not Midi.oneThread:
			raise Exception,"Midi hasn't been started"
		return Midi.oneThread.callback(f,data)

	@staticmethod
	def bound_value(v):
		if v < 0:
			return 0
		if v > 127:
			return 127
		return v

class MidiThread(Thread):

	def __init__(self):
		Thread.__init__(self)
		self.setDaemon(True)
		self.too_old_clockssecs = 1000 * 30   # 30 seconds

		self.midiinout_lock = thread.allocate_lock()
		self.scheduled_lock = thread.allocate_lock()
		self.midiin = {}
		self.midiin_add = None
		self.midiin_del = None
		self.midiout_add = None
		self.midiout_del = None
		self.thread_midiout = {}

		self.firstevent = 0
		self.nextevent = 0
		self.keepgoing = True
		# self.clocks_per_second = 192.0   # 96/quarter, 120 bpm
		self.timenow = Midi.time_now()
		self.scheduled = []
		self.next_scheduled = None
		self.callback_func = None
		self.callback_data = None
		self.outputcallback_func = None
		self.outputcallback_data = None
		
		self._timer_calls = []
		self._next_timer = None

	def num_scheduled(self):
		self.scheduled_lock.acquire()
		n = len(self.scheduled)
		self.scheduled_lock.release()
		return n

	def callback(self,f,data):
		self.callback_func = f
		self.callback_data = data

	def outputcallback(self,f,data):
		self.outputcallback_func = f
		self.outputcallback_data = data

	def run(self):
		try:
			while self.keepgoing:
				self.timenow = Midi.time_now()
				# print "LOOP self.timenow updated to %f" % self.timenow

				if self.next_scheduled <= self.timenow:
					self._send_scheduled(self.timenow)

				# make sure midiin doesn't change during loop
				self.midiinout_lock.acquire()
				if self.midiin_add:
					for i in self.midiin_add:
						self.midiin[i] = self.midiin_add[i]
					self.midiin_add = None

				if self.midiin_del:
					for i in self.midiin_del:
						del self.midiin[i]
					self.midiin_del = None

				if self.midiout_add:
					for i in self.midiout_add:
						self.thread_midiout[i] = self.midiout_add[i]
					self.midiout_add = None

				if self.midiout_del:
					for i in self.midiout_del:
						del self.thread_midiout[i]
					self.midiout_del = None
				self.midiinout_lock.release()

				if self._next_timer <= self.timenow:
					self._invoke_timer_callbacks(self.timenow)

				for k in self.midiin:
					v = self.midiin[k]
					if not v:
						continue
					if not v.is_open():
						continue
					if v.poll():
						try:
							d = v.read(1)
						except:
							print "EXCEPTION while reading MIDI Input = %s" % format_exc()
						if d != None:
							bytes = d[0][0]
							tm = d[0][1]
							self._gotmidi(v,bytes,tm)
				sleep(0.001)

			if Midi.debug:
				print "Closing MIDI inputs..."
			for k in self.midiin:
				v = self.midiin[k]
				if v:
					try:
						v.close()
					except:
						print "Exception in v.close_input: %s"  % format_exc()
						pass
					self.midiin[k] = None
			if Midi.debug:
				print "Closing MIDI outputs..."
			for k in self.thread_midiout:
				v = self.thread_midiout[k]
				if v:
					try:
						v.close()
					except:
						print "Exception in v.close_output: %s"  % format_exc()
						pass
					self.midiin[k] = None
			return

		except:
			print "EXCEPTION in MidiThread.run()!? = %s" % format_exc()

	def _add_midiin(self,mi):
		self.midiinout_lock.acquire()
		if self.midiin_add == None:
			self.midiin_add = {}
		self.midiin_add[mi] = mi  # NEW CODE
		self.midiinout_lock.release()

	def _remove_midiin(self,m):
		self.midiinout_lock.acquire()
		if self.midiin_del == None:
			self.midiin_del = {}
		self.midiin_del[m] = m  # NEW CODE
		self.midiinout_lock.release()

	def _add_midiout(self,mi):
		self.midiinout_lock.acquire()
		if self.midiout_add == None:
			self.midiout_add = {}
		self.midiout_add[mi] = mi  # NEW CODE
		self.midiinout_lock.release()

	def _remove_midiout(self,m):
		self.midiinout_lock.acquire()
		if self.midiout_del == None:
			self.midiout_del = {}
		self.midiout_del[m] = m   # NEW CODE
		self.midiinout_lock.release()

	def _send_scheduled(self,now):

		while True:

			self.scheduled_lock.acquire()
			if self.next_scheduled == None or self.next_scheduled > now:
				self.scheduled_lock.release()
				break
			s = self.scheduled.pop(0)
			self.scheduled_lock.release()

			if not s.output.is_open():
				print "Scheduled output device isn't open?"
			else:
				try:
					if Midi.debug:
						dbg = "Writing: %s to %s now=%f s.time=%f  self.timenow=%f" % (s.msg,s.output.name,now,s.time,self.timenow)
						print dbg
					# If output callback is set, use it,
					# otherwise write to portmidi output.
					if self.outputcallback_func:
						try:
							self.outputcallback_func(s,self.outputcallback_data)
						except:
							print "Exception in midi output callback: "+format_exc()
					else:
						if hasattr(s.output,"write_msg"):
							s.output.write_msg(s.msg)
						else:
							s.msg.write(s.output)
				except:
					# print "out=",s.msg
					print "Error writing MIDI output: %s" % sys.exc_info()[1]
			self.scheduled_lock.acquire()
			if len(self.scheduled) == 0:
				self.next_scheduled = None
				self.scheduled_lock.release()
				return
			self.next_scheduled = self.scheduled[0].time
			self.scheduled_lock.release()

	def _insert_in_schedule(self,msg):
		# Insert into scheduled list, can be optimized
		inserted = False
		ix = 0
		self.scheduled_lock.acquire()
		for i in self.scheduled:
			if msg.time < i.time:
				inserted = True
				self.scheduled.insert(ix,msg)
				break
			ix = ix + 1
		if not inserted:
			self.scheduled.append(msg)

		self.next_scheduled = self.scheduled[0].time
		self.scheduled_lock.release()

	def printschedule(self,label=""):
		self.scheduled_lock.acquire()
		if len(self.scheduled) == 0:
			print "=== Schedule list === EMPTY!!!!"
			self.scheduled_lock.release()
			return
		print "=== Schedule list %s ===" % label
		for i in self.scheduled:
			print "i=",i
		print "next_scheduled=",self.next_scheduled
		print "========="
		self.scheduled_lock.release()

	def clocks2secs(self,clocks):
		print "clocks2secs!  clocks_per_second=%f" % Midi.clocks_per_second
		return clocks / Midi.clocks_per_second

	def schedule(self,output,msg,time=None):
		if not output.is_open():
			raise Exception, "schedule(): output device isn't open?"
		if time == None:
			tm0 = self.timenow
		else:
			tm0 = time

		# Non-sequenced MidiMsg have no clocks value
		if isinstance(msg,MidiMsg):
			# print "Scheduling non-seq tm0=",tm0
			m = ScheduledMidiMsg(tm0,msg,output=output)
			self._insert_in_schedule(m)
			return

		if not isinstance(msg,SequencedEvent):
			raise Exception,"schedule needs a SequencedMidiMsg or MidiMsg"

		tm1 = tm0 + self.clocks2secs(msg.clocks)

		dbg = "SCHEDULE time=%f tm0=%f tm1=%f timenow=%f" % (time,tm0,tm1,self.timenow)
		print dbg

		if isinstance(msg,SequencedMidiMsg):
			n1 = ScheduledMidiMsg(tm1,msg.msg,output=output)
			# print "Scheduling seq tm1=",tm1
			self._insert_in_schedule(n1)

		elif isinstance(msg,SequencedNote):
			tm2 = tm0 + self.clocks2secs(msg.clocks+msg.duration)
			n1 = ScheduledMidiMsg(tm1,
				NoteOn(
					pitch=msg.pitch,
					channel=msg.channel,
					velocity=msg.velocity,
					),
				output = output
				)
			n2 = ScheduledMidiMsg(tm2,
				NoteOff(
					pitch=msg.pitch,
					channel=msg.channel,
					velocity=msg.releasevelocity,
					),
				output = output
				)
			self._insert_in_schedule(n1)
			self._insert_in_schedule(n2)
		else:
			raise Exception,"schedule isn't prepared to handle msg=",msg

	def _gotmidi(self,device,bytes,tm):
		b0 = bytes[0]
		b1 = bytes[1]
		b2 = bytes[2]
		b3 = bytes[3]
		secs = tm / 1000.0

		if Midi.debug:
			print "b0123=",b0, b1, b2, b3, " tm=",tm," time=",time.time()

		# Realtime messages can occur anytime
		if (b0 & 0xf8) == 0xf8:
			# could be 0xf8, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe or 0xff
			m = RealTime(b0)
			m.device = device
			self._push_input_msg(m,secs)
			return

		finished = True

		# If we're in the process of receiving a sysex...
		m = device.sysex
		if m != None:
			# ... and we receive a non-realtime status byte
			# or an EOX, we finish the sysex off
			if (b0 & 0x80) == 0x80 and b0 != EOX:
				# non-realtime status bytes always terminate it
				if Midi.debug:
					print "PUSHING SYSEX ended by status "
				self._push_input_msg(m,secs)
				device.sysex = None
				# Set m to None in order to make sure that
				# we continue on and process the status msg
				m = None
			else:
				# Otherwise append bytes to the sysex until EOX
				m.append(b0)
				if b0 != EOX:
				  m.append(b1)
				  if b1 != EOX:
				    m.append(b2)
				    if b2 != EOX:
				      m.append(b3)
				      if b3 != EOX:
					# If no EOX, then we don't want to
					# finish off the sysex
				        finished = False
				if finished:
					device.sysex = None

			if m != None:
				# Keep the time when the sysex was created
				if finished:
					self._push_input_msg(m,secs)
				return

		if (b0 & 0xf0) == 0x80:
			m = NoteOff(channel=(b0&0x0f)+1,pitch=b1,velocity=b2)
		elif (b0 & 0xf0) == 0x90:
			if b2 == 0:
				m = NoteOff(channel=(b0&0x0f)+1,pitch=b1,velocity=b2)
			else:
				m = NoteOn(channel=(b0&0x0f)+1,pitch=b1,velocity=b2)
		elif (b0 & 0xf0) == 0xa0:
			m = Pressure(channel=(b0&0x0f)+1,pitch=b1,pressure=b2)
		elif (b0 & 0xf0) == 0xb0:
			m = Controller(channel=(b0&0x0f)+1,controller=b1,value=b2)
		elif (b0 & 0xf0) == 0xc0:
			m = Program(channel=(b0&0x0f)+1,program=b1)
		elif (b0 & 0xf0) == 0xd0:
			m = ChannelPressure(channel=(b0&0x0f)+1,pressure=b1)
		elif (b0 & 0xf0) == 0xe0:
			m = PitchBend(channel=(b0&0x0f)+1,value=(b1&0x3f)+((b2&0x3f)<<6))
		elif (b0 & 0xf0) == 0xf0:
			m = SysEx(b0)
			device.sysex = m
			m.append(b1)
			if b1 != EOX:
			  m.append(b2)
			  if b2 != EOX:
			    m.append(b3)
			    if b3 != EOX:
			      # If no EOX, then we don't want to
			      # finish off the sysex
			      finished = False
			if finished:
				device.sysex = None

		if m == None:
			print "Unexpected, m==None?  b0=",b0
		else:
			m.device = device
			if finished:
				self._push_input_msg(m,secs)

	def _push_input_msg(self,midimsg,tm):

		e = MidiEvent(midimsg,tm)
		if self.callback_func:
			try:
				self.callback_func(e,self.callback_data)
			except:
				print "Exception in midi callback: "+format_exc()

	def _insert_timer(self, timerEvent):
		# Insert into scheduled list, can be optimized
		inserted = False
		ix = 0
		for i in self._timer_calls:
			if timerEvent.time < i.time:
				inserted = True
				self._timer_calls.insert(ix, timerEvent)
				break
			ix = ix + 1
		if not inserted:
			self._timer_calls.append(timerEvent)

		self._next_timer = self._timer_calls[0].time

	def _invoke_timer_callbacks(self, now):
		if self._next_timer is None:
			return
		to_reschedule = []
		while self._next_timer <= now:
			s = self._timer_calls.pop(0)
			try:
				nextTime = s.invoke(now)
				if nextTime:
					to_reschedule.append(s)
			except:
				print "Error invoking scheduled callback: %s" % format_exc()
			# break out of the loop if no more calls for now (or at all)
			if not self._timer_calls:
				break
			else:
				self._next_timer = self._timer_calls[0].time
		[self._insert_timer(s) for s in to_reschedule]
		self._next_timer = self._timer_calls and self._timer_calls[0].time or None
			
	def schedule_callback(self, func, time=None, *args, **kwargs):
		"""
		Schedule a function to be invoked during the main MIDI processing
		loop.
		
		@param func: the function to invoke. The function must
			take the current Midi thread time as an
			argument, followed by the time for which the callback was
			requested. The function can also take additional positional
			and keyword arguments, to preserve state between invocations.
			
			The function must either return a single value (if the function
			only requires the two time arguments) or a tuple (if it requires
			additional arguments). The single value or the first value of the
			tuple specifies the time at which to next invoke the callback.
			To permanently remove the callback from the schedule, the function
			should return None. To specify values of positional arguments for
			the next invocation, the function should return them as a sequence,
			in the second element of the tuple. Values for keyword arguments
			should be returned as a dictionary in the third element.
		@param time: the time to invoke the function, or None to schedule
			the function for immediate invocation
		@param *args: additional arguments to pass to the function
		@param **kwargs: additional keyword arguments to pass to the function
		"""
		if not time:
			time = Midi.time_now()
		timerEvent = TimerEvent(time, func, *args, **kwargs)
		self._insert_timer(timerEvent)

class MidiBaseHardwareInput:

	def __init__(self,input_name):
		raise Exception,"MidiBaseHardwareInput, no input matches: %s" % input_name

	def device_index(self):
		return 1;

	def open(self):
		raise Exception, "MidiBaseHardwareInput, unable to open "+self.name

	def close(self):
		pass

	def __str__(self):
		return 'MidiInput(name="%s")' % (self.name)

	def to_xml(self):
		return '<midi_input name="%s"/>' % (self.name)


class MidiBaseHardwareOutput:

	def __init__(self,output_name):
		raise Exception,"MidiBaseHardwareOutput, no output matches: %s" % input_name

	def device_index(self):
		return 2;

	def open(self):
		if not Midi.oneThread:
			raise Exception,"Midi hasn't been started"
		raise Exception,"MidiBaseHardwareOutput, no output matches: %s" % input_name

	def close(self):
		pass

	def schedule(self,msg,time=None):
		Midi.schedule(self,msg,time)


	def __str__(self):
		return 'MidiOutput(name="%s")' % (self.name)

	def to_xml(self):
		return '<midi_output name="%s"/>' % (self.name)


"""
This is executed when module is loaded
"""
