// Program piocli can compile PIO sequences.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"zappem.net/pub/io/pious"
)

var (
	debug  = flag.Bool("debug", false, "use to output debugging info")
	name   = flag.String("name", "", "name output program")
	src    = flag.String("src", "", "comma separated path(s) to .pio source file(s)")
	tinygo = flag.Bool("tinygo", false, "output program as a tinygo compatible package of name --name")
)

func main() {
	flag.Parse()

	if *src == "" {
		log.Fatalf("%s --src=<program.pio>[,...] required argument", os.Args[0])
	}

	var ps []*pious.Program
	for _, f := range strings.Split(*src, ",") {
		text, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("%s failed to read %q: %v", os.Args[0], f, err)
		}
		p, err := pious.NewProgram(string(text))
		if err != nil {
			log.Fatalf("%s failed to assemble %q: %v", os.Args[0], *src, err)
		}
		ps = append(ps, p)
	}

	var p *pious.Program
	title := *name

	if len(ps) == 1 {
		p = ps[0]
		if title != "" {
			p.Attr.Name = title
		}
	} else {
		if title == "" {
			title = "combined"
		}
		var err error
		p, err = pious.Cat(title, ps...)
		if err != nil {
			log.Fatalf("cat of pio files failed: %v", err)
		}
	}
	if *debug {
		log.Printf("compiled: %#v", p)
	}
	if *tinygo {
		fmt.Print(strings.Join(p.MakePackage(fmt.Sprint("From sources: ", *src)), "\n"))
	} else {
		// TODO when using pious.Cat() with different .side_set values
		// the disassembler fails to reproduce the code. Need to warn
		// about this.
		for _, line := range p.Disassemble() {
			fmt.Printf("%s\n", line)
		}
	}
}
