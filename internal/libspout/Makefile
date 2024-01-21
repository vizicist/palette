OBJ = libspout_go.o SpoutDirectX.o SpoutGLDXinterop.o SpoutGLextensions.o SpoutMemoryShare.o SpoutReceiver.o SpoutSDK.o SpoutSender.o SpoutSenderNames.o SpoutSharedMemory.o

CC = g++
SPOUT2SRC = SpoutSDK/Source

default: libspout.a

%.o : $(SPOUT2SRC)/%.cpp
	$(CC) -c -I$(SPOUT2SRC) -Wno-int-to-pointer-cast -Wno-pointer-arith -Wno-conversion-null $(CFLAGS) $(CPPFLAGS) $< -o $@

libspout_go.o : libspout_go.cpp
	$(CC) -c -I$(SPOUT2SRC) -Wno-int-to-pointer-cast -Wno-pointer-arith -Wno-conversion-null $(CFLAGS) $(CPPFLAGS) $< -o $@

# SpoutCopy.cpp needs to be compiled a bit differently
COPYOBJ = SpoutCopy.o

SpoutCopy.o : $(SPOUT2SRC)/SpoutCopy.cpp
	$(CC) -c -msse4.1 $(CFLAGS) $(CPPFLAGS) $< -o $@

clean:
	-rm -f *.o

clobber:
	-rm -f *.o *.a

libspout.a: $(OBJ) $(COPYOBJ)
	-rm -f libspout.a
	ar q libspout.a $(OBJ) $(COPYOBJ)

