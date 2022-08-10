package iz6502

import (
	"testing"
)

func TestMythical65c24T8(t *testing.T) {
	m := new(Flat256KMemory)
	s := NewMythical65c24T8(m)
	s.abWidth = AB24;

	m.Poke(0x0ffff, 0xea)
	m.Poke(0x10000, 0xea)
	m.Poke(0x3ffff, 0xaa)
	m.Poke(vector24Reset, 0xba)
	m.Poke(vector24Reset+1, 0xdc)
	m.Poke(vector24Reset+2, 0xfe)
	s.reg.setPC(0x0ffff)

	if m.Peek(0x00ffff) != 0xea {
		t.Fatalf("0x00ffff: %02x instead of ea\n", m.Peek(0x00ffff))
	}
	if m.Peek(0x10000) != 0xea {
		t.Fatalf("0x10000: %02x instead of ea\n", m.Peek(0x10000))
	}
	if m.Peek(0x20000) != 0x00 {
		t.Fatalf("0x10000: %02x instead of 00\n", m.Peek(0x20000))
	}
	if m.Peek(0x3ffff) != 0xaa {
		t.Fatalf("0x3ffff: %02x instead of aa\n", m.Peek(0x3ffff))
	}
	if m.Peek(0xffffff) != 0xaa {
		t.Fatalf("0xffffff: %02x instead of aa\n", m.Peek(0xffffff))
	}

	s.ExecuteInstruction()
	pc := s.reg.getPC()
	if pc != 0x010000 {
		t.Fatalf("65c24T8 PC $00ffff + 1 = $%06x\n", pc)
	}

	s.ExecuteInstruction()
	pc = s.reg.getPC()
	if pc != 0x010001 {
		t.Fatalf("65c24T8 PC $10000 + 1 = $%6x\n", pc)
	}

	s.Reset()
	pc = s.reg.getPC()
	if pc != 0x00fedcba {
		t.Fatalf("65c24T8 RESET PC = $%06x\n", pc)
	}
}

func TestWiderPC(t *testing.T) {
	var r registers
	data := uint32(0xffc600)
	r.setPC(data)
	if r.getPC() != data {
		t.Error("Error storing and loading 24-bit PC")
	}
}
