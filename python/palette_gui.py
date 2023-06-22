# A GUI for Palette

from tkinter import ttk
from tkinter import font
import tkinter as tk
from tkinter import messagebox

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

import palette

FourPatches = False
RecMode = False
# DoAttractStuff = False

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

def killApp():
    log("killApp called!")
    global PaletteApp
    PaletteApp.killme = True
    time.sleep(1.00)
    # the .killme method doesn't seem to work all the time
    PaletteApp.destroy()
    log("killApp after destroy!")

def controlCHandler(sig, frame):
    log("controlCHandler called!")
    killApp()

palette.NoticeKeyboardInterrupt(controlCHandler)

class Params():
    def __init__(self):
        pass

def on_closing():
    log("on_closing called!")
    killApp()

class ProGuiApp(tk.Tk):

    def __init__(self,
            patchname,
            patchnames,
            visiblepagenames,
            guisize
            ):

        self.guisize = guisize
        log("ProGuiApp guisize=",guisize)
        self.killme = False

        self.currentMode = ""
        self.nextMode = ""
        self.lastLoadType = ""
        self.lastLoadName = ""
        self.isLooping = False

        log("PALETTE_GUI_LEVEL env = "+ os.environ.get("PALETTE_GUI_LEVEL",""))

        self.defaultGuiLevel = int(os.environ.get("PALETTE_GUI_LEVEL","0"))

        self.currentPageName = None

        self.setGuiLevel(self.defaultGuiLevel)

        self.thumbFactor = 0.1

        # These are the same in both normal and advanced
        self.selectDisplayPerRow = 3

        # Normal layout values
        if self.guisize == "palette":
            self.paramDisplayRows = 22
            self.frameSizeOfControlNormal = 0.06
            self.frameSizeOfSelectNormal = 1.0 - self.frameSizeOfControlNormal
            self.frameSizeOfPatchChooserNormal = 0.0
            self.selectDisplayRowsNormal = 14

            self.frameSizeOfControlAdvanced = 0.06 # 0.15
            self.frameSizeOfPatchChooserAdvanced = 0.13
            self.frameSizeOfSelectAdvanced = 1.0 - self.frameSizeOfControlAdvanced - self.frameSizeOfPatchChooserAdvanced
            self.selectDisplayRowsAdvanced = 11

            self.performButtonPadx = 3
            self.performButtonPady = 2
            self.performButtonsPerRow = 6
        elif self.guisize == "medium":
            self.paramDisplayRows = 23
            self.frameSizeOfControlNormal = 0.085
            self.frameSizeOfPatchChooserNormal = 0.0
            self.frameSizeOfSelectNormal = 1.0 - self.frameSizeOfControlNormal
            self.selectDisplayRowsNormal = 14

            self.frameSizeOfControlAdvanced = 0.085 #  0.19
            self.frameSizeOfPatchChooserAdvanced = 0.14
            self.frameSizeOfSelectAdvanced = 1.0 - self.frameSizeOfControlAdvanced - self.frameSizeOfPatchChooserAdvanced
            self.selectDisplayRowsAdvanced = 12 # 9

            self.performButtonPadx = 3
            self.performButtonPady = 3
            self.performButtonsPerRow = 6
        else:
            if self.guisize != "small":
                log("Unknown guisize=",self.guisize," assuming small")
            self.paramDisplayRows = 23
            # self.frameSizeOfControlNormal = 0.085
            self.frameSizeOfControlNormal = 0.10
            self.frameSizeOfPatchChooserNormal = 0.0
            self.frameSizeOfSelectNormal = 1.0 - self.frameSizeOfControlNormal
            self.selectDisplayRowsNormal = 12

            self.frameSizeOfControlAdvanced = 0.085 #  0.19
            self.frameSizeOfPatchChooserAdvanced = 0.12
            self.frameSizeOfSelectAdvanced = 1.0 - self.frameSizeOfControlAdvanced - self.frameSizeOfPatchChooserAdvanced
            self.selectDisplayRowsAdvanced = 11 # 9

            self.performButtonPadx = 3
            self.performButtonPady = 3
            self.performButtonsPerRow = 6

        df = (self.frameSizeOfSelectAdvanced + self.frameSizeOfControlAdvanced + self.frameSizeOfPatchChooserAdvanced) - 1.0
        if df > 0.01 or df < -0.01:
            log("Hey, page sizes don't add up to 1.0")


        self.selectButtonPadx = 5
        self.selectButtonPady = 3

        setFontSizes(self.guisize)

        tk.Tk.__init__(self)

        self.AllPageNames = {
                "engine":0,
                "quad":0,
                "patch":0,
                "sound":0,
                "visual":0,
                "effect":0,
                "misc":0}

        self.visiblePageNames = visiblepagenames
        self.PatchNames = collections.OrderedDict()
        num = 1
        for ch in patchnames:
            self.PatchNames[ch] = num
            num = num + 1

        self.readParamDefs()

        self.allPatchesSelected = True
        self.Patches = {}
        for patchName in self.PatchNames:
            p = Patch(self,patchName)
            self.Patches[p] = patchName

        self.engine = Engine(self)

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
        self.showPatchFeedback = True
        self.showCursorFeedback = False

        self.windowName = "Palette Control"
        self.winfo_toplevel().title(self.windowName)

        self.escapeCount = 0
        self.lastEscape = time.time()
        self.lastClearLoop = time.time()
        self.layoutDone = False

        self.initLayout()
        self.resetAll()

    def placePatchChooser(self):
        if self.guiLevel > 0:
            self.patchChooser.place(in_=self.topContainer, relx=0, rely=self.patchChooserPageY, relwidth=1, relheight=self.frameSizeOfPatchChooser)
        else:
            self.patchChooser.place_forget()

    def forgetPatchChooser(self):
        self.patchChooser.place_forget()

    def scrollWheel(self,event):
        if self.editMode:
            scrollbar = self.editPage[self.currentPageName].scrollbar
        else:
            scrollbar = self.selectorPage[self.currentPageName].scrollbar
        scrollbar.scrollWheel(event)

    def mainLoop(self):
        # doneLoading = False
        while self.killme == False:
            # log("time="+str(time.time()))
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
    
            try:
                time.sleep(0.001)
            except:
                log("mainLoop sleep interrupted")
                pass

            now = time.time()

            if self.nextMode != "":

                # log("nextMode=",self.nextMode)
                # switch to a new Mode 
                if self.nextMode == "layout":
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

            elif self.currentMode == "attract":
                # self.doAttractAction()
                pass

            elif self.currentMode == "help":
                self.doHelpAction()

            elif self.currentMode == "startup":
                self.doStartupAction()

            else:
                log("Invalid value for currentMode: ",self.currentMode)
                self.currentMode = ""

            # log("time C ="+str(time.time()))

        log("mainLoop is returning")

    def loopingOnOff(self):
        self.loopingClear()
        if self.isLooping:
            self.loopingOff()
            palette.palette_engine_api("audio_reset")
        else:
            self.loopingOn()

    def loopingOn(self):

        self.performPage.setPerformButtonText("Looping","LOOPING_IS ON ",'PerformButtonHighlight.TLabel')
        self.update()

        self.isLooping = True
        log("loopingOn")

        s, err = palette.palette_engine_get("engine.looping_override")
        if err != None:
            log("Error in getting value of engine.looping_override")
            return
        force = palette.boolValueOfString(s)
        if force:
            palette.palette_engine_set("engine.looping_on", "true")
            forcefade, err = palette.palette_engine_get("engine.looping_fade")
            if err != None:
                log("Error in getting value of engine.looping_fade")
                return
            forcebeats, err = palette.palette_engine_get("engine.looping_beats")
            if err != None:
                log("Error in getting value of engine.looping_beats")
                return

        for patch in self.Patches:

            palette.palette_patch_set(patch.name(), "misc.looping_on", "true")
            if force:
                palette.palette_patch_set(patch.name(), "misc.looping_fade", forcefade)
                palette.palette_patch_set(patch.name(), "misc.looping_beats", forcebeats)

            # This is overkill
            self.refreshValues("misc",patch)

    def loopingOff(self):
        self.performPage.setPerformButtonText("Looping","LOOPING_IS OFF",'PerformButton.TLabel')
        self.update()

        self.isLooping = False
        log("loopingOff")

        palette.palette_engine_set("engine.looping_on", "false")

        for patch in self.Patches:
            palette.palette_patch_set(patch.name(), "misc.looping_on", "false")
            self.refreshValues("misc",patch)

    def loopingClear(self):
        log("loopingClear")
        for patch in self.Patches:
            palette.palette_patch_api(patch.name(), "clear", "")

    def startNormalMode(self):
        # self.startupFrame.place_forget()
        # log("startNormalMode: setting nextMode to normal")
        self.resetMinMaxXY()
        self.nextMode = "normal"
        self.attractFrame.place_forget()
        self.helpFrame.place_forget()
        self.resetVisibility()

    def startAttractMode(self):

        # log("startAttractMode: setting nextMode to attract")
        self.nextMode = "attract"
        self.lastAttractSpriteTime = 0
        self.selectFrame.place_forget()
        self.performFrame.place_forget()
        self.patchChooser.place_forget()
        self.helpFrame.place_forget()
        self.attractFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=1)

    def startHelpMode(self):
        self.selectFrame.place_forget()
        self.performFrame.place_forget()
        self.patchChooser.place_forget()
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
        self.patchChooser = self.makePatchChooserFrame(parent=self.topContainer,controller=self)

        self.performPage = PagePerformMain(parent=self.performFrame, controller=self)

        self.topContainer.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.topContainer.bind_all("<MouseWheel>", self.scrollWheel)

        self.performPage.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

        self.resetVisibility()

        # select the initial patch
        self.patchChooserCallback(patchname)
        self.allPatchesSelected = True
        self.patchChooser.refreshColors()

    def popup(self,msg):
        usemessagebox = True
        if usemessagebox:
            windowName = "Palette Message"
            messagebox.showinfo(windowName,msg)
        else:
            # XXX - The problem with this approach is that
            # the window can end up on a different monitor
            # than the palette gui.
            win = tk.Toplevel(highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3, background=ColorBg)
            win.wm_title(windowName)
            win.iconbitmap(palette.configFilePath("palette.ico"))
    
            l = tk.Label(win, text=msg, background=ColorBg, foreground=ColorText)
            l.grid(row=0, column=0)
    
            b = ttk.Button(win, text="Okay", command=win.destroy)
            b.grid(row=1, column=0,pady=(10,30))

    def doStartupAction(self):
        pass

    def randomSprite(self,patch,downup):
        x = random.random()
        y = random.random()
        z = 0.6 - random.random() / 2.0
        palette.SendSpriteEvent("0",x,y,z,patch)

    def doHelpAction(self):
        pass

    def doSelectorAction(self):
        if self.selectorAction == "LOAD":
            self.selectorLoadAndSend(self.currentPageName,self.selectorValue)

        elif self.selectorAction == "IMPORT":
            self.selectorImportAndSend(self.currentPageName,self.selectorValue)

        elif self.selectorAction == "INIT":
            self.selectorApply("init",self.currentPageName)

        elif self.selectorAction == "RAND":
            self.selectorApply("rand",self.currentPageName)

        self.selectorAction = ""
    
    def resetVisibility(self):
        self.editMode = False
        self.setFrameSizes()

        # default page selection
        self.selectPage("quad")

        self.placeFrames()

        self.pageHeader.repack()

    def placeFrames(self):
        if self.guiLevel == 0:
            self.performFrame.place(in_=self.topContainer, relx=0, rely=self.performPageY, relwidth=1, relheight=self.frameSizeOfControl)
            self.selectFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=self.frameSizeOfSelect)
            self.placePatchChooser()
        else:
            self.performFrame.place(in_=self.topContainer, relx=0, rely=self.performPageY, relwidth=1, relheight=self.frameSizeOfControl)
            self.selectFrame.place(in_=self.topContainer, relx=0, rely=0, relwidth=1, relheight=self.frameSizeOfSelect)
            self.placePatchChooser()

    def setFrameSizes(self):

        if self.guiLevel == 0:
            self.frameSizeOfControl = self.frameSizeOfControlNormal
            self.frameSizeOfSelect = self.frameSizeOfSelectNormal
            self.frameSizeOfPatchChooser = self.frameSizeOfPatchChooserNormal
            self.selectDisplayRows = self.selectDisplayRowsNormal

        elif self.currentPageName == "quad":
            # quad page layout used to omit the PatchChooser,
            # but I put it back, so this is the same as below
            self.frameSizeOfControl = self.frameSizeOfControlAdvanced
            self.frameSizeOfSelect = self.frameSizeOfSelectAdvanced
            self.frameSizeOfPatchChooser = self.frameSizeOfPatchChooserAdvanced
            self.selectDisplayRows = self.selectDisplayRowsAdvanced

        else:
            # Advanced is any guiLevel>0
            self.frameSizeOfControl = self.frameSizeOfControlAdvanced
            self.frameSizeOfSelect = self.frameSizeOfSelectAdvanced
            self.frameSizeOfPatchChooser = self.frameSizeOfPatchChooserAdvanced
            self.selectDisplayRows = self.selectDisplayRowsAdvanced

        y = 0
        self.selectPageY = y
        y += self.frameSizeOfSelect
        self.patchChooserPageY = y
        y += self.frameSizeOfPatchChooser
        self.performPageY = y
        y += self.frameSizeOfControl
 
    def makePatchChooserFrame(self,parent,controller):
        f = PatchChooser(parent,controller)
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

        # These are the pages of buttons for selecting sound/visual/effect/misc/etc..
        # Each one has a SelectorPage with the saved buttons,
        # and an EditPage with all the parameters
        for pagename in self.visiblePageNames:
            self.makeSelectorPage(f, pagename, PageSelector)
            self.makeEditPage(f,pagename)

        return f

    def unattract(self):
        log("Screen pressed, stopping attract mode, setting nextMode to normal")
        palette.palette_engine_api("attract",
            "\"onoff\": \"false\"")
        self.nextMode = "normal"

    def unhelp(self):
        self.checkForGuiCycle()
        self.nextMode = "normal"

    def makeAttractFrame(self,container):

        path = palette.configFilePath("sppro_attractscreen.png")
        self.attractImage = tk.PhotoImage(file=path)

        f = tk.Frame(container,
            highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3)
        button = ttk.Button(f, image=self.attractImage, style='Attract.TLabel',
            command=self.unattract)
        button.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        return f

    def makeHelpFrame(self,container):

        path = palette.configFilePath("sppro_helpscreen.png")
        self.helpImage = tk.PhotoImage(file=path)

        f = tk.Frame(container,
            highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3)
        button = ttk.Button(f, image=self.helpImage, style='Attract.TLabel',
            command=self.unhelp)
        button.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        self.FullWidth = button.winfo_width()
        self.FullHeight = button.winfo_height()
        button.bind("<Motion>", self.motionCallback)
        return f

    # The routines here maintain the min/max of the mouse
    # position while the help page is displayed.
    # If you sweep from the upper left to the lower right of the help page,
    # it will cycle through the gui levels
    def resetMinMaxXY(self):
        self.minX = 10000
        self.minY = 10000
        self.maxX = 0
        self.maxY = 0
    
    def motionCallback(self,e):
        if e.x > self.maxX:
            self.maxX = e.x
        if e.y > self.maxY:
            self.maxY = e.y
        if e.x < self.minX:
            self.minX = e.x
        if e.y < self.minY:
            self.minY = e.y

    def checkForGuiCycle(self):
        smallx = PaletteAppSize["width"] / 4
        smally = PaletteAppSize["height"] / 4
        bigx = PaletteAppSize["width"] * 3 / 4
        bigy = PaletteAppSize["height"] * 3 / 4
        if self.minX < smallx and self.minY < smally and self.maxX > bigx and self.maxY > bigy:
            self.cycleGuiLevel()

    def updateSelectorPage(self,pagename,files):
        page = self.selectorPage[pagename]
        page.vals = files
        page.doLayout()
       
    def makeSelectorPage(self,parent,pagename,pagemaker):
        vals = palette.savedListAll(pagename)

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

        # A second click on a page header will toggle editMode
        if self.guiLevel > 0 and self.currentPageName == pagename:
            self.editMode = not self.editMode

        self.selectPage(pagename)
        self.lastLoadType = ""
        self.lastLoadName = ""

        if self.editMode and pagename == "engine":
            self.refreshEngineValues(pagename)
        elif self.editMode and pagename != "patch":
            if self.allPatchesSelected:
                patch = self.patchNamed("A")
            else:
                patch = self.CurrPatch

            self.refreshValues(pagename,patch)

        self.setFrameSizes()
        self.placeFrames()

    def refreshEngineValues(self,pagename):
        page = self.editPage[pagename]
        for name in self.engine.params:
            ptype = self.paramTypeOf[name]
            if pagename != "patch" and ptype != pagename:
                continue
            value, err = palette.palette_engine_get(name)
            if err != None:
                log("Error in getting value of "+name)
                continue
            if isinstance(value,tuple):
                value = value[0]
            w = page.paramValueWidget[name]
            value = page.normalizeJsonValue(name,value)
            w.config(text=value)
            # Need to set the value in local params values 
            # log("refreshEngineValue, name=",name," value=",value)
            self.engine.setValue(name,value)

    def refreshValues(self,pagename,patch):
        page = self.editPage[pagename]
        for name in patch.params:
            ptype = self.paramTypeOf[name]
            if pagename != "patch" and ptype != pagename:
                continue
            value, err = palette.palette_patch_api(patch.name(), "get",
                "\"name\": \"" + name + "\"")
            if err != None:
                log("Error in getting value of "+name)
                continue
            if isinstance(value,tuple):
                value = value[0]
            w = page.paramValueWidget[name]
            value = page.normalizeJsonValue(name,value)
            w.config(text=value)
            # Need to set the value in local params values 
            patch.setValue("",name,value)

    def patchNamed(self,patchName):
        lastResort = None
        for patch in self.Patches:
            lastResort = patch
            if patch.name() == patchName:
                return patch
        log("There is no Patch named ",patchName,", last resort, using",lastResort.name())
        return lastResort

    def selectPage(self,pagename):

        self.currentPageName = pagename
        self.pageHeader.highlightPageButton(pagename)

        self.forgetPages(self.selectorPage)
        self.forgetPages(self.editPage)

        self.placePatchChooser()

        if self.guiLevel > 0 and self.editMode:
            page = self.editPage[pagename]
        else:
            page = self.selectorPage[pagename]

        self.setFrameSizes()

        if not self.editMode:
            page.doLayout()

        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        page.tkraise()

    def doAllPatches(self):
        return self.allPatchesSelected or self.currentPageName=="patch"

    def sendParamValues(self,values):
        log("sendParamValues: ",str(values))
        for name in values:
            v = values[name]
            if self.doAllPatches():
                for patch in self.Patches:
                    patch.sendParamValue(name,v)
            else:
                self.CurrPatch.sendParamValue(name,v)

    def changeAndSendValue(self,paramType,basename,value,sendit=True):

        if paramType == "engine":
            self.engine.paramValues[basename] = value
            if sendit:
                self.sendEngineValue(basename,value)

        elif self.doAllPatches():
            for patch in self.Patches:
                patch.setValue(paramType,basename,value)
                if sendit:
                    patch.sendValue(paramType,basename)

        else:
            self.CurrPatch.setValue(paramType,basename,value)
            if sendit:
                self.CurrPatch.sendValue(paramType,basename)

    def sendEngineValue(self,basename,value):

        if not basename.startswith("engine."):
            basename = "engine." + basename
        palette.palette_engine_set(basename,str(value))

    def selectorApply(self,apply,paramType):

        # log("selectorApply 1")
        if not self.editMode:
            log("Hmmm, selectorAply should only be called in editMode")
            return

        if paramType == "patch":
            log("selectorApply not implemented on patch")
        else:

            log("selectorApply: before applyToAllParams")
            self.applyToAllParams(apply,paramType,False)  # don't send yet
            log("selectorApply: after applyToAllParams")
            self.refreshPage()

            if paramType == "engine":
                paramlistjson = self.paramListOfType("engine",self.engine.getValue)
                palette.palette_engine_api("setparams", paramlistjson)
                self.engine.sendParamsOfType(paramType)

            elif self.allPatchesSelected:
                log("Sending ",paramType," params to all patch")
                for patch in self.Patches:
                    patch.sendParamsOfType(paramType)
                log("After sending ",paramType," params to all patch")
            else:
                log("Sending ",paramType," params to patch ",self.CurrPatch.name())
                self.CurrPatch.sendParamsOfType(paramType)

    def applyToAllParams(self,apply,paramType,sendit=True):
        # loop through all the parameters of a given type
        for name in self.allParamsJson:
            j = self.allParamsJson[name]
            if paramType != "patch" and j["paramstype"] != paramType:
                continue
            valuetype = j["valuetype"]
            (p,basename) = patchOfParam(name)
            if p != None:
                log("Unexpected patch in param value?")
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
                self.changeAndSendValue(paramType,basename,v,sendit)

    def selectorLoadAndSend(self,paramType,savedname):
        if self.editMode:
            log("HEY!! selectorLoadAndSend shouldn't be used in editMode?")
            return
        self.loadAndSend(paramType,savedname)

    def loadAndSend(self,category,filename):

        # a second click on the same saved will switch to edit mode
        # (same thing as click on the page header)
        # THIS MAY BE A BAD IDEA
        # if savedtype == self.lastLoadType and savedname == self.lastLoadName:
        #     self.clickPage(savedtype)

        self.lastLoadType = category
        self.lastLoadName = filename
        fullsavedname = category+"."+str(filename)
        self.editPage[category].paramsnameVar.set(filename)

        if category == "engine":
            # log("Loading","category","engine","filename",filename)
            palette.palette_quadpro_api("load",
                "\"filename\": \"" + filename + "\""
                ", \"category\": \"" + category + "\"")
        elif category == "quad":
            if self.guiLevel == 0 or self.allPatchesSelected:
                # in casual instrument mode, loading a quad will ignore the patch selections
                # because in casual mode, the patch selectors aren't shown.
                # In non-casual mode (guiLevel>0) we do this if allPatchesSelected
                log("Loading","category","quad","filename",filename)
                palette.palette_quadpro_api("load",
                    "\"filename\": \"" + filename + "\""
                    ", \"category\": \"" + category + "\"")
                log("After Loading","category","quad","filename",filename)
            else:
                # Otherwise, in "pro" mode,
                # the quad is loaded only into a single patch
                patchName = self.CurrPatch.name()
                self.patchLoad(patchName,category,filename)
                # palette.palette_quadpro_api("save",
                #     "\"filename\": \"" + "_Current" + "\""
                #     ", \"category\": \"" + "quad" + "\"")

        elif self.allPatchesSelected:
            for patch in self.Patches:
                self.patchLoad(patch.name(),category,filename)
        else:
            patchName = self.CurrPatch.name()
            self.patchLoad(patchName,category,filename)

        log("Before checking looping")
        if self.isLooping:
            log("loadAndSend: reloading looping On")
            self.loopingOn()
        else:
            log("loadAndSend: reloading looping Off")
            self.loopingOff()

    def patchLoad(self,patchName,category,filename):
        # log("patchLoad","patch",patchName,"category",category,"filename",filename)
        palette.palette_patch_api(patchName, "load",
            "\"category\": \"" + category + "\""
            ", \"filename\": \"" + filename + "\"")

    def selectorLoadAndSendRand(self,savedType):

        if self.editMode:
            log("HEY!! selectorLoadAndSendRand shouldn't be used in editMode?")
            return

        saved = palette.savedListAll(savedType)
        nsaved = len(saved)
        if nsaved == 0:
            log("selectorLoadAndSendRand: no saved of type "+savedType+"?")
            return
        i = random.randint(0,nsaved-1)
        savedname = saved[i]
        self.loadAndSend(savedType,savedname)

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

        if paramType == "patch":
            self.loadPageJson(self.editPage["patch"],j)
            self.sendPage(self.editPage["patch"])
        elif paramType == "quad":
            log("Hey, does quad need work here?  FFF")
        else:
            log("Hey, Is this code obsolete, paramType="+paramType)
            # self.readOtherParamsJsonIntoSnapAndPreset(paramType,j)
            # log("Sending",paramType,"params to patch",self.CurrPatch.name())
            # self.CurrPatch.sendParamsOfType(paramType)

    def patchChooserCallback(self,patchName):
        self.CurrPatch = self.patchNamed(patchName)

        self.allPatchesSelected = False

        if len(self.PatchNames) > 1:
            self.patchChooser.refreshPatchColors()

        self.refreshPage()

        self.performPage.updatePerformButtonLabels(self.CurrPatch)

        self.editPage[self.currentPageName].updateParamView()

    def copyPatchToPage(self,patch,pageName):
        patchValues = patch.getValues()
        page = self.editPage[pageName]
        for nm in patchValues:
            page.changeValueText(nm,patchValues[nm])

    def refreshPage(self):
        if self.editMode:
            # If we're in edit mode,
            # make sure the values are updated from the Patch values
            if self.allPatchesSelected:
                self.copyPatchToPage(self.patchNamed("A"),self.currentPageName)
            else:
                self.copyPatchToPage(self.CurrPatch,self.currentPageName)

    def clear(self):
        if self.allPatchesSelected:
            for patch in self.Patches:
                patch.clear()
            palette.palette_engine_api("audio_reset")
        else:
            self.CurrPatch.clear()
        self.checkEscape()
 
    def checkEscape(self):

        # click the Clear button 4 times quickly to change the GuiLevel
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
        # cycle through 0,1 (used to be more levels)
        self.setGuiLevel((self.guiLevel + 1) % 2)
        self.resetVisibility()
        self.performPage.updatePerformButtonLabels(self.CurrPatch)
        log("GuiLevel",self.guiLevel)

    def setGuiLevel(self,level):
        self.guiLevel = level
#        self.setScaleList()

    def sendANO(self):
        for patch in self.Patches:
            patch.sendANO()

    def startHelp(self):
        self.nextMode = "help"

    def startProcess(self,processName):
        palette.palette_engine_api("startprocess","\"process\": \"" + processName + "\"")

    def stopProcess(self,processName):
        palette.palette_engine_api("stopprocess","\"process\": \"" + processName + "\"")

    def startAll(self):
        palette.palette_engine_api("startall")

    def stopAll(self):
        palette.palette_engine_api("stopall")

    def startRecording(self):
        palette.palette_engine_api("startrecording")

    def stopRecording(self):
        palette.palette_engine_api("stoprecording")

    def exit(self):
        os._exit(0)  # This is a hard exit, killing all the background threads

    def resetAll(self):

        log("ResetAll")

        self.loopingClear()
        # self.loopingOff()

        palette.palette_engine_api("audio_reset")

        self.setGuiLevel(self.defaultGuiLevel)

        self.allPatchesSelected = True
        self.CurrPatch = self.patchNamed("A")
        self.patchChooser.refreshPatchColors()
        self.sendANO()

        for patch in self.Patches:
            patch.clear()

        self.performPage.updatePerformButtonLabels(self.CurrPatch)

        self.resetVisibility()

        s, err = palette.palette_engine_get("engine.looping_override")
        if err != None:
            log("Error in getting value of engine.looping_override")
            return
        force = palette.boolValueOfString(s)
        if force:
            s, err = palette.palette_engine_get("engine.looping_on")
            if err != None:
                forceon = False
            else:
                forceon = palette.boolValueOfString(s)
            if forceon:
                self.loopingOn()
            else:
                self.loopingOff()
        else:
            self.loopingOff()

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

        # for convenience, add "paramstype" field to newParamsJson
        for name in self.newParamsJson:
            parts = name.split(".")
            prefix = ""
            if len(parts) > 1:
                prefix = parts[0]
            self.newParamsJson[name]["paramstype"] = prefix

        self.allParamsJson = self.convertParamdefsToParams(self.newParamsJson)

        self.paramenums = palette.readJsonPath(palette.configFilePath("paramenums.json"))
        self.allEffectsJson = palette.readJsonPath(palette.configFilePath("resolume.json"))
        self.paramValueTypeOf = {}
        self.paramsOfType = {}
        self.paramTypeOf = {}
        for name in self.newParamsJson:
            self.paramValueTypeOf[name] = self.newParamsJson[name]["valuetype"]

        # Construct lists of the parameters, pulled from Params.json
        for t in self.AllPageNames:
            self.paramsOfType[t] = collections.OrderedDict()

        self.newParamNames = []
        for x in sorted(self.newParamsJson.keys()):
            if not isVisibleParameter(x):
                continue
            self.newParamNames.append(x)
            self.newParamsJson[x]["name"] = x
            t = self.newParamsJson[x]["paramstype"]
            if t != "channel":
                self.paramsOfType[t][x] = self.newParamsJson[x]
                self.paramTypeOf[x] = self.newParamsJson[x]["paramstype"]

        # In addition to creating parameters for "patch",
        # we create all the parameters for the "quad" settings by
        # duplicating all the parameters for each patch (A,B,C,D)
        # and including the misc.* parameters.
        for name in self.newParamNames:

            self.paramValueTypeOf[name] = self.newParamsJson[name]["valuetype"]
            if "misc." in name:
                ptype = "patch"  # misc parameters are patch parameters
            elif "engine." in name:
                ptype = "engine"
            else:
                ptype = "patch"
            self.paramsOfType[ptype][name] = self.newParamsJson[name]

            if FourPatches:
                # We prepend A-, B-, etc to the parameter name for quad parameters,
                # to create entries for "quad" things
                # in paramValueTypeOf and paramsOfType["quad"]
                for patch in self.PatchNames:
                    quadName = PatchParamName(patch,name)
                    self.paramValueTypeOf[quadName] = self.newParamsJson[name]["valuetype"]
                    self.paramsOfType["quad"][quadName] = self.newParamsJson[name]

        # The things here get ADDED to the ones already read in from paramenums.json
        for pt in {"sound", "visual", "effect", "misc"}:
            if pt in self.paramenums:
                log("WARNING! pt=",pt," is already in paramenums.json!")
            else:
                self.paramenums[pt] = palette.savedListAll(pt)

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

    def paramListOfType(self,paramType,functoget):
        paramlist = ""
        sep = ""
        for name in self.newParamsJson:
            if not isVisibleParameter(name):
                continue
            j = self.newParamsJson[name]
            if paramType == "patch" or j["paramstype"] == paramType:
                paramname = name
                v = functoget(paramname)
                paramlist = paramlist + sep + "\"" + name + "\" : \"" + str(v) + "\""
                sep = ", "

        return paramlist

class Engine():
    
    def __init__(self, controller):
        self.paramValues = {}
        self.paramEnabled = {}
        self.controller = controller
        self.params = self.controller.paramsOfType["engine"]
        self.setInitValues()

    def setValue(self,paramName,val):
        if not paramName in self.paramValues:
            log("Hey, setValue fullname=",paramName,"not in paramValues?")
            return
        self.paramValues[paramName] = val
 
    def setInitValues(self):
        for paramName in self.params:
            self.paramValues[paramName] = self.params[paramName]["init"]

    def getValue(self,paramName):
        return self.paramValues[paramName]

class Patch():

    def __init__(self, controller, patchName):
        self.paramValues = {}
        self.paramEnabled = {}
        self.controller = controller
        self.params = self.controller.paramsOfType["patch"]
        self.setInitValues()
        self.patchName = patchName

    def name(self):
        return self.patchName

    def setInitValues(self):
        for paramName in self.params:
            self.paramValues[paramName] = self.params[paramName]["init"]

    def getValues(self):
        return self.paramValues

    def setValue(self,paramType,paramName,val):
        if not paramType == "" and not paramName.startswith(paramType):
            paramName = paramType + "." + paramName
        if not paramName in self.paramValues:
            log("Hey, setValue fullname=",paramName,"not in paramValues?")
            return
        self.paramValues[paramName] = val
 
    def getValue(self,paramName):
        return self.paramValues[paramName]
 
    def sendValue(self,paramType,paramName):
        if not paramType == "" and not paramName.startswith(paramType):
            paramName = paramType + "." + paramName
        if not paramName in self.paramValues:
            log("Hey, sendValue paramName=",paramName,"not in paramValues?")
            return
        val = self.paramValues[paramName]
        if not paramName in self.controller.paramTypeOf:
            log("Unrecognized parameter: ",paramName)
            return
        paramType = self.controller.paramTypeOf[paramName]
        # fullParamName = paramType + "." + paramName
        fullParamName = paramName
        palette.palette_patch_set(self.name(),fullParamName , str(val) )

    def sendParamsOfType(self,paramType):
        for pt in ["sound","visual","effect","misc"]:
            if paramType == "patch" or paramType == pt:
                paramlistjson = self.controller.paramListOfType(pt,self.getValue)
                palette.palette_patch_api(self.name(), "setparams", paramlistjson)

    def getPerformIndex(self,name):
        return self.performIndex[name]

    def setPerformIndex(self,name,index):
        self.performIndex[name] = index

    def sendANO(self):
        palette.palette_quadpro_api("ANO")

    def clear(self):
        palette.palette_patch_api(self.name(), "clear", "")

class PatchChooser(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)

        self.controller = controller
        self.parent = parent
        self.patchLabel = {}
        self.patchFrame = {}
        self.patchCanvas = {}
        self.canvasHeight = 60
        self.canvasWidth = 200
        self.PatchNum2Name = ["X","A","B","C","D"]

        self.makePatchFrame(self,"A",0.05,0.07)
        self.makePatchFrame(self,"B",0.15,0.53)
        self.makePatchFrame(self,"C",0.55,0.53)
        self.makePatchFrame(self,"D",0.65,0.07)

        self.makeAllButton(self,0.5,0.15)

        self.config(background=ColorBg)

    def makePatchFrame(self,parent,patch,x0,y0):

        self.patchFrame[patch] = tk.Frame(self)
        self.patchFrame[patch].place(relx=x0,rely=y0,relwidth=0.3,relheight=0.4)
        self.patchFrame[patch].config(borderwidth=2,relief="solid",background=ColorUnHigh)
        self.patchFrame[patch].bind("<Button-1>", lambda p=patch: self.patchCallback(p))

        if self.controller.showCursorFeedback:
            self.patchCanvas[patch] = tk.Canvas(self.patchFrame[patch], width=self.canvasWidth, height=self.canvasHeight, border=0)
            self.patchCanvas[patch].pack(side=tk.TOP)
            self.patchCanvas[patch].config(background=ColorUnHigh)

    def makeAllButton(self,parent,x0,y0):

        self.patchAllButton = tk.Frame(self)
        self.patchAllButton.place(relx=x0-0.05,rely=y0-0.05,relwidth=0.1,relheight=0.275)
        self.patchAllButton.config(borderwidth=2,relief="solid",background=ColorUnHigh)
        self.patchAllButton.bind("<Button-1>", self.globalCallback)

        self.patchGlobalLabel = ttk.Label(self.patchAllButton, text="*")
        self.patchGlobalLabel.pack(side=tk.TOP)
        self.patchGlobalLabel.configure(style='GlobalButton.TLabel')
        self.patchGlobalLabel.bind("<Button-1>", self.globalCallback)

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

        self.controller.allPatchesSelected = not self.controller.allPatchesSelected
        self.refreshColors()

    def refreshColors(self):
        if self.controller.allPatchesSelected:
            color = ColorHigh
        else:
            color = ColorUnHigh
        self.patchAllButton.config(background=color)
        self.patchGlobalLabel.config(background=color)
        for patch in self.controller.Patches:
            if self.controller.allPatchesSelected or patch != self.controller.CurrPatch:
                self.colorPatch(patch.name(),color)
            else:
                self.colorPatch(patch.name(),ColorHigh)

    def colorPatch(self,patchName,color):
        self.patchFrame[patchName].config(background=color)
        if self.controller.showCursorFeedback:
            self.patchCanvas[patchName].config(background=color)

    def highlightPatchBorder(self,patch,highlighted):
        if highlighted:
            w = 4
        else:
            w = 2
        self.patchFrame[patch].config(borderwidth=w)

    def drawOval(self,patch,highlighted,x,y,z):
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
            color = self.controller.layerColor(patch)
        self.patchCanvas[patch].create_oval(x-z,y-z,x+z,y+z,outline=color)

    def patchCallback(self,e):
        if self.controller.guiLevel==0:
            return
        for pad in self.patchFrame:
            if e.widget == self.patchFrame[pad]:
                self.controller.allPatchesSelected = False
                self.controller.patchChooserCallback(pad)
                self.refreshColors()
                return
        log("No pad found in padCallback!?")

    def refreshPatchColors(self):
        if self.controller.allPatchesSelected:
            color = ColorHigh
        else:
            color = ColorUnHigh
        self.patchAllButton.config(background=color)
        self.patchGlobalLabel.config(background=color)

        for pad in self.controller.Patches:
            if self.controller.allPatchesSelected or pad != self.controller.CurrPatch:
                self.colorPatch(pad.name(),color)
            else:
                self.colorPatch(pad.name(),ColorHigh)

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

        self.textPrefix = {}
        self.textPrefix["quad"] = ttk.Label(self.titleFrame, text="/", style='PageSep.TLabel',background=ColorBg)
        self.textPrefix["patch"] = ttk.Label(self.titleFrame, text="/", style='PageSep.TLabel',background=ColorBg)
        self.textPrefix["misc"] = ttk.Label(self.titleFrame, text="=", style='PageSep.TLabel',background=ColorBg)
        self.textPrefix["sound"] = ttk.Label(self.titleFrame, text="+", style='PageSep.TLabel',background=ColorBg)
        self.textPrefix["visual"] = ttk.Label(self.titleFrame, text="+", style='PageSep.TLabel',background=ColorBg)
        self.textPrefix["effect"] = ttk.Label(self.titleFrame, text="+", style='PageSep.TLabel',background=ColorBg)

        self.repack()

    def repack(self):

        if self.controller.guiLevel == 0:
            # clear plaement of everything
            for pg in self.controller.visiblePageNames:
                self.pageButton[pg].pack_forget()
            for t in self.textPrefix:
                self.textPrefix[t].pack_forget()
            # guiLevel 0 is just the title
            self.PaletteTitle.config(text="Space Palette Pro",justify=tk.CENTER)
            self.PaletteTitle.pack(side=tk.TOP,pady=0)
        else:
            self.PaletteTitle.pack_forget()
            for pg in self.controller.visiblePageNames:
                padx = 2
                if pg in self.textPrefix:
                    self.textPrefix[pg].pack(side=tk.LEFT,padx=0)
                    padx = 0
                self.pageButton[pg].pack(side=tk.LEFT,padx=padx)
            
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

        self.updateSavedNames()
        self.paramsFrame = self.makeParamsArea(self)
        self.scrollbar = ScrollBar(parent=self, notify=self)

        # On the "quad" editing page, the parameter values aren't shown,
        # just the buttons to import/export/save
        if pagename != "quad" and pagename != "patch":
            self.paramsFrame.pack(side=tk.LEFT, pady=0)
            self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=10, padx=5)
            self.updateParamView()

        defname = self.controller.selectorPage[pagename].defaultVal()
        self.setSavedNameInComboBox(defname)

    def updateSavedNames(self):
        self.savedNames = palette.savedListAll(self.pagename)
        self.comboParamsname.configure(values=self.savedNames)

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
            self.paramLabelWidget[name].bind("<Button-1>", lambda event,nm=name: self.nameClicked(nm))
            # self.paramEnabled[name] = True

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

        if self.pagename != "quad" and self.pagename != "patch":
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

        b = ttk.Label(f, text="Save As", style='RandEtcButton.TLabel')
        b.bind("<Button-1>", lambda event:self.saveCallback())
        b.pack(side=tk.LEFT, pady=5, padx=2)

        # The following things don't get placed initially,
        # they're revealed when the Save button is pressed.

        self.comboParamsname = ttk.Combobox(f, textvariable=self.paramsnameVar,
                font=comboFont, style='custom.TCombobox')
        self.comboParamsname.bind("<<ComboboxSelected>>", lambda event,v=self.paramsnameVar : self.checkThenGotoParamsFile(v.get()))
        self.comboParamsname.bind("<Return>", lambda event,v=self.paramsnameVar : self.checkThenGotoParamsFileReturn(v.get()))

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

    def nameClicked(self,name):
        log("nameClicked! name=",name)
        # self.paramEnabled[name] = not self.paramEnabled[name]
        # if self.paramEnabled[name]:
        #     self.paramLabelWidget[name].config(background=ColorBg)
        # else:
        #     self.paramLabelWidget[name].config(background=ColorRed)

    def widg_cget(self,widg, name):
        cg = widg.cget(name)
        if isinstance(cg,tuple):
            return cg[0]
        return cg

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
            txt = self.widg_cget(widg,"text")
            if txt == "":
                txt = self.params[name]["init"]
            v = int(txt)
            vrange = int(mx) - int(mn)
            if amount == -3:
                delta = -int(vrange/10)
            if amount == -2:
                delta = -int(vrange/100)
                if delta == 0:
                    delta = -1
            if amount == -1:
                delta = -1
            if amount == 1:
                delta = 1
            if amount == 2:
                delta = int(vrange/100)
                if delta == 0:
                    delta = 1
            if amount == 3:
                delta = int(vrange/10)

            newval = v + delta
        elif t == "double" or t == "float":
            cg = self.widg_cget(widg,"text")
            v = float(cg)
            vrange = float(mx) - float(mn)
            if amount == -3:
                v = v - (vrange/10)
            if amount == -2:
                v = v - (vrange/100)
            if amount == -1:
                v = v - (vrange/1000)
            if amount == 1:
                v = v + (vrange/1000)
            if amount == 2:
                v = v + (vrange/100)
            if amount == 3:
                v = v + (vrange/10)
            # log("amount=",amount," mx=",mx," v=",v)
            newval = v
        elif t == "string":
            widgtext = self.widg_cget(widg,"text")
            # Not sure why cget returns different things,
            # sometimes tuple, sometimes string
            if type(widgtext) == type("string"):
                v = widgtext
            else:
                v = str(widgtext[0])
            try:
                vals = self.controller.paramenums[self.params[name]["min"]]
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

        # log("adjustValue ValueWidget name=",name," value=",newval)

        paramType = self.controller.paramTypeOf[name]
        self.controller.changeAndSendValue(paramType,name,newval)

    def listOfType(self,typesname):
        return self.controller.paramenums[typesname]

    def getValue(self,name):
        t = self.controller.paramValueTypeOf[name]
        widg = self.paramValueWidget[name]
        s = self.widg_cget(widg,"text")
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
        log("In checkThenGoToParamsFile for name=",name)
        return

    def checkThenGotoParamsFileReturn(self, name):
        log("In checkThenGoToParamsFileReturn for name=",name)
        return

    def setSavedNameInComboBox(self,name):
        self.paramsname = name
        try:
            n = self.savedNames.index(name)
            self.comboParamsname.current(n)
        except:
            pass

#     def loadOtherSaved(self,name):
# 
#         path = palette.searchSavedFilePath(self.pagename,name)
#         try:
#             f = open(path)
#         except:
#             log("Unable to load saved: ",path)
#             return
# 
#         j = json.load(f)
#         savedvals = j["params"]
# 
#         self.controller.sendParamValues(savedvals)
# 
#     def loadSnapNamed(self,name,doLift=True):
# 
#         log("\n=== loadSnapNamed ",name)
# 
#         self.controller.readSnapParamsFileIntoPage(name,"patch")
# 
#         self.comboParamsname.configure(values=self.paramFiles)
# 
#         self.setSavedNameInComboBox(name)
# 
#         for p in self.params:
#             self.changeValue(p,self.getValue(p))
# 
#         if doLift:
#             self.lift()
# 
#     def oldstartEditing(self,name,doLift=True):
# 
#         log("=== startEditing pagename=%s name=%s" % (self.pagename,name))
#         if self.pagename == "quad":
#             log("Are you getting here?")
#             # self.controller.readPresetSaved(name)
#         else:
#             self.controller.readSnapParamsFileIntoPage(name,self.pagename)
# 
#         self.comboParamsname.configure(values=self.paramFiles)
# 
#         self.setSavedNameInComboBox(name)
# 
#         # self.oldStartEditing()

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
        self.saveSaved(name)

        self.updateSavedNames()
        self.controller.updateSelectorPage(self.pagename,self.savedNames)
        self.saveCancelCallback()

    def saveSaved(self,filename):

        if self.pagename == "quad":
            result, err = palette.palette_quadpro_api("save",
                    "\"filename\": \"" + filename + "\"")
            if err != None:
                log("Error saving saved:",filename," err=",err)

        elif self.pagename == "engine":
            result, err = palette.palette_engine_api("save",
                    "\"filename\": \"" + filename + "\"")
            if err != None:
                log("Error saving saved:",filename," err=",err)


        else:
            # Patch-specific pages
            if self.controller.allPatchesSelected:
                msg = "\n   You can't save a "+self.pagename+" when more than one patch is selected.   \n\nPlease select the patch you want to save.\n"
                self.controller.popup(msg)
                return

            patch = self.controller.CurrPatch.name()
            result, err = palette.palette_patch_api(patch,"save",
                    "\"category\": \"" + self.pagename + "\""
                    ", \"filename\": \"" + filename + "\"")
            if err != None:
                log("Error saving saved:",filename," err=",err)

    def jsonParamDump(self,section):
        newjson = {}
        newjson["params"] = {}
        if section == "patch":
            for name in self.params:
                newjson["params"][name] = {}
                w = self.paramValueWidget[name]
                newjson["params"][name] = self.normalizeJsonValue(name,w.cget("text"))
        else:
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
        self.buttonNames = []

        self.makePerformButton("COMPLETE_RESET", self.controller.resetAll)
        self.makePerformButton("HELP_ ", self.controller.startHelp)
        self.makePerformButton("Looping", self.controller.loopingOnOff)
        # self.makePerformButton("Looping_OFF", self.controller.loopingOff)
        self.makePerformButton("LOOPING_CLEAR", self.controller.loopingClear)
        # if self.controller.defaultGuiLevel > 0:
        #     self.makePerformButton("Clear_ ", self.controller.clear)
        # These shouldn't be shown in casual mode
        # self.makePerformButton("Start_All", self.controller.startAll)
        # self.makePerformButton("Stop_All", self.controller.stopAll)
        # self.makePerformButton("Exit", self.controller.exit)
        # self.makePerformButton("Start_Recording", self.controller.startRecording)
        # self.makePerformButton("Stop_Recording", self.controller.stopRecording)

    def button_cget(self,button,name):
        text = button.cget(name)
        if isinstance(text,tuple):
            return text[0]
        return text

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
                text = self.button_cget(button,"text")

            if isTwoLine(text):
                text = text.replace(palette.LineSep,"\n",1)

            ipady = 0
            button.config(text=text)
            # log("setting perform button to text=",text)

            guiLevel = self.controller.guiLevel
            style = 'PerformButton.TLabel'
            button.config(text=text, width=11, style=style)
            button.grid(row=row,column=col, padx=self.controller.performButtonPadx,pady=self.controller.performButtonPady,ipady=ipady)
            col += 1
            if col >= self.controller.performButtonsPerRow:
                col = 0
                row += 1

    def changeButton(self,name,text,highlight):
        if not name in self.performButton:
            log("changeButton name=",name," not in performButton")
            return
        if highlight:
            style = 'PerformButtonHighlight.TLabel'
        else:
            style = 'PerformButton.TLabel'
        self.setPerformButtonText(name,text,style)
        self.performButton[name].config(style=style,text=text)
        
    def makePerformButton(self,name,f=None,text=None):
        if f == None:
            cmd = lambda nm=name: self.performCallback(nm)
        else:
            cmd = f
        self.performButton[name] = ttk.Button(self.frame, width=10, command=cmd)
        self.setPerformButtonText(name,text,'PerformButton.TLabel')
        self.buttonNames.append(name)

    def setPerformButtonText(self,name,text,style):
        if text == None:
            text = name
        if isTwoLine(text):
            text = text.replace(palette.LineSep,"\n",1)
        self.performButton[name].config(text=text, style=style)

    def performCallback(self,name):

        controller = self.controller

        log("Perform Button Pressed",name)

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

        if self.controller.guiLevel == 0:
            nrows = self.controller.selectDisplayRowsNormal
        else:
            if self.pagename == "quad":
                nrows = self.controller.selectDisplayRowsAdvanced - 4
            else:
                nrows = self.controller.selectDisplayRowsAdvanced
        nbuttons = self.controller.selectDisplayPerRow * nrows
        nvals = len(self.vals)
        if nvals <= nbuttons or self.controller.guiLevel == 0:
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

        self.controller.setFrameSizes()  # hack

        # log("doLayout page=",self.pagename," nbuttons=",nbuttons, "nvals=",nvals, "nrows=",nrows,"buttonwidth=",buttonwidth)

        self.valsframe.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, pady=5)

        for i in self.selectButtons:
            self.selectButtons[i].grid_forget()

        i = 0
        for r in range(0,self.controller.selectDisplayRows):
            for c in range(0,self.controller.selectDisplayPerRow):
                if valindex < len(self.vals):

                    selectButtonText = self.vals[valindex]
                    istwo = isTwoLine(selectButtonText)
                    if istwo:
                        style='SelectButton.TLabel'
                        selectButtonText = selectButtonText.replace(palette.LineSep,"\n",1)
                        selectButtonText = selectButtonText.replace(palette.LineSep," ")
                    else:
                        style='SelectButton.TLabel'
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
                s = 'SelectButtonHighlight.TLabel'
            else:
                s = 'SelectButton.TLabel'
            self.selectButtons[i].config(style=s)

def afterWindowIsDisplayed(windowName,guisize,*args):

    time.sleep(1.0) # wait for window to be visible so nircmdc sees it
    guimaximize = "false"
    # guisize is palette/small/medium
    if guisize == "palette":
        guirect = "-800 0 800 1280"
        guimaximize = "true"
    elif guisize == "small":
        guirect = "0 0 400 640"
    elif guisize == "medium":
        guirect = "0 0 500 800"
    else:
        log("BAD VALUE FOR guisize=",guisize," assuming small")
        guirect = "0 0 400 640"

    cmd = "nircmdc.exe win setsize stitle \""+windowName+"\" "+guirect
    log("Resizing GUI, guisize=",guisize," cmd=",cmd)
    os.system(cmd)

    if guimaximize == "true":
        # remove the title bar and maximize it
        cmd = "nircmdc.exe win -style stitle \""+windowName+"\" 0x00CA0000"
        log("Maximizing gui cmd 1 =",cmd)
        os.system(cmd)
        cmd = "nircmdc.exe win max stitle \""+windowName+"\""
        log("Maximizing gui cmd 2 =",cmd)
        os.system(cmd)

    global PaletteApp
    PaletteApp.nextMode = "layout"

def isVisibleParameter(name):
    parts = name.split(".")
    if len(parts) > 0 and parts[-1].startswith("_") :
        return False
    return True

def patchOfParam(paramname):
    pad = paramname[0]
    # This code assumes that all real parameter names are lower-case
    if pad == "A" or pad == "B" or pad == "C" or pad == "D":
        baseparam = paramname[2:]
        return (pad,baseparam)
    else:
        return (None,paramname)

def PatchParamName(patch,param):
    return patch + "-" + param

def isTwoLine(text):
    return text.find(palette.LineSep) >= 0 or text.find("\n") >= 0

def initMain(app):
    app.iconbitmap(palette.configFilePath("palette.ico"))
    app.protocol("WM_DELETE_WINDOW", on_closing)
    app.mainLoop()

def setFontSizes(guisize):
    global selectButtonFont, largestFont, smallFont
    global comboFont, largerFont, largeFont, performButtonFont
    global paramNameFont, paramValueFont, paramAdjustFont

    f = 'Open Sans Regular'

    if guisize == "palette":
        selectButtonFont = (f, int(20))
        largestFont = (f, int(24))
        comboFont = (f, int(20))
        largerFont = (f, int(20))
        largeFont = (f, int(16))
        smallFont = (f, int(12))
        performButtonFont = (f, int(14))
        paramNameFont = (f, int(18))
        paramValueFont = (f, int(18))
        paramAdjustFont = (f, int(20))
    elif guisize == "medium":
        selectButtonFont = (f, int(12))
        comboFont = (f, int(10))
        largestFont = (f, int(16))
        largerFont = (f, int(12))
        largeFont = (f, int(10))
        smallFont = (f, int(6))
        performButtonFont = (f, int(10))
        paramNameFont = (f, int(12))
        paramValueFont = (f, int(12))
        paramAdjustFont = (f, int(10))
    else:
        if guisize != "small":
            log("Unknown guisize=",guisize)
        selectButtonFont = (f, int(10))

        largestFont = (f, int(12))
        largerFont = (f, int(10))
        largeFont = (f, int(8))
        smallFont = (f, int(6))
        comboFont = (f, int(10))
        performButtonFont = (f, int(10))
        paramNameFont = (f, int(8))
        paramValueFont = (f, int(8))
        paramAdjustFont = (f, int(6))

def makeStyles(app):
    app.option_add('*TCombobox*Listbox.font', comboFont)

    s = ttk.Style()

    s.configure('.', font=largeFont, background=ColorBg, foreground=ColorText)

    s.configure('PageButtonEnabled.TLabel', background=ColorHigh, relief="flat", justify=tk.CENTER, font=largestFont)
    s.configure('PageButtonDisabled.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, font=largestFont)
    s.configure('PageSep.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, font=largerFont)

    s.configure('RandEtcButton.TLabel', font=largerFont, foreground=ColorText, background=ColorButton)
    s.configure('RandEtcButtonHigh.TLabel', font=largerFont, foreground=ColorText, background=ColorHigh)

    s.configure('ParamName.TLabel', font=paramNameFont, foreground=ColorText, justify=tk.LEFT)
    s.configure('ParamValue.TLabel', font=paramValueFont, foreground=ColorText, borderwidth=2, justify=tk.RIGHT, background=ColorBg)
    s.configure('ParamAdjust.TLabel', foreground=ColorText, borderwidth=2, anchor=tk.CENTER, background=ColorButton, font=paramAdjustFont)

    s.configure('GlobalButton.TLabel', font=largestFont, background=ColorButton, relief="flat", justify=tk.CENTER)

    s.configure('Loading.TLabel', background=ColorButton, foreground=ColorWhite, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largestFont)
    s.configure('Attract.TLabel', background=ColorBg, foreground=ColorWhite, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largestFont)

    s.configure('SelectButton.TLabel', foreground=ColorText, font=selectButtonFont, background=ColorButton, anchor=tk.CENTER, justify=tk.CENTER)
    s.configure('SelectButtonHighlight.TLabel', foreground=ColorText, font=selectButtonFont, background=ColorHigh, anchor=tk.CENTER, justify=tk.CENTER)

    s.configure('RecordingButton.TLabel', background=ColorRed, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largeFont)

    s.configure('PerformButton.TLabel', foreground=ColorText, background=ColorButton, relief="flat", justify=tk.CENTER,
        anchor=tk.CENTER, font=performButtonFont)
    s.configure('PerformButtonHighlight.TLabel', foreground=ColorText, background=ColorHigh, relief="flat", justify=tk.CENTER,
        anchor=tk.CENTER, font=performButtonFont)

    s.configure('custom.TCombobox', foreground=ColorComboText, background=ColorBg)

    s.map('SelectButton.TLabel',
        foreground=[('disabled', 'yellow'),
                    ('pressed', ColorText),
                    ('active', ColorText)],
        background=[('disabled', 'yellow'),
                    ('pressed', ColorHigh),
                    ('active', ColorButton)]
        )

    # s.map('PerformButton.TLabel', foreground=ColorText, font=selectButtonFont, background=ColorHigh, anchor=tk.CENTER, justify=tk.CENTER)
    # s.map('PerformButton.TLabel',
    #     foreground=[('disabled', 'yellow'),
    #                 ('pressed', ColorText),
    #                 ('active', ColorText)],
    #    background=[('disabled', 'yellow'),
    #                ('pressed', ColorHigh),
    #                ('active', ColorButton)]
    #    )

    # s.map('PerformButtonHighlight.TLabel',
    #     foreground=[('disabled', 'yellow'),
    #                 ('pressed', ColorHigh),
    #                 ('active', ColorHigh)],
    #     background=[('disabled', 'yellow'),
    #                 ('pressed', ColorHigh),
    #                 ('active', ColorHigh)]
    #     )

def log(*args):
    final = args[0]
    if len(args) > 1:
        for s in args[1:]:
            final += " " + str(s)
    palette.log(final)

def status_thread(app):  # runs in background thread

    while True:

        time.sleep(3.0)

        status, err = palette.palette_engine_api("status","")
        if err != None:
            log("engine.status: err=",err)
            continue

        if status == None:
            log("Hey, output of status is None?")
            continue

        # log("status is ",status)
        jstatus = json.loads(status)
        attractMode = jstatus["attractmode"]
        if attractMode == "true":
            if PaletteApp.currentMode != "attract":
                log("Turning Attract Mode On!")
                PaletteApp.nextMode = "attract"
        else:
            if PaletteApp.currentMode != "normal" and PaletteApp.currentMode != "help":
                log("Turning Attract Mode Off!")
                PaletteApp.nextMode = "normal"

if __name__ == "__main__":

    log("GUI started")

    # Default is all four patches
    patches = os.environ.get("PALETTE_PATCHES","ABCD")
    npatches = len(patches)
    if npatches == 1:
        # You can set patchs to "B", for example
        patchname = patches[0]
        patchnames = patches
    elif npatches == 4:
        patchname = patches[0]
        patchnames = patches
        FourPatches = True
    else:
        log("Unexpected number of patches: ",patches)

    visiblepagenames = {
        "engine":"Engine",
        "quad":"Quad",
        "patch":"Patch",
        "misc":"Misc",
        "sound":"Sound",
        "visual":"Visual",
        "effect":"Effect",
    }

    # guisize is palette/medium/small
    guisize = os.environ.get("PALETTE_GUI_SIZE","small")

    palette.palette_api_setup()

    global PaletteApp
    PaletteApp = ProGuiApp(patchname,patchnames,visiblepagenames,guisize)

    makeStyles(PaletteApp)

    log("guisize=",guisize)
    if guisize == "palette":
        # Fixed size - the guisize is really only used to reposition it.
        # Should check to see whether the size matches the 800x1280 expectation
        PaletteAppSize = {"width":800,"height":1280}
    elif guisize == "small":
        PaletteAppSize = {"width":400,"height":640}
    elif guisize == "medium":
        PaletteAppSize = {"width":500,"height":800}
    else:
        log("BAD VALUE FOR guisize=",guisize)
        PaletteAppSize = {"width":400,"height":640}

    PaletteApp.wm_geometry("%dx%d" % (PaletteAppSize["width"],PaletteAppSize["height"]))

    log("PALETTEAPP size = ",PaletteAppSize)
    PaletteApp.nextMode = ""
    PaletteApp.currentMode = ""

    threading.Timer(0.0, afterWindowIsDisplayed, args=[PaletteApp.windowName,guisize], kwargs=None).start()

    statusThread = threading.Thread(target=status_thread,args=(PaletteApp,))   # timer thread
    statusThread.daemon = True
    statusThread.start()  # start timer loop

    initMain(PaletteApp)
