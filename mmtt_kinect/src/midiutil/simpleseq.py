from Tkinter import *
import tkMessageBox
import tkSimpleDialog
from midiseq import *
import Queue

appTitle = "Super Simpler Sequencer"

aboutText = """A simple multi-track sequencer that 
demonstrates recording and playback using
the nosuch MIDI library and PyPortMidi.

Shortcut keys:
r\ttoggle recording
space\ttoggle playback
"""

def notYet():
    tkMessageBox.showinfo(appTitle, "This command is not implemented yet.")

class ImageProvider(object):
    """
    Icons that this object provides are base64-encoded gif renderings of
    icons from the Tango Desktop Project. The Tango Desktop Project
    licenses these images under the Creative Commons Attribution Share-
    Alike License. For more information about the Tango Desktop Project
    and its terms of use, see 
    http://tango.freedesktop.org/Tango_Desktop_Project.
    """
    _midi_disconnected_gif=\
'''R0lGODlhIAAgAPcAAAAAAAAzZgAA/wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAh5AP8JHEiwoMGDCBMqHCig4cKHBRtKhAhRokWKCy1OxJhQo0OOHS+CRBjg38aRBkuiVKgSJQCS
Kwe+LNgy5r+ZNhHizCkQ506ePnkSDCq0p8yiRpMW/bmyptKmOmM6LcgU4tSVE6+i1FjUowChXsF6
7CqS7FekaAUGBAA7
'''
    _midi_connected_merged_gif=\
'''R0lGODlhIAAgAPcAAAAAAAAzZgAA/wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAhuAP8JHEiwoMGDCBMqHCig4cKHBRtKhAhRokWKCy1OxJhQo0OOHS+CRBjg38aRBkuiVKhyJUmX
CVvCLChz5sCaNv/htLlzZk+YP10GXTkUZdGRR0Em5bg0o8OmCjXmNOkxp0cBVqtmPbl1qteCAQEA
Ow==
'''
    _midi_connected_gif=\
'''R0lGODlhIAAgAPcAAAAAAAAzZgAA/wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAhlAP8JHEiwoMGDCBMqHCig4cKHBRtKhAhRokWKCy1OxJhQo0OOHS+CVLhxpMEAJheiTMnS5MqW
J2HKpPhypsCaNnMSxDmTp86cPmEG/SlzKMiJRjlqtOlRwMymTz0yFTnVKdGcAQEAOw==
'''
    _edit_clear_gif=\
'''R0lGODlhEAAQAPcAAAAAAKsbDbg4HYBRCoVVC4tZDYpaDYRWEohcGoldHI5kJa1EFapLG69GKLtK
KKZpCqlrCqttC7JtHrJyDLJ2C7x6D8F9EMhtM5yIAZyLAKGIAaGRCaKSDKOSDqOUEqSTEKWTEaWV
EqWVE6WVFKWVFaeWFqeXGKeYGamZHa2eIq2cMbGiJbChK7CiLbSkKLSnOr2oML6uMrGVar+zVr+0
V8KzN8W1Nce2NMa3Nsa2Oc29Qc2/SMG2XMO4YMK4YcKwftexYtrCA9vDBNzEB9zFENzGENzGFd/K
JuPLEeXNFOfQGO7XI/PbKvbeL/vkN/zlPP3mPM/BSdjIT9LDUNbIVtzNWN3PXszDfeTSSeTTTOXU
Tu3aRubWVuHTW+fXW+fYX+LUZ+fZbejabOzcaOrccuvdd+ved/bYYfvlRP3pUv3qX/PjZf3rYf3q
Y/3rbP3sa/3sbf3uffvqhP3vhPjqiPvti/3vigAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAQABAA
AAiwAP8pSPCvoMGDBhE8iDAAocN/ByZYgKDg35WHBgtUsGBARQaMBWUQoKAhhguQBX9g0GFGAoMW
IDdIGXMhABAyO0D0cDgChgMBZ9q4iUNnBwceCBsskPNmDpw0aNaUibKCxMEadtSwSQOFiZIiWMCU
OGiizhMnTZYkERKECw4aB1lY2aIEyZAgQbJU6eAwhRgieI14oRLi4YscX45oCWPjBEgUU7rc+DAD
5T8PInw4DAgAOw==
'''

    _list_remove_gif=\
'''R0lGODlhEAAQAPcAAAAAADRlpFyDtn2m13+o14Oq2Iat2ZCz2pK02pS225++4LTM5bXM5rfO5rvR
57zR58DT6AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAQABAA
AAg/AP8JHEiwoMGDCBMqXMiw4UEBASJKnCiAYAAIDh402NiAgccAFhckSIDggIECBAYoADkQ4sSX
FR3KnEmzJs2AADs=
'''

    _audio_volume_muted_gif=\
'''R0lGODlhEAAQAPcAAAAAAFVXU1dZVVdZVlhaVlhaV1tdWVxeWl1fW11fXF9hXmFjX2NlYWRlY6QA
AKUEBKYHB6sTE68bG8MAAMYAAMYBAccHB8gAAMwAAMwBAc0HB8kODs4KCs4MDM4NDcoTE9AXF9AY
GNEaGtMjI9MmJtQrK9UuLttMTN1ZWc93d5ycmqCinqenpaiopqmrp6mrqLW3s7m7t76/vOyiosLD
wMbHxMfIxcjIxszMys3Oy8/QzdHSz9LS0NPU0dfX1fLBwfTMzPXR0fbT0+Xl4/nh4fnk5Pvr6/vs
7Pf39v319f329vj4+Pr6+fr6+v78/P79/f///wAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAQABAA
AAiuAP8JHEiwoMGDCAUqSFiwQYCECQgwYNFARoGEBYbsKGADyYGEBJg0IbAECgKQOnoQ4OHjooQI
MGFK+EeARg0COTZCmGBhA4UPFSZAIAAjhoAbOAo40PBDCIkiMzI4IOBihYAXNZSWSOLkyRMjIaYO
OKBiQQulGE44gfLEBAYHBRcEcCCiyBMnToJ4gFvQgIMQRI6gUAKkA9+CDy6AGIHBBIcLDw6meOCg
cuUHKQICADs=
'''
    _media_record_small_gif=\
'''R0lGODlhEAAQAPcAAAAAANUeHtIgINUhIdUiItQjI9YmJtghIdorK9osLNotLdYzM9c6OtsxMdwz
M90+PuMcHOUdHeUeHuYfH+cgIOchIeQlJeghIekiIuojI+okJOslJewlJewmJu4nJ+kuLuA8POA+
PuE/P+c/P+s3N+w1Nek5Oek7O9pBQdpERN5hYeNHR+pCQuhFRelPT+dXV+tTU+pXV+xWVupcXOpf
X+xYWO1aWvJTU/BUVPJaWuBmZuFnZ+pgYOpiYutjY+tkZOlmZutnZ+9hYetqautsbOttbetubuN9
feZ9fe5wcO55ee58fPNjY/FkZPBlZfFoaPNpafRxcfV0dPV2dvJ5eeeDg+6BgeqUlOmYmPKGhvSD
g/CIiPGKivOMjPSRkfWYmPWamviTk+yiou+xsfeiovihofC6uvG7u/K+vvHAwPPBwfXLy/XR0fbS
0vbV1frm5vrs7Prt7fvx8f319fz29v75+f78/P/+/gAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAQABAA
AAi2AP8JHEiwoMGDCP/RGYNEBRY3CdUUANLli5IHO+YYROOgDJgtVriQScLgDkE5AsAsMVKEyJAg
Xl5cISiGR5YePn70oDGDRpgBBHdQiTGiRYwaMmC40KIgzsAUUyxAkFDhwwkWJ56EYDPwiBATEShg
yJDhwgQoB+oMNCOiiYQLGjpwyEAiyoKCKJzg0LDBQ4cSURKsKfiGgI0oOW4wkdKgysE5OgKAWIHA
QJqE/+y0OQMHs+d/AQEAOw==
'''

    _list_add_gif=\
'''R0lGODlhIAAgAPcAAAAAADRlpDhopjpppztrpzxppz1rp0BuqWmY0Wua0Wya0W6c0HKfz3Cd0HGe
0HOgz3yl14Go2ISr2Yiu2Imv2Ymu2oyx2Yyw2o6y2pCz2pC02pG025O23JS225W33Je43Je43Zq6
3Zy73Zy735y83Z283p2835++3p++36LA4KXC4anF4qzH47PL5bTM5rjO57nP6LvQ6L7S6cHU6sPX
6sPW68XX68bY7AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAjeAP8JHEiwoMGDCBMqXMiwocOHECNKnEixosEAGDNaJHjghscbAjYO7HjiBEiRAkmaDIlS5cmW
N0qUoMFSZIGYJWbUhJgR4wABAmyUMKHTgNGjARgGsMG0aQ0aJkzIiAHjxQsXLlokXRhgBgmZIkKA
8OBhA4YKFSREiABhxFaFAVg8eMCAgYMGC/IqUJAAgV8EE94mjMuCxQoVKlKgWKBg7AYNFixMCKy0
Z8YSez1Y1iiRQAi+GwRbJPChLwbRFQl08GsBNUUCGvxScD2RQIYLk2lL3Iyyt+/fwIMLTxgQADs=
'''

    _preferences_system_gif=\
'''R0lGODlhIAAgAPcAAAAAACZPiilSiilSiyxVjC5UjSxUji1Vjy9Xjy5WkDVWgjhXgDhYgDpZgTxc
hzJakzVclDRdlThelzlhlj9jkDlimj9mnz5noD9ooEZkjUNkkEZom0dqnElqk05tlEhqnVRyl1Jx
mVRzm1Z0nFV1nlh2nENpoUVqoEZroktypE1xpU91qVF2qFd4pVZ5plB5rVZ6qld6q1h5pFt/r1N6
sFR8sWF1i2l4jGJ/q1eCuF2CtFqEulqEu1yHvmmDqGOGtGOIuW2Rv2uRwW2RwG+Wx3CUwnSWxXaZ
xnKYyHSayXueynmey3+hy4iKhY2Pi4ePnY+QjI+RjYuSmoiQno2RmZCSjZGSjpKTj5KUjpKUj5OU
kJSVkJaYlJeYlZialZialpmbl5uemZuempydmJ2empGXpZGYpZObp5KcqJOdqZeeqp+gnJ+gnY6i
vJmgrZyksJ2lsZylsp+msZ2mtaGjn6GhoaKkoKOkoaSmoqGmr6aopaeqpqmppqiqp6usqaysrKGn
sqOpsqKqtKSrtaestKSstaWtt6eut6OtuaOtu6qvua+xrauzvqywuK+0vK60vrCwrrCyrre3t7G2
vre4tbi5trm6uLu8ubu8ury9u7y8vL6+vIagxI+kwoGl0YSm0Y6r0I6r0ZGv1JOw1Je02Kizwaq0
wKS93LK6w7S5wbe8w7e8xLm9xrq+x7u/yL/AvqrA3KzD37vDzr7Cyb/CyrrE0LzI1r7K1r/L2r/N
2rTI4cDAwMHCwMLCwsTFwcXFxcnKyMvLy8zMzM3Nzc3Ozc7Pzs/Pz8HI0MLL1MTK0MDM2cLP3cbP
2crO0sbS387S1tDR0NPT09PU0tPU09TV0tXV1NXV1dXW1NXW1dbW1tfX19fY1tfY19jY19jY2NnZ
2dna2dra2tvb29vc2tzc3N3d3d7e3t/f38LT5+Dg4OLi4uTk5OXl5efn5+jo6Onp6erq6uvr6+zs
7PDw8PLy8vPz8/T09PX19QAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAj/AP8JHEiwoMGDCBMqXMiwocMxX/Y4dNjED7lieMQsfKLGIZZi8N5h6oNwVRpbpcxQfMfuHZuD
huLgWrYs0ZOGUNqxU7cpU0E3iJzRzIUGYZOCTeS9WwcsUsEyzJTZujXn0UEvlaoM1PKv3jx5vC4V
fIKsVi1TZxDiqdatiVtq9OjJm6eHEkE4qI4xEjQloa9K7+IJjueunLh04LYMBLQqmRs5tBZ2YUcY
Xrx28Lz96eUty79CrZq9meWQS7h79uzBG6dN0y5h3aSoehbI1cR/XdZE0lOlibRhw4TlSfVMEKvb
B7VcM5Zn0rNBVJAfbGMDjqNnhxoB+yKdYAJOMQrI8VJEKBu7V2G6/0vABN2oFQTeeBP3zZ0lO9IN
HCHSI0cQCTds8w054byzCB+3HTBEEjxo0MAPEfyTBTfhkCPOO31A4pABRiSxQwcMkGBCCwJlkQ05
5JjTzh12KYQDBEogUYQHDYiAAgcEVYGNYemwQ8YmCRGAwSdLgAICAyGc8IFBWVhzTjnmlAMGQg/Q
4IkosJTAgAcpbICQFdN8E4wk0URhUCcXkBKLLjIskAELCShkBTSa1PELHQepcEooLzDgwAwHMHQF
McFAY8VBCAAhBAUKwGCBDw050QUUCQ2gwwgu1CCAegcJMEEFAXAq6qicBgQAOw==
'''

    _media_record_gif=\
'''R0lGODlhIAAgAPcAAAAAAM0EBMwFBc0GBs4JCc4UFNAPD9AQENASEtATE9AUFNAXF9AcHNEdHc8p
KdYgINciItQnJ9UtLdgiItYwMNI9Pdk0NNk2NuIaGuMbG+McHOMeHuMfH+QcHOUdHeUeHuYfH+Yg
IOciIuUkJOYqKuYrK+ctLeghIekiIukjI+skJOolJeonJ+wlJewmJuwnJ+4oKOc0NOU9Pek3N+w0
NNdCQtFOTtlCQtdbW9ZcXNtaWtxaWtpgYNxhYedGRulBQelEROpHR+hJSelKSulLS+pNTehOTulP
T+xISO9NTetQUOtTU+lVVepWVuxXV+pYWOlZWepaWulbW+tcXOldXepeXupfX/BOTvBaWvFbW/Jd
XfJeXulgYOphYepjY+pkZOplZetnZ+tpaepqaupra+psbOptbetvb+xpae1qautwcOtxcepycu1w
cOx1dex3d+14eO16eu97e+5+fu5/f/BhYfJra/NsbPRubvZ1dfV4ePZ5ee6Dg/aHh/OKivKPj/WK
ivOQkPKVlfWTk/WUlPWbm/ednfajo/enp/mpqfmwsPm0tPq1tQAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAj/AP8JHEiwoMGDCBMqXMiwocOHECNKnEixokEbBQYIKGDDokAHF+ggYrToUBwLDiommKNIEB84
b+b8SQQnwUQFgQq5YbNGzRkzY8oM8lMgYoU2hMaQWTpGTBgwXrj4SVMBogRDX7x4+QImaxcrVKJA
IRThYQ40cqRQqcK2yhQpUZ4ckeHECA6HDABxYdLkiV8nS4DEGJEBAwk7DBwS6ONDyJAgP2aQCAHi
w4cOGTjoIaBYz4YMHT6ACHGiNOXLITY7bGCnhAYPIE6gSEEbxQkQHkzYaeCwx+PQIWavaLEiBYoQ
H5AQ4fHwAR7KslW4eOFChW0ReB5ArDEES+wUK1zATHBR/EQWIjciIqijZQUKFS1ctFDBYksdBBMN
HMmThAaLFzRckYcSB1REAQRF3LHHHncUMQEFHv2zwwIBBLCADhFmqOGGHHbo4YcRBQQAOw==
'''

    _media_playback_start_gif=\
'''R0lGODlhIAAgAPcAAAAAAEZHQ0pLR0pMR1BRTVNUUFVWUlhZVllaVltcV2BhXGFiX2ZnYmZmY2ho
ZGhqZGtsZ25wbHBybHFybnR0cXV2cnt9enx8e3+Ae4CCfYCCfoOFfoKEgIWFgoaGhIyOjI+Qi5eY
lJeYlZmbmJucmp2fm6GioKiopqusqre3trm6t7q7uLu8ur29vMHCwczMy9DRz9LT0dfY1uHh3+Ll
4OPm4eTm5Obo5efq5ujq5unr5+rr6uns6Ort6evt6uvt6+vu6uzu6uzu6+3v7O3u7e3w7O7w7e7w
7u/x7/Dx7/Dy7/Hx8PHy8PL08fP08vT18/T09PT29PX29fb39vf49vf49/j4+Pn6+fr7+v3+/QAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAACH5BAEAAP8ALAAAAAAgACAA
AAjuAP8JHEiwoMGDCBMqXMiwocOHECNKnKiQA8WFARp4uIgwABYXB0hwLBjgCRAhJRaMHFhSh44i
NyxQWNnSJY8jMRZ84FjTpY4eR1YguBggSo8ePpL66DHkhwgHEwNIGWKkqtWqTGxMuBAxQBUlTJgo
QTIkCI8bNWgQgVHAxEOpOGzymKsjxw0eS1IIgDigig8hRo4kUZLkyBEoMh6MiEjAihEmTqJImTLl
yg4NHSYasJKkSZQqWLJAQVHhYgLHkKlYeRGBBUcFVXoYkTIDw4mVDKQcIRICxEqBEK6o2PB7oIQM
LYorX868ufPn0BMGBAA7
'''
    
    def __init__(self):
        self._list_add = None
        self._list_remove = None
        self._preferences_system = None
        self._media_record = None
        self._media_record_small = None
        self._media_playback_start = None
        self._audio_volume_muted = None
        self._edit_clear = None
        self._midi_connected_merged = None
        self._midi_connected = None
        self._midi_disconnected = None
    
    def _get_image(self, imageName):
        attributeName = "_%s" % imageName.replace('-', '_')
        image = getattr(self, attributeName)
        if image is None:
            image = self._make_image(imageName)
            setattr(self, attributeName, image)
        return image
    
    def _make_image(self, which):
        attributeName = '_%s_gif' % which.replace('-', '_')
        return PhotoImage(data=getattr(self, attributeName))

    audio_volume_muted = property(fget=lambda self:self._get_image("audio-volume-muted"))
    edit_clear = property(fget=lambda self:self._get_image("edit-clear"))
    list_add = property(fget=lambda self:self._get_image("list-add"))
    list_remove = property(fget=lambda self:self._get_image("list-remove"))
    media_playback_start = property(fget=lambda self:self._get_image("media-playback-start"))
    media_record = property(fget=lambda self:self._get_image("media-record"))
    media_record_small = property(fget=lambda self:self._get_image("media-record-small"))
    midi_connected = property(fget=lambda self:self._get_image("midi-connected"))
    midi_connected_merged = property(fget=lambda self:self._get_image("midi-connected-merged"))
    midi_disconnected = property(fget=lambda self:self._get_image("midi-disconnected"))
    preferences_system = property(fget=lambda self:self._get_image("preferences-system"))
    
class AboutDialog(tkSimpleDialog.Dialog):
    def __init__(self, parent):
        tkSimpleDialog.Dialog.__init__(self, parent, 
            title="About %s" % appTitle)
    
    def body(self, master):
        label = Label(master, text=aboutText, anchor=W, justify=LEFT)
        label.pack(side=TOP, fill=BOTH)

class DeviceMapCanvas(Canvas):
    def __init__(self, master, **kwargs):
        Canvas.__init__(self, master, **kwargs)

    def addConnection(self, fromCoords, toCoords, key):
        # add dotted line between the coords, save in self._shapes
        line = self.create_line(fromCoords[0], fromCoords[1], toCoords[0], toCoords[1],
            tags=key, dash=(2,2))

    def addMergedConnection(self, fromCoords, toCoords, key):
        # add solid line between the coords, save in self._shapes
        self.create_line(fromCoords[0], fromCoords[1], toCoords[0], toCoords[1], 
            tags=key)

    def removeConnection(self, key):
        # remove the specified shape from self._shapes and diagram
        self.delete(key)


class MapDevicesDialog(tkSimpleDialog.Dialog):
    """
    Present an interface to map input from Midi input devices to Midi
    output devices.
    """
    def __init__(self, parent, imageProvider, deviceMap):
        # have to set this before invoking __init__ from the base class,
        # because the __init__ invokes the body method, which needs to
        # read values from the device map
        self._deviceMap = deviceMap
        self._additions = {}
        self._removals = []
        self._currentInputSelection = []
        self._currentOutputSelection = []
        self._imageProvider = imageProvider
        tkSimpleDialog.Dialog.__init__(self, parent, title="Map Devices")
    
    def _connectionKey(self, inputName, outputName):
        return ('%s+%s' % (inputName, outputName)).replace(' ', '')

    def _enableMappingButtons(self):
        inputName = self._getListSelection(self._inputDeviceList)
        outputName = self._getListSelection(self._outputDeviceList)
        buttons = [self._connectButton, self._connectMergeButton, 
            self._disconnectButton]
        if inputName is None or outputName is None:
            [button.config(state=DISABLED) for button in buttons]
        else:
            toBeConnected = (inputName, outputName) in self._additions
            canConnect = self._deviceMap.canMap(inputName, outputName) and \
                not toBeConnected
            enabledState = canConnect and NORMAL or DISABLED
            [button.config(state=enabledState) for button in buttons[0:2]]
            enabledState = (self._deviceMap.mappingExists(inputName, 
                outputName) or toBeConnected) and NORMAL or DISABLED
            self._disconnectButton.config(state=enabledState)
    
    def _getListSelection(self, listBox):
        selectedIndexes = listBox.curselection()
        if selectedIndexes:
            return listBox.get(int(selectedIndexes[0]))
    
    def _onConnect(self, merge=False):
        inputName = self._getListSelection(self._inputDeviceList)
        outputName = self._getListSelection(self._outputDeviceList)
        self._additions[(inputName, outputName)] = merge
        self._showMapping(inputName, outputName, merge)
        self._enableMappingButtons()
        
    def _onDisconnect(self):
        inputName = self._getListSelection(self._inputDeviceList)
        outputName = self._getListSelection(self._outputDeviceList)
        if (inputName, outputName) in self._additions:
            del self._additions[(inputName, outputName)]
        else:
            self._removals.append((inputName, outputName))
        self._unshowMapping(inputName, outputName)
        self._enableMappingButtons()

    def _pollListboxes(self):
        inputSelection = self._inputDeviceList.curselection()
        outputSelection = self._outputDeviceList.curselection()
        if inputSelection != self._currentInputSelection or \
            outputSelection != self._currentOutputSelection:
            self._enableMappingButtons()
            self._currentInputSelection = inputSelection
            self._currentOutputSelection = outputSelection
        self.after(250, self._pollListboxes)
            
    def _showMapping(self, inputName, outputName, merged):
        inputIndex = MidiInput.devices().index(inputName)
        outputIndex = MidiOutput.devices().index(outputName)
        # bbox doesn't give accurate y coordinate, so we have to calculate
        # these from the height of the first item plus some margin allowance
        inputItemRect = list(self._inputDeviceList.bbox(0))
        inputItemRect[1] += inputIndex * (1 + inputItemRect[3])
        outputItemRect = list(self._outputDeviceList.bbox(0))
        outputItemRect[1] += outputIndex * (1 + outputItemRect[3])
        inputCoords = (0, inputItemRect[1] + (inputItemRect[3] / 2))
        outputCoords = (self._deviceMapCanvas.winfo_width(), 
            outputItemRect[1] + (outputItemRect[3] / 2))
        mapKey = self._connectionKey(inputName, outputName)
        if merged:
            self._deviceMapCanvas.addMergedConnection(inputCoords, 
                outputCoords, mapKey)
        else:
            self._deviceMapCanvas.addConnection(inputCoords, outputCoords,
                mapKey)
                    
    def _showMappings(self, master):
        master.update_idletasks()
        self._inputDeviceList.see(0)
        self._outputDeviceList.see(0)
        for inputName in MidiInput.devices():
            deviceMapInfo = self._deviceMap.getMapping(inputName)
            if deviceMapInfo:
                for outputName, merged in deviceMapInfo:
                    self._showMapping(inputName, outputName, merged)

    def _sizeListboxes(self, listboxes):
        width = max([len(item) for item in \
            listboxes[0].get(0, END) + listboxes[1].get(0, END)])
        height = max([listboxes[0].size(), listboxes[1].size()])
        [listbox.config(width=width, height=height) for listbox in listboxes]

    def _unshowMapping(self, inputName, outputName):
        self._deviceMapCanvas.removeConnection(self._connectionKey(inputName,
            outputName))

    def apply(self):
        for devices, merged in self._additions.items():
            self._deviceMap.addMapping(devices[0], devices[1], merged)
        for inputName, outputName in self._removals:
            self._deviceMap.removeMapping(inputName, outputName)

    def body(self, master):
        self._buttonPanel = Frame(master)
        self._connectMergeButton = Button(self._buttonPanel, text="Merge", 
            image=self._imageProvider.midi_connected_merged,
            command=lambda:self._onConnect(merge=True))
        self._connectMergeButton.pack(side=LEFT)
        self._connectButton = Button(self._buttonPanel, text="Connect", 
            image=self._imageProvider.midi_connected,
            command=lambda:self._onConnect(merge=False))
        self._connectButton.pack(side=LEFT)
        self._disconnectButton = Button(self._buttonPanel, text="Disconnect", 
            image=self._imageProvider.midi_disconnected,
            command=self._onDisconnect)
        self._disconnectButton.pack(side=LEFT, padx=4)
        self._buttonPanel.grid(row=0, column=1)
        
        label = Label(master, text="Input Devices")
        label.grid(row=1, column=0, sticky=W)
        self._inputDeviceList = Listbox(master, exportselection=0)
        [self._inputDeviceList.insert(END, item) for item in MidiInput.devices()]
        self._inputDeviceList.grid(row=2, column=0, sticky=N+S)

        label = Label(master, text="Output Devices")
        label.grid(row=1, column=2, sticky=W)
        self._outputDeviceList = Listbox(master, exportselection=0)
        self._outputDeviceList.grid(row=2, column=2, sticky=N+S)
        [self._outputDeviceList.insert(END, item) for item in MidiOutput.devices()]
        
        self._sizeListboxes([self._inputDeviceList, self._outputDeviceList])
        
        self._deviceMapCanvas = DeviceMapCanvas(master, width=1, height=1)
        self._deviceMapCanvas.grid(row=2, column=1, sticky=W+E+N+S)
        self._showMappings(master)
        self._enableMappingButtons()

        self._pollListboxes()

class ScrolledCanvasFrame(Frame):
    """
    A scrollable container.
    """
    def __init__(self, master, bd=2, relief=SUNKEN):
        Frame.__init__(self, master, bd=bd, relief=relief)

        self.grid_rowconfigure(0, weight=1)
        self.grid_columnconfigure(0, weight=1)

        self.xscrollbar = Scrollbar(self, orient=HORIZONTAL)
        self.xscrollbar.grid(row=1, column=0, sticky=E+W)

        self.yscrollbar = Scrollbar(self)
        self.yscrollbar.grid(row=0, column=1, sticky=N+S)

        self.canvas = Canvas(self, bd=0,
            xscrollcommand=self.xscrollbar.set,
            yscrollcommand=self.yscrollbar.set)

        self.canvas.grid(row=0, column=0, sticky=N+S+E+W)

        self.xscrollbar.config(command=self.canvas.xview)
        self.yscrollbar.config(command=self.canvas.yview)
        
        self.contentFrame = Frame(self.canvas)
        self._configureContentFrame()
        
    def _configureContentFrame(self):        
        self.canvas.create_window(0, 0, window=self.contentFrame, 
            anchor='nw', tags="content")

    def adjustContentDimensions(self):        
        self.contentFrame.update_idletasks()
        self.contentFrame.config(width=self.contentFrame.winfo_reqwidth(),
            height=self.contentFrame.winfo_reqheight())
        self.canvas.config(scrollregion=(0, 0, self.contentFrame.winfo_width(), 
            self.contentFrame.winfo_height()))

class SequencerStop(object):
    """
    A sentinel object that represents the final input from a background
    process.
    """
    pass

_stopMsg = SequencerStop()

EVT_TRACK_FEEDBACK = "<<Track Feedback>>"

class SequencerApp:
    """
    A simple multitrack Midi sequencer.
    """
    def __init__(self, root):
        root.title("Super Simple Sequencer")

        self._root = root
        self._root.minsize(width=800, height=600)
        
        self._imageProvider = ImageProvider()
        self._trackFrames = []
        
        self._midiEventQueue = None
        
        # create variables bound to entry widgets
        self._recording = IntVar()
        self._playing = IntVar()
        self._tempo = StringVar()
        
        self._makeMenuBar(root)
        self._makeToolBar(root)
        self._bindEvents(root)
        
        self._clientCanvasFrame = ScrolledCanvasFrame(root)
        self._trackGridFrame = self._clientCanvasFrame.contentFrame
        self._clientCanvasFrame.pack(side=TOP, fill=BOTH, expand=1)
        
        self._sequencer = MidiSequencer()
        self._sequencer.start()
        # having started Midi, ensure that the application stops it on exit
        root.protocol("WM_DELETE_WINDOW", self._onClose)
        
        self._tempo.set(str(self._sequencer.tempo))
        
        self._bufferSize = 32 * 384 # start with 32 bars of 4/4 time

        # add one track to start with
        self._onAddTrack()

        return
    
    def _bindEvents(self, root):
        root.bind("<space>", self._onTogglePlayback)
        root.bind("r", self._onToggleRecording)
        root.bind("a", self._onAddTrack)
        root.bind(EVT_TRACK_FEEDBACK, self._processMidiEvents)
    
    def _makeMenuBar(self, root):
	    menubar = Menu(root)
	    # create a pulldown menu, and add it to the menu bar
	    filemenu = Menu(menubar, tearoff=0)
	    filemenu.add_command(label="Open", command=notYet)
	    filemenu.add_command(label="Save", command=notYet)
	    filemenu.add_separator()
	    filemenu.add_command(label="Exit", command=self._onClose)
	    menubar.add_cascade(label="File", menu=filemenu)
	    
	    helpmenu = Menu(menubar, tearoff=0)
	    helpmenu.add_command(label="About", command=self._onAbout)
	    menubar.add_cascade(label="Help", menu=helpmenu)
	    # display the menu
	    root.config(menu=menubar)

    def _makeToolBar(self, root):
        toolbar = Frame(root)
        # Add Track
        self._addTrackButton = Button(toolbar, text="Add Track",
            image=self._imageProvider.list_add, command=self._onAddTrack)
        self._addTrackButton.pack(side=LEFT, padx=2, pady=2)
        # Map Devices
        self._mapDevicesButton = Button(toolbar, text="Map Devices",
            image=self._imageProvider.preferences_system, 
            command=self._onMapDevices)
        self._mapDevicesButton.pack(side=LEFT, padx=2, pady=2)
        # use a frame to effect separation of buttons into groups
        playRecordFrame = Frame(toolbar)
        # Record
        self._recordButton = Checkbutton(playRecordFrame, text="Record", 
            indicatoron=0, image=self._imageProvider.media_record,
            command=self._onToggleRecording, variable=self._recording)
        self._recordButton.pack(side=LEFT, padx=2, pady=2)
        # Play
        playButton = Checkbutton(playRecordFrame, text="Play", indicatoron=0,
            image=self._imageProvider.media_playback_start, 
            command=self._onTogglePlayback, variable=self._playing)
        playButton.pack(side=LEFT, padx=2, pady=2)
        # Set Tempo
        tempoEntry = Spinbox(playRecordFrame, width=4, from_=1, to=255, 
            command=self._onSetTempo, textvariable=self._tempo)
        tempoEntry.pack(side=LEFT, padx=2, pady=2)
        playRecordFrame.pack(side=LEFT, padx=8)
        toolbar.pack(side=TOP, fill=X)
    
    def _onAbout(self):
        AboutDialog(self._root)
    
    def _onClose(self):
        # Stop all Midi activity and close the application.
        if not self._playing.get():
            self._sequencer.stop()
            self._root.destroy()
            
    def _onAddTrack(self, event=None):
        if not self._playing.get():
            trackNumber = len(self._trackFrames) + 1
            track = self._sequencer.sequence.appendTrack(channel=1, 
                record=True)
            trackFrame = TrackFrame(self._trackGridFrame, self._imageProvider,
                track, trackNumber, self._bufferSize, self._onDeleteTrack)
            self._trackFrames.append(trackFrame)
            self._clientCanvasFrame.adjustContentDimensions()
    
    def _onDeleteTrack(self, trackFrame):
        renumberFrom = trackFrame.getTrackNumber() - 1
        del self._sequencer.sequence[renumberFrom]
        trackFrame.forget()
        self._trackFrames.remove(trackFrame)
        for trackIndex in range(renumberFrom, len(self._trackFrames)):
            self._trackFrames[trackIndex].setTrackNumber(trackIndex + 1)
        self._clientCanvasFrame.adjustContentDimensions()

    def _onFeedback(self, messages):
        stillGoing = (self._recording.get() and self._playing.get()) or \
            (self._playing.get() and self._sequencer.playing)
        if not stillGoing:
            messages.append(_stopMsg)
        if messages:
            self._midiEventQueue.put(messages)
            self._root.event_generate(EVT_TRACK_FEEDBACK, when='tail')
        return stillGoing
    
    def _onMapDevices(self):
        MapDevicesDialog(self._root, self._imageProvider, 
            self._sequencer.deviceMap)
    
    def _onSetTempo(self):
        self._sequencer.tempo = int(self._tempo.get())

    def _onTogglePlayback(self, event=None):
        if event:
            self._playing.set(not self._playing.get())
        playing = self._playing.get()
        recording = self._recording.get()
        if playing or recording:
            self._midiEventQueue = Queue.Queue()
        if recording:
            if playing:
                self._sequencer.startRecording(feedbackHandler=self._onFeedback)
            else:
                # Play was just toggled off, with recording on
                self._sequencer.stopRecording()
                # Turn off the Recording button too
                self._toggleRecordingButton()
        else:
            # not recording, just toggling playback
            if playing:
                self._sequencer.startPlayback(feedbackHandler=self._onFeedback)
            else:
                self._sequencer.stopPlayback()
        self._setControlEnabledState(playing)
    
    def _onToggleRecording(self, event=None):
        if event:
            self._toggleRecordingButton()
    
    def _processMidiEvents(self, event):
        # process events queued by background processor
        while self._midiEventQueue.qsize( ):
            messages = self._midiEventQueue.get()            
            for queuedInput in messages:
                if isinstance(queuedInput, SequencerStop):
                    if self._playing.get():
                        # The UI says we're playing, but the sequencer is done.
                        self._togglePlayingButton()
                elif not hasattr(queuedInput, "track"):
                    # the event is for all tracks
                    if isinstance(queuedInput, TickMsg) and \
                        (queuedInput.clocks + 1) > self._bufferSize:
                        self._resizeBuffers()
                    for frame in self._trackFrames:
                        frame.onMidiInput(queuedInput)
                else:
                    self._trackFrames[queuedInput.track].onMidiInput(queuedInput.msg)
    
    def _resizeBuffers(self, addedSize = 1536):
        # default value adds room for 8 bars of 4/4 time
        self._bufferSize += addedSize
        for frame in self._trackFrames:
            frame.resizeBuffer(self._bufferSize)
        self._clientCanvasFrame.adjustContentDimensions()
    
    def _setControlEnabledState(self, playing):
        # various controls should be disabled during playback/recording
        enabledState = playing and DISABLED or NORMAL
        self._recordButton.config(state=enabledState)        
        self._addTrackButton.config(state=enabledState)
        self._mapDevicesButton.config(state=enabledState)
        for frame in self._trackFrames:
            frame.inPlayback(playing)
    
    def _togglePlayingButton(self):
        self._playing.set(not self._playing.get())
        self._onTogglePlayback()
        
    def _toggleRecordingButton(self):
        self._recording.set(not self._recording.get())


class TrackFrame(LabelFrame):
    """
    Display controls for recording one track in the sequencer.
    """
    def __init__(self, parent, imageProvider, track, trackNumber, totalClocks,
        deleteCallback):
        """
        Create a TrackFrame instance.
        
        @param parent: the parent widget for the TrackFrame
        @param track: the L{SequencerTrack} for the track's Midi content
        @param trackNumber: the 1-based number of the track
        @param totalClocks: the initial "buffer size" in Midi clocks for 
            the track events (can be resized as needed)
        @param deleteCallback: the function to invoke to delete the track
            frame and its underlying Midi content
        """
        LabelFrame.__init__(self, parent)
        self._track = track
        self.setTrackNumber(trackNumber)
        self._deleteCallback = deleteCallback
        self._armedForRecording = IntVar()
        self._armedForRecording.set(track.recording)
        self._muted = IntVar()
        self._muted.set(track.mute)
        self._clocksPerX = 2
        self._addControlBox(imageProvider)
        self._addEventCanvas(totalClocks)
        self.pack(side=TOP)
    
    def _addControlBox(self, imageProvider):
        controlBox = Frame(self)
        self._addControls(controlBox, imageProvider)
        controlBox.pack(side=LEFT, anchor=N)
    
    def _addControls(self, master, imageProvider):
        row1 = Frame(master)
        row1.pack(side=TOP, anchor=W)

        row2 = Frame(master)
        row2.pack(side=TOP, anchor=W, pady=2)
        
        self._recordButton = Checkbutton(row1, text="Record", 
            image=imageProvider.media_record_small, indicatoron=0, 
            variable=self._armedForRecording, 
            command=self._onToggleTrackRecording)
        self._recordButton.pack(side=LEFT)
        self._muteButton = Checkbutton(row1, text="Mute", 
            image=imageProvider.audio_volume_muted,
            indicatoron=0, variable=self._muted, 
            command=self._onToggleTrackMuted)
        self._muteButton.pack(side=LEFT)

        self._eraseButton = Button(row1, text="Erase", 
            image=imageProvider.edit_clear, command=self._onEraseTrack)
        self._eraseButton.pack(side=LEFT, padx=4)
        self._deleteButton = Button(row1, text="Delete", 
            image=imageProvider.list_remove, command=self._onDeleteTrack)
        self._deleteButton.pack(side=LEFT)
        
        Label(row2, text="Ch:").pack(side=LEFT, padx=2)
        self._channelButton = Spinbox(row2, width=3, from_=1, to=16, 
            command=self._onSetTrackChannel)
        self._onSetTrackChannel()
        self._channelButton.pack(side=LEFT)

    def _addEventCanvas(self, totalClocks, totalPitches=120):
        self._eventCanvasBorderOffset = 2
        self._totalPitches = totalPitches
        timelineSize = (self._eventCanvasBorderOffset + \
            (totalClocks / self._clocksPerX), 
            self._eventCanvasBorderOffset + totalPitches)
        self._eventCanvas = Canvas(self, width=timelineSize[0], 
            height=timelineSize[1], bg="white", bd=0)
        self._eventCanvas.pack(side=LEFT, anchor=N)
        self._eventCanvas.create_rectangle(self._eventCanvasBorderOffset, 
            self._eventCanvasBorderOffset, 
            timelineSize[0], timelineSize[1] + 1, outline="dark blue", 
            tags=("border"))
        self._eventCanvas.create_line(-1, -1, -1, -1, fill="maroon", 
            tags="tick")
    
    def _getLineCoordsAtClock(self, clock):
        lineX = (clock / self._clocksPerX) + 1
        lineY = self._eventCanvasBorderOffset + 1
        return (lineX, lineY, lineX, lineY + self._totalPitches)        
    
    def _onDeleteTrack(self):
        self._deleteCallback(self)
    
    def _onEraseTrack(self):
        self._track.erase()
        self._eventCanvas.delete("note || barline")
    
    def _onSetTrackChannel(self):
        self._track.channel = int(self._channelButton.get())
    
    def _onToggleTrackMuted(self):
        self._track.mute = bool(self._muted.get())
    
    def _onToggleTrackRecording(self):
        self._track.recording = bool(self._armedForRecording.get())
    
    def _moveTick(self, clock):        
        x, y, x2, y2 = self._getLineCoordsAtClock(clock)
        self._eventCanvas.coords("tick", x, y, x2, y2)
    
    def inPlayback(self, playing):
        enabledState = playing and DISABLED or NORMAL
        self._recordButton.config(state=enabledState)
        self._muteButton.config(state=enabledState)        
        self._channelButton.config(state=enabledState)
        self._eraseButton.config(state=enabledState)
        self._deleteButton.config(state=enabledState)

    def onMidiInput(self, event):
        """
        Represent the track's Midi input events as a piano-roll style
        rendering.
        """
        if isinstance(event, TickMsg):           
            self._moveTick(event.clocks)
        elif isinstance(event, SequencedNote):
            noteX = round(event.clocks / self._clocksPerX) + \
                self._eventCanvasBorderOffset
            noteX2 = noteX + round(event.duration / self._clocksPerX)
            # pitches go from low to high; 0 - 119
            # canvas coordinates go from high to low
            noteY = ((self._totalPitches -1) - event.pitch) + \
                self._eventCanvasBorderOffset
            noteY2 = noteY
            self._eventCanvas.create_rectangle(noteX, noteY, noteX2, noteY2,
                tags="note")
        elif isinstance(event, NewBarMsg):
            x, y, x2, y2 = self._getLineCoordsAtClock((1 + event.bar) * \
                event.clocksPerBar)
            self._eventCanvas.create_line(x, y, x2, y2, tags="barline",
                fill="dark grey")
        
    def resizeBuffer(self, newSize):
        """
        Resize the event canvas.
        
        @newSize: the new buffer size, in Midi clocks
        """
        newWidth = (newSize / self._clocksPerX)
        self._eventCanvas.config(width=newWidth)
        self._eventCanvas.coords("border", self._eventCanvasBorderOffset, 
            self._eventCanvasBorderOffset, newWidth, 
            self._eventCanvasBorderOffset + self._totalPitches)

    def getTrackNumber(self):
        return self._trackNumber
    def setTrackNumber(self, number):
        self._trackNumber = number
        trackLabel = "Track %d" % number
        self.config(text=trackLabel)
    
def main():
	root = Tk()
	app = SequencerApp(root)
	root.mainloop()
main()












