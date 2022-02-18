"""
This module provides an interface to MIDI things for debugging.
"""

import sys
import time
import traceback
import thread
import threading
import copy
import string

from threading import Thread,Lock
from math import sqrt
from ctypes import *
from time import sleep
from traceback import format_exc
from array import array

from nosuch.midiutil import *

class MidiDebugHardware(MidiBaseHardware):

	def __init__(self):
		pass

	def input_devices(self):
		return ['debug']

	def output_devices(self):
		return ['debug']

	def get_input(self,input_name):
		return MidiDebugHardwareInput(input_name)

	def get_output(self,output_name):
		return MidiDebugHardwareOutput(output_name)

class MidiDebugHardwareInput(MidiBaseHardwareInput):

	def __init__(self,input_name):
		pass

	def open(self):
		if Midi.oneThread:
			Midi.oneThread._add_midiin(self)

	def is_open(self):
		return True

	def poll(self):
		return False  # fake input, never has anything

	def close(self):
		if Midi.oneThread:
			Midi.oneThread._remove_midiin(self)

	def __str__(self):
		return 'MidiInput(name="debug")'

	def to_xml(self):
		return '<midi_input name="debug"/>'


class MidiDebugHardwareOutput(MidiBaseHardwareOutput):

	def __init__(self,output_name):
		pass

	def is_open(self):
		return True

	def open(self):
		pass

	def close(self):
		pass

	def write_short(self,b1,b2,b3):
		print "MidiDebugHardwareOutput bytes = %02x %02x %02x" % (b1,b2,b3)

	def write_sysex(self,bytes):
		print "MidiDebugHardwareOutput sysex bytes = %02x %02x %02x ..." % (bytes[0],bytes[1],bytes[2])

	def schedule(self,msg,time=None):
		Midi.schedule(self,msg,time)

	def __str__(self):
		return 'MidiOutput(name="debug")'

	def to_xml(self):
		return '<midi_output name="debug"/>'


"""
This is executed when module is loaded
"""
