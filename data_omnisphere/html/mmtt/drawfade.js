// By Simon Sarris
// www.simonsarris.com
// sarris@acm.org
//
// Last update December 2011
//
// Free to use and distribute at will
// So long as you are nice to people, etc

// Constructor for Shape objects to hold data for all drawn objects.
// For now they will just be defined as rectangles.
//
//

function Shape(x, y, sz, alpha, fill) {
  // This is a very simple and unsafe constructor. All we're doing is checking if the values exist.
  // "x || 0" just means "if there is a value for x, use that. Otherwise use 0."
  // But we aren't checking anything else! We could put "Lalala" for the value of x 
  this.x = x || 0;
  this.y = y || 0;
  this.sz = sz || 1;
  if ( this.sz < 10 ) {
	  this.sz = 10;
  }
  this.fill = fill || '#AAAAAA';
  this.alpha = alpha;
}

Shape.prototype.fade = function() {
  this.alpha = this.alpha - 0.006;
  if ( this.alpha < 0.0 ) {
	  this.alpha = 0.0;
  }
  return this.alpha;
}
// Draws this shape to a given context
Shape.prototype.draw = function(ctx) {

  var fill = this.fill;
  ctx.fillStyle = fill.replace('[[alpha]]',this.alpha);

  ctx.beginPath();
  ctx.arc(this.x, this.y, this.sz, 0 , 2 * Math.PI, false);
  ctx.lineWidth=4;
  ctx.strokeStyle = ctx.fillStyle;
  ctx.stroke();

  // ctx.fillRect(this.x, this.y, this.sz, this.sz);
}

function CanvasState(canvas) {
  // **** First some setup! ****
  
  this.canvas = canvas;
  this.width = canvas.width;
  this.height = canvas.height;
  this.ctx = canvas.getContext('2d');

  // This complicates things a little but but fixes mouse co-ordinate problems
  // when there's a border or padding. See getMouse for more detail
  // var stylePaddingLeft, stylePaddingTop, styleBorderLeft, styleBorderTop;
  // if (document.defaultView && document.defaultView.getComputedStyle) {
    // this.stylePaddingLeft = parseInt(document.defaultView.getComputedStyle(canvas, null)['paddingLeft'], 10)      || 0;
    // this.stylePaddingTop  = parseInt(document.defaultView.getComputedStyle(canvas, null)['paddingTop'], 10)       || 0;
    // this.styleBorderLeft  = parseInt(document.defaultView.getComputedStyle(canvas, null)['borderLeftWidth'], 10)  || 0;
    // this.styleBorderTop   = parseInt(document.defaultView.getComputedStyle(canvas, null)['borderTopWidth'], 10)   || 0;
  // }
  // Some pages have fixed-position bars (like the stumbleupon bar) at the top or left of the page
  // They will mess up mouse coordinates and this fixes that
  // var html = document.body.parentNode;
  // this.htmlTop = html.offsetTop;
  // this.htmlLeft = html.offsetLeft;

  // **** Keep track of state! ****
  
  this.valid = false; // when set to false, the canvas will redraw everything
  this.shapes = [];  // the collection of things to be drawn
  
  // **** Then events! ****
  
  var myState = this;
  
  // **** Options! ****
  
  this.interval = 20;
  setInterval(function() { myState.draw(); }, myState.interval);
}

CanvasState.prototype.addShape = function(shape) {
  this.shapes.push(shape);
  this.valid = false;
}

CanvasState.prototype.clear = function() {
  this.ctx.clearRect(0, 0, this.width, this.height);
}

// While draw is called as often as the INTERVAL variable demands,
// It only ever does something if the canvas gets invalidated by our code
CanvasState.prototype.draw = function() {
  // if our state is invalid, redraw and validate!
  if (!this.valid ) {
    var ctx = this.ctx;
    var shapes = this.shapes;
    this.clear();
    
    // ** Add stuff you want drawn in the background all the time here **
    
    newvalid = true;

    try {
	    // draw all shapes
	    var l = shapes.length;
	    for (var i = 0; i < l; i++) {
	      var shape = shapes[i];
	      // alert("i="+i+" shapes.length="+shapes.length+" shape="+shape);
	      if ( shape.fade() > 0.0 ) {
		  shape.draw(ctx);
		  newvalid = false;
	      } else {
		  shapes.splice(i,1);
		  // alert("faded i="+i+" to 0, shapes size is now "+shapes.length);
		  i--;
	      }
	    }
    } catch(err) {
    }
    
    // ** Add stuff you want drawn on top all the time here **
    
    this.valid = newvalid;
  }
}

// If you dont want to use <body onLoad='init()'>
// You could uncomment this init() reference and place the script reference inside the body tag

	var shapeOfCursor = [0, 1, 2, 3];
	var colorForCursorRegion = [
		"rgba(255,0,0,[[alpha]])",
		"rgba(0,255,0,[[alpha]])",
		"rgba(255,255,0,[[alpha]])",
		"rgba(0,0,255,[[alpha]])",
	];

	var cursorDown = [0, 0, 0, 0];
	var maxCursors = 4;
	var ws;
	var osc_sid = [];
	var date = new Date();
	var time0 = date.getTime();
        var canvas;
        var context;
	var thestate;

    window.onload = function() {

	// We want to isolate the first part of the url (after http://)
	var url = document.location.href;
	var i = url.indexOf("//");
	if ( i >= 0 ) {
	    url = url.substr(i+2);
	}
	var i = url.indexOf("/");
	if ( i >= 0 ) {
	    url = url.substr(0,i);
	}
	var wurl = "ws://" + url + "/websocket";

	if ("WebSocket" in window) {
		ws = new WebSocket(wurl);
	} else {
		ws = new MozWebSocket(wurl);
	}
	ws.onmessage = handleJSON;

	ws.onopen = function() {
		// console.log("EVENT onopen!");
	}
	ws.onclose = function() {
		// console.log("EVENT onclose!");
	}
	ws.onerror = function() {
		// console.log("EVENT onerror!");
	}

	thestate = new CanvasState(document.getElementById('myCanvas'));

        canvas = document.getElementById('myCanvas');
        context = canvas.getContext('2d');

	function drawcursor(c) {
		x = c.x * canvas.width;
		y = (1.0 - (c.y*1.6)) * canvas.height;
		sz = (c.z*c.z) * 20;
		sz = sz * sz;

		// debugMessage("x="+x);
		// debugMessage("y="+y);
		// debugMessage("sz="+sz);

		thestate.addShape(new Shape(x,y,sz,0.1,
				colorForCursorRegion[c.region]));
	}
	function Cursor(region) {
		this.touched = false
		this.region = region
	}
	function handleOscMessage(m) {
	    if ( m.address == "/tuio/25Dblb" ) {
		var args = m.args;
		var cmd = args[0];
		if ( cmd == "alive" ) {
			var c;
			for ( sid in osc_sid ) {
				c = osc_sid[sid];
				c.touched = false;
			}
			for ( n=1; n<args.length; n++ ) {
				sid = args[n];
				if ( ! ( sid in osc_sid ) ) {
					region = Math.floor(sid / 1000) % maxCursors;
					osc_sid[sid] = new Cursor(region);
					// debugMessage("New cursor region="+region+" sid="+sid);
				}
				c = osc_sid[sid];
				c.touched = true;
			}
			for ( sid in osc_sid ) {
				c = osc_sid[sid];
				if ( c.touched == false ) {
					// debugMessage("Deleting cursor sid="+sid);
					delete osc_sid[sid]
				}
			}
		} else if ( cmd == "set" ) {
			sid = args[1];
			osc_sid[sid].x = args[2];
			osc_sid[sid].y = args[3];
			osc_sid[sid].z = args[4];
			logOscMessage("args[2] = "+args[2]+" sid="+sid+" osc_sid.x="+osc_sid[sid].x);
			drawcursor(osc_sid[sid]);
			logOscMessage(JSON.stringify(m));
		} else if ( cmd == "fseq" ) {
			// debugMessage("fseq!");
		} else {
			debugMessage("No hander in 25Dblb for "+cmd);
		}
	    }
	}

	function handleJSON ( event ) {
		var data;

		try {
			data = JSON.parse(event.data);
		} catch(err) {
			debugMessage("Exception in JSON.parse, err="+err.message);
			return;
		}

		if ( data == undefined ) {
			logOscMessage("Bad JSON data="+event.data);
		} else if ( data.messages != undefined ) {
			// JSON that has a "messages" key is an OSC bundle, an array
			// of individual OSC messages
			for ( m in data.messages ) {
				handleOscMessage(data.messages[m]);
			}
		} else if ( data.address != undefined ) {
			// JSON that has an "address" key is a single OSC message
			handleOscMessage(data);
		} else if ( data.message != undefined ) {
			logOtherMessage(data.message);
		} else {
			logOtherMessage("Unrecognized JSON data="+event.data);
		}
	}
	function debugMessage ( msg ) {
		// console.log(msg);
		logOtherMessage(msg);
	}
	function logOscMessage ( line ) {
		line = "Time: "+getTime()+" : "+line;
		var area = document.getElementById("oscmessages");
		addMessageThrottle(line,area);
	}
	function logOtherMessage ( line ) {
		line = "Time: "+getTime()+" : "+line;
		var area = document.getElementById("othermessages");
		addMessageThrottle(line,area);
	}

	function getTime() {
	    var d = new Date();
	    var tm = (d.getTime() - time0) / 1000.0;
	    return tm.toFixed(3);
	}

	function addMessageThrottle ( line, area ) {
		var s = "";
		var olds = area.innerHTML.split("\n");
		var maxlines = 32;
		for(var i=olds.length-maxlines; i<olds.length; i++) {
			if(i>=0) s += olds[i] + "\n";  
		}
		s += line;
		area.innerHTML = s;
		area.scrollTop = area.scrollHeight;
	}
    };
