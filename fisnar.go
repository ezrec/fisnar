// Copyright 2023, Jason S. McMullan <jason.mcmullan@gmail.com>

package fisnar

type Closer interface {
	Close() error
}

type Mover interface {
	Halt() error
	Home() error
	MoveTo(x, y, z float32) error
	LineTo(x, y, z float32) error
	SetSpeed(speed float32) error
	WaitFor() error
	Position() (x, y, z float32, err error)
}

type Porter interface {
	Output(port int, enabled bool) error
	Input(port int) (enabled bool, err error)
}

type Dispenser interface {
	Porter
	SetDispenser(enabled bool) error
}
