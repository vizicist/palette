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

PerformLabels["quantstyle"] = [
    {"label":"Fret_Quantize", "value":"frets"},
    {"label":"Pressure_Quantize", "value":"pressure"},
    {"label":"Fixed_Time Quant", "value":"fixed"},
    {"label":"No_Quant",  "value":"none"},
]
PerformLabels["volstyle"] = [
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

def add_to_params(params,p):
    if params == "":
        return p
    return p + "," + params

def palette_patch_api(patch, api, params=""):
    if patch == "":
        log("palette_patch_api: no patch specified?")
    if DebugApi:
        log("palette_patch_api: patch="+patch+" api="+api+" params="+params)
    return palette_api("patch."+api,add_to_params(params,"\"patch\":\""+patch+"\""))

def palette_patch_set(patch, name, value):
    if patch == "":
        log("palette_patch_set: no patch specified?")
    return palette_api("patch.set",
            "\"patch\": \"" + patch + "\"" + \
            ", \"name\": \"" + name + "\"" + \
            ", \"value\": \"" + str(value) + "\"")

def palette_quadpro_api(api, params=""):
    return palette_api("quadpro."+api,params)

def palette_quadpro_set(name, value):
    return palette_api("quadpro.set",
            "\"name\": \"" + name + "\"" + \
            ", \"value\": \"" + str(value) + "\"")

def sprint(*args, end='', **kwargs):
    sio = io.StringIO()
    print(*args, **kwargs, end=end, file=sio)
    return sio.getvalue()

def palette_engine_api(api, params=""):
    return palette_api("engine."+api,params)

def logFilePath(nm):
    return os.path.join(localPaletteDir(),"logs",nm)

def configFilePath(nm):
    return os.path.join(localPaletteDir(),PaletteDataPath(),"config",nm)

def engineFilePath(nm):
    return os.path.join(savedPath(),"engine",nm)

def savedPath():
    return os.path.join(localPaletteDir(),PaletteDataPath(),"saved")

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
# under which are config and saved.
# The value comes from the local.json file
def PaletteDataPath():
    global paletteDataPath
    if paletteDataPath != "":
        return paletteDataPath

    datapath = os.path.join(localPaletteDir(),"data_omnisphere")
    localjsonpath = os.path.join(localPaletteDir(),"local.json")
    if os.path.isfile(localjsonpath):
        vals = readJsonPath(localjsonpath)
        if "datapath" in vals:
            datapath = vals["datapath"]

    paletteDataPath = datapath
    return paletteDataPath

# Combine saved in the savedPath list
def savedListAll(savedType):
    savedpath = savedPath()
    paths = savedpath.split(";")
    allvals = []
    for dir in paths:
        saveddir = os.path.join(dir,savedType)
        if os.path.isdir(saveddir):
            vals = listOfJsonFiles(saveddir)
            for v in vals:
                if not v in allvals and v[0] != "_":
                    allvals.append(v)
    sortvals = []
    for v in sorted(allvals):
        sortvals.append(v)
    return sortvals

# This one always returns the local (first) directory in the savedpath,
# which is usually the user's CommonProgramFiles version
def localSavedFilePath(savedType, nm, suffix=".json"):
    savedpath = savedPath()
    paths = savedpath.split(";")
    localdir = paths[0]
    if not os.path.isdir(localdir):
        log("No saved directory?  dir=",localdir)
        localdir = "."
    return os.path.join(localdir,savedType, nm+suffix)

# Look through all the directories in savedpath to find file
def searchSavedFilePath(savedType, nm, suffix=".json"):
    savedpath = savedPath()
    paths = savedpath.split(";")
    # the local saved directory is the first one in the path
    finalpath = "."
    for dir in paths:
        if os.path.isdir(dir):
            finalpath = os.path.join(dir,savedType, nm+suffix)
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

    success = False
    while not success:
        # Acquire lock before sending
        ApiLock.acquire()

        requestError = None
        try:
            url = "http://127.0.0.1:3330/api"
            req = requests.post(url=url,data=params,timeout=600.0)
            result = req.text
        except (requests.ConnectionError,requests.Timeout,Exception) as err:
            requestError = err
    
        ApiLock.release()

        if requestError == None:
            success = True
        else:
            # log("palette_api: Exception = "+str(requestError))
            log("palette_api: failed connection, api=%s is being retried" % api)

    if result == "":
        log("palette_api: result is empty?")
        result = "{}"

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
        path = engineFilePath("default.json")
        if not os.path.isfile(path):
            log("No file? path=",path)
            return defvalue
        if Verbose:
            log("Loading ",path)
        SettingsJson = readJsonPath(path)

    if SettingsJson != None and "params" in SettingsJson and s in SettingsJson["params"]:
        return SettingsJson[s]
    else:
        return defvalue

paletteDir = None
def PaletteDir():
    global paletteDir
    if paletteDir == None:
        paletteDir = os.environ.get("PALETTE")
        if paletteDir == None:
            log("PALETTE environment variable needs to be defined.")
            exit()
    return paletteDir

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

