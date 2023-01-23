// Copyright 2023, Jason S. McMullan <jason.mcmullan@gmail.com>

package main

import (
	"fmt"
	"math"
	"os"
	"time"

	fisnar "github.com/ezrec/fisnar"
	dxf_document "github.com/rpaloschi/dxf-go/document"
	dxf_entities "github.com/rpaloschi/dxf-go/entities"
	ms3 "github.com/soypat/glgl/math/ms3"
	flag "github.com/spf13/pflag"
)

type mainOptions struct {
	Dispense  bool
	Port      string
	Speed     float32
	DotTimeMs uint
	Offset    []float32
	ZHop      float32
	Scale     float32
}

func circlePath(circle *dxf_entities.Circle) (path []ms3.Vec) {
	center := ms3.Vec{X: float32(circle.Center.X),
		Y: float32(circle.Center.Y),
		Z: float32(circle.Center.Z),
	}

	normal := ms3.Vec{X: float32(circle.ExtrusionDirection.X),
		Y: float32(circle.ExtrusionDirection.Y),
		Z: float32(circle.ExtrusionDirection.Z),
	}

	subdivisions := 128
	angle := 2.0 * math.Pi / float32(subdivisions)

	for n := 0; n < (subdivisions + 1); n++ {
		rot := ms3.QuatRotate(angle*float32(n), normal)
		vec := rot.Rotate(ms3.Vec{X: float32(circle.Radius), Y: 0, Z: 0})
		path = append(path, ms3.Add(vec, center))
	}

	return
}

func VecEqual(a, b ms3.Vec) bool {
	err := 0.01
	return (math.Abs(float64(a.X-b.X)) < err) &&
		(math.Abs(float64(a.Y-b.Y)) < err) &&
		(math.Abs(float64(a.Z-b.Z)) < err)
}

func execute(options *mainOptions, dxfFilename string) (err error) {
	file, err := os.Open(dxfFilename)
	if err != nil {
		return
	}
	defer file.Close()

	doc, err := dxf_document.DxfDocumentFromStream(file)
	if err != nil {
		return
	}

	offset := ms3.Vec{X: options.Offset[0], Y: options.Offset[1], Z: options.Offset[2]}

	paths := []([]ms3.Vec){}

	for _, entity := range doc.Entities.Entities {
		path := []ms3.Vec{}

		if polyline, ok := entity.(*dxf_entities.Polyline); ok {
			for _, v := range polyline.Vertices {
				path = append(path, ms3.Vec{X: float32(v.Location.X), Y: float32(v.Location.Y), Z: float32(v.Location.Z)})
			}
		} else if line, ok := entity.(*dxf_entities.Line); ok {
			path = append(path, ms3.Vec{X: float32(line.Start.X), Y: float32(line.Start.Y), Z: float32(line.Start.Z)},
				ms3.Vec{X: float32(line.End.X), Y: float32(line.End.Y), Z: float32(line.End.Z)})
		} else if circle, ok := entity.(*dxf_entities.Circle); ok {
			path = append(path, circlePath(circle)...)
		} else {
			fmt.Printf("%+v\n", entity)
		}

		if len(path) == 0 {
			continue
		}

		if len(paths) > 1 {
			k := len(paths) - 1
			n := len(paths[k]) - 1
			if VecEqual(paths[k][n], path[0]) {
				paths[k] = append(paths[k], path[1:]...)
			} else {
				paths = append(paths, path)
			}
		} else {
			paths = append(paths, path)
		}
	}

	if len(paths) == 0 {
		return
	}

	// Add scaling then offset to paths
	for _, path := range paths {
		for k, vec := range path {
			scaled := ms3.Scale(options.Scale, vec)
			path[k] = ms3.Add(scaled, offset)
		}
	}

	machine, err := fisnar.OpenF4200N(options.Port)
	if err != nil {
		return
	}
	defer machine.Close()

	err = machine.Home()
	if err != nil {
		return
	}

	err = machine.SetSpeed(options.Speed)
	if err != nil {
		return
	}

	for _, path := range paths {
		fmt.Printf("Move: %v (\n", path[0])
		// Hop by Z
		machine.MoveTo(path[0].X, path[0].Y, path[0].Z-options.ZHop)
		machine.WaitFor()
		machine.MoveTo(path[0].X, path[0].Y, path[0].Z)
		if options.Dispense {
			machine.SetDispenser(true)
		}
		if len(path) == 1 {
			fmt.Printf("  Dot\n")
			// Dispense dot.
			time.Sleep(time.Duration(options.DotTimeMs) * time.Millisecond)
		} else {
			for _, vec := range path {
				fmt.Printf("  .. %v\n", vec)
				if options.Dispense {
					machine.LineTo(vec.X, vec.Y, vec.Z)
				} else {
					machine.MoveTo(vec.X, vec.Y, vec.Z)
				}
			}
			machine.WaitFor()
		}
		if options.Dispense {
			machine.SetDispenser(false)
		}
		// Hop by Z
		last := path[len(path)-1]
		machine.MoveTo(last.X, last.Y, last.Z-options.ZHop)
		fmt.Printf(")\n")
	}

	machine.WaitFor()

	return
}

func main() {
	options := mainOptions{
		Offset: []float32{},
	}

	flag.BoolVar(&options.Dispense, "dispense", false, "Dispense")
	flag.UintVar(&options.DotTimeMs, "dot-time-ms", 100, "Dot extrusion time")
	flag.Float32Var(&options.Speed, "speed", 10, "Speed, in mm/second")
	flag.StringVar(&options.Port, "port", "/dev/ttyUSB0", "Serial port")
	flag.Float32SliceVar(&options.Offset, "offset", []float32{0.0, 0.0, 0.0}, "X,Y,Z offset of work plane, in mm")
	flag.Float32Var(&options.ZHop, "z-hop", 0.0, "Hop in Z when moving between dispense lines")
	flag.Float32Var(&options.Scale, "scale", 1.0, "Scale output by this value")
	flag.SetInterspersed(true)
	flag.Parse()

	if len(flag.Args()) != 1 {
		fmt.Fprintf(os.Stderr, "Usage:\n    %s [options] <filename.dxf>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	for len(options.Offset) < 3 {
		options.Offset = append(options.Offset, 0.0)
	}

	err := execute(&options, flag.Args()[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
