import requests

import os
import io
import json
import glob
import collections
import time
import signal
import sys

# import _thread as thread

import threading
import platform

DebugApi = False
Verbose = False

LineSep = "_"

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
	{"label":"*Newage_Scale",    "value":"newage"},
	{"label":"*Arabian_Scale",   "value":"arabian"},
	# {"label":"*Chromatic_Scale", "value":"chromatic"},
    # {"label":"*Dorian_Scale","value":"dorian"},
	{"label":"*Fifths_Scale",    "value":"fifths"},
    {"label":"*Harminor_Scale",  "value":"harminor"},
    # {"label":"*Lydian_Scale","value":"lydian"},
    {"label":"*Melminor_Scale",  "value":"melminor"},
    {"label":"*Raga_Scale",     "value":"raga1"},
]
PerformScales = [
	{"label":"*Newage_Scale",    "value":"newage"},
    # {"label":"*Aeolian_Scale",   "value":"aeolian"},
 	{"label":"*Arabian_Scale",   "value":"arabian"},
 	{"label":"*Chromatic_Scale", "value":"chromatic"},
    # {"label":"*Dorian_Scale","value":"dorian"},
 	{"label":"*Fifths_Scale",    "value":"fifths"},
    {"label":"*Harminor_Scale",  "value":"harminor"},
    # {"label":"*Ionian_Scale","value":"ionian"},
    # {"label":"*Locrian_Scale",   "value":"locrian"},
    # {"label":"*Lydian_Scale","value":"lydian"},
    {"label":"*Melminor_Scale",  "value":"melminor"},
    # {"label":"*Mixolydian_Scale","value":"mixolydian"},
    {"label":"*Phrygian_Scale",  "value":"phrygian"},
    {"label":"*Raga_Scale",     "value":"raga1"},
    # {"label":"*Raga2_Scale", "value":"raga2"},
    # {"label":"*Raga3_Scale", "value":"raga3"},
    # {"label":"*Raga4_Scale", "value":"raga4"},
]
PerformDefaultVal["scale"] = 0

PerformLabels["quant"] = [
    {"label":"Fret_Quantize", "value":"frets"},
    {"label":"Pressure_Quantize", "value":"pressure"},
    {"label":"Fixed_Time Quant", "value":"fixed"},
    {"label":"No_Quant",  "value":"none"},
]
PerformLabels["vol"] = [
    {"label":"Pressure_Velocity", "value":"pressure"},
    {"label":"Fixed_Vol", "value":"fixed"},
]

PerformLabels["loopingfade"] = [
    {"label":"Loop Fade_Slowest", "value":0.7},
    {"label":"Loop Fade_Slower", "value":0.6},
    {"label":"Loop Fade_Slow", "value":0.5},
    {"label":"Loop Fade_Med",  "value":0.4},
    {"label":"Loop Fade_Fast", "value":0.2},
    {"label":"Loop Fade_Faster", "value":0.1},
    {"label":"Loop Fade_Fastest", "value":0.05},
    {"label":"Loop_Forever", "value":1.0},
]
PerformDefaultVal["loopingfade"] = 7

PerformLabels["deltaztrig"] = [
    {"label":"Retrigger_Pressure OFF", "value":1.0},
    {"label":"Retrigger_Pressure ON", "value":0.1},
]
PerformDefaultVal["deltaztrig"] = 0

PerformLabels["deltaytrig"] = [
    {"label":"Retrigger_Vertical OFF", "value":1.0},
    {"label":"Retrigger_Vertical ON", "value":0.1},
]
PerformDefaultVal["deltaytrig"] = 0

PerformLabels["loopingonoff"] = [
    {"label":"Looping_is OFF",  "value":"off"},
    # {"label":"Looping_is ON", "value":"recplay"},
    {"label":"Looping_is ON", "value":"recplay"},
    {"label":"Loop_Playback Only", "value":"play"},
]

PerformLabels["midithru"] = [
    {"label":"MIDI Thru_On",  "value":True},
    {"label":"MIDI Thru_Off",  "value":False},  # default value at startup
]
PerformDefaultVal["midithru"] = 1

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
    {"label":"*Tempo_Normal",  "value":1.0},
    {"label":"*Tempo_Slow", "value":0.85},
    {"label":"*Tempo_Slower", "value":0.70},
    {"label":"*Tempo_Slowest", "value":0.55},
    {"label":"*Tempo_Fast", "value":1.5},
    {"label":"*Tempo_Faster", "value":2.0},
    {"label":"*Tempo_Fastest", "value":4.0},
]
PerformDefaultVal["tempo"] = 0

# NOTE: the order of these things needs to match
# the order in oneRouter.transposeValues in router.go
GlobalPerformLabels["transpose"] = [
    {"label":"*Transpose_0",  "value":0},
    {"label":"*Transpose_-2",  "value":-2},
    {"label":"*Transpose_3",  "value":3},
    {"label":"*Transpose_-5",  "value":-5},
]
PerformDefaultVal["transpose"] = 0

GlobalPerformLabels["transposeauto"] = [
    {"label":"*Transpose_Auto On",  "value":True},
    {"label":"*Transpose_Auto Off",  "value":False},
]
PerformDefaultVal["transposeauto"] = 0

def log(*args):
    s = sprint(*args)
    if s.endswith("\n"):
        s = s[0:-1]
    print(s)
    sys.stdout.flush()

def palette_region_api(region, api, params=""):
    if region == "":
        log("palette_region_api: no region specified?  Assuming *")
        region = "*"
    p = "\"region\":\""+region+"\""
    if params == "":
        params = p
    else:
        params = p + "," + params
    return palette_api(api,params)

def sprint(*args, end='', **kwargs):
    sio = io.StringIO()
    print(*args, **kwargs, end=end, file=sio)
    return sio.getvalue()

def palette_global_api(api, params=""):
    return palette_api("global."+api,params)

def logFilePath(nm):
    return os.path.join(localPaletteDir(),"logs",nm)

def configFilePath(nm):
    return os.path.join(localPaletteDir(),PaletteDataPath(),"config",nm)

def presetsPath():
    return os.path.join(localPaletteDir(),PaletteDataPath(),"presets")

def localPaletteDir():
    common = os.environ.get("CommonProgramFiles")
    if common == None:
        log("Expecting CommonProgramFiles to be set, assuming .")
        common = "."
    return os.path.join(common,"Palette")

def paletteSubDir(subdir):
    return os.path.join(localPaletteDir(), subdir)

paletteDataPath = ""

# This is the name of the data_* directory
# under which are config and presets.
# The value comes from the local.json file
def PaletteDataPath():
    global paletteDataPath
    if paletteDataPath != "":
        return paletteDataPath

    dir = "data_default"
    # local.json can override it
    path = os.path.join(localPaletteDir(),"local.json")
    if os.path.isfile(path):
        vals = readJsonPath(path)
        if "datapath" in vals:
            dir = vals["datapath"]

    if os.path.dirname(dir) != ".":
        paletteDataPath = dir
    else:
        paletteDataPath = os.path.join(localPaletteDir(),dir)

    return paletteDataPath

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
                if not v in allvals and v[0] != "_":
                    allvals.append(v)
    sortvals = []
    for v in sorted(allvals):
        sortvals.append(v)
    return sortvals

# This one always returns the local (first) directory in the presetspath,
# which is usually the user's CommonProgramFiles version
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

def boolValueOfString(v):
    return True if (v!=0 and v!="0" and v!="off" and v!="false" and v!="False") else False

ApiLock = threading.Lock()
PaletteOutputEventSubject = "palette.output.event"
PaletteInputEventSubject = "palette.input.event"
PaletteAPIEventSubject = "palette.api"

def publish_event(subject,params):
    log("public_event needs work params=",params.encode())

def palette_event(params):
    palette_api("event",params)

def palette_api(api,params):

    global ApiLock

    result = None

    if params != "" and params[0] == "{":
        return None, "palette_api: invalid curly brace in params=%s\n" % (params)
    else:
        if params == "":
            params = "{ \"api\":\""+api+"\" }"
        else:
            params = "{ \"api\":\""+api+"\", "+params+" }"

    if DebugApi:
        s = params
        lim = 100
        if len(s) > lim:
            s = s[0:lim] + " ..."
        log("palette_api: params=",s)

    # Acquire lock before sending
    ApiLock.acquire()

    try:
        url = "http://127.0.0.1:3330/api"
        req = requests.post(url=url,data=params,timeout=10.0)
        result = req.text
    except requests.ConnectionError:
        log("ConnectionError for url=",url)
        result = ""
    except requests.Timeout:
        log("Timeout for url=",url," api=",params)
        result = ""
    except:
        log("Timeout or other exception in palette_api, for api=",params)
        result = ""

    ApiLock.release()

    if result == "":
        return None, "No result from calling params=%s\n" % (params)

    resultjson = json.loads(result)

    err = None
    if "error" in resultjson:
        err = resultjson["error"]
    res = None
    if "result" in resultjson:
        res = resultjson["result"]

    if err != None:
        log("palette_api: api=%s err=%s" % (api,err))

    return (res,err)

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

def ConfigValue(s,defvalue=""):
    global SettingsJson
    if SettingsJson == None:
        path = configFilePath("settings.json")
        if not os.path.isfile(path):
            log("No file? path=",path)
            return defvalue
        if Verbose:
            log("Loading ",path)
        SettingsJson = readJsonPath(path)

    if SettingsJson != None and s in SettingsJson:
        return SettingsJson[s]
    else:
        return defvalue

def ConfigFloat(s,defvalue=0.0):
    s = ConfigValue(s)
    if s == "":
        s = str(defvalue)
    return float(s)

paletteDir = None
def PaletteDir():
    global paletteDir
    if paletteDir == None:
        paletteDir = os.environ.get("PALETTE")
        if paletteDir == None:
            log("PALETTE environment variable needs to be defined.")
            exit()
    return paletteDir


def SendCursorEvent(cid,ddu,x,y,z,region="A"):
    event = "cursor_" + ddu
    e = ("\"region\": \"" + region + "\", " + \
        "\"cid\": \"" + str(cid) + "\", " + \
        "\"event\": \"" + event + "\", " + \
        "\"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\"")  % (x,y,z)
    palette_event(e)

def SendSpriteEvent(cid,x,y,z,region="A"):
    event = "sprite"
    e = ("\"region\": \"" + region + "\", " + \
        "\"cid\": \"" + str(cid) + "\", " + \
        "\"event\": \"" + event + "\", " + \
        "\"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\"")  % (x,y,z)
    palette_event(e)

def SendMIDIEvent(device,timesofar,msg,region="A"):
    bytestr = "0x"
    for b in msg.bytes():
        bytestr += ("%02x" % b)

    e = ("\"event\": \"midi\", " + \
        "\"device\": \"%s\", " + \
        "\"region\": \"" + region + "\", " + \
        "\"time\": \"%f\", " + \
        "\"bytes\": \"%s\"") % \
            (device, timesofar, bytestr)

    palette_event(e)

def SendMIDITimeReset():
    palette_event("\"event\": \"midi_reset\"")

def SendMIDIAudioReset():
    palette_event("\"event\": \"audio_reset\"")

def IgnoreKeyboardInterrupt():
    """
    Sets the response to a SIGINT (keyboard interrupt) to ignore.
    """
    return signal.signal(signal.SIGINT,signal.SIG_IGN)
 
def NoticeKeyboardInterrupt(sighandler):
    """
    Sets the response to a SIGINT (keyboard interrupt)
    """
    return signal.signal(signal.SIGINT, sighandler)

