package iz6502

import (
	"testing"
)
/*
func TestMythical65c24T8NoUndocumented(t *testing.T) {
	m := new(FlatMemory)
	s := NewMythical65c24T8(m)
	s.abWidth = AB16;

	for i := 0; i < 256; i++ {
		if s.opcodes[i].cycles == 0 {
			t.Errorf("Opcode missing for $%02x.", i)
		}
	}
}

func TestMythical65c24T8asNMOS(t *testing.T) {
	m := new(Flat256KMemory)
	s := NewMythical65c24T8(m)

	m.loadBinary("testdata/6502_functional_test.bin")
	//executeSuite(t, s, 0x200, 240, false, 255)
}

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
*/
func TestExtraCycles(t *testing.T) {
	m := new(Flat256KMemory)
	s := NewMythical65c24T8(m)
	s.SetTrace(false)

    // INITIAL PC:beab A:9e X:b0 Y:1b S:b P:ad
    // FINAL PC:beae A:24 X:b0 Y:1b S:b P:6d
    // RAM [beab: fd][beac: 89][bead: eb][ec39: 7a][beae: 10]
	// SBC nnnn,X
	m.Poke(0xbeab, 0xfd)
	m.Poke(0xbeac, 0x89)
	m.Poke(0xbead, 0xeb)
	m.Poke(0xbeae, 0x10)
	m.Poke(0xec39, 0x7a)
	s.reg.setPC(0xbeab)
	s.reg.setA(0x9e)
	s.reg.setX(0xb0)
	s.ExecuteInstruction()

    // INITIAL PC:201e A:a2 X:44 Y:8f S:e0 P:ef
    // FINAL PC:2020 A:29 X:44 Y:8f S:e0 P:2d
    // RAM [201e: 71][201f: 1b][2020: 9a][1b: f6][1c: 3f][4085: 26]
	// ADC (aa),Y
	m.Poke(0x201e, 0x71)
	m.Poke(0x201f, 0x1b)
	m.Poke(0x2020, 0x9a)
	m.Poke(0x001b, 0xf6)
	m.Poke(0x001c, 0x3f)
	m.Poke(0x4085, 0x26)
	s.reg.setPC(0x201e)
	s.reg.setA(0x29)
	s.reg.setY(0x8f)
	s.ExecuteInstruction()

	// Took 6 cycles, it should be 5 for Name:7d 47 6c
    // INITIAL PC:6f85 A:e5 X:d4 Y:c7 S:f P:ef
    // FINAL PC:6f88 A:86 X:d4 Y:c7 S:f P:2d
    // RAM [6f85: 7d][6f86: 47][6f87: 6c][6c1b: 29][6d1b: 40][6f88: 1a]
    // ADC aaaa,X
	m.Poke(0x6f85, 0x7d)
	m.Poke(0x6f86, 0x47)
	m.Poke(0x6f87, 0x6c)
	m.Poke(0x6c1b, 0x29)
	m.Poke(0x6c1b, 0x40)
	s.reg.setPC(0x6f85)
	s.reg.setA(0xe5)
	s.reg.setX(0xd4)
	s.ExecuteInstruction()

	// Took 5 cycles, it should be 4 for Name:7d 2c be
    // INITIAL PC:5bb7 A:69 X:8d Y:ca S:46 P:2c
    // FINAL PC:5bba A:79 X:8d Y:ca S:46 P:2d
    // RAM [5bb7: 7d][5bb8: 2c][5bb9: be][beb9: aa][5bba: e2]
    // ADC aaaa,X
	m.Poke(0x5bb7, 0x7d)
	m.Poke(0x5bb8, 0x2c)
	m.Poke(0x5bb9, 0xbe)
	m.Poke(0x5bba, 0xe2)
	m.Poke(0xbeb9, 0xaa)
	s.reg.setPC(0x5bb7)
	s.reg.setA(0x69)
	s.reg.setX(0x8d)
	s.ExecuteInstruction()
}
