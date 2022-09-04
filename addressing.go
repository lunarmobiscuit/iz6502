package iz6502

import "fmt"

const (
	modeImplicit = iota + 1
	modeImplicitX
	modeImplicitY
	modeAccumulator
	modeImmediate
	modeZeroPage
	modeZeroPageX
	modeZeroPageY
	modeRelative
	modeAbsolute
	modeAbsoluteX
	modeAbsoluteX65c02
	modeAbsoluteY
	modeIndirect
	modeIndexedIndirectX
	modeIndirectIndexedY
	// Added on the 65c02
	modeIndirect65c02Fix
	modeIndirectZeroPage
	modeAbsoluteIndexedIndirectX
	modeZeroPageAndRelative
)

func getWordInLine(line []uint8) uint32 {
	return uint32(line[1]) | (uint32(line[2]) << 8)
}

func get24BitsInLine(line []uint8) uint32 {
	return uint32(line[1]) | (uint32(line[2]) << 8) | (uint32(line[3]) << 16)
}

func resolveValue(s *State, line []uint8, opcode opcode) uint32 {
	switch opcode.addressMode {
	case modeAccumulator:
		return s.reg.getA(s.rWidth)
	case modeImplicitX:
		return s.reg.getX(s.rWidth)
	case modeImplicitY:
		return s.reg.getY(s.rWidth)
	case modeImmediate:
		switch s.rWidth {
		case R24:
			return uint32(line[1]) | uint32(line[2]) << 8 | uint32(line[3]) << 16
		case R16:
			return uint32(line[1]) | uint32(line[2]) << 8
		default:
			return uint32(line[1])
		}
	}

	// The value is in memory
	address := resolveAddress(s, line, opcode)
	switch s.rWidth {
	case R24:
		return uint32(s.mem.Peek(address)) | uint32(s.mem.Peek(address+1)) << 8 | uint32(s.mem.Peek(address+2)) << 16
	case R16:
		return uint32(s.mem.Peek(address)) | uint32(s.mem.Peek(address+1)) << 8
	default:
		return uint32(s.mem.Peek(address))
	}
}

func resolveSetValue(s *State, line []uint8, opcode opcode, value uint32) {
	switch opcode.addressMode {
	case modeAccumulator:
		s.reg.setA(s.rWidth, value)
		return
	case modeImplicitX:
		s.reg.setX(s.rWidth, value)
		return
	case modeImplicitY:
		s.reg.setY(s.rWidth, value)
		return
	}

	// The value is in memory
	address := resolveAddress(s, line, opcode)
	switch s.rWidth {
	case R24:
		s.mem.Poke(address, uint8(value))
		s.mem.Poke(address+1, uint8(value >> 8))
		s.mem.Poke(address+2, uint8(value >> 16))
	case R16:
		s.mem.Poke(address, uint8(value))
		s.mem.Poke(address+1, uint8(value >> 8))
	default:
		s.mem.Poke(address, uint8(value))
	}

	// On writes, the possible extra cycle crossing page boundaries is
	// added and already accounted for on NMOS
	if opcode.addressMode != modeAbsoluteX65c02 {
		s.extraCycleCrossingBoundaries = false
	}
}

func resolveAddress(s *State, line []uint8, opcode opcode) uint32 {
	var address uint32
	extraCycle := false

	switch opcode.addressMode {
	case modeZeroPage:
		address = uint32(line[1])
	case modeZeroPageX:
		address = (uint32(line[1]) + s.reg.getX(s.rWidth)) & 0x0FF
	case modeZeroPageY:
		address = (uint32(line[1]) + s.reg.getY(s.rWidth)) & 0x0FF
	case modeAbsolute:
		switch s.abWidth {
		case AB24:
			address = get24BitsInLine(line)
		default:
			address = getWordInLine(line)
		}
	case modeAbsoluteX65c02:
		fallthrough
	case modeAbsoluteX:
		switch s.abWidth {
		case AB24:
			base := get24BitsInLine(line)
			address, extraCycle = addOffset(s, base, s.reg.getX(s.rWidth))
		default:
			base := getWordInLine(line)
			address, extraCycle = addOffset(s, base, s.reg.getX(s.rWidth))
		}
	case modeAbsoluteY:
		switch s.abWidth {
		case AB24:
			base := get24BitsInLine(line)
			address, extraCycle = addOffset(s, base, s.reg.getY(s.rWidth))
		default:
			base := getWordInLine(line)
			address, extraCycle = addOffset(s, base, s.reg.getY(s.rWidth))
		}
	case modeIndexedIndirectX:
		switch s.abWidth {
		case AB24:
			addressAddress := uint32(line[1]) + s.reg.getX(s.rWidth)
			address = uint32(getZeroPage24Bits(s.mem, addressAddress))
		default:
			addressAddress := uint32(line[1]) + s.reg.getX(s.rWidth)
			// 24T8 BACKWARD COMPATIBILITY - in 16-bit mode (zp,X) wraps within 64K
			for addressAddress > 0x0ffff {
				addressAddress -= 0x10000
			}
			address = uint32(getZeroPageWord(s.mem, addressAddress))
		}
	case modeIndirect:
		switch s.abWidth {
		case AB24:
			addressAddress := get24BitsInLine(line)
			address = get24Bits(s.mem, addressAddress)
		default:
			addressAddress := uint32(getWordInLine(line))
			// 24T8 BACKWARD COMPATIBILITY - in 16-bit mode (aaaa) wraps within 64K
			for addressAddress > 0x0ffff {
				addressAddress -= 0x10000
			}
			address = uint32(getWordNoCrossPage(s.mem, addressAddress))
		}
	case modeIndirect65c02Fix:
		switch s.abWidth {
		case AB24:
			addressAddress := get24BitsInLine(line)
			address = get24Bits(s.mem, addressAddress)
		default:
			addressAddress := uint32(getWordInLine(line))
			address = uint32(getWord(s.mem, addressAddress))
			// 24T8 BACKWARD COMPATIBILITY - in 16-bit mode (aaaa) wraps within 64K
			for addressAddress > 0x0ffff {
				addressAddress -= 0x10000
			}
		}
	case modeIndirectIndexedY:
		switch s.abWidth {
			case AB24:
				base := uint32(getZeroPage24Bits(s.mem, uint32(line[1])))
				address, extraCycle = addOffset(s, base, s.reg.getY(s.rWidth))
			default:
				base := uint32(getZeroPageWord(s.mem, uint32(line[1])))
				address, extraCycle = addOffset(s, base, s.reg.getY(s.rWidth))
		}
	// 65c02 additions
	case modeIndirectZeroPage:
		switch s.abWidth {
			case AB24:
				address = uint32(getZeroPage24Bits(s.mem, uint32(line[1])))
			default:
				address = uint32(getZeroPageWord(s.mem, uint32(line[1])))
		}
	case modeAbsoluteIndexedIndirectX:
		switch s.abWidth {
			case AB24:
				addressAddress := get24BitsInLine(line) + s.reg.getX(s.rWidth)
				address = get24Bits(s.mem, addressAddress)
			default:
				addressAddress := getWordInLine(line) + s.reg.getX(s.rWidth)
				// 24T8 BACKWARD COMPATIBILITY - in 16-bit mode (aaaa,x) wraps within 64K
				for addressAddress > 0x0ffff {
					addressAddress -= 0x10000
				}
				address = uint32(getWord(s.mem, addressAddress))
		}
	case modeRelative:
		// This assumes that PC is already pointing to the next instruction
		base := s.reg.getPC()
		switch s.abWidth {
			case AB24:
				address, extraCycle = addOffsetRelative16(s, base, getWordInLine(line))
			default:
				address, extraCycle = addOffsetRelative(s, base, line[1])
		}
	case modeZeroPageAndRelative:
		// Two addressing modes combined. We refer to the second one, relative,
		// placed one byte after the zeropage reference
		base := s.reg.getPC()
		address, _ = addOffsetRelative(s, base, line[2])
	default:
		panic("Assert failed. Missing addressing mode")
	}

	if extraCycle {
		s.extraCycleCrossingBoundaries = true
	}

	return address
}

/*
Note: extra cycle on reads when crossing page boundaries.

Only for:
	modeAbsoluteX
	modeAbsoluteY
	modeIndirectIndexedY
	modeRelative
	modeZeroPageAndRelative
That is when we add a 8 bit offset to a 16 bit base. The reason is
that if don't have a page crossing the CPU optimizes one cycle assuming
that the MSB addition won't change. If it does we spend this extra cycle.

Note that for writes we don't add a cycle in this case. There is no
optimization that could make a double write. The regular cycle count
is always the same with no optimization.
*/
func addOffset(s *State, base uint32, offset uint32) (uint32, bool) {
	dest := base + offset
	switch s.abWidth {
		case AB24:
			// 24-bit addressing leaves the address as-is
		default:
			// 24T8 BACKWARD COMPATIBILITY - in 16-bit mode offsets wrap within 64K
			for dest > 0x0ffff {
				dest -= 0x10000
			}
	}
	if (base & 0x00ff00) != (dest & 0x00ff00) {
		return dest, true
	} else {
		return dest, false
	}
}

func addOffsetRelative(s *State, base uint32, offset uint8) (uint32, bool) {
	dest := base + uint32(int8(offset))
	switch s.abWidth {
		case AB24:
			// 24-bit addressing leaves the address as-is
		default:
			// 24T8 BACKWARD COMPATIBILITY - in 16-bit mode offsets wrap within 64K
			if s.reg.pc < 0x0ffff { // but not if the PC is above 64K
				for dest > 0x0ffff {
					dest -= 0x10000
				}
			}
	}
	if (base & 0xff00) != (dest & 0xff00) {
		return dest, true
	} else {
		return dest, false
	}
}

func addOffsetRelative16(s *State, base uint32, offset uint32) (uint32, bool) {
	dest := base + uint32(int16(offset))
	if (base & 0x00ff00) != (dest & 0x00ff00) {
		return dest, true
	} else {
		return dest, false
	}
}

func addressModeString(addressMode int) string {
	switch (addressMode) {
	case modeImplicit: return "modeImplicit"
	case modeImplicitX: return "modeImplicitX"
	case modeImplicitY: return "modeImplicitY"
	case modeAccumulator: return "modeAccumulator"
	case modeImmediate: return "modeImmediate"
	case modeZeroPage: return "modeZeroPage"
	case modeZeroPageX: return "modeZeroPageX"
	case modeZeroPageY: return "modeZeroPageY"
	case modeAbsolute: return "modeAbsolute"
	case modeAbsoluteX: return "modeAbsoluteX"
	case modeAbsoluteX65c02: return "modeAbsoluteX65c02"
	case modeAbsoluteY: return "modeAbsoluteY"
	case modeIndirect: return "modeIndirect"
	case modeIndexedIndirectX: return "modeIndexedIndirectX"
	case modeIndirectIndexedY: return "modeIndirectIndexedY"
	case modeIndirect65c02Fix: return "modeIndirect65c02Fix"
	case modeIndirectZeroPage: return "modeIndirectZeroPage"
	case modeAbsoluteIndexedIndirectX: return "modeAbsoluteIndexedIndirectX"
	case modeZeroPageAndRelative: return "modeZeroPageAndRelative"
	default: return fmt.Sprintf("modeUnknown %d", addressMode)
	}
}

func lineString(s *State, line []uint8, opcode opcode) string {
	t := opcode.name
	switch opcode.addressMode {
	case modeImplicit:
	case modeImplicitX:
	case modeImplicitY:
		//Nothing
	case modeAccumulator:
		t += " A"
	case modeImmediate:
		switch s.rWidth {
		case R24:
			t += fmt.Sprintf(" #$%02x%02x%02x", line[1], line[2], line[3])
		case R16:
			t += fmt.Sprintf(" #$%02x%02x", line[1], line[2])
		default:
			t += fmt.Sprintf(" #$%02x", line[1])
		}
	case modeZeroPage:
		t += fmt.Sprintf(" $%02x", line[1])
	case modeZeroPageX:
		t += fmt.Sprintf(" $%02x,X", line[1])
	case modeZeroPageY:
		t += fmt.Sprintf(" $%02x,Y", line[1])
	case modeRelative:
		t += fmt.Sprintf(" *%+x", int8(line[1]))
	case modeAbsolute:
		switch s.abWidth {
		case AB24:
			t += fmt.Sprintf(" $%06x", get24BitsInLine(line))
		default:
			t += fmt.Sprintf(" $%04x", getWordInLine(line))
		}
	case modeAbsoluteX65c02:
		fallthrough
	case modeAbsoluteX:
		switch s.abWidth {
		case AB24:
			t += fmt.Sprintf(" $%06x,X", get24BitsInLine(line))
		default:
			t += fmt.Sprintf(" $%04x,X", getWordInLine(line))
		}
	case modeAbsoluteY:
		switch s.abWidth {
		case AB24:
			t += fmt.Sprintf(" $%06x,Y", get24BitsInLine(line))
		default:
			t += fmt.Sprintf(" $%04x,Y", getWordInLine(line))
		}
	case modeIndirect65c02Fix:
		fallthrough
	case modeIndirect:
		switch s.abWidth {
		case AB24:
			t += fmt.Sprintf(" ($%06x)", get24BitsInLine(line))
		default:
			t += fmt.Sprintf(" ($%04x)", getWordInLine(line))
		}
	case modeIndexedIndirectX:
		t += fmt.Sprintf(" ($%02x,X)", line[1])
	case modeIndirectIndexedY:
		t += fmt.Sprintf(" ($%02x),Y", line[1])
	// 65c02 additions:
	case modeIndirectZeroPage:
		t += fmt.Sprintf(" ($%02x)", line[1])
	case modeAbsoluteIndexedIndirectX:
		switch s.abWidth {
		case AB24:
			t += fmt.Sprintf(" ($%06x,X)", get24BitsInLine(line))
		default:
			t += fmt.Sprintf(" ($%04x,X)", getWordInLine(line))
		}
	case modeZeroPageAndRelative:
		t += fmt.Sprintf(" $%02x %+x", line[1], int8(line[2]))
	default:
		t += "UNKNOWN MODE"
	}
	return t
}
