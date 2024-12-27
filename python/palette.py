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

def palette_quad_api(api, params=""):
    return palette_api("quad."+api,params)

def palette_quad_set(name, value):
    return palette_api("quad.set",
            "\"name\": \"" + name + "\"" + \
            ", \"value\": \"" + str(value) + "\"")

def sprint(*args, end='', **kwargs):
    sio = io.StringIO()
    print(*args, **kwargs, end=end, file=sio)
    return sio.getvalue()

def palette_global_api(api, params=""):
    return palette_api("global."+api,params)

def palette_engine_set(name, value):
    return palette_api("global.set",
            "\"name\": \"" + name + "\"" + \
            ", \"value\": \"" + str(value) + "\"")

def palette_engine_get(name):
    return palette_api("global.get", "\"name\": \"" + name + "\"")

def configFilePath(nm):
    return os.path.join(PaletteDataPath(),"config",nm)

def savedPath():
    return os.path.join(PaletteDataPath(),"saved")

def localPaletteDir():
    common = os.environ.get("CommonProgramFiles")
    if common == None:
        log("Expecting CommonProgramFiles to be set, assuming .")
        common = "."
    return os.path.join(common,"Palette")

def paletteSubDir(subdir):
    return os.path.join(localPaletteDir(), subdir)

paletteDataPath = ""
FullDataPath = ""

# This is the name of the data_* directory
# containing config and saved.
# This logic should be identical to PaletteDataPath() in the Go code
def PaletteDataPath():
    global paletteDataPath
    global FullDataPath # cache the full path

    if FullDataPath != "":
        return FullDataPath

    palette_data = os.environ.get("PALETTE_DATA","omnisphere")
    datadir = "data_" + palette_data

	# If PALETTE_SOURCE is defined, datapath is relative to that
	# otherwise, it's relative to the PALETTE directory in Common Files.
    palette_source = os.environ.get("PALETTE_SOURCE","")
    if palette_source != "":
        FullDataPath = os.path.join(palette_source, datadir)
    else:
        FullDataPath = os.path.join(localPaletteDir(), datadir)
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

# ...     r = session.get("http://httpbin.org/status/503")

def palette_api_setup():
    global ApiSession
    session = requests.Session()
    adapter = HTTPAdapter(max_retries=Retry(total=999, backoff_factor=1, allowed_methods=None, status_forcelist=[429, 500, 502, 503, 504]))
    session.mount("http://", adapter)
    session.mount("https://", adapter)

def audio_reset():
    # log("palette.audio_reset")
    palette_global_api("audio_reset")

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

        requestError = None
        try:
            url = "http://127.0.0.1:3330/api"
            # log("palette_api: pre requests.post")
            req = requests.post(url=url,data=params,timeout=6000.0)
            # log("palette_api: post requests.post")
            result = req.text
        except (requests.ConnectionError,requests.Timeout,Exception) as err:
            log("palette_api: Connection exception!")
            requestError = err
            # log("ConnectionError exception: %s" % format_exc())
        # except:
        #     log("palette_api: unknown exception!?")
        #     log("Unexpected exception: %s" % format_exc())
        #     requestError = "unknown"
    
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
    value, err = palette_api("global.get",
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

