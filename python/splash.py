from tkinter import ttk
from tkinter import font
from tkinter import messagebox
import tkinter as tk
import sys
import string
import threading
from subprocess import call, Popen

backcolor = 'black'
backcolor = '#038387'   # teal
backcolor = '#BF6700'   # orangey
forecolor = 'white'

class Fullscreen_Window:

    def __init__(self):
        self.tk = tk.Tk()
        self.tk.title("Space Palette Message")
        self.tk.configure(background=backcolor)
        # self.tk.attributes('-zoomed', True)  # This just maximizes it so we can see the window. It's nothing to do with fullscreen.
        self.frame = tk.Frame(self.tk)
        self.frame.pack()
        self.state = False
        self.tk.bind("<Escape>", self.end_fullscreen)

        smallFont = ('Times', 30)
        mediumFont = ('Times', 60)
        largeFont = ('Times', 96)
        paletteFont = ('Poor Richard', 96)

        rootStyle = ttk.Style()
        rootStyle.configure('.', font=paletteFont)
        rootStyle.configure('palette.TLabel', font=paletteFont)
        rootStyle.configure('large.TLabel', font=largeFont)
        rootStyle.configure('medium.TLabel', font=mediumFont)
        rootStyle.configure('small.TLabel', font=smallFont)

        lbl = ttk.Label(self.tk, text="", style='large.TLabel',
		anchor=tk.CENTER, background=backcolor, foreground=forecolor)
        lbl.pack(side="top", fill="both", expand=True)

        lbl = ttk.Label(self.tk, text="Space Palette", style='TLabel',
		anchor=tk.CENTER, background=backcolor, foreground=forecolor)
        lbl.pack(side="top", fill="both", expand=True)

        lbl = ttk.Label(self.tk, text="", style='small.TLabel',
		anchor=tk.CENTER, background=backcolor, foreground=forecolor)
        lbl.pack(side="top", fill="both", expand=True)

        for arg in sys.argv[1:]:
            arg = string.replace(arg,"\\n","\n")
            lbl = ttk.Label(self.tk, text=arg, style='medium.TLabel',
		anchor=tk.CENTER, background=backcolor, foreground=forecolor)
            lbl.pack(side="top", fill="both", expand=True)

        lbl = ttk.Label(self.tk, text="", anchor=tk.CENTER, background=backcolor, foreground=forecolor)
        lbl.pack(side="top", fill="both", expand=True)


    def end_fullscreen(self,e):
        sys.exit(0)
        self.tk.attributes("-fullscreen", False)

def resizeit():
    # call(["c:/local/bin/nircmd.exe","win","setsize","stitle","Space Palette Message","1280","-40","1280","900"])
    call(["c:/local/bin/nircmd.exe","win","setsize","stitle","Space Palette Message","1360","-20","1366","768"])

if __name__ == '__main__':
    w = Fullscreen_Window()
    threading.Timer(2.0,resizeit).start()
    w.tk.mainloop() 
