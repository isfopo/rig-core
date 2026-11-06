package midi

import (
	"fmt"

	"gitlab.com/gomidi/midi/v2/drivers"
	"gitlab.com/gomidi/midi/v2/drivers/rtmididrv"
)

// OpenVirtualIn creates a named virtual MIDI input port on the RtMidi
// backend. It is intended for apps that want to appear as a named MIDI
// source/sink in the OS MIDI bus (and, under PipeWire, as a named port).
// Non-RtMidi engines (such as the test engine) do not support it.
func (e *Engine) OpenVirtualIn(name string) (drivers.In, error) {
	rd, ok := e.drv.(*rtmididrv.Driver)
	if !ok {
		return nil, fmt.Errorf("midi: virtual ports require the rtmidi driver")
	}
	return rd.OpenVirtualIn(name)
}

// OpenVirtualOut creates a named virtual MIDI output port on the RtMidi
// backend.
func (e *Engine) OpenVirtualOut(name string) (drivers.Out, error) {
	rd, ok := e.drv.(*rtmididrv.Driver)
	if !ok {
		return nil, fmt.Errorf("midi: virtual ports require the rtmidi driver")
	}
	return rd.OpenVirtualOut(name)
}
