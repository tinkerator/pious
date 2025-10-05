# pious - a go package for editing RP PIO sequences

## Overview

The [RP2350B
processor](https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf)
contains a PIO engine that can be used to implement realtime
communication protocol exchanges. This Go based project is one to help
work with PIO sequences.

## Status

The package only supports assembling and disassembling known PIO
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
read: .program clock
read:   set     pindirs, 1
read: wrap_target:
read:   set     pins, 0 [1]
read:   set     pins, 1 [1]
```

That output matches the `pio/clock.pio` input.

## TODO

- Figure out how to generate `tinygo` compatible output.
- Support side-set feature.

## Support

This is a personal project aiming at exploring the capabilities of the
[pico2-ice](http://pico2-ice.tinyvision.ai/) board. As such, **no
support can be expected**. That being said, if you find a bug, or want
to request a feature, use the [github pious bug
tracker](https://github.com/tinkerator/pious/issues).

## License information

See the [LICENSE](LICENSE) file: the same BSD 3-clause license as that
used by [golang](https://golang.org/LICENSE) itself.
