package pious

import (
	"errors"
)

type Flags uint

const (
	flagCondition Flags = 1 << iota
	flagAddress
	flagPolSource
	flagWIndex
	flagISource
	flagBitCount
	flagMDestination
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

const (
	idxJMP = iota
	idxWAIT
	idxIN
	idxOUT
	idxNOP
	idxPUSH
	idxMOV1
	idxPULL
	idxMOV2
	idxIRQ
	idxSET
)

var instructions = []Instruction{
	{token: "jmp", mask: 0xe000, bits: 0x0000, flags: flagCondition | flagAddress},
	{token: "wait", mask: 0xe000, bits: 0x2000, flags: flagPolSource | flagWIndex},
	{token: "in", mask: 0xe000, bits: 0x4000, flags: flagISource | flagBitCount},
	{token: "out", mask: 0xe000, bits: 0x6000, flags: flagDestination | flagBitCount},
	{token: "nop", mask: 0xe0ff, bits: 0x8042, flags: 0},
	{token: "push", mask: 0xe09f, bits: 0x8000, flags: flagIfF | flagBlk},
	{token: "mov", mask: 0xe074, bits: 0x8010, flags: flagFromXIdxlIndex},
	{token: "pull", mask: 0xe09f, bits: 0x8080, flags: flagIfE | flagBlk},
	{token: "mov", mask: 0xe000, bits: 0xa000, flags: flagMDestination | flagOp | flagMSource},
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

// disMDestinations disassembles a destination type for moves.
var disMDestinations = []string{
	"pins",
	"x",
	"y",
	"pindirs",
	"exec",
	"pc",
	"isr",
	"osr",
}

// disISources holds in source choices.
var disISources = []string{
	"pins",
	"x",
	"y",
	"null",
	"",
	"",
	"isr",
	"osr",
}

// disMSources holds mov source choices.
var disMSources = []string{
	"pins",
	"x",
	"y",
	"null",
	"",
	"status",
	"isr",
	"osr",
}

var disBitSource = []string{
	"gpio",
	"pin",
	"irq",
	"jmppin",
}

var (
	ErrBad   = errors.New("invalid instruction")
	ErrEmpty = errors.New("empty instruction")
)

// Program holds a binary representation of a PIO program.
type Program struct {
	// Name names the PIO program
	Name string

	// Origin identifies the starting PC of the PIO program.
	Origin uint16

	// Wrap indicates where to wrap the PC value, and WrapTarget
	// is the value it is wrapped to.
	Wrap, WrapTarget uint16

	// SideSet indicates how many delay bits of the code syntax
	// are reserved for side-set pin value setting.
	SideSet uint16

	// Labels holds the jump label to offset mapping.
	Labels map[string]uint16

	// Targets holds the reverse of the jump table, with values
	// sorted lexicographically.
	Targets map[uint16][]string

	// Code holds the instructions that make up the executable PIO
	// program.
	Code []uint16
}
