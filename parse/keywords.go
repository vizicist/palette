package parse

var Keywords map[string]int = map[string]int{
	"function": FUNC,
	"return":   RETURN,
	"if":       IF,
	"else":     ELSE,
	"while":    WHILE,
	"for":      FOR,
	"in":       SYM_IN, // to avoid conflict on windows
	"break":    BREAK,
	"continue": CONTINUE,
	"task":     TASK,
	"eval":     EVAL,
	"vol":      VOL, // sorry, I'm just used to 'vol'
	"volume":   VOL,
	"vel":      VOL,
	"velocity": VOL,
	"chan":     CHAN,
	"channel":  CHAN,
	"pitch":    PITCH,
	"time":     TIME,
	"dur":      DUR,
	"duration": DUR,
	"length":   LENGTH,
	"number":   NUMBER,
	"type":     TYPE,
	"defined":  DEFINED,
	"undefine": UNDEFINE,
	"delete":   SYM_DELETE,
	"flags":    FLAGS,
	"varg":     VARG,
	"attrib":   ATTRIB,
	"nargs":    NARGS,
	"typeof":   TYPEOF,
	"xy":       XY,
	"port":     PORT,
	"":         0,
}