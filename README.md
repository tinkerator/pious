# pious - a go package supporting RP PIO sequences

## Overview

The [RP2350B
processor](https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf)
contains a PIO engine that can be used to implement realtime
communication protocol exchanges. This Go based project is one to help
work with PIO sequences.

## Status

The package supports assembling and disassembling known PIO
instructions. The tests are extracted from known assembly output.

To explore:

```
$ git clone https://github.com/tinkerator/pious.git
$ cd pious
$ go test -v
=== RUN   TestDisassemble
--- PASS: TestDisassemble (0.00s)
=== RUN   TestAssemble
--- PASS: TestAssemble (0.20s)
PASS
ok  	zappem.net/pub/io/pious	0.204s
```

An example assembling and then disassembling a `.pio` program:

```
$ go run examples/piocli.go --src pio/clock.pio
.program clock
	set	pindirs, 1
.wrap_target
	set	pins, 0 [1]
	set	pins, 1 [1]
.wrap
```

That output matches the `pio/clock.pio` input.

You can prepare a [tinygo](https://tinygo.org/) compatible package
that uses the [`rp2-pio`](github.com/tinygo-org/pio/rp2-pio) package
to manage a PIO sequence like this in the form of a package, `clock`
in this case:

```
$ go install examples/piocli.go
$ ~/go/bin/piocli --src pio/clock.pio --tinygo > /tmp/clock.go
$ grep func /tmp/clock.go
func (s *StateMachine) Start() {
func (s *StateMachine) Activate(run bool) {
func Assign(block *pio.PIO) (*Engine, error) {
func (e *Engine) ConfigureClock(pinBase machine.Pin) (*StateMachine, error) {
```

The way to initialize this PIO code is to select a GPIO (`pinBase`)
and use tinygo code like this:
```
e, _ := clock.Assign(rp2pio.PIO0)
s, _ := e.ConfigureClock(machine.GPIO6)
s.Start()
```

You can disable or enable the running PIO clock driving `machine.GPIO6`
using `s.Activate(false)` and `s.Activate(true)` respectively.

## Reference

The PIO Instruction set has 10 instruction types. One of these (`nop`)
is a conventional alias for a pointless `mov` instruction. Full
details are provided in the [RP2350
Datasheet](https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf),
but we provide a quick summary here:

NOTE: All instructions that assign values have an assignment direction
      to the left, that is the syntax specifies the destination to the
      left of the source.

- `jmp` is the control flow jump instruction, it sets the next
  execution address. If it includes a condition then the condition
  must evaluate to true for the jump to be performed, otherwise PIO
  program execution continues at the next instruction.

- `wait` causes execution of the PIO program to stall until some
  condition becomes true. The first argument, 1 or 0, indicates what
  polarity of value is being waited for. If omitted 1 is assumed. You
  can wait for `gpio`, `pin`, `irq` or `jmppin` (pin offset from some
  base index).

- `in` shifts a counted number of bits into the `isr` register. (Bit
  shift _direction_ is a configuration setting for the executing state
  machine, and only refers to the end of `isr` that the bits are
  removed from.). The source of the bits is provided with the
  instruction and is one of: `pins`, `x`, `y`, `null` (zeros), `isr`,
  `osr`.  If _automatic push_ is enabled, then `isr` is pushed into
  `rxfifo` when it is sufficiently empty. Operations stall in such
  cases when the `rxfifo` becomes full.

- `out` shifts a counted number of bits out of the `osr`
  register. (Bit shift _direction_ is a configuration setting for the
  executing state machine, and only refers to the end of `osr` that
  the bits are inserted into.) The destination for the shift is one
  of: `pins`, `x`, `y`, `null` (discard), `pindirs`, `pc`, `isr`,
  `exec`. The shift wholly sets the destination register with
  unshifted bits being set to zero. In this way, we are setting the
  register with a subset of the `osr` bits. If _automatic pull_ is
  enabled, then `osr` is refilled from `txfifo` when it is
  sufficiently empty, or stalls until something has been inserted into
  the `txfifo`.

- `push` can be used to more explicitly push (as opposed to _automatic
  push_) whole 32-bit `isr` values into the `rxfifo`.

- `pull` can be used to more explicitly pull (as opposed to _automatic
  pull_) whole 32-bit `osr` values from the `rxfifo`.

- `mov` is used to move between registers or a register: `pins`, `x`,
  `y`, `isr`, `osr`, and an indexed element in the `rxfifo`. Some
  values can also be used as sources: `null`, `status`, and an indexed
  element of the `rxfifo`. Some values can also be used as
  destinations: `pindirs`, `exec` (force execution of a datum as an
  instruction), `pc` (indirect jump).

- `irq` generate an interrupt with the indicated index.

- `set` sets a register value from an immediate 5-bit value. Larger
  values need to be provided through `out` or `mov` operations.

## Examples

The `pio/` subdirectory contains some PIO example source files. These
are used to validate the package, and provide some references for
writing your own PIO code.

So far, we have:

- [`pio/clock.pio`](pio/clock.pio) a 2-cycle clock output (pico2 75 MHz)
- [`pio/sider.pio`](pio/sider.pio) a simple SPI transfer loop (pico2 75 MHz)

## TODO

Things I'm thinking about exploring:

- Figure out how to fully clear a PIO block (i.e. reclaim it at
  runtime). Uses block.ClearProgramSection(), but need to disable
  state machines first.

- Figure out how to adjust the PIO frequency. My initial attempts
  don't appear to be reliable with the rp2-pio code yet.

- PIO simulator for debugging.

## Support

This is a personal project aiming at exploring the capabilities of the
[pico2-ice](http://pico2-ice.tinyvision.ai/) board. As such, **no
support can be expected**. That being said, if you find a bug, or want
to request a feature, use the [github pious bug
tracker](https://github.com/tinkerator/pious/issues).

## License information

See the [LICENSE](LICENSE) file: the same BSD 3-clause license as that
used by [golang](https://golang.org/LICENSE) itself.
