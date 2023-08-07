%{

package parse

%}

%union {
	node *Node
}

%token	<node>	VAR UNDEF MACRO TOGLOBSYM QMARK2 DOLLAR2 WHILE DOTDOTDOT
%token	<node>	IF ELSE FOR SYM_IN BEINGREAD EVAL BREAK CONTINUE TASK
%token	<node>	SYM_DELETE UNDEFINE RETURN FUNC DEFINED READONLY ONCHANGE GLOBALDEC
%token	<node>	CLASS METHOD KW_NEW NARGS TYPEOF XY
%token	<node>	DUR VOL TIME CHAN PITCH LENGTH NUMBER TYPE ATTRIB FLAGS VARG PORT
%token	<node>	PHRASE
%token	<node>	STRING NAME
%token	<node>	INTEGER OBJECT
%token	<node>	DOUBLE
%token	<node>	SEQUENCE
%token	<node>	SELECTION
%token	<node>	FUNCCALL
%token	<node>	PRMLIST
%token	<node>	PARAM
%token	<node>	ARRITEMEQ
%token	<node>	EQUALS
%token	<node>	PLUSEQ MINUSEQ MULEQ DIVEQ AMPEQ INC DEC PREINC PREDEC
%token	<node>	POSTINC POSTDEC OREQ XOREQ RSHIFTEQ LSHIFTEQ
%type	<node>	list expr stmt stmts nosemi optstmt stmtnv
%type	<node>	optrelx
%type	<node>	equals
%type	<node>	method arglist narglist methdef methdefs
%type	<node>	uniqvar var dottype globvar
%type	<node>	args prmlist prms arritem arrlist globlist
%type	<node>	methname
%type	<node>	'$'
%right	'=' PLUSEQ MINUSEQ MULEQ DIVEQ AMPEQ OREQ XOREQ RSHIFTEQ LSHIFTEQ
%right	'?'
%right	':'
%left	OR
%left	AND
%left	'|'
%left	'^'
%left	'&'
%nonassoc	GT GE LT LE EQ NE REGEXEQ
%left	LSHIFT RSHIFT
%left	'+' '-'
%left	'*' '/'
%left	UNARYMINUS BANG '~'
%left	'%' '.'
%left	INC DEC
%left   SEQUENCE
%%
list	: 			
		stmts		{
			HandleProgram(Pklex,$1)
		}
	;
stmts	: /* nothing */		{
							$$ = makeNodeNil()
							}
	| stmt stmts {
		nn := &Node{ stype: SEQUENCE,
						children: []*Node{$1},
						}
		if $2 == nil {
			// do nothing
		} else if $2.stype != SEQUENCE {
			nn.children = append(nn.children,$2)
		} else {
			for i:=0; i<len($2.children); i++  {
				child := $2.children[i]
				nn.children = append(nn.children,child)
			}
		}
		$$ = nn
	}
	;
optstmt : /* nothing */		{ }
	| nosemi
	;
stmt	: ';' {
			$$ = &Node{
				stype: ';',
				children: []*Node{},
			}
			}
	/// | '\n' {
	/// 		$$ = &Node{
	/// 			stype: '\n',
	/// 			children: []*Node{},
	/// 		}
	/// }
	| nosemi
	| nosemi ';'
	; 
nosemi	: stmtnv
	| expr
stmtnv	: RETURN		{
						$$ = &Node{
							stype: RETURN,
							children: []*Node{},
						}
						}
	| RETURN '(' ')'	{
						$$ = &Node{
							stype: RETURN,
							children: []*Node{},
						}
						}
	| RETURN '(' expr ')'	{
						$$ = &Node{
							stype: RETURN,
							children: []*Node{$3},
						}
						}
	| BREAK			{
					$$ = &Node{
						stype: BREAK,
						children: []*Node{},
					}
					}
	| CONTINUE		{
					$$ = &Node{
						stype: CONTINUE,
						children: []*Node{},
					}
					}
	| GLOBALDEC globlist {
					$$ = &Node{
						stype: GLOBALDEC,
						children: []*Node{$2},
					}
				}
	;
expr	: '{' stmts '}'		{ $$ = $2 }
	| SYM_DELETE expr
		{
					$$ = &Node{
						stype: SYM_DELETE,
						children: []*Node{$2},
					}
		}
	| READONLY var {
					$$ = &Node{
						stype: READONLY,
						children: []*Node{$2},
					}
		}
	| ONCHANGE '(' var ',' expr ')' {
					$$ = &Node{
						stype: ONCHANGE,
						children: []*Node{$3,$5},
					}
		}
	| WHILE '(' expr ')' stmt {
					$$ = &Node{
						stype: WHILE,
						children: []*Node{$3,$5},
					}
		}

	| FOR '(' optstmt ';' optrelx ';' optstmt ')' stmt {
			$$ = &Node{
				stype: FOR,
				children: []*Node{$3,$5,$7,$9},
			}
		}

	| FOR '(' var SYM_IN expr ')' stmt {
			$$ = &Node{
				stype: FOR,
				children: []*Node{$3,$5,$7},
			}
		}

	| IF '(' expr ')' stmt {		/* else-less if */
			$$ = &Node{
				stype: IF,
				children: []*Node{$3,$5},
			}
		}
	| IF '(' expr ')' stmt ELSE stmt { /* if with else */
			$$ = &Node{
				stype: IF,
				children: []*Node{$3,$5,$7},
			}
		}
	| KW_NEW var '(' narglist ')'	{
			$$ = &Node{
				stype: KW_NEW,
				children: []*Node{$2,$4},
			}
			}
	| KW_NEW '(' expr ')' var '(' narglist ')'	
			{
			$$ = &Node{
				stype: KW_NEW,
				children: []*Node{$3,$5,$7},
			}
			}
	| UNDEFINE var {
			$$ = &Node{
				stype: UNDEFINE,
				children: []*Node{$2},
			}
			}
	| UNDEFINE '(' var ')' {
			$$ = &Node{
				stype: UNDEFINE,
				children: []*Node{$3},
			}
			}
	| '[' arrlist ']' {
			$$ = &Node{
				stype: '[',
				children: []*Node{$2},
			}
		}
	| INTEGER
	| DOUBLE
	| STRING
	| PHRASE
	| var
	| QMARK2
	| DOUBLE dottype	{
			$$ = &Node{
				stype: DOUBLE,
				children: []*Node{$2},
			}
			}
	| expr '[' expr ']'	{
			$$ = &Node{
				stype: '[',
				children: []*Node{$1,$3},
			}
			}
	| expr '{' expr '}' {
			$$ = &Node{
				stype: SELECTION,
				children: []*Node{$1,$3},
			}
			}
	| expr '?' expr ':' expr {
			$$ = &Node{
				stype: '?',
				children: []*Node{$1,$3,$5},
			}
			}
	| '(' expr ')'		{
			$$ = &Node{
				stype: '(',
				children: []*Node{$2},
			}
			}
	| DEFINED '(' var ')'	{
			$$ = &Node{
				stype: DEFINED,
				children: []*Node{$3},
			}
			}
	| DEFINED '(' '$' ')'	{
			$$ = &Node{
				stype: DEFINED,
				children: []*Node{$3},
			}
			}
	| DEFINED '(' DOLLAR2 ')'  {
			$$ = &Node{
				stype: DEFINED,
				children: []*Node{$3},
			}
			}
	| DEFINED '(' expr '.' method ')'	{
			$$ = &Node{
				stype: DEFINED,
				children: []*Node{$3,$5},
			}
			}
	| DEFINED var		{
			$$ = &Node{
				stype: DEFINED,
				children: []*Node{$2},
			}
			}
	| expr '%' expr	{
			$$ = &Node{
				stype: '%',
				children: []*Node{$1,$3},
			}
			}
	| expr '+' expr	{
			$$ = &Node{
				stype: '+',
				children: []*Node{$1,$3},
			}
			}
	| expr '-' expr	{
			$$ = &Node{
				stype: '-',
				children: []*Node{$1,$3},
			}
			}
	| expr '*' expr	{
			$$ = &Node{
				stype: '*',
				children: []*Node{$1,$3},
			}
			}
	| expr '/' expr	{
			$$ = &Node{
				stype: '/',
				children: []*Node{$1,$3},
			}
			}
	| expr '|' expr	{
			$$ = &Node{
				stype: '|',
				children: []*Node{$1,$3},
			}
			}
	| expr '&' expr	{
			$$ = &Node{
				stype: '&',
				children: []*Node{$1,$3},
			}
			}
	| expr '^' expr	{
			$$ = &Node{
				stype: '^',
				children: []*Node{$1,$3},
			}
			}
	| expr LSHIFT expr {
			$$ = &Node{
				stype: LSHIFT,
				children: []*Node{$1,$3},
			}
			}
	| expr RSHIFT expr {
			$$ = &Node{
				stype: RSHIFT,
				children: []*Node{$1,$3},
			}
			}
	| '-' expr   %prec UNARYMINUS   {
			$$ = &Node{
				stype: UNARYMINUS,
				children: []*Node{$2},
			}
			}
	| '~' expr   	{
			$$ = &Node{
				stype: '~',
				children: []*Node{$2},
			}
			}
	| expr GT expr		{
			$$ = &Node{
				stype: GT,
				children: []*Node{$1,$3},
			}
			}
	| expr LT expr		{
			$$ = &Node{
				stype: LT,
				children: []*Node{$1,$3},
			}
			}
	| expr GE expr		{
			$$ = &Node{
				stype: GE,
				children: []*Node{$1,$3},
			}
			}
	| expr LE expr		{
			$$ = &Node{
				stype: LE,
				children: []*Node{$1,$3},
			}
			}
	| expr EQ expr		{
			$$ = &Node{
				stype: EQ,
				children: []*Node{$1,$3},
			}
			}
	| expr REGEXEQ expr	{
			$$ = &Node{
				stype: REGEXEQ,
				children: []*Node{$1,$3},
			}
			}
	| expr NE expr		{
			$$ = &Node{
				stype: NE,
				children: []*Node{$1,$3},
			}
			}
	| BANG expr		{
			$$ = &Node{
				stype: BANG,
				children: []*Node{$2},
			}
			}
	| expr SYM_IN expr		{
			$$ = &Node{
				stype: SYM_IN,
				children: []*Node{$1,$3},
			}
			}
	| expr AND expr {
					$$ = &Node{
						stype: AND,
						children: []*Node{$1,$3},
					};
			}
	| expr OR expr {
					$$ = &Node{
						stype: OR,
						children: []*Node{$1,$3},
					};
			}
	| expr equals expr	{
			$$ = &Node{
				stype: EQUALS,
				children: []*Node{$1,$2,$3},
			}
			}
	| expr INC		{
			$$ = &Node{
				stype: INC,
				children: []*Node{$1,$2},
			}
			}
	| expr DEC		{
			$$ = &Node{
				stype: DEC,
				children: []*Node{$1,$2},
			}
			}
	| INC expr		{
			$$ = &Node{
				stype: PREINC,
				children: []*Node{$1,$2},
			}
			}
	| DEC expr 		{
			$$ = &Node{
				stype: PREDEC,
				children: []*Node{$1,$2},
			}
			}
	| EVAL expr 		{
			$$ = &Node{
				stype: EVAL,
				children: []*Node{$2},
			}
			}
	| '$' {
			$$ = &Node{
				stype: '$',
				children: []*Node{},
			}
			}
	| DOLLAR2
	| OBJECT
	| expr '.' dottype	{
			$$ = &Node{
				stype: '.',
				children: []*Node{$1,$3},
			}
			}
	| expr '.' method	{
			$$ = &Node{
				stype: '.',
				children: []*Node{$1,$3},
			}
			}
	| expr '.' method '(' arglist ')' {
			$$ = &Node{
				stype: FUNCCALL,
				children: []*Node{$1,$3,$5},
			}
			}
	| TASK var '(' arglist ')' {
			$$ = &Node{
				stype: TASK,
				children: []*Node{$2,$4},
			}
			}
	| TASK expr '(' arglist ')' {
			$$ = &Node{
				stype: TASK,
				children: []*Node{$2,$4},
			}
			}
	| TASK expr '.' method '(' arglist ')' {
			$$ = &Node{
				stype: TASK,
				children: []*Node{$2,$4,$6},
			}
			}
	| var '(' arglist ')'	{
				$$ = &Node{
					stype: FUNCCALL,
					children: []*Node{$1,$3},
				}
			}
	| expr '(' arglist ')'	{
				$$ = &Node{
					stype: FUNCCALL,
					children: []*Node{$1,$3},
				}
			}
	| NARGS '(' ')'
	| TYPEOF '(' expr ')'	{
				$$ = &Node{
					stype: TYPEOF,
					children: []*Node{$3},
				}
			}
	| XY '(' expr ',' expr ')'	{
				$$ = &Node{
					stype: XY,
					children: []*Node{$3,$5},
				}
			}
	| XY '(' expr ',' expr ',' expr ',' expr ')'	{
				$$ = &Node{
					stype: XY,
					children: []*Node{$3,$5,$7,$9},
				}
			}
	| GLOBALDEC '(' globvar ')' {
				$$ = &Node{
					stype: GLOBALDEC,
					children: []*Node{$3},
				}
		}
	| FUNC '(' expr ')'	{
				$$ = &Node{
					stype: FUNC,
					children: []*Node{$3},
				}
			}
	| FUNC var '(' prmlist ')' '{' stmts '}'	{
				$$ = &Node{
					stype: FUNC,
					children: []*Node{$2,$4,$7},
				}
			}
	| FUNC var '{' stmts '}'	{
				$$ = &Node{
					stype: FUNC,
					children: []*Node{$2,$4},
				}
			}
	| FUNC uniqvar '(' prmlist ')' '{' stmts '}'	{
				$$ = &Node{
					stype: FUNC,
					children: []*Node{$2,$4,$7},
				}
			}
	| FUNC uniqvar '{' stmts '}'	{
				$$ = &Node{
					stype: FUNC,
					children: []*Node{$2,$4},
				}
			}
	| CLASS var '{' methdefs '}'	{
				$$ = &Node{
					stype: CLASS,
					children: []*Node{$2,$4},
				}
			}
	;
methdefs: {
				$$ = &Node{
					stype: METHOD,
					children: []*Node{},
				}
		}
	| methdef methdefs
	;
methdef	: METHOD methname '(' prmlist ')' '{' stmts '}'	{
				$$ = &Node{
					stype: METHOD,
					children: []*Node{$2,$4,$7},
				}
			}
	| METHOD methname '{' stmts '}'	{
				$$ = &Node{
					stype: METHOD,
					children: []*Node{$2,$4},
				}
			}
	;
dottype	: VOL
	| DUR
	| CHAN
	| PORT
	| TIME
	| PITCH
	| LENGTH
	| TYPE
	| ATTRIB
	| FLAGS
	| NUMBER
	;
equals	: PLUSEQ { $$ = &Node{ stype: PLUSEQ } }
	| MINUSEQ { $$ = &Node{ stype: MINUSEQ }}
	| MULEQ { $$ = &Node{ stype: MULEQ }}
	| DIVEQ { $$ = &Node{ stype: DIVEQ }}
	| OREQ { $$ = &Node{ stype: OREQ } }
	| AMPEQ { $$ = &Node{ stype: AMPEQ }}
	| XOREQ { $$ = &Node{ stype: XOREQ }}
	| RSHIFTEQ { $$ = &Node{ stype: RSHIFTEQ }}
	| LSHIFTEQ { $$ = &Node{ stype: LSHIFTEQ }}
	| '='		{
					$$ = &Node{
						stype: '=',
						children: []*Node{},
					};
					}
	;
optrelx	: /* nothing */ { $$ = nil }
	| expr
	;
uniqvar : '?' {
					$$ = &Node{
						stype: '?',
						children: []*Node{},
					};
		}
	;
prmlist	: /* nothing */		{
				$$ = nil
				}
	| prms 			{
					$$ = &Node{
						stype: PRMLIST,
						children: []*Node{$1,nil},
					}
				}
	| DOTDOTDOT		{
					$$ = &Node{
						stype: PRMLIST,
						children: []*Node{$1,nil},
					}
				}
	| prms ',' DOTDOTDOT	{
					$$ = &Node{
						stype: PRMLIST,
						children: []*Node{$1,$3},
					}
				}
	;
prms	: var {
				$$ = &Node{
						stype: PARAM,
						children: []*Node{$1},
					}
			}
	| prms ',' var		{
				$$ = &Node{
						stype: PARAM,
						children: []*Node{$1,$3},
					}
				}
	;
arglist	: /* nothing */		{ $$ = nil }
	| args
	| DOTDOTDOT		{
					$$ = &Node{
						stype: DOTDOTDOT,
						children: []*Node{},
					}
				}
	| args ',' DOTDOTDOT	{
					$$ = &Node{
						stype: DOTDOTDOT,
						children: []*Node{$1},
					}
				}
	| VARG '(' expr ')'	{
					$$ = &Node{
						stype: VARG,
						children: []*Node{$3},
					}
				}
	| args ',' VARG	'(' expr ')' {
					$$ = &Node{
						stype: VARG,
						children: []*Node{$1,$5},
					}
				}
	;
narglist: /* nothing */		{ $$ = nil }
	| args
	| DOTDOTDOT		{
					$$ = &Node{
						stype: DOTDOTDOT,
						children: []*Node{},
					}
				}
	| args ',' DOTDOTDOT	{
					$$ = &Node{
						stype: DOTDOTDOT,
						children: []*Node{$1},
					}
				}
	| VARG '(' expr ')'	{
					$$ = &Node{
						stype: VARG,
						children: []*Node{$3},
					}
				}
	| args ',' VARG	'(' expr ')' {
					$$ = &Node{
						stype: VARG,
						children: []*Node{$1,$5},
					}
				}
	;
args	: expr
	| args ',' expr		{
					$$ = &Node{
						stype: ',',
						children: []*Node{$1,$3},
					}
				}
	;
arritem	: expr '=' expr {
					$$ = &Node{
						stype: ARRITEMEQ,
						children: []*Node{$1,$3},
					}
}
	;
arrlist	: /* nothing */		{ $$ = nil }
	| arritem  { $$ = $1 }
	| arrlist ',' arritem	{
					$$ = &Node{
						stype: ',',
						children: []*Node{$1,$3},
					}
				}
	;
globvar	: var
	;
globlist: globvar
	| globlist ',' globvar
	;
var	: NAME
	| VOL
	| DUR
	| CHAN
	| PORT
	| TIME
	| PITCH
	| LENGTH
	| TYPE
	| ATTRIB
	| FLAGS
	| NUMBER
	| CLASS
	;
methname: NAME
	| SYM_DELETE
	| CLASS
	;
method	: NAME
	| SYM_DELETE  {
					$$ = &Node{
						stype: SYM_DELETE,
						children: []*Node{},
					}
				}
	| CLASS	  {
					$$ = &Node{
						stype: CLASS,
						children: []*Node{},
					}
				}
	| '(' expr ')'	{
				$$ = $2;
				}
	;
%%

