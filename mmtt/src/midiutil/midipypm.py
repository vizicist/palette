"""
This module provides an interface to MIDI things based on pyportmidi (pypm).
"""

import sys
import time
import traceback
import thread
import threading
import copy
import pygame.pypm
import string
import nosuch.midifile

from threading import Thread,Lock
from math import sqrt
from ctypes import *
from time import sleep
from traceback import format_exc
from array import array

from pygame.pypm import CountDevices, GetDeviceInfo, GetDefaultInputDeviceID, GetDefaultOutputDeviceID

from nosuch.midifile import *
from nosuch.midiutil import *

class MidiPypmHardware(MidiBaseHardware):

	def __init__(self):
		# self.midiin = {}
		# self.midiin_add = None
		# self.midiin_del = None
		# self.midiout_del = None
		# self.hard_midiout = {}
		pass

	def time_now(self):
		# we want it in seconds, not milliseconds
		return pygame.pypm.Time() / 1000.0

	def input_devices(self):
		ndevices = CountDevices()
		arr = []
		for n in range(ndevices):
			v = GetDeviceInfo(n)
			hasinput = v[2]
			name = v[1]
			if hasinput:
				arr.append(name)
		return arr

	def output_devices(self):
		ndevices = CountDevices()
		arr = []
		for n in range(ndevices):
			v = GetDeviceInfo(n)
			hasoutput = v[3]
			name = v[1]
			if hasoutput:
				arr.append(name)
		return arr

	def get_input(self,input_name):
		return MidiPypmHardwareInput(input_name)

	def get_output(self,output_name):
		return MidiPypmHardwareOutput(output_name)

class MidiPypmHardwareInput(MidiBaseHardwareInput):

	def __init__(self,input_name):
		if input_name == None:
			n = GetDefaultInputDeviceID()
			found = GetDeviceInfo(n)
		else:
			ndevices = CountDevices()
			found = None
			for n in range(ndevices):
				v = GetDeviceInfo(n)
				name = v[1]
				hasinput = v[2]
				inuse = v[4]
				if hasinput and name == input_name:
					found = v
					break
		if not found:
			raise Exception,"No input matches: %s" % input_name

		self.index = n
		self.name = found[1]
		self.hasinput = found[2]
		self.hasoutput = found[3]
		self.inuse = found[4]
		self.sysex = None
		self.input = None

	def open(self):
		# Get fresh value, inuse might have changed?
		v = GetDeviceInfo(self.index)
		if self.inuse:
			raise Exception, "Device "+self.name+" is already open by something else"
		try:
			self.input = pygame.pypm.Input(self.index)
		except: 
			raise Exception, "Unable to open "+self.name+" (is it open by another process?)"
		if not Midi.oneThread:
			raise Exception,"Midi hasn't been started"
		self.sysex = None
		if Midi.oneThread:
			Midi.oneThread._add_midiin(self)

	def is_open(self):
		return(self.input != None)

	def poll(self):
		return(self.input.Poll())

	def read(self,n):
		return(self.input.Read(n))

	def close(self):
		del self.input
		self.input = None
		if Midi.oneThread:
			Midi.oneThread._remove_midiin(self)

	def __str__(self):
		return 'MidiInput(name="%s" index="%d")' % (self.name,self.index)

	def to_xml(self):
		return '<midi_input name="%s" index="%d"/>' % (self.name,self.index)


class MidiPypmHardwareOutput(MidiBaseHardwareOutput):

	def __init__(self,output_name):
		if output_name == None:
			n = GetDefaultOutputDeviceID()
			found = GetDeviceInfo(n)
		else:
			ndevices = CountDevices()
			found = None
			for n in range(ndevices):
				v = GetDeviceInfo(n)
				name = v[1]
				hasoutput = v[3]
				inuse = v[4]
				if hasoutput and name == output_name:
					found = v
					break
		if not found:
			raise Exception,"No output matches: %s" % output_name

		self.index = n
		self.name = found[1]
		self.hasinput = found[2]
		self.hasoutput = found[3]
		self.inuse = found[4]
		self.pm_output = None

	def open(self):
		v = GetDeviceInfo(self.index)
		if self.inuse:
			raise Exception, "Device "+self.name+" is already open by something else"
		if not Midi.oneThread:
			raise Exception,"Midi hasn't been started"
		try:
			# If the output is already open, this crashes python,
			# need to figure out why.
			self.pm_output = pygame.pypm.Output(self.index,0)
		except: 
			raise Exception, "Unable to open "+self.name+" : "+format_exc()
		if Midi.oneThread:
			Midi.oneThread._add_midiout(self)

	def is_open(self):
		return (self.pm_output != None)

	def write_short(self,*bytes):
		self.pm_output.WriteShort(*bytes)

	def write_sysex(self,bytes):
		self.pm_output.WriteSysEx(0,bytes)

	def close(self):
		del self.pm_output
		self.pm_output = None
		if Midi.oneThread:
			Midi.oneThread._remove_midiout(self)

	def schedule(self,msg,time=None):
		# if not Midi.oneThread:
		# 	raise Exception,"Midi hasn't been started"
		# if time == None:
		# 	time = Midi.time_now()
		Midi.schedule(self,msg,time)


	def __str__(self):
		return 'MidiOutput(name="%s" index="%d")' % (self.name,self.index)

	def to_xml(self):
		return '<midi_output name="%s" index="%d"/>' % (self.name,self.index)


"""
This is executed when module is loaded
"""
