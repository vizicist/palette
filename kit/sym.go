package kit

import (
	"crypto/md5"
	"log"
)

var Keywords Htablep
var Macros Htablep
var Topht Htablep
var Freeht Htablep
var Currct *Context
var Topct *Context

var Merge, Now, Clicks, Debug, Sync, Optimize, Mergefilter, Nowoffset Symlongp
var Mergeport1, Mergeport2 Symlongp
var Debugwait, Debugmidi, Debugrun, Debugfifo, Debugmouse, Debuggesture Symlongp
var Clocksperclick, Clicksperclock, Graphics, Debugdraw, Debugmalloc Symlongp
var Millicount, Throttle2, Loadverbose, Warnnegative, Midifilenoteoff Symlongp
var Drawcount, Mousedisable, Forceinputport, Showsync, Echoport Symlongp
var Inputistty, Debugoff, Fakewrap, Mfsysextype Symlongp
var Tempotrack, Onoffmerge, Defrelease, Grablimit, Mfformat, Defoutport Symlongp
var Filter, Record, Recsched, Throttle, Recfilter, Recinput, Recsysex Symlongp
var Lowcorelim, Arraysort, Midithrottle, Defpriority Symlongp
var Taskaddr, Debuginst, Usewindfifos, Prepoll, Printsplit Symlongp
var Novalval, Eofval, Intrval, Debugkill, Debugkill1, Linetrace Symlongp
var Abortonint, Abortonerr, Redrawignoretime, Resizeignoretime Symlongp
var Consecho, Checkcount, Isofuncwarn, Consupdown, Monitor_fnum Symlongp
var Consecho_fnum, Slashcheck, Directcount, SubstrCount Symlongp
var Mousefnum, Consinfnum, Consoutfnum, Midi_in_fnum, Mousefifolimit Symlongp
var Saveglobalsize, Warningsleep, Millires, Milliwarn, Resizefix Symlongp
var Minbardx, Kobjectoffset, Midi_out_fnum, Mousemoveevents Symlongp
var Numinst1, Numinst2, Offsetpitch, Offsetfilter, DoDirectinput Symlongp
var Offsetportfilter Symlongp

var Rebootfuncd, Nullfuncd, Errorfuncd *Datum
var Intrfuncd, Nullvald *Datum
var Zeroval Datum
var Noval Datum
var Nullval Datum
var _Dnumtmp_ Datum

var Colorfuncd *Datum
var Redrawfuncd *Datum
var Resizefuncd *Datum
var Exitfuncd *Datum
var Track, Wpick Htablepp
var Chancolormap Htablepp
var Chancolors Symlongp

func newcontext(s Symbolp, sz int) {
	c := &Context{}
	c.symbols = newht(sz)
	c.cfunc = s
	c.next = Currct
	c.localnum = 1
	c.paramnum = 1
	Currct = c
}

func popcontext() {
	if Currct == nil || Currct.next == nil {
		execerror("popcontext called too many times!\n")
	}
	nextc := Currct.next
	// kfree(Currct)
	Currct = nextc
}

var Free_sy Symbolp

func newsy() Symbolp {

	var s Symbolp

	// First check the free list and use those nodes, before using
	// the newly allocated stuff.
	if Free_sy != nil {
		s = Free_sy
		Free_sy = Free_sy.next
	} else {
		s = &Symbol{}
	}

	s.stype = UNDEF
	s.stackpos = 0 // i.e. it's global
	s.flags = 0
	s.onchange = nil
	s.next = nil
	return s
}

func freesy(sy Symbolp) {
	/* Add it to the list of free symbols */
	sy.next = Free_sy
	Free_sy = sy
}

func findsym(p string, symbols Htablep) Symbolp {
	key := strdatum(p)
	h := hashtable(symbols, key, H_LOOK)
	if h != nil {
		return h.val.u.(Symbolp)
	} else {
		return nil
	}
}

func findobjsym(p string,o Kobjectp,foundobj *Kobjectp) Symbolp {
	symbols := o.symbols
	if symbols == nil {
		log.Printf("Internal error - findobjsym finds NULL symbols!\n")
		return nil
	}
	key := strdatum(p)
	h := hashtable(symbols,key,H_LOOK)
	if h != nil {
		if foundobj {
			*foundobj = o
		}
		return h.val.u.sym
	}
	
	// Not found, try inherited objects
	if o.inheritfrom != nil {
		for o2:=o.inheritfrom; o2!=nil; o2=o2.nextinherit {
			s=findobjsym(p,o2,foundobj)
			if s != nil {
				return s
			}
		}
	}
	return nil
}

var unum = 0

func uniqvar(pre string) Symbolp {
	
	buff := pre[0:20]
	sprintf(strend(buff),"%s%ld",NONAMEPREFIX,unum)
	unum++
	if unum > (math.MaxInt32-4) {
		execerror("uniqvar() has run out of names!?")
	}
	return globalinstallnew(buff,VAR)
}

// lookup(p) - find p in symbol table
func lookup(p string) Symbolp {
	s := findsym(p,Currct.symbols)
	if s != nil {
		return s
	}
	if Currct != Topct {
		s = findsym(p,Topct.symbols)
		if s != nil {
			return s
		}
	}
	return nil
}

func localinstall(p string,t int) Symbolp {
	s := syminstall(p,Currct.symbols,t)
	if Inparams {
		s.stackpos = Currct.paramnum
		Currct.paramnum++
	} else {
		s.stackpos = -(Currct.localnum)	// okay if chars are unsigned
		Currct.localnum++
	}
	return s
}

var Starting = 1

func globalinstall(p string,t int) Symbolp {
	s := findsym(p,Topct.symbols)
	if s != nil {
		return s
	}
	s = syminstall(p,Topct.symbols,t)
	return s
}

// Use this variation if you know that the symbol is new
func
globalinstallnew(p string,t int) Symbolp {
	return syminstall(p,Topct.symbols,t)
}

func syminstall(p string,symbols Htablep,t int) Symbolp {

	var s Symbolp

	key := strdatum(p);
	h := hashtable(symbols,key,H_INSERT)
	if h==NULL {
		execerror("Unexpected h==NULL in syminstall!?")
	}
	if isnoval(h.val) {
		s := newsy()
		s.name = key
		s.stype = t
		s.stackpos = 0
		s.sd = Noval
		h.val = symdatum(s)
	} else {
		if h.val.stype != D_SYM {
			execerror("Unexpected h.val.stype!=D_SYM in syminstall!?")
		}
		s = h.val.u.(Symbolp)
	}
	return s
}

func clearsym(s Symbolp) {

	//// #ifdef OLDSTUFF
	//// 	Codep cp;
	//// 	BLTINCODE bc;
	//// #endif

	if s.stype == VAR {
		dp := symdataptr(s)
		switch dp.dtype {
		case D_ARR:
			if dp.u.(Htablep) != nil {
				arrdecruse(dp.u.(Htablep))
				dp.u = nil
			}
			break
		case D_PHR:
			phr := dp.u.(Phrasep)
			if phr != nil {
				phdecruse(phr)
				dp.u = nil
			}
			break
		case D_CODEP:

			//// 			// BUG FIX - 5/4/97 - no longer free it
			//// #ifdef OLDSTUFF
			//// 			cp = dp->u.codep;
			//// 			bc = ((cp==NULL) ? 0 : BLTINOF(cp));
			//// 			// If it's a built-in function, then the codep was
			//// 			// allocated by funcdp(), so we can free it.
			//// 			if ( bc != 0 ) {
			//// 				// kfree(cp);
			//// 				dp->u.codep = NULL;
			//// 			}
			//// #endif

			dp.u = nil
			break
		default:
			break
		}
	}
}

type keywordinfo struct {
	name string
	kval int
}

var keywords = []keywordinfo{
	"function",	FUNC,
	"return",	RETURN,
	"if",		IF,
	"else",		ELSE,
	"while",	WHILE,
	"for",		FOR,
	"in",		SYM_IN,  /* to avoid conflict on windows */
	"break",	BREAK,
	"continue",	CONTINUE,
	"task",		TASK,
	"eval",		EVAL,
	"vol",		VOL,	/* sorry, I'm just used to 'vol' */
	"volume",	VOL,
	"vel",		VOL,
	"velocity",	VOL,
	"chan",		CHAN,
	"channel",	CHAN,
	"pitch",	PITCH,
	"time",		TIME,
	"dur",		DUR,
	"duration",	DUR,
	"length",	LENGTH,
	"number",	NUMBER,
	"type",		TYPE,
	"defined",	DEFINED,
	"undefine",	UNDEFINE,
	"delete",	SYM_DELETE,
	"readonly",	READONLY,
	"onchange",	ONCHANGE,
	"flags",	FLAGS,
	"varg",		VARG,
	"attrib",	ATTRIB,
	"global",	GLOBALDEC,
	"class",	CLASS,
	"method",	METHOD,
	"new",		KW_NEW,
	"nargs",	NARGS,
	"typeof",	TYPEOF,
	"xy",		XY,
	"port",		PORT,
	0,		0,
}

func neednum(s string,d Datum) int {
	if d.dtype != D_NUM && d.dtype != D_DBL {
		execerror("%s expects a number, got %s!",s,atypestr(d.dtype));
	}
	return roundval(d)
}

func needfunc(s string,d Datum) Codep {
	if d.dtype != D_CODEP {
		execerror("%s expects a function, got %s!",s,atypestr(d.dtype))
	}
	return d.u.codep
}

func needobj(s string, d Datum) Kobjectp {
	if d.dtype != D_OBJ {
		execerror("%s expects an object, got %s!",s,atypestr(d.dtype))
	}
	return d.u.obj
}

func needfifo(s string,d Datum) *Fifo {
	
	if d.dtype != D_NUM && d.dtype != D_DBL {
		execerror("%s expects a fifo id (i.e. a number), but got %s!",
			s,atypestr(d.dtype))
	}
	n := roundval(d)
	f := fifoptr(n)
	return f
}

func needvalidfifo(s string,d Datum) *Fifo {
	f := needfifo(s,d)
	if f == nil {
		execerror("%s expects a fifo id, and %ld is not a valid fifo id!",s,numval(d))
	}
	return f
}

func needstr(char *s,Datum d) string {
	if d.dtype != D_STR {
		execerror("%s expects a string, got %s!",s,atypestr(d.dtype))
	}
	return d.u.str
}

func needarr(s string,d Datum ) Htablep {
	if d.dtype != D_ARR {
		execerror("%s expects an array, got %s!",s,atypestr(d.dtype))
	}
	return d.u.(Htablep)
}

func needphr(s string ,d Datum) Phrasep {
	if d.dtype != D_PHR {
		execerror("%s expects a phrase, got %s!",s,atypestr(d.dtype))
	}
	return d.u.(Phrasep)
}

func datumstr(Datum d) (s Symstr) {
	if isnoval(d) {
		execerror("Attempt to convert uninitialized value (Noval) to string!?")
	}
		
	switch ( d.tdype ) {
	case D_NUM:
		s = strconv.Itoa(d.u.(int))
	case D_PHR:
		s = phrstr(d.u.(Phrasep),0)
	case D_DBL:
		s = log.Snprintf("%f",d.u.(float))
	case D_STR:
		s = d.u.(string)
	case D_ARR:
		/* we re-use the routines in main.c for doing */
		/* printing into a buffer.  I supposed we could really */
		/* do this for all the data types here. */
		stackbuffclear()
		prdatum(d,stackbuff,1)
		s = stackbuffstr()
	case D_OBJ:
		var id int
		obj := d.u.(Kobjectp)
		if obj != nil {
			id = obj.id
		} else {
			id = -1
		}
		id += *Kobjectoffset
		s = log.Snprintf("$%d",id)
	default:
		s = ""
	}
	return s
}

func newarrdatum(used int ,size int) Datum {
	d := Datum{dtype:D_ARR}
	if size <= 0 {
		size = ARRAYHASHSIZE
	}
	arr := newht(size)
	arr.h_used = used
	arr.h_tobe = 0
	d.u = arr
	return d
}

func globarray(name string) Htablepp {
	s := globalinstall(name,VAR)
	s.stype = VAR
	dp := symdataptr(s)
	*dp = newarrdatum(1,0)
	arr := dp.u.(Htablep)
	return &arr
}

func
phrsplit(p Phrasep) Datum {
	MAXSIMUL := 128
	activent := make([]Noteptr,MAXSIMUL)
	activetime := make([]int,MAXSIMUL)
	// Noteptr n, newn
	// Phrasep p2;
	// Symbolp s;
	// Datum d, da;
	// long tm2;
	// int i, j, k;
	// Htablep arr;
	// int samesection, nactive = 0;
	// long t, elapse, closest, now = 0L;

	
	da := newarrdatum(0,0)
	arr := da.u.(Htablep)

	arrnum := 0
	closest := 0
	samesection := false
	
	n := firstnote(p)
	
	for ; n!=nil || nactive>0;  {
		
		// find out which event is closer: the end of a pending
		// note, or the start of the next one (if there is one).
		
		if n != nil {
			closest = n.clicks - now
			samesection = 1
		} else {
			closest = 30000
			samesection = 0
		}
		
		// Check the ending times of the pending notes, to see
		// if any of them end before the next note starts.  If so,
		// we want to create a new section. 
		for k=0; k<nactive; k++ {
			t := activetime[k]
			if t <= closest {
				closest = t
				samesection = 0
			}
		}
		
		// We want to let that amount of time elapse.
		elapse = closest
		if samesection!=0 && (closest == 0 || nactive == 0) {
			goto addtosame
		}
		
		// We're going to create a new element in the split array
		
		d := numdatum(arrnum)
		arrnum++
		s := arraysym(arr,d,H_INSERT)
		p2 := newph(1)
		
		// add all active notes to the phrase
		tm2 = now+elapse
		for k=0; k<nactive; k++ {
			newn = ntcopy(activent[k])
			if timeof(newn) < now {
				overhang := now - timeof(newn)
				timeof(newn) += overhang
				durof(newn) -= overhang
			}
			if endof(newn) > tm2 {
				durof(newn) = tm2-timeof(newn)
			}
			ntinsert(newn,p2)
		}
		p2.p_leng = tm2;
		*symdataptr(s) = phrdatum(p2)
		
		// If any notes are pending, take into account elapsed
		// time, and if they expire, get rid of them.
		for i:=0; i<nactive; i++ {
			activetime[i] -= elapse
			if activetime[i] <= 0 {
				// Remove this note from the list by
				// shifting everything down.
				for j=i+1; j<nactive; j++ {
					activetime[j-1] = activetime[j];
					activent[j-1] = activent[j];
				}
				nactive--
				i--	// don't advance loop index */
			}
		}
		
	addtosame:
		if samesection {
			// add this new note (and all others that start at the 
			// same time) to the list of active ones 
			thistime := timeof(n);
			for ; n!=nil && timeof(n) == thistime ; {
				if nactive >= MAXSIMUL {
					execerror("Too many simultaneous notes in expression (limit is %d)\n",MAXSIMUL)
				}
				activent[nactive] = n
				activetime[nactive] = durof(n)
				nactive++
				// advance to next note
				n = nextnote(n)
			}
		}
		now += elapse
	}
	return da
}

// sep contains the list of possible separator characters.
// multiple consecutive separator characters are treated as one.
func strsplit(str string,sep string) Datum {
	// char buffer[128];
	// char *buff, *p, *endp;
	// char *word = NULL;
	// int isasep;
	// Datum da;
	// Htablep arr;
	// long n;
	// int state;

	
	// An inital scan to figure out how big an array we need
	state := 0
	nwords := 0
	slen := strlen(str)
	for i:=0; state >= 0 && i<slen ; i++ {
		isasep := strings.ContainsAny(str[i],sep)
		switch ( state ) {
		case 0:	/* before word */
			if ! isasep {
				state = 1
			}
		case 1:	/* scanning word */
			if isasep {
				nwords++
				state=0
			}
		}
	}
	if state == 1 {
		nwords++
	}
		
	da = newarrdatum(0,n)
	arr = da.u.(Htablep)
	
	state := 0
	nwords := 0
	var wordi int
	for i=0; state >= 0 && i < slen ; i++ {
		
		isasep := strings.ContainsAny(str[i],sep)
		switch ( state ) {
		case 0:	/* before word */
			if ! isasep {
				wordi := i
				state = 1
			}
		case 1:	/* scanning word */
			if isasep {
				word := str[wordi:i]
				setarrayelem(arr,nwords,word)
				nwords++
				state=0
			}
		}
	}
	if state == 1 {
		setarrayelem(arr,nwords,word)
	}
	// if slen >= sizeof(buffer) {
	// 	kfree(buff);
	// }
	return da
}

func setarraydata(arr Htablep,i Datum,d Datum) {
	if isnoval(i) {
		execerror("Can't use undefined value as array index\n")
	}
	s := arraysym(arr,i,H_INSERT)
	*symdataptr(s) = d
}

func setarrayelem(arr Htablep ,n int,p string) {
	setarraydata(arr,numdatum(n),strdatum(p))
}

func fputdatum(f os.File,d Datum) {
	str := datumstr(d)
	nl := false
	if (*Printsplit) != 0 {
		nl = true
	}
	
	if d.dtype == D_STR {
		putc('"',f)
	}
	slen := len()
	for i:=0; i<slen; i++ {
		p = ""
		i++
		if nl>0 && i>nl {
			i = 0
			fputs("\\\n",f)
		}
		switch (c) {
		case '\n':
			p = "\\n";
		case '\r':
			p = "\\r";
		case '"':
			p = "\\\"";
		case '\\':
			p = "\\\\";
		}
		if ( p ) {
			fputs(p,f)
		} else {
			putc(c,f)
		}
	}
	if d.dtype == D_STR {
		putc('"',f)
	}
}

type binum struct {
	name string
	val int
	ptovar *Symlongp
}

const MAXINT = math.MaxInt32

var binums = []binum {
	"Noval", MAXINT, &Novalval,
	"Eof", MAXINT-1, &Eofval,
	"Interrupt", MAXINT-2, &Intrval,
	"Merge", 1, &Merge,
	"Mergeport1", 0, &Mergeport1,    // default output
	"Mergeport2", -1, &Mergeport2,   //
	"Mergefilter", 0, &Mergefilter,
	"Clicks", (long)(DEFCLICKS), &Clicks,
	"Debug", 0, &Debug,
	"Optimize", 1, &Optimize,
	"Debugwait", 0, &Debugwait,
	"Debugoff", 0, &Debugoff,
	"Fakewrap", 0, &Fakewrap,
	"Debugrun", 0, &Debugrun,
	"Debuginst", 0, &Debuginst,
	"Debugkill", 0, &Debugkill,
	"Debugfifo", 0, &Debugfifo,
	"Debugmalloc", 0, &Debugmalloc,
	"Debugdraw", 0, &Debugdraw,
	"Debugmouse", 0, &Debugmouse,
	"Debugmidi", 0, &Debugmidi,
	"Debuggesture", 0, &Debuggesture,
	"Now", -1, &Now,
	"Nowoffset", 0, &Nowoffset,
	"Sync", 0, &Sync,
	"Showsync", 0, &Showsync,
	"Clocksperclick", 1, &Clocksperclick,
	"Clicksperclock", 1, &Clicksperclock,
	"Filter", 0, &Filter,	/* bitmask for message filtering */
	"Record", 1, &Record,		/* If 0, recording is disabled */
	"Recsched", 0, &Recsched,	/* If 1, record scheduled stuff */
	"Recinput", 1, &Recinput,	/* If 1, record midi input */
	"Recsysex", 1, &Recsysex,	/* If 1, record sysex */
	"Recfilter", 0, &Recfilter,	/* per-channel bitmask turns off recording */
	"Lowcore", DEFLOWLIM, &Lowcorelim,
	"Millicount", 0, &Millicount,	/* see mdep.c */
	"Throttle2", 100, &Throttle2,
	"Drawcount", 8, &Drawcount,
	"Mousedisable", 0, &Mousedisable,
	"Forceinputport", -1, &Forceinputport,
	"Checkcount", 20, &Checkcount,
	"Loadverbose", 0, &Loadverbose,
	"Warnnegative", 1, &Warnnegative,
	"Midifilenoteoff", 1, &Midifilenoteoff,
	"Isofuncwarn", 1, &Isofuncwarn,
	"Inputistty", 0, &Inputistty,
	"Arraysort", 0, &Arraysort,
	"Taskaddr", 0, &Taskaddr,
	"Tempotrack", 0, &Tempotrack,
	"Onoffmerge", 1, &Onoffmerge,
	"Defrelease", 0, &Defrelease,
	"Defoutport", 0, &Defoutport,
	"Echoport", 0, &Echoport,
	"Grablimit", 1000, &Grablimit,
	"Mfformat", 0, &Mfformat,
	"Mfsysextype", 0, &Mfsysextype,
	"Trace", 1, &Linetrace,
	"Abortonint", 0, &Abortonint,
	"Abortonerr", 0, &Abortonerr,
	"Debugkill1", 0, &Debugkill1,
	"Consecho", 1, &Consecho,
	"Slashcheck", 1, &Slashcheck,
	"Directcount", 0, &Directcount,
	"SubstrCount", 0, &SubstrCount,
	"Consupdown", 0, &Consupdown,
	"Prepoll", 0, &Prepoll,
	"Printsplit", 77, &Printsplit,
	"Midithrottle", 128, &Midithrottle,
	"Throttle", 100, &Throttle,
	"Defpriority", 500, &Defpriority,
	"Redrawignoretime", 100, &Redrawignoretime,
	"Resizeignoretime", 100, &Resizeignoretime,
	"Graphics", 1, &Graphics,
	"Consinfifo", -1, &Consinfnum,
	"Consoutfifo", -1, &Consoutfnum,
	"Mousefifo", -1, &Mousefnum,
	"Midiinfifo", -1, &Midi_in_fnum,
	"Midioutfifo", -1, &Midi_out_fnum,
	"Monitorfifo", -1, &Monitor_fnum,
	"Consechofifo", -1, &Consecho_fnum,
	"Saveglobalsize", 256, &Saveglobalsize,
	"Warningsleep", 0, &Warningsleep,
	"Millires", 1, &Millires,
	"Milliwarn", 2, &Milliwarn,
	"Resizefix", 1, &Resizefix,
	"Mousemoveevents", 0, &Mousemoveevents,
	"Objectoffset", 0, &Kobjectoffset,
	"Showtext", 1, &Showtext,
	"Showbar", 4*DEFCLICKS, &Showbar,
	"Sweepquant", 1, &Sweepquant,
	"Menuymargin", 2, &Menuymargin,
	"Menusize", 12, &Menusize,
	"Dragquant", 1, &Dragquant,
	"Menuscrollwidth", 15, &Menuscrollwidth,
	"Textscrollsize", 200, &Textscrollsize,
	"Menujump", 0, &Menujump,
	"Panraster", 1, &Panraster,
	"Bendrange", 1024*16, &Bendrange,
	"Bendoffset", 64, &Bendoffset,
	"Volstem", 0, &Volstem,
	"Volstemsize", 4, &Volstemsize,
	"Colors", 2, &Colors,
	"Colornotes", 1, &Colornotes,
	"Chancolors", 0, &Chancolors,
	"Inverse", 0, &Inverse,
	"Usewindfifos", 0, &Usewindfifos,
	"Mousefifolimit", 1, &Mousefifolimit,
	"Minbardx", 8, &Minbardx,
	"Numinst1", 0, &Numinst1,
	"Numinst2", 0, &Numinst2,
	"Directinput", 0, &DoDirectinput,
	"Offsetpitch", 0, &Offsetpitch,
	"Offsetportfilter", -1, &Offsetportfilter,
	"Offsetfilter", 1<<9, &Offsetfilter,/* per-channel bitmask turns off
		offset effect, default turns it off for channel 10, drums */
	"", 0, nil,
}

var Currphr, Recphr Phrasepp

type biphr struct {
	char *name;
	Phrasepp *ptophr;
}

var biphrs = []biphr{
	"Current", &Currphr,
	"Recorded", &Recphr,
	0, 0,
}

var Keypath, Machine, Keyerasechar, Keykillchar, Keyroot Symstrp
var Printsep, Printend, Musicpath Symstrp
var Pathsep, Dirseparator, Devmidi, Version, Initconfig, Nullvalsymp Symstrp
var Fontname, Icon, Windowsys, Drawwindow, Picktrack Symstrp

type bistr struct {
	name string
	val string
	ptostr *Symstrp
}

var bistrs = []bistr{
	"Keyroot", "", &Keyroot,
	"Keypath", "", &Keypath,
	"Musicpath", "", &Musicpath,
	"Machine", MACHINE, &Machine,
	"Devmidi", "", &Devmidi,
	"Printsep", " ", &Printsep,
	"Printend", "\n", &Printend,
	"Pathseparator", PATHSEP, &Pathsep,
	"Dirseparator", SEPARATOR, &Dirseparator,
	"Version", KEYVERSION, &Version,
	"Initconfig", "", &Initconfig,
	"Killchar", "", &Keykillchar,
	"Erasechar", "", &Keyerasechar,
	"Font", "", &Fontname,
	"Icon", "", &Icon,
	"Windowsys", "", &Windowsys,
	"Nullval", "", &Nullvalsymp,
	"", "", 0,
}

func installnum(name string,pvar *Symlongp,defval int) {

	var s Symbolp
	/* Only install and set value if not already present */
	s := lookup(name)
	if s == nil {
		s = globalinstallnew(name,VAR)
		*symdataptr(s) = numdatum(defval)
	}
	*pvar = (Symlongp)( &(symdataptr(s).u.val) )
}

func installstr(name string, str string) {
	s := globalinstallnew(uniqstr(name),VAR)
	*symdataptr(s) = strdatum(uniqstr(str))
}

// build a Datum that is a function pointer, pointing to a built-in function
func funcdp(s Symbolp, f BLTINCODE) Datum {

	sz := Codesize[IC_BLTIN] + varinum_size(0) + Codesize[IC_SYM];
	cp := (Codep) kmalloc(sz,"funcdp")
	// keyerrfile("CP 0 = %lld, sz=%d\n", (intptr_t)cp,sz);
	
	*Numinst1 += sz
	
	d := Datum{dtype : D_CODEP, u: cp}
	
	cp = put_bltincode(f,cp)
	cp = put_numcode(0,cp)
	cp = put_symcode(s,cp)
	return d
}

// Pre-defined macros.  It is REQUIRED that these values match the 
// corresponding values in phrase.h and grid.h.  For example, the value
// of P_STORE must match STORE, NT_NOTE must match NOTE, etc.

var Stdmacros = []string{
	// These are values for nt.type, also used as bit-vals for
	// the value of Filter.
	"MIDIBYTES 1", // NT_LE3BYTES is not here - not user-visible
	"NOTE 2",
	"NOTEON 4",
	"NOTEOFF 8",
	"CHANPRESSURE 16", "CONTROLLER 32", "PROGRAM 64", "PRESSURE 128",
		"PITCHBEND 256", "SYSEX 512", "POSITION 1024", "CLOCK 2048",
		"SONG 4096", "STARTSTOPCONT 8192", "SYSEXTEXT 16384",
		
	"Nullstr \"\"",
	
	// Values for action() types.  The values are intended to not
	// overlap the values for interrupt(), to avoid misuse and
	// also to leave open the possibility of merging the two.
	"BUTTON1DOWN 1024", "BUTTON2DOWN 2048", "BUTTON12DOWN 4096",
	"BUTTON1UP 8192", "BUTTON2UP 16384", "BUTTON12UP 32768",
	"BUTTON1DRAG 65536", "BUTTON2DRAG 131072", "BUTTON12DRAG 262144",
	"MOVING 524288",
	// values for setmouse() and sweep()
	"NOTHING 0", "ARROW 1", "SWEEP 2", "CROSS 3",
		"LEFTRIGHT 4", "UPDOWN 5", "ANYWHERE 6", "BUSY 7",
		"DRAG 8", "BRUSH 9", "INVOKE 10", "POINT 11", "CLOSEST 12",
		"DRAW 13",
	// values for cut()
	"NORMAL 0", "TRUNCATE 1", "INCLUSIVE 2",
	"CUT_TIME 3", "CUT_FLAGS 4", "CUT_TYPE 5",
	"CUT_CHANNEL 6", "CUT_NOTTYPE 7",
	// values for menudo()
	"MENU_NOCHOICE -1", "MENU_BACKUP -2", "MENU_UNDEFINED -3",
	"MENU_MOVE -4", "MENU_DELETE -5",
	// values for draw()
	"CLEAR 0", "STORE 1", "XOR 2",
	// values for window()
	"TEXT 1", "PHRASE 2",
	// values for style()
	"NOBORDER 0", "BORDER 1", "BUTTON 2", "MENUBUTTON 3", "PRESSEDBUTTON 4",
	// values for kill() signals
	"KILL 1",
	nil
}

// initsyms - install constants and built-ins in table */
void
func initsyms() {
	// 	int i;
	// 	Symbolp s;
	// 	Datum *dp;
	// 	char *p;
	
	Zeroval = numdatum(0)
	Noval = numdatum(MAXLONG)
	Nullstr = uniqstr("")
	
	Keywords = newht(113)	/* no good reason for 113 */
	for i := 0; ; i++ {
		p := keywords[i].name
		if p == nil {
			break
		}
		syminstall(uniqstr(p), Keywords, keywords[i].kval)
	}
	
	for n, p := range binums {
		// Don't need to uniqstr(p), because installnum does it.
		installnum(p,binums[i].ptovar,binums[i].val);
	}
	for i:=0; ; i++ {
		p = binums[i].name
		if p == nil {
			break
		}
		/* Don't need to uniqstr(p), because installnum does it. */
		installnum(p,binums[i].ptovar,binums[i].val);
	}
	
	 	for (i=0; (p=biphrs[i].name)!=NULL; i++) {
	 		s = globalinstallnew(uniqstr(p),VAR);
	 		dp = symdataptr(s);
	 		*dp = phrdatum(newph(1));
	 		*(biphrs[i].ptophr) = &(dp->u.phr);
	 		s->stackpos = 0;	/* i.e. it's global */
	 	}
	
	 	for (i=0; (p=bistrs[i].name)!=NULL; i++) {
	 		s = globalinstallnew(uniqstr(p),VAR);
	 		dp = symdataptr(s);
	 		*dp = strdatum(uniqstr(bistrs[i].val));
	 		*(bistrs[i].ptostr) = &(dp->u.str);
	 	}
	
	 	for (i=0; (p=builtins[i].name)!=NULL; i++) {
	 		s = globalinstallnew(uniqstr(p), VAR);
	 		dp = symdataptr(s);
	 		*dp = funcdp(s,builtins[i].bltindex);
	 	}
	
	 	Rebootfuncd = symdataptr(lookup(uniqstr("Rebootfunc")));
	 	Nullfuncd = symdataptr(lookup(uniqstr("nullfunc")));
	 	Errorfuncd = symdataptr(lookup(uniqstr("Errorfunc")));
	 	Intrfuncd = symdataptr(lookup(uniqstr("Intrfunc")));
	 	Nullvald = symdataptr(lookup(uniqstr("Nullval")));
	 	Nullval = *Nullvald;
	
	 	Colorfuncd = symdataptr(lookup(uniqstr("Colorfunc")));
	 	Redrawfuncd = symdataptr(lookup(uniqstr("Redrawfunc")));
	 	Resizefuncd = symdataptr(lookup(uniqstr("Resizefunc")));
	 	Exitfuncd = symdataptr(lookup(uniqstr("Exitfunc")));
	 	Track = globarray(uniqstr("Track"));
	 	Chancolormap = globarray(uniqstr("Chancolormap"));
	
	 	Macros = newht(113);	/* no good reason for 113 */
	
	 	for ( i=0; (p=Stdmacros[i]) != NULL;  i++ ) {
	 		/* Some compilers make strings read-only */
	 		p = strsave(p);
	 		macrodefine(p,0);
	 		free(p);
	 	}
	 	sprintf(Msg1,"MAXCLICKS=%ld",(long)(MAXCLICKS));
	 	macrodefine(Msg1,0);
	 	sprintf(Msg1,"MAXPRIORITY=%ld",(long)(MAXPRIORITY));
	 	macrodefine(Msg1,0);
	
	 	*Inputistty = mdep_fisatty(Fin) ? 1 : 0;
	 	if ( *Inputistty == 0 )
	 		*Consecho = 0;
	 	Starting = 0;
	
	 	*Keypath = uniqstr(mdep_keypath());
	 	*Musicpath = uniqstr(mdep_musicpath());
}

////
//// void
//// initsyms2(void)
//// {
//// 	if ( **Keyerasechar == '\0' ) {
//// 		char str[2];
//// 		str[0] = Erasechar;
//// 		str[1] = '\0';
//// 		*Keyerasechar = uniqstr(str);
//// 	}
//// 	if ( **Keykillchar == '\0' ) {
//// 		char str[2];
//// 		str[0] = Killchar;
//// 		str[1] = '\0';
//// 		*Keykillchar = uniqstr(str);
//// 	}
//// }
////
//// Datum Str_x0, Str_y0, Str_x1, Str_y1, Str_x, Str_y, Str_button;
//// Datum Str_type, Str_mouse, Str_drag, Str_move, Str_up, Str_down;
//// Datum Str_highest, Str_lowest, Str_earliest, Str_latest, Str_modifier;
//// Datum Str_default, Str_w, Str_r, Str_init;
//// Datum Str_get, Str_set, Str_newline;
//// Datum Str_red, Str_green, Str_blue, Str_grey, Str_surface;
//// Datum Str_finger, Str_hand, Str_xvel, Str_yvel;
//// Datum Str_proximity, Str_orientation, Str_eccentricity;
//// Datum Str_width, Str_height;
//// #ifdef MDEP_OSC_SUPPORT
//// Datum Str_elements, Str_seconds, Str_fraction;
//// #endif
////
//// void
//// initstrs(void)
//// {
//// 	Str_type = strdatum(uniqstr("type"));
//// 	Str_mouse = strdatum(uniqstr("mouse"));
//// 	Str_drag = strdatum(uniqstr("mousedrag"));
//// 	Str_move = strdatum(uniqstr("mousemove"));
//// 	Str_up = strdatum(uniqstr("mouseup"));
//// 	Str_down = strdatum(uniqstr("mousedown"));
//// 	Str_x = strdatum(uniqstr("x"));
//// 	Str_y = strdatum(uniqstr("y"));
//// 	Str_x0 = strdatum(uniqstr("x0"));
//// 	Str_y0 = strdatum(uniqstr("y0"));
//// 	Str_x1 = strdatum(uniqstr("x1"));
//// 	Str_y1 = strdatum(uniqstr("y1"));
//// 	Str_button = strdatum(uniqstr("button"));
//// 	Str_modifier = strdatum(uniqstr("modifier"));
//// 	Str_highest = strdatum(uniqstr("highest"));
//// 	Str_lowest = strdatum(uniqstr("lowest"));
//// 	Str_earliest = strdatum(uniqstr("earliest"));
//// 	Str_latest = strdatum(uniqstr("latest"));
//// 	Str_default = strdatum(uniqstr("default"));
//// 	Str_w = strdatum(uniqstr("w"));
//// 	Str_r = strdatum(uniqstr("r"));
//// 	Str_init = strdatum(uniqstr("init"));
//// 	Str_get = strdatum(uniqstr("get"));
//// 	Str_set = strdatum(uniqstr("set"));
//// 	Str_newline = strdatum(uniqstr("\n"));
//// 	Str_red = strdatum(uniqstr("red"));
//// 	Str_green = strdatum(uniqstr("green"));
//// 	Str_blue = strdatum(uniqstr("blue"));
//// 	Str_grey = strdatum(uniqstr("grey"));
//// 	Str_surface = strdatum(uniqstr("surface"));
//// 	Str_finger = strdatum(uniqstr("finger"));
//// 	Str_hand = strdatum(uniqstr("hand"));
//// 	Str_xvel = strdatum(uniqstr("xvel"));
//// 	Str_yvel = strdatum(uniqstr("yvel"));
//// 	Str_proximity = strdatum(uniqstr("proximity"));
//// 	Str_orientation = strdatum(uniqstr("orientation"));
//// 	Str_eccentricity = strdatum(uniqstr("eccentricity"));
//// 	Str_height = strdatum(uniqstr("height"));
//// 	Str_width = strdatum(uniqstr("width"));
//// #ifdef MDEP_OSC_SUPPORT
//// 	Str_elements = strdatum(uniqstr("elements"));
//// 	Str_seconds = strdatum(uniqstr("seconds"));
//// 	Str_fraction = strdatum(uniqstr("fraction"));
//// #endif
//// }
////
//// static FILE *Mf;
////
//// void
//// pfprint(char *s)
//// {
//// 	fputs(s,Mf);
//// }
////
//// void
//// phtofile(FILE *f,Phrasep p)
//// {
//// 	Mf = f;
//// 	phprint(pfprint,p,0);
//// 	putc('\n',f);
//// 	if ( fflush(f) )
//// 		mdep_popup("Unexpected error from fflush()!?");
//// }
////
//// void
//// vartofile(Symbolp s, char *fname)
//// {
//// 	FILE *f;
////
//// 	if ( fname==NULL || *fname == '\0' )
//// 		return;
////
//// 	if ( stdioname(fname) )
//// 		f = stdout;
//// 	else if ( *fname == '|' ) {
//// #ifdef PIPES
//// 		f = popen(fname+1,"w");
//// 		if ( f == NULL ) {
//// 			eprint("Can't open pipe: %s\n",fname+1);
//// 			return;
//// 		}
//// #else
//// 		eprint("No pipes!\n");
//// 		return;
//// #endif
//// 	}
//// 	else {
//// 		f = getnopen(fname,"w");
//// 		if ( f == NULL ) {
//// 			eprint("Can't open %s\n",fname);
//// 			return;
//// 		}
//// 	}
////
//// 	phtofile(f,symdataptr(s)->u.phr);
////
//// 	if ( f != stdout ) {
//// 		if ( *fname != '|' )
//// 			getnclose(fname);
//// #ifdef PIPES
//// 		else {
//// 			if ( pclose(f) < 0 )
//// 				eprint("Error in pclose!?\n");
//// 		}
//// #endif
//// 	}
//// }
////
//// /* Map the contents of a file (or output of a pipe) into a phrase */
//// /* variable. Note that if the file can't be read or the pipe can't */
//// /* be opened, it's a silent error. */
////
//// void
//// filetovar(register Symbolp s, char *fname)
//// {
//// 	FILE *f;
//// 	Phrasep ph;
////
//// 	if ( fname==NULL || *fname == '\0' )
//// 		return;
////
//// 	if ( stdioname(fname) )
//// 		f = stdin;
//// 	else if ( *fname == '|' ) {
//// 		/* It's a pipe... */
//// #ifdef PIPES
//// 		f = popen(fname+1,"r");
//// #else
//// 		warning("No pipes!");
//// 		return;
//// #endif
//// 	}
//// 	else {
//// 		/* a normal file */
////
//// 		/* Use KEYPATH value to look for files. */
//// 		char *pf = mpathsearch(fname);
//// 		if ( pf )
//// 			fname = pf;
////
//// 		f = getnopen(fname,"r");
//// 	}
//// 	if ( f == NULL || feof(f) )
//// 		return;		/* Silence.  Might be appropriate to */
//// 				/* make some noise when a pipe fails. */
//// 	clearsym(s);
//// 	s->stype = VAR;
//// 	ph = filetoph(f,fname);
//// 	phincruse(ph);
//// 	*symdataptr(s) = phrdatum(ph);
////
//// 	if ( f != stdin ) {
//// 		if ( *fname != '|' )
//// 			getnclose(fname);
//// #ifdef PIPES
//// 		else
//// 			if ( pclose(f) < 0 )
//// 				eprint("Error in pclose!?\n");
//// #endif
//// 	}
////
//// }

var Free_hn Hnodep

func newhn() Hnodep {

	var hn Hnodep

	/* First check the free list and use those nodes, before using */
	/* the newly allocated stuff. */
	if Free_hn != nil {
		hn = Free_hn
		Free_hn = Free_hn.next
	} else {
		hn = &Hnode{}
	}

	hn.next = nil
	hn.val = symdatum(nil)
	/* hn.key = NULL; */
	return (hn)
}

func freehn(hn Hnodep) {

	if hn == nil {
		execerror("Hey, hn==NULL in freehn\n")
	}
	switch hn.val.dtype {
	case D_SYM:
		sym := hn.val.u.(Symbolp)
		if sym != nil {
			clearsym(sym)
			freesy(sym)
		}
	case D_TASK:
		t := hn.val.u.(Ktaskp)
		if t != nil {
			freetp(t)
		}
	case D_FIFO:
		ff := hn.val.u.(*Fifo)
		if ff != nil {
			freeff(ff)
		}
	case D_WIND:
		// do nothing
	default:
		log.Printf("Hey, type=%d in clearhn, should something go here??\n", hn.val.dtype)
	}
	hn.val = Noval

	hn.next = Free_hn
	Free_hn = hn
}

//// #ifdef OLDSTUFF
//// void
//// chkfreeht() {
//// 	register Htablep ht;
//// 	if ( Freeht == NULL || Freeht->h_next == NULL )
//// 		return;
//// 	for ( ht=Freeht->h_next; ht!=NULL; ht=ht->h_next ) {
//// 		if ( ht == Freeht ) {
//// 			eprint("INFINITE LOOP IN FREEHT LIST!!!\n");
//// 			abort();
//// 		}
//// 	}
//// }
//// #endif
////

// To avoid freeing and re-allocating the large chunks of memory
// used for the hash tables, we keep them around and reuse them.
func newht(size int) Htablep {
	var ht Htablep

	/* eprint("(newht(%d ",size); */
	// See if there's a saved table we can use
	for ht = Freeht; ht != nil; ht = ht.h_next {
		if ht.size == size {
			break
		}
	}
	if ht != nil {
		// Remove from Freeht list
		if ht.h_prev == nil {
			// it's the first one in the Freeht list
			Freeht = ht.h_next
			if Freeht != nil {
				Freeht.h_prev = nil
			}
		} else if ht.h_next == nil {
			// it's the last one in the Freeht list
			ht.h_prev.h_next = nil
		} else {
			ht.h_next.h_prev = ht.h_prev
			ht.h_prev.h_next = ht.h_next
		}
	} else {
		ht = &Htable{}
		ht.size = size
		ht.nodetable = make([]Hnodep, size)
		// initialize entire table to NULLS
		// pp = h + size;
		// while ( pp-- != h ) {
		// 	*pp =  NULL;
		// }
	}

	ht.count = 0
	ht.h_used = 0
	ht.h_tobe = 0
	ht.h_next = nil
	ht.h_prev = nil
	ht.h_state = 0
	if Topht != nil {
		Topht.h_prev = ht
		ht.h_next = Topht
	}
	Topht = ht
	return (ht)
}

func clearht(ht Htablep) {
	// as we're freeing the Hnodes pointed to by this hash table,
	// we zero out the table, in preparation for its reuse.
	if ht.count != 0 {
		for i, hn := range ht.nodetable {
			var nexthn Hnodep
			for tmp := hn; tmp != nil; tmp = nexthn {
				nexthn = tmp.next
				// freehn(hn)
			}
			ht.nodetable[i] = nil
		}
	}
	ht.count = 0
}

func freeht(ht Htablep) {

	var ht2 Htablep

	clearht(ht)

	/* If it's in the Htobechecked list... */
	for ht2 := Htobechecked; ht2 != nil; ht2 = ht2.h_next {
		if ht2 == ht {
			break
		}
	}
	/* remove it */
	if ht2 != nil {
		if ht2.h_next != nil {
			ht2.h_next.h_prev = ht2.h_prev
		}
		if ht2 == Htobechecked {
			Htobechecked = ht2.h_next
		} else {
			ht2.h_prev.h_next = ht2.h_next
		}
	}

	for ht2 := Freeht; ht2 != nil; ht2 = ht2.h_next {
		if ht == ht2 {
			log.Printf("HEY!, Trying to free an ht node that's already in the Free list!!\n")
			// abort()
		}
	}
	/* Add to Freeht list */
	if Freeht != nil {
		Freeht.h_prev = ht
	}
	ht.h_next = Freeht
	ht.h_prev = nil
	ht.h_used = 0
	ht.h_tobe = 0
	ht.h_state = 0
	Freeht = ht
}

////
//// void
//// htlists(void)
//// {
//// 	Htablep ht3;
//// 	eprint("   Here's the Freeht list:");
//// 	for(ht3=Freeht;ht3!=NULL;ht3=ht3->h_next)eprint("(%lld,sz%d,u%d,t%d)",(intptr_t)ht3,ht3->size,ht3->h_used,ht3->h_tobe);
//// 	eprint("\n");
//// 	eprint("   Here's the Htobechecked list:");
//// 	for(ht3=Htobechecked;ht3!=NULL;ht3=ht3->h_next)eprint("(%lld,sz%d,u%d,t%d)",(intptr_t)ht3,ht3->size,ht3->h_used,ht3->h_tobe);
//// 	eprint("\n");
//// 	eprint("   Here's the Topht list:");
//// 	for(ht3=Topht;ht3!=NULL;ht3=ht3->h_next)eprint("(%lld,sz%d,u%d,t%d)",(intptr_t)ht3,ht3->size,ht3->h_used,ht3->h_tobe);
//// 	eprint("\n");
//// }
////
//// Htablep Stringtable = NULL;
////
//// /* uniqstr uses the same Hnode definition as is used for array element */
//// /* hash tables, even though the only type of value stored is a string. */
////
//// Symstr
//// uniqstr(char *s)
//// {
//// 	Hnodepp table;
//// 	Hnodep h, toph;
//// 	int v;
////
//// 	if ( Stringtable == NULL ) {
//// 		char *p = getenv("STRHASHSIZE");
//// 		Stringtable = newht( p ? atoi(p) : 1009 );
//// 	}
////
//// 	{
//// 		register unsigned int t = 0;
//// 		register int c;
//// 		register char *p = s;
////
//// 		/* compute hash value of string */
//// 		while ( (c=(*p++)) != '\0' ) {
//// 			t += c;
//// 			t <<= 3;
//// 		}
//// 		v = t % (Stringtable->size);
//// 	}
////
//// 	table = Stringtable->nodetable;
//// 	toph = table[v];
//// 	if ( toph == NULL ) {
//// 		/* no collision */
//// 		h = newhn();
//// 		h->key.u.str = kmalloc((unsigned)strlen(s)+1,"uniqstr");
////
//// 		strcpy((char*)(h->key.u.str),s);
//// 		/* h->sym is unused, key and value are the same */
//// 	}
//// 	else {
//// 		Hnodep prev;
////
//// 		/* quick test for first node in list, most common case */
//// 		if ( strcmp(toph->key.u.str,s) == 0  )
//// 			return(toph->key.u.str);
////
//// 		/* Look through entire list */
//// 		h = toph;
//// 		for ( prev=h; (h=h->next) != NULL; prev=h ) {
//// 			if ( strcmp(h->key.u.str,s) == 0 )
//// 				break;
//// 		}
//// 		if ( h == NULL ) {
//// 			/* string wasn't found, add it */
//// 			h = newhn();
//// 			h->key.u.str = kmalloc((unsigned)strlen(s)+1,"uniqstr");
////
//// 			strcpy((char*)(h->key.u.str),s);
//// 			/* h->sym is unused, key and value are the same */
//// 		}
//// 		else {
//// 			/* Symstr found.  Delete it from it's current */
//// 			/* position so we can move it to the top. */
//// 			prev->next = h->next;
//// 		}
//// 	}
//// 	/* Whether we've just allocated a new node, or whether we've */
//// 	/* found the node somewhere in the list, we insert it at the */
//// 	/* top of the list.  Ie. the lists are constantly re-arranging */
//// 	/* themselves to put the most recently seen entries on top. */
//// 	h->next = toph;
//// 	table[v] = h;
//// 	return(h->key.u.str);
//// }
////
//// int
//// isundefd(Symbolp s)
//// {
//// 	Datum d;
////
//// 	if ( s->stype == UNDEF )
//// 		return(1);
//// 	d = *symdataptr(s);
//// 	if ( isnoval(d) )
//// 		return(1);
//// 	else
//// 		return(0);
//// }
////

func hashvalue(s string) int {
	hmd5 := md5.Sum([]byte(s))
	// hsha1 := sha1.Sum([]byte(s))
	// hsha2 := sha256.Sum256([]byte(s))
	var res int
	for _, v := range hmd5[0:8] {
		res <<= 8
		res |= int(v)
	}
	return res
}

/*
 * Look for an element in the hash table.
 * Values of 'action':
 *     H_INSERT ==> look for, and if not found, insert
 *     H_LOOK ==> look for, but don't insert
 *     H_DELETE ==> look for and delete
 */

func hashtable(ht Htablep, key Datum, action int) Hnodep {

	var table []Hnodep
	var h, toph Hnodep
	var v int

	table = ht.nodetable

	// base the hash value on the 'uniqstr'ed pointer
	switch key.dtype {
	case D_NUM:
		v = key.u.(int) % (ht.size)
	case D_STR:
		v = hashvalue(key.u.(string)) % (ht.size)
	case D_OBJ:
		// use raw obj id for the hash value
		kobj := key.u.(Kobjectp)
		v = kobj.id % (ht.size)
	default:
		execerror("hashtable isn't prepared for that key.type")
	}

	/* look in hash table of existing elements */
	toph = table[v]
	if toph != nil {

		/* collision */
		////
		/* quick test for first node in list, most common case */
		if dcompare(key, toph.key) == 0 {
			if action != H_DELETE {
				return toph
			}
			/* delete from list and free */
			table[v] = toph.next
			freehn(toph)
			ht.count--
			return (nil)
		}
		////
		/* Look through entire list */
		h = toph
		nc := 0
		prev := h
		for {
			h = h.next
			if h == nil {
				break
			}
			nc++
			if dcompare(key, h.key) == 0 {
				break
			}
			prev = h
		}
		if h != nil {
			/* Found.  Delete it from it's current */
			/* position so we can either move it to the top, */
			/* or leave it deleted. */
			prev.next = h.next
			if action == H_DELETE {
				/* delete it */
				freehn(h)
				ht.count--
				return nil
			}
			/* move it to the top of the collision list */
			h.next = toph
			table[v] = h
			return (h)
		}
	}
	////
	/* it wasn't found */
	if action == H_DELETE {
		return nil
	}
	////
	if action == H_LOOK {
		return nil
	}
	////
	h = newhn()
	h.key = key
	h.val = Noval
	ht.count++
	////
	/* Add to top of collision list */
	h.next = toph
	table[v] = h
	////
	return (h)
}

////
//// /*
////  * Look for the symbol for a particular array element, given
////  * a pointer to the main array symbol, and the subscript value.
////  * Values of 'action':
////  *     H_INSERT ==> look for symbol, and if not found, insert
////  *     H_LOOK ==> look for symbol, don't insert
////  *     H_DELETE ==> look for symbol and delete it
////  */
////
//// Symbolp
//// arraysym(Htablep arr,Datum subs,int action)
//// {
//// 	Symbolp s = NULL;
//// 	Symbolp ns;
//// 	Hnodep h;
//// 	Datum key;
////
//// 	if ( arr == NULL )
//// 		execerror("Internal error: arr==0 in arraysym!?");
////
//// 	key = dtoindex(subs);
////
//// 	switch (action) {
//// 	case H_LOOK:
//// 		h = hashtable(arr,key,action);
//// 		if ( h )
//// 			s = h->val.u.sym;
//// 		break;
//// 	case H_INSERT:
//// 		h = hashtable(arr,key,action);
//// 		if ( isnoval(h->val) ) {
//// 			/* New element, initialized to null string */
//// 			ns = newsy();
//// 			ns->name = key;
//// 			ns->stype = VAR;
//// 			*symdataptr(ns) = strdatum(Nullstr);
//// 			h->val = symdatum(ns);
//// 		}
//// 		s = h->val.u.sym;
//// 		break;
//// 	case H_DELETE:
//// 		(void) hashtable(arr,key,action);
//// 		break;
//// 	default:
//// 		execerror("Internal error: bad action in arraysym!?");
//// 	}
//// 	return(s);
//// }
////
//// int
//// arrsize(Htablep arr)
//// {
//// 	return arr->count;
//// }
////
//// int
//// dtcmp(Datum *d1,Datum *d2)
//// {
//// 	return dcompare(*d1,*d2);
//// }
////
//// static int elsize;	/* element size */
//// static INTFUNC2P qscompare;
////
//// /*
////  * Quick Sort routine.
////  * Code by Duane Morse (...!noao!terak!anasazi!duane)
////  * Based on Knuth's ART OF COMPUTER PROGRAMMING, VOL III, pp 114-117.
////  */
////
//// /* Exchange the contents of two vectors.  n is the size of vectors in bytes. */
//// static void
//// memexch(register unsigned char *s1,register unsigned char *s2,register int n)
//// {
//// 	register unsigned char c;
//// 	while (n--) {
//// 		c = *s1;
//// 		*s1++ = *s2;
//// 		*s2++ = c;
//// 	}
//// }
////
//// static void
//// mysort(unsigned char *vec,int nel)
//// {
//// 	register short i, j;
//// 	register unsigned char *iptr, *jptr, *kptr;
////
//// begin:
//// 	if (nel == 2) {	/* If 2 items, check them by hand. */
//// 		if ((*qscompare)(vec, vec + elsize) > 0)
//// 			memexch(vec, vec + elsize, elsize);
//// 		return;
//// 	}
//// 	j = (short) nel;
//// 	i = 0;
//// 	kptr = vec;
//// 	iptr = vec;
//// 	jptr = vec + elsize * nel;
//// 	while (--j > i) {
////
//// 		/* From the righthand side, find first value */
//// 		/* that should be to the left of k. */
//// 		jptr -= elsize;
//// 		if ((*qscompare)(jptr, kptr) > 0)
//// 			continue;
////
//// 		/* Now from the lefthand side, find first value */
//// 		/* that should be to right of k. */
////
//// 		iptr += elsize;
//// 		while(++i < j && (*qscompare)(iptr, kptr) <= 0)
//// 			iptr += elsize;
////
//// 		if (i >= j)
//// 			break;
////
//// 		/* Exchange the two items; k will eventually end up between them. */
//// 		memexch(jptr, iptr, elsize);
//// 	}
//// 	/* Move item 0 into position.  */
//// 	memexch(vec, iptr, elsize);
//// 	/* Now sort the two partitions. */
//// 	if ((nel -= (i + 1)) > 1)
//// 		mysort(iptr + elsize, nel);
////
//// 	/* To save a little time, just start the routine over by hand. */
//// 	if (i > 1) {
//// 		nel = i;
//// 		goto begin;
//// 	}
//// }
////
//// static void
//// pqsort(unsigned char *vec,int nel,int esize,INTFUNC2P compptr)
//// {
//// 	if (nel < 2)
//// 		return;
//// 	elsize = esize;
//// 	qscompare = compptr;
//// 	mysort(vec, nel);
//// }
////
//// /* Return a Noval-terminated list of the index values of an array.  */
//// Datum *
//// arrlist(Htablep arr,int *asize,int sortit)
//// {
//// 	register Hnodepp pp;
//// 	register Hnodep h;
//// 	register Datum *lp;
//// 	register int hsize;
//// 	Datum *list;
////
//// 	pp = arr->nodetable;
//// 	hsize = arr->size;
//// 	*asize = arrsize(arr);
//// 	list = (Datum *) kmalloc((*asize+1)*sizeof(Datum),"arrlist");
////
//// 	lp = list;
//// 	/* visit each slot in the hash table */
//// 	while ( hsize-- > 0 ) {
//// 		/* and traverse its list */
//// 		for ( h=(*pp++); h!=NULL; h=h->next ) {
//// 			*lp++ = h->val.u.sym->name;
//// 		}
//// 	}
//// 	*lp++ = Noval;
//// 	if ( sortit )
//// 		pqsort((unsigned char *)list,*asize,(int)sizeof(Datum),(INTFUNC2P)dtcmp);
//// 	return(list);
//// }
////
//// void
//// hashvisit(Htablep arr,HNODEFUNC f)
//// {
//// 	register Hnodepp pp;
//// 	register Hnodep h;
//// 	register int hsize;
////
//// 	pp = arr->nodetable;
//// 	hsize = arr->size;
//// 	/* visit each slot in the hash table */
//// 	while ( hsize-- > 0 ) {
//// 		/* and traverse its list */
//// 		for ( h=(*pp++); h!=NULL; h=h->next ) {
//// 			if ( (*f)(h) )
//// 				return;	/* used to be break, apparent mistake */
//// 		}
//// 	}
//// }
////
