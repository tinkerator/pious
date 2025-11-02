// The pious package provides functions to assemble and disassemble
// RP2350 PIO code. This package was written after reading the PIO
// details in the [RP2350 Datasheet].
//
// [RP2350 Datasheet]: https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf
package pious

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Disassemble disassembles a PIO instruction.
func Disassemble(instr uint16, p *Program) (string, error) {
	var dec Instruction
	var cmd int
	var decoded []string
	for cmd, dec = range instructions {
		if dec.mask&instr == dec.bits {
			decoded = append(decoded, fmt.Sprint(dec.token, "\t"))
			break
		}
	}
	if len(decoded) == 0 {
		return fmt.Sprintf("unknown <%04x>", instr), ErrBad
	}

	if dec.flags&flagCondition != 0 {
		offset := 0b111 & (instr >> 5)
		if offset != 0 {
			decoded = append(decoded, disCondition[offset]+" ")
		}
	}
	if dec.flags&flagAddress != 0 {
		addr := uint16(instr & 0b11111)
		noSym := true
		if p != nil {
			if sym, ok := p.Targets[addr]; ok {
				decoded = append(decoded, sym[0])
				noSym = false
			}
		}
		if noSym {
			decoded = append(decoded, fmt.Sprint(addr))
		}
	}
	if dec.flags&flagPolSource != 0 {
		poll := (instr >> 5) & 0b111
		index := instr & 0b11111
		src := poll & 0b11
		decoded = append(decoded, fmt.Sprint(poll>>2, " "), fmt.Sprint(disBitSource[src], " "))
		switch src {
		case 0b00, 0b01:
			decoded = append(decoded, fmt.Sprint(index))
		case 0b10:
			idxmode := index >> 3
			index = index & 0b111
			switch idxmode {
			case 0b00:
				decoded = append(decoded, fmt.Sprint(index))
			case 0b01:
				decoded = append(decoded, fmt.Sprint("prev ", index))
			case 0b10:
				decoded = append(decoded, fmt.Sprint(index, " rel"))
			case 0b11:
				decoded = append(decoded, fmt.Sprint("next ", index))
			}
		case 0b11:
			if index&0b11100 != 0 {
				return fmt.Sprintf("unknown <%04x>", instr), ErrBad
			}
			decoded = append(decoded, fmt.Sprint("+ ", index))
		}
	} else if dec.flags&flagWIndex != 0 {
		// without flagPolSource?
		return fmt.Sprintf("unknown <%04x>", instr), ErrBad
	}
	if dec.flags&flagISource != 0 {
		src := (instr >> 5) & 0b111
		tok := disISources[src]
		if tok == "" {
			return fmt.Sprintf("unknown <%04x>", instr), ErrBad
		}
		decoded = append(decoded, tok+" ")
	}

	if dec.flags&flagIfF != 0 {
		if instr&(1<<6) != 0 {
			decoded = append(decoded, "iffull ")
		}
	}
	if dec.flags&flagIfE != 0 {
		if instr&(1<<6) != 0 {
			decoded = append(decoded, "ifempty ")
		}
	}
	if dec.flags&flagBlk != 0 {
		if instr&(1<<5) != 0 {
			decoded = append(decoded, "block")
		} else {
			decoded = append(decoded, "noblock")
		}
	}

	if dec.flags&flagDestination != 0 {
		dest := (instr >> 5) & 0b111
		if cmd == idxSET && (dest == 0b011 || dest >= 0b101) {
			return "invalid destination", ErrBad
		}
		decoded = append(decoded, fmt.Sprintf("%s, ", disDestinations[dest]))
	}
	if dec.flags&flagMDestination != 0 {
		dest := (instr >> 5) & 0b111
		decoded = append(decoded, fmt.Sprintf("%s, ", disMDestinations[dest]))
	}
	if dec.flags&flagBitCount != 0 {
		bc := instr & 0b11111
		if bc == 0 {
			bc = 32
		}
		decoded = append(decoded, fmt.Sprint(bc))
	}
	if dec.flags&flagOp != 0 {
		op := (instr >> 3) & 0b11
		switch op {
		case 0b11:
			return fmt.Sprintf("invalid <%04x>", instr), ErrBad
		case 0b10:
			decoded = append(decoded, "::")
		case 0b01:
			decoded = append(decoded, "!")
		}
	}
	if dec.flags&flagMSource != 0 {
		src := instr & 0b111
		if src == 0b100 {
			return fmt.Sprintf("invalid <%04x>", instr), ErrBad
		}
		decoded = append(decoded, fmt.Sprintf("%s", disMSources[src]))
	}
	if dec.flags&flagData != 0 {
		decoded = append(decoded, fmt.Sprint(instr&0b11111))
	}
	if dec.flags&flagFromXIdxlIndex != 0 {
		if instr&(1<<7) != 0 {
			// from rxfifo
			if instr&(1<<3) != 0 {
				decoded = append(decoded, fmt.Sprintf("osr, rxfifo[%d]", instr&0b11))
			} else {
				if instr&0b111 != 0 {
					return fmt.Sprintf("invalid <%04x>", instr), ErrBad
				}
				decoded = append(decoded, "osr, rxfifo[y]")
			}
		} else {
			// to rxfifo
			if instr&(1<<3) != 0 {
				decoded = append(decoded, fmt.Sprintf("rxfifo[%d], isr", instr&0b11))
			} else {
				if instr&0b111 != 0 {
					return fmt.Sprintf("invalid <%04x>", instr), ErrBad
				}
				decoded = append(decoded, "rxfifo[y], isr")
			}
		}
	}
	if dec.flags&flagClrWaitIdxModeIndex != 0 {
		but := ""
		if clr := instr & (1 << 6); clr != 0 {
			but = "clear "
		} else if wait := instr & (1 << 5); wait != 0 {
			but = "wait "
		}
		idxmode := (instr >> 3) & 0b11
		index := instr & 0b111
		switch idxmode {
		case 0b00:
			decoded = append(decoded, fmt.Sprint(but, index))
		case 0b01:
			decoded = append(decoded, fmt.Sprint("prev ", but, index))
		case 0b10:
			decoded = append(decoded, fmt.Sprint(but, index, " rel"))
		case 0b11:
			decoded = append(decoded, fmt.Sprint("next ", but, index))
		}
	}

	sideMask := uint16(0b11111)
	if p != nil && p.Attr.SideSet != 0 {
		if p.Attr.SideSetOpt {
			side := (instr & 0b0111100000000) >> (8 + 4 - p.Attr.SideSet)
			if (instr & 0b1000000000000) != 0 {
				decoded = append(decoded, fmt.Sprintf("\tside %d", side))
			} else if side != 0 {
				return fmt.Sprintf("invalid opt side-set <%04x>", instr), ErrBad
			}
			sideMask = sideMask >> 1
		} else {
			side := (instr & 0b1111100000000) >> (8 + 5 - p.Attr.SideSet)
			decoded = append(decoded, fmt.Sprintf("\tside %d", side))
		}
		sideMask = sideMask >> p.Attr.SideSet
	}
	if delay := (instr >> 8) & sideMask; delay != 0 {
		decoded = append(decoded, fmt.Sprintf(" [%d]", delay))
	}
	return strings.Join(decoded, ""), nil
}

// parseConst returns a positive integer less than or equal to 32,
// either indirectly via the consts lookup, or because the supplied
// token is an integer.
func parseConst(token string, consts map[string]uint16) (uint16, error) {
	if consts != nil {
		if n, ok := consts[token]; ok {
			return n, nil
		}
	}
	n, err := strconv.Atoi(token)
	if n > 32 || n < 0 {
		return 0, ErrBad
	}
	return uint16(n), err
}

var tokenizer = regexp.MustCompile("([, \r\t]+|//.*|;.*)")

// Assemble converts a string of assembly code into its uint16
// representation. The parsing is more relaxed than the official
// syntax.
func Assemble(code string, p *Program) (uint16, error) {
	tokens := tokenizer.Split(code, -1)
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "" {
			tokens = append(tokens[:i], tokens[i+1:]...)
		}
	}
	if len(tokens) == 0 {
		return 0, ErrEmpty
	}
	var labels map[string]uint16
	if p != nil {
		labels = p.Labels
	}
	for i, dec := range instructions {
		if tokens[0] != dec.token {
			continue
		}
		instr := dec.bits
		if dec.flags == 0 && len(tokens) == 1 {
			return instr, nil
		}
		if len(tokens) == 1 {
			return 0, ErrBad
		}
		k := 1
		switch i {
		case idxJMP:
			for j, op := range disCondition {
				if op == tokens[k] {
					instr = instr | uint16(j<<5)
					k++
					break
				}
			}
			n, err := parseConst(tokens[k], labels)
			if err != nil {
				return 0, err
			}
			if n == 32 {
				return 0, ErrBad
			}
			instr = instr | uint16(n)
			k++
		case idxWAIT:
			if len(tokens) < 3 {
				return 0, ErrBad
			}
			if n, err := parseConst(tokens[k], nil); err == nil {
				if n > 1 {
					return 0, ErrBad
				}
				instr = instr | uint16(n<<7)
				k++
			}
			if k >= len(tokens) {
				return 0, ErrBad
			}
			found := false
			src := 0
			for i, bits := range disBitSource {
				if bits == tokens[k] {
					src = i
					k++
					found = true
					break
				}
			}
			if !found || k >= len(tokens) {
				return 0, ErrBad
			}
			instr = instr | uint16(src<<5)
			switch src {
			case 0b00, 0b01:
				n, err := parseConst(tokens[k], nil)
				if err != nil {
					return 0, err
				}
				if n > 31 {
					return 0, ErrBad
				}
				k++
				instr = instr | uint16(n)
			case 0b10:
				n, err := parseConst(tokens[k], nil)
				if err == nil {
					if n > 7 {
						return 0, ErrBad
					}
					k++
					instr = instr | uint16(n)
					if k < len(tokens) && "rel" == tokens[k] {
						instr = instr | 0b10000
						k++
					}
					break
				}
				switch tokens[k] {
				case "prev":
					instr = instr | 0b01000
				case "next":
					instr = instr | 0b11000
				default:
					return 0, ErrBad
				}
				k++
				n, err = parseConst(tokens[k], nil)
				if err != nil || n > 7 {
					return 0, ErrBad
				}
				instr = instr | uint16(n)
				k++
			case 0b11:
				if k+2 > len(tokens) || "+" != tokens[k] {
					return 0, ErrBad
				}
				n, err := parseConst(tokens[k+1], nil)
				if err != nil {
					return 0, err
				}
				if n > 3 {
					return 0, ErrBad
				}
				instr = instr | uint16(n)
				k += 2
			}
		case idxIN:
			if len(tokens) < 3 {
				return 0, ErrBad
			}
			for j, src := range disISources {
				if src == "" {
					continue
				}
				if src == tokens[k] {
					instr = instr | uint16(j<<5)
					k++
					break
				}
			}
			if k != 2 {
				return 0, ErrBad
			}
			n, err := parseConst(tokens[k], labels)
			if err != nil {
				return 0, err
			}
			if n == 0 {
				return 0, ErrBad
			}
			instr = instr | uint16(n&0b11111)
			k++
		case idxOUT:
			if len(tokens) < 3 {
				return 0, ErrBad
			}
			for j, src := range disDestinations {
				if src == tokens[k] {
					instr = instr | uint16(j<<5)
					k++
					break
				}
			}
			if k != 2 {
				return 0, ErrBad
			}
			n, err := parseConst(tokens[k], labels)
			if err != nil {
				return 0, err
			}
			if n == 0 {
				return 0, ErrBad
			}
			instr = instr | uint16(n&0b11111)
			k++
		case idxNOP:
		case idxPULL, idxPUSH:
			block := uint16(0b100000)
			if k < len(tokens) {
				if (idxPUSH == i && "iffull" == tokens[k]) || (idxPULL == i && "ifempty" == tokens[k]) {
					instr = instr | 0b1000000
					k++
				}
			}
			if k < len(tokens) {
				switch tokens[k] {
				case "noblock":
					block = 0
					k++
				case "block":
					k++
				}
			}
			instr = instr | block
		case idxMOV1:
			if len(tokens) < 3 {
				return 0, ErrBad
			}
			var fifo, detail string
			if strings.HasPrefix(tokens[k], "rxfifo[") {
				fifo = tokens[k]
				if detail = tokens[k+1]; detail != "isr" {
					return 0, ErrBad
				}
			} else if strings.HasPrefix(tokens[k+1], "rxfifo[") {
				fifo = tokens[k+1]
				if detail = tokens[k]; detail != "osr" {
					return 0, ErrBad
				}
				instr = instr | (1 << 7)
			} else {
				continue
			}
			k += 2
			if fifo[len(fifo)-1] != ']' {
				return 0, ErrBad
			}
			offset := fifo[7 : len(fifo)-1]
			if offset != "y" {
				n, err := parseConst(offset, nil)
				if err != nil || n > 7 {
					return 0, ErrBad
				}
				instr = instr | (1 << 3) | uint16(n)
			}
		case idxMOV2:
			if len(tokens) < 3 {
				return 0, ErrBad
			}
			found := false
			for i, dest := range disMDestinations {
				if dest == tokens[k] {
					instr = instr | uint16(i<<5)
					found = true
					k++
					break
				}
			}
			if !found {
				continue
			}
			var src string
			if tok := tokens[k]; strings.HasPrefix(tok, "!") {
				instr = instr | (0b01 << 3)
				src = tok[1:]
				k++
			} else if strings.HasPrefix(tok, "::") {
				instr = instr | (0b10 << 3)
				src = tok[2:]
				k++
			}
			if src == "" {
				if k >= len(tokens) {
					return 0, ErrBad
				}
				src = tokens[k]
				k++
			}
			found = false
			for i, from := range disMSources {
				if from == src {
					instr = instr | uint16(i)
					found = true
					break
				}
			}
		case idxSET:
			if len(tokens) < 3 {
				return 0, ErrBad
			}
			found := false
			for i, dest := range disDestinations {
				if dest == tokens[k] {
					instr = instr | uint16(i<<5)
					k++
					found = true
					if p != nil && i == 0 /* pins */ && p.Attr.Set == 0 {
						p.Attr.Set = 1
					}
					break
				}
			}
			if !found || k >= len(tokens) {
				return 0, ErrBad
			}
			n, err := parseConst(tokens[k], labels)
			if err != nil {
				return 0, err
			}
			k++
			instr = instr | uint16(n)
		case idxIRQ:
			if len(tokens) < 2 {
				return 0, ErrBad
			}
			idxMode := 0
			switch tokens[1] {
			case "prev":
				idxMode = 0b01
				k++
			case "next":
				idxMode = 0b11
				k++
			}
			if k >= len(tokens) {
				return 0, ErrBad
			}
			switch tokens[k] {
			case "nowait", "set":
				k++
			case "clear":
				instr = instr | 0b1000000
				k++
			case "wait":
				instr = instr | 0b100000
				k++
			}
			if k >= len(tokens) {
				return 0, ErrBad
			}
			n, err := parseConst(tokens[k], nil)
			if err != nil {
				return 0, err
			}
			if n > 7 {
				return 0, ErrBad
			}
			instr = instr | uint16(n)
			k++
			if k < len(tokens) && "rel" == tokens[k] {
				if idxMode != 0 {
					return 0, ErrBad
				}
				idxMode = 0b10
				k++
			}
			instr = instr | uint16(idxMode<<3)
		default:
			return 0, ErrBad
		}

		var sideVal uint16
		sideMask := uint16(0b11111)
		if p != nil && p.Attr.SideSet > 0 {
			hasSide := k <= len(tokens)-2 && tokens[k] == "side"
			if hasSide {
				n, err := parseConst(tokens[k+1], nil)
				if err != nil {
					return 0, err
				}
				if limit := (uint16(1) << p.Attr.SideSet); n >= limit {
					return 0, fmt.Errorf("too large for side-set %d bits needed", p.Attr.SideSet)
				}
				if p.Attr.SideSetOpt {
					sideVal = 0b1000000000000 | (n << (8 + 4 - p.Attr.SideSet))
				} else {
					sideVal = n << (8 + 5 - p.Attr.SideSet)
				}
				k = k + 2
			} else if !p.Attr.SideSetOpt {
				return 0, fmt.Errorf("omitted non-optional side-set %d bits needed", p.Attr.SideSet)
			}
			if p.Attr.SideSetOpt {
				sideMask = sideMask >> 1
			}
			sideMask = sideMask >> p.Attr.SideSet
		}
		// parse a delay value
		if k != len(tokens) {
			if delay := tokens[k]; len(delay) >= 3 && delay[0] == '[' && delay[len(delay)-1] == ']' {
				n, err := parseConst(delay[1:len(delay)-1], nil)
				if err != nil {
					return 0, err
				}
				if n&sideMask != n {
					return 0, ErrBad
				}
				instr = instr | sideVal | uint16(n<<8)
				k++
			}
		} else {
			instr = instr | sideVal
		}
		if k != 1 {
			return instr, nil
		}
	}
	return 0, ErrBad
}

// buildTargets computes the inverse label map for a program.
func (p *Program) buildTargets() {
	targets := make(map[uint16][]string)
	for label, addr := range p.Labels {
		targets[addr] = append(targets[addr], label)
	}
	// Sorted order.
	for addr, names := range targets {
		sort.Strings(names)
		targets[addr] = names
	}
	p.Targets = targets
}

// NewProgram compiles a PIO program from source. The source format is
// intended to be compatible with that described in the [RP2350
// Datasheet].
func NewProgram(source string) (*Program, error) {
	lines := strings.Split(source, "\n")
	var code []uint16
	var program string
	wrap := uint16(0xffff)
	wrapTarget := uint16(0xffff)
	p := &Program{
		Labels: make(map[string]uint16),
	}
	for i, line := range lines {
		instr, err := Assemble(line, p)
		if err == nil {
			code = append(code, instr)
			continue
		}
		// not a known instruction, so interpret it as
		// something else.
		tokens := tokenizer.Split(line, -1)
		for i := 0; i < len(tokens); i++ {
			if tokens[i] == "" {
				tokens = append(tokens[:i], tokens[i+1:]...)
			}
		}
		if len(tokens) == 0 {
			continue
		}
		switch tokens[0] {
		case ".program":
			if len(tokens) != 2 {
				return nil, fmt.Errorf("failed to parse line %d: %q", i, line)
			}
			p.Attr.Name = tokens[1]
		case ".wrap":
			if len(tokens) != 1 || wrap != uint16(0xffff) {
				return nil, fmt.Errorf("bad wrap line %d: %q", i, line)
			}
			wrap = uint16(len(code)) - 1
		case ".wrap_target":
			if len(tokens) != 1 || wrapTarget != uint16(0xffff) {
				return nil, fmt.Errorf("bad wrap line %d: %q", i, line)
			}
			wrapTarget = uint16(len(code))
		case ".origin":
			if len(tokens) != 1 {
				return nil, fmt.Errorf("syntax error for .origin at line %d: %q", i, line)
			}
			p.Attr.Origin = uint16(len(code))
		case ".side_set":
			if len(tokens) < 2 || len(code) != 0 {
				return nil, fmt.Errorf("too late to set side_set line %d: %q", i, line)
			}
			p.Attr.SideSet, err = parseConst(tokens[1], nil)
			if err != nil {
				return nil, fmt.Errorf("bad side_set value line %d: %q: %v", i, line, err)
			}
			k := 2
			if len(tokens) > k && tokens[k] == "opt" {
				p.Attr.SideSetOpt = true
				if p.Attr.SideSet > 4 {
					return nil, fmt.Errorf("max optional side_set value is 4, got %d at line %d: %q", p.Attr.SideSet, i, line)
				}
				k++
			} else if p.Attr.SideSet > 5 {
				return nil, fmt.Errorf("max side_set value is 5, got %d at line %d: %q", p.Attr.SideSet, i, line)
			}
			if len(tokens) == k {
				break
			}
			if tokens[k] != "pindirs" {
				return nil, fmt.Errorf("no pindirs at line %d: %q", i, line)
			}
			if len(tokens) > k+1 {
				return nil, fmt.Errorf("syntax error at line %d: %q", i, line)
			}
			p.Attr.SideSetPindirs = true
		case ".set":
			if len(tokens) != 2 || len(code) != 0 {
				return nil, fmt.Errorf("too late to set count line %d: %q", i, line)
			}
			p.Attr.Set, err = parseConst(tokens[1], nil)
			if err != nil {
				return nil, fmt.Errorf("bad set value line %d: %q: %v", i, line, err)
			}
			if p.Attr.Set > 5 {
				return nil, fmt.Errorf("max set value is 5, got %d at line %d: %q", p.Attr.Set, i, line)
			}
		case ".out":
			if len(code) != 0 {
				return nil, fmt.Errorf("too late to .out at line %d: %q", i, line)
			}
			if len(tokens) < 2 {
				return nil, fmt.Errorf(".out requires a pin value at line %d: %q", i, line)
			}
			p.Attr.Out, err = parseConst(tokens[1], nil)
			if err != nil || p.Attr.Out == 0 {
				return nil, fmt.Errorf(".out requires bit count > 0 and <= 32 at line %d: %q", i, line)
			}
			k := 2
			if len(tokens) > k {
				switch tokens[k] {
				case "left", "right":
					p.Attr.OutLeft = tokens[k] == "left"
					k++
				default:
				}
			}
			if len(tokens) == k {
				break
			}
			if tokens[k] != "auto" {
				return nil, fmt.Errorf("expecting \"auto\" at line %d: %q", i, line)
			}
			k++
			if k == len(tokens) {
				break
			}
			p.Attr.OutThreshold, err = parseConst(tokens[k], nil)
			if err != nil || p.Attr.OutThreshold == 0 {
				return nil, fmt.Errorf("expecting threshold in range (0,32] at line %d: %q", i, line)
			}
			if p.Attr.OutThreshold == 32 {
				p.Attr.OutThreshold = 0
			}
			k++
			if k != len(tokens) {
				return nil, fmt.Errorf(".out syntax error at line %d: %q", i, line)
			}
		case ".in":
			if len(code) != 0 {
				return nil, fmt.Errorf("too late to .in at line %d: %q", i, line)
			}
			if len(tokens) < 2 {
				return nil, fmt.Errorf(".in requires a pin value at line %d: %q", i, line)
			}
			p.Attr.In, err = parseConst(tokens[1], nil)
			if err != nil || p.Attr.In == 0 {
				return nil, fmt.Errorf(".in requires bit count > 0 and <= 32 at line %d: %q", i, line)
			}
			k := 2
			if len(tokens) > k {
				switch tokens[k] {
				case "left", "right":
					p.Attr.InLeft = tokens[k] == "left"
					k++
				default:
				}
			}
			if len(tokens) == k {
				break
			}
			if tokens[k] != "auto" {
				return nil, fmt.Errorf("expecting \"auto\" at line %d: %q", i, line)
			}
			k++
			if k == len(tokens) {
				break
			}
			p.Attr.InThreshold, err = parseConst(tokens[k], nil)
			if err != nil || p.Attr.InThreshold == 0 {
				return nil, fmt.Errorf("expecting threshold in range (0,32] at line %d: %q", i, line)
			}
			if p.Attr.InThreshold == 32 {
				p.Attr.InThreshold = 0
			}
			k++
			if k != len(tokens) {
				return nil, fmt.Errorf(".in syntax error at line %d: %q", i, line)
			}
		default:
			if len(tokens) == 0 || tokens[0] == "" {
				continue
			}
			if len(tokens) != 1 || !strings.HasSuffix(tokens[0], ":") {
				return nil, fmt.Errorf("unable to parse line %d: %q as %v", i, line, tokens)
			}
			label := tokens[0]
			label = label[:len(label)-1]
			if label == "" {
				return nil, fmt.Errorf("missing label line %d: %q", i, line)
			}
			if value, hit := p.Labels[label]; hit {
				return nil, fmt.Errorf("duplicate label %q declared at line %d of value %d", label, i, value)
			}
			p.Labels[label] = uint16(len(code))
		}
	}
	if program == "" {
		program = "unknown"
	}
	if wrap == uint16(0xffff) {
		wrap = uint16(len(code))
	}
	if wrapTarget == uint16(0xffff) {
		wrapTarget = 0
	}
	p.buildTargets()
	p.Attr.Wrap = wrap
	p.Attr.WrapTarget = wrapTarget
	p.Code = code
	return p, nil
}

// Disassemble disassembles a whole program, p, into a slice of string lines.
func (p *Program) Disassemble() []string {
	listing := []string{
		fmt.Sprint(".program ", p.Attr.Name),
	}
	if p.Attr.In != 0 {
		var suffix string
		if p.Attr.InThreshold != 0 {
			suffix = fmt.Sprint(" auto ", p.Attr.InThreshold)
		}
		if p.Attr.InLeft {
			listing = append(listing, fmt.Sprintf(".in %d left%s", p.Attr.In, suffix))
		} else {
			listing = append(listing, fmt.Sprintf(".in %d right%s", p.Attr.In, suffix))
		}
	}
	if p.Attr.Out != 0 {
		var suffix string
		if p.Attr.OutThreshold != 0 {
			suffix = fmt.Sprint(" auto ", p.Attr.OutThreshold)
		}
		if p.Attr.OutLeft {
			listing = append(listing, fmt.Sprintf(".out %d left%s", p.Attr.Out, suffix))
		} else {
			listing = append(listing, fmt.Sprintf(".out %d right%s", p.Attr.Out, suffix))
		}
	}
	if p.Attr.SideSet != 0 {
		var parts []string
		if p.Attr.SideSetOpt {
			parts = append(parts, " opt")
		}
		if p.Attr.SideSetPindirs {
			parts = append(parts, " pindirs")
		}
		listing = append(listing, fmt.Sprint(".side_set ", p.Attr.SideSet, strings.Join(parts, "")))
	}
	if p.Attr.Set != 0 {
		listing = append(listing, fmt.Sprint(".set ", p.Attr.Set))
	}
	for i, code := range p.Code {
		if uint16(i) == p.Attr.WrapTarget {
			listing = append(listing, ".wrap_target")
		}
		if uint16(i) == p.Attr.Origin && p.Attr.Origin != 0 {
			listing = append(listing, ".origin")
		}
		if list, ok := p.Targets[uint16(i)]; ok {
			for _, sym := range list {
				listing = append(listing, fmt.Sprintf("%s:", sym))
			}
		}
		text, err := Disassemble(code, p)
		if err != nil {
			panic(fmt.Sprintf("error at code offset %d: %v", i, err))
		}
		listing = append(listing, fmt.Sprintf("\t%s", text))
		if uint16(i) == p.Attr.Wrap {
			listing = append(listing, ".wrap")
		}
	}
	if list, ok := p.Targets[uint16(len(p.Code))]; ok {
		for _, sym := range list {
			listing = append(listing, fmt.Sprintf("%s:", sym))
		}
	}
	if p.Attr.Wrap == uint16(len(p.Code)) {
		listing = append(listing, ".wrap")
	}
	return listing
}

// jumpCodeAdjust recognizes that a code is a jump code and applies a
// delta and returns that this is a jump and the recoded version of
// the code.
func jumpCodeAdjust(code uint16, delta uint16) (recode uint16) {
	ins := instructions[idxJMP]
	if code&ins.mask != ins.bits {
		recode = code
		return
	}
	is := (code & 0b11111) + delta
	recode = (is & 0b11111) | (code & ^uint16(0b11111))
	return
}

// Cat merges together a number of programs to create a combination
// program with multiple entry and wrapping targets. The idea is that
// different state machines running within one of the PIO<N> units can
// perform different PIO tasks.
func Cat(name string, ps ...*Program) (*Program, error) {
	prog := &Program{
		Attr: Settings{
			Name: name,
		},
		Labels: make(map[string]uint16),
	}
	var offset uint16
	for i, p := range ps {
		attr := Settings{
			Name:           p.Attr.Name,
			Origin:         offset + p.Attr.Origin,
			Wrap:           offset + p.Attr.Wrap,
			WrapTarget:     offset + p.Attr.WrapTarget,
			SideSet:        p.Attr.SideSet,
			SideSetOpt:     p.Attr.SideSetOpt,
			SideSetPindirs: p.Attr.SideSetPindirs,
			Set:            p.Attr.Set,
			Out:            p.Attr.Out,
			OutLeft:        p.Attr.OutLeft,
			OutAuto:        p.Attr.OutAuto,
			OutThreshold:   p.Attr.OutThreshold,
			In:             p.Attr.In,
			InLeft:         p.Attr.InLeft,
			InAuto:         p.Attr.InAuto,
			InThreshold:    p.Attr.InThreshold,
		}
		prog.Labels[fmt.Sprint(p.Attr.Name, i, "_origin")] = offset + p.Attr.Origin
		prog.Labels[fmt.Sprint(p.Attr.Name, i, "_wrap")] = offset + p.Attr.Wrap
		prog.Labels[fmt.Sprint(p.Attr.Name, i, "_wrap_target")] = offset + p.Attr.WrapTarget
		for label, val := range p.Labels {
			prog.Labels[fmt.Sprint(p.Attr.Name, i, "_", label)] = offset + val
		}
		for _, c := range p.Code {
			prog.Code = append(prog.Code, jumpCodeAdjust(c, offset))
		}
		offset += uint16(len(p.Code))
		prog.Modules = append(prog.Modules, attr)
	}
	if len(prog.Code) > 32 {
		return nil, fmt.Errorf("combined code for %q too long: %d > 32", name, len(prog.Code))
	}
	prog.buildTargets()
	prog.Attr.Wrap = uint16(len(prog.Code))

	return prog, nil
}

var cCaseRE = regexp.MustCompile(`_[a-zA-Z]`)

// camelCase rewrites a symbol to be more Go friendly.
func camelCase(text string) string {
	return cCaseRE.ReplaceAllStringFunc(text, func(a string) string {
		return strings.ToUpper(a[1:])
	})
}
