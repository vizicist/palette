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
from codenamize import codenamize

import asyncio
from nats.aio.client import Client as NATS
from nats.aio.errors import ErrConnectionClosed, ErrTimeout, ErrNoServers

import palette

IsQuad = False
RecMode = False

ColorWhite = '#ffffff'
ColorBlack = '#000000'

ColorBg = '#000000'
ColorText = '#ffffff'
ColorComboText = '#000000'    # black
ColorButton = '#333333'  
ColorScrollbar = '#333333'  
ColorThumb = '#00ffff'  

ColorRed = '#ff0000'
ColorBlue = '#0000ff'
ColorGreen = '#00ff00'
ColorHigh = '#006666'
ColorBright = '#00ffff'
ColorAqua = '#00ffff'
ColorUnHigh = '#888888'

signal.signal(signal.SIGINT,signal.SIG_IGN)

class Params():
    def __init__(self):
        pass

class ProGuiApp(tk.Tk):

    def __init__(self,
            padname,
            padnames,
            visiblepagenames,
            guisize
            ):

        self.guisize = guisize

        self.currentMode = ""
        self.nextMode = ""

        s = palette.ConfigValue("guilevel")
        if s == "":
            self.defaultGuiLevel = 0
        else:
            self.defaultGuiLevel = int(s)

        self.currentPageName = None

        self.setGuiLevel(self.defaultGuiLevel)

        self.thumbFactor = 0.1

        self.lastAnything = 0
        self.resetAfterInactivity = int(palette.ConfigValue("attracttimeout"))

        # These are the same in both normal and advanced
        self.selectDisplayPerRow = 3

        # Normal layout values
        if guisize == "large":
            self.paramDisplayRows = 19
            self.frameSizeOfControlNormal = 0.06
            self.frameSizeOfSelectNormal = 1.0 - self.frameSizeOfControlNormal
            self.frameSizeOfPadChooserNormal = 0.0
            self.selectDisplayRowsNormal = 15
            self.frameSizeOfControlAdvanced = 0.15
            self.frameSizeOfPadChooserAdvanced = 0.13
            self.frameSizeOfSelectAdvanced = 1.0 - self.frameSizeOfControlAdvanced - self.frameSizeOfPadChooserAdvanced
            self.performButtonPadx = 3
            self.performButtonPady = 2
            self.performButtonsPerRow = 6
            self.selectDisplayRowsAdvanced = 10
        else:
            self.paramDisplayRows = 18
            self.frameSizeOfControlNormal = 0.085
            self.frameSizeOfSelectNormal = 1.0 - self.frameSizeOfControlNormal
            self.frameSizeOfPadChooserNormal = 0.0
            self.selectDisplayRowsNormal = 14
            self.frameSizeOfControlAdvanced = 0.19
            self.frameSizeOfPadChooserAdvanced = 0.14
            self.frameSizeOfSelectAdvanced = 1.0 - self.frameSizeOfControlAdvanced - self.frameSizeOfPadChooserAdvanced
            self.performButtonPadx = 3
            self.performButtonPady = 3
            self.performButtonsPerRow = 6
            self.selectDisplayRowsAdvanced = 8

        self.frameSizeOfSelectAdvancedQuad = self.frameSizeOfSelectAdvanced + self.frameSizeOfPadChooserAdvanced
        if (self.frameSizeOfSelectAdvanced + self.frameSizeOfControlAdvanced + self.frameSizeOfPadChooserAdvanced) != 1.0:
            log("Hey, page sizes don't add up to 1.0")


        self.selectButtonPadx = 5
        self.selectButtonPady = 3

        # self.setFrameSizes()
        self.setScaleList()

        setFontSizes(self.guisize)

        tk.Tk.__init__(self)

        self.AllPageNames = {
                "quad":0,
                "snap":0,
                "sound":0,
                "visual":0,
                "effect":0,
                "misc":0}

        self.visiblePageNames = visiblepagenames
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

        self.allPadsSelected = True

        self.frames = {}
        self.editPage = {}
        self.selectorPage = {}
        
        self.selectorAction = ""
        self.selectorButtonIndex = 0
        self.selectorValue = ""
        self.activeCursors = {}
        self.activeTime = {}
        self.editMode = False
        self.showSound = False
        self.showPadFeedback = True
        self.showCursorFeedback = False

        # The values in globalPerformIndex are indexes into palette.PerformLabels
        # and the defaults point to the first entry in the label list
        self.globalPerformIndex = {}
        for s in palette.GlobalPerformLabels:
            self.globalPerformIndex[s] = 0

        self.windowName = "Palette Control"
        self.winfo_toplevel().title(self.windowName)

        self.escapeCount = 0
        self.lastEscape = time.time()
        self.lastClearLoop = time.time()
        self.resetLastAnything()
        self.layoutDone = False
    
    def setScaleList(self):
        if self.guiLevel == 0:
            palette.GlobalPerformLabels["scale"] = palette.SimpleScales
        else:
            palette.GlobalPerformLabels["scale"] = palette.PerformScales

    def placePadChooser(self):
        if self.guiLevel > 0 and self.currentPageName != "quad":
            self.padChooser.place(in_=self.topContainer, relx=0, rely=self.padChooserPageY, relwidth=1, relheight=self.frameSizeOfPadChooser)
        else:
            self.padChooser.place_forget()

    def forgetPadChooser(self):
        self.padChooser.place_forget()

    def scrollWheel(self,event):
        if self.editMode:
            scrollbar = self.editPage[self.currentPageName].scrollbar
        else:
            scrollbar = self.selectorPage[self.currentPageName].scrollbar
        scrollbar.scrollWheel(event)

    def mainLoop(self):
        # doneLoading = False
        while True:
            try:
                self.update_idletasks()
                self.update()
            except tk.TclError:
                s = traceback.format_exc()
                if s.find("application has been destroyed") >= 0:
                    log("Application has been closed!")
                else:
                    traceback.print_exc(file=sys.stdout)
                break
            except:
                traceback.print_exc(file=sys.stdout)
                break
    
            time.sleep(0.001)

            now = time.time()

            if self.nextMode != "":

                # switch to a new Mode 
                if self.nextMode == "layout":
                    self.initLayout()
                    if self.resetAfterInactivity>0:
                        self.nextMode = "attract"
                        self.startAttractMode()
                    else:
                        self.nextMode = "normal"
                        self.startNormalMode()

                elif self.nextMode == "help":
                    self.startHelpMode()

                elif self.nextMode == "attract":
                    self.startAttractMode()

                elif self.nextMode == "normal":
                    self.startNormalMode()

                else:
                    log("Invalid value for nextMode: ",self.nextMode)

                self.currentMode = self.nextMode
                self.nextMode = ""

            if self.currentMode == "":
                continue

            if self.currentMode == "normal":
                self.doSelectorAction()
                if self.resetAfterInactivity>0 and (now - self.lastAnything) > self.resetAfterInactivity:
                    log("No activity, starting attract mode...")
                    self.resetAll()
                    self.nextMode = "attract"

            elif self.currentMode == "attract":
                self.doAttractAction()

            elif self.currentMode == "help":
                self.doHelpAction()

            elif self.currentMode == "startup":
                self.doStartupAction()

            else:
                log("Invalid value for currentMode: ",self.currentMode)
                self.currentMode = ""

    def startNormalMode(self):
        # self.startupFrame.place_forget()
        self.attractFrame.place_forget()
        self.helpFrame.place_forget()
        self.resetVisibility()

    def startAttractMode(self):
        self.lastAttractAction = 0
        self.selectFrame.place_forget()
        self.performFrame.place_forget()
        self.padChooser.place_forget()
        self.helpFrame.place_forget()
        self.attractFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=1)
        # self.selectorLoadAndSend("quad","Square_Fantasia")
        # self.selectorLoadAndSend("quad","Circular_Garden")
        attractPreset = palette.ConfigValue("attractpreset")
        self.selectorLoadAndSend("quad",attractPreset)

    def startHelpMode(self):
        self.selectFrame.place_forget()
        self.performFrame.place_forget()
        self.padChooser.place_forget()
        self.attractFrame.place_forget()
        self.helpFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=1)

    def initLayout(self):

        if self.layoutDone == True:
            log("Hey! initLayout called twice!?")
            return
        self.layoutDone = True

        self.topContainer = tk.Frame(self, background=ColorBg)

        self.selectFrame = self.makeSelectFrame(self.topContainer)
        self.performFrame = self.makePerformFrame(self.topContainer)
        self.attractFrame = self.makeAttractFrame(self.topContainer)
        self.helpFrame = self.makeHelpFrame(self.topContainer)
        self.padChooser = self.makePadChooserFrame(parent=self.topContainer,controller=self)

        self.performPage = PagePerformMain(parent=self.performFrame, controller=self)

        self.topContainer.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.topContainer.bind_all("<MouseWheel>", self.scrollWheel)

        self.performPage.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

        # self.selectPage("snap")
        self.resetVisibility()

        # select the initial pad
        self.padChooserCallback(padname)
        self.allPadsSelected = True
        self.padChooser.refreshColors()

    def doStartupAction(self):
        pass

    def randomSprite(self,region,downup):
        x = random.random()
        y = random.random()
        z = 0.6 - random.random() / 2.0
        palette.SendSpriteEvent("0",x,y,z,region)

    def doAttractAction(self):
        now = time.time()
        dt = now - self.lastAttractAction
        if dt > 0.5:
            regions = ["A","B","C","D"]
            i = int(random.random()*99) % 4
            region = regions[i]
            self.randomSprite(region,"down")
            self.randomSprite(region,"up")
            self.lastAttractAction = now

    def doHelpAction(self):
        pass

    def doSelectorAction(self):
        reset = True
        if self.selectorAction == "LOAD":
            self.selectorLoadAndSend(self.currentPageName,self.selectorValue)

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

    def resetVisibility(self):
        for pg in self.visiblePageNames:
            if self.guiLevel > 0:
                self.pageHeader.pageButton[pg].pack(side=tk.LEFT,padx=5)
            else:
                self.pageHeader.pageButton[pg].pack_forget()

        self.editMode = False
        if IsQuad and self.guiLevel == 0:
            self.selectPage("quad")
        else:
            self.selectPage("snap")

        self.setFrameSizes()
        self.placeFrames()

        self.pageHeader.repack()

    def placeFrames(self):
        if self.guiLevel == 0:
            self.performFrame.place(in_=self.topContainer, relx=0, rely=self.performPageY, relwidth=1, relheight=self.frameSizeOfControl)
            self.selectFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=self.frameSizeOfSelect)
            self.placePadChooser()
        else:
            self.performFrame.place(in_=self.topContainer, relx=0, rely=self.performPageY, relwidth=1, relheight=self.frameSizeOfControl)
            self.selectFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=self.frameSizeOfSelect)
            self.placePadChooser()

    def setFrameSizes(self):

        if self.guiLevel == 0:
            self.frameSizeOfControl = self.frameSizeOfControlNormal
            self.frameSizeOfSelect = self.frameSizeOfSelectNormal
            self.frameSizeOfPadChooser = self.frameSizeOfPadChooserNormal
            self.selectDisplayRows = self.selectDisplayRowsNormal

        elif self.currentPageName == "quad":

            self.frameSizeOfControl = self.frameSizeOfControlAdvanced
            self.frameSizeOfSelect = self.frameSizeOfSelectAdvancedQuad
            self.frameSizeOfPadChooser = 0.0
            # Note - on Quad page, the PadChooser is not shown, and
            # selectDisplayRowsNormal is used
            self.selectDisplayRows = self.selectDisplayRowsNormal

        else:
            # Advanced is any guiLevel>0
            self.frameSizeOfControl = self.frameSizeOfControlAdvanced
            self.frameSizeOfSelect = self.frameSizeOfSelectAdvanced
            self.frameSizeOfPadChooser = self.frameSizeOfPadChooserAdvanced
            self.selectDisplayRows = self.selectDisplayRowsAdvanced

        y = 0
        self.selectPageY = y
        y += self.frameSizeOfSelect
        self.padChooserPageY = y
        y += self.frameSizeOfPadChooser
        self.performPageY = y
        y += self.frameSizeOfControl
 
    def saveQuad(self,name):

        quadPath = palette.localPresetsFilePath("quad",name)
        quadValues = {}

        for pad in self.Pads:
            padvalues = pad.getValues()
            for paramName in padvalues:
                fullName = PadParamName(pad.name(),paramName)
                quadValues[fullName] = padvalues[paramName]

        j = { "params": quadValues }
        log("Saving quad in",quadPath)
        SaveJsonInPath(j,quadPath)

    def loadQuad(self,presetName):
        fpath = palette.searchPresetsFilePath("quad", presetName)
        debug("readQuadPreset: fpath=",fpath)
        try:
            f = open(fpath)
        except:
            log("No such file?  CC fpath=",fpath)
            return
        j = json.load(f)
        self.loadQuadJson(j)
        f.close()

    def loadQuadJson(self,quadJson):

        if self.editMode:
            log("loadQuadJson shouldn't be called in edit mode")
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

    def makePadChooserFrame(self,parent,controller):
        f = PadChooser(parent,controller)
        f.config(background=ColorBg)
        return f

    def makePerformFrame(self,parent):
        return tk.Frame(parent,
            highlightbackground=ColorAqua, highlightcolor=ColorAqua, highlightthickness=3)

    def makeSelectFrame(self,container):

        f = tk.Frame(container,
            highlightbackground=ColorAqua, highlightcolor=ColorAqua, highlightthickness=0)

        self.pageHeader = PageHeader(parent=f, controller=self)
        self.pageHeader.pack(side=tk.TOP,fill=tk.BOTH)

        # These are the pages of buttons for selecting set/patch/sound/visual/etc..
        # Each one has a SelectorPage with the preset buttons,
        # and an EditPage with all the parameters of the preset
        for pagename in self.visiblePageNames:
            self.makeSelectorPage(f, pagename, PageSelector)
            self.makeEditPage(f,pagename)

        return f

    # def makeStartupFrame(self,container):
    #     f = tk.Frame(container,
    #         highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3)
    #     lbl = ttk.Label(f, text="  Space Palette Pro is Loading...", style='Loading.TLabel',
    #         foreground=ColorText, background=ColorBg, relief="flat", justify=tk.CENTER, font=hugeFont)
    #     lbl.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
    #     return f

    def unattract(self):
        self.nextMode = "normal"
        self.resetLastAnything()

    def unhelp(self):
        self.nextMode = "normal"
        self.resetLastAnything()

    def makeAttractFrame(self,container):

        path = palette.configFilePath("attractscreen.png")
        self.attractImage = tk.PhotoImage(file=path)

        f = tk.Frame(container,
            highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3)
        button = ttk.Button(f, image=self.attractImage, style='Attract.TLabel',
            command=self.unattract)
        button.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        return f

    def makeHelpFrame(self,container):

        path = palette.configFilePath("helpscreen.png")
        self.helpImage = tk.PhotoImage(file=path)

        f = tk.Frame(container,
            highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3)
        button = ttk.Button(f, image=self.helpImage, style='Attract.TLabel',
            command=self.unhelp)
        button.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
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
        page.config(highlightbackground=ColorAqua, highlightcolor=ColorAqua, highlightthickness=3)

    def makeEditPage(self,parent,pagename):
        page = PageEditParams(parent=parent, controller=self,
                    pagename=pagename, params=self.paramsOfType[pagename])
        self.editPage[pagename] = page
        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

    def forgetPages(self,pages):
        for pg in pages:
            pages[pg].pack_forget()

    def clickPage(self,pagename):

        # A second click on a page header will toggle editMode if GuiLevel>1
        if self.guiLevel > 1 and self.currentPageName == pagename:
            self.editMode = not self.editMode

        self.selectPage(pagename)

        if pagename != "quad" and self.editMode:
            if self.allPadsSelected:
                pad = self.padNamed("A")
            else:
                pad = self.CurrPad

            page = self.editPage[pagename]
            page.setValues(pad.getValues())

        self.setFrameSizes()
        self.placeFrames()

    def padNamed(self,padName):
        lastResort = None
        for pad in self.Pads:
            lastResort = pad
            if pad.name() == padName:
                return pad
        log("There is no Pad named",padName,", last resort, using",lastResort.name())
        return lastResort

    def selectPage(self,pagename):

        self.currentPageName = pagename
        self.pageHeader.highlightPageButton(pagename)

        self.forgetPages(self.selectorPage)
        self.forgetPages(self.editPage)

        self.placePadChooser()

        if self.guiLevel > 1 and self.editMode:
            page = self.editPage[pagename]
        else:
            page = self.selectorPage[pagename]

        self.setFrameSizes()

        if not self.editMode:
            page.doLayout()

        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        page.tkraise()

    def sendParamValues(self,values):
        log("sendParamValues: "+str(values))
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

    def selectorApply(self,apply,paramType):

        log("selectorApply 1")
        if not self.editMode:
            log("Hmmm, selectorAply should only be called in editMode")
            return

        if paramType == "quad":
            log("selectorApply not implemented on quad")
        else:

            self.applyToAllParams(apply,paramType)
            self.refreshPage()

            self.saveCurrent()

            if self.allPadsSelected:
                log("Sending",paramType,"params to all pads")
                for pad in self.Pads:
                    pad.sendParamsOfType(paramType)
            else:
                log("Sending",paramType,"params to pad",self.CurrPad.name())
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
                log("Unexpected pad in param value?")
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
                            log("Hey, randmax=",max," isn't a valid enum value for name=",name)
                    else:
                        i = random.randint(0,len(enums)-1)
                        v = enums[i]

            if v != "":
                self.changeAndSendValue(basename,v)

        self.saveCurrent()

    def saveCurrent(self):
        if self.allPadsSelected:
            for pad in self.Pads:
                pad.saveCurrent()
        else:
            self.CurrPad.saveCurrent()

    def selectorLoadAndSend(self,paramType,presetname):

        if self.editMode:
            log("HEY!! selectorLoadAndSend shouldn't be used in editMode?")
            return

        if paramType == "quad":
            log("Loading",paramType,presetname)
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
            log("Mismatched paramstype in JSON!")
            return

        if paramType == "snap":
            self.loadPageJson(self.editPage["snap"],j)
            self.sendPage(self.editPage["snap"])
        elif paramType == "quad":
            log("Hey, does quad need work here?  FFF")
        else:
            self.readOtherParamsJsonIntoSnapAndQuad(paramType,j)
            log("Sending",paramType,"params to pad",self.CurrPad.name())
            self.CurrPad.sendParamsOfType(paramType)

    def padChooserCallback(self,padName):
        self.CurrPad = self.padNamed(padName)

        self.allPadsSelected = False

        if len(self.PadNames) > 1:
            self.padChooser.refreshPadColors()

        self.refreshPage()

        self.performPage.updatePerformButtonLabels(self.CurrPad)
        self.CurrPad.sendPerformVals()

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

    def sendGlobalPerformVal(self,name):

        index = self.globalPerformIndex[name]
        val = palette.GlobalPerformLabels[name][index]["value"]

        if name == "tempo":
            palette.palette_global_api("set_tempo_factor", "\"value\": \""+str(val) + "\"")
        elif name == "transpose":
            palette.palette_global_api("set_transpose", "\"value\": \""+str(val) + "\"")
        elif name == "scale":
            palette.palette_global_api("set_scale", "\"value\": \""+str(val) + "\"")
        elif name == "autotranspose":
            palette.palette_global_api("set_autotranspose", "\"onoff\": \""+str(val) + "\"")

    def combPadLoop(self,pad):
        palette.palette_region_api(self.CurrPad.name(), "loop_comb", "")

    def combLoop(self):
        self.resetLastAnything()
        self.combPadLoop(self.CurrPad.name())

    def clear(self):
        if self.allPadsSelected:
            for pad in self.Pads:
                pad.clearLoop()
            palette.palette_global_api("audio_reset")
        else:
            self.CurrPad.clearLoop()
        self.checkEscape()
 
    def checkEscape(self):

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

    def cycleGuiLevel(self):
        # cycle through 0,1,2
        self.setGuiLevel((self.guiLevel + 1) % 3)
        self.resetVisibility()
        self.performPage.updatePerformButtonLabels(self.CurrPad)
        log("GuiLevel",self.guiLevel)

    def setGuiLevel(self,level):
        self.guiLevel = level
        self.setScaleList()

    def sendANO(self):
        for pad in self.Pads:
            pad.sendANO()

    def startHelp(self):
        self.nextMode = "help"

    def resetAll(self):

        log("ResetAll")

        palette.palette_global_api("audio_reset")

        self.setGuiLevel(self.defaultGuiLevel)
        self.resetLastAnything()

        self.allPadsSelected = True
        self.CurrPad = self.padNamed("A")
        self.padChooser.refreshPadColors()
        self.sendANO()

        for pad in self.Pads:
            pad.useExternalScale(False)
            pad.clearExternalScale()

        for s in palette.GlobalPerformLabels:
            self.globalPerformIndex[s] = 0

        for name in palette.GlobalPerformLabels:
            self.sendGlobalPerformVal(name)

        for pad in self.Pads:
            pad.clearLoop()
            pad.setDefaultPerform()
            pad.sendPerformVals()

        self.performPage.updatePerformButtonLabels(self.CurrPad)

        self.resetVisibility()

    def sendQuad(self):
        for pad in self.Pads:
            debug("Sending all parameters for pad = ",pad.name())
            for pt in ["sound","visual","effect"]:
                paramlistjson = pad.paramListOfType(pt)
                palette.palette_region_api(pad.name(), pt+".set_params", paramlistjson)

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
        log("Reading",path)
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
            self.paramValueTypeOf[name] = self.allParamsJson[name]["valuetype"]
            self.paramsOfType["snap"][name] = self.allParamsJson[name]

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
                log("WARNING! pt=",pt," is already in paramenums.json!")
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
                log("Unable to handle param name: ",name)
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

        for name in palette.GlobalPerformLabels:
            self.performIndex[name] = 0

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
            debug("No Current settings for pad=",self.padName," path=",path)
        else:
            log("Loaded",self.padName,"from",path)

    def loadPresetValues(self,paramType,presetName):
        fpath = palette.searchPresetsFilePath(paramType, presetName)
        if not self.loadFile(fpath):
            log("Unable to load preset file: ",fpath)
        else:
            log("Loaded",self.padName,"from",fpath)

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
        debug("Saving",self.padName,"in",self.snapPath)
        SaveJsonInPath(j,self.snapPath)

    def loadJson(self,j):
        # the self.params has all (i.e. snap) parameters, but
        # we only want to load whatever's in the json we're given.
        for paramName in j["params"]:
            self.setValue(paramName,j["params"][paramName])

    def setValue(self,paramName,val):
        if not paramName in self.paramValues:
            log("Hey, Pad.setValue paramName=",paramName,"not in paramValues?")
            return
        self.paramValues[paramName] = val
 
    def getValue(self,paramName):
        return self.paramValues[paramName]
 
    def sendValue(self,paramName):
        if not paramName in self.paramValues:
            log("Hey, Pad.sendValue paramName=",paramName,"not in paramValues?")
            return
        val = self.paramValues[paramName]
        if not paramName in self.controller.paramTypeOf:
            log("Unrecognized parameter: ",paramName)
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
            self.sendPerformVals()

    def sendPerformVals(self):
        for name in palette.PerformLabels:
            self.sendPerformVal(name)

    def getPerformIndex(self,name):
        return self.performIndex[name]

    def setPerformIndex(self,name,index):
        self.performIndex[name] = index

    def sendANO(self):
        palette.palette_region_api(self.name(), "ANO")

    def clearExternalScale(self):
        palette.palette_region_api(self.name(), "clearexternalscale")

    def useExternalScale(self,onoff):
        palette.palette_region_api(self.name(), "midi_usescale", "\"onoff\": \"" + str(onoff) + "\"")

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
                log("Unrecognized value of loopingonoff - %s\n" % val)
                return

            palette.palette_region_api(self.name(), "loop_recording", '"onoff": "'+str(reconoff)+'"')
            palette.palette_region_api(self.name(), "loop_playing", '"onoff": "'+str(playonoff)+'"')

        elif name == "loopinglength":
            palette.palette_region_api(self.name(), "loop_length", '"length": "'+str(val)+'"')

        elif name == "loopingfade":
            palette.palette_region_api(self.name(), "loop_fade", '"fade": "'+str(val)+'"')

        elif name == "loopingset":
            palette.palette_region_api(self.name(), "loop_set", '"set": "'+str(val)+'"')

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
            palette.palette_region_api(self.name(), "midi_thru", "\"onoff\": \"" + str(val) + "\"")

        elif name == "midisetscale":
            palette.palette_region_api(self.name(), "midi_setscale", "\"onoff\": \"" + str(val) + "\"")

        elif name == "midiusescale":
            self.useExternalScale(val)

        elif name == "midiquantized":
            palette.palette_region_api(self.name(), "midi_quantized", "\"onoff\": \"" + str(val) + "\"")

        elif name == "midithruscadjust":
            palette.palette_region_api(self.name(), "midi_thruscadjust", "\"onoff\": \"" + str(val) + "\"")

        elif name == "transpose":
            log("HEY, transpose shouldn't be here")
            # palette.palette_global_api(self.name(), "set_transpose", "\"value\": \""+str(val) + "\"")

        else:
            log("SendPerformVal: unhandled name=",name)

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

        self.makePadFrame(self,"A",0.05,0.07)
        self.makePadFrame(self,"B",0.15,0.53)
        self.makePadFrame(self,"C",0.55,0.53)
        self.makePadFrame(self,"D",0.65,0.07)

        self.makeGlobalButton(self,0.5,0.15)

        self.config(background=ColorBg)

    def makePadFrame(self,parent,pad,x0,y0):

        self.padFrame[pad] = tk.Frame(self)
        self.padFrame[pad].place(relx=x0,rely=y0,relwidth=0.3,relheight=0.4)
        self.padFrame[pad].config(borderwidth=2,relief="solid",background=ColorUnHigh)
        self.padFrame[pad].bind("<Button-1>", lambda p=pad: self.padCallback(p))

        if self.controller.showCursorFeedback:
            self.padCanvas[pad] = tk.Canvas(self.padFrame[pad], width=self.canvasWidth, height=self.canvasHeight, border=0)
            self.padCanvas[pad].pack(side=tk.TOP)
            self.padCanvas[pad].config(background=ColorUnHigh)

    def makeGlobalButton(self,parent,x0,y0):

        self.padGlobalButton = tk.Frame(self)
        self.padGlobalButton.place(relx=x0-0.05,rely=y0-0.05,relwidth=0.1,relheight=0.275)
        self.padGlobalButton.config(borderwidth=2,relief="solid",background=ColorUnHigh)
        self.padGlobalButton.bind("<Button-1>", self.globalCallback)

        self.padGlobalLabel = ttk.Label(self.padGlobalButton, text="*")
        self.padGlobalLabel.pack(side=tk.TOP)
        self.padGlobalLabel.configure(style='GlobalButton.TLabel')
        self.padGlobalLabel.bind("<Button-1>", self.globalCallback)

    def globalCallback(self,e):

        # If you hit * 4 times quickly it
        # will cycle through the advanced modes
        now = time.time()
        dt = now - self.controller.lastEscape
        if dt < 0.75:
            self.controller.escapeCount += 1
        else:
            self.controller.escapeCount = 0
        self.controller.lastEscape = now

        if self.controller.escapeCount == 3:
            self.controller.cycleAdvancedLevel()
            return

        if self.controller.guiLevel==0:
            return

        self.controller.allPadsSelected = not self.controller.allPadsSelected
        self.refreshColors()

    def refreshColors(self):
        if self.controller.allPadsSelected:
            color = ColorHigh
        else:
            color = ColorUnHigh
        self.padGlobalButton.config(background=color)
        self.padGlobalLabel.config(background=color)
        for pad in self.controller.Pads:
            if self.controller.allPadsSelected or pad != self.controller.CurrPad:
                self.colorPad(pad.name(),color)
            else:
                self.colorPad(pad.name(),ColorHigh)

    def colorPad(self,padName,color):
        self.padFrame[padName].config(background=color)
        if self.controller.showCursorFeedback:
            self.padCanvas[padName].config(background=color)

    def highlightPadBorder(self,pad,highlighted):
        if highlighted:
            w = 4
        else:
            w = 2
        self.padFrame[pad].config(borderwidth=w)

    def drawOval(self,pad,highlighted,x,y,z):
        log("drawOval x=",x," y=",y," z=",z)
        x = x * self.canvasWidth
        y = y * self.canvasHeight
        z = z * self.canvasWidth
        log("================= adjusted x=",x," y=",y," z=",z)
        if z < 10:
            z = 10
        elif z > (self.canvasWidth/4):
            z = self.canvasWidth/4
        if highlighted:
            color = ColorRed
        else:
            color = self.controller.padColor(pad)
        self.padCanvas[pad].create_oval(x-z,y-z,x+z,y+z,outline=color)

    def padCallback(self,e):
        if self.controller.guiLevel==0:
            return
        for pad in self.padFrame:
            if e.widget == self.padFrame[pad]:
                self.controller.allPadsSelected = False
                self.controller.padChooserCallback(pad)
                self.refreshColors()
                return
        log("No pad found in padCallback!?")

    def refreshPadColors(self):
        if self.controller.allPadsSelected:
            color = ColorHigh
        else:
            color = ColorUnHigh
        self.padGlobalButton.config(background=color)
        self.padGlobalLabel.config(background=color)

        for pad in self.controller.Pads:
            if self.controller.allPadsSelected or pad != self.controller.CurrPad:
                self.colorPad(pad.name(),color)
            else:
                self.colorPad(pad.name(),ColorHigh)

class PageHeader(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)
        self.controller = controller

        self.titleFrame = tk.Frame(self, background=ColorBg)
        self.titleFrame.pack(side=tk.TOP, fill=tk.X, expand=True)

        self.PaletteTitle = ttk.Label(self.titleFrame, style='PageButtonDisabled.TLabel',background=ColorBg)

        self.pageButton = {}
        for pageName in self.controller.visiblePageNames:
            realText = self.controller.visiblePageNames[pageName]
            self.pageButton[pageName] = ttk.Button(self.titleFrame, text=realText, style='PageButtonDisabled.TLabel',
                command=lambda nm=pageName: self.controller.clickPage(nm))

        self.repack()

    def repack(self):
        if self.controller.guiLevel == 0:
            self.PaletteTitle.config(text="Space Palette Pro",justify=tk.CENTER)
            self.PaletteTitle.pack(side=tk.TOP,pady=10)
            for pageName in self.controller.visiblePageNames:
                self.pageButton[pageName].pack_forget()
        else:
            self.PaletteTitle.config(text="  Presets:  ",justify=tk.LEFT)
            self.PaletteTitle.pack(side=tk.LEFT,pady=10)
            for pageName in self.controller.visiblePageNames:
                self.pageButton[pageName].pack(side=tk.LEFT,padx=5)
            
    def highlightPageButton(self,pagename):
        for nm in self.pageButton:
            if nm == pagename:
                self.pageButton[nm].config(style='PageButtonEnabled.TLabel')
            else:
                self.pageButton[nm].config(style='PageButtonDisabled.TLabel')

class PageEditParams(tk.Frame):

    def __init__(self, parent, controller, pagename, params):
        tk.Frame.__init__(self, parent)
        self.controller = controller

        self.config(background=ColorBg,
            highlightbackground=ColorAqua, highlightcolor=ColorAqua, highlightthickness=3)

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

        self.controller = container.controller

        f = tk.Frame(container, background=ColorBg)
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

            # log("MakeParamsArea pagename=",self.pagename," Param=",name)
            self.paramRowName.append(name)
            self.paramLabelWidget[name] = ttk.Label(f, width=20, text=name, style='ParamName.TLabel')
            self.paramLabelWidget[name].config()

            self.paramValueWidget[name] = ttk.Label(f, width=10, anchor=tk.E, style='ParamValue.TLabel')
            self.paramValueWidget[name].bind("<Button-1>", lambda event,nm=name: self.valueClicked(nm))

        # The widgets for << < . . > >> are static, in the displayed rows
        for row in range(0,self.controller.paramDisplayRows):
            f2 = tk.Frame(f, background=ColorBg)
            self.adjustButton(f2,row,"<<", -3)
            self.adjustButton(f2,row,"<", -2)
            self.adjustButton(f2,row,"-", -1)
            self.adjustButton(f2,row,"+", 1)
            self.adjustButton(f2,row,">", 2)
            self.adjustButton(f2,row,">>", 3)
            self.paramAdjustFrame[row] = f2

        return f

    def makeButtonArea(self):
        f = tk.Frame(self, background=ColorBg)

        if self.pagename != "quad":
            self.initButton = ttk.Label(f, text="Init", style='RandEtcButton.TLabel')
            self.initButton.bind("<Button-1>", lambda event:self.initCallback())
            self.initButton.bind("<ButtonRelease-1>", lambda event:self.initRelease())
            self.initButton.pack(side=tk.LEFT, padx=2)

            self.randButton = ttk.Label(f, text="Rand", style='RandEtcButton.TLabel')
            self.randButton.bind("<Button-1>", lambda event:self.randCallback())
            self.randButton.bind("<ButtonRelease-1>", lambda event:self.randRelease())
            self.randButton.pack(side=tk.LEFT, padx=2)

        # import/export needs to be resurrected
        showImport = False
        if showImport:
            self.importButton = ttk.Label(f, text="Imp", style='RandEtcButton.TLabel')
            self.importButton.bind("<Button-1>", lambda event:self.saveImportCallback())
            self.importButton.bind("<ButtonRelease-1>", lambda event:self.saveImportRelease())
            self.importButton.pack(side=tk.LEFT, padx=2)

            self.exportButton = ttk.Label(f, text="Exp", style='RandEtcButton.TLabel')
            self.exportButton.bind("<Button-1>", lambda event:self.saveExportCallback())
            self.exportButton.bind("<ButtonRelease-1>", lambda event:self.saveExportRelease())
            self.exportButton.pack(side=tk.LEFT, padx=2)

        b = ttk.Label(f, text="Save", style='RandEtcButton.TLabel')
        b.bind("<Button-1>", lambda event:self.saveCallback())
        b.pack(side=tk.LEFT, pady=5, padx=2)

        # The following things don't get placed initially,
        # they're revealed when the Save button is pressed.

        self.comboParamsname = ttk.Combobox(f, textvariable=self.paramsnameVar,
                font=comboFont, style='custom.TCombobox')
        self.comboParamsname.bind("<<ComboboxSelected>>", lambda event,v=self.paramsnameVar : self.checkThenGotoParamsFile(v.get()))
        self.comboParamsname.bind("<Return>", lambda event,v=self.paramsnameVar : self.checkThenGotoParamsFile(v.get()))

        self.okButton = ttk.Label(f, text="OK", style='RandEtcButton.TLabel')
        self.okButton.bind("<Button-1>", lambda event:self.saveOkCallback())

        self.cancelButton = ttk.Label(f, text="Cancel", style='RandEtcButton.TLabel')
        self.cancelButton.bind("<Button-1>", lambda event:self.saveCancelCallback())

        return f

    def scrollNotify(self,sfy,tag):
        nparams = len(self.params)
        self.valuesDisplayOffset = int((nparams-self.controller.paramDisplayRows) * sfy)
        # log("valuesDisplayOffset=",self.valuesDisplayOffset)
        self.updateParamView()

    def updateParamView(self):

        for r in range(0,self.controller.paramDisplayRows):
            self.paramAdjustFrame[r].grid_forget()

        px = 0
        row = 0
        # log("updateParamView valuesDisplayOffset=",self.valuesDisplayOffset)
        for name in self.params:
            showrow = row - self.valuesDisplayOffset
            showme = (showrow >= 0 and showrow < self.controller.paramDisplayRows)
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
        log("valueClicked! name=",name)

    def adjustValue(self,row,amount):
        # log("adjustValue valuesDisplayOffset=",self.valuesDisplayOffset)
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
            # log("amount=",amount," mx=",mx," v=",v)
            newval = v
        elif t == "string":
            v = str(widg.cget("text"))
            vals = self.controller.paramenums[self.params[name]["min"]]
            try:
                i = vals.index(v.strip())
            except:
                log("Unable to find v=",v)
                i = 0
            # log("string v=",v," t=",t," vals=",vals," existing i=",i)
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

        self.controller.changeAndSendValue(name,newval)
        self.controller.saveCurrent()

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
        # log("Edit page, changing ALL parameters in setValues")
        for name in self.params:
            if name in values:
                self.changeValueText(name,values[name])

    def changeValueText(self,name,v):
        # log("CHANGE VALUE LABEL EDIT PAGE=",self.pagename," name=",name," v=",v)
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
                log("Error when trying convert v=",v)
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
            log("Unable to load preset: ",path)
            return

        j = json.load(f)
        presetvals = j["params"]

        self.controller.sendParamValues(presetvals)

    def loadSnapNamed(self,name,doLift=True):

        log("\n=== loadSnapNamed ",name)

        self.controller.readSnapParamsFileIntoPage(name,"snap")

        self.comboParamsname.configure(values=self.paramFiles)

        self.setPresetNameInComboBox(name)

        for p in self.params:
            self.changeValue(p,self.getValue(p))

        if doLift:
            self.lift()

    def oldstartEditing(self,name,doLift=True):

        log("=== startEditing pagename=%s name=%s" % (self.pagename,name))
        if self.pagename == "quad":
            log("Are you getting here?")
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
        # s = pyperclip.paste()
        # self.controller.selectorValue = s
        self.controller.selectorAction = "RAND"
        self.forgetAll()
        self.randButton.config(style='RandEtcButtonHigh.TLabel')

    def randRelease(self):
        self.randButton.config(style='RandEtcButton.TLabel')

    def initCallback(self):
        # s = pyperclip.paste()
        # self.controller.selectorValue = s
        self.controller.selectorAction = "INIT"
        self.forgetAll()
        self.initButton.config(style='RandEtcButtonHigh.TLabel')

    def initRelease(self):
        self.initButton.config(style='RandEtcButton.TLabel')

    def saveExportCallback(self):
        j = self.jsonParamDump()
        j["paramsname"] = self.paramsnameVar.get()
        j["paramstype"] = self.pagename 
        s = json.dumps(j, sort_keys=True, indent=4, separators=(',',':'))
        pyperclip.copy(s)
        self.forgetAll()
        self.exportButton.config(style='RandEtcButtonHigh.TLabel')

    def saveExportRelease(self):
        self.exportButton.config(style='RandEtcButton.TLabel')

    def saveImportCallback(self):
        s = pyperclip.paste()
        if s == "":
            log("Nothing in copy/paste buffer")
            return
        if s[0] != "{":
            log("Bad format in copy buffer, expecting Json")
            return
        self.controller.selectorValue = s
        self.controller.selectorAction = "IMPORT"
        self.forgetAll()
        self.importButton.config(style='RandEtcButtonHigh.TLabel')

    def saveImportRelease(self):
        self.importButton.config(style='RandEtcButton.TLabel')

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
        log("Edit page is saving JSON in:",fpath)
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
        self.controller = parent.controller
        self.notify = notify
        self.tag = tag
        self.config(background=ColorBg)

        self.scroll = tk.Canvas(self, background=ColorScrollbar, highlightthickness=0)
        self.scroll.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        # try - self.scroll.pack(side=tk.TOP, width=200, height=400)
        # self.scroll.place(in_=self, width=200, height=400)
        self.scroll.bind("<Button-1>", self.scrollClick)
        self.scroll.bind("<B1-Motion>", self.scrollMotion)
        # self.scroll.bind("<MouseWheel>", self.scrollWheel)

        self.thumb = tk.Canvas(self.scroll, background=ColorThumb, highlightthickness=0)
        self.thumb.place(in_=self.scroll, relx=0, rely=0.0, relwidth=1, relheight=self.controller.thumbFactor )
        self.thumb.bind("<Button-1>", self.thumbClick)
        self.thumb.bind("<B1-Motion>", self.thumbMotion)

        self.currentY = 0.0
        self.currentThumbY = 0.0

    def thumbClick(self,event):
        thumbHeight = self.thumb.winfo_height()
        # log("\nthumbClick event.y = ",event.y," thumbHeight=",thumbHeight)
        dy = event.y - (thumbHeight/2) 
        self.scrollMoveBy(dy)

    def thumbMotion(self,event):
        thumbHeight = self.thumb.winfo_height()
        # log("\nthumbMotion event.y = ",event.y," thumbHeight=",thumbHeight)
        dy = event.y - (thumbHeight/2) 
        self.scrollMoveBy(dy)

    def scrollClick(self,event):
        dy = event.y - self.currentY
        # log("\nscrollClick event.y=",event.y," dy=",dy)
        self.scrollMoveBy(dy)

    def scrollMotion(self,event):
        dy = event.y - self.currentY
        # log("\nscrollMotion event.y=",event.y," dy=",dy)
        self.scrollMoveBy(dy)

    def scrollWheel(self,event):
        scrollHeight = self.scroll.winfo_height()
        dy = int(scrollHeight * self.controller.thumbFactor)
        dy = dy * 4
        if event.delta > 0:
            amount = -dy
        else:
            amount = dy
        # log("\nscrollWheel delta=",event.delta," dy=",dy," amount=",amount)
        self.scrollMoveBy(amount)

    def scrollMoveBy(self,dy):
        scrollHeight = self.scroll.winfo_height()

        # log("scrollMove dy=",dy,"  currentY=",self.currentY,"  scrollHeight=",scrollHeight)
        dy = dy / 16  # scale it down
        newy = self.currentY + dy
        if newy < 0.0:
            newy = 0.0
        elif newy > scrollHeight:
            newy = scrollHeight

        if newy == self.currentY:
            # log("scrollMove no change, do nothing")
            return

        self.currentY = newy

        fy = self.currentY / scrollHeight

        if fy < 0.0:
            fy = 0.0
        elif fy > 1.0:
            fy = 1.0

        thumbHalfHeight = self.controller.thumbFactor / 2.0
        if fy < thumbHalfHeight:
            fthumby = thumbHalfHeight
        elif fy > (1.0-thumbHalfHeight):
            fthumby = 1.0 - thumbHalfHeight
        else:
            fthumby = fy

        fthumby -= thumbHalfHeight

        # log("currentY=",self.currentY," fy=",fy," fthumby=",fthumby)
        self.thumb.place(in_=self.scroll, relx=0, rely=fthumby, relwidth=1, relheight=self.controller.thumbFactor )
        self.notify.scrollNotify(fy,self.tag)
        # log("END OF MOVEBY\n")

class PagePerformMain(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=ColorBg)

        self.frame = tk.Frame(self, background=ColorBg)
        self.frame.pack(side=tk.TOP, fill=tk.BOTH, expand=True, pady=5)

        self.performButton = {}
        self.advancedButtons = {}
        self.buttonNames = []

        self.makePerformButton("Reset_All", self.controller.resetAll)
        self.makePerformButton("Clear_ ", self.controller.clear)
        self.makePerformButton("Help_ ", self.controller.startHelp)

        # More advanced buttons
        self.makePerformButtonAdvanced("loopingonoff",None)
        self.makePerformButtonAdvanced("loopingfade",None)
        self.makePerformButtonAdvanced("loopinglength",None)
        self.makePerformButtonAdvanced("scale",None)
        self.makePerformButtonAdvanced("transpose",None)
        self.makePerformButtonAdvanced("autotranspose",None)
        self.makePerformButtonAdvanced("Notes_Off", self.controller.sendANO)

        self.makePerformButtonAdvanced("quant",None)
        self.makePerformButtonAdvanced("vol",None)
        self.makePerformButtonAdvanced("midithru",None)
        self.makePerformButtonAdvanced("midisetscale",None)
        self.makePerformButtonAdvanced("midiusescale",None)
        self.makePerformButtonAdvanced("midithruscadjust",None)
        self.makePerformButtonAdvanced("midiquantized",None)

        ### self.makePerformButtonAdvanced("Comb_Notes", self.controller.combLoop)

    def makePerformButtonAdvanced(self,name,f):
        self.advancedButtons[name] = 0
        self.makePerformButton(name,f)

    def updatePerformButtonLabels(self,pad):
        # self.controller.performButtonsPerRow = 5
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

            guiLevel = self.controller.guiLevel
            if name == "TBD" or (guiLevel==0 and name in self.advancedButtons):
                button.grid_forget()
            else:
                style = 'PerformButton.TLabel'
                button.config(text=text, width=11, style=style)
                button.grid(row=row,column=col, padx=self.controller.performButtonPadx,pady=self.controller.performButtonPady,ipady=ipady)
            col += 1
            if col >= self.controller.performButtonsPerRow:
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

        log("Perform Button Pressed",name)
        if name in palette.PerformLabels:

            if controller.currentPageName == "quad" or controller.allPadsSelected:
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
            log("UNHANDLED performCallback name=",name)

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

        self.config(background=ColorBg)

        self.selectButtons = {}
        self.selectOffset = 0

        self.valsframe = tk.Frame(self, background=ColorBg)
        self.valsframe.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, pady=10)

        self.scrollbar = ScrollBar(parent=self, notify=self)

        # self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=11, padx=5)
        # self.doLayout()

    def scrollNotify(self,sfy,tag):
        # log("scrollNotify sfy=",sfy," tag=",tag)
        nparams = len(self.vals)
        selectPerPage = self.controller.selectDisplayRows * self.controller.selectDisplayPerRow
        tmp = int(sfy * (nparams-selectPerPage))
        self.selectOffset = int(tmp / self.controller.selectDisplayPerRow) * self.controller.selectDisplayPerRow
        # silly code
        if self.selectOffset > (nparams-selectPerPage-self.controller.selectDisplayPerRow):
            self.selectOffset = nparams - selectPerPage
        if self.selectOffset < 0:
            self.selectOffset = 0
        self.doLayout()

    def doLayout(self):
        valindex = self.selectOffset

        ipadx = 0
        ipady = 0
        padx = self.controller.selectButtonPadx
        pady = self.controller.selectButtonPady

        if self.controller.guiLevel == 0 or self.pagename == "quad":
            nrows = self.controller.selectDisplayRowsNormal
        else:
            nrows = self.controller.selectDisplayRowsAdvanced
        nbuttons = self.controller.selectDisplayPerRow * nrows
        nvals = len(self.vals)
        if nvals <= nbuttons:
            # get rid of the scrollbar and adjust the button layout factors
            self.scrollbar.pack_forget()
            buttonwidth=17
            ipadx = 2
            padx -= 3
        else:
            # scrollbar is present
            self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=11, padx=4)
            buttonwidth=13
            pady -= 1

        # log("doLayout page=",self.pagename," nbuttons=",nbuttons, "nvals=",nvals, "nrows=",nrows,"buttonwidth=",buttonwidth)

        self.valsframe.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, pady=10)

        for i in self.selectButtons:
            self.selectButtons[i].grid_forget()

        i = 0
        for r in range(0,self.controller.selectDisplayRows):
            for c in range(0,self.controller.selectDisplayPerRow):
                if valindex < len(self.vals):

                    selectButtonText = self.vals[valindex]
                    istwo = isTwoLine(selectButtonText)
                    if istwo:
                        style='PresetButton.TLabel'
                        selectButtonText = selectButtonText.replace(palette.LineSep,"\n",1)
                        selectButtonText = selectButtonText.replace(palette.LineSep," ")
                    else:
                        style='PresetButton.TLabel'
                        selectButtonText = selectButtonText + "\n"

                    # First time here, we create the Button
                    if not i in self.selectButtons:
                        if i > len(self.vals):
                            log("Hey, i > len(self.vals) ??")
                        self.selectButtons[i] = ttk.Button(self.valsframe, width=buttonwidth, style=style)

                    self.selectButtons[i].grid(row=r,column=c,padx=padx,pady=pady,ipady=ipady,ipadx=ipadx)
                    self.selectButtons[i].config(text=selectButtonText, width=buttonwidth,
                        command=lambda val=self.vals[valindex],buttoni=i:self.selectorCallback(val,buttoni))
                    valindex += 1

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
                s = 'PresetButtonHighlight.TLabel'
            else:
                s = 'PresetButton.TLabel'
            self.selectButtons[i].config(style=s)

def afterWindowIsDisplayed(windowName,guisize,*args):
    global PaletteApp
    resizecoords = palette.ConfigValue("guiresize")
    # Hardcoded - the small size doesn't pay attention to guiresize
    if guisize == "large" and resizecoords != "":
        time.sleep(1.0) # wait for window to be visible so nircmdc sees it
        log("Resizing GUI")
        cmd = "nircmdc.exe win setsize stitle \""+windowName+"\" "+resizecoords
        os.system(cmd)
        cmd = "nircmdc.exe win -style stitle \""+windowName+"\" 0x00CA0000"
        os.system(cmd)
        cmd = "nircmdc.exe win max stitle \""+windowName+"\""
        os.system(cmd)
    PaletteApp.nextMode = "layout"

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
        log("SaveJsonInPath: unable to open path=",fpath)
        return
    f.write(json.dumps(j, sort_keys=True, indent=4, separators=(',',':')))
    # To avoid complaints from editors, add a final newline
    f.write("\n")
    f.close()

def initMain(app):
    app.iconbitmap(palette.configFilePath("palette.ico"))
    app.mainLoop()

def setFontSizes(guisize):
    global presetButtonFont, largestFont
    global hugeFont, comboFont, largerFont, largeFont, performButtonFont
    global padLabelFont, paramNameFont, paramValueFont, paramAdjustFont

    f = 'Open Sans Regular'

    if guisize == "large":
        presetButtonFont = (f, int(20))
        largestFont = (f, int(24))
        hugeFont = (f, int(36))
        comboFont = (f, int(20))
        largerFont = (f, int(20))
        largeFont = (f, int(16))
        performButtonFont = (f, int(14))
        padLabelFont = (f, int(22))
        paramNameFont = (f, int(18))
        paramValueFont = (f, int(18))
        paramAdjustFont = (f, int(20))
    else:
        presetButtonFont = (f, int(10))
        largestFont = (f, int(12))
        hugeFont = (f, int(18))
        comboFont = (f, int(10))
        largerFont = (f, int(10))
        largeFont = (f, int(8))
        performButtonFont = (f, int(7))
        padLabelFont = (f, int(8))
        paramNameFont = (f, int(8))
        paramValueFont = (f, int(8))
        paramAdjustFont = (f, int(6))

def makeStyles(app):
    app.option_add('*TCombobox*Listbox.font', comboFont)

    s = ttk.Style()

    s.configure('.', font=largeFont, background=ColorBg, foreground=ColorText)

    s.configure('PageButtonEnabled.TLabel', background=ColorHigh, relief="flat", justify=tk.CENTER, font=largestFont)
    s.configure('PageButtonDisabled.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, font=largestFont)

    s.configure('RandEtcButton.TLabel', font=largerFont, foreground=ColorText, background=ColorButton)
    s.configure('RandEtcButtonHigh.TLabel', font=largerFont, foreground=ColorText, background=ColorHigh)

    s.configure('ParamName.TLabel', font=paramNameFont, foreground=ColorText, justify=tk.LEFT)
    s.configure('ParamValue.TLabel', font=paramValueFont, foreground=ColorText, borderwidth=2, justify=tk.RIGHT, background=ColorBg)
    s.configure('ParamAdjust.TLabel', foreground=ColorText, borderwidth=2, anchor=tk.CENTER, background=ColorButton, font=paramAdjustFont)

    s.configure('GlobalButton.TLabel', font=largestFont, background=ColorButton, relief="flat", justify=tk.CENTER)

    s.configure('PerformMessage.TLabel', background=ColorBg, foreground=ColorRed, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=performButtonFont)

    s.configure('Loading.TLabel', background=ColorButton, foreground=ColorWhite, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largestFont)
    s.configure('Attract.TLabel', background=ColorBg, foreground=ColorWhite, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largestFont)
    s.configure('PerformHeader.TLabel', background=ColorButton, foreground=ColorBright, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=performButtonFont)

    s.configure('PresetButton.TLabel', foreground=ColorText, font=presetButtonFont, background=ColorButton, anchor=tk.CENTER, justify=tk.CENTER)
    s.configure('PresetButtonHighlight.TLabel', foreground=ColorText, font=presetButtonFont, background=ColorHigh, anchor=tk.CENTER, justify=tk.CENTER)

    s.configure('RecordingButton.TLabel', background=ColorRed, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largeFont)

    s.configure('PerformButton.TLabel', foreground=ColorText, background=ColorButton, relief="flat", justify=tk.CENTER,
        anchor=tk.CENTER, font=performButtonFont)

    s.configure('custom.TCombobox', foreground=ColorComboText, background=ColorBg)

    s.map('Patch.TLabel',
        foreground=[('disabled', 'yellow'),
                    ('pressed', ColorText),
                    ('active', ColorText)],
        background=[('disabled', 'yellow'),
                    ('pressed', ColorHigh),
                    ('active', ColorButton)]
        )
    s.map('PresetButton.TLabel',
        foreground=[('disabled', 'yellow'),
                    ('pressed', ColorText),
                    ('active', ColorText)],
        background=[('disabled', 'yellow'),
                    ('pressed', ColorHigh),
                    ('active', ColorButton)]
        )
    s.map('PerformButton.TLabel',
        foreground=[('disabled', 'yellow'),
                    ('pressed', ColorText),
                    ('active', ColorText)],
        background=[('disabled', 'yellow'),
                    ('pressed', ColorHigh),
                    ('active', ColorButton)]
        )

def log(*args):
    palette.log(*args)

def debug(*args):
    palette.debug(*args)

def loginit():
    palette.loginit()

KillNATS = False

async def nats_listen_for_palette(loop,app):

    nc = NATS()
    await nc.connect("nats://127.0.0.1:4222", loop=loop)

    async def palette_event_handler(msg):
        subject = msg.subject
        reply = msg.reply
        data = msg.data.decode()
        if data == "exit":
            global KillNATS
            KillNATS = True

        # log("Received a message on '{subject} {reply}': {data}".format(
        #     subject=subject, reply=reply, data=data))

        if subject == "palette.event":
            j = json.loads(data)
            event = j["event"]
            if event == "alive":
                secs = j["seconds"]
                count = j["cursorcount"]
                if int(count) > 0:
                    # log("cursor events, calling resetLastAnything!!")
                    global PaletteApp
                    PaletteApp.resetLastAnything()


    # "*" matches any token, at any level of the subject.
    await nc.subscribe("palette.event", cb=palette_event_handler)

    # WAIT FOR INCOMING MESSAGES
    while KillNATS == False:
        await asyncio.sleep(.1)

    log("KillNATS caused exit")

    # Gracefully close the connection.
    await nc.drain()


def background_thread(app):  # runs in background thread

    loop = asyncio.new_event_loop()
    loop.run_until_complete(nats_listen_for_palette(loop,app))
    loop.close()
    log("background_thread has finished?")

if __name__ == "__main__":

    loginit()
    log("")
    log("Start GUI")
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
            "quad":"Quad",
            "snap":"Pad",
            "sound":"Sound",
            "visual":"Visual",
            "effect":"Effect",
        }
    else:
        log("Unexpected number of pads: ",pads)

    guisize = palette.ConfigValue("guisize")
    if len(sys.argv) > 1:
        guisize = sys.argv[1]

    if guisize == "":
        guisize = "large"

    global PaletteApp
    PaletteApp = ProGuiApp(padname,padnames,visiblepagenames,guisize)

    makeStyles(PaletteApp)

    if guisize == "large":
        PaletteApp.wm_geometry("%dx%d" % (800,1280))
    elif guisize == "small":
        PaletteApp.wm_geometry("%dx%d" % (400,640))
    else:
        log("Unexpected value of guisize (%s)\n" % guisize)

    PaletteApp.nextMode = ""
    PaletteApp.currentMode = ""

    threading.Timer(0.0, afterWindowIsDisplayed, args=[PaletteApp.windowName,guisize], kwargs=None).start()

    thd = threading.Thread(target=background_thread,args=(PaletteApp,))   # timer thread
    thd.daemon = True
    thd.start()  # start timer loop

    initMain(PaletteApp)
