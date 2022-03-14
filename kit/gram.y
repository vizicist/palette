%{
package kit
import (
	"text/scanner"
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
%}
%union{
	token Token
	expr  Expression
}
%type<expr> program
%type<expr> expr
%token<token> NUMBER NOT AND OR IS
%left IS
%left '<' GE '>' LE
%left AND
%left OR
%right NOT
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
		$$ = NumExpr{literal: $1.literal}
	}
	| NOT expr
	{
		$$ = UnaryExpr{operator: "!", right: $2}
	}
	| expr AND expr
	{
		$$ = AssocExpr{left: $1, operator: "&&", right: $3}
	}
	| '(' expr ')'
	{
		$$ = ParenExpr{SubExpr: $2}
	}
	| expr OR expr
	{
		$$ = AssocExpr{left: $1, operator: "||", right: $3}
	}
	| NUMBER '<' NUMBER
	{
		$$ = BinOpExpr{left: NumExpr{literal: $1.literal}, operator: "<", right: NumExpr{literal: $3.literal}}
	}
	| NUMBER '>' NUMBER
	{
		$$ = BinOpExpr{left: NumExpr{literal: $1.literal}, operator: ">", right: NumExpr{literal: $3.literal}}
	}
	| NUMBER IS NUMBER
	{
		$$ = BinOpExpr{left: NumExpr{literal: $1.literal}, operator: "=", right: NumExpr{literal: $3.literal}}
	}
	| NUMBER GE NUMBER
	{
		$$ = BinOpExpr{left: NumExpr{literal: $1.literal}, operator: ">=", right: NumExpr{literal: $3.literal}}
	}
	| NUMBER LE NUMBER
	{
		$$ = BinOpExpr{left: NumExpr{literal: $1.literal}, operator: "<=", right: NumExpr{literal: $3.literal}}
	}
%%
type Lexer struct {
	scanner.Scanner
	Vars map[string]interface{}
	result Expression
}
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
			return ! right
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
func Parse(exp string, vars map[string]interface{}) bool {
	l := new(Lexer)
	l.Vars = vars
	l.Init(strings.NewReader(exp))
	yyParse(l)
	return Eval(l.result)
}
