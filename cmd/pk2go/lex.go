package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The parser expects the lexer to return 0 on EOF.  Give it a name
// for clarity.
const eof = 0

const LexDebug = false

var Clicks = 96

type Symstr string

// The parser uses the type <prefix>Lex as a lexer. It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type pkLex struct {
	line       []byte
	peek       []rune
	yytext     string
	macrosused int
	outf       *os.File
}

func (x *pkLex) stuff(s string) {
	for _, r := range s {
		x.peek = append(x.peek, r)
	}
}

// Push back one rune
func (x *pkLex) unget(r rune) {
	x.peek = append(x.peek, r)
}

// Return the next rune for the lexer.
func (x *pkLex) next() rune {
	if len(x.peek) != 0 {
		r := rune(x.peek[0])
		x.peek = x.peek[1:]
		return r
	}
	if len(x.line) == 0 {
		return eof
	}
	c, size := utf8.DecodeRune(x.line)
	x.line = x.line[size:]
	if c == utf8.RuneError && size == 1 {
		log.Print("invalid utf8")
		return x.next()
	}
	return c
}

func (x *pkLex) scaninputtill(lookfor string) (scanned string, echar rune) {
	for {
		c := x.next()
		if c == eof {
			return scanned, eof
		}
		scanned += string(c)
		if strings.ContainsRune(lookfor, c) {
			return scanned, c
		}

		// scan nested things like parens, brackets, etc
		s := ""
		ec := rune(999) // just so it's not eof
		switch c {
		case '(':
			s, ec = x.scaninputtill(")")
		case '{':
			s, ec = x.scaninputtill("}")
		case '[':
			s, ec = x.scaninputtill("]")
		case '"':
			s, ec = x.scaninputtill("\"")
		case '\'':
			s, ec = x.scaninputtill("'")
		}
		if ec == eof {
			return scanned, eof
		}
		if s != "" {
			scanned += s
		}
	}
	// NOTREACHED
}

// The parser calls this method on a parse error.
func (x *pkLex) Error(s string) {
	log.Printf("parse error: %s", s)
}

func isname1char(c rune) bool {
	return isalpha(c) || c == '_'
}

func isnamechar(c rune) bool {
	return (isalnum(c) || c == '_')
}

func isalnum(c rune) bool {
	return ((c) >= 'a' && (c) <= 'z') || ((c) >= 'A' && (c) <= 'Z') || (c) == '_' || ((c) >= '0' && (c) <= '9')
}

func isalpha(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func isspace(c rune) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isdigit(c rune) bool {
	return c >= '0' && c <= '9'
}

func hexchar(r rune) int {
	c := int(r)
	if c >= '0' && c <= '9' {
		return c - '0'
	}
	if c >= 'A' && c <= 'F' {
		return c - 'A' + 10
	}
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 10
	}
	return -1
}

// The parser calls this method to get each new token. This
// implementation returns operators and NUM.
func (x *pkLex) Lex(yylval *pkSymType) int {
	var c rune
	var retval int
	// var nextc int

	// lastc := 0
	bmult := 1

	x.macrosused = 0

restart:
	// Skip initial white space
	for {
		c = x.next()
		// if !(c == ' ' || c == '\t' || c == '\n' || c == '\r') {
		if !(c == ' ' || c == '\t' || c == '\n' || c == '\r') {
			break
		}
	}

	if c == eof {
		if LexDebug {
			eprint("Lex returns eof, successfully parsed everything?")
		}
		return 0
	}

	// Pyytext = Yytext
	// *Pyytext++ = c;
	x.yytext = string(c)

	if isname1char(c) {
		for {
			c = x.next()
			if c == eof || !isnamechar(c) {
				break
			}
			// XXX - There's a better way to do this
			x.yytext += string(c)
		}
		x.unget(c)
		// yyunget(c)
		// *Pyytext = '\0';

		s, ok := Keywords[x.yytext]
		if ok {
			retval = s
			yylval.node = makeNodeOfType(s)
			yylval.node.str = x.yytext
			goto getout
		}
		if x.macroeval(x.yytext) {
			goto restart
		}
		yylval.node = makeNodeOfName(x.yytext)
		retval = NAME
		goto getout
	}
	if c == '#' {
		x.eatpound()
		goto restart
	}
	if c == '.' {
		c = x.next()
		// allow numbers to start with .
		if isdigit(c) {
			retval = x.readnumber(yylval)
			goto getout
		}
		if c == '.' {
			c = x.next()
			if c == '.' {
				retval = DOTDOTDOT
				goto getout
			}
			eprint(".. is not valid - perhaps you meant ... ?")
		}
		x.unget(c)
		retval = '.'
		goto getout
	}
	if c == '0' { // octal or hex numbers
		tot := 0
		r := x.next()
		if r == 'x' { // hex
			for {
				r = x.next()
				if r == eof {
					break
				}
				h := hexchar(r)
				if h < 0 {
					break
				}
				tot = 16*tot + h
			}
		} else if isdigit(r) { // octal
			e := 0
			digit := int(r) - '0'
			for {
				if digit < 0 || digit > 9 {
					break
				}
				if (digit == 8 || digit == 9) && e == 0 {
					eprint("Invalid octal number!")
				}
				e++
				tot = 8*tot + digit
				r = x.next()
				if r == eof {
					break
				}
			}
		} else if r == '.' {
			x.unget(r)
			retval = x.readnumber(yylval)
			goto getout
		}

		if r == 'b' || r == 'q' {
			bmult = Clicks
		} else {
			x.unget(r)
		}

		yylval.node = makeNodeInteger(tot * bmult)
		retval = INTEGER
		goto getout
	}
	if isdigit(c) { // integers and floats
		x.unget(c)
		x.yytext = ""
		retval = x.readnumber(yylval)
		goto getout
	}
	if c == '\'' {
		scanned, echar := x.scaninputtill("'")
		if echar != '\'' {
			eprint("Unterminated phrase in input?")
			goto getout
		}
		x.yytext = string(c) + scanned
		yylval.node = makeNodePhrase(x.yytext)
		retval = PHRASE
		goto getout
	}
	// strings
	if c == '"' {
		for {
			ch := x.next()
			if ch == '"' {
				x.yytext += string(ch)
				break
			}

			if ch == eof {
				eprint("missing ending-quote on string")
			}
			if ch == '\n' {
				eprint("newline inside string")
			}
			// interpret \-characters
			if ch == '\\' {
				ch := x.next()
				switch ch {
				case '\n':
					// escaped newlines are ignored
					// __ = x.next()
					ch = 0
					continue
				case '0':
					// Handle \0ddd numbers
					n := 0
					for i := 0; i < 3; i++ {
						ch = x.next()
						if !isdigit(ch) {
							break
						}
						n = n*8 + (int(ch) - '0')
					}
					x.unget(ch)
					// ch = rune(n)
				case 'b':
					ch = '\b'
				case 'f':
					ch = '\f'
				case 'n':
					ch = '\n'
				case 'r':
					ch = '\r'
				case 't':
					ch = '\t'
				case 'v':
					ch = '\v'
				case '"':
					ch = '"'
				case '\'':
					ch = '\''
				case '\\':
					ch = '\\'
				default:
					eprint(fmt.Sprintf("Unrecognized backslashed character (%c) is ignored\n", ch))
					continue
				}
			}
			if ch != 0 {
				x.yytext += string(ch)
			}
		}
		yylval.node = makeNodeString(x.yytext)
		retval = STRING
		goto getout
	}
	if c == '$' {
		c = x.next()
		if c == '$' {
			retval = DOLLAR2
			goto getout
		}
		if isdigit(c) || c == '-' {
			var n int
			var sgn int
			if c == '-' {
				n = 0
				sgn = -1
			} else {
				n = int(c) - '0'
				sgn = 1
			}
			for {
				c := x.next()
				if c == eof {
					break
				}
				if !isdigit(c) {
					x.unget(c)
					break
				}
				n = n*10 + int(c) - '0'
			}
			yylval.node = makeNodeObject(n * sgn) // + *Kobjectoffset ?
			retval = OBJECT
			goto getout
		}
		x.unget(c)
		retval = '$'
		goto getout
	}
	switch c {
	case '\n':
		retval = '\n'
		// goto restart
	case '\r':
		retval = '\r'
	case '?':
		retval = x.follow('?', QMARK2, '?')
	case '=':
		retval = x.follow('=', EQ, '=')
	case '+':
		retval = x.follo2('=', PLUSEQ, '+', INC, '+')
	case '-':
		retval = x.follo2('=', MINUSEQ, '-', DEC, '-')
	case '*':
		retval = x.follow('=', MULEQ, '*')
	case '>':
		retval = x.follo3('=', GE, '>', '=', RSHIFT, RSHIFTEQ, GT)
	case '<':
		retval = x.follo3('=', LE, '<', '=', LSHIFT, LSHIFTEQ, LT)
	case '!':
		retval = x.follow('=', NE, BANG)
	case '&':
		retval = x.follo2('=', AMPEQ, '&', AND, '&')
	case '|':
		retval = x.follo2('=', OREQ, '|', OR, '|')
	case '/':
		retval = x.follow('=', DIVEQ, '/')
	case '^':
		retval = x.follow('=', XOREQ, '^')
	case '~':
		retval = x.follow('~', REGEXEQ, '~')
	default:
		yylval.node = makeNodeOfType(int(c))
		retval = int(c)
	}
getout:
	if LexDebug {
		eprint(fmt.Sprintf("Lex returns retval=%d yytext=%s yylval=%s", retval, x.yytext, yylval.node))
	}
	return (retval)
}

func (x *pkLex) readnumber(yylval *pkSymType) int {
	f, isdouble := x.dblread()
	if isdouble {
		yylval.node = makeNodeDouble(f)
		return DOUBLE
	} else {
		yylval.node = makeNodeNumber(int(f))
		return NUMBER
	}
}

func eprint(s string) {
	if strings.Contains(s, "TJT!") {
		s = ": " + s
	}
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	os.Stderr.Write([]byte(s))
}

func (x *pkLex) eatpound() {
	// comments extend from a '#' (at the beginning of a word)
	// to the end of the line.
	line := "#"
	for {
		c := x.next()
		if c == eof {
			break
		}
		if c == '\n' || c == '\r' {
			break
		}
		line += string(c)
	}

	// Could be a macro or #include, though
	if strings.HasPrefix(line, "#define") {
		eprint("Need to implement #define")
		// macrodefine(Yytext+7,1);
	} else if strings.HasPrefix(line, "#include") {
		eprint("Need to implement #include")
		// pinclude(Yytext+8);
	}
}

func (x *pkLex) dblread() (val float64, isdouble bool) {

	var lastc rune
	var c rune
	bmult := 1
	isdouble = false
	for ; ; lastc = c {
		c = x.next()
		if c == eof {
			break
		}
		if isdigit(c) {
			x.yytext += string(c)
			continue
		}
		if c == '.' {
			// foundadot:
			// look ahead to see if it's a float
			nextc := x.next()
			x.unget(nextc)
			if isdigit(nextc) || nextc == 'e' {
				isdouble = true
				x.yytext += string(c)
				continue
			}
			x.unget('.')
			// a number followed by a '.' *and* then
			// followed by a non-digit is probably
			// an expression like ph%2.pitch, so we
			// just return the integer.
			break
		}
		if c == 'e' {
			x.yytext += string(c)
			isdouble = true
			continue
		}
		// An integer with a 'b' or 'q' suffix is multiplied by
		// the number of clicks in a quarter note.
		if !isdouble && (c == 'q' || c == 'b') {
			bmult = Clicks
			// and we're done
			break
		}
		if (c == '+' || c == '-') && lastc == 'e' {
			x.yytext += string(c)
			isdouble = true
			continue
		}
		x.unget(c)
		break
	}
	// *Pyytext = '\0';

	if isdouble {
		f, err := strconv.ParseFloat(x.yytext, 64)
		if err != nil {
			eprint(fmt.Sprintf("Unrecognized float? (%s) assuming 0.0", x.yytext))
			f = 0.0
		}
		val = f
	} else {
		i, err := strconv.Atoi(x.yytext)
		if err != nil {
			eprint(fmt.Sprintf("Unrecognized integer? (%s) assuming 0", x.yytext))
			i = 0
		} else {
			i = i * bmult
		}
		val = float64(i)
	}
	return val, isdouble
}

/* follow() - look ahead for >=, etc. */
func (x *pkLex) follow(expect int, ifyes int, ifno int) int {
	ch := x.next()
	if int(ch) == expect {
		return ifyes
	}
	x.unget(ch)
	return ifno
}

func (x *pkLex) follo3(expect1 int, ifyes1 int, expect2 int, expect3 int, ifyes2 int, ifyes3 int, ifno int) int {
	ch := x.next()
	if int(ch) == expect1 {
		return ifyes1
	}
	if int(ch) == expect2 {
		ch3 := x.next()
		if int(ch3) == expect3 {
			return ifyes3
		}
		x.unget(ch3)
		return ifyes2
	}
	x.unget(ch)
	return ifno
}

func (x *pkLex) follo2(expect1 int, ifyes1 int, expect2 int, ifyes2 int, ifno int) int {
	ch := x.next()
	if int(ch) == expect1 {
		return ifyes1
	}
	if int(ch) == expect2 {
		return ifyes2
	}
	x.unget(ch)
	return ifno
}
