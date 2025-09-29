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
		if d, err := Disassemble(v.c); err != nil {
			t.Errorf("test %d failed: %v", i, err)
		} else if d != v.d {
			t.Errorf("test %d failed got=%q want=%q", i, d, v.d)
		}
	}
}
