// Copyright 2023, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"fmt"
	"os"

	fisnar "github.com/ezrec/fisnar"
	flag "github.com/spf13/pflag"
)

type mainOptions struct {
	Port  string
	Speed float32
}

func execute(options *mainOptions) (err error) {
	var machine interface {
		fisnar.Closer
		fisnar.Ioporter
		fisnar.Mover
		fisnar.Dispenser
	}

	machine, err = fisnar.OpenF4200N(options.Port)
	if err != nil {
		return
	}
	defer machine.Close()

	err = machine.Home()
	if err != nil {
		return
	}

	for i := 8; i < 32; i++ {
		fmt.Printf("OUT %d,1\n", i)
		err = machine.Out(i, true)
		if err != nil {
			return
		}
	}

	return
}

func main() {
	var options mainOptions

	flag.Float32Var(&options.Speed, "speed", 10, "Speed, in mm/second")
	flag.StringVar(&options.Port, "port", "/dev/ttyUSB0", "Serial port")
	flag.SetInterspersed(true)
	flag.Parse()

	err := execute(&options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
