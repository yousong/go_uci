package go_uci

import (
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// package := [<section>]
// section := "config" <section-type> [<name>] "\n" [<option>]
// option := "option" <name> <value>
// option := "list" <name> <value>
// value := " [^"]* " <value>
// value := ' [^']* ' <value>
// value := [^ ]+ <value>

const (
	PACKAGE = "package"
	CONFIG  = "config"
	SECTION = "section"
	OPTION  = "option"
	LIST    = "list"
	NEWLINE = "\n"
)

type lexer struct {
	r   io.Reader
	rd  []rune
	tok []string

	row   int
	col   int
	debug bool
}

func newLexer(r io.Reader) *lexer {
	l := &lexer{
		r: r,

		row: 1,
		col: 0,
	}
	return l
}

func (l *lexer) incRow() {
	l.row += 1
	l.col = 0
}

func (l *lexer) incCol() {
	l.col += 1
}

func (l *lexer) catchErr(perr *error) {
	if val := recover(); val != nil {
		if err, ok := val.(error); ok {
			*perr = errors.Wrapf(err, "row %d, col %d", l.row, l.col)
			return
		}
		panic(val)
	}
}

var (
	errRune   = errors.New("bad rune")
	errSyntax = errors.New("bad syntax")
)

func (l *lexer) putRune(r rune) {
	l.rd = append(l.rd, r)
}

func (l *lexer) nextRune() (r rune) {
	defer func() {
		if l.debug {
			fmt.Printf("r: ,%c,\n", r)
		}
	}()
	if len(l.rd) > 0 {
		r := l.rd[0]
		l.rd = l.rd[1:]
		return r
	}
	b := make([]byte, utf8.UTFMax)
	for i := 0; i < len(b); i++ {
		n, err := l.r.Read(b[i : i+1])
		if n > 0 {
			if utf8.FullRune(b) {
				r, sz := utf8.DecodeRune(b[:i+1])
				if sz != i+1 {
					panic(errors.Wrapf(errRune, "decode rune"))
				}
				return r
			}
		}
		if err != nil {
			// discard non-full rune
			panic(errors.Wrap(err, "read"))
		}
	}
	panic(errRune)
}

func (l *lexer) putTok(s string) {
	l.tok = append(l.tok, s)
}

func (l *lexer) next() (s string, err error) {
	defer func() {
		if l.debug {
			fmt.Printf("tok: %q, err: %v\n", s, err)
		}
	}()
	if len(l.tok) > 0 {
		r := l.tok[0]
		l.tok = l.tok[1:]
		return r, nil
	}

	var (
		dquote = false
		squote = false
		esc    = false
		r      = []rune{}
	)

	defer func() {
		if err == nil {
			s = string(r)
			return
		}
		if errors.Cause(err) == io.EOF {
			if squote {
				err = errors.Wrap(err, "unclosed single quote")
			} else if dquote {
				err = errors.Wrap(err, "unclosed double quote")
			}
			s = string(r)
			if esc {
				s += "\\"
			}
			if s == "" {
				panic(io.EOF)
			}
			err = nil
		}
	}()
	defer l.catchErr(&err)

	l.skipSpace()
	for {
		c := l.nextRune()
		switch {
		case c == '"':
			if !esc {
				if !squote {
					dquote = !dquote
				} else {
					r = append(r, '"')
				}
			} else {
				r = append(r, '"')
				esc = false
			}
		case c == '\'':
			if !esc {
				if !dquote {
					squote = !squote
				} else {
					r = append(r, '\'')
				}
			} else {
				r = append(r, '\'')
				esc = false
			}
		case c == '\\':
			if squote {
				r = append(r, '\\')
				break
			}
			if esc {
				r = append(r, '\\')
				esc = false
				break
			}
			esc = true
		case c == '\n':
			l.incRow()
			if squote || dquote {
				r = append(r, '\n')
				if esc {
					esc = false
				}
				break
			}
			if esc {
				esc = false
				break
			}
			if len(r) > 0 {
				l.putRune(c)
			} else {
				r = append(r, '\n')
			}
			return
		case c == '#':
			if squote || dquote || esc {
				r = append(r, '#')
				break
			}
			for {
				c := l.nextRune()
				l.incCol()
				if c == '\n' {
					l.incRow()
					if len(r) > 0 {
						l.putRune(c)
					} else {
						r = append(r, c)
					}
					return
				}
			}
		case l.isspace(c):
			if esc || squote || dquote {
				r = append(r, ' ')
				if esc {
					esc = false
				}
			} else {
				return
			}
		default:
			if !esc {
				r = append(r, c)
			} else {
				r = append(r, l.escape(c))
				esc = false
			}
		}
	}
}

func (l *lexer) fetch() string {
	s, err := l.next()
	if err != nil {
		panic(err)
	}
	return s
}

func (l *lexer) fetchConst(s string) {
	got := l.fetch()
	if s != got {
		panic(errors.Wrap(errSyntax, "want "+s+", got "+got))
	}
}

func (l *lexer) fetchType() string {
	got := l.fetch()
	if !l.validTypeName(got) {
		panic(errors.Wrap(errSyntax, "want type"))
	}
	return got
}

func (l *lexer) fetchName() string {
	got := l.fetch()
	if !l.validName(got) {
		panic(errors.Wrap(errSyntax, "want name"))
	}
	return got
}

func (l *lexer) err(msg string) error {
	return fmt.Errorf("line %d, column %d: %s", l.row, l.col, msg)
}

func (l *lexer) escape(c rune) rune {
	switch c {
	case 'n':
		return '\n'
	case 't':
		return '\t'
	case 'v':
		return '\v'
	}
	return c
}

func (l *lexer) validPackageName(s string) bool {
	return l.validateStr(s, true, false)
}

func (l *lexer) validTypeName(s string) bool {
	return l.validateStr(s, false, true)
}

func (l *lexer) validName(s string) bool {
	return l.validateStr(s, false, false)
}

func (l *lexer) validateStr(s string, isPackage, isType bool) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if unicode.IsLetter(c) || unicode.IsNumber(c) || c == '_' {
			continue
		}
		if isPackage && c == '-' {
			continue
		}
		if !isType || (c < 33) || (c > 126) {
			return false
		}
	}
	return true
}

func (l *lexer) skipSpace() {
	for {
		c := l.nextRune()
		if !l.isspace(c) {
			l.putRune(c)
			return
		}
		l.incCol()
	}
}

func (l *lexer) isspace(c rune) bool {
	if c == '\n' {
		return false
	}
	if unicode.IsSpace(c) {
		return true
	}
	return false
}

type parser struct {
	l *lexer

	eof   bool
	flags uint32
}

const (
	f_allow_eof = 1 << iota
	f_debug
)

func newParser(r io.Reader) *parser {
	l := newLexer(r)
	p := &parser{
		l: l,
	}
	return p
}

func (p *parser) setFlag(f uint32) {
	p.flags |= f
	p.l.debug = (p.flags&f_debug != 0)
}

func (p *parser) clearFlag(f uint32) {
	p.flags ^= f
	p.l.debug = (p.flags&f_debug != 0)
}

func (p *parser) allowEof() bool {
	return (p.flags & f_allow_eof) != 0
}

// skipNewline skips at least 1 newlines.  If other tokens or EOF
// were encountered in this process, it panics
func (p *parser) skipNewline() {
	nl := false
	for {
		tok := p.l.fetch()
		if tok != NEWLINE {
			p.l.putTok(tok)
			break
		}
		nl = true
	}
	if !nl {
		panic(errors.Wrap(errSyntax, "newline"))
	}
}

// skipNewline0 skips 0 or more newlines.  If EOF were encountered, it panics
func (p *parser) skipNewline0() {
	for {
		tok := p.l.fetch()
		if tok != NEWLINE {
			p.l.putTok(tok)
			break
		}
	}
}

func (p *parser) parsePackage() (p_ Package, err error) {
	defer func() {
		if panicVal := recover(); panicVal != nil {
			if panicErr, ok := panicVal.(error); ok {
				if errors.Cause(panicErr) == io.EOF && p.allowEof() {
					err = nil
					return
				}
				err = panicErr
				return
			}
			panic(panicVal)
		}
	}()

	p.setFlag(f_allow_eof)
	p.skipNewline0()
	w0 := p.l.fetch()
	if w0 == PACKAGE {
		p.clearFlag(f_allow_eof)
		w1 := p.l.fetch()
		if !p.l.validPackageName(w1) {
			panic("bad package name")
		}
		p_.Name = w1
		p.setFlag(f_allow_eof)
		p.skipNewline()
	} else {
		p.l.putTok(w0)
	}
	for {
		s_ := p.parseSection()
		p_.Sections = append(p_.Sections, s_)
	}
}

func (p *parser) parseSection() (s Section) {
	s.Options = map[string]Option{}
	p.setFlag(f_allow_eof)
	p.l.fetchConst(CONFIG)
	p.clearFlag(f_allow_eof)
	s.Type = p.l.fetchType()

	defer func() {
		if panicVal := recover(); panicVal != nil {
			if panicErr, ok := panicVal.(error); ok {
				if errors.Cause(panicErr) == io.EOF && p.allowEof() {
					return
				}
			}
			panic(panicVal)
		}
	}()

	{
		p.setFlag(f_allow_eof)
		w2 := p.l.fetch()
		if w2 == NEWLINE {
			// put it back so that later process has sth to skip
			p.l.putTok(w2)
		} else {
			if !p.l.validName(w2) {
				panic("bad section name")
			}
			s.Name = w2
		}
		p.skipNewline()
	}
	for {
		opt, eos := p.parseOption()
		if eos {
			return s
		}
		if old, ok := s.Options[opt.Name]; !ok {
			s.Options[opt.Name] = opt
		} else if old.IsList() && opt.IsList() {
			old.vals = append(old.vals, opt.vals...)
		} else if !old.IsList() && !opt.IsList() {
			old.val = opt.val
		} else {
			panic("option has different type")
		}
		p.setFlag(f_allow_eof)
		p.skipNewline()
	}
}

func (p *parser) parseOption() (Option, bool) {
	opt := Option{}

	p.setFlag(f_allow_eof)
	tokTyp := p.l.fetch()
	switch tokTyp {
	case OPTION, LIST:
	case CONFIG:
		p.l.putTok(tokTyp)
		return opt, true
	default:
		panic(errors.Wrapf(errSyntax, "want option, list, got %s", tokTyp))
	}
	p.clearFlag(f_allow_eof)

	name := p.l.fetchName()
	val := p.l.fetch()
	opt.Name = name
	if tokTyp == OPTION {
		opt.val = val
	} else if tokTyp == LIST {
		opt.vals = []string{val}
	}
	return opt, false
}
