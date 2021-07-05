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
global AllPadsSelected
AllPadsSelected = False
global DefaultAdvanced
DefaultAdvanced = 1

ControlPageNames = {
    "main":"Main",
}

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

        advanced = palette.ConfigValue("advanced")
        if advanced == "":
            level = 0
        else:
            level = int(advanced)
        self.setAdvanced(level)

        self.performHeader = None

        self.perpadPerformVal = {}
        self.globalPerformVal = {}
        for s in palette.PerPadPerformLabels:
            self.perpadPerformVal[s] = {}
        for s in palette.GlobalPerformLabels:
            self.globalPerformVal[s] = {}

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
        self.resetLastAnything()

        self.topContainer.pack(side=tk.TOP, fill=tk.BOTH, expand=True)
        self.performHeader.pack(side=tk.TOP,fill=tk.X)
        self.setPerformMessage("")
        self.selectPerformPage("main")
        self.selectPage("snap")
        self.resetVisibility()

        self.topContainer.bind_all("<MouseWheel>", self.scrollWheel)

        self.setupVals()

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
                    print("Not loading any initial snapshot")

            if palette.resetAfterInactivity>0 and (now - self.lastAnything) > palette.resetAfterInactivity:
                print("Resetting after no activity!!")
                self.resetLastAnything()

                global DefaultAdvanced
                self.setAdvanced(DefaultAdvanced)
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
        return True

    def readOtherParamsJsonIntoSnapAndQuad(self,paramstype,paramsname,j):
        paramvals = j["params"]
        if paramstype == "quad":
            print("Unexpected value in readParamsFileIntoSnapAndQuad")
            return

        # The params in the file may not include all of the
        # parameters for the given paramstype, so we loop through
        # all the parameters of a given type
        for name in self.allParamsJson:
            allj = self.allParamsJson[name]
            t = allj["paramstype"]
            if paramstype == "snap":
                if t != "sound" and t != "visual" and t != "effect":
                    continue
            else:
                if t != paramstype:
                    continue
            (pad,base) = padOfParam(name)
            if pad != None:
                print("Unexpected non-Non pad??")
                continue
            if base in paramvals:
                v = paramvals[base]
                if "value" in v:
                    # Old versions of the param files used nested structure with "enabled" and "value"
                    print("IS THIS CODE USED ANYMORE?")
                    v = v["value"]
            else:
                v = allj["init"]

            global AllPadsSelected
            if AllPadsSelected:
                for pad in self.PadNames:
                    self.changeValueAndSend(pad,base,v,False)
                    self.changeValueInQuad(pad,base,v)
            else:
                self.changeValueAndSend(self.PadName,base,v,False)
                self.changeValueInQuad(self.PadName,base,v)

    def readOtherParamsFile(self,paramstype,paramsname):
        if isSnapshotName(paramsname):
            fpath = palette.configFilePath(paramsname+".json")
        else:
            fpath = palette.searchPresetsFilePath(paramstype, paramsname)
        print("readOtherParamsFile: path=",fpath)
        try:
            f = open(fpath)
        except:
            print("No such file?  BB fpath=",fpath)
            return
        j = json.load(f)
        self.loadPageJson(self.editPage[paramstype],j,paramstype)
        f.close()

    def readSnapParamsFileIntoPage(self,paramsname,pagename):
        print("\nREAD SNAP PARAMS FILE paramsname=",paramsname)
        # Read parameters from a json file
        if isSnapshotName(paramsname):
            fpath = palette.configFilePath(paramsname+".json")
        else:
            fpath = palette.searchPresetsFilePath("snap", paramsname)
        print("readSnapParamsFile: path=",fpath)
        try:
            f = open(fpath)
        except:
            print("No such file?  BB fpath=",fpath)
            return
        j = json.load(f)
        self.loadPageJson(self.editPage[pagename],j)
        f.close()

    def readQuadParamsFile(self,paramsname):
        fpath = palette.searchPresetsFilePath("quad", paramsname)
        print("readQuadParamsFile: fpath=",fpath)
        try:
            f = open(fpath)
        except:
            print("No such file?  CC fpath=",fpath)
            return
        j = json.load(f)
        self.loadQuadJson(j,self.editPage["quad"])
        f.close()

    def loadPageJson(self,page,j,paramstype=None):
        # If parameters (of the desired type) exist that aren't in j, add their default value
        snappage = self.editPage["snap"]
        for name in self.allParamsJson:
            allj = self.allParamsJson[name]
            (_,base) = padOfParam(name)
            pt = allj["paramstype"]
            if paramstype == None or pt == paramstype:
                fullname = base
                if not fullname in j["params"]:
                    j["params"][fullname] = allj["init"]

        for name in j["params"]:
            v = j["params"][name]
            page.changeValueLabel(name,v)
            snappage.changeValueLabel(name,v)

    def loadQuadJson(self,j,quadpage):

        # If parameters exist that aren't in j, add their default value to j.
        # This helps when new parameters are added that aren't in existing preset files.
        for name in self.allParamsJson:
            allj = self.allParamsJson[name]
            (_,base) = padOfParam(name)
            # paramsType = allj["paramstype"]
            for pad in self.PadNames:
                fullname = pad + "-" + base
                if not fullname in j["params"]:
                    j["params"][fullname] = allj["init"]

        for name in j["params"]:
            v = j["params"][name]
            quadpage.changeValueLabel(name,v)

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

        self.editPage["snap"].canRevert = True

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
                    paramstype=pagename, params=self.paramsOfType[pagename])
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

        # A second click on a page header will toggle editMode if advanced>1
        if self.advancedLevel > 1 and self.currentPageName == pagename:
            self.editMode = not self.editMode

        self.selectPage(pagename)
        if pagename == "quad":
            pass
        elif pagename == "snap":
            pass
        else:
            if self.editMode:
                global AllPadsSelected
                if not AllPadsSelected:
                    self.loadOther(pagename,CurrentPadFilename(self.PadName))
                    self.sendOther(pagename)

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

        if self.advancedLevel > 1 and self.editMode:
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

    def sendPadParamValue(self,pad,paramname,val):
        print("sendPadParamValue pad=",pad," name=",paramname," val=",val)
        paramType = self.paramTypeOf[paramname]
        # if paramType == "effect":
        #     self.sendPadOneEffectVal(pad,paramname,val)
        # else:
        palette.palette_region_api(self.PadName,paramType+".set_param",
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
                fullparam = baseparam
            if not baseparam in self.paramTypeOf:
                print("param ",baseparam," isn't in paramTypeOf?")
                continue
            if self.paramTypeOf[baseparam] == "effect":
                val = page.getValue(fullparam)
                self.sendPadOneEffectVal(pad,fullparam,val)
                continue
            if not self.paramTypeOf[baseparam] in self.AllPageNames:
                print("Not sending param=",origp)
                continue
            v = page.getValue(origp)
            if paramstype == "snap":
                if pad:
                    if baseparam in self.paramsOfType[paramstype]:
                        self.sendPadParamValue(pad,baseparam,v)
            elif paramstype == "quad":
                print("HEY, DOES quad need some work here? AAA")
            else:
                if baseparam in self.paramsOfType[paramstype]:
                    self.sendPadParamValue(self.PadName,origp,v)

    def paramCallback(self,paramname,newval):

        # print("paramCallback! paramname=",paramname," newval=",newval)

        (pad,baseparam) = padOfParam(paramname)
        paramstype = self.allParamsJson[baseparam]["paramstype"]

        if self.currentPageName == "snap":
            if pad:
                print("Should this happen? current page is snap, but paramname has a pad?")
                # Change the value on the other (per-param-type) editing page
                self.editPage[paramstype].changeValueLabel(baseparam,newval)
                # we still send the changed parameter out to the appropriate pad
                self.sendPadParamValue(pad,baseparam,newval)
            else:
                self.changeValueInQuad(self.PadName,baseparam,newval)
                self.sendPadParamValue(self.PadName,baseparam,newval)

        elif paramstype == "quad":
            print("HEY, DOES quad need some work here? BBB")

        else:
            if not pad:
                print("Unexpected not pad??")
                pad = self.PadName
            self.changeValueAndSend(pad,baseparam,newval,True)

        if not pad:
            # XXX todo
            global AllPadsSelected
            if AllPadsSelected:
                for pad in self.PadNames:
                    self.saveSnapshot(pad)
            else:
                pad = self.PadName
                self.saveSnapshot(pad)
        else:
            self.saveSnapshot(pad)

    def changeValueAndSend(self,pad,basename,newval,send):
        global AllPadsSelected
        if AllPadsSelected:
            self.changeValueInSnap(basename,newval)
            for qpad in self.PadNames:
                self.changeValueInQuad(qpad,basename,newval)
                if send:
                    self.sendPadParamValue(qpad,basename,newval)
        else:
            self.changeValueInSnap(basename,newval)
            self.changeValueInQuad(pad,basename,newval)
            if send:
                self.sendPadParamValue(pad,basename,newval)

    def changeValueInSnap(self,paramname,newval):
        self.editPage["snap"].changeValueLabel(paramname,newval)
        self.editPage["snap"].setChanged()

    def changeValueInQuad(self,pad,paramname,newval):
        global IsQuad
        if IsQuad:
            quadName = pad + "-" + paramname
            self.editPage["quad"].changeValueLabel(quadName,newval)
            self.editPage["quad"].setChanged()

    def savePrevious(self):
        frompath = CurrentPadPath(self.PadName)
        if os.path.exists(frompath):
            topath = CurrentPadPreviousPath(self.PadName)
            palette.copyFile(frompath,topath)

    def restorePrevious(self):
        frompath = CurrentPadPreviousPath(self.PadName)
        if os.path.exists(frompath):
            topath = CurrentPadPath(self.PadName)
            palette.copyFile(frompath,topath)

    def selectorApply(self,apply,paramstype,val):
        if paramstype == "snap" or paramstype == "quad":
            print("selectorApply not yet implemented on snap/quad")
        else:
            self.applyToAllParams(apply,paramstype,val)
            page = self.editPage["snap"]
            self.sendSnapPad(self.PadName,page,paramstype)

    def applyToAllParams(self,apply,paramstype,val):
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
                print("XXXXX - Should this code use self.changeValue?")
                editpage.changeValueLabel(base,v)
                self.changePadParamValue(self.PadName,name,v)
                self.editPage["snap"].changeValueLabel(base,v)
                global IsQuad
                if IsQuad:
                    quadParamName = self.PadName + "-" + name
                    self.editPage["quad"].changeValueLabel(quadParamName,v)

        self.saveSnapshot(self.PadName)

    def saveSnapshot(self,pad):
        snappage = self.editPage["snap"]
        snappage.saveJsonInPath(CurrentPadPath(pad))
        snappage.setChanged()

    def selectorLoadAndSend(self,paramstype,presetname,buttoni):
        if paramstype == "snap":
            self.savePrevious()
            print("Should be highlighting buttoni=",buttoni)
            self.loadSnap(presetname)
            self.sendSnap()
        elif paramstype == "quad":
            self.savePrevious()
            print("Should be highlighting buttoni=",buttoni)
            self.loadQuad(presetname)
            self.sendQuad()
        else:
            # we want to set the values in the editing page
            self.loadOther(paramstype,presetname)
            self.sendOther(paramstype)

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
            self.loadPageJson(self.editPage["snap"],j)
            self.sendPage(self.editPage["snap"])
        elif paramstype == "quad":
            print("Hey, does quad need work here?  FFF")
        else:
            self.readOtherParamsJsonIntoSnapAndQuad(paramstype,paramsname,j)
            page = self.editPage[paramstype]
            self.sendSnapPad(self.PadName,page,paramstype)

    def padChooserCallback(self,pad):
        self.PadName = pad

        if len(self.PadNames) > 1:
            self.selectHeader.padChooser.refreshColors()

        if self.editMode:
            # XXX - If in sound/visual/edit, should load values from "snap" page?
            snapname = CurrentPadFilename(pad)
            self.editPage[self.currentPageName].startEditing(snapname)

        performControl = self.performPage["main"]
        performControl.updatePerformButtonLabels(self.PadName)

    def loadOther(self,pagename,snapname):
        page = self.editPage[pagename]
        page.loadOtherNamed(snapname)
        self.saveSnapshot(self.PadName)
 
    def loadSnap(self,snapname):
        snappage = self.editPage["snap"]
        snappage.loadSnapNamed(snapname)
        self.saveSnapshot(self.PadName)

    def revertToBackup(self):
        frompath = CurrentPadBackupPath(self.PadName)
        topath = CurrentPadPath(self.PadName)
        palette.copyFile(frompath,topath)
        print("Reverting Backup Copying ",frompath," to ",topath)

    def loadQuad(self,quadname):

        print("\n=== loadQuad ",quadname)

        quadpage = self.editPage["quad"]

        self.readQuadParamsFile(quadname)

        # quadpage.comboParamsname.configure(values=self.paramFiles)

        for pad in self.PadNames:
            self.updatePageFromQuad("snap",pad)
            self.saveSnapshot(pad)

        # for p in self.params:
        #     self.changeValueLabel(p,self.getValue(p))
        # self.lift()

    def updatePageFromQuad(self,pagename,frompad):
        quadpage = self.editPage["quad"]
        page = self.editPage[pagename]
        for name in page.params:
            qname = frompad + "-" + name
            v = quadpage.getValue(qname)
            page.changeValueLabel(name,v)

    def updateQuadFromSnap(self,frompad):
        quadpage = self.editPage["quad"]
        snappage = self.editPage["snap"]
        for name in snappage.params:
            v = snappage.getValue(name)
            qname = frompad + "-" + name
            quadpage.changeValueLabel(qname,v)
 
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

            palette.palette_region_api(self.PadName, "loop_recording", '"onoff": "'+str(reconoff)+'"')
            palette.palette_region_api(self.PadName, "loop_playing", '"onoff": "'+str(playonoff)+'"')

        elif name == "loopinglength":
            v = self.perpadPerformVal["loopinglength"][pad]["value"]
            palette.palette_region_api(self.PadName, "loop_length", '"length": "'+str(v)+'"')

        elif name == "loopingfade":
            fade = self.perpadPerformVal["loopingfade"][pad]["value"]
            palette.palette_region_api(self.PadName, "loop_fade", '"fadelength": "'+str(fade)+'"')

        elif name == "quant":
            val = self.perpadPerformVal["quant"][pad]["value"]
            palette.palette_region_api(self.PadName, "set_param",
                "\"param\": \"" + "misc.quant" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")
        elif name == "scale":
            val = self.perpadPerformVal["scale"][pad]["value"]
            palette.palette_region_api(self.PadName, "set_param",
                "\"param\": \"" + "misc.scale" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")

        elif name == "vol":
            val = self.perpadPerformVal["vol"][pad]["value"]
            # NOTE: "voltype" here rather than "vol" - should make consistent someday
            palette.palette_region_api(self.PadName, "set_param",
                "\"param\": \"" + "misc.vol" + "\"" + \
                ", \"value\": \"" + str(val) + "\"")

        elif name == "comb":
            val = 1.0
            palette.palette_region_api(self.PadName, "loop_comb",
                "\"value\": \"" + str(val) + "\"")

        elif name == "midithru":
            thru = self.perpadPerformVal["midithru"][pad]["value"]
            palette.palette_region_api(self.PadName, "midi_thru", "\"thru\": \"" + str(thru) + "\"")

        elif name == "useexternalscale":
            onoff = self.perpadPerformVal["useexternalscale"][pad]["value"]
            palette.palette_region_api(self.PadName, "useexternalscale", "\"onoff\": \"" + str(onoff) + "\"")

        elif name == "midiquantized":
            quantized = self.perpadPerformVal["midiquantized"][pad]["value"]
            palette.palette_region_api(self.PadName, "midi_quantized", "\"quantized\": \"" + str(quantized) + "\"")

        elif name == "transpose":
            val = self.globalPerformVal["transpose"][pad]["value"]
            palette.palette_region_api(self.PadName, "set_transpose", "\"value\": \""+str(val) + "\"")

    def sendGlobalPerformVal(self,name):

        if name == "tempo":
            val = self.globalPerformVal["tempo"]["value"]
            palette.palette_global_api("set_tempo_factor", "\"value\": \""+str(val) + "\"")

        # elif name == "configname":
        #     config = self.globalPerformVal["configname"]["value"]
        #     palette.setConfigName(config)
        #     print("CONFIGNAME setting to ",palette.getConfigName())

    def clearPadLoop(self,pad):
        palette.palette_region_api(self.PadName, "loop_clear", "")

    def combPadLoop(self,pad):
        palette.palette_region_api(self.PadName, "loop_comb", "")

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
            if self.advancedLevel == 0:
                self.showAllPages = False
            elif self.advancedLevel == 1:
                self.showAllPages = True
            elif self.advancedLevel == 2:
                self.showAllPages = True
            else:
                print("Unrecognized advanced value: ",level)

    def resetAll(self):

        palette.palette_global_api("audio_reset")

        self.resetLastAnything()
        self.sendANO()
        self.clearExternalScale()

        for pad in self.PadNames:
            for name in palette.PerPadPerformLabels:
                self.sendPadPerformVal(pad,name)

        for name in palette.GlobalPerformLabels:
            self.sendGlobalPerformVal(name)

        self.setPerformMessage("")
        for pad in self.PadNames:
            self.clearPadLoop(pad)

        self.performPage["main"].updatePerformButtonLabels(self.PadName)

    def setupVals(self):

        for pad in self.PadNames:
            for name in palette.PerPadPerformLabels:
                self.perpadPerformVal[name][pad] = palette.PerPadPerformLabels[name][0]

        for name in palette.GlobalPerformLabels:
            self.globalPerformVal[name] = palette.GlobalPerformLabels[name][0]

    def clearExternalScale(self):
        palette.palette_region_api(self.PadName, "clearexternalscale")

    def sendANO(self):
        palette.palette_region_api(self.PadName, "ANO")

    def sendOther(self,pagename):
        page = self.editPage[pagename]
        if AllPadsSelected == True:
            for pad in self.PadNames:
                self.sendOtherPad(pad,page,paramstype=pagename)
        else:
            self.sendOtherPad(self.PadName,page,paramstype=pagename)

    def sendSnap(self):
        page = self.editPage["snap"]
        if AllPadsSelected == True:
            for pad in self.PadNames:
                self.sendSnapPad(pad,page,None)
        else:
            self.sendSnapPad(self.PadName,page,None)

    def sendQuad(self):
        for pad in self.PadNames:
            print("Sending all parameters for pad = ",pad)
            for pt in ["sound","visual","effect"]:
                paramlistjson = self.quadParamListJson(pt,pad)
                # print("sendQuad calling set_params for pad=",pad," pt=",pt," json=",paramlistjson)
                palette.palette_region_api(pad, pt+".set_params", paramlistjson)

    def quadParamListJson(self,paramstype,pad):
        page = self.editPage["quad"]
        paramlist = ""
        sep = ""
        for name in self.allParamsJson:
            j = self.allParamsJson[name]
            if j["paramstype"] == paramstype:
                paramname = pad + "-" + name
                v = page.getValue(paramname)
                paramlist = paramlist + sep + "\"" + name + "\" : \"" + str(v) + "\""
                sep = ", "

        return paramlist

    def snapParamListJson(self,paramstype,pad,page):
        paramlist = ""
        sep = ""
        for name in self.allParamsJson:
            j = self.allParamsJson[name]
            if j["paramstype"] == paramstype:
                # paramname = pad + "_" + name
                paramname = name
                v = page.getValue(paramname)
                paramlist = paramlist + sep + "\"" + name + "\" : \"" + str(v) + "\""
                sep = ", "

        return paramlist

    def sendSnapPad(self,pad,page,paramstype):
        for pt in ["sound","visual","effect"]:
            paramlistjson = self.snapParamListJson(pt,pad,page)
            if paramstype == None or paramstype == pt:
                palette.palette_region_api(pad, pt+".set_params", paramlistjson)

        if paramstype == None:
            for name in palette.PerPadPerformLabels:
                self.sendPadPerformVal(pad,name)

    def sendOtherPad(self,pad,page,paramstype):
        paramlistjson = self.snapParamListJson(paramstype,pad,page)
        palette.palette_region_api(pad, paramstype+".set_params", paramlistjson)

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
                    quadName = pad + "-" + name
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
        self.padGlobalButton.bind("<Button-1>", self.globalCallback)

        self.padGlobalLabel = ttk.Label(self.padGlobalButton, text="*")
        self.padGlobalLabel.pack(side=tk.TOP)
        self.padGlobalLabel.configure(style='ChooserDisabled.TLabel')
        # self.padGlobalLabel.config(background=palette.ColorUnHigh)
        self.padGlobalLabel.bind("<Button-1>", self.globalCallback)

    def globalCallback(self,e):
        global AllPadsSelected
        AllPadsSelected = not AllPadsSelected
        print("AllPadsSelected = ",AllPadsSelected)
        self.refreshColors()

    def refreshColors(self):
        global AllPadsSelected
        if AllPadsSelected:
            color = palette.ColorHigh
        else:
            color = palette.ColorUnHigh
        self.padGlobalButton.config(background=color)
        self.padGlobalLabel.config(background=color)
        for p in self.controller.PadNames:
            if AllPadsSelected or p != self.controller.PadName:
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
        # if self.controller.advancedLevel==0:
        #    return
        for pad in self.padFrame:
            if e.widget == self.padFrame[pad] or e.widget == self.padLabel[pad]:
                global AllPadsSelected
                AllPadsSelected = False
                self.controller.padChooserCallback(pad)
                self.refreshColors()
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

    def __init__(self, parent, controller, paramstype, params):
        tk.Frame.__init__(self, parent)
        self.controller = controller
        self.config(background=palette.ColorBg)

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

        self.updateParamFiles()
        self.paramsFrame = self.makeParamsArea(self)
        self.scrollbar = ScrollBar(parent=self, notify=self)

        # On the "quad" and "snap" pages, the parameter values aren't shown,
        # just the buttons to import/export/save
        # if not (paramstype == "quad" or paramstype == "snap"):

        self.paramsFrame.pack(side=tk.LEFT, pady=0)
        self.scrollbar.pack(side=tk.LEFT, fill=tk.Y, expand=True, pady=10, padx=5)
        self.updateParamView()

        defname = self.controller.selectorPage[paramstype].defaultVal()
        self.setParamsName(defname)

    def updateParamFiles(self):
        files = palette.presetsListAll(self.paramstype)
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

            # print("MakeParamsArea paramstype=",self.paramstype," Param=",name)
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

        if self.paramstype != "snap" and self.paramstype != "quad":
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
            # ignore names not on this page
            return
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
        if not isSnapshotName(name):
            try:
                n = self.paramFiles.index(name)
                self.comboParamsname.current(n)
            except:
                pass

    def loadOtherNamed(self,name):

        print("\n=== loadOtherNamed ",name)

        self.controller.readOtherParamsFile(self.paramstype,name)

        self.comboParamsname.configure(values=self.paramFiles)

        self.setParamsName(name)

        for p in self.params:
            self.changeValueLabel(p,self.getValue(p))

    def loadSnapNamed(self,name,doLift=True,clearChange=True):

        print("\n=== loadSnapNamed ",name)

        self.controller.readSnapParamsFileIntoPage(name,"snap")

        self.comboParamsname.configure(values=self.paramFiles)

        self.setParamsName(name)

        for p in self.params:
            self.changeValueLabel(p,self.getValue(p))

        if doLift:
            self.lift()

    def startEditing(self,name,doLift=True,clearChange=True):

        print("\n=== startEditing paramstype=%s name=%s" % (self.paramstype,name))
        if self.paramstype == "quad":
            print("\nAre you getting here?\n")
            self.controller.readQuadParamsFile(name)
        else:
            self.controller.readSnapParamsFileIntoPage(name,self.paramstype)

        self.comboParamsname.configure(values=self.paramFiles)

        self.setParamsName(name)

        # self.oldStartEditing()

    def oldStartEditing(self,name,doSend,doLift):

        if self.paramstype != "snap" and self.paramstype != "quad" and self.paramstype != "quad":
            self.clearChanged()
        if doLift:
            self.lift()

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
        if isSnapshotName(name):
            return

        if self.paramstype == "quad":
            for pad in self.controller.PadNames:
                self.loadSnapNamed(CurrentPadFilename(pad))
                self.controller.updateQuadFromSnap(pad)
            self.saveJson("quad",name)
        else:
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
        if self.paramstype != "snap" and self.paramstype != "quad":
            print("HEY! revert should only work on the snap or quad page")
            return
        if self.ischanged:
            self.controller.revertToBackup()
            # We assume startEditing() will load CurrentPad
            snapname = CurrentPadFilename(self.controller.PadName)
            self.startEditing(snapname)
            self.controller.sendSnap()
            self.clearChanged()

    def saveJson(self,section,paramsname,suffix=".json"):

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
                text = text.replace(palette.LineSep,"\n",1)

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
            text = text.replace(palette.LineSep,"\n",1)
        self.performButton[name].config(text=text, width=10, style='PerformButton.TLabel')

    def performCallback(self,name):
        controller = self.controller
        controller.resetLastAnything()
        if name in palette.PerPadPerformLabels:
            v = controller.perpadPerformVal[name][controller.PadName]
            nv = controller.nextValue(palette.PerPadPerformLabels[name],v)
            text = nv["label"]
            if isTwoLine(text):
                text = text.replace(palette.LineSep,"\n",1)
            self.performButton[name].config(text=text)

            controller.perpadPerformVal[name][controller.PadName] = nv
            controller.sendPadPerformVal(controller.PadName,name)

        elif name in palette.GlobalPerformLabels:
            v = controller.globalPerformVal[name]
            nv = controller.nextValue(palette.GlobalPerformLabels[name],v)
            text = nv["label"]
            if isTwoLine(text):
                text = text.replace(palette.LineSep,"\n",1)
            self.performButton[name].config(text=text)

            controller.globalPerformVal[name] = nv
            controller.sendGlobalPerformVal(name)
        else:
            print("UNHANDLED performCallback name=",name)

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
    if pad == "A" or pad == "B" or pad == "C" or pad == "D":
        baseparam = paramname[2:]
        return (pad,baseparam)
    else:
        return (None,paramname)

def isTwoLine(text):
    return text.find(palette.LineSep) >= 0 or text.find("\n") >= 0

def isSnapshotName(name):
    return name.startswith("CurrentPad")

def CurrentPadFilename(pad):
    return "CurrentPad_"+pad

def CurrentPadPath(pad):
    nm = CurrentPadFilename(pad)
    return palette.configFilePath(nm+".json")

def CurrentPadBackupPath(pad):
    nm = CurrentPadFilename(pad)
    return palette.configFilePath(nm+".backup")

def CurrentPadPreviousPath(pad):
    nm = CurrentPadFilename(pad)
    return palette.configFilePath(nm+".previous")

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
    palette.PerPadPerformLabels["scale"] = palette.SimpleScales

    global app
    app = ProGuiApp(padname,padnames,visiblepagenames)

    palette.makeStyles(app)


    app.wm_geometry("%dx%d" % (GuiWidth,GuiHeight))

    delay = 0.0

    threading.Timer(delay, startgui).start()

    initMain(app)
