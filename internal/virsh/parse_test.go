package virsh

import (
	"reflect"
	"testing"
)

func TestParseList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Domain
	}{
		{
			name: "running + shut off domains",
			in: ` Id   Name      State
----------------------------------------------------
 1    resolute  running

 -    demo      shut off

`,
			want: []Domain{
				{ID: "1", Name: "resolute", State: "running"},
				{ID: "-", Name: "demo", State: "shut off"},
			},
		},
		{
			name: "state containing a space (in shutdown)",
			in: ` Id   Name   State
----
 2    win11  in shutdown
`,
			want: []Domain{{ID: "2", Name: "win11", State: "in shutdown"}},
		},
		{
			name: "empty list: header and separator only",
			in: ` Id   Name   State
---------------------------------------

`,
			want: nil,
		},
		{
			name: "name with varied characters",
			in: ` Id   Name         State
----
 -    a-b_c.1     paused
`,
			want: []Domain{{ID: "-", Name: "a-b_c.1", State: "paused"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseList(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseList() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseDomInfo(t *testing.T) {
	in := `Id:             -
Name:           demo
UUID:           1234-abcd
OS Type:        hvm
State:          shut off
CPU(s):         2
Max memory:     2097152 KiB
Persistent:     yes
Autostart:      disable
Managed save:   no
`
	d := parseDomInfoOrdered(in)
	if len(d.Lines) != 10 {
		t.Fatalf("got %d lines, want 10: %+v", len(d.Lines), d.Lines)
	}
	m := parseDomInfoMap(in)
	for key, want := range map[string]string{
		"Name":      "demo",
		"State":     "shut off",
		"Autostart": "disable",
		"CPU(s)":    "2",
	} {
		if got := m[key]; got != want {
			t.Errorf("m[%q] = %q, want %q", key, got, want)
		}
	}
	if got := d.Get("Autostart"); got != "disable" {
		t.Errorf("Get(Autostart) = %q", got)
	}
	if got := d.Get("missing"); got != "" {
		t.Errorf("Get(missing) = %q, want empty", got)
	}
}
