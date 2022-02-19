

1 => int on;

// init OSC /////////////////////////////////////////
OscSend send; //send OSC
send.setHost( "localhost", 9001);
OscRecv recv; //recv OSC
9000 => recv.port;
recv.listen(); // start listening (launch thread)


// OSC classes and functions ////
class OscListener { // general listener template
    OscEvent e;
                    
    fun void action(){} //to be extended

    fun void bind( string arg ){
            recv.event(arg) @=> e; // bind event to address
            spork ~ oscShred( ); // start listening to incomming osc
    }
                    
    fun void oscShred() {
            while (on){
                    e => now;
                    if ( e.nextMsg()){ action(); }
            }
            //<<<"exiting shred", me>>>;
            //machine.remove( s.id() );
    }
}
/////////////////////////////////////////

// Listeners
class Listener1 extends OscListener{
    fun void action(){ <<<"chuck: listener1", e.getInt()>>>; }
}


// create instance and bind address
Listener1 lis1;
lis1.bind( "/test, i" );



<<<"chuck osc receiver up and runing ...">>>;

//entering main loop //
while (on){ // quits when on is false
	send.startMsg( "/test", "i" );
	1 => send.addInt; // send connected message to graphics
	0.2::second => now; 
}

<<<"quiting ChucK">>>; //print quit

