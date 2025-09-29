# pious - a go package for editing RP PIO sequences

## Overview

The [RP2350B
processor](https://datasheets.raspberrypi.com/rp2350/rp2350-datasheet.pdf)
contains a PIO engine that can be used to implement realtime
communication protocol exchanges. This Go based project is one to help
work with PIO sequences.

## Status

The package currently only supports disassembling known PIO
instructions. The tests are all extracted from known assembly output.

To explore:

```
$ git clone https://github.com/tinkerator/pious.git
$ cd pious
$ go test -v
=== RUN   TestDisassemble
--- PASS: TestDisassemble (0.00s)
PASS
ok  	zappem.net/pub/io/pious	0.003s
```

## TODO

Add more test vectors for the `pious.Disassemble()` function. It
doesn't currently have 100% coverage of the instruction set.

## Support

This is a personal project aiming at exploring the capabilities of the
[pico2-ice](http://pico2-ice.tinyvision.ai/) board. As such, **no
support can be expected**. That being said, if you find a bug, or want
to request a feature, use the [github pious bug
tracker](https://github.com/tinkerator/pious/issues).

## License information

See the [LICENSE](LICENSE) file: the same BSD 3-clause license as that
used by [golang](https://golang.org/LICENSE) itself.
