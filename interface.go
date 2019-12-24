package go_uci

import (
	"strings"
)

type Package struct {
	Name     string
	Sections []Section
}

func (p *Package) sectionByType(typ string) []*Section {
	r := []*Section{}
	for i := range p.Sections {
		sec := &p.Sections[i]
		if sec.Type == typ {
			r = append(r, sec)
		}
	}
	return r
}

type Section struct {
	Type    string
	Name    string
	Options map[string]Option
}

type Option struct {
	Name string

	val  string
	vals []string
}

func (o *Option) Bool() bool {
	switch o.val {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func (o *Option) List() []string {
	if len(o.vals) > 0 {
		return o.vals
	}
	return []string{o.val}
}

func (o *Option) Value() string {
	if len(o.vals) > 0 {
		return strings.Join(o.vals, " ")
	}
	return o.val
}

func (o *Option) IsList() bool {
	return len(o.vals) > 0
}
