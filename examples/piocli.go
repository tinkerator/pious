// Program piocli can compile PIO sequences.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"zappem.net/pub/io/pious"
)

var (
	src   = flag.String("src", "", "path to .pio source file")
	debug = flag.Bool("debug", false, "use to output debugging info")
)

func main() {
	flag.Parse()

	if *src == "" {
		log.Fatalf("%s --src=<program.pio> required argument", os.Args[0])
	}
	text, err := os.ReadFile(*src)
	if err != nil {
		log.Fatalf("%s failed to read %q", os.Args[0], err)
	}
	p, err := pious.NewProgram(string(text))
	if err != nil {
		log.Fatalf("%s failed to assemble %q: %v", os.Args[0], *src, err)
	}
	if *debug {
		log.Printf("compiled: %#v", p)
	}
	for _, line := range p.Disassemble() {
		fmt.Printf("%s\n", line)
	}
}
