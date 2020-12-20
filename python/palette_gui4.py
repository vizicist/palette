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

RecMode = False
StartupMode = True

# ColorBg = '#bbbbbb'
ColorWhite = '#ffffff'
ColorBlack = '#000000'

ColorBg = '#000000'
ColorText = '#ffffff'
ColorComboText = '#000000'    # black
# ColorButton = '#888888'  
ColorButton = '#333333'  
ColorScrollbar = '#333333'  
ColorThumb = '#00ffff'  

ColorRed = '#ff0000'
ColorBlue = '#0000ff'
ColorGreen = '#00ff00'
# ColorHigh = '#006666'
ColorHigh = '#006666'
ColorBright = '#00ffff'
ColorAqua = '#00ffff'
ColorUnHigh = '#888888'

LineSep = "_"

# resetAfterInactivity = 90.0
resetAfterInactivity = -1

global patchFont, patchTwoLineFont, sliderFont, largestFont, hugeFont, comboFont, largerFont, largeFont, performFont, mediumFont, padLabelFont
global paramDisplayRows, selectDisplayRows, selectDisplayPerRow
global pageSizeOfControlNormal, pageSizeOfSelectNormal, pageSizeOfControlAdvanced, pageSizeOfSelectAdvanced
global pageSizeOfControl, pageSizeOfSelect
global performButtonPadx, performButtonPady
global selectButtonPadx, selectButtonPady

def palette_region_api(region, meth, params=""):
    if region == "":
        print("palette_region_api: no region specified?  Assuming A")
        region = "A"
    if region == "*":
        print("palette_region_api: What do I do with a * region?")
        region = "A"
    p = "\"region\":\""+region+"\""
    if params == "":
        params = p
    else:
        params = p + "," + params
    return palette.palette_api("region."+meth,params)

def palette_global_api(meth, params=""):
    return palette.palette_api("global."+meth,params)

def setFontSizes(fontFactor):
    global patchFont, patchTwoLineFont, sliderFont, largestFont, hugeFont, comboFont, largerFont, largeFont, performFont, mediumFont, padLabelFont
    f = 'Helvetica'
    f = 'Lucida Sans'
    patchFont = (f, int(20*fontFactor))
    patchTwoLineFont = (f, int(18*fontFactor))
    sliderFont = (f, int(12*fontFactor))
    largestFont = (f, int(24*fontFactor))
    hugeFont = (f, int(36*fontFactor))
    comboFont = (f, int(20*fontFactor))
    largerFont = (f, int(20*fontFactor))
    largeFont = (f, int(16*fontFactor))
    performFont = (f, int(18*fontFactor))
    mediumFont = (f, int(12*fontFactor))
    padLabelFont = (f, int(16*fontFactor))

OneBeat = 96

PageNames = {
    "snap":"Snapshot",
    "sound":"Sound",
    "visual":"Visual",
    "effect":"Effect",
    "sliders":"Sliders",
}
ControlPageNames = {
    "main":"Main",
    # "sliders1":"Slider1",
    # "sliders2":"Slider2",
    # "sliders3":"Slider3",
}

PerPadPerformLabels = {}
GlobalPerformLabels = {}
PerPadPerformLabels["loopinglength"] = [
    {"label":"Loop Length_8 beats",  "value":8*OneBeat},
    {"label":"Loop Length_16 beats", "value":16*OneBeat},
    {"label":"Loop Length_32 beats", "value":32*OneBeat},
    {"label":"Loop Length_64 beats", "value":64*OneBeat},
    {"label":"Loop Length_4 beats", "value":4*OneBeat},
]
SimpleScales = [
	{"label":"Newage_Scale",    "value":"newage"},
	{"label":"Arabian_Scale",   "value":"arabian"},
	{"label":"Chromatic_Scale", "value":"chromatic"},
    {"label":"Dorian_Scale","value":"dorian"},
	{"label":"Fifths_Scale",    "value":"fifths"},
    {"label":"Harminor_Scale",  "value":"harminor"},
    {"label":"Lydian_Scale","value":"lydian"},
    {"label":"Melminor_Scale",  "value":"melminor"},
]
PerformScales = [
	{"label":"Newage_Scale",    "value":"newage"},
    {"label":"Aeolian_Scale",   "value":"aeolian"},
 	{"label":"Arabian_Scale",   "value":"arabian"},
 	{"label":"Chromatic_Scale", "value":"chromatic"},
    {"label":"Dorian_Scale","value":"dorian"},
 	{"label":"Fifths_Scale",    "value":"fifths"},
    {"label":"Harminor_Scale",  "value":"harminor"},
    {"label":"Ionian_Scale","value":"ionian"},
    {"label":"Locrian_Scale",   "value":"locrian"},
    {"label":"Lydian_Scale","value":"lydian"},
    {"label":"Melminor_Scale",  "value":"melminor"},
    {"label":"Mixolydian_Scale","value":"mixolydian"},
    {"label":"Phrygian_Scale",  "value":"phrygian"},
    {"label":"Raga1_Scale",     "value":"raga1"},
    {"label":"Raga2_Scale", "value":"raga2"},
    {"label":"Raga3_Scale", "value":"raga3"},
    {"label":"Raga4_Scale", "value":"raga4"},
]

PerPadPerformLabels["quant"] = [
    {"label":"Fret_Quantize", "value":"frets"},
    {"label":"Pressure_Quantize", "value":"pressure"},
    {"label":"Fixed_Time Quant", "value":"fixed"},
    {"label":"No_Quant",  "value":"none"},
]
PerPadPerformLabels["vol"] = [
    {"label":"Pressure_Vol", "value":"pressure"},
    {"label":"Fixed_Vol", "value":"fixed"},
]
PerPadPerformLabels["loopingfade"] = [
    {"label":"Loop Fade_Med",  "value":0.4},
    {"label":"Loop Fade_Slow", "value":0.5},
    {"label":"Loop Fade_Slower", "value":0.6},
    {"label":"Loop Fade_Slowest", "value":0.7},
    {"label":"Loop_Forever", "value":1.0},
    {"label":"Loop Fade_Fast", "value":0.2},
    {"label":"Loop Fade_Faster", "value":0.1},
    {"label":"Loop Fade_Fastest", "value":0.05},
]
PerPadPerformLabels["loopingonoff"] = [
    {"label":"Looping_is OFF",  "value":"off"},
    {"label":"Looping_REC+PLAY", "value":"recplay"},
    {"label":"Looping_PLAY ONLY", "value":"play"},
]
PerPadPerformLabels["midithru"] = [
    {"label":"MIDI Input_Thru",  "value":"thru"},
    {"label":"MIDI Input_Set Scale",  "value":"setscale"},
    {"label":"MIDI Input_Thru Scadjust", "value":"thruscadjust"},
    {"label":"MIDI Input_Disabled",  "value":"disabled"},
]
PerPadPerformLabels["useexternalscale"] = [
    {"label":"External Scale_Off",  "value":False},
    {"label":"External Scale_On",  "value":True},
]
PerPadPerformLabels["midiquantized"] = [
    {"label":"MIDI Thru_Unquantized",  "value":False},
    {"label":"MIDI Thru_Quantized",  "value":True},
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

global TopContainer
TopContainer = None
global TopApp
TopApp = None

class ProGuiApp(tk.Tk):
    def __init__(self):
        tk.Tk.__init__(self)

        self.PadNames = collections.OrderedDict({ "A":1, "B":2, "C":3, "D":4 })
        self.PadName = "A"

        self.readParamDefs()
        self.frames = {}
        self.editPage = {}
        self.performPage = {}
        self.selectorPage = {}
        self.currentPageName = None
        self.codenamindex = 0
        self.selectorAction = ""
        self.selectorButtonIndex = 0
        self.selectorValue = ""
        self.activeCursors = {}
        self.activeTime = {}
        self.editMode = False
        self.showAllPages = False
        self.showSliders = True
        self.showSound = False
        self.showPadFeedback = True
        self.showCursorFeedback = False
        self.setAdvanced(0)  # default to non-advanced mode

        self.performHeader = None

        self.perpadPerformVal = {}
        self.globalPerformVal = {}
        for s in PerPadPerformLabels:
            self.perpadPerformVal[s] = {}
        for s in GlobalPerformLabels:
            self.globalPerformVal[s] = {}

        self.topContainer = tk.Frame(self, background=ColorBg)
        global TopContainer
        TopContainer = self.topContainer
        global TopApp
        TopApp = self

        self.selectFrame = self.makeSelectFrame(self.topContainer)
        self.performContainer = tk.Frame(self.topContainer,
            highlightbackground=ColorAqua, highlightcolor=ColorAqua, highlightthickness=3)
        self.performHeader = PerformHeader(parent=self.performContainer, controller=self)
        self.startupFrame = self.makeStartupFrame(self.topContainer)

        # These are the pages for performance things
        self.performPage["main"] = PagePerformMain(parent=self.performContainer, controller=self)
        # self.performPage["sliders1"] = PagePerformSliders(parent=self.performContainer, controller=self, slidersNum=1)
        # self.performPage["sliders2"] = PagePerformSliders(parent=self.performContainer, controller=self, slidersNum=2)
        # self.performPage["sliders3"] = PagePerformSliders(parent=self.performContainer, controller=self, slidersNum=3)

        self.winfo_toplevel().title("Palette")

        self.escapeCount = 0
        self.lastEscape = time.time()
        self.resetLastAnything()

        self.topContainer.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.performHeader.pack(side=tk.TOP,fill=tk.X)
        self.setPerformMessage("")
        self.selectPerformPage("main")
        self.selectEditPage("snap")
        self.resetVisibility()

        self.topContainer.bind_all("<MouseWheel>", self.scrollWheel)

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
                    try:
                        self.loadSnap("CurrentSnapshot")
                        self.sendSnap()
                    except:
                        self.loadSnap("Basic_Chaos")
                        self.sendSnap()

            if resetAfterInactivity>0 and (now - self.lastAnything) > resetAfterInactivity:
                print("Resetting after no activity!!")
                self.resetLastAnything()
                self.setAdvanced(0)
                self.resetAll()

                self.resetVisibility()
                self.selectPerformPage("main")
                self.performPage["main"].updatePerformButtonLabels(self.PadName)
    
            reset = True

            if self.selectorAction == "LOAD":
                self.selectorLoadAndSend(self.currentPageName,self.selectorValue,self.selectorButtonIndex)
    
            elif self.selectorAction == "IMPORT":
                self.selectorImportAndSend(self.currentPageName,self.selectorValue)

            elif self.selectorAction == "INIT":
                self.selectorApply("init",self.currentPageName,self.selectorValue)

            elif self.selectorAction == "RAND":
                self.selectorApply("rand",self.currentPageName,self.selectorValue)

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
            self.resetVisibility()

    def resetVisibility(self):
        global RecMode
        sh = self.selectHeader
        ch = self.performHeader
        if self.showAllPages:
            for pg in PageNames:
                if not self.showSliders and pg == "sliders":
                    sh.pageButton[pg].pack_forget()
                elif not self.showSound and pg == "sound":
                    sh.pageButton[pg].pack_forget()
                else:
                    sh.pageButton[pg].pack(side=tk.LEFT,padx=5)
            for pg in ControlPageNames:
                # no control page names get displayed if we're not showing slider pages
                if not self.showSliders or not self.showSound:
                    ch.pageButton[pg].pack_forget()
                else:
                    ch.pageButton[pg].pack(side=tk.LEFT,padx=5)
            ch.performMessageLabel.pack_forget()

        elif RecMode:
            for pg in PageNames:
                sh.pageButton[pg].pack_forget()
            for pg in ControlPageNames:
                ch.pageButton[pg].pack_forget()
            ch.performMessageLabel.pack(side=tk.LEFT, fill=tk.X, expand=True, padx=50)
        else:
            for pg in PageNames:
                sh.pageButton[pg].pack_forget()
            for pg in ControlPageNames:
                ch.pageButton[pg].pack_forget()
            ch.performMessageLabel.pack_forget()

        self.editMode = False
        self.selectEditPage("snap")

        pg = self.performPage["main"]

        global pageSizeOfSelect, pageSizeOfControl
        if self.advancedLevel == 0:
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

    def paramIsPerPad(self,name):
        if name[0:6] == "slider":
            return False
        else:
            return True

    def readParamsFileIntoSnap(self,paramstype,paramsname):
        # This is only called from non-snap pages
        if self.currentPageName == "snap" or paramstype == "snap":
            print("Unexpected snap value in readParamsFileIntoSnap")
            return
        # Read parameters from a json file (but NOT a snap file)

        fpath = palette.presetsFilePath(paramstype, paramsname)

        print("Reading JSON:",fpath)
        f = open(fpath)
        j = json.load(f)
        self.readParamsJsonIntoSnap(paramstype,paramsname,j)
        f.close()

    def readParamsJsonIntoSnap(self,paramstype,paramsname,j):
        paramvals = j["params"]
        if paramstype == "snap":
            print("Unexpected value in readParamsFileIntoSnap")
            return

        # The params in the file may not include all of the
        # parameters for the given paramstype, so we loop through
        # all the parameters of a given type
        snappage = self.editPage["snap"]
        editpage = self.editPage[paramstype]
        for name in self.allParamsJson:
            allj = self.allParamsJson[name]
            if allj["paramstype"] != paramstype:
                continue
            (_,base) = padOfParam(name)
            if base in paramvals:
                v = paramvals[base]
                if "value" in v:
                    # Old versions of the param files used nested structure with "enabled" and "value"
                    v = v["value"]
            else:
                v = allj["init"]
            editpage.changeValueLabel(base,v)

            self.changePadParamValue(self.PadName,paramstype,name,v)

            if self.showSliders and paramstype == "sliders" and self.currentPerformPageName[0:7]=="sliders":
                i = sliderIndexOfParam(name)
                if i != None:
                    suffix = name[7:]
                    if suffix == "param":
                        self.performPage[self.currentPerformPageName].sliderNameChanged(v,i)

        snappage.setChanged()
        snappage.saveJsonInPath(CurrentSnapshotPath())

    def changePadParamValue(self,pad,paramstype,paramname,v):
        snappage = self.editPage["snap"]
        # if self.paramIsPerPad(paramname):
        #     snapParamName = pad + "_" + paramname
        # else:
        #     snapParamName = paramname
        snapParamName = paramname

        if paramstype != "sliders":
            snappage.changeValueLabel(snapParamName,v)


    def readSnapParamsFile(self,paramsname):
        # print("\nREAD SNAP PARAMS FILE paramsname=",paramsname)
        # Read parameters from a json file
        if paramsname == "CurrentSnapshot":
            fpath = palette.localconfigFilePath(paramsname+".json")
        else:
            fpath = palette.presetsFilePath("snap", paramsname)
        print("Reading ",fpath)
        try:
            f = open(fpath)
        except:
            print("No such file?  fpath=",fpath)
            return
        j = json.load(f)
        self.loadSnapJson(j)
        f.close()

    def loadSnapJson(self,j):
        snappage = self.editPage["snap"]
        for name in self.allParamsJson:
            allj = self.allParamsJson[name]
            (_,base) = padOfParam(name)
            paramsType = allj["paramstype"]
            if paramsType != "sliders":
                # fullname = self.PadName + "_" + base
                fullname = base
                if not fullname in j["params"]:
                    j["params"][fullname] = allj["init"]

        for name in j["params"]:
            v = j["params"][name]
            snappage.changeValueLabel(name,v)

    def makeSelectFrame(self,container):

         # This is the area at the very top
        f = tk.Frame(container,
            highlightbackground=ColorAqua, highlightcolor=ColorAqua, highlightthickness=3)

        self.selectHeader = SelectHeader(parent=f, controller=self)
        self.selectHeader.pack(side=tk.TOP,fill=tk.BOTH)

        # These are the pages of buttons for selecting set/patch/sound/visual/etc..
        global PageNames
        for pagename in PageNames:
            self.makeSelectorPage(f, pagename, PageSelector)

        # These are the pages you can switch between for editing
        for pagename in PageNames:
            self.makeEditPage(f,pagename)

        self.editPage["snap"].canRevert = True

        return f

    def makeStartupFrame(self,container):
        f = tk.Frame(container,
            highlightbackground=ColorBg, highlightcolor=ColorAqua, highlightthickness=3)
        self.startupLabel = ttk.Label(f, text="               Palette is Loading...", style='Header.TLabel',
            foreground=ColorText, background=ColorBg, relief="flat", justify=tk.CENTER, font=largestFont)
        self.startupLabel.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        return f

    def updateSelectorPage(self,pagename,files):
        page = self.selectorPage[pagename]
        page.vals = files
        page.doLayout()
       
    def makeSelectorPage(self,parent,pagename,pagemaker):
        vals = palette.presetsListAll(pagename)

        # XXX - this PREVIOUS stuff actually works,
        # XXX - but doesn't properly highlight the previous selection
        # if pagename == "snap":
        #     vals.append("PREVIOUS")

        page = pagemaker(parent, self, vals, pagename)

        self.selectorPage[pagename] = page
        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

    def makeEditPage(self,parent,pagename):
        page = PageEditParams(parent=parent, controller=self,
                    paramstype=pagename, params=self.paramsOfType[pagename])
        self.editPage[pagename] = page
        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)

    def forgetPages(self,pages):
        for pg in pages:
            pages[pg].pack_forget()

    def togglePageButons(self):
        # if self.advancedLevel == 0:
        #     return
        self.showAllPages = not self.showAllPages
        self.resetVisibility()

    def clickEditPage(self,pagename):

        # A second click on the page header will toggle editMode
        if self.currentPageName == pagename:
            self.editMode = not self.editMode
        self.selectEditPage(pagename)

    def selectEditPage(self,pagename):
        self.currentPageName = pagename
        self.selectHeader.highlightPageButton(pagename)

        self.forgetPages(self.selectorPage)
        self.forgetPages(self.editPage)

        if self.editMode:
            page = self.editPage[pagename]
        else:
            page = self.selectorPage[pagename]

        page.pack(side=tk.TOP,fill=tk.BOTH,expand=True)
        page.tkraise()

        if self.editMode:
            self.editPage[pagename].startEditing("CurrentSnapshot",doLift=False)

    def selectPerformPage(self,pagename):
        self.currentPerformPageName = pagename
        self.performHeader.highlightPageButton(pagename)
        for pg in self.performPage:
            if pg == pagename:
                self.performPage[pg].pack(side=tk.TOP,fill=tk.BOTH,expand=True)
            else:
                self.performPage[pg].pack_forget()

        self.performPage[pagename].tkraise()

    def sliderCallback(self,param,val,modify):
        # print("CONTROLLER sliderCallback param=",param," val=",val," modify=",modify)
        if self.allParamsJson[param]["valuetype"] == "string":
            print("NOT YET IMPLEMENTED!  STRING slider")
            return
        mn = float(self.allParamsJson[param]["min"])
        mx = float(self.allParamsJson[param]["max"])
        v = (mx-mn) * float(val)
        self.sendSliderParamValue(self.PadName,param,v)

    def sendSliderParamValue(self,pad,paramname,val):
        self.sendPadParamValue(pad,paramname,val)

        # snapparam = pad + "_" + paramname
        snapparam = paramname
        self.editPage["snap"].changeValueLabel(snapparam,val)

        for pg in {"sound","visual","effect"}:
            if self.editPage[pg].hasParameter(paramname):
                self.editPage[pg].changeValueLabel(paramname,val)

    def sendPadParamValue(self,pad,paramname,val):
        paramType = self.paramTypeOf[paramname]
        # if paramType == "effect":
        #     self.sendPadOneEffectVal(pad,paramname,val)
        # else:
        palette_region_api(self.PadName,paramType+".set_param",
            "\"param\": \"" + paramname + "\"" + \
            ", \"value\": \"" + str(val) + "\"" )

    def sendParams(self,params,paramstype):
        page = self.editPage[paramstype]
        for origp in params:
            (pad,baseparam) = padOfParam(origp)
            fullparam = origp
            if pad == None:
                pad = self.PadName
            else:
                # fullparam = self.PadName + "_" + baseparam
                fullparam = baseparam
            if not baseparam in self.paramTypeOf:
                print("param ",baseparam," isn't in paramTypeOf?")
                continue
            if self.paramTypeOf[baseparam] == "effect":
                val = page.getValue(fullparam)
                self.sendPadOneEffectVal(pad,fullparam,val)
                continue
            if not self.paramTypeOf[baseparam] in PageNames:
                print("Not sending param=",origp)
                continue
            v = page.getValue(origp)
            if paramstype == "snap":
                if pad:
                    if baseparam in self.paramsOfType[paramstype]:
                        self.sendPadParamValue(pad,baseparam,v)
            else:
                if baseparam in self.paramsOfType[paramstype]:
                    self.sendPadParamValue(self.PadName,origp,v)

    def paramCallback(self,paramname,newval):

        # print("paramCallback! paramname=",paramname," newval=",newval)

        (pad,baseparam) = padOfParam(paramname)

        if self.currentPageName == "snap":
            if pad:
                # Change the value on the other (per-param-type) editing page
                paramstype = self.allParamsJson[baseparam]["paramstype"]
                self.editPage[paramstype].changeValueLabel(baseparam,newval)
                # we still send the changed parameter out to the appropriate pad
                self.sendPadParamValue(pad,baseparam,newval)
        else:
            # change the corresponding value on the snap page
            if self.currentPageName != "sliders":
                # fullparamname = self.PadName + "_" + paramname
                fullparamname = paramname
                self.editPage["snap"].changeValueLabel(fullparamname,newval)
                self.editPage["snap"].setChanged()
                self.sendPadParamValue(self.PadName,paramname,newval)

        if self.showSliders:
            i = sliderIndexOfParam(paramname)
            if i != None:
                self.performPage[self.currentPerformPageName].sliderNameChanged(newval,i)

        # print("SAVING CURRENTSNAPSHOT!!")
        self.editPage["snap"].saveJsonInPath(CurrentSnapshotPath())
        self.editPage["snap"].setChanged()

    def savePrevious(self):
        frompath = CurrentSnapshotPath()
        topath = CurrentSnapshotPreviousPath()
        palette.copyFile(frompath,topath)
        # print("SAVE PREVIOUS Copying ",frompath," to ",topath)

    def restorePrevious(self):
        frompath = CurrentSnapshotPreviousPath()
        topath = CurrentSnapshotPath()
        palette.copyFile(frompath,topath)
        # print("RESTORING PREVIOUS Copying ",frompath," to ",topath)

    def selectorApply(self,apply,paramstype,val):
        if paramstype == "snap" or paramstype == "sliders":
            print("selectorApply not yet implemented on snap or sliders")
        else:
            self.applyToAllParams(apply,paramstype,val)
            self.sendSnapPad(self.PadName,paramstype)

    def applyToAllParams(self,apply,paramstype,val):
        snappage = self.editPage["snap"]
        editpage = self.editPage[paramstype]
        # loop through all the parameters of a given type
        for name in self.allParamsJson:
            j = self.allParamsJson[name]
            if j["paramstype"] != paramstype:
                continue
            valuetype = j["valuetype"]
            (_,base) = padOfParam(name)
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
                editpage.changeValueLabel(base,v)
                self.changePadParamValue(self.PadName,paramstype,name,v)

        snappage.setChanged()
        snappage.saveJsonInPath(CurrentSnapshotPath())

    def selectorLoadAndSend(self,paramstype,val,buttoni):
        if paramstype == "snap":
            if val == "PREVIOUS":
                self.restorePrevious()
                val = "CurrentSnapshot"
                print("Should be highlighting buttoni=",buttoni)
            else:
                self.savePrevious()
                ("Should be highlighting buttoni=",buttoni)
            self.loadSnap(val)
            self.sendSnap()
        else:
            # if we've selected a slider preset
            if self.showSliders and paramstype == "sliders":
                if self.currentPerformPageName[0:7] == "sliders":
                    self.performPage[self.currentPerformPageName].setSliders(val)

            # even for sliders (which aren't in the snap settings),
            # we want to set the values in the editing page
            self.readParamsFileIntoSnap(paramstype,val)

            if paramstype != "sliders":
                self.sendSnapPad(self.PadName,paramstype)

    def selectorImportAndSend(self,paramstype,val):
        j = json.loads(val)
        if "paramsname" in j:
            paramsname = j["paramsname"]
        else:
            paramsname = "NoValue"

        if "paramstype" in j:
            jparamstype = j["paramstype"]
        else:
            jparamstype = "NoValue"

        if jparamstype != paramstype:
            # XXX - this error will be common, need a visible message
            print("Mismatched paramstype in JSON!")
            return

        if paramstype == "snap":
            self.loadSnapJson(j)
            self.sendSnap()
        else:
            self.readParamsJsonIntoSnap(paramstype,paramsname,j)
            if paramstype != "sliders":
                self.sendSnapPad(self.PadName,paramstype)

    def padChooserCallback(self,pad):
        print("Setting PadName to ",pad)
        self.PadName = pad
        self.selectHeader.padChooser.refreshColors()

        if self.editMode:
            self.editPage[self.currentPageName].startEditing("CurrentSnapshot")

        performControl = self.performPage["main"]
        performControl.updatePerformButtonLabels(self.PadName)

    def loadSnap(self,snapname):
        snappage = self.editPage["snap"]
        snappage.startEditing(snapname,doLift=False)
        snappage.saveJsonInPath(CurrentSnapshotPath())
        snappage.saveJsonInPath(CurrentSnapshotBackupPath())
        snappage.lift()
        return True

    def revertToBackup(self):
        frompath = CurrentSnapshotBackupPath()
        topath = CurrentSnapshotPath()
        palette.copyFile(frompath,topath)
        print("Reverting Backup Copying ",frompath," to ",topath)

    def nextValue(self,arr,v):
            found = -1
            for i in range(len(arr)):
                if arr[i]["value"] == v["value"]:
                    found = i
                    break
            found = (found + 1) % len(arr)
            return arr[found]

    def sendPadPerformVal(self,pad,name):
        # print("sendPadPerformVal pad=",pad," name=",name)
        if name == "loopingonoff":
            val = self.perpadPerformVal["loopingonoff"][pad]["value"]
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

            palette_region_api(self.PadName, "loop_recording", '"onoff": "'+str(reconoff)+'"')
            palette_region_api(self.PadName, "loop_playing", '"onoff": "'+str(playonoff)+'"')

        elif name == "loopinglength":
            v = self.perpadPerformVal["loopinglength"][pad]["value"]
            palette_region_api(self.PadName, "loop_length", '"length": "'+str(v)+'"')

        elif name == "loopingfade":
            fade = self.perpadPerformVal["loopingfade"][pad]["value"]
            palette_region_api(self.PadName, "loop_fade", '"fadelength": "'+str(fade)+'"')

        elif name == "quant":
            val = self.perpadPerformVal["quant"][pad]["value"]
            palette_region_api(self.PadName, "set_param",
                "\"param\": \"" + "misc.quant" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")
        elif name == "scale":
            val = self.perpadPerformVal["scale"][pad]["value"]
            palette_region_api(self.PadName, "set_param",
                "\"param\": \"" + "misc.scale" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")

        elif name == "vol":
            val = self.perpadPerformVal["vol"][pad]["value"]
            # NOTE: "voltype" here rather than "vol" - should make consistent someday
            palette_region_api(self.PadName, "set_param",
                "\"param\": \"" + "misc.vol" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")

        elif name == "comb":
            val = 1.0
            palette_region_api(self.PadName, "loop_comb",
                "\"value\": \"" + str(val) + "\"")

        elif name == "midithru":
            thru = self.perpadPerformVal["midithru"][pad]["value"]
            palette_region_api(self.PadName, "midi_thru", "\"thru\": \"" + str(thru) + "\"")

        elif name == "useexternalscale":
            onoff = self.perpadPerformVal["useexternalscale"][pad]["value"]
            palette_region_api(self.PadName, "useexternalscale", "\"onoff\": \"" + str(onoff) + "\"")

        elif name == "midiquantized":
            quantized = self.perpadPerformVal["midiquantized"][pad]["value"]
            palette_region_api(self.PadName, "midi_quantized", "\"quantized\": \"" + str(quantized) + "\"")

        elif name == "transpose":
            val = self.globalPerformVal["transpose"][pad]["value"]
            palette_region_api(self.PadName, "set_transpose", "\"value\": \""+str(val) + "\"")

    def sendGlobalPerformVal(self,name):

        if name == "tempo":
            val = self.globalPerformVal["tempo"]["value"]
            palette_global_api("set_tempo_factor", "\"value\": \""+str(val) + "\"")

        # elif name == "configname":
        #     config = self.globalPerformVal["configname"]["value"]
        #     palette.setConfigName(config)
        #     print("CONFIGNAME setting to ",palette.getConfigName())

    def clearPadLoop(self,pad):
        palette_region_api(self.PadName, "loop_clear", "")

    def combPadLoop(self,pad):
        palette_region_api(self.PadName, "loop_comb", "")

    def combLoop(self):
        self.resetLastAnything()
        self.combPadLoop(self.PadName)

    def clearLoop(self):
        self.resetLastAnything()
        self.clearPadLoop(self.PadName)

    def cycleAdvancedLevel(self):
            # cycle through 0,1,2
            self.setAdvanced((self.advancedLevel + 1) % 3)
            self.resetVisibility()
            self.performPage["main"].updatePerformButtonLabels(self.PadName)

    def setAdvanced(self,level):
            self.advancedLevel = level
            print("setAdvanced, level is ",self.advancedLevel)
            self.escapeCount = 0
            if level == 0:
                self.showAllPages = False
                self.showSliders = False
                PerPadPerformLabels["scale"] = SimpleScales
            elif level == 1:
                self.showAllPages = True
                self.showSliders = False
                PerPadPerformLabels["scale"] = SimpleScales

    def resetAll(self):

        palette_global_api("audio_reset")

        self.resetLastAnything()
        self.sendANO()
        self.clearExternalScale()

        for pad in self.PadNames:
            for name in PerPadPerformLabels:
                self.perpadPerformVal[name][pad] = PerPadPerformLabels[name][0]
                self.sendPadPerformVal(pad,name)

        for name in GlobalPerformLabels:
            self.globalPerformVal[name] = GlobalPerformLabels[name][0]
            self.sendGlobalPerformVal(name)

        self.setPerformMessage("")
        for pad in self.PadNames:
            self.clearPadLoop(pad)

        self.performPage["main"].updatePerformButtonLabels(self.PadName)

    def clearExternalScale(self):
        palette_region_api(self.PadName, "clearexternalscale")

    def sendANO(self):
        palette_region_api(self.PadName, "ANO")

    def sendSnap(self):
        self.sendSnapPad(self.PadName)

    def paramListJson(self,paramstype,pad):
        paramlist = ""
        sep = ""
        for name in self.allParamsJson:
            j = self.allParamsJson[name]
            if j["paramstype"] == paramstype:
                # paramname = pad + "_" + name
                paramname = name
                v = self.editPage["snap"].getValue(paramname)
                paramlist = paramlist + sep + "\"" + name + "\" : \"" + str(v) + "\""
                sep = ", "

        return paramlist

    def sendSnapPad(self,pad,paramstype=None):
        for pt in ["sound","visual","effect"]:
            paramlistjson = self.paramListJson(pt,pad)
            if paramstype == None or paramstype == pt:
                palette_region_api(self.PadName, pt+".set_params", paramlistjson)

        if paramstype == None:
            for name in PerPadPerformLabels:
                self.sendPadPerformVal(pad,name)

    def synthesizeParamsJson(self):

        # rework the contents of paramdefs.json so that
        # each "effect.*" parameter is followed by an copy
        # with "2" appended to the parameter name.

        s = ""
        path = palette.configFilePath("paramdefs.json")
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
        for t in PageNames:
            self.paramsOfType[t] = collections.OrderedDict()

        self.allParamNames = []
        for x in sorted(self.allParamsJson.keys()):
            self.allParamNames.append(x)
            self.allParamsJson[x]["name"] = x
            t = self.allParamsJson[x]["paramstype"]
            if t != "channel" and t != "misc":
                self.paramsOfType[t][x] = self.allParamsJson[x]
                self.paramTypeOf[x] = self.allParamsJson[x]["paramstype"]

        # Create all the parameters for the "snap" settings by
        # duplicating all the parameters for each pad (A,B,C,D).
        for x in self.allParamNames:
            paramType = self.allParamsJson[x]["paramstype"]
            if paramType == "sliders":
                continue
            # We no longer prepend A_, B_, C_, D_
            # padParamName = self.PadName + "_" + x
            padParamName = x
            self.paramValueTypeOf[padParamName] = self.allParamsJson[x]["valuetype"]
            self.paramsOfType["snap"][padParamName] = self.allParamsJson[x]

        for x in self.allParamNames:
            paramType = self.allParamsJson[x]["paramstype"]
            if paramType == "sliders":
                self.paramValueTypeOf[x] = self.allParamsJson[x]["valuetype"]

        # The things here get ADDED to the ones already read in from paramenums.json
        for pt in {"sound", "visual", "effect", "sliders"}:
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

        self.paramenums["sliderParam"] = self.allParamNames

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

            # allparamsjson[parambasename] = {
            #     "valuetype": newparamsjson[name]["valuetype"],
            #     "min": newparamsjson[name]["min"],
            #     "max": newparamsjson[name]["max"],
            #     "randmin": newparamsjson[name]["randmin"],
            #     "randmax": newparamsjson[name]["randmax"],
            #     "init": newparamsjson[name]["init"],
            #     "comment": newparamsjson[name]["comment"]
            #     }

        # allparamsjson = palette.readJsonPath(palette.configFilePath("Params.json"))
        return allparamsjson

    def sendPadOneEffectVal(self,pad,name,val):
        # Effect parameters that have ":" in their name are plugin parameters
        i = name.find(":")
        if i > 0:
            if val == "":
                v = 0.0
            else:
                v = float(val)
            self.sendPadOneEffectParam(pad,name[0:i],name[i+1:],v)
        else:
            onoff = palette.boolValueOfString(val)
            self.sendPadOneEffectOnOff(pad,name,onoff)


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
        # canvas = tk.Canvas(self, background=ColorAqua, highlightthickness=0, height=4)
        # canvas.pack(side=tk.TOP,fill=tk.X)

        self.makePadFrame(self,"A",0.05,0.05)
        self.makePadFrame(self,"B",0.15,0.55)
        self.makePadFrame(self,"C",0.55,0.55)
        self.makePadFrame(self,"D",0.65,0.05)

        self.makeGlobalButton(self,0.5,0.15)
        self.padGlobalOn = True

        self.config(background=ColorBg)

    def setPadLabel(self,pad,label):
        # self.padLabel[pad].config(text=label)
        # print("setPadLabel needs to draw in canvas")
        pass

    def makePadFrame(self,parent,pad,x0,y0):

        self.padFrame[pad] = tk.Frame(self)
        self.padFrame[pad].place(relx=x0,rely=y0,relwidth=0.3,relheight=0.4)
        self.padFrame[pad].config(borderwidth=2,relief="solid",background=ColorUnHigh)
        self.padFrame[pad].bind("<Button-1>", lambda p=pad: self.padCallback(p))

        # self.padLabel[pad] = ttk.Label(self.padFrame[pad], text="")
        # self.padLabel[pad].pack(side=tk.TOP)
        # self.padLabel[pad].config(background=ColorUnHigh,font=padLabelFont)
        # self.padLabel[pad].bind("<Button-1>", lambda p=pad: self.padCallback(p))

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
        self.padGlobalLabel.configure(style='GlobalDisabled.TLabel')
        # self.padGlobalLabel.config(background=ColorUnHigh)
        self.padGlobalLabel.bind("<Button-1>", self.globalCallback)

    def globalCallback(self,e):
        self.padGlobalOn = not self.padGlobalOn
        self.refreshColors()

    def refreshColors(self):
        if self.padGlobalOn:
            color = ColorHigh
        else:
            color = ColorUnHigh
        self.padGlobalButton.config(background=color)
        self.padGlobalLabel.config(background=color)
        for p in self.controller.PadNames:
            if self.padGlobalOn or p != self.controller.PadName:
                self.colorPad(p,color)
            else:
                self.colorPad(p,ColorHigh)

    def colorPad(self,pad,color):
        self.padFrame[pad].config(background=color)
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
            color = ColorRed
        else:
            color = self.controller.padColor(pad)
        self.padCanvas[pad].create_oval(x-z,y-z,x+z,y+z,outline=color)
        # self.padFrame[pad].config(background=color)

    def padCallback(self,e):
        # if self.controller.advancedLevel==0:
        #    return
        for pad in self.padFrame:
            if e.widget == self.padFrame[pad]:
                self.padGlobalOn = False
                self.controller.padChooserCallback(pad)
                self.refreshColors()
                return
        print("No pad found in padCallback!?")


class SelectHeader(tk.Frame):

    def __init__(self, parent, controller):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=ColorBg)

        self.titleFrame = tk.Frame(self, background=ColorBg)
        self.titleFrame.pack(side=tk.TOP, fill=tk.X, expand=True)

        self.pageButton = {}

        self.headerButton = ttk.Button(self.titleFrame, text="Preset", style='Header.TLabel',
            command=lambda : self.controller.togglePageButons())
        self.headerButton.pack(side=tk.LEFT)

        for i in PageNames:
            self.makeHeaderButton(i,PageNames[i])

        self.padChooser = PadChooser(parent=parent, controller=controller)
        self.padChooser.place(in_=self.titleFrame, relx=0.7, rely=0, relwidth=0.3, relheight=1.0)

    def spacer(self,height):
        spacer = tk.Canvas(self, background=ColorBg, highlightthickness=0, height=height)
        spacer.pack(side=tk.TOP)

    def makeHeaderButton(self,pageName,pageTitle):
        # print("makeHeaderButton name=",pageName)
        self.pageButton[pageName] = ttk.Button(self.titleFrame, text=pageTitle, style='HeaderDisabled.TLabel',
            command=lambda nm=pageName: self.controller.clickEditPage(nm))
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
        self.config(background=ColorBg)

        self.titleFrame = tk.Frame(self, background=ColorBg)
        self.titleFrame.pack(side=tk.TOP, fill=tk.X, expand=True)

        self.pageButton = {}
        # self.performHeaderLabel("Control")
        self.headerButton("main","Main")
        # self.headerButton("sliders1","Sliders1")
        # self.headerButton("sliders2","Sliders2")
        # self.headerButton("sliders3","Sliders3")

        self.performHeaderInfo("")

        self.repack()

    def repack(self):
        for pageName in self.pageButton:
            if not self.controller.showSliders and pageName != "main":
                self.pageButton[pageName].pack_forget()
            else:
                self.pageButton[pageName].pack(side=tk.LEFT,padx=5)

    def spacer(self,height):
        w = tk.Canvas(self, background=ColorBg, highlightthickness=0, height=height)
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
        self.performMessageLabel = ttk.Label(self.titleFrame, text=text, background=ColorBg, style='PerformMessage.TLabel')
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

    def __init__(self, parent, controller, paramstype, params):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=ColorBg)

        self.ischanged = False
        self.canRevert = False
        self.params = params
        self.paramsnameVar = tk.StringVar()
        self.paramsname = ""
        # Should probably rename paramstype (and other params* names)
        # to avoid confusion with paramname
        self.paramstype = paramstype

        saveArea = self.makeButtonArea()
        saveArea.pack(side=tk.TOP, fill=tk.X)

        # f = tk.Frame(self, background=ColorBg)
        self.paramsFrame = self.makeParamsArea(self)
        self.paramsFrame.pack(side=tk.LEFT, pady=0)

        self.scrollbar = ScrollBar(parent=self, notify=self)
        self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=10, padx=5)

        self.updateParamFiles()
        self.updateParamView()

        defname = self.controller.selectorPage[paramstype].defaultVal()
        self.setParamsName(defname)

    def updateParamFiles(self):
        files = palette.presetsListAll(self.paramstype)
        self.paramFiles = files
        self.comboParamsname.configure(values=self.paramFiles)

    def makeParamsArea(self,container):

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

            # print("MakeParamsArea paramstype=",self.paramstype," Param=",name)
            self.paramRowName.append(name)
            self.paramLabelWidget[name] = ttk.Label(f, width=22, text=name, style='ParamName.TLabel')
            self.paramLabelWidget[name].config()

            self.paramValueWidget[name] = ttk.Label(f, width=10, anchor=tk.E, style='ParamValue.TLabel')
            self.paramValueWidget[name].bind("<Button-1>", lambda event,nm=name: self.valueClicked(nm))

        # The widgets for << < . . > >> are static, in the displayed rows
        for row in range(0,paramDisplayRows):
            f2 = tk.Frame(f, background=ColorBg)
            self.adjustButton(f2,row,"<<", -3)
            self.adjustButton(f2,row,"<", -2)
            self.adjustButton(f2,row,".", -1)
            self.adjustButton(f2,row,".", 1)
            self.adjustButton(f2,row,">", 2)
            self.adjustButton(f2,row,">>", 3)
            self.paramAdjustFrame[row] = f2

        return f

    def makeButtonArea(self):
        f = tk.Frame(self, background=ColorBg)

        if self.paramstype != "snap":
            self.initButton = ttk.Label(f, text="Init", style='Button.TLabel')
            self.initButton.bind("<Button-1>", lambda event:self.initCallback())
            self.initButton.bind("<ButtonRelease-1>", lambda event:self.initRelease())
            self.initButton.pack(side=tk.LEFT, padx=2)

            self.randButton = ttk.Label(f, text="Rnd", style='Button.TLabel')
            self.randButton.bind("<Button-1>", lambda event:self.randCallback())
            self.randButton.bind("<ButtonRelease-1>", lambda event:self.randRelease())
            self.randButton.pack(side=tk.LEFT, padx=2)

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

        self.revertButton = ttk.Label(f, text="", style='Button.TLabel')
        self.revertButton.bind("<Button-1>", lambda event:self.revert())

        self.comboParamsname = ttk.Combobox(f, textvariable=self.paramsnameVar,
                font=comboFont, style='custom.TCombobox')
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
        v = None
        s = widg.cget("text")
        if t == "bool":
            if s == "":
                v = False
            else:
                v = palette.boolValueOfString(s)
        elif t == "int":
            if s == "":
                v = 0
            else:
                v = int(s)
        elif t == "double" or t == "float":
            if s == "":
                v = 0.0
            else:
                v = float(s)
        elif t == "string":
            v = str(s).strip()
        if v == None:
            print("Hmmm, getValue of paramstype=",self.paramstype," name=",name," returns None?")
        return v

    def hasParameter(self,name):
        return (name in self.paramValueWidget)

    def changeValueLabel(self,name,v,refresh=False):
        # print("CHANGE VALUE LABEL EDIT PAGE=",self.paramstype," name=",name," v=",v)
        if not name in self.paramValueWidget:
            # Old parameters may still be in snapshots
            # print("Hmm, ",name," not a current parameter!?")
            return
        # print("CHANGE VALUE LABEL EDIT OK!")
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

    def setParamsName(self,name):
        self.paramsname = name
        if name != "CurrentSnapshot":
            try:
                n = self.paramFiles.index(name)
                self.comboParamsname.current(n)
            except:
                pass

    def startEditing(self,name,doLift=True,clearChange=True,doSend=False):

        self.comboParamsname.configure(values=self.paramFiles)

        self.setParamsName(name)

        snappage = self.controller.editPage["snap"]

        if name == "CurrentSnapshot" and self.paramstype != "snap":
            # pull param values from "snap" page (i.e. the CurrentSnapshot)
            for p in snappage.params:
                snapv = snappage.getValue(p)
                (_,baseparam) = padOfParam(p)
                # if it's a pad parameter for the current pad
                ptype = self.controller.allParamsJson[baseparam]["paramstype"]
                # if it's a parameter for the current page
                if ptype == self.paramstype:
                    # get the value from the CurrentSnapshot
                    self.changeValueLabel(baseparam,snapv)

                # if it's a slider param
                slider = sliderIndexOfParam(p)
                if self.paramstype == "sliders" and slider:
                    self.changeValueLabel(p,snapv)
        else:
            if self.paramstype == "snap":
                self.controller.readSnapParamsFile(self.paramsname)
            else:
                self.controller.readParamsFileIntoSnap(self.paramstype,self.paramsname)

            for p in self.params:
                self.changeValueLabel(p,self.getValue(p))

            if doSend:
                self.controller.sendParams(self.params,self.paramstype)

        if self.paramstype != "snap" and clearChange:
            self.clearChanged()
        if doLift:
            self.lift()

    def loadCallback(self):
        paramsname = self.paramsnameVar.get()
        self.loadParams(paramsname,doSend=True)

    def loadParams(self,name,doSend=False):
        print("loadParams name=",name,"doSend=",doSend)
        self.startEditing(name,doSend=doSend)
        # After loading on the snap page, we return to CurrentSnapshot
        if name != "CurrentSnapshot":
            self.saveJsonInPath(CurrentSnapshotPath())
            self.saveJsonInPath(CurrentSnapshotPreviousPath())
            self.startEditing("CurrentSnapshot")

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
        j["paramstype"] = self.paramstype 
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
        if name == "CurrentSnapshot":
            return

        self.saveJson(self.paramstype,name)
        self.clearChanged()

        self.updateParamFiles()
        self.controller.updateSelectorPage(self.paramstype,self.paramFiles)
        self.saveCancelCallback()

    def clearChanged(self):
        self.ischanged = False
        self.revertButton.pack_forget()

    def setChanged(self):
        self.ischanged = True
        self.revertButton.config(text="Revert")
        if self.canRevert:
            self.revertButton.pack(side=tk.LEFT, expand=True, padx=4)

    def revert(self):
        if self.paramstype != "snap":
            print("HEY! revert should only work on the snap page")
            return
        if self.ischanged:
            self.controller.revertToBackup()
            # We assume startEditing() will load CurrentSnapshot
            self.startEditing("CurrentSnapshot")
            self.controller.sendSnap()
            self.clearChanged()

    def saveJson(self,section,paramsname,suffix=".json"):
        if section == "snap" and paramsname == "CurrentSnapshot":
            print("HEY!  use saveJsonInPath!")
            return
        # Note: saving always happens in the localPresetsFilePath,
        # even if the original one was loaded from a different directory
        fpath = palette.localPresetsFilePath(section,paramsname,suffix)
        self.saveJsonInPath(fpath)

    def jsonParamDump(self):
        newjson = {}
        newjson["params"] = {}
        for name in self.params:
            newjson["params"][name] = {}
            w = self.paramValueWidget[name]
            newjson["params"][name] = self.normalizeJsonValue(name,w.cget("text"))
        return newjson

    def saveJsonInPath(self,fpath):
        newjson = self.jsonParamDump()
        print("Saving JSON:",fpath)
        f = open(fpath,"w")
        f.write(json.dumps(newjson, sort_keys=True, indent=4, separators=(',',':')))
        # To avoid complaints from editors, add a final newline
        f.write("\n")
        f.close()

    # Return value of normalizeJsonValue is always a string
    def normalizeJsonValue(self,name,v):
        t = self.controller.paramValueTypeOf[name]
        if t == "bool":
            return "true" if palette.boolValueOfString(v) else "false"
        if t == "int":
            v = int(v)
            mn = int(self.params[name]["min"])
            mx = int(self.params[name]["max"])
            v = mn if v < mn else mx if v > mx else v
            return ("%6d" % (int(float(v)))).strip()
        if t == "double" or t == "float":
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
        self.config(background=ColorBg)

        self.scroll = tk.Canvas(self, background=ColorScrollbar, highlightthickness=0)
        self.scroll.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.scroll.bind("<Button-1>", self.scrollClick)
        self.scroll.bind("<B1-Motion>", self.scrollMotion)
        # self.scroll.bind("<MouseWheel>", self.scrollWheel)

        self.thumb = tk.Canvas(self.scroll, background=ColorThumb, highlightthickness=0)
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
        self.config(background=ColorBg)

        self.frame = tk.Frame(self, background=ColorBg)
        self.frame.pack(side=tk.TOP, fill=tk.BOTH, expand=True, pady=5)

        self.performButton = {}
        self.buttonNames = []

        self.makePerformButton("loopingonoff")
        self.makePerformButton("loopinglength")
        self.makePerformButton("loopingfade")
        self.makePerformButton("Loop_Clear", self.controller.clearLoop)
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

            if name in self.controller.perpadPerformVal:
                text = self.controller.perpadPerformVal[name][pad]["label"]
            elif name in self.controller.globalPerformVal:
                text = self.controller.globalPerformVal[name]["label"]
            else:
                text = button.cget("text")

            if isTwoLine(text):
                text = text.replace(LineSep,"\n",1)

            ipady = 0
            button.config(text=text)

            if name == "TBD" or (self.controller.advancedLevel==0 and name in self.advancedButtons):
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
            text = text.replace(LineSep,"\n",1)
        self.performButton[name].config(text=text, width=10, style='PerformButton.TLabel')

    def performCallback(self,name):
        controller = self.controller
        controller.resetLastAnything()
        if name in PerPadPerformLabels:
            v = controller.perpadPerformVal[name][controller.PadName]
            nv = controller.nextValue(PerPadPerformLabels[name],v)
            text = nv["label"]
            if isTwoLine(text):
                text = text.replace(LineSep,"\n",1)
            self.performButton[name].config(text=text)

            controller.perpadPerformVal[name][controller.PadName] = nv
            controller.sendPadPerformVal(controller.PadName,name)

        elif name in GlobalPerformLabels:
            v = controller.globalPerformVal[name]
            nv = controller.nextValue(GlobalPerformLabels[name],v)
            text = nv["label"]
            if isTwoLine(text):
                text = text.replace(LineSep,"\n",1)
            self.performButton[name].config(text=text)

            controller.globalPerformVal[name] = nv
            controller.sendGlobalPerformVal(name)
        else:
            print("UNHANDLED performCallback name=",name)

class PagePerformSliders(tk.Frame):

    def __init__(self, parent,controller,slidersNum):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=ColorBg)
        # self.sliderGlobalOn = True

        self.sliderScale = {}
        self.sliderLabel1 = {}
        self.sliderParam = {}
        self.sliderModify = {}
        self.slidersNum = slidersNum

        self.valsframe = tk.Frame(self, background=ColorBg)
        self.valsframe.pack(side=tk.LEFT, fill=tk.BOTH, expand=True, padx=10)

        self.globalframe = tk.Frame(self, background=ColorBg)
        self.globalframe.pack(side=tk.LEFT,padx=10)

        slidersname = self.controller.selectorPage["sliders"].defaultVal()

        self.makeSliders()
        if slidersname != "":
            self.setSliders(slidersname)

        self.doLayout()

    def makeSliders(self):
        self.spacer = ttk.Label(self.valsframe,text="",style='Slider.TLabel')
        for i in range(8):
            self.sliderScale[i] = ScrollBar(self.valsframe,notify=self,tag=i)
            self.sliderLabel1[i] = ttk.Label(self.valsframe,width=8,style='Slider.TLabel')

    def scrollNotify(self,sfy,tag):
        # v = 1.0 - sfy
        v = sfy
        val = v * v * v
        print("scrollNotify sfy=",sfy," val=",val)
        self.sliderValueChanged(val,tag)

    def doLayout(self):
        self.spacer.grid(row=0,column=0)
        for i in range(0,8):
            rx = (i / 8.0)
            rw = (1.0 / 8.0) - 0.05
            self.sliderScale[i].place(relx=rx+0.03,rely=0.08,relheight=0.75,relwidth=rw)
            self.sliderLabel1[i].place(relx=rx,rely=0.85)
            self.sliderLabel1[i].config(text=self.sliderParam[i])

    def sliderValueChanged(self,val,i):
        param = self.sliderParam[i]
        modify = self.sliderModify[i]
        self.controller.sliderCallback(param,val,modify)
    
    def sliderNameChanged(self,newname,i):
        self.sliderLabel1[i].config(text=newname)
        self.sliderParam[i] = newname
    
    def setSliders(self,slidersname):
        j = palette.readJsonPath(palette.presetsFilePath("sliders", slidersname))
        # print("slidersname=",slidersname," j=",j)
        p = j["params"]
        for i in range(8):
            # s = self.slider[i]
            self.sliderModify[i] = p["slider%dmodify" % (i+1)]
            self.sliderParam[i]  = p["slider%dparam" % (i+1)]

            self.sliderLabel1[i].config(text=self.sliderParam[i])

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
                        selectButtonText = selectButtonText.replace(LineSep,"\n",1)
                        selectButtonText = selectButtonText.replace(LineSep," ")
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

def makeStyles(app):
    app.option_add('*TCombobox*Listbox.font', comboFont)

    s = ttk.Style()

    s.configure('.', font=largestFont, background=ColorBg, foreground=ColorText)
    s.configure('Enabled.TLabel', foreground='green', background=ColorHigh, relief="flat", justify=tk.CENTER, font=mediumFont)
    s.configure('Disabled.TLabel', foreground='red', background=ColorButton, relief="flat", justify=tk.CENTER, font=mediumFont)
    s.configure('Edit.TLabel', font=largestFont, foreground=ColorText, background=ColorBg)

    s.configure('Button.TLabel', font=largerFont, foreground=ColorText, background=ColorButton)
    s.configure('ButtonHigh.TLabel', font=largerFont, foreground=ColorText, background=ColorHigh)

    s.configure('Red.TLabel', foreground='red', justify=tk.CENTER, background=ColorBg)
    s.configure('Bright.TLabel', foreground=ColorBright, justify=tk.CENTER, background=ColorBg)
    s.configure('TinyBright.TLabel', foreground=ColorBright, justify=tk.CENTER, background=ColorBg, font=mediumFont)
    s.configure('Black.TLabel', foreground='black')

    s.configure('Date.TLabel', font=largeFont, foreground=ColorText, justify=tk.LEFT)

    s.configure('PLAY.TLabel', font=largestFont, foreground=ColorText, background=ColorButton, justify=tk.LEFT)

    s.configure('ParamName.TLabel', font=largerFont, foreground=ColorText, justify=tk.LEFT)
    s.configure('ParamValue.TLabel', foreground=ColorText, borderwidth=2, justify=tk.RIGHT, background=ColorBg, font=largerFont)
    s.configure('ParamAdjust.TLabel', foreground=ColorText, borderwidth=2, anchor=tk.CENTER, background=ColorButton, font=largeFont)

    s.configure('HeaderEnabled.TLabel', background=ColorHigh, relief="flat", justify=tk.CENTER, font=largestFont)
    s.configure('HeaderDisabled.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, font=largestFont)

    s.configure('GlobalEnabled.TLabel', background=ColorHigh, relief="flat", justify=tk.CENTER, font=hugeFont)
    s.configure('GlobalDisabled.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, font=hugeFont)

    s.configure('PerformMessage.TLabel', background=ColorBg, foreground=ColorRed, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=performFont)

    s.configure('Header.TLabel', background=ColorButton, foreground=ColorWhite, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largestFont)
    s.configure('PerformHeader.TLabel', background=ColorButton, foreground=ColorBright, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=performFont)

    s.configure('ScrollButton.TLabel', foreground=ColorText, font=largestFont, background=ColorScrollbar, anchor=tk.CENTER)

    s.configure('Patch.TLabel', foreground=ColorText, font=patchFont, background=ColorButton, anchor=tk.CENTER, justify=tk.CENTER)
    s.configure('PatchHighlight.TLabel', foreground=ColorText, font=patchFont, background=ColorRed, anchor=tk.CENTER, justify=tk.CENTER)

    s.configure('PatchTwoLine.TLabel', foreground=ColorText, font=patchTwoLineFont, background=ColorButton, anchor=tk.CENTER, justify=tk.CENTER)
    s.configure('PatchTwoLineHighlight.TLabel', foreground=ColorText, font=patchTwoLineFont, background=ColorHigh, anchor=tk.CENTER, justify=tk.CENTER)

    s.configure('Slider.TLabel', foreground=ColorText, font=sliderFont, background=ColorBg, anchor=tk.CENTER)

    s.configure('RecordingButton.TLabel', background=ColorRed, relief="flat", justify=tk.CENTER, align=tk.CENTER, font=largeFont)

    s.configure('PerformButton.TLabel', foreground=ColorText, background=ColorButton, relief="flat", justify=tk.CENTER,
        anchor=tk.CENTER, font=performFont)

    s.configure('RecordingName.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, anchor=tk.CENTER, font=largestFont)
    s.configure('RecordingButton.TLabel', background=ColorButton, relief="flat", justify=tk.CENTER, anchor=tk.CENTER, font=largerFont)

    s.configure('custom.TCombobox', foreground=ColorComboText, background=ColorBg)

    s.map('Patch.TLabel',
        foreground=[('disabled', 'yellow'),
                    ('pressed', ColorText),
                    ('active', ColorText)],
        background=[('disabled', 'yellow'),
                    ('pressed', ColorHigh),
                    ('active', ColorButton)]
        )
    s.map('PatchTwoLine.TLabel',
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
 
def startgui():
    # print("STARTGUI called")
    global StartupMode
    StartupMode = False
    # global startupPhase
    # startupPhase = ""

def padOfParam(paramname):
    pad = paramname[0]
    if pad == "A" or pad == "B" or pad == "C" or pad == "D":
        baseparam = paramname[2:]
        return (pad,baseparam)
    else:
        return (None,paramname)

def sliderIndexOfParam(paramname):
    if paramname[0:6] == "slider":
        # Assumes 1-digit slider numbers.  The parameter names exposed
        # as strings use 1-8, but the index returned here is 0-7
        return int(paramname[6:7]) - 1
    else:
        return None

def isTwoLine(text):
    return text.find(LineSep) >= 0 or text.find("\n") >= 0

def CurrentSnapshotPath():
    return palette.localconfigFilePath("CurrentSnapshot.json")

def CurrentSnapshotBackupPath():
    return palette.localconfigFilePath("CurrentSnapshot.backup")

def CurrentSnapshotPreviousPath():
    return palette.localconfigFilePath("CurrentSnapshot.previous")

def initMain(app):
    app.iconbitmap(palette.configFilePath("palette.ico"))
    app.mainLoop()

if __name__ == "__main__":

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

    setFontSizes(fontFactor)

    global app
    app = ProGuiApp()

    makeStyles(app)

    app.wm_geometry("%dx%d" % (GuiWidth,GuiHeight))

    delay = 0.0

    threading.Timer(delay, startgui).start()

    initMain(app)
