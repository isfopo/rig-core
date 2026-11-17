package midi

import (
	"bytes"
	"testing"

	"github.com/BurntSushi/toml"
	"gitlab.com/gomidi/midi/v2"
)

func noteEvent(ch, note, vel uint8) Event {
	return Event{Msg: midi.NoteOn(ch, note, vel)}
}

func ccEvent(ch, cc, val uint8) Event {
	return Event{Msg: midi.ControlChange(ch, cc, val)}
}

func intPtr(v int) *int { return &v }

func TestMappingValidate(t *testing.T) {
	cases := []struct {
		m  Mapping
		ok bool
	}{
		{Mapping{Type: "note", Number: 36, Action: "record"}, true},
		{Mapping{Type: "cc", Number: 7, Action: "volume"}, true},
		{Mapping{Type: "note_on", Number: 36, Action: "record"}, true},
		{Mapping{Type: "control_change", Number: 7, Action: "volume"}, true},
		{Mapping{Type: "foo", Number: 1, Action: "x"}, false},
		{Mapping{Type: "note", Number: 36, Action: ""}, false},
		{Mapping{Type: "note", Channel: 17, Number: 36, Action: "x"}, false},
		{Mapping{Type: "note", Channel: -1, Number: 36, Action: "x"}, false},
		{Mapping{Type: "note", Number: 200, Action: "x"}, false},
		{Mapping{Type: "cc", Number: 7, Action: "x", Value: intPtr(200)}, false},
	}
	for i, c := range cases {
		err := c.m.Validate()
		if c.ok && err != nil {
			t.Errorf("case %d: unexpected err: %v", i, err)
		}
		if !c.ok && err == nil {
			t.Errorf("case %d: expected err", i)
		}
	}
}

func TestMappingTableValidate(t *testing.T) {
	tbl := MappingTable{
		{Type: "note", Number: 36, Action: "record"},
		{Type: "cc", Number: 7, Action: "volume"},
	}
	if err := tbl.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := MappingTable{{Type: "nope", Number: 1, Action: "x"}}
	if err := bad.Validate(); err == nil {
		t.Fatal("expected err")
	}
}

func TestMatchNote(t *testing.T) {
	tbl := MappingTable{
		{Device: "Launchpad", Channel: 1, Type: "note", Number: 36, Action: "record"},
		{Type: "note", Number: 37, Action: "play"}, // any device/channel
	}
	ev := noteEvent(0, 36, 100) // channel 1
	m, ok := tbl.Match("Launchpad", ev)
	if !ok || m.Action != "record" {
		t.Fatalf("got %v %v", m, ok)
	}
	if _, ok := tbl.Match("Other", ev); ok {
		t.Fatal("unexpected match")
	}
	ev2 := noteEvent(2, 37, 50) // channel 3
	m, ok = tbl.Match("Foo", ev2)
	if !ok || m.Action != "play" {
		t.Fatalf("got %v %v", m, ok)
	}
}

func TestMatchCCAndValue(t *testing.T) {
	v := 127
	tbl := MappingTable{
		{Type: "cc", Number: 7, Action: "volume", Value: &v},
		{Type: "cc", Number: 7, Action: "volume_any"},
	}
	ev := ccEvent(0, 7, 127)
	m, ok := tbl.Match("", ev)
	if !ok || m.Action != "volume" {
		t.Fatalf("got %v %v", m, ok)
	}
	ev2 := ccEvent(0, 7, 10)
	m, ok = tbl.Match("", ev2)
	if !ok || m.Action != "volume_any" {
		t.Fatalf("got %v %v", m, ok)
	}
}

func TestNoteOnVelocityZeroNoMatch(t *testing.T) {
	tbl := MappingTable{{Type: "note", Number: 36, Action: "record"}}
	ev := Event{Msg: midi.NoteOn(0, 36, 0)}
	if _, ok := tbl.Match("", ev); ok {
		t.Fatal("velocity-0 note-on should not match")
	}
}

func TestForDeviceAndDevices(t *testing.T) {
	tbl := MappingTable{
		{Device: "Launchpad", Type: "note", Number: 1, Action: "a"},
		{Device: "launchpad", Type: "note", Number: 2, Action: "b"},
		{Device: "", Type: "note", Number: 3, Action: "c"},
	}
	sub := tbl.ForDevice("Launchpad")
	if len(sub) != 3 {
		t.Fatalf("len=%d (want 3: exact, case variant, wildcard)", len(sub))
	}
	devs := tbl.Devices()
	if len(devs) != 2 {
		t.Fatalf("devices=%v (want 2 unique names)", devs)
	}
}

func TestTOMLRoundTrip(t *testing.T) {
	type cfg struct {
		Mappings MappingTable `toml:"mappings"`
	}
	in := cfg{Mappings: MappingTable{
		{Device: "Launchpad", Channel: 1, Type: "note", Number: 36, Action: "record"},
		{Device: "Launchpad", Channel: 0, Type: "cc", Number: 7, Action: "volume", Value: intPtr(100)},
	}}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(in); err != nil {
		t.Fatal(err)
	}
	var out cfg
	if _, err := toml.Decode(buf.String(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Mappings) != 2 {
		t.Fatalf("len=%d", len(out.Mappings))
	}
	if out.Mappings[0].Action != "record" {
		t.Fatalf("action=%s", out.Mappings[0].Action)
	}
	if out.Mappings[1].Value == nil || *out.Mappings[1].Value != 100 {
		t.Fatalf("value=%v", out.Mappings[1].Value)
	}
}

func TestActionHelper(t *testing.T) {
	tbl := MappingTable{{Type: "note", Number: 36, Action: "record"}}
	ev := noteEvent(0, 36, 100)
	act, ok := tbl.Action("", ev)
	if !ok || act != "record" {
		t.Fatalf("got %s %v", act, ok)
	}
}
