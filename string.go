package go_uci

import (
	"bytes"
	"fmt"
)

type buf struct {
	bytes.Buffer
}

func (b *buf) WriteStringf(f string, v ...interface{}) {
	s := fmt.Sprintf(f, v...)
	b.WriteString(s)
}

func quoted(s string) string {
	b := bytes.NewBufferString("'")
	for _, c := range s {
		if c == '\'' {
			b.WriteString("'\"'\"'")
		} else {
			b.WriteRune(c)
		}
	}
	b.WriteString("'")
	return b.String()
}

func (o *Option) String() string {
	b := &buf{}
	o.write(b)
	return b.String()
}

func (o *Option) write(b *buf) {
	if len(o.vals) > 0 {
		for _, val := range o.vals {
			b.WriteStringf("\tlist\t%s\t%s\n", o.Name, quoted(val))
		}
	} else {
		b.WriteStringf("\toption\t%s\t%s\n", o.Name, quoted(o.val))
	}
}

func (s *Section) String() string {
	b := &buf{}
	s.write(b)
	return b.String()
}

func (s *Section) write(b *buf) {
	b.WriteStringf("config %s", s.Type)
	if s.Name != "" {
		b.WriteStringf(" %s", quoted(s.Name))
	}
	b.WriteRune('\n')

	for _, opt := range s.Options {
		opt.write(b)
	}
}

func (p *Package) String() string {
	b := &buf{}
	p.write(b)
	return b.String()
}

func (p *Package) write(b *buf) {
	for _, s := range p.Sections {
		s.write(b)
		b.WriteRune('\n')
	}
}
