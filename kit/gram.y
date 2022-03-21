%{
package kit

import (
	// "text/scanner"
	"strconv"
	"strings"
	"fmt"
)

type Expression interface{}
type ParenExpr struct {
	SubExpr Expression
}
type Token struct {
	token   int
	literal string
}
type NumExpr struct {
	literal string
}
type BinOpExpr struct {
	left     NumExpr
	operator string
	right    NumExpr
}
type AssocExpr struct {
	left     Expression
	operator string
	right    Expression
}
type UnaryExpr struct {
	operator string
	right    Expression
}

type Lexer struct {
	// scanner.Scanner
	reader *strings.Reader
	Vars   map[string]interface{}
	result Expression
	Yytext string
	yylval yySymType
}

type Instnode struct {
	code   Instcode
	inext  Instnodep
	offset int // only used in inodes2code() */
}

type Instnodep *Instnode
type Phrasep *Phrase

%}
%union{
	token Token
	expr  Expression
	sym Symbolp	/* symbol table pointer */
	in Instnodep /* machine instruction */
	num int /* number of arguments */
	val int	/* numeric constant */
	dbl float32	/* floating constant */
	str string /* string constant */
	phr Phrasep	/* phrase constant */
}

// %token<token> NUMBER NOT AND OR IS
/// %left '<' GE '>' LE
/// %left AND
/// %left OR

%left IS
%right NOT

%type   <expr> program
/// %type   <expr> expr
%token	<sym>	VAR UNDEF MACRO TOGLOBSYM QMARK2 DOLLAR2 WHILE DOTDOTDOT
%token	<sym>	IF ELSE FOR SYM_IN BEINGREAD EVAL BREAK CONTINUE TASK
%token	<sym>	SYM_DELETE UNDEFINE RETURN FUNC DEFINED READONLY ONCHANGE GLOBALDEC
%token	<sym>	CLASS METHOD KW_NEW NARGS TYPEOF XY
%token	<sym>	DUR VOL TIME CHAN PITCH LENGTH NUMBER TYPE ATTRIB FLAGS VARG PORT
%token	<phr>	PHRASE
%token	<str>	STRING NAME
%token	<val>	INTEGER OBJECT
%token	<dbl>	DOUBLE
%token	<num>	PLUSEQ MINUSEQ MULEQ DIVEQ AMPEQ INC DEC
%token	<num>	POSTINC POSTDEC OREQ XOREQ RSHIFTEQ LSHIFTEQ
%type	<in>	expr
/// %type   <in>    stmt stmts nosemi optstmt funcstart stmtnv
/// %type	<in>	tcond tfcond optrelx forin1 forin2 forinend end goto
/// %type	<in>	select1 select2 select3 and or equals
/// %type	<in>	prefunc1 prefunc3 preobj method arglist narglist
/// %type	<sym>	uniqvar var dottype globvar uniqm
/// %type	<num>	args prmlist prms arrlist
/// %type	<str>	methname
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

%%
program
	: expr
	{
		$$ = $1
		yylex.(*Lexer).result = $$
	}
expr
	: NUMBER
	{
		$$ = &Instnode{}
	}
%%

/*
func (l *Lexer) Lex(lval *yySymType) int {
	token := l.Scan()
	lit := l.TokenText()
	tok := int(token)
	switch tok {
	case scanner.Int:
		tok = NUMBER
	default:
		switch lit {
		case "IS":
			tok = IS
		case "NOT":
			tok = NOT
		case "AND":
			tok = AND
		case "OR":
			tok = OR
		case "<=":
			tok = LE
		case ">=":
			tok = GE
		default:
			if v, ok := l.Vars[lit]; ok {
				switch v.(type) {
				case int:
					tok = NUMBER
					lit = strconv.Itoa(v.(int))
				}
			}
		}
	}
	lval.token = Token{token: tok, literal: lit}
	return tok
}
*/
func (l *Lexer) Error(e string) {
	panic(e)
}
func EvalN(e Expression) int {
	switch t := e.(type) {
	case NumExpr:
		num, _ := strconv.Atoi(t.literal)
		return num
	}
	return 0
}
func Eval(e Expression) bool {
	switch t := e.(type) {
	case ParenExpr:
		fmt.Println("Sub")
		return Eval(t.SubExpr)
	case UnaryExpr:
		fmt.Println(t.operator)
		right := Eval(t.right)
		switch t.operator {
		case "!":
			return !right
		}
	case AssocExpr:
		fmt.Println("Assoc")
		left := Eval(t.left)
		right := Eval(t.right)
		switch t.operator {
		case "||":
			return left || right
		case "&&":
			return left && right
		}
	case BinOpExpr:
		fmt.Println("BinOp")
		left := EvalN(t.left)
		right := EvalN(t.right)
		switch t.operator {
		case ">":
			return left > right
		case "<":
			return left < right
		case "=":
			return left == right
		}
	default:
		fmt.Printf("unsuported expr[%+v]", t)
	}
	return false
}
func Parse(exp string, vars map[string]interface{}) (err error) {
	defer func() {
		if r := recover(); r!=nil {
			err = fmt.Errorf("recovered from %s", r)
		}
	}()
	l := new(Lexer)
	l.Vars = vars
	l.Init(exp)
	yyParse(l)
	b := Eval(l.result)
	if b == false {
		return fmt.Errorf("eval returned false")
	} else {
		return nil
	}
}