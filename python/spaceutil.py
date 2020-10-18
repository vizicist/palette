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

guiOscPort = 3943
resolumeOscPort = 7000

PadLayer = {
        "A":"1", "B":"2", "C":"3", "D":"4"
}

DebugApi = False
DebugOsc = False
DebugOscFull = False

def boolValueOfString(v):
    return True if (v!=0 and v!="0" and v!="off" and v!="false" and v!="False") else False

ApiLock = threading.Lock()

def palette_api(meth, params=None):

    subject = "palette."+platform.node()+".api"
    r1,err = invoke_jsonrpc(subject,meth,params)
    if err != None:
        print("API of ",meth," returned err=",err)

    r2 = ""
    palettecentral = ConfigValue("palettecentral")
    if palettecentral != "":
        subject2 = "palette."+palettecentral+".api"
        r2,err = invoke_jsonrpc(subject2,meth,params)
        if err != None:
            print("palettecentral API of ",meth," returned err=",err)

    return r1,r2

def palette_event(params):

    subject = "palette.event"
    global ApiLock

    if DebugApi:
        print("invoke_event: params=",params)

    # Acquire lock before sending
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
        req = "{ \"api\": \"%s\", \"params\": \"%s\"}" % (api,escaped)
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
    # print("invoke_jsonrpc: resultstr=",resultstr)
    resultjson = json.loads(resultstr)
    # print("invoke_jsonrpc: resultjson=",resultjson)

    err = None
    if "error" in resultjson:
        # print("ERROR: %s\n" % (resultjson["error"]))
        err = resultjson["error"]
    res = None
    if "result" in resultjson:
        # print("RESULT: %s\n" % (resultjson["result"]))
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

def readJsonPath(path):
    f = open(path)
    j = json.load(f, object_pairs_hook=collections.OrderedDict)
    f.close()
    return j

global PaletteDir
PaletteDir = os.environ.get("PALETTE")
if PaletteDir == None:
    print("PALETTE environment variable needs to be defined.")
    exit()

global ConfigDir
d = os.environ.get("PALETTECONFIG")
if d == None:
    ConfigDir = os.path.join(PaletteDir, "config")
else:
    ConfigDir = d

global PresetsDir
d = os.environ.get("PALETTEPRESETS")
if d == None:
    PresetsDir = os.path.join(PaletteDir, "presets")
else:
    PresetsDir = d

LocalJson = None

def ConfigValue(s):

    global LocalJson
    if LocalJson == None:
        path = configFilePath("settings.json")
        print("Loading from ",path)
        LocalJson = readJsonPath(configFilePath("settings.json"))

    if s in LocalJson:
        return LocalJson[s]
    else:
        return ""

def configFilePath(nm):
    return os.path.join(ConfigDir, nm)

def paramFilePath(section, nm, suffix=".json"):
    return os.path.join(PresetsDir,section, nm+suffix)

def SendCursorEvent(ddu,x,y,z):
    e = "{ \"region\": \"A\", \"event\": \"cursor." + ddu + "\", \"x\": \"%f\", \"y\": \"%f\", \"z\": \"%f\" }"  % (x,y,z)
    print(e)
    palette_event(e)

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