// Copyright 2023, Jason S. McMullan <jason.mcmullan@gmail.com>

package fisnar

import (
	"fmt"
	"io"
	"strings"
	"time"

	"go.bug.st/serial"
)

const (
	F4200N_DISPENSER_PORT = 12
)

type F4200N struct {
	Stream io.ReadWriteCloser
}

func OpenF4200N(port string) (machine *F4200N, err error) {
	mode := serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	stream, err := serial.Open(port, &mode)
	if err != nil {
		return
	}

	// FISNAR must respond to any command within 5s.
	stream.SetReadTimeout(time.Duration(10) * time.Second)

	machine = &F4200N{
		Stream: stream,
	}

	// Issue reset
	machine.Stream.Write([]byte{0xdf})
	reply, err := machine.awaitReply()
	if err != nil {
		stream.Close()
		machine = nil
		return
	}

	if !strings.HasPrefix(reply, "<<") || !strings.HasSuffix(reply, ">>") {
		err = fmt.Errorf("Invalid identification string '%v', expected '<< ... >>'", reply)
		stream.Close()
		machine = nil
		return
	}

	return
}

func (f *F4200N) Close() error {
	return f.Stream.Close()
}

func (f *F4200N) emitCommand(format string, a ...any) (err error) {
	out := fmt.Sprintf(format, a...)
	out += "\r"
	_, err = f.Stream.Write([]byte(out))
	if err != nil {
		return
	}

	return
}

func (f *F4200N) awaitReply() (line string, err error) {
	buf := []byte{0}
	out := []byte{}

	for {
		n, err := f.Stream.Read(buf)
		if err != nil || n != 1 {
			break
		}

		if buf[0] == byte('\r') {
			continue
		}

		if buf[0] == byte('\n') {
			break
		}

		out = append(out, buf[0])
	}

	line = string(out)

	return
}

func (f *F4200N) sendCommand(format string, a ...any) (err error) {
	err = f.emitCommand(format, a...)
	if err != nil {
		return
	}
	ack, err := f.awaitReply()
	if err != nil {
		return
	}
	if ack != "ok" {
		err = fmt.Errorf("Unexpected response, expected 'ok', got: '%v'", ack)
	}
	return
}

func (f *F4200N) sendCommandWithReply(format string, a ...any) (reply string, err error) {
	err = f.emitCommand(format, a...)
	if err != nil {
		return
	}
	reply, err = f.awaitReply()
	if err != nil {
		return
	}
	ack, err := f.awaitReply()
	if err != nil {
		return
	}
	if ack != "ok" {
		err = fmt.Errorf("Unexpected response, expected 'ok', got: '%v'", ack)
	}
	return
}

func (f *F4200N) Halt() error {
	return f.sendCommand("ST")
}

func (f *F4200N) Home() error {
	return f.sendCommand("HM")
}

func (f *F4200N) MoveTo(x, y, z float32) error {
	return f.sendCommand("MA %.3f,%.3f,%.3f", x, y, z)
}

func (f *F4200N) LineTo(x, y, z float32) error {
	return f.sendCommand("LA %.3f,%.3f,%.3f", x, y, z)
}

func (f *F4200N) SetSpeed(speed float32) error {
	return f.sendCommand("SP %d", int(speed))
}

func (f *F4200N) SetDispenser(enabled bool) error {
	return f.Output(F4200N_DISPENSER_PORT, enabled)
}

func (f *F4200N) WaitFor() error {
	return f.sendCommand("ID")
}

func (f *F4200N) Output(port int, enabled bool) error {
	var value int
	if enabled {
		value = 1
	}
	return f.sendCommand("OUT %d,%d", port, value)
}

func (f *F4200N) Input(port int) (enabled bool, err error) {
	reply, err := f.sendCommandWithReply("IN %d")
	if err != nil {
		return
	}
	var value int

	_, err = fmt.Sscanf(reply, "%d", &value)
	if err != nil {
		return
	}

	if value != 0 {
		enabled = true
	}

	return
}

func (f *F4200N) Position() (x, y, z float32, err error) {
	reply, err := f.sendCommandWithReply("PA")
	if err != nil {
		return
	}

	_, err = fmt.Sscanf(reply, "%f,%f,%f", &x, &y, &z)
	if err != nil {
		return
	}

	return
}
