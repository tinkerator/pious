// This file was built from reading the PIO details in the following
// document:
//
//  https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf

package pious

import (
	"errors"
	"fmt"
	"strings"
)

type Flags uint

const (
	flagCondition Flags = 1 << iota
	flagAddress
	flagPolSource
	flagWIndex
	flagISource
	flagBitCount
	flagDestination
	flagIfF
	flagBlk
	flagFromXIdxlIndex
	flagIfE
	flagOp
	flagMSource
	flagClrWaitIdxModeIndex
	flagData
)

type Instruction struct {
	token      string
	mask, bits uint16
	flags      Flags
}

var instructions = []Instruction{
	{token: "jmp", mask: 0xe000, bits: 0x0000, flags: flagCondition | flagAddress},
	{token: "wait", mask: 0xe000, bits: 0x2000, flags: flagPolSource | flagWIndex},
	{token: "in", mask: 0xe000, bits: 0x4000, flags: flagISource | flagBitCount},
	{token: "out", mask: 0xe000, bits: 0x6000, flags: flagDestination | flagBitCount},
	{token: "push", mask: 0xe09f, bits: 0x8000, flags: flagIfF | flagBlk},
	{token: "nop", mask: 0xe0ff, bits: 0x8042, flags: 0},
	{token: "mov", mask: 0xe074, bits: 0x8010, flags: flagFromXIdxlIndex},
	{token: "pull", mask: 0xe09f, bits: 0x8080, flags: flagIfE | flagBlk},
	{token: "mov", mask: 0xe000, bits: 0xa000, flags: flagDestination | flagOp | flagMSource},
	{token: "irq", mask: 0xe080, bits: 0xc000, flags: flagClrWaitIdxModeIndex},
	{token: "set", mask: 0xe000, bits: 0xe000, flags: flagDestination | flagData},
}

// disCondition disassembles a jump condition.
var disCondition = []string{
	"",
	"!x",
	"x--",
	"!y",
	"y--",
	"x!=y",
	"pin",
	"!osre",
}

// disDestinations disassembles a destination type.
var disDestinations = []string{
	"pins",
	"x",
	"y",
	"null",
	"pindirs",
	"pc",
	"isr",
	"exec",
}

var ErrBad = errors.New("invalid instruction")

// Disassemble disassembles a PIO instruction.
func Disassemble(instr uint16) (string, error) {
	var dec Instruction
	var decoded []string
	for _, dec = range instructions {
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
		decoded = append(decoded, fmt.Sprint(instr&0b11111))
	}
	if dec.flags&flagPolSource != 0 {
		poll := (instr >> 5) & 0b111
		decoded = append(decoded, fmt.Sprint(poll>>2, " "))
		index := instr & 0b11111
		src := poll & 0b11
		switch src {
		case 0b00:
			decoded = append(decoded, fmt.Sprint("gpio ", index))
		case 0b01:
			decoded = append(decoded, fmt.Sprint("pin ", index))
		case 0b10:
			decoded = append(decoded, "irq ")
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
			decoded = append(decoded, fmt.Sprint("jmppin + ", index))
		}
	} else if dec.flags&flagWIndex != 0 {
		// without flagPolSource?
		return fmt.Sprintf("unknown <%04x>", instr), ErrBad
	}
	if dec.flags&flagISource != 0 {
		src := (instr >> 5) & 0b111
		switch src {
		case 0b000:
			decoded = append(decoded, "pins ")
		case 0b001:
			decoded = append(decoded, "x ")
		case 0b010:
			decoded = append(decoded, "y ")
		case 0b011:
			decoded = append(decoded, "null ")
		case 0b100, 0b101:
			return fmt.Sprintf("unknown <%04x>", instr), ErrBad
		case 0b110:
			decoded = append(decoded, "isr ")
		case 0b111:
			decoded = append(decoded, "osr ")
		}
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
		decoded = append(decoded, fmt.Sprintf("%s, ", disDestinations[dest]))
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
		decoded = append(decoded, fmt.Sprintf("%s", disDestinations[src]))
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
					decoded = append(decoded, fmt.Sprint(instr&0b11111))
				}
				decoded = append(decoded, "osr, rxfifo[y]")
			}
		} else {
			// to rxfifo
			if instr&(1<<3) != 0 {
				decoded = append(decoded, fmt.Sprintf("rxfifo[%d], isr", instr&0b11))
			} else {
				if instr&0b111 != 0 {
					decoded = append(decoded, fmt.Sprint(instr&0b11111))
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
