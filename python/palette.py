import asyncio
import nats
from nats.aio.client import Client as NATS
from nats.aio.errors import ErrTimeout, ErrNoServers
from nats.aio.nuid import NUID

import os
import io
import json
import glob
import collections
import time
import signal

# try:
#     import thread
# except ImportError:
import _thread as thread

import threading
import platform

DebugApi = False
Verbose = False
MyNuid = ""

LineSep = "_"

# resetAfterInactivity = 90.0
resetAfterInactivity = -1

OneBeat = 96

PerformLabels = {}
GlobalPerformLabels = {}

PerformDefaultVal = {} # these values are indexes into PerformLabels

PerformLabels["loopinglength"] = [
    {"label":"Loop Length_8 beats",  "value":8*OneBeat},
    {"label":"Loop Length_16 beats", "value":16*OneBeat},
    {"label":"Loop Length_32 beats", "value":32*OneBeat},
    {"label":"Loop Length_64 beats", "value":64*OneBeat},
    {"label":"Loop Length_4 beats", "value":4*OneBeat},
]
SimpleScales = [
	{"label":"Newage_Scale",    "value":"newage"},
	{"label":"Arabian_Scale",   "value":"arabian"},
	# {"label":"Chromatic_Scale", "value":"chromatic"},
    # {"label":"Dorian_Scale","value":"dorian"},
	{"label":"Fifths_Scale",    "value":"fifths"},
    {"label":"Harminor_Scale",  "value":"harminor"},
    # {"label":"Lydian_Scale","value":"lydian"},
    {"label":"Melminor_Scale",  "value":"melminor"},
    {"label":"Raga_Scale",     "value":"raga1"},
]
PerformScales = [
	{"label":"Newage_Scale",    "value":"newage"},
    # {"label":"Aeolian_Scale",   "value":"aeolian"},
 	{"label":"Arabian_Scale",   "value":"arabian"},
 	{"label":"Chromatic_Scale", "value":"chromatic"},
    # {"label":"Dorian_Scale","value":"dorian"},
 	{"label":"Fifths_Scale",    "value":"fifths"},
    {"label":"Harminor_Scale",  "value":"harminor"},
    # {"label":"Ionian_Scale","value":"ionian"},
    # {"label":"Locrian_Scale",   "value":"locrian"},
    # {"label":"Lydian_Scale","value":"lydian"},
    {"label":"Melminor_Scale",  "value":"melminor"},
    # {"label":"Mixolydian_Scale","value":"mixolydian"},
    {"label":"Phrygian_Scale",  "value":"phrygian"},
    {"label":"Raga_Scale",     "value":"raga1"},
    # {"label":"Raga2_Scale", "value":"raga2"},
    # {"label":"Raga3_Scale", "value":"raga3"},
    # {"label":"Raga4_Scale", "value":"raga4"},
]

PerformLabels["quant"] = [
    {"label":"Fret_Quantize", "value":"frets"},
    {"label":"Pressure_Quantize", "value":"pressure"},
    {"label":"Fixed_Time Quant", "value":"fixed"},
    {"label":"No_Quant",  "value":"none"},
]
PerformLabels["vol"] = [
    {"label":"Pressure_Vol", "value":"pressure"},
    {"label":"Fixed_Vol", "value":"fixed"},
]

PerformLabels["loopingfade"] = [
    {"label":"Loop Fade_Fastest", "value":0.05},
    {"label":"Loop Fade_Faster", "value":0.1},
    {"label":"Loop Fade_Fast", "value":0.2},
    {"label":"Loop Fade_Med",  "value":0.4},
    {"label":"Loop Fade_Slow", "value":0.5},
    {"label":"Loop Fade_Slower", "value":0.6},
    {"label":"Loop Fade_Slowest", "value":0.7},
    {"label":"Loop_Forever", "value":1.0},
]
PerformDefaultVal["loopingfade"] = 0

PerformLabels["loopingonoff"] = [
    {"label":"Looping_is OFF",  "value":"off"},
    # {"label":"Looping_is ON", "value":"recplay"},
    {"label":"Looping_is ON", "value":"recplay"},
    {"label":"Loop_Playback Only", "value":"play"},
]
PerformLabels["midithru"] = [
    {"label":"MIDI Thru_Off",  "value":False},  # default value at startup
    {"label":"MIDI Thru_On",  "value":True},
]
PerformLabels["midisetscale"] = [
    {"label":"MIDI Set Scale_Off",  "value":False},
    {"label":"MIDI Set Scale_On",  "value":True},
]
PerformLabels["midiusescale"] = [
    {"label":"MIDI Use Scale_Off",  "value":False},
    {"label":"MIDI Use Scale_On",  "value":True},
]
PerformLabels["midithruscadjust"] = [
    {"label":"MIDI Thru_Scadjust Off",  "value":False},
    {"label":"MIDI Thru_Scadjust On",  "value":True},
]
PerformLabels["midiquantized"] = [
    {"label":"MIDI Thru_NoQuant",  "value":False},
    {"label":"MIDI Thru_Quant",  "value":True},
]
GlobalPerformLabels["tempo"] = [
    {"label":"Tempo_Normal",  "value":1.0},
    {"label":"Tempo_Slow", "value":0.85},
    {"label":"Tempo_Slower", "value":0.70},
    {"label":"Tempo_Slowest", "value":0.55},
    {"label":"Tempo_Fast", "value":1.5},
    {"label":"Tempo_Faster", "value":2.0},
    {"label":"Tempo_Fastest", "value":4.0},
]
GlobalPerformLabels["transpose"] = [
    {"label":"Transpose_0",  "value":0},
    {"label":"Transpose_3",  "value":3},
    {"label":"Transpose_-2",  "value":-2},
    {"label":"Transpose_5",  "value":5},
]


def palette_region_api(region, meth, params=""):
    if region == "":
        log("palette_region_api: no region specified?  Assuming A")
        region = "A"
    if region == "*":
        log("palette_region_api: What do I do with a * region?")
        region = "A"
    p = "\"region\":\""+region+"\""
    if params == "":
        params = p
    else:
        params = p + "," + params
    return palette_api("region."+meth,params)

def palette_global_api(meth, params=""):
    return palette_api("global."+meth,params)

def configFilePath(nm):
    return os.path.join(paletteSubDir("config"),nm)

def logFilePath(nm):
    return os.path.join(paletteSubDir("logs"),nm)

def paletteSubDir(subdir):
    local = os.environ.get("LOCALAPPDATA")
    if local == None:
        log("Expecting LOCALAPPDATA to be set, assuming .")
        local = "."
    return os.path.join(local, "Palette", subdir)

def presetsPath():
    p = ConfigValue("presetspath")
    p = p.replace("%PALETTE%",PaletteDir())
    lad = os.environ.get("LOCALAPPDATA")
    if lad != None:
        p = p.replace("%LOCALAPPDATA%",lad)
    return p

# Combine presets in the presetsPath list
def presetsListAll(presetType):
    presetspath = presetsPath()
    paths = presetspath.split(";")
    allvals = []
    for dir in paths:
        presetdir = os.path.join(dir,presetType)
        if os.path.isdir(presetdir):
            vals = listOfJsonFiles(presetdir)
            for v in vals:
                if not v in allvals:
                    allvals.append(v)
    sortvals = []
    for v in sorted(allvals):
        sortvals.append(v)
    return sortvals

# This one always returns the local (first) directory in the presetspath,
# which is usually the user's LOCALAPPDATA version
def localPresetsFilePath(presetType, nm, suffix=".json"):
    presetspath = presetsPath()
    paths = presetspath.split(";")
    localdir = paths[0]
    if not os.path.isdir(localdir):
        log("No presets directory?  dir=",localdir)
        localdir = "."
    return os.path.join(localdir,presetType, nm+suffix)

# Look through all the directories in presetspath to find file
def searchPresetsFilePath(presetType, nm, suffix=".json"):
    presetspath = presetsPath()
    paths = presetspath.split(";")
    # the local presets directory is the first one in the path
    finalpath = "."
    for dir in paths:
        if os.path.isdir(dir):
            finalpath = os.path.join(dir,presetType, nm+suffix)
            if os.path.exists(finalpath):
                break
    return finalpath

def readJsonPath(path):
    f = open(path)
    j = json.load(f, object_pairs_hook=collections.OrderedDict)
    f.close()
    return j

def MyNUID():
    global MyNuid
    if MyNuid != "":
        return MyNuid
    path = configFilePath("nuid.json")
    if not os.path.isfile(path):
        log("Missing nuid.json file? path=",path)
        return "MissingNUIDFile"
    nuidjson = readJsonPath(path)
    if "nuid" in nuidjson:
        return nuidjson["nuid"]
    return "NoNUIDInNUIDFile"

def FakeNUID(nuid):
    global MyNuid
    MyNuid = nuid

def boolValueOfString(v):
    return True if (v!=0 and v!="0" and v!="off" and v!="false" and v!="False") else False

ApiLock = threading.Lock()
PythonNUID = MyNUID() + "_python"

def palette_api(meth, params=None):

    fullparams = "{ " + params + "}"
    r1,err = invoke_jsonrpc("palette.api",meth,fullparams)
    if err != None:
        log("API of ",meth," returned err=",err)
    return r1

def palette_publish(subject,params):

    if DebugApi:
        log("palette_publish: params=",params)

    # Acquire lock before sending
    global ApiLock
    ApiLock.acquire()
    try:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        loop.run_until_complete(publish_event(subject,params))
        loop.close()

    except ErrTimeout:
        log("palette_event: publish timed out, subject=%s params=%s\n" % (subject,params))

    ApiLock.release()

async def publish_event(subject,params):
    NC = NATS()
    await NC.connect(servers=["nats://127.0.0.1:4222"])
    await NC.publish(subject, params.encode())
    await NC.close()


def invoke_jsonrpc(subject, api, params):

    global ApiLock

    result = None

    if DebugApi:
        s = params
        lim = 100
        if len(s) > lim:
            s = s[0:lim] + " ..."
        log("invoke_jsonrpc: api=",api," params=",s)

    # Acquire lock before sending
    ApiLock.acquire()
    try:
        if params == None:
            params = "{}"
        escaped = params.replace("\"","\\\"")
        req = "{ \"api\": \"%s\", \"nuid\": \"%s\", \"params\": \"%s\"}" % (api,MyNUID(),escaped)

        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        result = loop.run_until_complete(get_json_response(subject,req))
        loop.close()

    except ErrTimeout:
        log("invoke_jsonrpc: request timed out, subject=%s api=%s\n" % (subject,api))

    ApiLock.release()

    if result == None:
        return None, "No result from calling api=%s params=%s\n" % (api,params)

    resultstr = result.data.decode()
    resultjson = json.loads(resultstr)

    err = None
    if "error" in resultjson:
        err = resultjson["error"]
    res = None
    if "result" in resultjson:
        res = resultjson["result"]

    return (res,err)

async def get_json_response(subject,req):
    NC = NATS()
    await NC.connect(servers=["nats://127.0.0.1:4222"])
    response = await NC.request(subject, req.encode(), timeout=2)
    await NC.close()
    return response

def mergeJsonParams(finalparams,tomerge):
    # The finalparams value contains just the param values, while tomerge contains objects with "value" and "enabled"
    for nm in tomerge:
        if "enabled" in tomerge[nm]:
            if tomerge[nm]["enabled"]:
                finalparams[nm] = tomerge[nm]["value"]
        else:
            finalparams[nm] = tomerge[nm]
    return finalparams

def listOfJsonFiles(dir,ignore=None):
    files = glob.glob(os.path.join(dir, '*.json'))
    names = list(map(lambda x: os.path.basename(x), files))
    names = list(map(lambda x: x.replace(".json", ""), names))
    # Want to make sure we return a sorted list
    files = []
    for n in sorted(names):
        if n != ignore:
            files.append(n)
    return files

def copyFile(frompath,topath):
    ffrom = open(frompath)
    fto = open(topath,"w")
    fto.write(ffrom.read())
    ffrom.close()
    fto.close()

SettingsJson = None
LocalSettingsJson = None

def ConfigValue(s):
    global SettingsJson
    global LocalSettingsJson
    if SettingsJson == None:
        path = configFilePath("settings.json")
        if not os.path.isfile(path):
            log("No file? path=",path)
            return ""
        if Verbose:
            log("Loading ",path)
        SettingsJson = readJsonPath(path)

    if LocalSettingsJson == None:
        path = configFilePath("settings.json")
        if os.path.isfile(path):
            if Verbose:
                log("Loading ",path)
            LocalSettingsJson = readJsonPath(path)

    if LocalSettingsJson != None and s in LocalSettingsJson:
        return LocalSettingsJson[s]
    elif SettingsJson != None and s in SettingsJson:
        return SettingsJson[s]
    else:
        return ""

paletteDir = None
def PaletteDir():
    global paletteDir
    if paletteDir == None:
        paletteDir = os.environ.get("PALETTE")
        if paletteDir == None:
            log("PALETTE environment variable needs to be defined.")
            exit()
    return paletteDir


def SendCursorEvent(cid,ddu,x,y,z):
    event = "cursor_" + ddu
    e = ("{ \"nuid\": \"" + PythonNUID + "\", " + \
        "\"cid\": \"" + str(cid) + "\", " + \
        "\"event\": \"" + event + "\", " + \
        "\"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\" }")  % (x,y,z)
    palette_publish("palette.event",e)

def SendSpriteEvent(cid,x,y,z):
    event = "sprite"
    e = ("{ \"nuid\": \"" + PythonNUID + "\", " + \
        "\"cid\": \"" + str(cid) + "\", " + \
        "\"event\": \"" + event + "\", " + \
        "\"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\" }")  % (x,y,z)
    palette_publish("palette.event",e)

def SendMIDIEvent(device,timesofar,msg):
    bytestr = "0x"
    for b in msg.bytes():
        bytestr += ("%02x" % b)

    event = "midi_" + msg.type
    e = ("{ \"nuid\": \"%s\", " + \
        "\"event\": \"%s\", " + \
        "\"device\": \"%s\", " + \
        "\"time\": \"%f\", " + \
        "\"bytes\": \"%s\" }") % \
            (PythonNUID, event, device, timesofar, bytestr)

    palette_publish("palette.event",e)

def SendMIDITimeReset():
    e = ("{ \"nuid\": \"%s\", " + \
        "\"event\": \"midi_time_reset\" }") % \
            (PythonNUID)
    palette_publish("palette.event",e)

def SendMIDIAudioReset():
    e = ("{ \"nuid\": \"%s\", " + \
        "\"event\": \"midi_audio_reset\" }") % \
            (PythonNUID)
    palette_publish("palette.event",e)

def IgnoreKeyboardInterrupt():
    """
    Sets the response to a SIGINT (keyboard interrupt) to ignore.
    """
    return signal.signal(signal.SIGINT,signal.SIG_IGN)
 
def NoticeKeyboardInterrupt():
    """
    Sets the response to a SIGINT (keyboard interrupt) to the
    default (raise KeyboardInterrupt).
    """
    return signal.signal(signal.SIGINT, signal.default_int_handler)

import logging

# global palettelogger
# global palettehandle

def sprint(*args, end='', **kwargs):
    sio = io.StringIO()
    print(*args, **kwargs, end=end, file=sio)
    return sio.getvalue()

def loginit():
    # logging.basicConfig(filename="gui.log",encoding='utf-8', level=logging.DEBUG)
    logpath = logFilePath("gui.log")
    logging.basicConfig(filename=logpath,
        encoding='utf-8',
        format='%(asctime)s %(message)s',
        level=logging.INFO)

    # global palettelogger
    # palettelogger = logging.getLogger("palette")
    # palettelogger.setLevel(logging.INFO)

    # global palettehandle
    # fh = logging.FileHandler(logpath)
    # formatter = logging.Formatter("%(asctime)s %(message)s")
    # fh.setFormatter(formatter)
    # palettelogger.addHandler(fh)

def log(*args):
    global palettelogger
    s = sprint(*args)
    logging.info(s)

def debug(*args):
    global palettelogger
    s = sprint(*args)
    logging.debug(s)