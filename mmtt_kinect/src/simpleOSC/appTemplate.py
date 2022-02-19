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


import osc

# just importing the osc module creates under the hood an outbound socket and the callback manager
# (osc addressManager). But we dont have to worry about that.


def myTest():
    """ a simple function that creates the necesary sockets and enters an enless
        loop sending and receiving OSC
    """
    osc.init()
    
    inSocket = osc.createListener('127.0.0.1', 9001) # in this case just using one socket
##    inSocket = osc.createListener() # this defaults to port 9001 as well

    # bind addresses to functions -> printStuff() function will be triggered everytime a
    # "/test" labeled message arrives
    osc.bind(printStuff, "/test")

    import time # in this example we will have a small delay in the while loop

    print 'ready to receive and send osc messages ...'
    
    while 1:
##        osc.sendMsg("/test", [444], "127.0.0.1", 9000) # send normal msg to a specific ip and port
        osc.sendMsg("/test", [444]) # !! it sends by default to localhost ip "127.0.0.1" and port 9000 

        # create and send a bundle
        bundle = osc.createBundle()
        osc.appendToBundle(bundle, "/test/bndlprt1", [1, 2, 3]) # 1st message appent to bundle
        osc.appendToBundle(bundle, "/test/bndlprt2", [4, 5, 6]) # 2nd message appent to bundle
##        osc.sendBundle(bundle, "127.0.0.1", 9000) # send it to a specific ip and port

        osc.sendBundle(bundle) # !! it sends by default to localhost ip "127.0.0.1" and port 9000 
        
        osc.getOSC(inSocket) # listen to incomming OSC in this socket

        time.sleep(0.5) # you don't need this, but otherwise we're sending as fast as possible.

        


""" Below some functions dealing with OSC messages RECEIVED to Python.

    Here you can set all the responders you need to deal with the incoming
    OSC messages. You need them to the callBackManager instance in the main
    loop and associate them to the desired OSC addreses like this for example
    addressManager.add(printStuff, "/print")
    it would associate the /print tagged messages with the printStuff() function
    defined in this module. You can have several callback functions in a separated module if you wish
"""

def printStuff(*msg):
    """deals with "print" tagged OSC addresses """

    print "printing in the printStuff function ", msg
    print "the oscaddress is ", msg[0][0]
    print "the value is ", msg[0][2]




if __name__ == '__main__': myTest()














