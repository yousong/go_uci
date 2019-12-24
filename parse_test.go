package go_uci

import (
	"bytes"
	"io"
	"testing"

	"github.com/pkg/errors"
)

func TestLexer(t *testing.T) {
	in := `
config 'type' 'section'
        # Cannot preserve trailling whitespace with assertEquals.
        option opt "\"Hello, \
World.
\'"
`
	r := bytes.NewBuffer([]byte(in))
	t.Log(in)
	l := newLexer(r)
	for {
		tok, err := l.next()
		if err != nil {
			if errors.Cause(err) != io.EOF {
				t.Fatalf("err: %v", err)
			}
			break
		}
		t.Logf("%q", tok)
	}
}

func TestParser(t *testing.T) {
	type Case struct {
		name  string
		data  string
		debug bool
	}
	cases := []Case{
		{
			name: "empty",
			data: ``,
		},
		{
			name: "empty line",
			data: "\n",
		},
		{
			name: "blanks",
			data: "  \t",
		},
		{
			name: "package",
			data: "package network",
		},
		{
			name: "package line",
			data: "package network\n",
		},
		{
			name: "common network",
			data: `
config interface 'loopback'
	option ifname 'lo'
	option proto 'static'
	option ipaddr '127.0.0.1'
	option netmask '255.0.0.0'

config globals 'globals'
	option ula_prefix 'fd9d:111a:353b::/48'

config interface 'lan'
	option type 'bridge'
	option ifname 'eth0'
	option proto 'static'
	option ipaddr '192.168.1.1'
	option netmask '255.255.255.0'
	option ip6assign '60'

config interface 'wan'
	option ifname 'eth1'
	option proto 'dhcp'

config interface 'wan6'
	option ifname 'eth1'
	option proto 'dhcpv6'
`,
		},
		{
			name:  "multiline",
			debug: true,
			data: `
config 'type' 'section'
        # Cannot preserve trailling whitespace with assertEquals.
        option opt "\"Hello, \
World.
\'"
`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := bytes.NewBufferString(c.data)
			parser := newParser(r)
			if c.debug {
				parser.setFlag(f_debug)
			}
			p, err := parser.parsePackage()
			if err != nil {
				t.Errorf("parse package: %v", err)
			}
			t.Logf("%s", p.String())
			//t.Logf("%#v", p.sectionByType("interface"))
			//t.Logf("%#v", p)
		})
	}
}
