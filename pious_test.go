package pious

import "testing"

func TestDisassemble(t *testing.T) {
	vs := []struct {
		c uint16
		d string
	}{
		{c: 0x80a0, d: "pull\tblock"},
		{c: 0x6040, d: "out\ty, 32"},
		{c: 0xa022, d: "mov\tx, y"},
		{c: 0xe001, d: "set\tpins, 1"},
		{c: 0x0044, d: "jmp\tx-- 4"},
		{c: 0xa022, d: "mov\tx, y"},
		{c: 0xe000, d: "set\tpins, 0"},
		{c: 0x0047, d: "jmp\tx-- 7"},
		{c: 0xa0c3, d: "mov\tisr, null"},
		{c: 0xe043, d: "set\ty, 3"},
		{c: 0x8010, d: "mov\trxfifo[y], isr"},
		{c: 0x0082, d: "jmp\ty-- 2"},
		{c: 0xa02b, d: "mov\tx, !null"},
		{c: 0xa0c9, d: "mov\tisr, !x"},
		{c: 0x8018, d: "mov\trxfifo[0], isr"},
		{c: 0x0045, d: "jmp\tx-- 5"},
		{c: 0x8018, d: "mov\trxfifo[0], isr"},
		{c: 0x8098, d: "mov\tosr, rxfifo[0]"},
	}
	for i, v := range vs {
		if d, err := Disassemble(v.c, nil); err != nil {
			t.Errorf("test %d failed: %v", i, err)
		} else if d != v.d {
			t.Errorf("test %d failed got=%q want=%q", i, d, v.d)
		}
	}
}

func TestAssemble(t *testing.T) {
	for _, p := range []*Program{
		nil,
		&Program{
			Attr: Settings{
				SideSet: 1,
			},
		},
		&Program{
			Attr: Settings{
				SideSet:    2,
				SideSetOpt: true,
			},
		},
	} {
		for i := 0; i <= 0xffff; i++ {
			d, err := Disassemble(uint16(i), p)
			if err != nil {
				// Un-comment the following to explore new
				// opcode support
				//t.Errorf("[%d] bad (%q): %v", i, d, err)
				continue
			}
			ts, err := Assemble(d, p)
			if want := uint16(i); err != nil || ts != want {
				if ins := instructions[idxIRQ]; ts^want == 0b100000 && ts&(ins.mask|0b1000000) == (ins.bits|0b1000000) {
					// special case for IRQ instructions:
					// the wait bit is ignored if clear is
					// set.
					continue
				}
				t.Errorf("[%d] bad (%q) got=%04x want=%04x (%016b, %016b): %v", i, d, ts, i, ts, i, err)
			}
		}
	}
}
