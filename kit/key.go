package kit

import "os"

/// /*
/// 	The following defines are normally set in mdep.h; they are described
/// 	here only to document the options available.
///
/// 	#define MOVEBITMAP	To use mdep_movebitmap when scrolling text
/// 				regions.  Speeds things up if mdep_pullbitmap
/// 				and mdep_putbitmap are slow.
///  */
///
/// #include "keyoptions.h"
///
/// #include "mdep.h"
///
/// #ifdef PYTHON
/// #include "Python.h"
/// #endif
///
/// #ifndef INT16
/// #define INT16 short
/// #endif
///
/// #ifndef UINT16
/// #define UINT16 unsigned INT16
/// #endif
///

type Unchar byte
type Codep *Unchar

// type Instnodep *Instnode
type Midimessp *Midimessdata

type Noteptr *Notedata

type Bytep string

type Phrasepp *Phrasep
type Symlongp *int
type Symstr string
type Symstrp *Symstr

type Hnodep *Hnode
type Hnodepp []Hnodep
type Kobjectp *Kobject

///
/// /* These macros can be overridden in mdep.h for systems that require */
/// /* special ways of opening text vs. binary files. */
/// #ifndef OPENTEXTFILE
/// #define OPENTEXTFILE(f,file,mode) f=fopen(file,mode)
/// #endif
/// #ifndef OPENBINFILE
/// #define OPENBINFILE(f,file,mode) f=fopen(file,mode)
/// #endif
///

type PORTHANDLE uint64

/// // #define DEBUG
/// // #define BIGDEBUG
/// // #define DEBUGEXEC
///
/// /* If MDEP_MALLOC is defined, then a machine-dependent mdep.h can */
/// /* provide its own macros for kmalloc and kfree. */
/// #ifndef MDEP_MALLOC
///
/// #ifdef MDEBUG
/// #define kmalloc(x,tag) dbgallocate(x,tag)
/// #else
/// #define kmalloc(x,tag) allocate(x,tag)
/// #endif
///
/// #define kfree(x) myfree((char *)(x))
///
/// #endif
///
/// /* It's important that dummyusage() NOT change the value of its argument! */
/// #ifndef dummyusage
/// #ifdef lint
/// #define dummyusage(x) x=x
/// #define dummyset(x) x=0
/// #else
/// #define dummyusage(x)
/// #define dummyset(x)
/// #endif
/// #endif
///

func isspace(c byte) bool {
	if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
		return true
	} else {
		return false
	}
}

func isdigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isalpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

/// #ifndef isalnum
/// #define isalnum(c) (((c)>='a'&&(c)<='z')||((c)>='A'&&(c)<='Z')||(c)=='_'||((c)>='0'&&(c)<='9'))
/// #endif
///
/// #ifdef __STDC__
/// #define NOARG void
/// #else
/// #define NOARG
/// #endif
///
/// #include "phrase.h"
///
/// typedef float DBLTYPE;
/// typedef int (*INTFUNC)(NOARG);

type BYTEFUNC func()

///
/// #ifdef __STDC__
/// typedef void (*STRFUNC)(Symstr);

type BLTINFUNC func() int
type BLTINCODE byte

/// typedef int (*HNODEFUNC)(Hnodep);
/// typedef void (*PATHFUNC)(char*,char*);
/// #else
/// typedef void (*STRFUNC)();
/// typedef void (*BLTINFUNC)();
/// typedef int (*HNODEFUNC)();
/// typedef void (*PATHFUNC)();
/// #endif
///

/// /* These are the values of the Datum type */
/// /* #define D_NONE 0 - no longer used */
const D_NUM = 1
const D_STR = 2
const D_PHR = 3
const D_SYM = 4
const D_DBL = 5
const D_ARR = 6
const D_CODEP = 7
const D_FRM = 8
const D_NOTE = 9
const D_DATUM = 10
const D_FIFO = 11
const D_TASK = 12
const D_WIND = 13
const D_OBJ = 14

// Watch out, these values must line up with the Bytefuncs array
const I_POPVAL = 0
const I_DBLPUSH = 1
const I_STRINGPUSH = 2
const I_PHRASEPUSH = 3
const I_ARREND = 4
const I_ARRAYPUSH = 5
const I_INCOND = 6
const I_DIVCODE = 7
const I_PAR = 8
const I_AMP = 9
const I_LSHIFT = 10
const I_RIGHTSHIFT = 11
const I_NEGATE = 12
const I_TILDA = 13
const I_LT = 14
const I_GT = 15
const I_LE = 16
const I_GE = 17
const I_NE = 18
const I_EQ = 19
const I_REGEXEQ = 20
const I_AND1 = 21
const I_OR1 = 22
const I_NOT = 23
const I_NOOP = 24
const I_POPIGNORE = 25
const I_DEFINED = 26
const I_OBJDEFINED = 27
const I_CURROBJDEFINED = 28
const I_REALOBJDEFINED = 29
const I_TASK = 30
const I_UNDEFINE = 31
const I_DOT = 32
const I_MODULO = 33
const I_ADDCODE = 34
const I_SUBCODE = 35
const I_MULCODE = 36
const I_XORCODE = 37
const I_DOTASSIGN = 38
const I_MODDOTASSIGN = 39
const I_MODASSIGN = 40
const I_VARASSIGN = 41
const I_DELETEIT = 42
const I_DELETEARRITEM = 43
const I_READONLYIT = 44
const I_ONCHANGEIT = 45
const I_EVAL = 46
const I_VAREVAL = 47
const I_OBJVAREVAL = 48
const I_FUNCNAMED = 49
const I_LVAREVAL = 50
const I_GVAREVAL = 51
const I_VARPUSH = 52
const I_OBJVARPUSH = 53
const I_CALLFUNC = 54
const I_OBJCALLFUNCPUSH = 55
const I_OBJCALLFUNC = 56
const I_ARRAY = 57
const I_LINENUM = 58
const I_FILENAME = 59
const I_FORIN1 = 60
const I_FORIN2 = 61
const I_POPNRETURN = 62
const I_STOP = 63
const I_SELECT1 = 64
const I_SELECT2 = 65
const I_SELECT3 = 66
const I_PRINT = 67
const I_GOTO = 68
const I_TFCONDEVAL = 69
const I_TCONDEVAL = 70
const I_CONSTANT = 71
const I_DOTDOTARG = 72
const I_VARG = 73
const I_CURROBJEVAL = 74
const I_CONSTOBJEVAL = 75
const I_ECURROBJEVAL = 76
const I_EREALOBJEVAL = 77
const I_RETURNV = 78
const I_RETURN = 79
const I_QMARK = 80
const I_FORINEND = 81
const I_DOSWEEPCONT = 82
const I_CLASSINIT = 83
const I_PUSHINFO = 84
const I_POPINFO = 85
const I_NARGS = 86
const I_TYPEOF = 87
const I_XY2 = 88
const I_XY4 = 89

// watch out, these values are tied to the Codesize array
const IC_NONE = 0
const IC_NUM = 1
const IC_STR = 2
const IC_DBL = 3
const IC_SYM = 4
const IC_PHR = 5
const IC_INST = 6
const IC_FUNC = 7
const IC_BLTIN = 8

// Watch out, these values must line up with the Bltinfuncs array
const BI_NONE = 0
const BI_SIZEOF = 1
const BI_OLDNARGS = 2
const BI_ARGV = 3
const BI_MIDIBYTES = 4
const BI_SUBSTR = 5
const BI_SBBYES = 6
const BI_RAND = 7
const BI_ERROR = 8
const BI_PRINTF = 9
const BI_READPHR = 10
const BI_EXIT = 11
const BI_OLDTYPEOF = 12
const BI_SPLIT = 13
const BI_CUT = 14
const BI_STRING = 15
const BI_INTEGER = 16
const BI_PHRASE = 17
const BI_FLOAT = 18
const BI_SYSTEM = 19
const BI_CHDIR = 20
const BI_TEMPO = 21
const BI_MILLICLOCK = 22
const BI_CURRTIME = 23
const BI_FILETIME = 24
const BI_GARBCOLLECT = 25
const BI_FUNKEY = 26
const BI_ASCII = 27
const BI_MIDIFILE = 28
const BI_REBOOT = 29
const BI_REFUNC = 30
const BI_DEBUG = 31
const BI_PATHSEARCH = 32
const BI_SYMBOLNAMED = 33
const BI_LIMITSOF = 34
const BI_SIN = 35
const BI_COS = 36
const BI_TAN = 37
const BI_ASIN = 38
const BI_ACOS = 39
const BI_ATAN = 40
const BI_SQRT = 41
const BI_POW = 42
const BI_EXP = 43
const BI_LOG = 44
const BI_LOG10 = 45
const BI_REALTIME = 46
const BI_FINISHOFF = 47
const BI_SPRINTF = 48
const BI_GET = 49
const BI_PUT = 50
const BI_OPEN = 51
const BI_FIFOSIZE = 52
const BI_FLUSH = 53
const BI_CLOSE = 54
const BI_TASKINFO = 55
const BI_KILL = 56
const BI_PRIORITY = 57
const BI_ONEXIT = 58
const BI_SLEEPTILL = 59
const BI_WAIT = 60
const BI_LOCK = 61
const BI_UNLOCK = 62
const BI_OBJECT = 63
const BI_OBJECTLIST = 64
const BI_WINDOBJECT = 65
const BI_SCREEN = 66
const BI_SETMOUSE = 67
const BI_MOUSEWARP = 68
const BI_BROWSEFILES = 69
const BI_COLORSET = 70
const BI_COLORMIX = 71
const BI_SYNC = 72
const BI_OLDXY = 73
const BI_CORELEFT = 74
const BI_PRSTACK = 75
const BI_PHDUMP = 76
const BI_NULLFUNC = 77

// Methods, much like built-in functions
const O_SETINIT = 78
const O_ADDCHILD = 79
const O_REMOVECHILD = 80
const O_CHILDUNDER = 81
const O_CHILDREN = 82
const O_INHERITED = 83
const O_ADDINHERIT = 84
const O_SIZE = 85
const O_REDRAW = 86
const O_CONTAINS = 87
const O_XMIN = 88
const O_YMIN = 89
const O_XMAX = 90
const O_YMAX = 91
const O_LINE = 92
const O_BOX = 93
const O_FILL = 94
const O_STYLE = 95
const O_MOUSEDO = 96
const O_TYPE = 97
const O_TEXTCENTER = 98
const O_TEXTLEFT = 99
const O_TEXTRIGHT = 100
const O_TEXTHEIGHT = 101
const O_TEXTWIDTH = 102
const O_SAVEUNDER = 103
const O_RESTOREUNDER = 104
const O_PRINTF = 105
const O_DRAWPHRASE = 106
const O_SCALETOGRID = 107
const O_VIEW = 108
const O_TRACKNAME = 109
const O_SWEEP = 110
const O_CLOSESTNOTE = 111
const O_MENUITEM = 112
const BI_LSDIR = 113
const BI_REKEYLIB = 114
const BI_FIFOCTL = 115
const BI_MDEP = 116
const BI_HELP = 117
const O_MENUITEMS = 118
const O_ELLIPSE = 119
const O_FILLELLIPSE = 120
const O_SCALETOWIND = 121
const BI_ATTRIBARRAY = 122
const BI_ONERROR = 123
const BI_MIDI = 124
const BI_BITMAP = 125
const BI_OBJECTINFO = 126
const O_FILLPOLYGON = 127

const IO_STD = 1
const IO_REDIR = 2

const CUT_NORMAL = 0
const CUT_TRUNCATE = 1
const CUT_INCLUSIVE = 2
const CUT_TIME = 3
const CUT_FLAGS = 4
const CUT_TYPE = 5
const CUT_CHANNEL = 6
const CUT_NOTTYPE = 7

// These values start here because they continue from NT_ON, NT_OFF, ...
const M_CHANPRESSURE = 16
const M_CONTROLLER = 32
const M_PROGRAM = 64
const M_PRESSURE = 128
const M_PITCHBEND = 256
const M_SYSEX = 512
const M_POSITION = 1024
const M_CLOCK = 2048
const M_SONG = 4096
const M_STARTSTOPCONT = 8192
const M_SYSEXTEXT = 16384

// Used for return values of mdep_waitfor()
const K_CONSOLE = 1
const K_MOUSE = 2
const K_MIDI = 4
const K_WINDEXPOSE = 8
const K_WINDRESIZE = 16
const K_TIMEOUT = 32
const K_ERROR = 64
const K_NOTHING = 128
const K_QUIT = 256
const K_PORT = 512

const T_STATUS = 1
const T_KILL = 2
const T_PAUSE = 4
const T_RESTART = 8

// For Symbol.flags
const S_READONLY = 1
const S_SEEN = 2

// Offset applied to return values of fromconsole() for function keys
const FKEYBIT = 1024
const KEYUPBIT = 2048
const KEYDOWNBIT = 4096
const NFKEYS = 24

// This flag allows 'phrase % number' operation to be zero or one-based
// I guess 1-based is more natural (ie. 'a,b,c'%1 == 'a'), but 0-based
// would make it more array-like.  Note that the existing library of
// keykit functions depends on phrases being one-based, so this can't
// be changed unless you change the library.  Not advised, but I can
// envision changing it if/when other strong reasons warrant it.

const PHRASEBASE = 1 // affects 'phrase % number' operation

// These are the amounts used for bulk allocation of various structures
const ALLOCSY = 64
const ALLOCIN = 512
const ALLOCSCH = 64
const ALLOCHN = 128
const ALLOCTF = 32
const ALLOCINT = 32
const ALLOCDN = 32
const ALLOCFD = 32
const ALLOCLK = 32
const ALLOCOBJ = 64

const H_INSERT = 0
const H_LOOK = 1
const H_DELETE = 2

const COMPLAIN = 1
const NOCOMPLAIN = 0

// default separator for split() on strings
const DEFSPLIT = " \t\n"

/// #define ISDEBUGON (Debug!=NULL && *Debug!=0)

// KeyKit programs get parsed and 'compiled' into lists of Inst's.
// Each program segment (e.g. a function) is kept in a separate list.
// The Inst's are maintained as linked lists rather than as arrays,
// so that they can be dynamically allocated (hence no program length
// restrictions) and so that separate intruction segements (e.g. for each
// user-defined function) can be more easily maintained.

type Inst struct {
	i interface{}
	/// 	BLTINCODE bltin;
	/// 	BYTEFUNC func;
	/// 	DBLTYPE dbl;
	/// 	long val;
	/// 	Symstr str;
	/// 	Symbolp sym;
	/// 	Instnodep in;
	/// 	Codep ip;
	/// 	Phrasep phr;
	/// 	Unchar bytes[8];
}

// values of ival in Inst
const IBREAK = 8
const ICONTINUE = 9

/// union str_union {
/// 	Symstr str;
/// 	Unchar bytes[8];
/// };
/// union dbl_union {
/// 	DBLTYPE dbl;
/// 	Unchar bytes[8];	/* WRONG! (when DBLTYPE is not of size 4) */
/// };
/// union sym_union {
/// 	Symbolp sym;
/// 	Unchar bytes[8];
/// };
/// union phr_union {
/// 	Phrasep phr;
/// 	Unchar bytes[8];
/// };
/// union ip_union {
/// 	Codep ip;
/// 	Unchar bytes[8];
/// };
///
const DONTPUSH = 0x800

// This is an arbitrary magic number
const FORINJUNK = 0x1234

type Instcode struct {
	u     Inst
	itype int
}

// Datum is the interpreter stack type
type Datum struct { // interpreter stack type
	dtype int16 // uses D_* values
	u     interface{}

	/// 	union Datumu {
	/// 		long	val;	/* D_NUM */
	/// 		DBLTYPE	dbl;	/* D_DBL */
	/// 		Symstr	str;
	/// 		Symbolp sym;
	/// 		Phrasep	phr;
	/// 		Htablep arr;	/* If type==D_ARR (array), this is a ptr to */
	/// 				/* ptr to elements (double indirect because */
	/// 				/* arrays are manipulated by reference) */
	/// 		Codep codep;		/* D_CODEP */
	/// 		struct Datum *frm;	/* D_FRM */
	/// 		struct Datum *datum;	/* D_DATUM */
	/// 		Noteptr note;		/* D_NOTE */
	/// 		struct Ktask *task;	/* D_TASK */
	/// 		struct Fifo *fifo;	/* D_FIFO */
	/// 		struct Kwind *wind;	/* D_WIND */
	/// 		Kobjectp obj;		/* D_OBJ */
	/// 	} u;

}

type Dnode struct {
	d    Datum
	next *Dnode
}

type Hnode struct {
	next Hnodep
	key  Datum
	val  Datum
}

const HT_TOBECHECKED = 1

type Htable struct {
	size      int /* size of nodetable */
	count     int /* number of actual elements */
	h_used    int16
	h_tobe    int16
	nodetable []Hnodep
	h_next    Htablep
	h_prev    Htablep
	h_state   int16 /* HT_TOBECHECKED or 0 */
}
type Htablep *Htable

///
/// typedef Htablep *Htablepp;
///
/// /* Symbol entries are created during the parsing of a keykit program, */
/// /* and are typically pointed-to by Inst entries.  To make array elements */
/// /* work like normal variables (ie. avoiding lots of special cases in the */
/// /* routines which execute Inst's), an array is represented by a head Symbol */
/// /* which points to a hash table pointing to Symbols, one per array element.*/
/// /* References to array elements get turned (at execution time) into */
/// /* references to the Symbol (possibly newly generated at execution time) */
/// /* for the desired element, after which it looks like a normal variable.  */
/// /* When an array is passed as an argument to a function, it is passed */
/// /* by reference, allowing modification of the passed array */
///
type Symbol struct { // symbol table entry
	next Symbolp
	name Datum // For normal variables, this is the name.
	// For array elements, it's the index value.
	stype    int   // UNDEF, VAR, MACRO, TOGLOBSYM, -or- a keyword
	stackpos int   // 0 is global, >0 is parameter, <0 is local
	flags    byte  // S_READONLY, etc.
	onchange Codep // to execute when variable changes value
	sd       Datum // Value of global VARs and array elements.
}

type Symbolp *Symbol

// When parsing a keykit program, Contexts are used to determine what
// Symbols are local (e.g. parameters within a user-defined function)
// and which are global.
type Context struct {
	next     *Context
	cfunc    Symbolp
	symbols  Htablep
	localnum int
	paramnum int
}

type bltinfo struct {
	name      string
	bltinfunc BLTINFUNC
	bltindex  BLTINCODE
}

var builtins []bltinfo
var Bytefuncs []BYTEFUNC
var Bytenames []string
var Bltinfuncs []BLTINFUNC

type Ktaskp *Ktask

const OFF_USER = 0
const OFF_INTERNAL = 1

const FIFOTYPE_UNTYPED = 0
const FIFOTYPE_BINARY = 1
const FIFOTYPE_LINE = 2
const FIFOTYPE_FIFO = 3
const FIFOTYPE_ARRAY = 4

type Sched struct {
	next    *Sched
	clicks  int     // scheduled time
	stype   uint8   // SCH_*
	offtype uint8   // for SCH_NOTEOFF)
	monitor bool    // If true, add to Monitorfifo
	phr     Phrasep // for SCH_PHRASE
	note    Noteptr // for SCH_NOTEOFF and SCH_PHRASE.
	task    Ktaskp
	repeat  int // if > 0, a repeat time.
}

type Tofree struct {
	note Noteptr
	next *Tofree
}

type Ktask struct {
	pc              *Unchar /* current instruction */
	nextrun         Ktaskp  /* Used for the Running list */
	stack           *Datum  /* the stack (duh) */
	stacksize       int     /* allocated size of stack */
	stackp          *Datum  /* next free spot on stack */
	stackend        *Datum  /* just past last allocated element */
	stackframe      *Datum  /* beginning of current stack frame */
	arg0            *Datum  /* argument 0 of current stack frame */
	state           int     /* T_FREE, T_RUNNING, etc. */
	nexted          int     /* number of nested instruction streams */
	tid             int     /* task id, >= 0 */
	priority        int     /* 0=normal, >0 is high priority */
	first           Codep   /* first instruction */
	schedcnt        int     /* number of scheduled events due to this task */
	cnt             int     /* number of instructions executed */
	tmp             int     /* for temporary use as a flag, counter, etc. */
	twait           Ktaskp  /* if state==T_WAITING, we're waiting for this */
	qmarkframe      *Datum  /* keeps track of ? (in ph{??.chan==1} ) */
	qmarknum        int     /* ? number (as in ph{??.number<10} ) */
	rminstruct      int     /* says if instructions should be freed */
	parent          Ktaskp
	anychild        int   /* If this task has any children */
	anywait         int   /* If any tasks are waiting for this one */
	fifo            *Fifo /* Task is blocked on this fifo. */
	onexit          Codep
	onexitargs      *Dnode
	ontaskerror     Codep
	ontaskerrorargs *Dnode
	ontaskerrormsg  Symstr
	nxt             Ktaskp /* Used for the Toptp and Freetp lists */
	tmplist         Ktaskp /* Used for temporary lists. */
	linenum         int
	filename        Symstr
	lock            *Lknode
	obj             Kobjectp /* object we're running method of */
	realobj         Kobjectp /* object we're running method on behalf of */
	method          Symstr
	pend_bltin      BLTINCODE /* pending function (when T_OBJBLOCKED). */
	pend_npassed    int       /* for pending function */
}

type Fifodata struct {
	next *Fifodata
	d    Datum
}

type Fifo struct {
	head         *Fifodata // Points to last "put"
	tail         *Fifodata // Points to next "get"
	size         int
	flags        int        // For FIFO_* bitflags, see above
	fp           *os.File   // If non-NULL, this is a file fifo
	t            Ktaskp     // This task is blocked on this fifo
	port         PORTHANDLE // If FIFO_ISPORT is set, this is used.
	num          int
	next         *Fifo
	fifoctl_type int    // type of data read from fifo
	linebuff     string // Saved data for FIFO_LINE
	linesize     int    // Total size of linebuff (for makeroom)
	linesofar    int    // How much actually used
}

type Lknode struct {
	name   Symstr // Only used in Toplk list.
	owner  *Ktask
	next   *Lknode // Only used in Toplk list.
	notify *Lknode // List of pending locks with same name
}

type Kobject struct {
	id          int
	symbols     Htablep
	inheritfrom Kobjectp /* list of objects we inherit from */
	nextinherit Kobjectp /* next in that list */
	children    Kobjectp
	nextsibling Kobjectp
	onext       Kobjectp
}

const MIDI_OUT_DEVICES = 64
const MIDI_IN_DEVICES = 64
const MIDI_IN_PORT_OFFSET = 64
const MAX_PORT_VALUE = 128 // MIDI_IN_DEVICES+MIDI_OUT_DEVICES
const PORTMAP_SIZE = 65    // MIDI_IN_DEVICES+1

type Midiport struct {
	opened   int
	name     Symstr
	private1 int // mdep layer can use this for whatever it wants
}

// The index into this array is the port number minus 1.
var Midiinputs []Midiport
var Midioutputs []Midiport

// Used for the first argument of the mdep_midi function.
const MIDI_OPEN_OUTPUT = 0
const MIDI_CLOSE_OUTPUT = 1
const MIDI_OPEN_INPUT = 2
const MIDI_CLOSE_INPUT = 3

// values of T->state
// T_RUNNING is a task that is currently free, available for use
const T_FREE = 0

// T_RUNNING is an active task
const T_RUNNING = 1

// T_BLOCKED is a task blocked on a fifo
const T_BLOCKED = 2

// T_SLEEPTILL is a task that is waiting because of a sleeptill() function
const T_SLEEPTILL = 3

// T_STOPPED is a task stopped
const T_STOPPED = 4

// T_SCHED is a task scheduled as a result of realtime()
const T_SCHED = 5

// T_WAITING is a task waiting for a signal.
const T_WAITING = 6

// T_LOCKWAIT is a task waiting for a lock.
const T_LOCKWAIT = 7

// Bits that can be set in Fifo.flags.  When a fifo is
// created, all of the bits default to 0.
const FIFO_OPEN = 1            // if set, FIFO is open
const FIFO_PIPE = (1 << 1)     // fifo is connected to a pipe (vs. a file)
const FIFO_WRITE = (1 << 2)    // fifo is used for writing
const FIFO_READ = (1 << 3)     // fifo is used for reading
const FIFO_SPECIAL = (1 << 4)  // fifo is special (MIDI, CONSOLE, MOUSE), it can't be closed by the user. */
const FIFO_NORETURN = (1 << 5) // tells whether to use ret()
const FIFO_APPEND = (1 << 6)   // fifo is writing
const FIFO_ISPORT = (1 << 7)   // fifo is attached to a mdep_openport()

func fifonum(f *Fifo) int {
	return f.num
}

///
const FIFOINC = 64

const SCH_NOTEOFF = 0
const SCH_PHRASE = 1
const SCH_WAKE = 2

const FREEABLE = 1
const NOTFREEABLE = 0

const MAXPRIORITY = 1000
const DEFPRIORITY = 500

/// #define disabled(s) ((s)->clicks==MAXCLICKS)
///
/// extern Sched *Topsched;
/// extern long Earliest;
/// extern Htablep Keywords;
/// extern Htablep Macros;
///
/// extern int Nblocked;
/// extern int Nwaiting;
/// extern int Nsleeptill;
///
/// extern Ktaskp T;
/// extern Ktaskp Tboot;
/// extern Ktaskp Running;
/// extern int Currpriority;
/// extern Codep Ipop;
/// extern Fifo *Midi_in_f, *Midi_out_f;
/// extern Fifo *Consinf, *Consoutf, *Mousef;
/// extern int Consolefd, Midifd, Displayfd;
/// extern int Default_fifotype;
/// extern Kobjectp Topobj;
/// extern long Nextobjid;
/// extern Codep Idosweep;
/// #ifdef OLDSTUFF
/// extern Codep Idodrag;
/// #endif
///
/// #define Pc (T->pc)
/// #define Stack (T->stack)
/// #define Stackp (T->stackp)
/// #define Stackend (T->stackend)
/// #define Firstframe (T->firstframe)
/// #define arg0_of_frame(f) ((f)-FRAMEHEADER-numval(*((f)-FRAME_VARSIZE_OFFSET)))
/// #define npassed_of_frame(f) ((f)-FRAME_NPASSED_OFFSET)
/// #define func_of_frame(f) ((f)-FRAME_FUNC_OFFSET)
/// #define fname_of_frame(f) ((f)-FRAME_FNAME_OFFSET)
/// #define ARG(n) (*(T->arg0+(n)))
///
/// #define isglobal(s) ((s)->stackpos==0)

func isnoval(d Datum) bool {
	dval := d.u.(int)
	noval := Noval.u.(int)
	if dval == noval && d.dtype == Noval.dtype {
		return true
	}
	return false
}

/// /* #define isnoval(d) (((d).type==Noval.type)&&((d).u.val==Noval.u.val)) */
/// #define isnoval(d) (((d).u.val==Noval.u.val) && ((d).type==Noval.type) )

/// #define CHKNOVAL(d,s) if(isnoval(d)){ \
/// 		sprintf(Msg1,"Uninitialized value (Noval) can't be handled by %s",s); \
/// 		execerror(Msg1); \
/// 	}

///
/// /* In function calls, there are PREARGSIZE things on the stack before */
/// /* the argument values.  Currently this is the func/obj/method values. */
/// #define PREARGSIZE 4
/// #define FRAME_PREARG_FUNC_OFFSET 0
/// #define FRAME_PREARG_REALOBJ_OFFSET 1
/// #define FRAME_PREARG_OBJ_OFFSET 2
/// #define FRAME_PREARG_METHOD_OFFSET 3
///
/// /* size of frame header */
/// #define FRAMEHEADER 9
///
/// /* offset of various things in the frame header */
/// #define FRAME_VARSIZE_OFFSET 1
/// #define FRAME_NPASSED_OFFSET 2
/// #define FRAME_METHOD_OFFSET 3
/// #define FRAME_OBJ_OFFSET 4
/// #define FRAME_REALOBJ_OFFSET 5
/// #define FRAME_PC_OFFSET 6
/// #define FRAME_FUNC_OFFSET 7
/// #define FRAME_FNAME_OFFSET 8
/// #define FRAME_LNUM_OFFSET 9
///
/// #ifdef __STDC__
/// typedef void (*PFCHAR)(char*);
/// typedef int (*INTFUNC2P)(unsigned char*, unsigned char*);
/// #else
/// typedef void (*PFCHAR)();
/// typedef int (*INTFUNC2P)();
/// #endif
///
/// extern Instcode Stopcode;
///
/// /* The following macros are for (premature as always) optimization */
///
/// #define symname(s) ((s)->name.type==D_STR?(s)->name.u.str:dtostr((s)->name))
///
/// extern Datum _Dnumtmp_;

func numdatum(l int) Datum {
	return Datum{dtype: D_NUM, u: l}
}

///
/// #define phnumused(p) ((p)->p_used)
/// #define phreallyused(p) ((p)->p_used+(p)->p_tobe)
///
/// #define phincruse(p) {if((p)!=NULL){((p)->p_tobe)++;}}

func phdecruse(p Phrasep) {
	if p != nil {
		p.p_tobe -= 1
		if p.p_used+p.p_tobe <= 0 {
			addtobechecked(p)
		}
	}
}

/// #define arrincruse(a) {if((a)!=NULL){((a)->h_tobe)++;}}

func arrdecruse(a Htablep) {
	if a != nil {
		a.h_tobe -= 1
		if a.h_used+(a.h_tobe) <= 0 {
			httobechecked(a)
		}
	}
}

///
/// #define incruse(d) {if((d).type==D_PHR)phincruse((d).u.phr) else if((d).type==D_ARR)arrincruse((d).u.arr)}
/// #define decruse(d) {if((d).type==D_PHR)phdecruse((d).u.phr) else if((d).type==D_ARR)arrdecruse((d).u.arr)}
///
/// #define peekinto(x) x = *(Stackp - 1)
/// #define popinto(x) \
/// 	if(Stackp==Stack) \
/// 		underflow(); \
/// 	else { \
/// 		x = *(--Stackp); \
/// 		decruse(x); \
/// 	}
/// #define popnodecr(x) \
/// 	if(Stackp==Stack) \
/// 		underflow(); \
/// 	else { \
/// 		x = *(--Stackp); \
/// 	}
/// #define pushchk if(Stackp>=Stackend)expandstack(T);
/// #define pushstk(x) *Stackp++ = (x);
///
/// /* #define enoughstack(n) if((Stackp+(n))>=Stackend)expandstack(T) */
///
/// /* Both pushm and pushexp are used to put things on the stack.  If it's an */
/// /* expression (ie. you don't want it evaluated multiple times), use pushexp(). */
///
/// #define pushm(x) pushchk;incruse(x);pushstk(x)
/// #define pushfunc(x) pushchk;pushstk(x)
/// #define pushnoinc(x) pushchk;pushstk(x)
/// #define pushnum(n) pushchk;pushstk(numdatum(n))
/// #define pushstr(p) pushchk;pushstk(strdatum(p))
/// #define pushm_notph(x) pushchk;incruse(x);pushstk(x)
/// #define pushexp(x) pushchk;{Datum dtmp;dtmp=(x);incruse(dtmp);pushstk(dtmp);}
///
/// #define pushnoinc_nochk(x) pushstk(x)
/// #define pushnum_nochk(n) pushstk(numdatum(n))
/// #define pushexp_nochk(x) {Datum dtmp;dtmp=(x);incruse(dtmp);pushstk(dtmp);}
/// #define pushfunc_nochk(x) pushstk(x)
///
/// #define setpc(i) Pc=(Unchar*)(i)
/// #define nextinode(in) ((in)->inext)
///
/// #define SCAN_FUNCCODE(p) *(p)++
/// #define SCAN_BLTINCODE(p) *(p)++
/// #define SKIP_SYMCODE(p) p+=Codesize[IC_SYM]
/// #define BLTINOF(p) *(p)
/// #define use_strcode() scan_strcode(&(Pc))
/// #define use_symcode() scan_symcode(&(Pc))
/// #define use_numcode() scan_numcode(&(Pc))
/// #define use_dblcode() scan_dblcode(&(Pc))
/// #define use_ipcode() scan_ipcode(&(Pc))
/// #define use_phrcode() scan_phrcode(&(Pc))
/// #define SCAN_NUMCODE(p) ((((B___=*(p)++) & 0xc0)==0)?(long)B___:scan_numcode1(&p,B___))
/// extern Unchar B___;
///
/// #define currsequence() (Seqnum)
/// #define usesequence() (++Seqnum)
///
/// #define codeis(c,v) (((c).u.func==(BYTEFUNC)(v)) && ((c).type==IC_FUNC))
///
/// #define chkrealoften() if(++Chkcount>*Throttle2){chkinput();chkoutput();Chkcount=0;}else
///
/// /* legal first characters of names */
/// #define isname1char(c) (isalpha(c)||c=='_')
/// /* legal subsequent characters of names */
/// #define isnamechar(c) (isalnum(c)||c=='_')
///
/// #ifdef FDEBUG
/// void prfunc(void (*)(NOARG));
/// #define PRFUNC(x) if(*Debug)prfunc(x),tprint("\n")
/// #else
/// #define PRFUNC(x)
/// #endif
///
func numval(d Datum) int {
	if d.dtype == D_NUM {
		return d.u.(int)
	} else {
		return getnumval(d, false)
	}
}

/// #define roundval(d) ((d).type==D_NUM?(d).u.val:getnumval(d,1))

func dblval(d Datum) float32 {
	if d.dtype == D_DBL {
		return d.u.(float32)
	} else {
		return getdblval(d)
	}
}

/// #define dblval(d) ((d).type==D_DBL?(double)((d).u.dbl):getdblval(d))

/// #define numtype(d) ((d).type==D_NUM?(d).type:getnmtype(d))
///
/// #define FSTACKSIZE	16	/* Maximum # of nested files being read */
/// #define NPARAMS		64	/* Maximum # of macro parameters */
/// #define INITSTACKSIZE	100
///
/// extern int Codesize[9];
/// extern int Indef, Inclass;
/// extern int Globaldecl;
/// extern int Inparams;
/// extern int Inselect;
/// extern int Paramnum;
/// extern int Niseg;
/// extern char *Pyytext;
/// extern char *Progname;
/// extern char *Yytext;
/// extern char *Buffer;
/// extern char *Msg1, *Msg2, *Msg3;
/// extern long Msg1size, Msg2size, Msg3size;
/// extern unsigned int Buffsize;
/// extern int yydebug;
/// extern FILE *Fin, *Fout;
/// extern Datum Zeroval, Noval, Nullval;
/// extern Datum Str_x0, Str_y0, Str_x1, Str_y1, Str_x, Str_y, Str_button;
/// extern Datum Str_type, Str_mouse, Str_down, Str_up, Str_drag, Str_move;
/// extern Datum Str_lowest, Str_highest, Str_earliest, Str_latest, Str_modifier;
/// extern Datum Str_default, Str_w, Str_r, Str_init;
/// extern Datum Str_get, Str_set, Str_newline;
/// extern Datum Str_red, Str_green, Str_blue, Str_grey, Str_surface;
/// extern Datum Str_finger, Str_hand, Str_xvel, Str_yvel;
/// extern Datum Str_width, Str_height;
/// extern Datum Str_proximity, Str_orientation, Str_eccentricity;
/// #ifdef MDEP_OSC_SUPPORT
/// extern Datum Str_elements, Str_seconds, Str_fraction;
/// #endif
/// extern int Doconsole, Gotanint;
/// extern Instnodep *Future, *Iseg, *Lastin;
/// extern Instnodep Beingread;
/// extern Symstr _Icstr;
/// extern DBLTYPE _Icdbl;
/// extern Symbolp _Icsym;
/// extern long _Icval;
/// extern Phrasep _Icphr;
/// extern Codep _Icin;
/// extern Phrasep Tobechecked;
/// extern Htablep Htobechecked;
/// extern int Chkstuff;
/// extern int Keycnt;
/// extern int Argc;
/// extern char **Argv;
/// extern Codep Fkeyfunc[NFKEYS];
/// extern int Errors;
/// extern int Errfileit;
/// extern int Pmode;
/// extern int Lineno;
///
/// /*
///  * first index is 0 (for defaults) or 0-based input port number+1,
///  * second index is 0-based channel.  The value is the 1-based output port #.
///  */
/// extern int Portmap[PORTMAP_SIZE][16];
///
/// extern Context *Topct, *Currct;
/// extern Htablep Topht;
/// extern Htablep Tasktable;
/// extern char *Scachars[];
/// extern Datum *Errorfuncd, *Rebootfuncd, *Printfuncd;
/// extern Datum *Intrfuncd;
/// extern long Tempo;
/// extern float Milliperclick;
/// extern int Setintr;
/// extern int Macrosused;
/// extern char *Infile;
/// extern char Tty[];
/// extern int Erasechar, Killchar, Intrchar, Eofchar;
/// extern int Nonotealloc;
/// extern long Seqnum;
/// extern Datum *Colorfuncd, *Redrawfuncd, *Resizefuncd, *Exitfuncd;
/// extern Htablepp Track;
/// extern Htablepp Chancolormap;
/// extern int Midiok;
/// extern long Chkcount;
///
/// /* Global keykit variables */

// var Clicks, Merge, Debug, Now, Sync, Lag, Graphics, Mergefilter Symlongp

/// extern Symlongp Mergeport1, Mergeport2;
/// extern Symlongp Debugwait, Debugmidi, Debugrun, Optimize, Debugfifo, Debugmouse;
/// extern Symlongp Clocksperclick, Clicksperclock, Inputistty, Recsysex;
/// extern Symlongp Filter, Record, Recsched, Debugdraw, Recfilter, Recinput;
/// extern Symlongp Loadverbose, Throttle2, Warnnegative, Midifilenoteoff;
/// extern Symlongp Drawcount, Mousedisable, Forceinputport, Mfsysextype;
/// extern Symlongp Lowcorelim, Arraysort, Tempotrack, Debugoff, Fakewrap;
/// extern Symlongp Defrelease, Onoffmerge, Grablimit, Mfformat, Defoutport;
/// extern Symlongp Taskaddr, Debuginst, Prepoll, Debugmalloc, Linetrace;
/// extern Symlongp Debugkill, Debugkill1, Consecho, Abortonint, Abortonerr;
/// extern Symlongp Checkcount, Isofuncwarn, Resizefix, Consupdown, Slashcheck;
/// extern Symlongp Novalval, Eofval, Intrval, Nowoffset, Directcount, SubstrCount;
/// extern Symlongp Printsplit, Throttle, Defpriority, Showsync, Echoport;
/// extern Symlongp Offsetpitch, Offsetfilter, Monitor_fnum, Consecho_fnum;
/// extern Symlongp Offsetportfilter;
/// extern Symlongp Consinfnum, Consoutfnum, Midi_in_fnum, Midi_out_fnum;
/// extern Symlongp Redrawignoretime, Resizeignoretime, Mousefnum, Warningsleep;
/// extern Symlongp Millires, Milliwarn, Mousefifolimit, Minbardx, Midithrottle;
/// extern Symlongp Numinst1, Numinst2, Kobjectoffset, Mousemoveevents;
/// extern Symlongp Debuggesture;
/// extern Symlongp Chancolors;
/// extern Phrasepp Currphr, Recphr;
/// extern Symstrp Keypath, Musicpath, Keyroot, Initconfig;
/// extern Symstrp Printsep, Printend, Pathsep, Dirseparator, Devmidi, Machine;
/// extern int Dbg, Inerror, Usestdio, ReadytoEval;
/// extern void (*Fatalfunc)(char *);
/// extern void (*Diagfunc)(char *);
/// extern void checkdebug();
///
/// int yyparse(NOARG);
///
/// #ifndef KEYKITRC
/// #define KEYKITRC "keykit.rc"
/// #endif
///
/// #ifndef ARRAYHASHSIZE
/// /* #define ARRAYHASHSIZE 503 */
/// #define ARRAYHASHSIZE 251
/// #endif
///
/// #ifndef DEFLOWLIM
/// #define DEFLOWLIM 50000
/// #endif
///
/// #ifndef BUFSIZ
/// #define BUFSIZ 512
/// #endif
///
/// #define BIGBUFSIZ 4096
///
/// #ifndef MILLICLOCK
/// #define MILLICLOCK mdep_milliclock()
/// #endif
///
/// #ifndef MIDISENDLIMIT
/// #define MIDISENDLIMIT 100
/// #endif
///
/// #ifndef MINTEMPO
/// #define MINTEMPO 10000
/// #endif
///
/// #ifdef STATMIDI
/// Hey, mdep_statmidi is no longer used!
/// #endif
///
/// #ifdef OLDSTUFF
/// #ifndef STATMIDI
/// #define STATMIDI mdep_statmidi()
/// #endif
/// #endif
///
/// #ifndef CORELEFT
/// #define CORELEFT mdep_coreleft()
/// #endif
///
/// #ifndef PATHSEP
/// #define PATHSEP ":"
/// #endif
///
/// #ifndef SEPARATOR
/// #define SEPARATOR "/"
/// #endif
///
/// /* The MAIN macro is a hook by which a machine-dependent modification */
/// /* to the main() calling sequence can be done.  */
/// #ifndef MAIN
/// #define MAIN(ac,av) main(ac,av)
/// #endif
///
/// #define NONAMEPREFIX "__"
///
/// #define KEYVERSION "8.0"
///
/// /* These values might, e.g., be set to 'p' and 'P', if that's what the */
/// /* real function keys put out.  Depends on what mdep_getconsole() in mdep.c does. */
/// #ifndef FKEY1
/// #define FKEY1 'a'
/// #define FKEY13 'm'
/// #endif
///
/// #ifndef PIPES
/// #ifdef unix
/// #define PIPES	1
/// #endif
/// #endif
///
/// #ifndef YYMAXDEPTH
/// #define YYMAXDEPTH 9600
/// #endif
///
/// #define MAX_POLYGON_POINTS 8
///
/// #define KEYNCOLORS 64
///
/// #include "grid.h"
/// #include "d_grid.h"
/// #include "d_kwind.h"
///
/// #include "d_main.h"
/// #include "d_task.h"
/// #include "d_fifo.h"
/// #include "d_code2.h"
/// #include "d_code.h"
/// #include "d_util.h"
/// #include "d_sym.h"
/// #include "d_bltin.h"
/// #include "d_meth.h"
/// #include "d_phrase.h"
/// #include "d_misc.h"
/// #include "d_fsm.h"
/// #include "d_keyto.h"
/// #include "d_mdep1.h"
/// #include "d_mdep2.h"
/// #include "d_mfin.h"
/// #include "d_midi.h"
/// #include "d_real.h"
/// #include "d_view.h"
/// #include "d_regex.h"
/// #include "d_clock.h"
/// #include "d_menu.h"
///
/// #ifdef FFF
/// extern FILE *FF;
/// #endif
/// #define funcinst(x) realfuncinst((BYTEFUNC)(x))
