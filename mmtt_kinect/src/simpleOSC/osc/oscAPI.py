"""     simpleOSC 0.2
    ixi software - July, 2006
    www.ixi-software.net

    simple API  for the Open SoundControl for Python (by Daniel Holth, Clinton
    McChesney --> pyKit.tar.gz file at http://wiretap.stetson.edu)
    Documentation at http://wiretap.stetson.edu/docs/pyKit/

    The main aim of this implementation is to provide with a simple way to deal
    with the OSC implementation that makes life easier to those who don't have
    understanding of sockets or programming. This would not be on your screen without the help
    of Daniel Holth.

    This library is free software; you can redistribute it and/or
    modify it under the terms of the GNU Lesser General Public
    License as published by the Free Software Foundation; either
    version 2.1 of the License, or (at your option) any later version.

    This library is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the GNU
    Lesser General Public License for more details.

    You should have received a copy of the GNU Lesser General Public
    License along with this library; if not, write to the Free Software
    Foundation, Inc., 59 Temple Place, Suite 330, Boston, MA  02111-1307  USA

    Thanks for the support to Buchsenhausen, Innsbruck, Austria.
"""

import OSC
import socket

# globals
outSocket = 0 # socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
addressManager = 0 # OSC.CallbackManager()


def init():
    """ instantiates address manager and outsocket as globals
    """
    global outSocket, addressManager
    outSocket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    addressManager = OSC.CallbackManager()


def createListener(ipAddr='127.0.0.1', port = 9001):
    """returns a new inbound socket
    defaults to ipAddr='127.0.0.1', port 9001
    """
    l = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    
    try :
        l.bind( (ipAddr, port) )
    except socket.error:
        print 'there was an error binding to ip %s and port %i , maybe the port is already taken by another process?' % (ipAddr. port)
        return 0
    
    l.setblocking(0) # if not this it waits for msgs to arrive blocking other events
##    l.settimeout(0) # does same as line above but avobe only boolean, this takes float
    
    return l


def bind(func, oscaddress):
    """ bind given oscaddresses with given functions in address manager
    """
    addressManager.add(func, oscaddress)


def sendMsg(oscAddress, dataArray=[], ipAddr='127.0.0.1', port=9000) :
    """create and send normal OSC msgs
        defaults to '127.0.0.1', port 9000
    """
    outSocket.sendto( createBinaryMsg(oscAddress, dataArray),  (ipAddr, port))


def createBundle():
    """create bundled type of OSC messages"""
    b = OSC.OSCMessage()
    b.address = ""
    b.append("#bundle")
    b.append(0)
    b.append(0)
    return b


def appendToBundle(bundle, oscAddress, dataArray):
    """create OSC mesage and append it to a given bundle"""
    bundle.append( createBinaryMsg(oscAddress, dataArray),  'b')


def sendBundle(bundle, ipAddr='127.0.0.1', port=9000) :
    """convert bundle to a binary and send it"""
    outSocket.sendto(bundle.message, (ipAddr, port))

##

def createBinaryMsg(oscAddress, dataArray):
    """create and return general type binary OSC msg"""
    m = OSC.OSCMessage()
    m.address = oscAddress

    for x in dataArray:  ## append each item of the array to the message
        m.append(x)

    # get the actual OSC to send
    return m.getBinary()

################################ receive osc from The Other.

def getOSC(inSocket):
    """try to get incoming OSC and send it to callback manager (for osc addresses)"""
    try:
        while 1:
            addressManager.handle( inSocket.recv(1024) ) # self.socket.recvfrom(2**13)
    except:
        return "no data arrived" # not data arrived





if __name__ == '__main__':
    # example of how to use oscAPI
    
##    inSocket = createListener("127.0.0.1", 9001)
    inSocket = createListener() # defaults to "127.0.0.1", 9001

    # add addresses to callback manager
    def printStuff(msg):
        """deals with "print" tagged OSC addresses
        """
        print "printing in the printStuff function ", msg
        print "the oscaddress is ", msg[0]
        print "the value is ", msg[2]

    bind(printStuff, "/test")

    #send normal msg, two ways
    sendMsg("/test", [1, 2, 3], "127.0.0.1", 9000)
    sendMsg("/test2", [1, 2, 3]) # defaults to "127.0.0.1", 9000
    sendMsg("/hello") # defaults to [], "127.0.0.1", 9000

    # create and send bundle, to ways to send
    bundle = createBundle()
    appendToBundle(bundle, "/testing/bundles", [1, 2, 3])
    appendToBundle(bundle, "/testing/bundles", [4, 5, 6])
    sendBundle(bundle, "127.0.0.1", 9000)
    sendBundle(bundle) # defaults to "127.0.0.1", 9000

    # receive OSC
    getOSC(inSocket)







