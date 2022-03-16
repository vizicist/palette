package kit

import (
	"fmt"
	"strings"
)

//// int
//// startparams(void)
//// {
//// 	if ( Inparams ) {
//// 		yyerror("Nested parameter lists!?!");
//// 		return 1;
//// 	}
//// 	Inparams=1;
//// 	return 0;
//// }
////
//// void
//// endparams(Instnodep codeptr,int nparams,Symbolp funcsym)
//// {
//// 	Inparams = 0;
//// 	STUFFCODE(codeptr,1, numinst(nparams));
//// 	STUFFCODE(codeptr,2, syminst(funcsym));
//// 	code(funcinst(I_FILENAME));
//// 	code(strinst(Infile?uniqstr(Infile):Nullstr));
//// }
////
//// void
//// patchlocals(Instnodep codeptr)
//// {
//// 	STUFFCODE(codeptr,3,numinst(Currct->localnum-1));
//// }
////
//// Symbolp
//// forceglobal(Symbolp s)
//// {
//// 	Symbolp gs;
//// 	Symstr up = symname(s);
//// 	s->stype = TOGLOBSYM;
//// 	gs = globalinstall(up, UNDEF);
//// 	s->sd.u.sym = gs;
//// 	return gs;
//// }
////
//// Symbolp
//// local2globalinstall(Symstr up)
//// {
//// 	Symbolp s, s2;
//// 	/* Add a symbol to the current local context, */
//// 	/* which points us to the global symbol. */
//// 	s = globalinstall(up, UNDEF);
//// 	s2 = localinstall(up, TOGLOBSYM);
//// 	s2->sd.u.sym = s;
//// 	return s;
//// }
////
//// Symbolp
//// installvar(Symstr up)
//// {
//// 	Symbolp s;
////
//// 	if ( Inparams )
//// 		return localinstall(up, UNDEF);
////
//// 	if ((s=findsym(up,Currct->symbols)) != 0) {
//// 		s->flags |= S_SEEN;
//// 		if ( s->stype == TOGLOBSYM ) {
//// 			s = s->sd.u.sym;
//// 		}
//// 		return s;
//// 	}
////
//// 	if ( Currct == Topct ) {
//// 		/* we know it's not there already, so use globalinstallnew() */
//// 		return globalinstallnew(up, UNDEF);
//// 	}
////
//// 	if ( Globaldecl != 0 )
//// 		return local2globalinstall(up);
////
//// 	/* See if it's a keyword or macro */
//// 	if ( (s=findsym(up,Topct->symbols)) != NULL ) {
//// 		if ( s->stype != UNDEF && s->stype != VAR )
//// 			return s;
//// 		if ( s->stype == VAR && s->sd.type == D_CODEP )
//// 			return s;
//// 	}
//// 	/* Upper-case names are, by default, global. */
//// 	if ( *up>='A' && *up<='Z' )
//// 		return local2globalinstall(up);
////
//// 	s = localinstall(up, UNDEF);
//// 	return s;
//// }
////

func (l *Lexer) yyinput() byte {
	b, err := l.reader.ReadByte()
	if err != nil {
		fmt.Printf("yyinput returns 0 err-%s\n", err)
		return 0
	}
	return b
}

func (l *Lexer) yyunget() {
	err := l.reader.UnreadByte()
	if err != nil {
		fmt.Printf("yyunget: unexpected err=%s\n", err)
	}
}

/* legal subsequent characters of names */
func (l *Lexer) isnamechar(c byte) bool {
	if l.isalnum(c) == true || c == '_' {
		return true
	} else {
		return false
	}
}

func (l *Lexer) isalnum(c byte) bool {
	return l.isalpha(c) || l.isnum(c)
}

func (l *Lexer) isalpha(c byte) bool {
	if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') {
		return false
	}
	return true
}

func (l *Lexer) isnum(c byte) bool {
	if c < '0' && c > '9' {
		return false
	}
	return true
}

/* legal first characters of names */
func (l *Lexer) isname1char(c byte) bool {
	if l.isalpha(c) == true || c == '_' {
		return true
	} else {
		return false
	}
}

func (l *Lexer) Init(s string) {
	l.reader = strings.NewReader(s)
}

func (l *Lexer) Lex(lval *yySymType) (retval int) {

	fmt.Printf("Lexer.Lex() start\n")

	//// 	register int c;
	//// 	int retval;
	//// 	int lastc = 0;
	//// 	int isdouble, nextc;
	//// 	long bmult = 1;
	//// 	long tot;
	////
	//// 	Macrosused = 0;
	////

restart:
	/* Skip initial white space */
	c := byte(' ')
	for c == ' ' || c == '\t' || c == '\n' {
		c = l.yyinput()
	}

	if c == 0 {
		retval = 0
		goto getout
	}

	l.Yytext = string(c)
	if l.isname1char(c) {
		// Symstr up;
		// Symbolp s;

		// scan the rest of the name
		for {
			c := l.yyinput()
			if c == 0 {
				break
			}
			if !l.isnamechar(c) {
				l.yyunget()
				break
			}
		}

		up := l.Yytext

		s := findsym(up, Keywords)
		if s != nil {
			retval = s.stype
			l.yylval.sym = s
			goto getout
		}
		s = findsym(up, Macros)
		if s != nil {
			macroeval(up)
			goto restart
		}
		l.yylval.str = up
		retval = NAME
		goto getout
	}
	//// 	if ( c == '#' ) {
	//// 		if ( eatpound() == 0 ) {
	//// #ifdef MYYYDEBUG
	//// 			if ( yydebug )
	//// 				printf("yylex eatpound returns 0, EOF\n");
	//// #endif
	//// 			return(0);
	//// 		}
	//// 		goto restart;
	//// 	}
	//// 	isdouble = 0;
	//// 	if ( c == '.' ) {
	//// 		c = yyinput();
	//// 		/* allow numbers to start with . */
	//// 		if ( isdigit(c) ) {
	//// 			isdouble = 1;
	//// 			goto dblread;
	//// 		}
	//// 		if ( c == '.' ) {
	//// 			c = yyinput();
	//// 			if ( c == '.' ) {
	//// 				retval = DOTDOTDOT;
	//// 				goto getout;
	//// 			}
	//// 			execerror(".. is not valid - perhaps you meant ... ?");
	//// 		}
	//// 		yyunget(c);
	//// 		retval = '.';
	//// 		goto getout;
	//// 	}
	//// 	if ( c == '0' ) {	/* octal or hex numbers */
	//// 		tot = 0;
	//// 		if ( (c=yyinput()) == 'x' ) {			/* hex */
	//// 			while ( (c=yyinput()) != EOF ) {
	//// 				int h = hexchar(c);
	//// 				if ( h<0 )
	//// 					break;
	//// 				tot = 16 * tot + h;
	//// 			}
	//// 		}
	//// 		else if ( isdigit(c) ) {			/* octal */
	//// 			int e = 0;
	//// 			do {
	//// 				if ( c < '0' || c > '9' )
	//// 					break;
	//// 				if ( (c=='8'||c=='9') && e++ == 0 )
	//// 					eprint("Invalid octal number!\n");
	//// 				tot = 8 * tot + (c-'0');
	//// 			} while ( (c=yyinput()) != EOF );
	//// 		}
	//// 		else if ( c == '.' )
	//// 			goto foundadot;
	////
	//// 		if ( c=='b' || c=='q' )
	//// 			bmult = ((Clicks==NULL)?(DEFCLICKS):(*Clicks));
	//// 		else
	//// 			yyunget(c);
	////
	//// 		yylval.val = tot * bmult;
	//// 		retval = INTEGER;
	//// 		goto getout;
	//// 	}
	//// 	if ( isdigit(c) ) {			/* integers and floats */
	//// 	    dblread:
	//// 		for ( ; (c=yyinput()) != EOF; lastc=c ) {
	//// 			if ( isdigit(c) )
	//// 				continue;
	//// 			if ( c == '.' ) {
	//// 			    foundadot:
	//// 				/* look ahead to see if it's a float */
	//// 				nextc = yyinput();
	//// 				yyunget(nextc);
	//// 				if ( isdigit(nextc) || nextc=='e' ) {
	//// 					isdouble = 1;
	//// 					continue;
	//// 				}
	//// 				yyunget('.');
	//// 				/* a number followed by a '.' *and* then */
	//// 				/* followed by a non-digit is probably */
	//// 				/* an expression like ph%2.pitch, so we */
	//// 				/* just return the integer. */
	//// 				break;
	//// 			}
	//// 			if ( c == 'e' ) {
	//// 				isdouble = 1;
	//// 				continue;
	//// 			}
	//// 			/* An integer with a 'b' or 'q' suffix is multiplied by */
	//// 			/* the number of clicks in a quarter note. */
	//// 			if ( ! isdouble && (c=='q' || c=='b') ) {
	//// 				bmult = ((Clicks==NULL)?(DEFCLICKS):(*Clicks));
	//// 				/* and we're done */
	//// 				break;
	//// 			}
	//// 			if ( (c=='+'||c=='-') && lastc=='e' ) {
	//// 				isdouble = 1;
	//// 				continue;
	//// 			}
	//// 			yyunget(c);
	//// 			break;
	//// 		}
	//// 		*Pyytext = '\0';
	////
	//// 		if ( isdouble ) {
	//// 			yylval.dbl = (DBLTYPE) atof(Yytext);
	//// 			retval = DOUBLE;
	//// 		}
	//// 		else {
	//// 			yylval.val = atol(Yytext) * bmult;
	//// 			retval = INTEGER;
	//// 		}
	//// 		goto getout;
	//// 	}
	//// 	if ( c == '\'' ) {
	//// 		yyunget(c);
	//// 		yylval.phr = yyphrase(yyinput);
	//// 		phincruse(yylval.phr);
	//// 		retval = PHRASE;
	//// 		goto getout;
	//// 	}
	//// 	/* strings ) */
	//// 	if ( c == '"' ) {
	//// 		int si = 0;
	//// 		int ch, n, i;
	////
	//// 		while ( (ch=yyinput()) != '"' ) {
	////
	//// 		    rechar:
	//// 			if ( ch == EOF )
	//// 				execerror("missing ending-quote on string");
	//// 			if ( ch == '\n' )
	//// 				execerror("Newline inside string?!");
	//// 			/* interpret \-characters */
	//// 			if ( ch == '\\' ) {
	//// 				switch ( ch=yyinput() ) {
	//// 				case '\n':
	//// 					/* escaped newlines are ignored */
	//// 					ch = yyinput();
	//// 					goto rechar;
	//// 					/* break; */
	//// 				case '0':
	//// 					/* Handle \0ddd numbers */
	//// 					for ( n=0,i=0; i<3; i++ ) {
	//// 						ch = yyinput();
	//// 						if ( ! isdigit(ch) )
	//// 							break;
	//// 						n = n*8 + ch - '0';
	//// 					}
	//// 					yyunget(ch);
	//// 					ch = n;
	//// 					break;
	//// 				case 'x':
	//// 					/* Handle \xfff numbers */
	//// 					for ( n=0,i=0; i<3; i++ ) {
	//// 						ch = hexchar(yyinput());
	//// 						if ( ch < 0 )
	//// 							break;
	//// 						n = n*16 + ch;
	//// 					}
	//// 					yyunget(ch);
	//// 					ch = n;
	//// 					break;
	//// #ifdef OLDSTUFF
	//// 				/* use this if \xFF has only 2 chars */
	//// 				case 'x':
	//// 					{ int h1, h2;
	//// 					/* Handle \xFF numbers */
	//// 					h1 = hexchar(yyinput());
	//// 					h2 = hexchar(yyinput());
	//// 					if ( h1<0 || h2<0 ) {
	//// 						eprint("Invalid hex number!\n");
	//// 						ch = 0;
	//// 					}
	//// 					else
	//// 						ch = h1*16 + h2;
	//// 					}
	//// 					break;
	//// #endif
	//// 				case 'b': ch ='\b'; break;
	//// 				case 'f': ch ='\f'; break;
	//// 				case 'n': ch ='\n'; break;
	//// 				case 'r': ch ='\r'; break;
	//// 				case 't': ch ='\t'; break;
	//// 				case 'v': ch ='\v'; break;
	//// 				case '"': ch ='"'; break;
	//// 				case '\'': ch ='\''; break;
	//// 				case '\\': ch = '\\'; break;
	//// 				default:
	//// 					if ( Slashcheck != NULL && *Slashcheck != 0 ) {
	//// 						eprint("Unrecognized backslashed character (%c) is ignored\n",ch);
	//// 						ch = yyinput();
	//// 						goto rechar;
	//// 					} else {
	//// 						yyunget(ch);
	//// 						ch = '\\';
	//// 					}
	//// 					break;
	//// 				}
	//// 			}
	//// 			makeroom((long)(si+2),&Msg1,&Msg1size); /* +1 for final '\0' */
	//// 			Msg1[si++] = ch;
	//// 		}
	//// 		Msg1[si] = '\0';
	//// 		yylval.str = uniqstr(Msg1);
	//// 		retval = STRING;
	//// 		goto getout;
	//// 	}
	//// 	if ( c == '$' ) {
	//// 		c = yyinput();
	//// 		if ( c == '$' ) {
	//// 			retval = DOLLAR2;
	//// 			goto getout;
	//// 		}
	//// 		if ( isdigit(c) || c == '-' ) {
	//// 			long n;
	//// 			int sgn;
	//// 			if ( c == '-' ) {
	//// 				n = 0;
	//// 				sgn = -1;
	//// 			}
	//// 			else {
	//// 				n = c - '0';
	//// 				sgn = 1;
	//// 			}
	//// 			while ( (c=yyinput()) != EOF ) {
	//// 				if ( ! isdigit(c) ) {
	//// 					yyunget(c);
	//// 					break;
	//// 				}
	//// 				n = n*10 + c - '0';
	//// 			}
	//// 			yylval.val = n*sgn + *Kobjectoffset;
	//// 			if ( yylval.val >= Nextobjid )
	//// 				Nextobjid = yylval.val + 1;
	//// 			retval = OBJECT;
	//// 			goto getout;
	//// 		}
	//// 		yyunget(c);
	//// 		retval = '$';
	//// 		goto getout;
	//// 	}
	//// 	switch(c) {
	//// 	case '\n': retval = '\n'; break;
	//// 	case '?':  retval = follow('?', QMARK2, '?'); break;
	//// 	case '=':  retval = follow('=', EQ, '='); break;
	//// 	case '+':  retval = follo2('=', PLUSEQ, '+', INC, '+'); break;
	//// 	case '-':  retval = follo2('=', MINUSEQ, '-', DEC, '-'); break;
	//// 	case '*':  retval = follow('=', MULEQ, '*'); break;
	//// 	case '>':  retval = follo3('=', GE, '>', '=',RSHIFT,RSHIFTEQ,GT);break;
	//// 	case '<':  retval = follo3('=', LE, '<', '=',LSHIFT,LSHIFTEQ,LT);break;
	//// 	case '!':  retval = follow('=', NE, BANG); break;
	//// 	case '&':  retval = follo2('=', AMPEQ, '&', AND, '&'); break;
	//// 	case '|':  retval = follo2('=', OREQ, '|', OR, '|'); break;
	//// 	case '/':  retval = follow('=', DIVEQ, '/'); break;
	//// 	case '^':  retval = follow('=', XOREQ, '^'); break;
	//// 	case '~':  retval = follow('~', REGEXEQ, '~'); break;
	//// 	default:   retval = c; break;
	//// 	}
	////     getout:
	//// 	*Pyytext = '\0';
	//// #ifdef MYYYDEBUG
	//// 	if ( yydebug )
	//// 		printf("yylex returns %d, Yytext=(%s)\n",retval,Yytext);
	//// #endif
	//// 	return(retval);

getout:

	fmt.Printf("Lexer.Lex returns %d\n", retval)
	return retval
}
