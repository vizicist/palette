default:
	nmake manifoldbin
	nmake PYTHON=c:\python26 clean install
	nmake PYTHON=c:\python27 clean install

FINALDIR = $(PYTHON)\lib\site-packages\nosuch

manifoldbin:
	copy *.py ..\..\bin\python\nosuch

install : \
	$(FINALDIR) \
	$(FINALDIR)\__init__.py \
	$(FINALDIR)\midiutil.py \
	$(FINALDIR)\midipypm.py \
	$(FINALDIR)\mididebug.py \
	$(FINALDIR)\midiosc.py \
	$(FINALDIR)\midifile.py

$(FINALDIR):
	mkdir $(FINALDIR)

$(FINALDIR)\__init__.py: __init__.py
	copy __init__.py $(FINALDIR)\__init__.py

$(FINALDIR)\midiutil.py: midiutil.py
	copy midiutil.py $(FINALDIR)\midiutil.py

$(FINALDIR)\midipypm.py: midipypm.py
	copy midipypm.py $(FINALDIR)\midipypm.py

$(FINALDIR)\mididebug.py: mididebug.py
	copy mididebug.py $(FINALDIR)\mididebug.py

$(FINALDIR)\midiosc.py: midiosc.py
	copy midiosc.py $(FINALDIR)\midiosc.py

$(FINALDIR)\midifile.py: midifile.py
	copy midifile.py $(FINALDIR)\midifile.py

clean :
	rm -f *~ *.bak *.pyc

dist :
	rm -f *~ *.bak *.pyc
	cd ..
	del /q python_nosuch_midiutil.zip
	zipdir nosuch_midiutil
	mv nosuch_midiutil.zip python_nosuch_midiutil.zip
	tosoft python_nosuch_midiutil.zip
	cd nosuch_midiutil
