package parse

import (
	"strings"
)

// Scan the macro definition in s, creating a new Macro structure.
func macrodefine(s string, checkkeyword int) {

	var echar int // should probably be rune?
	// var sym Symbolp

	i := strings.IndexFunc(s, isspace)
	i = strings.IndexFunc(s[i:], isnamechar)
	if i >= len(s) {
		// nothing after the name
		echar = 0
	} else {
		echar = int(s[i])
	}

	/*
			if ( checkkeyword ) {
				sym = findsym(nm,Keywords);
				if ( sym ) {
					eprint("Can't #define an existing symbol: %s\n",nm);
					return;
				}
			}
			sym = findsym(nm,Macros);
			if ( sym == 0 ) {
		        (void) syminstall(nm,Macros,MACRO);
			}
			else if ( sym->stype == UNDEF )
				sym->stype = MACRO;
			else if ( sym->stype != MACRO ) {
			}
	*/

	name := s[0:i]
	params := []string{}
	value := ""

	if echar == '(' {
		/* Gather parameter names */
		for {
			echar, paramname := scanparam(s, &i)
			params = append(params, paramname)
			if echar != ',' {
				break
			}
		}

		if echar != ')' {
			eprint("Improper #define format")
			return
		}
		scanspace(s, &i)
		value = s[i:0]
	}
	Macros[name] = NewMacro(name, value, params)
}

func scanspace(s string, pi *int) {
	for isspace(rune(s[*pi])) {
		*pi++
	}
}

func scanparam(s string, pi *int) (echar int, paramname string) {
	scanspace(s, pi)
	ibegin := *pi
	if isname1char(rune(s[*pi])) {
		for isnamechar(rune(s[*pi])) {
			(*pi)++
		}
	}
	if *pi >= len(s) {
		echar = 0
	} else {
		echar = int(s[*pi])
	}
	return echar, s[ibegin:*pi]
}

func scanarg(s string, pi *int) (echar int, paramname string) {
	return 0, s
	/* NEED TO RESURRECT
	scanspace(s,pi)
	ibegin := *pi
	if isname1char(rune(s[*pi])) {
		// scan
		for i,ch := range s[ibegin:] {

			if ! isnamechar(rune(ch)) {
				iend = i
				(*pi)++
				continue
			}
			break
		}
		name := s[ibegin:*pi]
	}
	if *pi >= len(s) {
		echar = 0;
	} else {
		echar = int(s[*pi])
	}
	return echar, s[ibegin:*pi]
	*/
}

/* Check to see if name is a macro, and if so, substitute its value (possibly*/
/* gathering the arguments and substituting them in the macro definition). */
/* The macro value is stuffed back onto the input stream. */
func (x *PkLex) macroeval(yytext string) bool {

	macro, ok := Macros[yytext]
	if !ok {
		return false
	}

	x.macrosused++
	if x.macrosused > 10 {
		eprint("Macros too deeply nested (recursive?)")
		return false
	}

	nparams := len(macro.params)
	if nparams == 0 {
		x.stuff(macro.template)
		return true
	}

	eprint("REST OF MACROEVAL has been aborted")
	return false

	/*
			// The macro has parameters, scan input till the paren
			for {
				c := x.next()
				if c == eof {
					eprint(fmt.Sprintf("Macro %s needs parameters!",macro.name))
					return false
				}
				if c == '(' {
					break
				}
			}

			buff := ""

			echar := 0
			args := []string{}
			template := macro.template
			i := 0
			for {
				var paramname string
				echar,paramname = scanparam(template[i:],"),")
				if echar == 0 || arg == "" {
					errstr = "Non-terminated call to macro"
					goto err
				}
				args = append(args,arg)
				if echar == ')' {
					break
				}
			}
			if len(args) > nparams {
				errstr = "Too many arguments in call to macro";
				goto err;
			}
			if len(args) < nparams {
				errstr = "Too few arguments in call to macro";
				goto err;
			}

			// now stuff the macro replacement value, and substitute any
			// parameters we find.
			final := ""
			template := macro.template
			for i,ch := range template {
				if ! isname1char(ch) {
					final += string(ch)
					continue;
				}
				// we've seen the start of a name; grab the rest
				namestart := i
				nameend := -1
				for i2,ch2 := range template[i:] {
					if ! isnamechar(cch) {
						nameend = i2
						break
					}
				}
				var name string
				if nameend < 0 {
					name = template[namestart:]
				} else {
					name = template[namestart:nameend]
				}
				// if it's a parameter name, substitute its value
				for n:=0; n < nparams; n++ {
					if ( strcmp(m.params[n],name) == 0 ) {
						// if it is a parameter, substitute the value
						x.stuff(args[n])
						break
					}
				}
			}
			x.stuff(buff);

			return;

		    err:
			execerror("%s %s",errstr,m.macroame)
			return;		 // should be NOTREACHED
	*/
}

func scanstringtill(s string, lookfor string) (scanned string, echar rune) {

	eprint("scanstringtill needs implementation")
	return "", 0

	/*
		for {
			echar = x.next()
			if echar == eof {
				return scanned,echar
			}
			if strings.IndexRune(lookfor, echar) >= 0 {
				return scanned, echar
			}
			scanned += string(echar)

			s := ""
			switch ( echar ) {
			case '(': s,echar = x.scaninputtill(")"); break;
			case '{': s,echar = x.scaninputtill("}"); break;
			case '[': s,echar = x.scaninputtill("]"); break;
			case '"': s,echar = x.scaninputtill("\""); break;
			case '\'': s,echar = x.scaninputtill("'"); break;
			}
			if s != "" {
				scanned += s
			}
		}
		// NOTREACHED
	*/
}
