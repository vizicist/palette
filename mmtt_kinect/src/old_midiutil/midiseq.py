#######################################################
#
# ***** NOTE *****
#
# This file hasn't been updated since some changes were made in the way
# Midi() works.    Basically, rather than using MidiInput.devices(),
# you need to explicitly pull in the MIDI hardware you're using, e.g.
#
#     m = MidiPypmHardware()
#
# and then use methods on that:
#
#     m.input_devices()   # returns an array of names
#     i = m.get_input(name)
# 
# The code in this file needs to be updated to reflect that change.
#
#######################################################



from nosuch.midiutil import *
from Queue import Queue
from threading import Thread

CLOCKS_PER_BEAT = 96.0

 
class MidiDispatcher(Thread):
    """
    Route incoming Midi events to one or more processors.
    """
    def __init__(self, inputCallbacks, inputProcessedCallback=None):
        Thread.__init__(self)
        self.setDaemon(True)
        self._inputQueue = Queue()
        self._callbacks = inputCallbacks
        self._inputProcessedCallback = inputProcessedCallback
    
    def execute(self):
        """
        Begin processing events from queue on a background thread.
        """
        Midi.callback(self.onMidiInput, "dummy")
        self.start()
    
    def onMidiInput(self, event, data):
        """
        Receive input from Midi devices (or simulated inpute from the
        application)
        """
        self._inputQueue.put((event, data))
        
    def run(self):
        """
        Process events enqueued by the onMidiInput method
        """
        get = self._inputQueue.get
        while True:
            # pass events to all callbacks
            midiInput = get()
            for c in self._callbacks:
                c(*midiInput)
            if self._inputProcessedCallback:
                self._inputProcessedCallback()

class MergedIO(object):
    """
    Automatically route incoming MIDI events to output devices.
    This enables, for example, playing a virtual instrument via
    a keyboard controller connected to a MIDI input device.
    """
    def __init__(self, deviceMap):
        """
        Create a MergedIO instance.
        
        @param deviceMap: a L{MidiDeviceMap} object that maps
            input devices to output devices
        """
        self._deviceMap = deviceMap
    
    def onMidiInput(self, midiEvent, data):
        """
        Route events received from Midi input devices to output
        devices.
        """
        # A device may be mapped to the input but not for automatic
        # merging; just send the events to the devices mapped for merging.
        if hasattr(midiEvent.midimsg, "device"):
            mergedOutputDevices = [device for device, merge in \
                self._deviceMap.getOutputDevices(midiEvent.midimsg.device) if merge]
            for d in mergedOutputDevices:
                Midi.oneThread.schedule(d, midiEvent.midimsg, midiEvent.time)

class MidiListener(object):
    """
    Base class for managing Midi event processing, typically throughout
    an entire application session.
    """
    def __init__(self):
        """
        Create a MidiListener instance.
        """
        self._openedInputDevices = {}
        self._openedOutputDevices = {}
        self._dispatcher = None
    
    def _createClients(self):
        # Subclasses override by returning a list of functions that 
        # will handle Midi input events. The function signatures must
        # match those expected by the Midi.callback method.
        return []
    
    def _startClients(self):
        # Subclasses that perform throughout the application session
        # can do startup work here.
        pass

    def _stopClients(self):
        # Subclasses that perform throughout the application session
        # can do cleanup work here.
        pass
    
    def _getOpenedDevice(self, deviceDict, deviceFactory, deviceName):
        # Internal helper function that opens a device exactly once
        if deviceName not in deviceDict:
            device = deviceFactory(deviceName)
            device.open()
            deviceDict[deviceName] = device
        return deviceDict[deviceName]
    
    def _onInputProcessed(self):
        # subclasses override
        pass
    
    def openInputDevice(self, name):
        """
        Open a Midi input device. This method can be called multiple times
        without causing a device conflict.
        
        @param name: the name of the device to open
        """
        return self._getOpenedDevice(self._openedInputDevices, MidiInput, name)
        
    def openOutputDevice(self, name):
        """
        Open a Midi output device. This method can be called multiple times
        without causing a device conflict.
        
        @param name: the name of the device to open
        """
        return self._getOpenedDevice(self._openedOutputDevices, MidiOutput, name)

    def start(self):
        """
        Start Midi processing.
        """
        Midi.startup()
        self._dispatcher = MidiDispatcher(self._createClients(),
            self._onInputProcessed)
        self._dispatcher.execute()
        self._startClients()
    
    def stop(self):
        """
        End Midi processing.
        """
        self._stopClients()
        Midi.shutdown()

class MidiSequencer(MidiListener):
    """
    Provide services related to recording and playing Midi sequences.
    """
    def __init__(self, deviceMap=None):
        """
        Create a MidiSequencer instance.        
        """
        MidiListener.__init__(self)
        self._deviceMap = deviceMap
        self._mergedIO = None
        self._metronome = None
        self._recorder = None
        self._playing = False
        self._playbackStartTime = None
        self._playbackThread = None
        self._beatsPerBar = 4
        self._feedbackQueue = Queue()
        self._feedbackHandler = None
    
    def _createClients(self):
        # Open input and output devices, and prepare to route events
        # from inputs to outputs
        self._mergedIO = MergedIO(self.deviceMap)
        # Create a metronome
        self._metronome = Metronome(self._getMetronomeDevice(), self.pushEvent)
        # Create a recorder (sequencer)
        self._recorder = MidiRecorder(self._onTrackInput)
        # return the functions that will handle events from Midi input
        return [self._onMidiInput, self._mergedIO.onMidiInput, 
            self._recorder.onMidiInput]
    
    def _emptyFeedbackQueue(self):
        while self._feedbackQueue.qsize():
            self._feedbackQueue.get_nowait()
    
    def _getDeviceMap(self):
        """
        Return the L{MidiDeviceMap} that defines routings between
        input and output devices.
        """
        if self._deviceMap is None:
            self._deviceMap = MidiDeviceMap(self.openInputDevice, 
                self.openOutputDevice)
            self._deviceMap.addDefaultDeviceMapping(merge=True)
        return self._deviceMap
    
    def _getDefaultOutputDeviceName(self):
        return MidiOutput.devices()[pypm.GetDefaultOutputDeviceID()]
    
    def _getFeedbackQueue(self):
        """
        Return a L{Queue} that supplies feedback events to the application.        
        """
        return self._feedbackQueue
    
    def _getMetronomeDevice(self):
        return self.openOutputDevice(self._getDefaultOutputDeviceName())
    
    def _getPlaying(self):
        """
        Return whether the sequencer is playing.
        """
        return self._playing
    
    def _getPlaybackPhrases(self, includeMutedTracks = False):
        # Yield copies, rather than originals, allowing simultaneous
        # playback and recording of a track (overdubbing).
        return (track.phrase[:] for track in self.recorder.sequence if \
            (includeMutedTracks or (not track.mute)))
    
    def _getRecorder(self):
        """
        Return the MidiRecorder object that the sequencer uses for recording
        sequences.
        """
        return self._recorder
    
    def _getSequence(self):
        return self._recorder.sequence
    def _setSequence(self, sequence):
        self._recorder.sequence = sequence

    def _getTempo(self):
        return int((Midi.oneThread.clocks_per_second / CLOCKS_PER_BEAT) * 60.0)
    def _setTempo(self, bpm):
        Midi.oneThread.clocks_per_second = (bpm / 60.0) * CLOCKS_PER_BEAT
    
    def _onMidiInput(self, event, data):
        # This is the one receiver of the TickMsg (sent by the Metronome) that
        # sends it back through the feedback queue.
        if self._feedbackHandler:
            msg = event.midimsg
            if isinstance(msg, TickMsg):
                if msg.clocks is not None:
                    self._feedbackQueue.put(msg)

    def _onInputProcessed(self):
        # subclasses override
        if self._feedbackHandler:
            feedbackMessages = []
            messageCount = self._feedbackQueue.qsize()
            if messageCount:
                feedbackMessages = [self._feedbackQueue.get_nowait() for \
                    i in range(messageCount)]
            keepCalling = self._feedbackHandler(feedbackMessages)
            if not keepCalling:
                self._feedbackHandler = None
                self._emptyFeedbackQueue()

    def _onTrackInput(self, msg):
        if self._feedbackHandler:
            self._feedbackQueue.put(msg)

    def _playMergedEvents(self, startTime):
        def _getNextMergedNote(mergedMessages):
            try:
                note = mergedMessages.next()
            except:
                note = None
            return note
            
        def _onClock(now, tickTime, mergedMessages, lastClocks, 
            nextNote):
            # this is invoked from a scheduled Midi callback
            nextNoteTime = None
            if self._playing:
                while nextNote and nextNote.clocks == lastClocks:
                    # play it
                    [outputDevice.schedule(nextNote.msg) for outputDevice, _ in \
                        self.deviceMap.getOutputDevices(nextNote.msg.device)]
                    # get the next note
                    nextNote = _getNextMergedNote(mergedMessages)
                if nextNote:
                    nextNoteTime = tickTime + Midi.oneThread.clocks2secs(\
                        nextNote.clocks - lastClocks)
                    lastClocks = nextNote.clocks
                else:
                    self._playing = False
            # return the next time for the callback to invoke this method,
            # along with the other required arguments, or None to stop
            return nextNoteTime and (nextNoteTime, [mergedMessages, 
                lastClocks, nextNote]) or None
            
        nextNoteTime = startTime and startTime or self._metronome.nextTickTime
        lastClocks = 0.0
        mergedMessages = (n for n in Phrase.merged(self._getPlaybackPhrases()))
        nextNote = _getNextMergedNote(mergedMessages)
        if nextNote:
            Midi.oneThread.schedule_callback(_onClock, nextNoteTime,
                mergedMessages, lastClocks, nextNote)
        else:
            self._playing = False            

    def _startClients(self):
        self._metronome.active = True

    def _stopClients(self):
        self._metronome.active = False

    deviceMap = property(fget=_getDeviceMap, doc=_getDeviceMap.__doc__)
    
    feedbackQueue = property(fget=_getFeedbackQueue, 
        doc=_getFeedbackQueue.__doc__)
    
    def getTotalClocks(self):
        """
        Return the length of the sequence, in Midi clocks.
        """
        return self._recorder.sequence.getTotalClocks()

    playing = property(fget=_getPlaying, doc=_getPlaying.__doc__)
    
    def pushEvent(self, event, data):
        """
        Simulate Midi input.
        """
        self._dispatcher.onMidiInput(event, data)
    
    recorder = property(fget=_getRecorder, doc=_getRecorder.__doc__)
    
    sequence = property(fget=_getSequence, fset=_setSequence, doc = \
        "Return or set the L{MultitrackSequence} for recording Midi input.")

    def startMetronome(self):
        """
        Start playing the metronome.
        """
        self._metronome.audible = True
        
    def stopMetronome(self):
        """
        Stop playing the metronome.
        """
        self._metronome.audible = False
        self._metronome.stopOutputTimer()
    
    def startPlayback(self, startTime=None, feedbackHandler=None):
        """
        Play the recorded sequence. Determine the output device(s) for each 
        recorded from the L{DeviceMap}.
        
        @param startTime: if specified, the time to begin playback; if omitted
            playback begins immediately
        """
        self._feedbackHandler = feedbackHandler
        self._playing = True
        if startTime is None:
            self._metronome.startOutputTimer()
        self._playMergedEvents(startTime)
    
    def startRecording(self, playMetronome=True, countOffBeats=8,
        endAfterBeats=None, feedbackHandler=None):
        """
        Start recording into the tracks that are armed for recording.
        
        @param playMetronome: C{True} to play the metronome during recording;
            C{False} to record without it.
        @param countOffBeats: the number of beats to count before recording
            (most useful when L{playMetronome} is C{True}
        @param endAfterBeats: if specified, the number of beats to record
            (not including the L{countOffBeats}); if omitted, record
            continuously until L{StopRecording} is invoked
        """
        self._feedbackHandler = feedbackHandler
        timenow = self._metronome.nextTickTime
        countOffClocks = countOffBeats * CLOCKS_PER_BEAT
        self._metronome.startOutputTimer(clock=-countOffClocks, 
            bar=-(countOffBeats/self._beatsPerBar))
        if playMetronome:
            self.startMetronome()
        startTime = timenow + \
            Midi.oneThread.clocks2secs(countOffClocks)
        stopTime = endAfterBeats and startTime + \
            Midi.oneThread.clocks2secs(endAfterBeats * CLOCKS_PER_BEAT) or None
        self._recorder.start(startTime, stopTime)
        # play what's been recorded (and not muted)
        self.startPlayback(startTime=startTime, feedbackHandler=feedbackHandler)
    
    def stopPlayback(self):
        """
        Stop playing the recorded sequence.
        """
        self._playing = False
        if self._playbackThread:
            self._playbackThread.join()
        self._feedbackHandler = None
        self._playbackThread = None
        self.stopMetronome()
    
    def stopRecording(self):
        """
        Stop recording.
        """
        self._recorder.stop()
        self.stopPlayback()
    
    tempo = property(fget=_getTempo, fset=_setTempo, 
        doc="Return or set the recording and playback tempo, in beats per minute")
    
class MidiDeviceMap(object):
    """
    Map input devices to output devices.
    """
    def __init__(self, inputDeviceFactory, outputDeviceFactory):
        """
        Create a MidiDeviceMap instance.
        
        @param inputDeviceFactory: a factory function for creating
            a L{MidiInputDevice} object, given a device name
        @param outputDeviceFactory: a factory function for creating
            a L{MidiOutputDevice} object, given a device name        
        """
        self._deviceNameMap = {}
        self._deviceMap = {}
        self._inputDeviceFactory = inputDeviceFactory
        self._outputDeviceFactory = outputDeviceFactory
    
    def addDefaultDeviceMapping(self, merge=True):
        """
        Map the default input device to the default output device.
        
        @param merge: immediately route events from the input device to
            the output device
        """
        if len(MidiInput.devices()):
            self.addMapping(MidiInput.devices()[pypm.GetDefaultInputDeviceID()],
                MidiOutput.devices()[pypm.GetDefaultOutputDeviceID()], merge)
    
    def addMapping(self, inputName, outputName, merge):
        """
        Map an input device to an output device.
        
        @param inputName: the name of the input device
        @param outputName: the name of the output device
        @param merge: immediately route events from the input device to
            the output device        
        """
        if not self.mappingExists(inputName, outputName):
            if inputName not in self._deviceNameMap:
                self._deviceNameMap[inputName] = []
            mappedOutputs = self._deviceNameMap[inputName]
            mappedOutputs.append((outputName, merge))
                
            inputDevice = self._inputDeviceFactory(inputName)
            outputDevice = self._outputDeviceFactory(outputName)
            if inputDevice not in self._deviceMap:
                mappedDevices = []
                self._deviceMap[inputDevice] = mappedDevices
            else:
                mappedDevices = self._deviceMap[inputDevice]
            mappedDevices.append((outputDevice, merge))
    
    def canMap(self, inputName, outputName):
        """
        Return whether an input device can be mapped to an output device.
        
        @param inputName: the name of the input device
        @param outputName: the name of the output device
        """
        return (inputName != outputName) and \
            not self.mappingExists(inputName, outputName)
    
    def getMapping(self, inputName):
        """
        Get the mapping for an input device.
        
        @param inputName: the name of the input device
        @return: a list of (deviceName, merged) tuples for each output
            device mapped to the input device, where deviceName is the
            name of an output device, and merged is a bool that
            indicates whether to immediately route input from the
            input device to the output device
        """
        return self._deviceNameMap.get(inputName, [])
    
    def getOutputDevices(self, inputDevice):
        """
        Return output devices mapped to an input device.
        
        @param inputDevice: the L{MidiInputDevice} object that represents
            the input device
        @return: a list of (device, merged) tuples for each output device
            mapped to the input device, where device is a L{MidiOutputDevice}
            and merged is a bool that indicates whether to immediately
            route input from the input device to the output device
        """
        return self._deviceMap.get(inputDevice, [])

    def mappingExists(self, inputName, outputName):
        """
        Return whether an input device is mapped to an output device.
        
        @param inputName: the name of the input device
        @param outputName: the name of the output device
        """
        return outputName in [name for name, _ in self.getMapping(inputName)]
    
    def removeMapping(self, inputName, outputName):
        """
        Remove the mapping between an input device and an output device.

        @param inputName: the name of the input device
        @param outputName: the name of the output device
        """
        if self.mappingExists(inputName, outputName):
            mappedParameters = self._deviceNameMap[inputName]
            for i in range(len(mappedParameters)):
                if mappedParameters[i][0] == outputName:
                    del mappedParameters[i]
                # same index in the actual device map
                inputDevice = self._inputDeviceFactory(inputName)
                del self._deviceMap[inputDevice][i]
                break

class TickMsg(MidiMsg):
    """
    Message sent by the Metronome for every Midi clock event.
    If the clocks field is a number, it represents the offset
    in Midi clocks from the beginning of the recording. Events
    sent during the countoff just prior to recording have negative 
    clock values; the start of recording is clock 0.
    """
    def __init__(self, clocks):
        MidiMsg.__init__(self, "tick")
        self.clocks = clocks
    
    def __str__(self):
        return "tick %d" % self.clocks

class NewBarMsg(MidiMsg):
    """
    Message sent by the Metronome at the beginning of each bar during
    recording. If the bar field is a a number, it is represents the
    number of bars from the beginning of the recording. Events sent
    during the countoff just prior to recording have negative bar
    values; the start of recording is bar 0. The clocksPerBar field
    contains the length of the bar, in Midi clocks.
    """
    def __init__(self, bar, clocksPerBar):
        MidiMsg.__init__(self, "newbar")
        self.bar = bar
        self.clocksPerBar = clocksPerBar
    
    def __str__(self):
        return "bar %d (%d clocks)" % (self.bar, self.clocksPerBar)

class TrackMsg(MidiMsg):
    """
    Message placed into the sequencer's feedback queue, for track-
    oriented rendering in the user interface. The track field contains
    the zero-based index of the track. The msg field contains a L{MidiMsg}
    of L{SequencedEvent}.
    """
    def __init__(self, track, msg):
        MidiMsg.__init__(self, "trackmsg")
        self.track = track
        self.msg = msg

class Metronome(object):
    """
    Keep time in Midi clocks. Play "beats" repeatedly over Midi output
    (e.g., during recording).
    """
    def __init__(self, outputDevice, inputHandler):
        """
        Create a Metronome instance.
        
        @param outputDevice: the MidiOutput device to which to send
            the metronome's output
        @param inputHandler: a callback function into which to send
            the metronome's tick events
        """
        self._outputDevice = outputDevice
        self._inputHandler = inputHandler
        self._phraseClocks = 0
        self._active = False
        self._audible = False
        self._outputThread = None
        self._restartPhrase = False
        self._timeMsgClock = None
        self._currentPhraseClock = None
        self._currentNote = None
        self._noteStack = None
        self._currentBar = None
        self._outputTimer = False
        self.nextTickTime = None
        self.phrase = self._defaultPhrase()
    
    def _defaultPhrase(self):
        # four quarter notes, with accents on 1 and 3
        phrase = Phrase()
        lastClock = 0
        for pitch, velocity, channel, duration in \
            zip([75] * 4, [80, 40, 50, 40], [10] * 4, [96] * 4):
            phrase.append(SequencedNote(pitch=pitch, velocity=velocity,
                channel=channel, clocks=lastClock,
                duration=duration))
            lastClock += duration
        return phrase
    
    def _getActive(self):
        return self._active
    
    def _setActive(self, value):
        if value and not self._active:
            # register to receive timer callbacks every Midi clock
            self.nextTickTime = Midi.time_now()
            Midi.oneThread.schedule_callback(self._onMidiClock, 
                self.nextTickTime)
    
    def _getAudible(self):
        return self._audible
    def _setAudible(self, value):
        if not self._audible:
            self._restartPhrase = True
        self._audible = value
        
    def _getPhrase(self):
        return self._phrase
    
    def _setPhrase(self, phrase):
        self._phrase = phrase
        self._phraseClocks = sum([n.duration for n in self._phrase])
    
    def _onMidiClock(self, now, tickTime):
        # place a tick message into the Midi input queue
        self._inputHandler(MidiEvent(TickMsg(self._timeMsgClock), tickTime),
            None)
        if self._restartPhrase:
            # The metronome phrase is either being played for the first
            # time since the metronome became audible, or it was fully
            # played and it is now time to start it over again.
            self._currentPhraseClock = 0
            self._noteStack = list(reversed(range(len(self._phrase))))
            self._currentNote = self._phrase[self._noteStack.pop()]
            self._restartPhrase = False                
        if self._audible:
            if self._currentPhraseClock == self._currentNote.clocks:
                # it's time to play the current metronome phrase note
                if self._currentPhraseClock == 0:
                    # beginning of the phrase == new bar
                    # place a new bar message into the Midi input queue
                    self._inputHandler(MidiEvent(\
                        NewBarMsg(self._currentBar, self._phraseClocks), 
                            tickTime), None)
                nextNote = copy.copy(self._currentNote)
                nextNote.clocks = 0
                # play the current metronome phrase note
                self._outputDevice.schedule(nextNote, tickTime)
                if self._noteStack:
                    # get the next metronome phrase note
                    self._currentNote = self._phrase[self._noteStack.pop()]
            if self._currentPhraseClock < self._phraseClocks:
                self._currentPhraseClock += 1
            else:
                # reached the end of the metronome phrase; start from
                # the beginning next time this method is invoked
                self._restartPhrase = True
                if self._outputTimer:
                    self._currentBar += 1
        if self._outputTimer:
            self._timeMsgClock += 1
        self.nextTickTime += Midi.oneThread.clocks2secs(1)
        return self.nextTickTime    
    
    active = property(fget=_getActive, fset=_setActive, 
        doc="Turn the metronome on or off.")
    
    audible = property(fget=_getAudible, fset=_setAudible,
        doc="Play or stop playing the metronome.")
    
    phrase = property(fget=_getPhrase, fset=_setPhrase,
        doc="Return or set the L{Phrase} object to play when the metronome is on.")
        
    def startOutputTimer(self, clock = 0, bar = 0):
        """
        Reset the Midi clock and current bar to specified values. The
        metronome will include the reset values in the next TimeMsg and
        NewBarMsg that it places into Mid input, and will automatically
        increment the values.
        
        @param clock: the new Midi clock value; can be negative (e.g.,
            to indicate countoff clocks prior to the beginning of recording)
        @param bar: the new bar offset; can be negative (e.g., to indicate
            the bar position prior to the beginning of recording)
        """
        self._timeMsgClock = clock
        self._currentBar = bar        
        self._outputTimer = True
    
    def stopOutputTimer(self):
        """
        Stop setting explicit clock and bar values for TimeMsg and NewBarMsg.
        """
        self._timeMsgClock = None
        self._currentPhraseClock = None
        self._currentNote = None
        self._noteStack = None
        self._currentBar = None
        self._outputTimer = False

class MidiRecorder(object):
    """
    Record one or more tracks of Midi input.
    """
    def __init__(self, callback):
        """
        Create a MidiRecorder instance.
        """
        self._on = False
        self._tracks = []
        self._sequence = MultitrackSequence(callback)
        self._timeStart = None
        self._timeStop = None
        self._lastClock = None
    
    def _getOn(self):
        """
        Return whether recording is in progress.
        """
        return self._on
     
    def onMidiInput(self, event, data):
        """
        Route incoming Midi events to the appropriate tracks.
        """
        if not self._on:
            return
        elif event.time >= self._timeStart:
            if self._timeStop and event.time > self._timeStop:
                self.stop()
            else:
                if isinstance(event.midimsg, TickMsg):
                    self._lastClock = event.midimsg.clocks
                else:
                    if isinstance(self._lastClock, int) and \
                        self._lastClock >= 0:
                        eventClocks = self._lastClock
                    else:
                        eventClocks = round((event.time - self._timeStart) * \
                            Midi.oneThread.clocks_per_second)
                    eventChannel = hasattr(event.midimsg, "channel") and \
                        event.midimsg.channel or None
                    for track in self._sequence:
                        if track.recording and (eventChannel is None or \
                            track.channel == eventChannel):
                            track.onMidiInput(\
                                SequencedMidiMsg(event.midimsg, eventClocks))
    
    def _getSequence(self):
        return self._sequence
    def _setSequence(self, sequence):
        self._sequence = sequence

    on = property(fget=_getOn, doc=_getOn.__doc__)

    sequence = property(fget=_getSequence, fset=_setSequence, doc = \
        "Return or set the L{MultitrackSequence} for recording Midi input.")
    
    def start(self, timeStart, timeStop=None):
        """
        Start recording Midi input.
        
        @param timeStart: the time at which to begin recording
        @param clocksAfterStart: if specified, the time at which to stop
            recording
        """
        self._timeStart = timeStart
        self._timeStop = timeStop
        
        self._on = True
        
    def stop(self):
        """
        Stop recording Midi input.
        """
        self._on = False
        self._lastClock = None

class SequencerTrack(object):
    """
    Manage settings for one track in a sequence.
    """
    def __init__(self, channel, recording, mute, callback, phrase=None):
        self.channel = channel
        self.recording = recording
        self.mute = mute
        self.phrase = phrase and phrase or Phrase()
        self._callback = callback
        self._pendingNoteOffs = {}
        self._barCount = 0
    
    def erase(self):
        """
        Erase all of the events in the track.
        """
        del self.phrase[:]
        self._barCount = 0
    
    def getTotalClocks(self):
        """
        Return the length of the track in Midi clocks.
        """
        return len(self.phrase) and self.phrase[-1].clocks or 0
    
    def onMidiInput(self, sequencedEvent):
        """
        Handle Midi input.
        """
        callbackMsg = None
        msg = sequencedEvent.msg
        if not isinstance(msg, (TickMsg, NewBarMsg)):
            # add normal Midi messages to the phrase
            self.phrase.append(sequencedEvent)
            # ensure that the phrase remains sorted by time
            if len(self.phrase) > 1 and sequencedEvent.clocks < \
                self.phrase[-2].clocks:
                self.phrase.sort(key=lambda e:e.clocks)
        # pair NoteOn messages to NoteOffs
        if isinstance(msg, NoteOn):
            self._pendingNoteOffs[msg.pitch] = sequencedEvent
        elif isinstance(msg, NoteOff):
            # Create a SequencedNote event from a paired NoteOn+NoteOff
            # and pass it to the callback function.
            # This mechanism can be used to simplify GUI rendering of
            # notes as they are recorded.
            noteOnEvent = self._pendingNoteOffs.get(msg.pitch, None)
            if noteOnEvent:
                del self._pendingNoteOffs[msg.pitch]
                callbackMsg = SequencedNote(msg.pitch, 
                    velocity=noteOnEvent.msg.velocity,
                    channel=msg.channel, clocks=noteOnEvent.clocks,
                    duration=sequencedEvent.clocks - noteOnEvent.clocks,
                    releasevelocity=msg.velocity)
        elif isinstance(msg, NewBarMsg) and msg.bar >= self._barCount:
            # When a new bar is recorded, notify the callback.
            self._barCount += 1
            callbackMsg = msg
        if self._callback and callbackMsg:
            self._callback(self, callbackMsg)
            
class MultitrackSequence(list):
    """
    A list of one or more L{SequencerTrack} objects.
    """
    def __init__(self, callback, tracksOrPhrases=None):
        list.__init__(self)
        if tracksOrPhrases is not None and isinstance(tracksOrPhrases,
            (MultitrackSequence, list)):
            [self.append(track) for track in tracksOrPhrases]
        self._callback = callback
    
    def _makeTrack(self, trackOrPhrase):
        if isinstance(trackOrPhrase, SequencerTrack):
            return trackOrPhrase
        else:
            # a phrase
            track = SequencerTrack(1, False, False, self._onTrackInput,
                trackOrPhrase)
    
    def _onTrackInput(self, track, msg):
        if self._callback:
            self._callback(TrackMsg(self.index(track), msg))
    
    def _validateItem(self, track):
		if not isinstance(track, (SequencerTrack, Phrase)):
			raise Exception, \
                "MultitrackSequences can only take SequencerTrack or Phrase objects!"
    
    def append(self, trackOrPhrase):
        """
        Append a track to the end of the list.
        
        @param trackOrPhrase: a L{SequencerTrack} or L{Phrase} object
        """
        self._validateItem(trackOrPhrase)
        list.append(self, self._makeTrack(trackOrPhrase))
            
    def appendTrack(self, phrase=None, channel=1, record=False, mute=False):
        """
        Add a new track to the end of the list.
        
        @param phrase: if specified, the L{Phrase} that stores the Midi 
            events for the track
        @param channel: the Midi channel for recording and playback of
            the track
        @param record: True if the track is armed for recording; False
            otherwise
        @param mute: True if the track is muted for playback; False
            otherwise
        """
        track = SequencerTrack(channel, record, mute, self._onTrackInput)
        self.append(track)
        return track
    
    def getTotalClocks(self):
        """
        Return the length of the sequence, in Midi clocks.
        """
        trackClocks = [track.getTotalClocks() for track in self]
        return trackClocks and max(trackClocks) or 0
    
    def insert(self, index, trackOrPhrase):
        """
        Insert a track into the sequence.
        
        @param index: the position into which to insert the track
        @param trackOrPhrase: a L{SequencerTrack} or L{Phrase} object
        """
        self._validateItem(trackOrPhrase)
        list.insert(self, index, self._makeTrack(trackOrPhrase))
    
    def insertTrack(self, index, phrase=None, channel=1, record=False, 
            mute=False):
        """
        Insert a new track into the list.
        
        @param index: the position into which to insert the track
        @param phrase: if specified, the L{Phrase} that stores the Midi 
            events for the track
        @param channel: the Midi channel for recording and playback of
            the track
        @param record: True if the track is armed for recording; False
            otherwise
        @param mute: True if the track is muted for playback; False
            otherwise
        """
        track = SequencerTrack(channel, record, mute, self._onTrackInput,
            phrase)
        self.insert(index, track)
        return track
