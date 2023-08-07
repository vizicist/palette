package parse

import (
	"fmt"
)

var treeIndent int

type Node struct {
	stype    int // e.g. NUMBER, STRING, etc.
	sym      PkSymType
	children []*Node
	num      int // for NUMBERs
	name     string
	phr      string
	str      string
	dbl      float64
}

func HandleProgram(pklex PkLexer, n *Node) {
	mylex, ok := pklex.(*PkLex)
	if !ok {
		mylex.Outf.WriteString("<<bad type for pklex>>")
	} else {

		tp := n.treePostProcess()

		pp := tp.treePrint()
		fmt.Printf("%s\n", pp)

		pp = n.prettyPrint()
		fmt.Printf("%s\n", pp)
		mylex.Outf.WriteString(pp)
	}
}

func (n *Node) String() string {
	return fmt.Sprintf("Node(stype=%d)", n.stype)
}

func (n *Node) PrettyChild(childnum int) string {
	if n.children == nil {
		return ""
	}
	if childnum >= len(n.children) {
		// eprint(fmt.Sprintf("PrettyChild given childnum=%d when len(children)==%d", childnum, len(n.children)))
		return ""
	}
	return n.children[childnum].prettyPrint()
}

func (n *Node) prettyPrint() string {
	if n == nil {
		return ""
	}
	switch n.stype {
	case NUMBER:
		return fmt.Sprintf("%d", n.num)
	case DOUBLE:
		return fmt.Sprintf("%f", n.dbl)
	case NAME:
		return n.name
	case PHRASE:
		return fmt.Sprintf("Phrase(\"%s\")", n.phr)
	case STRING:
		return n.str
	case OBJECT:
		return fmt.Sprintf("%d", n.num)
	case PARAM:
		switch len(n.children) {
		case 1:
			return n.PrettyChild(0) + " Kval"
		case 2:
			return n.PrettyChild(0) + "," + n.PrettyChild(1) + " Kval"
		default:
			return "(Unexpected children in PARAM)"
		}
	case PRMLIST:
		s0 := n.PrettyChild(0)
		s1 := n.PrettyChild(1)
		if s1 != "" {
			s0 += "," + s1
		}
		return s0
	case DOTDOTDOT:
		return "..."
	case SEQUENCE:
		nchildren := len(n.children)
		switch nchildren {
		case 0:
			return "<<wrong number of children for SEQUENCE?>>"
		case 1:
			return n.PrettyChild(0) + ";"
		default:
			s := n.PrettyChild(0)
			for i := 1; i < nchildren; i++ {
				s += " " + n.PrettyChild(i) + ";"
			}
			return s
		}
	case FUNC:
		switch len(n.children) {
		case 3:
			name := n.children[0].name
			params := n.PrettyChild(1)
			stmts := n.PrettyChild(2)
			return fmt.Sprintf("func %s ( %s ) { %s }", name, params, stmts)
		case 2:
			name := n.children[0].name
			stmts := n.PrettyChild(1)
			return fmt.Sprintf("func %s () { %s }", name, stmts)
		default:
			return "<<wrong number of children for FUNC>>"
		}
	case FUNCCALL:
		switch len(n.children) {
		case 2:
			name := n.PrettyChild(0)
			arglist := n.PrettyChild(1)
			return fmt.Sprintf("%s ( %s )", name, arglist)
		case 3:
			expr := n.PrettyChild(0)
			method := n.PrettyChild(1)
			arglist := n.PrettyChild(2)
			return fmt.Sprintf("%s.%s ( %s )", expr, method, arglist)
		default:
			return "<<wrong number of children for FUNCCALL>>"
		}
	case RETURN:
		val := n.PrettyChild(0)
		return fmt.Sprintf("return ( %s )", val)
	case UNARYMINUS:
		val := n.PrettyChild(0)
		return fmt.Sprintf("-%s", val)
	case '$':
		return "$"
	case '{':
		switch len(n.children) {
		case 1:
			val := n.PrettyChild(0)
			return fmt.Sprintf("{ %s }", val)
		case 2:
			return "<<should 2 children be handled here?>>"
		default:
			return "<<curly brace with bad children>>"
		}
	case '~':
		val := n.PrettyChild(0)
		return fmt.Sprintf("~%s", val)
	case BANG:
		val := n.PrettyChild(0)
		return fmt.Sprintf("!%s", val)
	case INC:
		val := n.PrettyChild(0)
		return fmt.Sprintf("%s++", val)
	case DEC:
		val := n.PrettyChild(0)
		return fmt.Sprintf("%s--", val)
	case PREINC:
		val := n.PrettyChild(0)
		return fmt.Sprintf("++%s", val)
	case PREDEC:
		val := n.PrettyChild(0)
		return fmt.Sprintf("--%s", val)
	case '[':
		if len(n.children) == 1 {
			return "[ " + n.PrettyChild(0) + " ]"
		} else {
			arr := n.PrettyChild(0)
			index := n.PrettyChild(1)
			return fmt.Sprintf("%s [ %s ]", arr, index)
		}
	case '%':
		left := n.PrettyChild(0)
		right := n.PrettyChild(1)
		return fmt.Sprintf("%s %% %s", left, right)
	case SYM_IN:
		left := n.PrettyChild(0)
		right := n.PrettyChild(1)
		return fmt.Sprintf("%s in %s", left, right)
	case SELECTION:
		ph := n.PrettyChild(0)
		slct := n.PrettyChild(1)
		return fmt.Sprintf("%s { %s }", ph, slct)
	case '?':
		cond := n.PrettyChild(0)
		tval := n.PrettyChild(1)
		fval := n.PrettyChild(2)
		return fmt.Sprintf("Ternary( %s , %s , %s )", cond, tval, fval)
	case NARGS:
		return "nargs()"
	case TIME, DUR, LENGTH, CHAN, PORT, PITCH:
		return n.str
	case LT, GT, EQ, NE, AND, OR, RSHIFT, LSHIFT:
		left := n.PrettyChild(0)
		right := n.PrettyChild(1)
		op := binaryOp(n.stype)
		return fmt.Sprintf("%s %s %s", left, op, right)
	case EQUALS:
		left := n.PrettyChild(0)
		var eq string
		switch n.children[1].stype {
		case PLUSEQ:
			eq = "+="
		case MINUSEQ:
			eq = "-="
		case MULEQ:
			eq = "*="
		case DIVEQ:
			eq = "/="
		case OREQ:
			eq = "|="
		case AMPEQ:
			eq = "&="
		case XOREQ:
			eq = "^="
		case RSHIFTEQ:
			eq = ">>="
		case LSHIFTEQ:
			eq = "<<="
		case '=':
			eq = "="
		default:
			eq = "???=???"
		}
		right := n.PrettyChild(2)
		return fmt.Sprintf("%s %s %s ;", left, eq, right)
	case ARRITEMEQ:
		node0 := n.PrettyChild(0)
		node1 := n.PrettyChild(1)
		return fmt.Sprintf("%s = %s", node0, node1)
	case '=', '+', '-', '*', '/':
		left := n.PrettyChild(0)
		right := n.PrettyChild(1)
		return fmt.Sprintf("%s %c %s", left, n.stype, right)
	case ',':
		// return "<<COMMA?>>"
		left := n.PrettyChild(0)
		right := n.PrettyChild(1)
		return fmt.Sprintf("%s , %s", left, right)
	case IF:
		switch len(n.children) {
		case 2:
			cond := n.PrettyChild(0)
			stmts := n.PrettyChild(1)
			return fmt.Sprintf("if ( %s ) { %s }", cond, stmts)
		case 3:
			cond := n.PrettyChild(0)
			iftrue := n.PrettyChild(1)
			iffalse := n.PrettyChild(2)
			return fmt.Sprintf("if ( %s ) { %s } else { %s }", cond, iftrue, iffalse)
		default:
			return "(Unexpected children in IF)"
		}
	case WHILE:
		expr := n.PrettyChild(0)
		stmts := n.PrettyChild(1)
		return fmt.Sprintf("while ( %s ) { %s }", expr, stmts)
	case FOR:
		switch len(n.children) {
		case 3:
			variable := n.PrettyChild(0)
			inval := n.PrettyChild(1)
			stmts := n.PrettyChild(2)
			return fmt.Sprintf("for ( %s in %s ) { %s }", variable, inval, stmts)
		case 4:
			init := n.PrettyChild(0)
			cond := n.PrettyChild(1)
			inc := n.PrettyChild(2)
			stmts := n.PrettyChild(3)
			return fmt.Sprintf("for ( %s ; %s ; %s ) { %s }", init, cond, inc, stmts)
		default:
			eprint("Wrong number of children for FOR")
			return "for (xxxxx) { }"
		}
	case '(':
		p := n.PrettyChild(0)
		return fmt.Sprintf("( %s )", p)
	case '.':
		left := n.PrettyChild(0)
		right := n.PrettyChild(1)
		return fmt.Sprintf("%s.%s", left, right)
	case '\r':
		return ""
	case '\n':
		return "\n"
	default:
		eprint(fmt.Sprintf("TJT! unknown node stype=%d", n.stype))
		return fmt.Sprintf("(node styple=%d)", n.stype)
	}
}

func nspaces(n int) string {
	s := ""
	for i := 0; i < treeIndent; i++ {
		s += "    "
	}
	return s
}

func (n *Node) treePostProcess() *Node {
	if n == nil {
		return nil
	}
	// This code looks for chained assignments (e.g. a = b = 1)
	// that are supported in keykit but not in go,
	// and adjusts them.  a=b=1 turns into b=1;a=b
	if n.stype == EQUALS {
		if len(n.children) == 3 && n.children[2].stype == EQUALS {
			firsteq := n.children[2]
			secondeq := &Node{
				stype: EQUALS,
				children: []*Node{
					n.children[0],
					{
						stype:    '=',
						children: []*Node{},
					},
					n.children[2].children[0],
				},
			}
			newn := &Node{
				stype: SEQUENCE,
				children: []*Node{
					firsteq,
					secondeq,
				},
			}
			return newn
		}
	}
	if n.stype == ',' {
		if len(n.children) == 3 && n.children[2].stype == ',' {
			firsteq := n.children[2]
			secondeq := &Node{
				stype: ',',
				children: []*Node{
					n.children[0],
					{
						stype:    '=',
						children: []*Node{},
					},
					n.children[2].children[0],
				},
			}
			newn := &Node{
				stype: SEQUENCE,
				children: []*Node{
					firsteq,
					secondeq,
				},
			}
			return newn
		}
	}
	for i := 0; i < len(n.children); i++ {
		child := n.children[i]
		n.children[i] = child.treePostProcess()
	}
	return n
}

func (n *Node) treePrint() string {
	s := nspaces(treeIndent)
	if n == nil {
		return s + "NIL\n"
	}
	if n.stype < 128 {
		s += fmt.Sprintf("%c", n.stype)
	} else {
		s += Token[n.stype]
	}
	if n.name != "" {
		s += " " + n.name
	}
	if n.str != "" {
		s += " str=" + n.str
	}
	if n.phr != "" {
		s += " phr=" + n.phr
	}
	if n.stype == NUMBER {
		s += fmt.Sprintf(" %d", n.num)
	}
	s += "\n"
	treeIndent++
	for i := 0; i < len(n.children); i++ {
		child := n.children[i]
		s += child.treePrint()
	}
	treeIndent--
	return s
}

func makeNodeNil() *Node {
	return nil
}

func binaryOp(i int) string {
	switch i {
	case AND:
		return "&&"
	case OR:
		return "||"
	case LT:
		return "<"
	case GT:
		return ">"
	case EQ:
		return "=="
	case NE:
		return "!="
	case RSHIFT:
		return ">>"
	case LSHIFT:
		return "<<"
	default:
		return "???"
	}
}

func makeNodeObject(n int) *Node {
	return &Node{
		stype: OBJECT,
		num:   n,
	}
}

func makeNodeDouble(f float64) *Node {
	return &Node{
		stype: DOUBLE,
		dbl:   f,
	}
}

func makeNodeNumber(n int) *Node {
	return &Node{
		stype: NUMBER,
		num:   n,
	}
}

func makeNodeString(str string) *Node {
	return &Node{
		stype: STRING,
		str:   str,
	}
}

func makeNodePhrase(phr string) *Node {
	return &Node{
		stype: PHRASE,
		phr:   phr,
	}
}

func makeNodeInteger(n int) *Node {
	return &Node{
		stype: NUMBER,
		num:   n,
	}
}

func makeNodeOfName(s string) *Node {
	return &Node{
		stype: NAME,
		name:  s,
	}
}

func makeNodeOfType(i int) *Node {
	return &Node{
		stype: i,
	}
}
