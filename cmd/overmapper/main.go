// See LICENSE.txt for licensing information.

package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"

	"github.com/drbig/overmapper"
)

var (
	mapx  = flag.Int("mapx", 180, "MapX")
	mapy  = flag.Int("mapy", 180, "MapY")
	scale = flag.Int("scale", 2, "drawing scale")
	level = flag.Int("level", 10, "level to dump data for")
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\t%s [options] save_path output_name.png...\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      \tVersion: %s, http://github.com/drbig/overmapper\n\n", overmapper.VERSION)
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(1)
	}

	m, err := overmapper.NewMap(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting info:", err)
		os.Exit(2)
	}

	fmt.Println(m, "- drawing...")

	overmapper.Config.MapX = *mapx
	overmapper.Config.MapY = *mapy
	overmapper.Config.Scale = *scale
	overmapper.Config.Level = *level

	i, err := m.Draw()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error drawing:", err)
		os.Exit(2)
	}

	fmt.Println("Saving...")

	o, err := os.Create(flag.Arg(1))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening output file:", err)
		os.Exit(2)
	}

	if err := png.Encode(o, i); err != nil {
		fmt.Fprintln(os.Stderr, "Error saving image:", err)
		os.Exit(3)
	}
}
