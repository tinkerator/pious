// The pious package provides functions to assemble and disassemble
// RP2350 PIO code. This package was written after reading the PIO
// details in the [RP2350 Datasheet].
//
// [RP2350 Datasheet] https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf
package pious

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Disassemble disassembles a PIO instruction.
func Disassemble(instr uint16, symbols map[uint16]string) (string, error) {
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
		if symbols != nil {
			if sym, ok := symbols[addr]; ok {
				decoded = append(decoded, sym)
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

	if delay := (instr >> 8) & 0b11111; delay != 0 {
		// TODO handle side set before delay
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
func Assemble(code string, consts map[string]uint16) (uint16, error) {
	tokens := tokenizer.Split(code, -1)
	for i := 0; i < len(tokens); i++ {
		if tokens[i] == "" {
			tokens = append(tokens[:i], tokens[i+1:]...)
		}
	}
	if len(tokens) == 0 {
		return 0, ErrEmpty
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
			n, err := parseConst(tokens[k], consts)
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
			n, err := parseConst(tokens[k], consts)
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
			n, err := parseConst(tokens[k], consts)
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
					break
				}
			}
			if !found || k >= len(tokens) {
				return 0, ErrBad
			}
			n, err := parseConst(tokens[k], consts)
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

		// parse a delay value
		if k != len(tokens) {
			if delay := tokens[k]; len(delay) >= 3 && delay[0] == '[' && delay[len(delay)-1] == ']' {
				n, err := parseConst(delay[1:len(delay)-1], nil)
				if err != nil {
					return 0, err
				}
				if n == 32 {
					return 0, ErrBad
				}
				instr = instr | uint16(n<<8)
				k++
			}
		}
		if k != 1 {
			return instr, nil
		}
	}
	return 0, ErrBad
}

// NewProgram compiles a PIO program from source. The source format is
// intended to be compatible with that described in the [RP2350
// Datasheet].
func NewProgram(source string) (*Program, error) {
	lines := strings.Split(source, "\n")
	var code []uint16
	labels := make(map[string]uint16)
	var program string
	for i, line := range lines {
		instr, err := Assemble(line, labels)
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
				program = tokens[1]
			}
		default:
			if len(tokens) != 1 || !strings.HasSuffix(tokens[0], ":") {
				return nil, fmt.Errorf("failed to parse line %d: %q", i, line)
			}
			label := tokens[0]
			label = label[:len(label)-1]
			if label == "" {
				return nil, fmt.Errorf("missing label line %d: %q", i, line)
			}
			if value, hit := labels[label]; hit {
				return nil, fmt.Errorf("duplicate label %q declared at line %d of value %d", label, i, value)
			}
			labels[label] = uint16(len(code))
		}
	}
	if program == "" {
		program = "unknown"
	}
	return &Program{
		Name:   program,
		Labels: labels,
		Code:   code,
	}, nil
}

func (p *Program) Disassemble() []string {
	labels := make(map[uint16][]string)
	for label, addr := range p.Labels {
		labels[addr] = append(labels[addr], label)
	}
	targets := make(map[uint16]string)
	for addr, label := range labels {
		sort.Strings(label)
		targets[addr] = label[0]
	}
	var listing []string
	for i, code := range p.Code {
		if list, ok := labels[uint16(i)]; ok {
			for _, sym := range list {
				listing = append(listing, fmt.Sprintf("%s:", sym))
			}
		}
		text, err := Disassemble(code, targets)
		if err != nil {
			panic(fmt.Sprintf("error at code offset %d: %v", i, err))
		}
		listing = append(listing, fmt.Sprintf("\t%s", text))
	}
	return listing
}
