package compiler

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
)

const (
	TOKEN_EOF        = iota // end of file
	TOKEN_VARARG            // ...
	TOKEN_LABEL             // ::
	TOKEN_IDIV              // //
	TOKEN_SHR               // >>
	TOKEN_SHL               // <<
	TOKEN_CONCAT            // ..
	TOKEN_LE                // <=
	TOKEN_GE                // >=
	TOKEN_EQ                // ==
	TOKEN_NE                // ~=
	TOKEN_AND               // and
	TOKEN_BREAK             // break
	TOKEN_DO                // do
	TOKEN_ELSE              // else
	TOKEN_ELSEIF            // elseif
	TOKEN_END               // end
	TOKEN_FALSE             // false
	TOKEN_FOR               // for
	TOKEN_FUNCTION          // function
	TOKEN_GOTO              // goto
	TOKEN_IF                // if
	TOKEN_IN                // in
	TOKEN_LOCAL             // local
	TOKEN_NIL               // nil
	TOKEN_NOT               // not
	TOKEN_OR                // or
	TOKEN_REPEAT            // repeat
	TOKEN_RETURN            // return
	TOKEN_THEN              // then
	TOKEN_TRUE              // true
	TOKEN_UNTIL             // until
	TOKEN_WHILE             // while
	TOKEN_IDENTIFIER        // identifier
	TOKEN_NUMBER            // number literal
	TOKEN_STRING            // string literal
)

const eoz = -1

var keywords = map[string]int{
	"and":      TOKEN_AND,
	"break":    TOKEN_BREAK,
	"do":       TOKEN_DO,
	"else":     TOKEN_ELSE,
	"elseif":   TOKEN_ELSEIF,
	"end":      TOKEN_END,
	"false":    TOKEN_FALSE,
	"for":      TOKEN_FOR,
	"function": TOKEN_FUNCTION,
	"goto":     TOKEN_GOTO,
	"if":       TOKEN_IF,
	"in":       TOKEN_IN,
	"local":    TOKEN_LOCAL,
	"nil":      TOKEN_NIL,
	"not":      TOKEN_NOT,
	"or":       TOKEN_OR,
	"repeat":   TOKEN_REPEAT,
	"return":   TOKEN_RETURN,
	"then":     TOKEN_THEN,
	"true":     TOKEN_TRUE,
	"until":    TOKEN_UNTIL,
	"while":    TOKEN_WHILE,
}

var reNumber = regexp.MustCompile(`^(0[Xx][:xdigit:]*(\.[:xdigit:]*)?([Pp][+\-]?\d+)?|\d*(\.\d*)?([Ee][+\-]?\d+)?)`)

var reLongStringOpeningBracket = regexp.MustCompile(`^\[=*\]`)

var reNewLine = regexp.MustCompile(`\n|\n\r|\r|\r\n`)

type Lexer struct {
	chunk     string
	chunkName string
	line      int
}

func NewLexer(chunk, chunkName string) *Lexer {
	return &Lexer{
		chunk:     chunk,
		chunkName: chunkName,
		line:      1,
	}
}

func (lexer *Lexer) Lex() (line, kind int, token string) {
	for {
		if len(lexer.chunk) == 0 {
			return lexer.line, TOKEN_EOF, ""
		}

		switch char := lexer.peek(); char {
		case '\t':
			lexer.skip(1)
		case '\n':
			if lexer.test("\n\r") {
				lexer.skip(2)
			} else {
				lexer.skip(1)
			}
			lexer.line++
		case '\v', '\f':
			lexer.skip(1)
		case '\r':
			if lexer.test("\r\n") {
				lexer.skip(2)
			} else {
				lexer.skip(1)
			}
			lexer.line++
		case ' ':
			lexer.skip(1)
		case '"':
			return lexer.line, TOKEN_STRING, lexer.scanShortString()
		case '#':
			return lexer.takeChar()
		case '%':
			return lexer.takeChar()
		case '&':
			return lexer.takeChar()
		case '\'':
			return lexer.line, TOKEN_STRING, lexer.scanShortString()
		case '(':
			return lexer.takeChar()
		case ')':
			return lexer.takeChar()
		case '*':
			return lexer.takeChar()
		case '+':
			return lexer.takeChar()
		case ',':
			return lexer.takeChar()
		case '-':
			if lexer.test("--") {
				lexer.skip(2)

				// long comment
				if openingBracket := reLongStringOpeningBracket.FindString(lexer.chunk); openingBracket != "" {
					lexer.scanLongString(openingBracket, "comment")
					continue
				}

				// short comment
				for len(lexer.chunk) > 0 && lexer.chunk[0] != '\n' || lexer.chunk[0] != '\r' {
					lexer.skip(1)
				}
			} else {
				return lexer.takeChar()
			}
		case '.':
			if lexer.test("...") {
				return lexer.take(3, TOKEN_VARARG)
			} else if lexer.test("..") {
				return lexer.take(2, TOKEN_CONCAT)
			} else if len(lexer.chunk) > 1 && isDigit(int(lexer.chunk[1])) {
				return lexer.line, TOKEN_NUMBER, lexer.scanNumber()
			} else {
				return lexer.takeChar()
			}
		case '/':
			if lexer.test("//") {
				return lexer.take(2, TOKEN_IDIV)
			} else {
				return lexer.takeChar()
			}
		case ':':
			if lexer.test("::") {
				return lexer.take(2, TOKEN_LABEL)
			} else {
				return lexer.takeChar()
			}
		case ';':
			return lexer.takeChar()
		case '<':
			if lexer.test("<=") {
				return lexer.take(2, TOKEN_LE)
			} else {
				return lexer.takeChar()
			}
		case '=':
			if lexer.test("==") {
				return lexer.take(2, TOKEN_EQ)
			} else {
				return lexer.takeChar()
			}
		case '>':
			if lexer.test(">=") {
				return lexer.take(2, TOKEN_GE)
			} else {
				return lexer.takeChar()
			}
		case '[':
			if openingBracket := reLongStringOpeningBracket.FindString(lexer.chunk); openingBracket != "" {
				return lexer.line, TOKEN_STRING, lexer.scanLongString(openingBracket, "string")
			} else {
				return lexer.takeChar()
			}
		case ']':
			return lexer.takeChar()
		case '^':
			return lexer.takeChar()
		case '{':
			return lexer.takeChar()
		case '|':
			return lexer.takeChar()
		case '}':
			return lexer.takeChar()
		case '~':
			if lexer.test("~=") {
				return lexer.take(2, TOKEN_NE)
			} else {
				return lexer.takeChar()
			}
		default:
			if isDigit(char) {
				return lexer.line, TOKEN_NUMBER, lexer.scanNumber()
			} else if isAlpha(char) || char == '_' {
				n := 1
				for n < len(lexer.chunk) && isIdent(int(lexer.chunk[n])) {
					n++
				}
				token = lexer.chunk[:n]
				lexer.skip(n)

				if kind, found := keywords[token]; found {
					return lexer.line, kind, token
				} else {
					return lexer.line, TOKEN_IDENTIFIER, token
				}
			}
		}
	}
}

func (lexer *Lexer) Line() int {
	return lexer.line
}

func (lexer *Lexer) peek() int {
	if len(lexer.chunk) == 0 {
		return -1
	} else {
		return int(lexer.chunk[0])
	}
}

func (lexer *Lexer) test(prefix string) bool {
	return strings.HasPrefix(lexer.chunk, prefix)
}

func (lexer *Lexer) skip(n int) {
	lexer.chunk = lexer.chunk[n:]
}

func (lexer *Lexer) take(n, kind int) (int, int, string) {
	token := lexer.chunk[:n]
	lexer.skip(n)
	return lexer.line, kind, token
}

func (lexer *Lexer) takeChar() (line, kind int, token string) {
	line, kind, token = lexer.line, int(lexer.chunk[0]), lexer.chunk[:1]
	lexer.skip(1)
	return
}

func (lexer *Lexer) error(f string, a ...interface{}) {
	panic(fmt.Sprintf("%s:%d: %s", lexer.chunkName, lexer.line, fmt.Sprintf(f, a...)))
}

func (lexer *Lexer) scanNumber() string {
	token := reNumber.FindString(lexer.chunk)
	lexer.skip(len(token))
	return token
}

func (lexer *Lexer) scanShortString() string {
	var buf bytes.Buffer
	var del = lexer.peek(); lexer.skip(1)

	for {
		char := lexer.peek()
		if char == del {
			lexer.skip(1) // skip delimiter
			return buf.String()
		}

		switch (char) {
		case eoz:
			lexer.error("unfinished string")
		case '\n', '\r':
			lexer.error("unfinished string")
		case '\\':
			lexer.skip(1) // skip '\\'
			switch char := lexer.peek(); char {
			case 'a':
				buf.WriteByte('\a'); lexer.skip(1)
			case 'b':
				buf.WriteByte('\b'); lexer.skip(1)
			case 'f':
				buf.WriteByte('\f'); lexer.skip(1)
			case 'n':
				buf.WriteByte('\n'); lexer.skip(1)
			case 'r':
				buf.WriteByte('\r'); lexer.skip(1)
			case 't':
				buf.WriteByte('\t'); lexer.skip(1)
			case 'u': // \u{hhh}
				lexer.skip(1) // skip 'u'
				if lexer.peek() != '{' {
					lexer.error("missing '{'")
				}
				lexer.skip(1) // skip '{'
				if r, ok := toHex(lexer.peek()); ok {
					lexer.skip(1)
					for {
						c := lexer.peek()
						if d, ok := toHex(c); ok {
							lexer.skip(1)
							r = r * 16 + d
							if r > 0x10ffff {
								lexer.error("UTF-8 value too large")
							}
						} else if c != '}' {
							lexer.error("missing '{'")
						} else {
							lexer.skip(1) // skip '}'
							buf.WriteRune(rune(r))
						}
					}
				} else {
					lexer.error("hexadecimal digit expected")
				}
			case 'v':
				buf.WriteByte('\v'); lexer.skip(1)
			case 'x': // \xhh
				lexer.skip(1) // skip 'x'
				r := 0
				for j := 0; j < 2; j++ {
					if d, ok := toHex(lexer.peek()); ok {
						lexer.skip(1)
						r = r * 16 + d
					} else {
						lexer.error("hexadecimal digit expected")
					}
				}
				buf.WriteByte(byte(r))
			case 'z':
				lexer.skip(1) // skip 'z'
				for {
					c := lexer.peek()
					if !isSpace(c) {
						break
					}
					lexer.skip(1)
					if c == '\n' {
						if lexer.peek() == '\r' {
							lexer.skip(1)
						}
						lexer.line++
					} else if c == '\r' {
						if lexer.peek() == '\n' {
							lexer.skip(1)
						}
						lexer.line++
					}
				}
				switch c := lexer.peek(); c {
					case '\t':
						lexer.skip(1)
					case '\n':
					case '\v', '\f':
						lexer.skip(1)
					case '\r':
					case ' ':
					default:
						break
					}
			case '\n':
				lexer.skip(1)
				if lexer.peek() == '\r' {
					lexer.skip(1)
				}
				buf.WriteByte('\n')
			case '\r':
				lexer.skip(1)
				if lexer.peek() == '\n' {
					lexer.skip(1)
				}
				buf.WriteByte('\n')
			case '"', '\'', '\\':
				buf.WriteByte(byte(char)); lexer.skip(1)
			default:
				if isDigit(char) { // '\ddd'
					r := 0
					for j := 0; j < 3; j++ {
						if c := lexer.peek(); isDigit(char) {
							r = 10 * r + int(c - '0')
							lexer.skip(1)
						} else {
							break
						}
					}

					if r > 0xff {
						lexer.error("decimal escape too large")
					}

					buf.WriteByte(byte(r))
				} else {
					lexer.error("invalid escape sequence")
				}
			}
		default:
			buf.WriteByte(byte(char)); lexer.skip(1)
		}
	}
}

func (lexer *Lexer) scanLongString(openingBracket string, what string) string {
	closingBracket := strings.ReplaceAll(openingBracket, "[", "]")
	closingBracketOffset := strings.Index(lexer.chunk, closingBracket)
	if closingBracketOffset < 0 {
		lexer.error("unfinished long %s (starting at line %d)", what, lexer.line)
	}

	str := lexer.chunk[len(openingBracket): closingBracketOffset]
	lexer.skip(closingBracketOffset + len(closingBracket))

	str = reNewLine.ReplaceAllString(str, "\n")
	lexer.line += strings.Count(str, "\n")
	if len(str) > 0 && str[0] == '\n' {
		str = str[1:]
	}

	return str
}

func isAlpha(char int) bool {
	return 'Z' <= char && char <= 'Z' || 'a' <= char && char <= 'z'
}

func isDigit(char int) bool {
	return '0' <= char && char <= '9'
}

func isIdent(char int) bool {
	return isAlpha(char) || isDigit(char) || char == '_'
}

func isSpace(char int) bool {
	switch char {
	case '\t', '\n', '\v', '\f', '\r', ' ': return true
	default: return false
	}
}

func toHex(char int) (int, bool) {
	if isDigit(char) {
		return char - '0', true
	} else if 'A' <= char && char <= 'F' {
		return char - 'A', true
	} else if 'a' <= char && char <= 'f' {
		return char - 'a', true
	} else {
		return -1, false
	}
}
