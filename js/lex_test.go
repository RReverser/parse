package js // import "github.com/tdewolff/parse/js"

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"testing"

	"github.com/tdewolff/test"
)

func helperStringify(t *testing.T, input string, index int) string {
	s := ""
	l := NewLexer(bytes.NewBufferString(input))
	for i := 0; i <= index; i++ {
		tt, data := l.Next()
		if tt == ErrorToken {
			if l.Err() != nil {
				s += tt.String() + "('" + l.Err().Error() + "')"
			} else {
				s += tt.String() + "(nil)"
			}
			break
		} else if tt == WhitespaceToken {
			continue
		} else {
			s += tt.String() + "('" + string(data) + "') "
		}
	}
	return s + " with code: " + strconv.Quote(input)
}

////////////////////////////////////////////////////////////////

type TTs []TokenType

func TestTokens(t *testing.T) {
	var tokenTests = []struct {
		js       string
		expected []TokenType
	}{
		{" \t\v\f\u00A0\uFEFF\u2000", TTs{}}, // WhitespaceToken
		{"\n\r\r\n\u2028\u2029", TTs{LineTerminatorToken}},
		{"5.2 .04 0x0F 5e99", TTs{NumericToken, NumericToken, NumericToken, NumericToken}},
		{"a = 'string'", TTs{IdentifierToken, PunctuatorToken, StringToken}},
		{"/*comment*/ //comment", TTs{CommentToken, CommentToken}},
		{"{ } ( ) [ ]", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{". ; , < > <=", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{">= == != === !==", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"+ - * % ++ --", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"<< >> >>> & | ^", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"! ~ && || ? :", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"= += -= *= %= <<=", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{">>= >>>= &= |= ^= =>", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"a = /.*/g;", TTs{IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken}},

		{"/*co\nm\u2028m/*ent*/ //co//mment\u2029//comment", TTs{CommentToken, CommentToken, LineTerminatorToken, CommentToken}},
		{"$ _\u200C \\u2000 \u200C", TTs{IdentifierToken, IdentifierToken, IdentifierToken, ErrorToken}},
		{">>>=>>>>=", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken}},
		{"1/", TTs{NumericToken, PunctuatorToken}},
		{"1/=", TTs{NumericToken, PunctuatorToken}},
		{"010xF", TTs{NumericToken, NumericToken, IdentifierToken}},
		{"50e+-0", TTs{NumericToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken}},
		{"'str\\i\\'ng'", TTs{StringToken}},
		{"'str\\\\'abc", TTs{StringToken, IdentifierToken}},
		{"'str\\\ni\\\\u00A0ng'", TTs{StringToken}},
		{"a = /[a-z/]/g", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		{"a=/=/g1", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		{"a = /'\\\\/\n", TTs{IdentifierToken, PunctuatorToken, RegexpToken, LineTerminatorToken}},
		{"a=/\\//g1", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		{"new RegExp(a + /\\d{1,2}/.source)", TTs{IdentifierToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, RegexpToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},

		{"0b0101 0o0707 0b17", TTs{NumericToken, NumericToken, NumericToken, NumericToken}},
		{"`template`", TTs{TemplateToken}},
		{"`a${x+y}b`", TTs{TemplateToken, IdentifierToken, PunctuatorToken, IdentifierToken, TemplateToken}},
		{"`temp\nlate`", TTs{TemplateToken}},
		{"`outer${{x: 10}}bar${ raw`nested${2}endnest` }end`", TTs{TemplateToken, PunctuatorToken, IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, TemplateToken, IdentifierToken, TemplateToken, NumericToken, TemplateToken, TemplateToken}},

		// early endings
		{"'string", TTs{StringToken}},
		{"'\n '\u2028", TTs{ErrorToken}},
		{"'str\\\U00100000ing\\0'", TTs{StringToken}},
		{"'strin\\00g'", TTs{StringToken}},
		{"/*comment", TTs{CommentToken}},
		{"a=/regexp", TTs{IdentifierToken, PunctuatorToken, RegexpToken}},
		{"\\u002", TTs{ErrorToken}},

		// coverage
		{"Ø a〉", TTs{IdentifierToken, IdentifierToken, ErrorToken}},
		{"0xg 0.f", TTs{NumericToken, IdentifierToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"0bg 0og", TTs{NumericToken, IdentifierToken, NumericToken, IdentifierToken}},
		{"\u00A0\uFEFF\u2000", TTs{}},
		{"\u2028\u2029", TTs{LineTerminatorToken}},
		{"\\u0029ident", TTs{IdentifierToken}},
		{"\\u{0029FEF}ident", TTs{IdentifierToken}},
		{"\\u{}", TTs{ErrorToken}},
		{"\\ugident", TTs{ErrorToken}},
		{"'str\u2028ing'", TTs{ErrorToken}},
		{"a=/\\\n", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, ErrorToken}},
		{"a=/x/\u200C\u3009", TTs{IdentifierToken, PunctuatorToken, RegexpToken, ErrorToken}},
		{"a=/x\n", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, IdentifierToken, LineTerminatorToken}},

		{"return /abc/;", TTs{IdentifierToken, RegexpToken, PunctuatorToken}},
		{"yield /abc/;", TTs{IdentifierToken, RegexpToken, PunctuatorToken}},
		{"{}/1/g", TTs{PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"({}/1/g)", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"({a:{}/1/g})", TTs{PunctuatorToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken}},
		{"+{}/1/g", TTs{PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"return {}/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"return\n{}/1/g", TTs{IdentifierToken, LineTerminatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"yield {}/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"typeof {}/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"void {}/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"0 in {}/1/g", TTs{NumericToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"case {}/1/g:", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"finally {}/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"else {}/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"catch(e){}/1/g", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"i(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"if(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken}},
		{"while(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken}},
		{"for(;;)/1/g", TTs{IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"with(0)/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, RegexpToken}},
		{"this/1/g", TTs{IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"case /1/g:", TTs{IdentifierToken, RegexpToken, PunctuatorToken}},
		{";function f(){}/1/g", TTs{PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"{}function f(){}/1/g", TTs{PunctuatorToken, PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"()function f(){}/1/g", TTs{PunctuatorToken, PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"[]function f(){}/1/g", TTs{PunctuatorToken, PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"x function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"0 function f(){}/1/g", TTs{NumericToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"/1/ function f(){}/1/g", TTs{RegexpToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"(function f(){}/1/g)", TTs{PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"[function f(){}/1/g]", TTs{PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"0,function f(){}/1/g", TTs{NumericToken, PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"+function f(){}/1/g", TTs{PunctuatorToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"case function f(){}/1/g:", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"throw function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"void function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"new function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"return function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"delete function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"x instanceof function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"x in function f(){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"case function (){}/1/g:", TTs{IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken, PunctuatorToken}},
		{"export default function (){}/1/g", TTs{IdentifierToken, IdentifierToken, IdentifierToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, PunctuatorToken, RegexpToken}},
		{"this.return/1/g", TTs{IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},
		{"(a+b)/1/g", TTs{PunctuatorToken, IdentifierToken, PunctuatorToken, IdentifierToken, PunctuatorToken, PunctuatorToken, NumericToken, PunctuatorToken, IdentifierToken}},

		// go fuzz
		{"`", TTs{ErrorToken}},
	}

	passed := 0

	for _, tt := range tokenTests {
		l := NewLexer(bytes.NewBufferString(tt.js))
		i := 0
		j := 0
		for {
			token, _ := l.Next()
			j++
			if token == WhitespaceToken {
				continue
			}
			expected := ErrorToken
			if i < len(tt.expected) {
				expected = tt.expected[i]
			}
			if token != expected {
				stringify := helperStringify(t, tt.js, j)
				test.String(t, token.String(), expected.String(), "token types must match at index "+strconv.Itoa(i)+" in "+stringify)
				break
			}
			if i == len(tt.expected) {
				stringify := helperStringify(t, tt.js, j)
				test.Error(t, l.Err(), io.EOF, "in "+stringify)
			}
			if expected == ErrorToken {
				passed++
				break
			}
			i++
		}
	}

	if passed != len(tokenTests) {
		t.Logf("Failed %d / %d token tests", len(tokenTests)-passed, len(tokenTests))
	}

	test.String(t, WhitespaceToken.String(), "Whitespace")
	test.String(t, TokenType(100).String(), "Invalid(100)")
}

////////////////////////////////////////////////////////////////

func ExampleNewLexer() {
	l := NewLexer(bytes.NewBufferString("var x = 'lorem ipsum';"))
	out := ""
	for {
		tt, data := l.Next()
		if tt == ErrorToken {
			break
		}
		out += string(data)
		l.Free(len(data))
	}
	fmt.Println(out)
	// Output: var x = 'lorem ipsum';
}
