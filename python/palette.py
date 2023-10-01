import requests

import os
import io
import json
import glob
import collections
import time
import signal
import sys
import socket

# import _thread as thread

import threading
import platform
from traceback import format_exc

from requests.adapters import HTTPAdapter
import urllib3
from urllib3 import Retry

ApiSession = None
ApiLock = threading.Lock()

DebugApi = False
Verbose = False

LineSep = "_"

OneBeat = 96

PerformLabels = {}
GlobalPerformLabels = {}

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
    # if DebugApi:
    #     log("palette_patch_api: patch="+patch+" api="+api+" params="+params)
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

def palette_engine_set(name, value):
    return palette_api("engine.set",
            "\"name\": \"" + name + "\"" + \
            ", \"value\": \"" + str(value) + "\"")

def palette_engine_get(name):
    return palette_api("engine.get", "\"name\": \"" + name + "\"")

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

FullDataPath = ""

# This is the name of the data directory containing config and saved
def PaletteDataPath():
    global FullDataPath
    if FullDataPath != "":
        return FullDataPath

    FullDataPath = os.path.join(localPaletteDir(),"data")
    return FullDataPath

# Combine saved in the savedPath list
def savedListAll(savedType,showall):
    savedpath = savedPath()
    paths = savedpath.split(";")
    allvals = []
    for dir in paths:
        saveddir = os.path.join(dir,savedType)
        if os.path.isdir(saveddir):
            vals = listOfJsonFiles(saveddir)
            for v in vals:
                if v in allvals:
                    continue # already in the list
                if v[0] == "_":
                    continue # never show
                # Curly-brace names aren't shown by default
                if showall==True or v[0] != "{":
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

def publish_event(subject,params):
    log("public_event needs work params=",params.encode())

def audio_reset():
    # log("palette.audio_reset")
    palette_engine_api("audio_reset")

import asyncio
import nats
from nats.errors import ConnectionClosedError, TimeoutError, NoServersError

NatsInitialized = False
NatsNc = None
NatsResponse = None

async def palette_nats_connect():
    global NatsNc
    NatsNc = await nats.connect("nats://127.0.0.1:4222")

async def palette_nats_request(subject,msg):
    global NatsResponse
    NatsResponse = await NatsNc.request(subject, msg, timeout=0.5)

async def palette_nats_subscribe(subject,handler):
    await NatsNc.subscribe(subject, handler)

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
        log("palette_api: api=",api," params=",s)

    success = False
    while not success:
        # Acquire lock before sending
        ApiLock.acquire()

        global NatsInitialized
        if NatsInitialized == False:
            # NatsNc = await nats.connect("nats://127.0.0.1:4222")

            loop = asyncio.get_event_loop()
            loop.run_until_complete(asyncio.gather(
                asyncio.ensure_future(palette_nats_connect())
            ))

            log("NatsNc = ",NatsNc)
            NatsInitialized = True

        requestError = None
        try:
            loop = asyncio.get_event_loop()
            p = params.encode('ascii', 'utf-8')
            paramsbytes = bytes(p)
            loop.run_until_complete(asyncio.gather(
                asyncio.ensure_future(palette_nats_request("toengine.api",paramsbytes))
            ))

            global NatsResponse
            result = NatsResponse.data.decode()
            
        except TimeoutError as err:
            print("Request timed out")
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

def GetParam(name):
    value, err = palette_api("engine.get",
        "\"name\": \"" + name + "\"")
    if err != None:
        log("Error in palette.GetParam for "+name)
        return ""
    return value

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

