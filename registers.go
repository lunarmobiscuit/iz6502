package iz6502

import "fmt"

const (
	regA    = 0
	regX    = 1
	regY    = 2
	regSP   = 3
	regNone = -1
	regP    = -2
)

const (
	flagN uint8 = 1 << 7
	flagV uint8 = 1 << 6
	flag5 uint8 = 1 << 5
	flagB uint8 = 1 << 4
	flagD uint8 = 1 << 3
	flagI uint8 = 1 << 2
	flagZ uint8 = 1 << 1
	flagC uint8 = 1 << 0
)

type registers struct {
	data [4]uint32
	p uint8
	pc uint32
}

func (r *registers) getRegister(width uint8, i int) uint32 {
	if i == regP {
		return uint32(r.p)
	} else {
		switch width {
		case R24, AB24: return r.data[i] & 0x0FFFFFF;
		case R16: return r.data[i] & 0x0FFFF;
		default: return r.data[i] & 0x0FF;
		}
	}
}

func (r *registers) getA(width uint8) uint32  { return r.getRegister(width, regA) }
func (r *registers) getX(width uint8) uint32  { return r.getRegister(width, regX) }
func (r *registers) getY(width uint8) uint32  { return r.getRegister(width, regY) }
func (r *registers) getP() uint8  { return r.p }
func (r *registers) getSP(width uint8) uint32  { return r.getRegister(width, regSP) }

func (r *registers) setRegister(width uint8, i int, v uint32) {
	if i == regP {
		r.p = uint8(v)
	} else {
		switch width {
		case R24, AB24: r.data[i] = v & 0x0FFFFFF
		case R16: r.data[i] = v & 0x0FFFF
		default: r.data[i] = v & 0x0FF
		}
	}
}
func (r *registers) setA(width uint8, v uint32)  { r.setRegister(width, regA, v) }
func (r *registers) setX(width uint8, v uint32)  { r.setRegister(width, regX, v) }
func (r *registers) setY(width uint8, v uint32)  { r.setRegister(width, regY, v) }
func (r *registers) setP(v uint8)  { r.p = v }
func (r *registers) setSP(width uint8, v uint32) { r.setRegister(width, regSP, v) }

func (r *registers) getPC() uint32 {
	return r.pc
}

func (r *registers) setPC(v uint32) {
	r.pc = v & 0x00ffffff
}

func (r *registers) getFlagBit(i uint8) uint8 {
	if r.getFlag(i) {
		return 1
	}
	return 0
}

func (r *registers) getFlag(i uint8) bool {
	return (r.p & i) != 0
}

func (r *registers) setFlag(i uint8) {
	r.p |= i
}

func (r *registers) clearFlag(i uint8) {
	r.p &^= i
}

func (r *registers) updateFlag(i uint8, v bool) {
	if v {
		r.setFlag(i)
	} else {
		r.clearFlag(i)
	}
}

func (r *registers) updateFlagZN(width uint8, t uint32) {
	switch width {
	case R24:
		t &= 0x0FFFFFF;
		r.updateFlag(flagN, t >= (1<<23))
	case R16:
		t &= 0x0FFFF;
		r.updateFlag(flagN, t >= (1<<15))
	default:
		t &= 0x0FF;
		r.updateFlag(flagN, t >= (1<<7))
	}
	r.updateFlag(flagZ, t == 0)
}

func (r *registers) updateFlag5B() {
	r.setFlag(flag5)
	r.clearFlag(flagB)
}

func (r registers) String() string {
	//ch := (r.getA() & 0x3F) + 0x40
	ch := uint8(r.getA(R08)) & 0x7F
	if ch < 0x20 {
		ch += 0x40
	}
	return fmt.Sprintf("A: %#06x(%v), X: %#06x, Y: %#06x, SP: %#06x, PC: %#06x, P: %#02x, (NV-BDIZC): %08b",
		r.getA(R24), string(ch), r.getX(R24), r.getY(R24), r.getSP(R24), r.getPC(), r.getP(), r.getP())
}
