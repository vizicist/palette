# A GUI for Palette

from tkinter import ttk
from tkinter import font
import tkinter as tk

import glob
import os
import sys
import time
import threading
import traceback
import json
import collections
import signal
import pyperclip
import random
from subprocess import call, Popen
from codenamize import codenamize

import palette

signal.signal(signal.SIGINT,signal.SIG_IGN)

global TopContainer
TopContainer = None
global TopApp
TopApp = None
global IsQuad
IsQuad = False
global ShowImport
ShowImport = False

ControlPageNames = {
    "main":"Main",
}

class Params():
    def __init__(self):
        pass

class ProGuiApp(tk.Tk):

    def __init__(self,
            padname,
            padnames,
            visiblepagenames
            ):

        tk.Tk.__init__(self)

        self.AllPageNames = {
                "quad":0,
                "snap":0,
                "sound":0,
                "visual":0,
                "effect":0,
                "misc":0}

        self.VisiblePageNames = visiblepagenames
        self.PadNames = collections.OrderedDict()
        num = 1
        for ch in padnames:
            self.PadNames[ch] = num
            num = num + 1

        self.readParamDefs()

        self.Pads = {}
        for padName in self.PadNames:
            p = Pad(self,padName)
            self.Pads[p] = padName
            p.loadCurrent()

        self.allPadsSelected = False

        self.frames = {}
        self.editPage = {}
        self.performPage = {}
        self.selectorPage = {}
        self.currentPageName = None
        
        self.selectorAction = ""
        self.selectorButtonIndex = 0
        self.selectorValue = ""
        self.activeCursors = {}
        self.activeTime = {}
        self.editMode = False
        self.showAllPages = False
        self.showSound = False
        self.showPadFeedback = True
        self.showCursorFeedback = False

        s = palette.ConfigValue("guilevel")
        if s == "":
            self.defaultGuiLevel = 0
        else:
            self.defaultGuiLevel = int(s)

        self.setGuiLevel(self.defaultGuiLevel)

        self.performHeader = None

        # The values in globalPerformIndex are indexes into palette.PerformLabels
        # and the defaults point to the first entry in the label list
        self.globalPerformIndex = {}
        for s in palette.GlobalPerformLabels:
            self.globalPerformIndex[s] = 0

        self.topContainer = tk.Frame(self, background=palette.ColorBg)
        global TopContainer
        TopContainer = self.topContainer
        global TopApp
        TopApp = self

        self.selectFrame = self.makeSelectFrame(self.topContainer)
        self.performContainer = tk.Frame(self.topContainer,
            highlightbackground=palette.ColorAqua, highlightcolor=palette.ColorAqua, highlightthickness=3)
        self.performHeader = PerformHeader(parent=self.performContainer, controller=self)
        self.startupFrame = self.makeStartupFrame(self.topContainer)

        # These are the pages for performance things
        self.performPage["main"] = PagePerformMain(parent=self.performContainer, controller=self)

        self.winfo_toplevel().title("Palette "+padnames)

        self.escapeCount = 0
        self.lastEscape = time.time()
        self.lastClearLoop = time.time()
        self.resetLastAnything()

        self.topContainer.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.performHeader.pack(side=tk.TOP,fill=tk.X)
        self.setPerformMessage("")
        self.selectPerformPage("main")
        self.selectPage("snap")
        self.resetVisibility()

        self.topContainer.bind_all("<MouseWheel>", self.scrollWheel)

        # select the initial pad
        self.padChooserCallback(padname)

    def scrollWheel(self,event):
        if self.editMode:
            scrollbar = self.editPage[self.currentPageName].scrollbar
        else:
            scrollbar = self.selectorPage[self.currentPageName].scrollbar
        scrollbar.scrollWheel(event)

    def mainLoop(self):
        doneLoading = False
        while True:
            try:
                self.update_idletasks()
                self.update()
            except tk.TclError:
                s = traceback.format_exc()
                if s.find("application has been destroyed") >= 0:
                    print("Application has been closed!")
                else:
                    traceback.print_exc(file=sys.stdout)
                break
            except:
                traceback.print_exc(file=sys.stdout)
                break
    
            time.sleep(0.001)

            now = time.time()

            if doneLoading:
                pass
            elif StartupMode:
                self.selectFrame.place_forget()
                self.performContainer.place_forget()
                self.startupFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=1)
            else:
                # do this once
                if doneLoading == False:
                    self.startupFrame.place_forget()
                    self.resetAll()
                    self.resetVisibility()
                    doneLoading = True
                    for pad in self.Pads:
                        pad.sendParamsOfType("snap")

            if palette.resetAfterInactivity>0 and (now - self.lastAnything) > palette.resetAfterInactivity:
                print("Resetting after no activity!!")
                self.resetLastAnything()

                self.resetAll()

                self.resetVisibility()
                self.selectPerformPage("main")
                self.performPage["main"].updatePerformButtonLabels(self.CurrPad)
    
            reset = True

            if self.selectorAction == "LOAD":
                self.selectorLoadAndSend(self.currentPageName,self.selectorValue,self.selectorButtonIndex)
    
            elif self.selectorAction == "IMPORT":
                self.selectorImportAndSend(self.currentPageName,self.selectorValue)

            elif self.selectorAction == "INIT":
                self.selectorApply("init",self.currentPageName)

            elif self.selectorAction == "RAND":
                self.selectorApply("rand",self.currentPageName)

            else:
                reset = False

            if reset:
                self.resetLastAnything()
            self.selectorAction = ""
    
    def resetLastAnything(self):
        self.lastAnything = time.time()

    def setPerformMessage(self,text):
        if self.performHeader != None:
            self.performHeader.performMessageLabel.config(text=text)

    def resetVisibility(self):
        sh = self.selectHeader
        ch = self.performHeader
        if self.showAllPages:
            for pg in self.VisiblePageNames:
                if palette.IncludeSound == False and not self.showSound and pg == "sound":
                    sh.pageButton[pg].pack_forget()
                else:
                    sh.pageButton[pg].pack(side=tk.LEFT,padx=5)
            for pg in ControlPageNames:
                ch.pageButton[pg].pack_forget()
            ch.performMessageLabel.pack_forget()
        
        # elif palette.RecMode:
        #     for pg in self.VisiblePageNames:
        #         sh.pageButton[pg].pack_forget()
        #     for pg in ControlPageNames:
        #         ch.pageButton[pg].pack_forget()
        #     ch.performMessageLabel.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=50)

        else:
            for pg in self.VisiblePageNames:
                sh.pageButton[pg].pack_forget()
            for pg in ControlPageNames:
                ch.pageButton[pg].pack_forget()
            ch.performMessageLabel.pack_forget()

            sh.pageButton["quad"].pack(side=tk.LEFT)

        self.editMode = False
        global IsQuad
        if IsQuad:
            self.selectPage("quad")
        else:
            self.selectPage("snap")

        pg = self.performPage["main"]

        global pageSizeOfSelect, pageSizeOfControl
        if self.guiLevel == 0:
            pageSizeOfControl = pageSizeOfControlNormal
            pageSizeOfSelect = pageSizeOfSelectNormal
        else:
            pageSizeOfControl = pageSizeOfControlAdvanced
            pageSizeOfSelect = pageSizeOfSelectAdvanced

        y = 0
        self.selectPageY = y
        y += pageSizeOfSelect
        self.performPageY = y
        y += pageSizeOfControl

        # self.selectFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=pageSizeOfSelect)
        self.performContainer.place(in_=self.topContainer, relx=0, rely=self.performPageY, relwidth=1, relheight=pageSizeOfControl)
        self.performContainer.place(in_=self.topContainer, relx=0, rely=self.performPageY, relwidth=1, relheight=pageSizeOfControl)
        self.selectFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=pageSizeOfSelect)

    def saveQuad(self,name):

        quadPath = palette.localPresetsFilePath("quad",name)
        quadValues = {}

        for pad in self.Pads:
            padvalues = pad.getValues()
            for paramName in padvalues:
                fullName = PadParamName(pad.name(),paramName)
                quadValues[fullName] = padvalues[paramName]

        j = { "params": quadValues }
        print("Saving quad in",quadPath)
        SaveJsonInPath(j,quadPath)

    def loadQuad(self,presetName):
        fpath = palette.searchPresetsFilePath("quad", presetName)
        print("readQuadPreset: fpath=",fpath)
        try:
            f = open(fpath)
        except:
            print("No such file?  CC fpath=",fpath)
            return
        j = json.load(f)
        self.loadQuadJson(j)
        f.close()

    def loadQuadJson(self,quadJson):

        if self.editMode:
            print("loadQuadJson shouldn't be called in edit mode")
            return

        # quadJson parameter names take the form A-shape,
        # the first letter is the pad name.
        quadParams = quadJson["params"]

        for pad in self.Pads:

            # If parameters exist that aren't in quadJson,
            # use their default value to j.
            # This helps when new parameters are added
            # that aren't in existing preset files.
            pad.setInitValues()

            for fullname in quadParams:
                (padnameofparam,baseparam) = padOfParam(fullname)
                if padnameofparam == pad.name():
                    pad.setValue(baseparam,quadParams[fullname])

    def makeSelectFrame(self,container):

         # This is the area at the very top
        f = tk.Frame(container,
            highlightbackground=palette.ColorAqua, highlightcolor=palette.ColorAqua, highlightthickness=3)

        self.selectHeader = SelectHeader(parent=f, controller=self)
        self.selectHeader.pack(side=tk.TOP,fill=tk.BOTH)

        # These are the pages of buttons for selecting set/patch/sound/visual/etc..
        # Each one has a SelectorPage with the preset buttons,
        # and an EditPage with all the parameters of the current preset
        for pagename in self.VisiblePageNames:
            self.makeSelectorPage(f, pagename, PageSelector)
            self.makeEditPage(f,pagename)

        return f

    def makeStartupFrame(self,container):
        f = tk.Frame(container,
            highlightbackground=palette.ColorBg, highlightcolor=palette.ColorAqua, highlightthickness=3)
        self.startupLabel = ttk.Label(f, text="               Palette is Loading...", style='Header.TLabel',
            foreground=palette.ColorText, background=palette.ColorBg, relief="flat", justify=tk.CENTER, font=palette.largestFont)
        self.startupLabel.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        return f

    def updateSelectorPage(self,pagename,files):
        page = self.selectorPage[pagename]
        page.vals = files
        page.doLayout()
       
    def makeSelectorPage(self,parent,pagename,pagemaker):
        vals = palette.presetsListAll(pagename)

        page = pagemaker(parent, self, vals, pagename)

        self.selectorPage[pagename] = page
        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

    def makeEditPage(self,parent,pagename):
        page = PageEditParams(parent=parent, controller=self,
                    pagename=pagename, params=self.paramsOfType[pagename])
        self.editPage[pagename] = page
        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

    def forgetPages(self,pages):
        for pg in pages:
            pages[pg].pack_forget()

    def togglePageButtons(self):
        # if self.advancedLevel == 0:
        #     return
        self.showAllPages = not self.showAllPages
        self.resetVisibility()

    def clickPage(self,pagename):

        # A second click on a page header will toggle editMode if GuiLevel>1
        if self.guiLevel > 1 and self.currentPageName == pagename:
            self.editMode = not self.editMode

        self.selectPage(pagename)

        if pagename != "quad" and self.editMode:
            if self.allPadsSelected:
                print("allPadsSelected: using values from Pad A")
                pad = self.padNamed("A")
            else:
                pad = self.CurrPad

            page = self.editPage[pagename]
            page.setValues(pad.getValues())

    def padNamed(self,padName):
        lastResort = None
        for pad in self.Pads:
            lastResort = pad
            if pad.name() == padName:
                return pad
        print("There is no Pad named",padName,", last resort, using",lastResort.name())
        return lastResort

    def selectPage(self,pagename):

        self.currentPageName = pagename
        self.selectHeader.highlightPageButton(pagename)

        self.forgetPages(self.selectorPage)
        self.forgetPages(self.editPage)

        if pagename == "quad":
            # we don't want to show the PadChooser when we're on the Quad page
            self.selectHeader.forgetPadChooser()
        else:
            self.selectHeader.placePadChooser()

        if self.guiLevel > 1 and self.editMode:
            page = self.editPage[pagename]
        else:
            page = self.selectorPage[pagename]

        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        page.tkraise()

    def selectPerformPage(self,pagename):
        self.currentPerformPageName = pagename
        self.performHeader.highlightPageButton(pagename)
        for pg in self.performPage:
            if pg == pagename:
                self.performPage[pg].pack(side=tk.TOP,fill=tk.BOTH,expand=True)
            else:
                self.performPage[pg].pack_forget()

        self.performPage[pagename].tkraise()

    def paramCallback(self,paramname,newval):

        # print("paramCallback! paramname=",paramname," newval=",newval)

        (pad,baseparam) = padOfParam(paramname)
        pagename = self.allParamsJson[baseparam]["paramstype"]

        if pagename == "quad":
            print("HEY, DOES quad need some work here? BBB")
        else:
            self.changeAndSendValue(baseparam,newval)

    def sendParamValues(self,values):
        print("sendParamValues: ",values)
        for name in values:
            v = values[name]
            if self.allPadsSelected:
                for pad in self.Pads:
                    pad.sendParamValue(name,v)
            else:
                self.CurrPad.sendParamValue(name,v)

    def changeAndSendValue(self,basename,newval):
        if self.allPadsSelected:
            for pad in self.Pads:
                pad.setValue(basename,newval)
                pad.sendValue(basename)
        else:
            self.CurrPad.setValue(basename,newval)
            self.CurrPad.sendValue(basename)

        # if self.editMode:
        #     page = self.editPage[self.currentPageName]
        #     page.changeValueText(basename,newval)

    def selectorApply(self,apply,paramType):

        if not self.editMode:
            print("Hmmm, selectorAply should only be called in editMode")
            return

        if paramType == "quad":
            print("selectorApply not implemented on quad")
        else:

            self.applyToAllParams(apply,paramType)

            self.refreshPage()
            self.saveCurrent()

            if self.allPadsSelected:
                print("Sending",paramType,"params to all pads")
                for pad in self.Pads:
                    pad.sendParamsOfType(paramType)
            else:
                print("Sending",paramType,"params to pad",self.CurrPad.name())
                self.CurrPad.sendParamsOfType(paramType)

    def applyToAllParams(self,apply,paramType):
        # loop through all the parameters of a given type
        for name in self.allParamsJson:
            j = self.allParamsJson[name]
            if paramType != "snap" and j["paramstype"] != paramType:
                continue
            valuetype = j["valuetype"]
            (p,basename) = padOfParam(name)
            if p != None:
                print("Unexpected pad in param value?")
                continue
            v = ""
            min = j["min"]
            max = j["max"]
            if "randmin" in j:
                min = j["randmin"]
            if "randmax" in j:
                max = j["randmax"]

            if valuetype == "float":
                if apply == "init":
                    v = j["init"]
                elif apply == "rand":
                    r = float(min) + (random.random() * (float(max)-float(min)))
                    v = "%f" % r
            elif valuetype == "int":
                if apply == "init":
                    v = j["init"]
                elif apply == "rand":
                    r = int(min) + int(random.random() * (int(max)-int(min)+1))
                    v = "%d" % r
            elif valuetype == "bool":
                if apply == "init":
                    v = j["init"]
                elif apply == "rand":
                    # if the max value of a bool is a float number,
                    # it's the probability of being true.
                    try:
                        f = float(max)
                    except:
                        f = 0.5
                    if random.random() <= f:
                        v = "true"
                    else:
                        v = "false"
            elif valuetype == "string":
                if apply == "init":
                    v = j["init"]
                elif apply == "rand":
                    enum = j["min"]
                    if enum in self.paramenums:
                        enums = self.paramenums[enum]
                    if max != min:  # implies that randmax was used
                        if max in enums:
                            v = max
                        else:
                            print("Hey, randmax=",max," isn't a valid enum value for name=",name)
                    else:
                        i = random.randint(0,len(enums)-1)
                        v = enums[i]

            if v != "":
                self.changeAndSendValue(basename,v)

    def saveCurrent(self):
        if self.allPadsSelected:
            for pad in self.Pads:
                pad.saveCurrent()
        else:
            self.CurrPad.saveCurrent()

    def selectorLoadAndSend(self,paramType,presetname,buttoni):

        if self.editMode:
            print("HEY!! selectorLoadAndSend shouldn't be used in editMode?")
            return

        if paramType == "quad":
            print("Should be highlighting buttoni=",buttoni)
            self.loadQuad(presetname)
            self.sendQuad()
        else:
            self.loadOther(paramType,presetname)
            self.sendOther(paramType)

    def sendOther(self,paramType):
        if self.allPadsSelected:
            for pad in self.Pads:
                pad.sendParamsOfType(paramType)
        else:
            self.CurrPad.sendParamsOfType(paramType)

    def selectorImportAndSend(self,paramType,val):
        j = json.loads(val)
        if "paramstype" in j:
            jparamstype = j["paramstype"]
        else:
            jparamstype = "NoValue"

        if jparamstype != paramType:
            # this error will be common, need a visible message
            print("Mismatched paramstype in JSON!")
            return

        if paramType == "snap":
            self.loadPageJson(self.editPage["snap"],j)
            self.sendPage(self.editPage["snap"])
        elif paramType == "quad":
            print("Hey, does quad need work here?  FFF")
        else:
            self.readOtherParamsJsonIntoSnapAndQuad(paramType,j)
            print("Sending",paramType,"params to pad",self.CurrPad.name())
            self.CurrPad.sendParamsOfType(paramType)

    def padChooserCallback(self,padName):
        self.CurrPad = self.padNamed(padName)

        self.allPadsSelected = False

        if len(self.PadNames) > 1:
            self.selectHeader.padChooser.refreshPadColors()

        self.refreshPage()

        performControl = self.performPage["main"]
        performControl.updatePerformButtonLabels(self.CurrPad)

        self.editPage[self.currentPageName].updateParamView()

    def copyPadToPage(self,pad,pageName):
        padvalues = pad.getValues()
        page = self.editPage[pageName]
        for nm in padvalues:
            page.changeValueText(nm,padvalues[nm])

    def loadOther(self,paramType,presetname):

        # Load values into the params of the currently-selected Pads
        if self.allPadsSelected:
            for pad in self.Pads:
                pad.loadValues(paramType,presetname)
        else:
            self.CurrPad.loadValues(paramType,presetname)

        # load values into the edit page
        # page = self.editPage[pagename]
        # page.loadOtherPreset(presetname)

    def refreshPage(self):
        if self.editMode:
            # If we're in edit mode,
            # make sure the values are updated from the Pad values
            if self.allPadsSelected:
                self.copyPadToPage(self.padNamed("A"),self.currentPageName)
            else:
                self.copyPadToPage(self.CurrPad,self.currentPageName)

    def oldnextValue(self,arr,v):
        found = -1
        for i in range(len(arr)):
            if arr[i]["value"] == v["value"]:
                found = i
                break
        found = (found + 1) % len(arr)
        return arr[found]

    def sendGlobalPerformVal(self,name):

        index = self.globalPerformIndex[name]
        val = palette.GlobalPerformLabels[name][index]["value"]

        if name == "tempo":
            palette.palette_global_api("set_tempo_factor", "\"value\": \""+str(val) + "\"")

        # elif name == "configname":
        #     config = self.globalPerformIndex["configname"]["value"]
        #     palette.setConfigName(config)
        #     print("CONFIGNAME setting to ",palette.getConfigName())

    def combPadLoop(self,pad):
        palette.palette_region_api(self.CurrPad.name(), "loop_comb", "")

    def combLoop(self):
        self.resetLastAnything()
        self.combPadLoop(self.CurrPad.name())

    def clearLoop(self):
        self.resetLastAnything()

        # click the Loop Clear button 4 times quickly to change the GuiLevel
        tm = time.time()
        since = tm - self.lastClearLoop
        self.lastClearLoop = tm
        if since < 0.5:
            self.escapeCount += 1
            if self.escapeCount > 2:
                self.cycleGuiLevel()
                self.escapeCount = 0
        else:
            self.escapeCount = 0

        if self.allPadsSelected:
            for pad in self.Pads:
                pad.clearLoop()
        else:
            self.CurrPad.clearLoop()

    def cycleGuiLevel(self):
        # cycle through 0,1,2
        self.setGuiLevel((self.guiLevel + 1) % 3)
        self.resetVisibility()
        self.performPage["main"].updatePerformButtonLabels(self.CurrPad)

    def setGuiLevel(self,level):
        print("Setting GuiLevel to",level)
        self.guiLevel = level
        if self.guiLevel == 0:
            self.showAllPages = False
        elif self.guiLevel == 1:
            self.showAllPages = True
        elif self.guiLevel == 2:
            self.showAllPages = True
        else:
            print("Unrecognized guiLevel value: ",level)

    def resetAll(self):

        palette.palette_global_api("audio_reset")

        self.setGuiLevel(self.defaultGuiLevel)
        self.resetLastAnything()
        self.sendANO()
        self.clearExternalScale()

        self.CurrPad = self.padNamed("A")
        self.selectHeader.padChooser.refreshPadColors()

        for pad in self.Pads:
            for name in palette.PerformLabels:
                pad.sendPerformVal(name)

        for name in palette.GlobalPerformLabels:
            self.sendGlobalPerformVal(name)

        self.setPerformMessage("")
        for pad in self.Pads:
            pad.clearLoop()
            pad.setDefaultPerform()

        self.performPage["main"].updatePerformButtonLabels(self.CurrPad)

        self.resetVisibility()

    def clearExternalScale(self):
        palette.palette_region_api(self.CurrPad.name(), "clearexternalscale")

    def sendANO(self):
        palette.palette_region_api(self.CurrPad.name(), "ANO")

    def sendQuad(self):
        for pad in self.Pads:
            print("Sending all parameters for pad = ",pad.name())
            for pt in ["sound","visual","effect"]:
                paramlistjson = pad.paramListOfType(pt)
                palette.palette_region_api(pad.name(), pt+".set_params", paramlistjson)

    # def padParamListJson(self,paramType,padName):
    #    pad = self.Pads[padName]
    #    paramlist = pad.paramListOfType(paramType)
    #    return paramlist

    def synthesizeParamsJson(self):

        # rework the contents of paramdefs.json so that
        # each "effect.*" parameter is followed by a copy
        # with "2" appended to the parameter name.
        # This is needed because in Resolume, there are
        # 2 copies of each effect (one chain of effects is
        # applied, and then the same set is applied again, in series).
        # This is a cheap way of allowing some of the later effects
        # to be applied before the earlier effects, even though
        # it doesn't allow total freedom in the effect ordering.

        s = ""
        path = palette.configFilePath("paramdefs.json")
        print("Reading path=",path)
        f = open(path,'r')
        lines = f.readlines() 
        for line in lines: 
            if "effect." in line:
                s += line.replace("effect.","effect.1-")
                s += line.replace("effect.","effect.2-")
            else:
                s += line
        j = json.loads(s, object_pairs_hook=collections.OrderedDict)
        return j

    def readParamDefs(self):

        # we assume newParamsJson an OrderedDict
        self.newParamsJson = self.synthesizeParamsJson()
        self.allParamsJson = self.convertParamdefsToParams(self.newParamsJson)

        self.paramenums = palette.readJsonPath(palette.configFilePath("paramenums.json"))
        self.allEffectsJson = palette.readJsonPath(palette.configFilePath("resolume.json"))
        self.paramValueTypeOf = {}
        self.paramsOfType = {}
        self.paramTypeOf = {}
        for name in self.allParamsJson:
            self.paramValueTypeOf[name] = self.allParamsJson[name]["valuetype"]

        # Construct lists of the parameters, pulled from Params.json
        for t in self.AllPageNames:
            self.paramsOfType[t] = collections.OrderedDict()

        self.allParamNames = []
        for x in sorted(self.allParamsJson.keys()):
            self.allParamNames.append(x)
            self.allParamsJson[x]["name"] = x
            t = self.allParamsJson[x]["paramstype"]
            if t != "channel":
                self.paramsOfType[t][x] = self.allParamsJson[x]
                self.paramTypeOf[x] = self.allParamsJson[x]["paramstype"]

        # In addition to creating parameters for "snap",
        # we create all the parameters for the "quad" settings by
        # duplicating all the parameters for each pad (A,B,C,D).
        for name in self.allParamNames:
            # paramType = self.allParamsJson[name]["paramstype"]
            self.paramValueTypeOf[name] = self.allParamsJson[name]["valuetype"]
            self.paramsOfType["snap"][name] = self.allParamsJson[name]

            global IsQuad
            if IsQuad:
                # We prepend A-, B-, etc to the parameter name for quad parameters,
                # to create entries for "quad" things
                # in paramValueTypeOf and paramsOfType["quad"]
                for pad in self.PadNames:
                    quadName = PadParamName(pad,name)
                    self.paramValueTypeOf[quadName] = self.allParamsJson[name]["valuetype"]
                    self.paramsOfType["quad"][quadName] = self.allParamsJson[name]

        # The things here get ADDED to the ones already read in from paramenums.json
        for pt in {"sound", "visual", "effect"}:
            if pt in self.paramenums:
                print("WARNING! pt=",pt," is already in paramenums.json!")
            else:
                self.paramenums[pt] = palette.presetsListAll(pt)

        j = palette.readJsonPath(palette.configFilePath("synths.json"))

        self.paramenums["synth"] = []
        names = []
        for o in j["synths"]:
            names.append(o["name"])
        for nm in sorted(names):
            self.paramenums["synth"].append(nm)

    # XXX - Someday, convert all the code to eliminate this.
    def convertParamdefsToParams(self,newparamsjson):
        # This silliness is to avoid needing to convert all the other
        # code that assumes the structure that was in the old Params.json file.
        allparamsjson = {}
        for name in newparamsjson:
            parts = name.split(".")
            if len(parts) != 2:
                print("Unable to handle param name: ",name)
                continue
            paramstype = parts[0]
            parambasename = parts[1]
            allparamsjson[parambasename] = {
                "paramstype": paramstype
            }
            for pn in newparamsjson[name]:
                allparamsjson[parambasename][pn] = newparamsjson[name][pn]

        return allparamsjson

class Pad():

    def __init__(self, controller, padName):
        self.paramValues = {}
        self.controller = controller
        self.params = self.controller.paramsOfType["snap"]
        self.setInitValues()
        self.snapPath = CurrentPadPath(padName)
        self.padName = padName
        self.setDefaultPerform()

    def setDefaultPerform(self):
        # The values in performIndex are indexes into the palette.PerformLabels array.
        self.performIndex = {}
        # Default value of performIndexs is 0
        for name in palette.PerformLabels:
            self.performIndex[name] = 0
        # but there can be exceptions, specified in PerformDefaultVal
        for name in palette.PerformDefaultVal:
            self.performIndex[name] = palette.PerformDefaultVal[name]

    def name(self):
        return self.padName

    def setInitValues(self):
        for paramName in self.params:
            self.paramValues[paramName] = self.params[paramName]["init"]

    def getValues(self):
        return self.paramValues

    def loadValues(self,paramType,presetname):
        self.loadPresetValues(paramType,presetname)
        self.saveCurrent()
 
    def loadCurrent(self):
        path = self.snapPath
        if not self.loadFile(path):
            print("No Current settings for pad=",self.padName," path=",path)
        else:
            print("Loaded pad=",self.padName," from path=",path)

    def loadPresetValues(self,paramType,presetName):
        fpath = palette.searchPresetsFilePath(paramType, presetName)
        if not self.loadFile(fpath):
            print("Unable to load preset file: ",fpath)
        else:
            print("Loaded Pad",self.padName,"from",fpath)

    def loadFile(self,path):
        try:
            f = open(path)
            j = json.load(f)
            self.loadJson(j)
            f.close()
            return True
        except:
            return False

    def saveCurrent(self):
        j = { "params": self.paramValues }
        print("Saving pad",self.padName,"in",self.snapPath)
        SaveJsonInPath(j,self.snapPath)

    def loadJson(self,j):
        # the self.params has all (i.e. snap) parameters, but
        # we only want to load whatever's in the json we're given.
        for paramName in j["params"]:
            self.setValue(paramName,j["params"][paramName])

    def setValue(self,paramName,val):
        if not paramName in self.paramValues:
            print("Hey, Pad.setValue paramName=",paramName,"not in paramValues?")
            return
        self.paramValues[paramName] = val
 
    def getValue(self,paramName):
        return self.paramValues[paramName]
 
    def sendValue(self,paramName):
        if not paramName in self.paramValues:
            print("Hey, Pad.sendValue paramName=",paramName,"not in paramValues?")
            return
        val = self.paramValues[paramName]
        if not paramName in self.controller.paramTypeOf:
            print("Unrecognized parameter: ",paramName)
            return
        paramType = self.controller.paramTypeOf[paramName]
        palette.palette_region_api(self.padName,paramType+".set_param",
            "\"param\": \"" + paramName + "\"" + \
            ", \"value\": \"" + str(val) + "\"" )

    def sendParamsOfType(self,paramType):
        for pt in ["sound","visual","effect"]:
            if paramType == "snap" or paramType == pt:
                paramlistjson = self.paramListOfType(pt)
                palette.palette_region_api(self.name(), pt+".set_params", paramlistjson)

        if paramType == "snap":
            for name in palette.PerformLabels:
                self.sendPerformVal(name)

    def getPerformIndex(self,name):
        return self.performIndex[name]

    def setPerformIndex(self,name,index):
        self.performIndex[name] = index

    def sendPerformVal(self,name):
        index = self.performIndex[name]
        labels = palette.PerformLabels[name]
        val = labels[index]["value"]
        if name == "loopingonoff":
            reconoff = False
            playonoff = False
            if val == "off":
                pass
            elif val == "recplay":
                reconoff = True
                playonoff = True
            elif val == "play":
                reconoff = False
                playonoff = True
            else:
                print("Unrecognized value of loopingonoff - %s\n" % val)
                return

            palette.palette_region_api(self.name(), "loop_recording", '"onoff": "'+str(reconoff)+'"')
            palette.palette_region_api(self.name(), "loop_playing", '"onoff": "'+str(playonoff)+'"')

        elif name == "loopinglength":
            palette.palette_region_api(self.name(), "loop_length", '"length": "'+str(val)+'"')

        elif name == "loopingfade":
            palette.palette_region_api(self.name(), "loop_fade", '"fadelength": "'+str(val)+'"')

        elif name == "quant":
            palette.palette_region_api(self.name(), "set_param",
                "\"param\": \"" + "misc.quant" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")
        elif name == "scale":
            palette.palette_region_api(self.name(), "set_param",
                "\"param\": \"" + "misc.scale" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")

        elif name == "vol":
            # NOTE: "voltype" here rather than "vol" - should make consistent someday
            palette.palette_region_api(self.name(), "set_param",
                "\"param\": \"" + "misc.vol" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")

        elif name == "comb":
            val = 1.0
            palette.palette_region_api(self.name(), "loop_comb",
                "\"value\": \"" + str(val) + "\"")

        elif name == "midithru":
            palette.palette_region_api(self.name(), "midi_thru", "\"thru\": \"" + str(val) + "\"")

        elif name == "useexternalscale":
            palette.palette_region_api(self.name(), "useexternalscale", "\"onoff\": \"" + str(val) + "\"")

        elif name == "midiquantized":
            palette.palette_region_api(self.name(), "midi_quantized", "\"quantized\": \"" + str(val) + "\"")

        elif name == "transpose":
            palette.palette_region_api(self.name(), "set_transpose", "\"value\": \""+str(val) + "\"")

    def clearLoop(self):
        palette.palette_region_api(self.name(), "loop_clear", "")
 
    def paramListOfType(self,paramType):
        paramlist = ""
        sep = ""
        # XXX - should this use self.params ?
        for name in self.controller.allParamsJson:
            j = self.controller.allParamsJson[name]
            if paramType == "snap" or j["paramstype"] == paramType:
                # paramname = pad + "_" + name
                paramname = name
                v = self.getValue(paramname)
                paramlist = paramlist + sep + "\"" + name + "\" : \"" + str(v) + "\""
                sep = ", "

        return paramlist

class PadChooser(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)

        self.controller = controller
        self.parent = parent
        self.padLabel = {}
        self.padFrame = {}
        self.padCanvas = {}
        self.canvasHeight = 60
        self.canvasWidth = 200
        self.PadNum2Name = ["X","A","B","C","D"]

        # separator line
        # canvas = tk.Canvas(self, background=palette.ColorAqua, highlightthickness=0, height=4)
        # canvas.pack(side=tk.TOP,fill=tk.X)

        self.makeGlobalButton(self,0.0,0.0)
        self.makePadFrame(self,"A",0.2,0.0)
        self.makePadFrame(self,"B",0.4,0.0)
        self.makePadFrame(self,"C",0.6,0.0)
        self.makePadFrame(self,"D",0.8,0.0)

        self.config(background=palette.ColorBg)

    def makePadFrame(self,parent,pad,x0,y0):

        self.padFrame[pad] = tk.Frame(self)
        self.padFrame[pad].place(relx=x0,rely=y0,relwidth=0.15,relheight=1.0)
        self.padFrame[pad].config(borderwidth=0,relief="solid",background=palette.ColorUnHigh)
        self.padFrame[pad].bind("<Button-1>", lambda p=pad: self.padCallback(p))

        self.padLabel[pad] = ttk.Label(self.padFrame[pad], text=pad)
        self.padLabel[pad].pack(side=tk.TOP)
        self.padLabel[pad].configure(style='ChooserDisabled.TLabel')
        self.padLabel[pad].bind("<Button-1>", lambda p=pad: self.padCallback(p))

        if self.controller.showCursorFeedback:
            self.padCanvas[pad] = tk.Canvas(self.padFrame[pad], width=self.canvasWidth, height=self.canvasHeight, border=0)
            self.padCanvas[pad].pack(side=tk.TOP)
            self.padCanvas[pad].config(background=palette.ColorUnHigh)

    def makeGlobalButton(self,parent,x0,y0):

        self.padGlobalButton = tk.Frame(self)
        self.padGlobalButton.place(relx=x0,rely=y0,relwidth=0.15,relheight=1.0)
        self.padGlobalButton.config(borderwidth=0,relief="solid",background=palette.ColorUnHigh)
        self.padGlobalButton.bind("<Button-1>", self.allPadsSelectedCallback)

        self.padGlobalLabel = ttk.Label(self.padGlobalButton, text="*")
        self.padGlobalLabel.pack(side=tk.TOP)
        self.padGlobalLabel.configure(style='ChooserDisabled.TLabel')
        # self.padGlobalLabel.config(background=palette.ColorUnHigh)
        self.padGlobalLabel.bind("<Button-1>", self.allPadsSelectedCallback)

    def allPadsSelectedCallback(self,e):
        self.controller.allPadsSelected = not self.controller.allPadsSelected
        self.controller.refreshPage()
        self.refreshPadColors()

    def refreshPadColors(self):
        if self.controller.allPadsSelected:
            color = palette.ColorHigh
        else:
            color = palette.ColorUnHigh
        self.padGlobalButton.config(background=color)
        self.padGlobalLabel.config(background=color)
        for p in self.controller.PadNames:
            if self.controller.allPadsSelected or p != self.controller.CurrPad.name():
                self.colorPad(p,color)
            else:
                self.colorPad(p,palette.ColorHigh)

    def colorPad(self,pad,color):
        self.padFrame[pad].config(background=color)
        self.padLabel[pad].config(background=color)
        # self.padLabel[pad].config(background=color)
        if self.controller.showCursorFeedback:
            self.padCanvas[pad].config(background=color)

    def highlightPadBorder(self,pad,highlighted):
        if highlighted:
            w = 4
        else:
            w = 2
        self.padFrame[pad].config(borderwidth=w)

    def drawOval(self,pad,highlighted,x,y,z):
        print("drawOval x=",x," y=",y," z=",z)
        x = x * self.canvasWidth
        y = y * self.canvasHeight
        z = z * self.canvasWidth
        print("================= adjusted x=",x," y=",y," z=",z)
        if z < 10:
            z = 10
        elif z > (self.canvasWidth/4):
            z = self.canvasWidth/4
        if highlighted:
            color = palette.ColorRed
        else:
            color = self.controller.padColor(pad)
        self.padCanvas[pad].create_oval(x-z,y-z,x+z,y+z,outline=color)
        # self.padFrame[pad].config(background=color)

    def padCallback(self,e):
        for pad in self.padFrame:
            if e.widget == self.padFrame[pad] or e.widget == self.padLabel[pad]:
                self.controller.padChooserCallback(pad)
                self.refreshPadColors()
                return
        print("No pad found in padCallback!?")


class SelectHeader(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=palette.ColorBg)

        self.titleFrame = tk.Frame(self, background=palette.ColorBg)
        self.titleFrame.pack(side=tk.TOP, fill=tk.X, expand=True)

        self.pageButton = {}

        # self.headerButton = ttk.Button(self.titleFrame, text="Preset", style='Header.TLabel',
        #     command=lambda : self.controller.togglePageButtons())
        # self.headerButton.pack(side=tk.LEFT)

        for i in self.controller.VisiblePageNames:
            self.makeHeaderButton(i,self.controller.VisiblePageNames[i])

        global IsQuad
        if IsQuad:
            self.padChooser = PadChooser(parent=parent, controller=controller)
            self.placePadChooser()

    def placePadChooser(self):
        self.padChooser.place(in_=self.titleFrame, relx=0.7, rely=0, relwidth=0.3, relheight=1.0)

    def forgetPadChooser(self):
        self.padChooser.place_forget()

    def spacer(self,height):
        spacer = tk.Canvas(self, background=palette.ColorBg, highlightthickness=0, height=height)
        spacer.pack(side=tk.TOP)

    def makeHeaderButton(self,pageName,pageTitle):
        # print("makeHeaderButton name=",pageName)

        # Hack so that the leftmost button is always Preset
        displayedPageTitle = pageTitle
        global IsQuad
        if IsQuad:
            if pageName == "quad":
                displayedPageTitle = "Preset"
        else:
            if pageName == "snap":
                displayedPageTitle = "Preset"

        self.pageButton[pageName] = ttk.Button(self.titleFrame, text=displayedPageTitle, style='HeaderDisabled.TLabel',
            command=lambda nm=pageName: self.controller.clickPage(nm))
        self.pageButton[pageName].pack(side=tk.LEFT,padx=5)

    def highlightPageButton(self,pagename):
        for nm in self.pageButton:
            if nm == pagename:
                self.pageButton[nm].config(style='HeaderEnabled.TLabel')
            else:
                self.pageButton[nm].config(style='HeaderDisabled.TLabel')

class PerformHeader(tk.Frame):

    def __init__(self, parent, controller):

        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=palette.ColorBg)

        self.titleFrame = tk.Frame(self, background=palette.ColorBg)
        self.titleFrame.pack(side=tk.TOP, fill=tk.X, expand=True)

        self.pageButton = {}
        # self.performHeaderLabel("Control")
        self.headerButton("main","Main")

        self.performHeaderInfo("")

        self.repack()

    def repack(self):
        for pageName in self.pageButton:
            if pageName != "main":
                self.pageButton[pageName].pack_forget()
            else:
                self.pageButton[pageName].pack(side=tk.LEFT,padx=5)

    def spacer(self,height):
        w = tk.Canvas(self, background=palette.ColorBg, highlightthickness=0, height=height)
        w.pack(side=tk.TOP)

    def performHeaderLabel(self,text):
        self.performLabel = ttk.Label(self.titleFrame, text=text, style='PerformHeader.TLabel')
        self.performLabel.pack(side=tk.LEFT)

    def setPerformHeaderLabel(self,text):
        self.performLabel.config(text=text)
        if text == "":
            self.performLabel.pack_forget()
        else:
            self.performLabel.pack(side=tk.LEFT,padx=5)

    def performHeaderInfo(self,text):
        self.performMessageLabel = ttk.Label(self.titleFrame, text=text, background=palette.ColorBg, style='PerformMessage.TLabel')
        # self.performMessageLabel.pack(side=tk.LEFT, padx=25, ipadx=25)

    def headerButton(self,pageName,pageTitle):
        self.pageButton[pageName] = ttk.Button(self.titleFrame, text=pageTitle, style='HeaderDisabled.TLabel',
            command=lambda nm=pageName: self.controller.selectPerformPage(nm))
        # self.pageButton[pageName].pack(side=tk.LEFT,padx=5)

    def highlightPageButton(self,pagename):
        for nm in self.pageButton:
            if nm == pagename:
                self.pageButton[nm].config(style='HeaderEnabled.TLabel')
            else:
                self.pageButton[nm].config(style='HeaderDisabled.TLabel')

class PageEditParams(tk.Frame):

    def __init__(self, parent, controller, pagename, params):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=palette.ColorBg)

        self.params = params
        self.paramsnameVar = tk.StringVar()
        self.paramsname = ""
        self.pagename = pagename

        saveArea = self.makeButtonArea()
        saveArea.pack(side=tk.TOP, fill=tk.X)

        self.updateParamFiles()
        self.paramsFrame = self.makeParamsArea(self)
        self.scrollbar = ScrollBar(parent=self, notify=self)

        # XXX this is old
        # On the "quad" and "snap" pages, the parameter values aren't shown,
        # just the buttons to import/export/save
        # if not (pagename == "quad" or pagename == "snap"):

        if pagename != "quad":
            self.paramsFrame.pack(side=tk.LEFT, pady=0)
            self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=10, padx=5)
            self.updateParamView()

        defname = self.controller.selectorPage[pagename].defaultVal()
        self.setPresetNameInComboBox(defname)

    def updateParamFiles(self):
        files = palette.presetsListAll(self.pagename)
        self.paramFiles = files
        self.comboParamsname.configure(values=self.paramFiles)

    def makeParamsArea(self,container):

        f = tk.Frame(container, background=palette.ColorBg)
        f.config(borderwidth=1, relief="flat")

        self.paramRowName = []

        self.valuesDisplayOffset = 0

        # Create all the parameter widgets.  Each parameter has its own
        # paramValueWidget, paramLabelWidget, and they get placed (or hidden)
        # based on where we are in the list - i.e. self.valuesDisplayOffset
        # However, the buttons for modifying the values are row-specific, not parameter-specific

        self.paramValueWidget = {}
        self.paramLabelWidget = {}
        self.paramAdjustFrame = {}

        for name in self.params:

            # print("MakeParamsArea pagename=",self.pagename," Param=",name)
            self.paramRowName.append(name)
            self.paramLabelWidget[name] = ttk.Label(f, width=22, text=name, style='ParamName.TLabel')
            self.paramLabelWidget[name].config()

            self.paramValueWidget[name] = ttk.Label(f, width=10, anchor=tk.E, style='ParamValue.TLabel')
            self.paramValueWidget[name].bind("<Button-1>", lambda event,nm=name: self.valueClicked(nm))

        # The widgets for << < . . > >> are static, in the displayed rows
        for row in range(0,paramDisplayRows):
            f2 = tk.Frame(f, background=palette.ColorBg)
            self.adjustButton(f2,row,"<<", -3)
            self.adjustButton(f2,row,"<", -2)
            self.adjustButton(f2,row,".", -1)
            self.adjustButton(f2,row,".", 1)
            self.adjustButton(f2,row,">", 2)
            self.adjustButton(f2,row,">>", 3)
            self.paramAdjustFrame[row] = f2

        return f

    def makeButtonArea(self):
        f = tk.Frame(self, background=palette.ColorBg)

        if self.pagename != "quad":
            self.initButton = ttk.Label(f, text="Init", style='Button.TLabel')
            self.initButton.bind("<Button-1>", lambda event:self.initCallback())
            self.initButton.bind("<ButtonRelease-1>", lambda event:self.initRelease())
            self.initButton.pack(side=tk.LEFT, padx=2)

            self.randButton = ttk.Label(f, text="Rnd", style='Button.TLabel')
            self.randButton.bind("<Button-1>", lambda event:self.randCallback())
            self.randButton.bind("<ButtonRelease-1>", lambda event:self.randRelease())
            self.randButton.pack(side=tk.LEFT, padx=2)

        if ShowImport:
            self.importButton = ttk.Label(f, text="Imp", style='Button.TLabel')
            self.importButton.bind("<Button-1>", lambda event:self.saveImportCallback())
            self.importButton.bind("<ButtonRelease-1>", lambda event:self.saveImportRelease())
            self.importButton.pack(side=tk.LEFT, padx=2)

            self.exportButton = ttk.Label(f, text="Exp", style='Button.TLabel')
            self.exportButton.bind("<Button-1>", lambda event:self.saveExportCallback())
            self.exportButton.bind("<ButtonRelease-1>", lambda event:self.saveExportRelease())
            self.exportButton.pack(side=tk.LEFT, padx=2)

        b = ttk.Label(f, text="Save", style='Button.TLabel')
        b.bind("<Button-1>", lambda event:self.saveCallback())
        b.pack(side=tk.LEFT, pady=5, padx=2)

        # The following things don't get placed initially,
        # they're revealed when the Save button is pressed.

        self.comboParamsname = ttk.Combobox(f, textvariable=self.paramsnameVar,
                font=palette.comboFont, style='custom.TCombobox')
        self.comboParamsname.bind("<<ComboboxSelected>>", lambda event,v=self.paramsnameVar : self.checkThenGotoParamsFile(v.get()))
        self.comboParamsname.bind("<Return>", lambda event,v=self.paramsnameVar : self.checkThenGotoParamsFile(v.get()))

        self.okButton = ttk.Label(f, text="OK", style='Button.TLabel')
        self.okButton.bind("<Button-1>", lambda event:self.saveOkCallback())

        self.cancelButton = ttk.Label(f, text="Cancel", style='Button.TLabel')
        self.cancelButton.bind("<Button-1>", lambda event:self.saveCancelCallback())

        return f

    def scrollNotify(self,sfy,tag):
        nparams = len(self.params)
        self.valuesDisplayOffset = int((nparams-paramDisplayRows) * sfy)
        # print("valuesDisplayOffset=",self.valuesDisplayOffset)
        self.updateParamView()

    def updateParamView(self):

        for r in range(0,paramDisplayRows):
            self.paramAdjustFrame[r].grid_forget()

        px = 0
        row = 0
        # print("updateParamView valuesDisplayOffset=",self.valuesDisplayOffset)
        for name in self.params:
            showrow = row - self.valuesDisplayOffset
            showme = (showrow >= 0 and showrow < paramDisplayRows)
            if showme:
                self.paramLabelWidget[name].grid(row=showrow, column=0, sticky=tk.W)
                self.paramValueWidget[name].grid(row=showrow, column=1, padx=px)
                self.paramAdjustFrame[showrow].grid(row=showrow,column=2,sticky=tk.W,padx=px,pady=0)
            else:
                self.paramLabelWidget[name].grid_forget()
                self.paramValueWidget[name].grid_forget()
            row += 1

    def adjustButton(self,frame,row,txt,adj):
        if row < len(self.params):
            # name = self.paramRowName[row]
            w = ttk.Label(frame, text=txt, style='ParamAdjust.TLabel', width=2)
            w.bind("<Button-1>", lambda event,r=row,a=adj: self.adjustValue(r,a))
            w.pack(side=tk.LEFT, padx=4)

    def valueClicked(self,name):
        print("valueClicked! name=",name)

    def adjustValue(self,row,amount):
        # print("adjustValue valuesDisplayOffset=",self.valuesDisplayOffset)
        paramrow = row + self.valuesDisplayOffset
        name = self.paramRowName[paramrow]
        t = self.controller.paramValueTypeOf[name]
        # XXX - should be geting value from controller, not from widget
        widg = self.paramValueWidget[name]
        mn = self.params[name]["min"]
        mx = self.params[name]["max"]
        if t == "bool":
            newval = True if amount>0 else False
        elif t == "int":
            v = int(widg.cget("text"))
            dv = int(mx) - int(mn)
            if amount == -3:
                v = v - (dv/10)
            if amount == -2:
                v = v - (dv/100)
            if amount == -1:
                v = v - 1
            if amount == 1:
                v = v + 1
            if amount == 2:
                v = v + (dv/100)
            if amount == 3:
                v = v + (dv/10)
            newval = v
        elif t == "double" or t == "float":
            v = float(widg.cget("text"))
            dv = float(mx) - float(mn)
            if amount == -3:
                v = v - (dv/10)
            if amount == -2:
                v = v - (dv/100)
            if amount == -1:
                v = v - (dv/1000)
            if amount == 1:
                v = v + (dv/1000)
            if amount == 2:
                v = v + (dv/100)
            if amount == 3:
                v = v + (dv/10)
            # print("amount=",amount," mx=",mx," v=",v)
            newval = v
        elif t == "string":
            v = str(widg.cget("text"))
            vals = self.controller.paramenums[self.params[name]["min"]]
            try:
                i = vals.index(v.strip())
            except:
                print("Unable to find v=",v)
                i = 0
            # print("string v=",v," t=",t," vals=",vals," existing i=",i)
            nvals = len(vals)
            mid = int(nvals/10)
            if amount == -3:
                i = 0
            elif amount == -2:
                i = i - mid
            elif amount == -1:
                i = i - 1
            elif amount == 1:
                i = i + 1
            elif amount == 2:
                i = i + mid
            elif amount == 3:
                i = nvals - 1

            if i < 0:
                i = 0
            elif i >= nvals:
                i = nvals - 1
            newval = vals[i]

        newval = self.normalizeJsonValue(name,newval)
        self.paramValueWidget[name].config(text=newval)

        # self.doAutoSave(name,newval)
        self.controller.paramCallback(name,newval)

    def listOfType(self,typesname):
        return self.controller.paramenums[typesname]

    def getValue(self,name):
        t = self.controller.paramValueTypeOf[name]
        widg = self.paramValueWidget[name]
        s = widg.cget("text")
        if t == "bool":
            if s == "":
                b = False
            else:
                b = palette.boolValueOfString(s)
            s = str(b)
        elif t == "int":
            if s == "":
                s = "0"
        elif t == "double" or t == "float":
            if s == "":
                s = "0.0"
        elif t == "string":
            s = s.strip()

        return s

    def hasParameter(self,name):
        return (name in self.paramValueWidget)

    def setValues(self,values):
        print("Edit page, changing ALL parameters in setValues")
        for name in self.params:
            if name in values:
                self.changeValueText(name,values[name])

    def changeValueText(self,name,v):
        # print("CHANGE VALUE LABEL EDIT PAGE=",self.pagename," name=",name," v=",v)
        if not name in self.paramValueWidget:
            # ignore names not on this page
            return
        # XXX - The controller should be sending text down here, and the controller
        # XXX - should have a getValueText() method that does the stuff here
        widg = self.paramValueWidget[name]
        t = self.controller.paramValueTypeOf[name]
        if t == "double" or t == "float":
            try:
                s = self.normalizeJsonValue(name,v)
            except:
                print("Error when trying convert v=",v)
                traceback.print_exc(file=sys.stdout)
            widg.config(text=s)
        elif t == "int":
            s = "%8d" % int(float(v))  # float() in case value is like 1.0
            widg.config(text=s)
        elif t == "bool":
            v = self.normalizeJsonValue(name,v)
            widg.config(text=v)
        elif t == "string":
            s = "%12s" % str(v)
            widg.config(text=s.strip())
        else:
            raise Exception("Unrecognized paramType value? t="+t)

    def checkThenGotoParamsFile(self, name):
        return

    def setPresetNameInComboBox(self,name):
        self.paramsname = name
        try:
            n = self.paramFiles.index(name)
            self.comboParamsname.current(n)
        except:
            pass

    def loadOtherPreset(self,name):

        path = palette.searchPresetsFilePath(self.pagename,name)
        try:
            f = open(path)
        except:
            print("Unable to load preset: ",path)
            return

        j = json.load(f)
        presetvals = j["params"]

        self.controller.sendParamValues(presetvals)

    def loadSnapNamed(self,name,doLift=True):

        print("\n=== loadSnapNamed ",name)

        self.controller.readSnapParamsFileIntoPage(name,"snap")

        self.comboParamsname.configure(values=self.paramFiles)

        self.setPresetNameInComboBox(name)

        for p in self.params:
            self.changeValue(p,self.getValue(p))

        if doLift:
            self.lift()

    def startEditing(self,name,doLift=True):

        print("=== startEditing pagename=%s name=%s" % (self.pagename,name))
        if self.pagename == "quad":
            print("Are you getting here?")
            # self.controller.readQuadPreset(name)
        else:
            self.controller.readSnapParamsFileIntoPage(name,self.pagename)

        self.comboParamsname.configure(values=self.paramFiles)

        self.setPresetNameInComboBox(name)

        # self.oldStartEditing()

    def forgetAll(self):
        self.comboParamsname.pack_forget()
        self.okButton.pack_forget()
        self.cancelButton.pack_forget()

    def saveCallback(self):
        self.comboParamsname.pack(side=tk.LEFT, padx=0)
        self.okButton.pack(side=tk.LEFT, padx=2)
        self.cancelButton.pack(side=tk.LEFT, padx=2)

    def saveCancelCallback(self):
        self.forgetAll()

    def randCallback(self):
        s = pyperclip.paste()
        self.controller.selectorValue = s
        self.controller.selectorAction = "RAND"
        self.forgetAll()
        s = 'ButtonHigh.TLabel'
        self.randButton.config(style=s)

    def randRelease(self):
        s = 'Button.TLabel'
        self.randButton.config(style=s)

    def initCallback(self):
        s = pyperclip.paste()
        self.controller.selectorValue = s
        self.controller.selectorAction = "INIT"
        self.forgetAll()
        s = 'ButtonHigh.TLabel'
        self.initButton.config(style=s)

    def initRelease(self):
        s = 'Button.TLabel'
        self.initButton.config(style=s)

    def saveExportCallback(self):
        j = self.jsonParamDump()
        j["paramsname"] = self.paramsnameVar.get()
        j["paramstype"] = self.pagename 
        s = json.dumps(j, sort_keys=True, indent=4, separators=(',',':'))
        pyperclip.copy(s)
        self.forgetAll()
        s = 'ButtonHigh.TLabel'
        self.exportButton.config(style=s)

    def saveExportRelease(self):
        s = 'Button.TLabel'
        self.exportButton.config(style=s)

    def saveImportCallback(self):
        s = pyperclip.paste()
        if s == "":
            print("Nothing in copy/paste buffer")
            return
        if s[0] != "{":
            print("Bad format in copy buffer, expecting Json")
            return
        self.controller.selectorValue = s
        self.controller.selectorAction = "IMPORT"
        self.forgetAll()
        s = 'ButtonHigh.TLabel'
        self.importButton.config(style=s)

    def saveImportRelease(self):
        s = 'Button.TLabel'
        self.importButton.config(style=s)

    def saveOkCallback(self):
        name = self.paramsnameVar.get()

        if self.pagename == "quad":
            self.controller.saveQuad(name)
        else:
            self.saveJson(self.pagename,name)
            # self.saveJson(self.pagename,name)

        self.updateParamFiles()
        self.controller.updateSelectorPage(self.pagename,self.paramFiles)
        self.saveCancelCallback()

    def saveJson(self,section,fname,suffix=".json"):

        # Note: saving always happens in the localPresetsFilePath,
        # even if the original one was loaded from a different directory
        fpath = palette.localPresetsFilePath(section,fname,suffix)
        j = self.jsonParamDump()
        print("Edit page is saving JSON in:",fpath)
        SaveJsonInPath(j,fpath)

    def jsonParamDump(self):
        newjson = {}
        newjson["params"] = {}
        for name in self.params:
            newjson["params"][name] = {}
            w = self.paramValueWidget[name]
            newjson["params"][name] = self.normalizeJsonValue(name,w.cget("text"))
        return newjson

    # Return value of normalizeJsonValue is always a string
    def normalizeJsonValue(self,name,v):
        t = self.controller.paramValueTypeOf[name]
        if t == "bool":
            return "true" if palette.boolValueOfString(v) else "false"
        if t == "int":
            if v == "":
                v = "0"
            v = int(v)
            mn = int(self.params[name]["min"])
            mx = int(self.params[name]["max"])
            v = mn if v < mn else mx if v > mx else v
            return ("%6d" % (int(float(v)))).strip()
        if t == "double" or t == "float":
            if v == "":
                v = "0.0"
            v = float(v)
            mn = float(self.params[name]["min"])
            mx = float(self.params[name]["max"])
            v = mn if v < mn else mx if v > mx else v
            return ("%6.3f" % (float(v))).strip()
        if t == "string":
            return str(v).strip()

        return "Unrecognized Type"

class ScrollBar(tk.Frame):

    def __init__(self, parent, notify, tag=None):
        tk.Frame.__init__(self, parent)
        self.notify = notify
        self.tag = tag
        self.config(background=palette.ColorBg)

        self.scroll = tk.Canvas(self, background=palette.ColorScrollbar, highlightthickness=0)
        self.scroll.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.scroll.bind("<Button-1>", self.scrollClick)
        self.scroll.bind("<B1-Motion>", self.scrollMotion)
        # self.scroll.bind("<MouseWheel>", self.scrollWheel)

        self.thumb = tk.Canvas(self.scroll, background=palette.ColorThumb, highlightthickness=0)
        self.thumb.place(in_=self.scroll, relx=0, rely=0.0, relwidth=1, relheight=thumbFactor )
        self.thumb.bind("<Button-1>", self.thumbClick)
        self.thumb.bind("<B1-Motion>", self.thumbMotion)

        self.currentY = 0.0
        self.currentThumbY = 0.0

    def thumbClick(self,event):
        thumbHeight = self.thumb.winfo_height()
        # print("\nthumbClick event.y = ",event.y," thumbHeight=",thumbHeight)
        dy = event.y - (thumbHeight/2) 
        self.scrollMoveBy(dy)

    def thumbMotion(self,event):
        thumbHeight = self.thumb.winfo_height()
        # print("\nthumbMotion event.y = ",event.y," thumbHeight=",thumbHeight)
        dy = event.y - (thumbHeight/2) 
        self.scrollMoveBy(dy)

    def scrollClick(self,event):
        dy = event.y - self.currentY
        # print("\nscrollClick event.y=",event.y," dy=",dy)
        self.scrollMoveBy(dy)

    def scrollMotion(self,event):
        dy = event.y - self.currentY
        # print("\nscrollMotion event.y=",event.y," dy=",dy)
        self.scrollMoveBy(dy)

    def scrollWheel(self,event):
        scrollHeight = self.scroll.winfo_height()
        dy = int(scrollHeight * thumbFactor)
        dy = dy * 4
        if event.delta > 0:
            amount = -dy
        else:
            amount = dy
        # print("\nscrollWheel delta=",event.delta," dy=",dy," amount=",amount)
        self.scrollMoveBy(amount)

    def scrollMoveBy(self,dy):
        scrollHeight = self.scroll.winfo_height()

        # print("scrollMove dy=",dy,"  currentY=",self.currentY,"  scrollHeight=",scrollHeight)
        dy = dy / 16  # scale it down
        newy = self.currentY + dy
        if newy < 0.0:
            newy = 0.0
        elif newy > scrollHeight:
            newy = scrollHeight

        if newy == self.currentY:
            # print("scrollMove no change, do nothing")
            return

        self.currentY = newy

        fy = self.currentY / scrollHeight

        if fy < 0.0:
            fy = 0.0
        elif fy > 1.0:
            fy = 1.0

        thumbHalfHeight = thumbFactor / 2.0
        if fy < thumbHalfHeight:
            fthumby = thumbHalfHeight
        elif fy > (1.0-thumbHalfHeight):
            fthumby = 1.0 - thumbHalfHeight
        else:
            fthumby = fy

        fthumby -= thumbHalfHeight

        # print("currentY=",self.currentY," fy=",fy," fthumby=",fthumby)
        self.thumb.place(in_=self.scroll, relx=0, rely=fthumby, relwidth=1, relheight=thumbFactor )
        self.notify.scrollNotify(fy,self.tag)
        # print("END OF MOVEBY\n")

class PagePerformMain(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=palette.ColorBg)

        self.frame = tk.Frame(self, background=palette.ColorBg)
        self.frame.pack(side=tk.TOP, fill=tk.BOTH, expand=True, pady=5)

        self.performButton = {}
        self.buttonNames = []

        self.makePerformButton("loopingonoff")
        self.makePerformButton("loopinglength")
        self.makePerformButton("loopingfade")
        self.makePerformButton("Loop_Clear", self.controller.clearLoop)

        # More advanced buttons
        # self.makePerformButton("transpose")
        # self.makePerformButton("useexternalscale")
        # self.makePerformButton("Notes_Off", self.controller.sendANO)

        # self.makePerformButton("quant")
        # self.makePerformButton("vol")
        # self.makePerformButton("scale")
        # self.makePerformButton("midithru")
        # self.makePerformButton("midiquantized")
        # self.makePerformButton("TBD1_ ")
        self.makePerformButton("Reset_All", self.controller.resetAll)

        ### self.makePerformButton("Comb_Notes", self.controller.combLoop)
        ### self.makePerformButton("useexternalscale")
        ### self.makePerformButton("midiquantized")

        # self.makePerformButton("configname")

        self.advancedButtons = {
            "recording", "quant", "vol", "scale", "tempo", "Comb_Notes",
            "midithru", "midiquantized", "Notes_Off", "All Notes_Off",
            "useexternalscale",
            # "configname"
        }

    def updatePerformButtonLabels(self,pad):
        performButtonsPerRow = 7
        col = 0
        row = 0
        for name in self.buttonNames:
            button = self.performButton[name]

            if name in palette.PerformLabels:
                index = pad.performIndex[name]
                text = palette.PerformLabels[name][index]["label"]
            elif name in palette.GlobalPerformLabels:
                index = self.controller.globalPerformIndex[name]
                text = palette.GlobalPerformLabels[name][index]["label"]
            else:
                text = button.cget("text")

            if isTwoLine(text):
                text = text.replace(palette.LineSep,"\n",1)

            ipady = 0
            button.config(text=text)

            if name == "TBD" or (self.controller.guiLevel==0 and name in self.advancedButtons):
                button.grid_forget()
            else:
                button.grid(row=row,column=col, padx=performButtonPadx,pady=performButtonPady,ipady=ipady)
            col += 1
            if col >= performButtonsPerRow:
                col = 0
                row += 1

    def makePerformButton(self,name,f=None,text=None):
        if f == None:
            cmd = lambda nm=name: self.performCallback(nm)
        else:
            cmd = f
        self.performButton[name] = ttk.Button(self.frame, width=10, command=cmd)
        self.setPerformButtonText(name,text)
        self.buttonNames.append(name)

    def setPerformButtonText(self,name,text):
        if text == None:
            text = name
        if isTwoLine(text):
            text = text.replace(palette.LineSep,"\n",1)
        self.performButton[name].config(text=text, width=10, style='PerformButton.TLabel')

    def performCallback(self,name):

        controller = self.controller
        controller.resetLastAnything()

        # The PerformLabels are per-pad perform values
        if name in palette.PerformLabels:

            if controller.allPadsSelected:

                # We do the CurrPad and then force all of the
                # other pads to whatever the newindex is for that one
                newtext = self.padPerformCallback(controller.CurrPad,name)
                newindex = controller.CurrPad.getPerformIndex(name)
                for pad in controller.Pads:
                    pad.setPerformIndex(name,newindex)
                    pad.sendPerformVal(name)
            else:
                newtext = self.padPerformCallback(controller.CurrPad,name)
                controller.CurrPad.sendPerformVal(name)

            self.performButton[name].config(text=newtext)

        elif name in palette.GlobalPerformLabels:

            newtext = self.globalPerformCallback(name)
            self.performButton[name].config(text=newtext)
            controller.sendGlobalPerformVal(name)

        else:
            print("UNHANDLED performCallback name=",name)

    def padPerformCallback(self,pad,name):
        labels = palette.PerformLabels[name]
        index = pad.performIndex[name]
        newindex = ( index + 1 ) % len(labels)
        newtext = labels[newindex]["label"]
        if isTwoLine(newtext):
            newtext = newtext.replace(palette.LineSep,"\n",1)
        pad.performIndex[name] = newindex
        return newtext

    def globalPerformCallback(self,name):
            controller = self.controller
            labels = palette.GlobalPerformLabels[name]
            index = controller.globalPerformIndex[name]
            newindex = ( index + 1 ) % len(labels)
            newtext = labels[newindex]["label"]
            if isTwoLine(newtext):
                newtext = newtext.replace(palette.LineSep,"\n",1)

            controller.globalPerformIndex[name] = newindex

            return newtext

class PageSelector(tk.Frame):

    def __init__(self, parent,controller,vals,pagename):
        tk.Frame.__init__(self, parent)
        self.vals = vals
        self.controller = controller
        self.pagename = pagename

        self.config(background=palette.ColorBg)

        self.selectButtons = {}
        self.selectOffset = 0

        self.valsframe = tk.Frame(self, background=palette.ColorBg)
        self.valsframe.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, pady=10)

        self.scrollbar = ScrollBar(parent=self, notify=self)
        self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=11, padx=5)

        self.doLayout()

    def scrollNotify(self,sfy,tag):
        # print("scrollNotify sfy=",sfy," tag=",tag)
        nparams = len(self.vals)
        selectPerPage = selectDisplayRows * selectDisplayPerRow
        tmp = int(sfy * (nparams-selectPerPage))
        self.selectOffset = int(tmp / selectDisplayPerRow) * selectDisplayPerRow
        # silly code
        if self.selectOffset > (nparams-selectPerPage-selectDisplayPerRow):
            self.selectOffset = nparams - selectPerPage
        if self.selectOffset < 0:
            self.selectOffset = 0
        self.doLayout()

    def doLayout(self):
        valindex = self.selectOffset
        i = 0
        for r in range(0,selectDisplayRows):
            for c in range(0,selectDisplayPerRow):
                if valindex < len(self.vals):

                    # First time here, we create the Button
                    selectButtonText = self.vals[valindex]
                    ipadx = 0
                    istwo = isTwoLine(selectButtonText)
                    if istwo:
                        style='PatchTwoLine.TLabel'
                        ipady = 0
                        width=13
                        selectButtonText = selectButtonText.replace(palette.LineSep,"\n",1)
                        selectButtonText = selectButtonText.replace(palette.LineSep," ")
                    else:
                        style='PatchTwoLine.TLabel'
                        selectButtonText = selectButtonText + "\n"
                        ipady = 0
                        width=13

                    if not i in self.selectButtons:
                        self.selectButtons[i] = ttk.Button(self.valsframe, width=width, style=style)

                    self.selectButtons[i].grid(row=r,column=c,padx=selectButtonPadx,pady=selectButtonPady,ipady=ipady,ipadx=ipadx)
                    self.selectButtons[i].config(text=selectButtonText,
                        command=lambda val=self.vals[valindex],buttoni=i:self.selectorCallback(val,buttoni))
                    valindex += 1
                else:
                    if i in self.selectButtons:
                        self.selectButtons[i].grid_forget()
                i += 1

    def defaultVal(self):
        if len(self.vals) > 0:
            return self.vals[0]
        else:
            return "default"

    def selectorCallback(self,val,buttoni):
        self.controller.selectorValue = val
        self.controller.selectorAction = "LOAD"
        self.controller.selectorButtonIndex = buttoni
        for i in self.selectButtons:
            if i == buttoni:
                s = 'PatchTwoLineHighlight.TLabel'
            else:
                s = 'PatchTwoLine.TLabel'
            self.selectButtons[i].config(style=s)

def startgui():
    # print("STARTGUI called")
    global StartupMode
    StartupMode = False
    # global startupPhase
    # startupPhase = ""

def padOfParam(paramname):
    pad = paramname[0]
    # This code assumes that all real parameter names are lower-case
    if pad == "A" or pad == "B" or pad == "C" or pad == "D":
        baseparam = paramname[2:]
        return (pad,baseparam)
    else:
        return (None,paramname)

def PadParamName(pad,param):
    return pad + "-" + param

def isTwoLine(text):
    return text.find(palette.LineSep) >= 0 or text.find("\n") >= 0

def CurrentPadFilename(pad):
    return "CurrentPad_"+pad

def CurrentPadPath(pad):
    nm = CurrentPadFilename(pad)
    return palette.configFilePath(nm+".json")

def SaveJsonInPath(j,fpath):
    f = open(fpath,"w")
    if not f:
        print("SaveJsonInPath: unable to open path=",fpath)
        return
    f.write(json.dumps(j, sort_keys=True, indent=4, separators=(',',':')))
    # To avoid complaints from editors, add a final newline
    f.write("\n")
    f.close()

def initMain(app):
    app.iconbitmap(palette.configFilePath("palette.ico"))
    app.mainLoop()

if __name__ == "__main__":

    print("\n========= Palette GUI starts\n")
    pads = palette.ConfigValue("pads")
    if pads == "":
        npads = 1
        pads = "A" # default if no pads config value

    npads = len(pads)
    if npads == 1:
        # You can set pads to "B", for example
        padname = pads[0]
        padnames = pads
        visiblepagenames = {
            "snap":"Pad",
            "sound":"Sound",
            "visual":"Visual",
            "effect":"Effect",
        }
    elif npads == 4:
        padname = pads[0]
        padnames = pads
        IsQuad = True
        visiblepagenames = {
            "quad":"Preset",
            "snap":"Pad",
            "sound":"Sound",
            "visual":"Visual",
            "effect":"Effect",
        }
    else:
        print("Unexpected number of pads: ",pads)

    gui_size = palette.ConfigValue("gui_size")
    if gui_size == "":
        gui_size = "small"   # default

    if gui_size == "small":
        # print("small size")
        GuiWidth = 520 ; GuiHeight = 560
        fontFactor = 0.5
        thumbFactor = 0.1

        selectDisplayRows = 13
        paramDisplayRows = 23
        selectDisplayPerRow = 4

        pageSizeOfSelectNormal = 0.912
        pageSizeOfControlNormal = 1.0 - pageSizeOfSelectNormal

        pageSizeOfSelectAdvanced = 0.86
        pageSizeOfControlAdvanced = 1.0 - pageSizeOfSelectAdvanced

        performButtonPadx = 6
        performButtonPady = 0

        selectButtonPadx = 5
        selectButtonPady = 3

    elif gui_size == "max":
        # print("max size")
        GuiWidth = 800 ; GuiHeight = 1280
        fontFactor = 1.0
        thumbFactor = 0.1
        paramDisplayRows = 20
        selectDisplayRows = 13
        selectDisplayPerRow = 4

        # 0.85 total
        pageSizeOfControlNormal = 0.17
        pageSizeOfSelectNormal = 0.68
        # 0.85 total
        pageSizeOfControlAdvanced = 0.27
        pageSizeOfSelectAdvanced = 0.58

        performButtonPadx = 8
        performButtonPady = 5

        selectButtonPadx = 10
        selectButtonPady = 5

    else:
        print("INVALID VALUE OF gui_size in config: ",gui_size)
        GuiWidth = 400 ; GuiHeight = 600

    palette.setFontSizes(fontFactor)
    palette.PerformLabels["scale"] = palette.SimpleScales

    global app
    app = ProGuiApp(padname,padnames,visiblepagenames)

    palette.makeStyles(app)


    app.wm_geometry("%dx%d" % (GuiWidth,GuiHeight))

    delay = 0.0

    threading.Timer(delay, startgui).start()

    initMain(app)
