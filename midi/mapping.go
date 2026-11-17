// Package midi — config-driven MIDI mapping table.
//
// This file adds a reusable, config-driven mapping table to the midi
// package: a small schema that maps an incoming MIDI message (filtered by
// device, channel, message type and number) to a named action. Apps define
// their own action vocabulary (record, play, clear, ...) and consume the
// table at runtime; the core only provides parsing, validation and matching.
package midi

import (
	"fmt"
	"strings"
)

// MappingType identifies the kind of MIDI message a mapping responds to.
type MappingType string

const (
	// MappingNote matches Note-On messages (with non-zero velocity).
	MappingNote MappingType = "note"
	// MappingCC matches Control Change messages.
	MappingCC MappingType = "cc"
)

// normalized returns the canonical MappingType for t, accepting common
// aliases ("note_on", "control", "control_change", ...). Unknown values are
// returned lower-cased and trimmed so validation can reject them.
func (t MappingType) normalized() MappingType {
	s := strings.ToLower(strings.TrimSpace(string(t)))
	switch s {
	case "note", "noteon", "note_on", "note-on":
		return MappingNote
	case "cc", "control", "controlchange", "control_change", "control-change":
		return MappingCC
	}
	return MappingType(s)
}

// Mapping is one entry in a MIDI mapping table. It is designed to round-trip
// through TOML (see the config package) and is validated by Validate.
//
// Channel is 1-based to match gear documentation; a value of 0 acts as a
// wildcard that matches any channel. Device is matched case-insensitively
// against the port name the event arrived on; an empty Device matches any
// device. Value, when set, requires an exact match on the velocity (note) or
// controller value (CC).
type Mapping struct {
	Device  string      `toml:"device"`
	Channel int         `toml:"channel"`
	Type    MappingType `toml:"type"`
	Number  int         `toml:"number"`
	Action  string      `toml:"action"`
	Value   *int        `toml:"value,omitempty"`
}

// NormalizedType returns the canonical type of m.
func (m Mapping) NormalizedType() MappingType { return m.Type.normalized() }

// Validate checks that m is well-formed.
func (m Mapping) Validate() error {
	switch m.NormalizedType() {
	case MappingNote, MappingCC:
	default:
		return fmt.Errorf("midi: mapping type %q is not supported", m.Type)
	}
	if m.Channel < 0 || m.Channel > 16 {
		return fmt.Errorf("midi: mapping channel %d out of range (0..16)", m.Channel)
	}
	if m.Number < 0 || m.Number > 127 {
		return fmt.Errorf("midi: mapping number %d out of range (0..127)", m.Number)
	}
	if strings.TrimSpace(m.Action) == "" {
		return fmt.Errorf("midi: mapping action is required")
	}
	if m.Value != nil && (*m.Value < 0 || *m.Value > 127) {
		return fmt.Errorf("midi: mapping value %d out of range (0..127)", *m.Value)
	}
	return nil
}

// MappingTable is a list of Mapping entries with matching and validation
// helpers. It is the type apps should use for their config field, e.g.
//
//	mappings midi.MappingTable `toml:"mappings"`
//
// TOML schema (one entry per mapping):
//
//	[[midi.mappings]]
//	device  = "Launchpad"   # port name; empty = any device
//	channel = 1             # 1-based; 0 = any channel
//	type    = "note"        # "note" (or note_on) | "cc" (or control/control_change)
//	number  = 36            # note or CC number (0..127)
//	action  = "record"      # app-defined action name
//	value   = 100           # optional exact velocity/CC value filter
//
// Usage:
//
//	var cfg struct {
//	    Mappings midi.MappingTable `toml:"mappings"`
//	}
//	// load with config.LoadValidated(...)
//	action, ok := cfg.Mappings.Action(portName, ev)
//	if ok {
//	    // dispatch action
//	}
type MappingTable []Mapping

// Validate validates every entry in the table.
func (t MappingTable) Validate() error {
	for i, m := range t {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("midi: mapping #%d: %w", i+1, err)
		}
	}
	return nil
}

// ForDevice returns the subset of entries that apply to device (case-
// insensitive). Entries with an empty Device match every device.
func (t MappingTable) ForDevice(device string) MappingTable {
	var out MappingTable
	for _, m := range t {
		if m.Device == "" || strings.EqualFold(m.Device, device) {
			out = append(out, m)
		}
	}
	return out
}

// Devices returns the unique, non-empty device names referenced by the table,
// in first-seen order.
func (t MappingTable) Devices() []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range t {
		if m.Device == "" || seen[m.Device] {
			continue
		}
		seen[m.Device] = true
		out = append(out, m.Device)
	}
	return out
}

// Match returns the first mapping in the table that matches the given device
// and event. An empty device argument matches entries whose Device is empty.
func (t MappingTable) Match(device string, ev Event) (Mapping, bool) {
	for _, m := range t {
		if m.Device != "" && !strings.EqualFold(m.Device, device) {
			continue
		}
		if !m.matchesEvent(ev) {
			continue
		}
		return m, true
	}
	return Mapping{}, false
}

// Action is a convenience wrapper around Match that returns only the action
// name.
func (t MappingTable) Action(device string, ev Event) (string, bool) {
	if m, ok := t.Match(device, ev); ok {
		return m.Action, true
	}
	return "", false
}

func (m Mapping) matchesEvent(ev Event) bool {
	msg := ev.Msg
	switch m.NormalizedType() {
	case MappingNote:
		var ch, note, vel uint8
		if !msg.GetNoteStart(&ch, &note, &vel) {
			return false
		}
		if vel == 0 {
			return false // note-on with velocity 0 is a note-off
		}
		if m.Channel != 0 && int(ch)+1 != m.Channel {
			return false
		}
		if int(note) != m.Number {
			return false
		}
		if m.Value != nil && int(vel) != *m.Value {
			return false
		}
		return true
	case MappingCC:
		var ch, cc, val uint8
		if !msg.GetControlChange(&ch, &cc, &val) {
			return false
		}
		if m.Channel != 0 && int(ch)+1 != m.Channel {
			return false
		}
		if int(cc) != m.Number {
			return false
		}
		if m.Value != nil && int(val) != *m.Value {
			return false
		}
		return true
	}
	return false
}
