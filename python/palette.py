import asyncio
import nats
from nats.aio.client import Client as NATS
from nats.aio.errors import ErrTimeout, ErrNoServers
from nats.aio.nuid import NUID

import os
import json
import glob
import collections
import time
import signal
from urllib import request, parse
try:
    import thread
except ImportError:
    import _thread as thread
import threading
import platform

PadLayer = {
        "A":"1", "B":"2", "C":"3", "D":"4"
}

DebugApi = False
Verbose = False
MyNuid = ""

def localconfigFilePath(nm):
    return os.path.join(localAppDataDir(), "config", nm)

def configFilePath(nm):
    # If PALETTESOURCE is defined, we use
    ps = os.environ.get("PALETTESOURCE")
    if ps != "":
        print("Using PALETTESOURCE to get configFilePath")
        return os.path.join(ps, "default", "config", nm)
    else:
        return os.path.join(PaletteDir(), "config", nm)

def localAppDataDir():
    local = os.environ.get("LOCALAPPDATA")
    if local == None:
        print("Expecting LOCALAPPDATA to be set, assuming .")
        local = "."
    return os.path.join(local,"Palette")

def presetsPath():
    p = ConfigValue("presetspath")
    p = p.replace("%PALETTE%",PaletteDir())
    p = p.replace("%LOCALAPPDATA%",os.environ.get("LOCALAPPDATA"))
    return p

# Combine presets in the presetsPath list
def presetsListAll(section):
    presetspath = presetsPath()
    paths = presetspath.split(";")
    allvals = []
    for dir in paths:
        sectiondir = os.path.join(dir,section)
        if os.path.isdir(sectiondir):
            vals = listOfJsonFiles(sectiondir)
            for v in vals:
                if not v in allvals:
                    allvals.append(v)
    sortvals = []
    for v in sorted(allvals):
        sortvals.append(v)
    return sortvals

# This one always returns the local (first) directory in the presetspath
def localPresetsFilePath(section, nm, suffix=".json"):
    presetspath = presetsPath()
    paths = presetspath.split(";")
    localdir = paths[0]
    if not os.path.isdir(localdir):
        print("No presets directory?  dir=",localdir)
        localdir = "."
    return os.path.join(localdir,section, nm+suffix)

# Look through all the directories in presetspath to find file
def presetsFilePath(section, nm, suffix=".json"):
    presetspath = presetsPath()
    paths = presetspath.split(";")
    # the local presets directory is the first one in the path
    finalpath = "."
    for dir in paths:
        if os.path.isdir(dir):
            finalpath = os.path.join(dir,section, nm+suffix)
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
    path = localconfigFilePath("nuid.json")
    if not os.path.isfile(path):
        print("Missing nuid.json file? path=",path)
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
        print("API of ",meth," returned err=",err)
    return r1

def palette_publish(subject,params):

    if DebugApi:
        print("invoke_event: params=",params)

    # Acquire lock before sending
    global ApiLock
    ApiLock.acquire()
    try:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        loop.run_until_complete(publish_event(subject,params))
        loop.close()

    except ErrTimeout:
        print("palette_event: publish timed out, subject=%s params=%s\n" % (subject,params))

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
        print("invoke_jsonrpc: api=",api," params=",params)

    # Acquire lock before sending
    ApiLock.acquire()
    try:
        if params == None:
            params = "{}"
        escaped = params.replace("\"","\\\"")
        req = "{ \"api\": \"%s\", \"nuid\": \"%s\", \"params\": \"%s\"}" % (api,MyNUID(),escaped)
        if DebugApi:
            print("SENDING subject=",subject," req=",req)

        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        result = loop.run_until_complete(get_json_response(subject,req))
        loop.close()

    except ErrTimeout:
        print("invoke_jsonrpc: request timed out, subject=%s api=%s\n" % (subject,api))

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
            print("No file? path=",path)
            return ""
        if Verbose:
            print("Loading ",path)
        SettingsJson = readJsonPath(path)

    if LocalSettingsJson == None:
        path = localconfigFilePath("settings.json")
        if os.path.isfile(path):
            if Verbose:
                print("Loading ",path)
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
            print("PALETTE environment variable needs to be defined.")
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