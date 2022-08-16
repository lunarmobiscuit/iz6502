package iz6502

func buildOpTransfer(regSrc int, regDst int) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		var value uint32
		if regSrc == regSP {
			value = s.reg.getSP(s.sWidth)
			s.reg.setRegister(s.rWidth, regDst, value)
		} else if regDst == regSP {
			value = s.reg.getRegister(s.rWidth, regSrc)
			s.reg.setSP(s.sWidth, value)
		} else {
			value = s.reg.getRegister(s.rWidth, regSrc)
			s.reg.setRegister(s.rWidth, regDst, value)
		}

		if regDst != regSP {
			s.reg.updateFlagZN(s.rWidth, value)
		}
	}
}

func buildOpIncDec(inc bool) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := resolveValue(s, line, opcode)
		if opcode.addressMode == modeAbsoluteX || opcode.addressMode == modeAbsoluteY {
			// Double read, needed to pass A2Audit for the Language Card
			value = resolveValue(s, line, opcode)
		}
		if inc {
			value++
		} else {
			value--
		}
		s.reg.updateFlagZN(s.rWidth, value)
		resolveSetValue(s, line, opcode, value)
	}
}

func buildOpShift(isLeft bool, isRotate bool) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := resolveValue(s, line, opcode)
		oldCarry := s.reg.getFlagBit(flagC)
		var carry bool
		if isLeft {
			switch s.rWidth {
			case R24: carry = (value & 0x0800000) != 0
			case R16: carry = (value & 0x08000) != 0
			default: carry = (value & 0x080) != 0
			}
			
			value <<= 1
			if isRotate {
				value += uint32(oldCarry)
			}

			switch s.rWidth {
			case R24:
				carry = (value & 0x1000000) != 0
				value &= 0x0FFFFFF;
			case R16:
				carry = (value & 0x10000) != 0
				value &= 0x0FFFF;
			default:
				carry = (value & 0x100) != 0
				value &= 0x0FF;
			}
		} else {
			carry = (value & 0x01) != 0
			value >>= 1
			if isRotate {
				switch s.rWidth {
				case R24:
					value += uint32(oldCarry) << 23
					value &= 0x0FFFFFF;
				case R16:
					value += uint32(oldCarry) << 15
					value &= 0x0FFFF;
				default:
					value += uint32(oldCarry) << 7
					value &= 0x0FF;
				}
				
			}
		}
		s.reg.updateFlag(flagC, carry)
		s.reg.updateFlagZN(s.rWidth, value)
		resolveSetValue(s, line, opcode, value)
	}
}

func buildOpLoad(regDst int) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := resolveValue(s, line, opcode)
		s.reg.setRegister(s.rWidth, regDst, value)
		s.reg.updateFlagZN(s.rWidth, value)
	}
}

func buildOpStore(regSrc int) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := s.reg.getRegister(s.rWidth, regSrc)
		resolveSetValue(s, line, opcode, value)
	}
}

func buildOpUpdateFlag(flag uint8, value bool) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		s.reg.updateFlag(flag, value)
	}
}

func buildOpBranch(flag uint8, test bool) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		if s.reg.getFlag(flag) == test {
			s.extraCycleBranchTaken = true
			address := resolveAddress(s, line, opcode)
			s.reg.setPC(address)
		}
	}
}

func buildOpBranchOnBit(bit uint8, test bool) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		// Note that those operations have two addressing modes:
		// one for the zero page value, another for the relative jump.
		// We will have to resolve the first one here.
		value := s.mem.Peek(uint32(line[1]))
		bitValue := ((value >> bit) & 1) == 1

		if bitValue == test {
			address := resolveAddress(s, line, opcode)
			s.reg.setPC(address)
		}
	}
}

func buildOpSetBit(bit uint8, set bool) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := resolveValue(s, line, opcode)
		if set {
			value = value | (1 << bit)
		} else {
			value = value &^ (1 << bit)
		}
		resolveSetValue(s, line, opcode, value)
	}
}

func opBIT(s *State, line []uint8, opcode opcode) {
	value := resolveValue(s, line, opcode)
	acc := s.reg.getA(s.rWidth)
	s.reg.updateFlag(flagZ, value&acc == 0)
	// The immediate addressing mode (65C02 or 65816 only) does not affect N & V.
	if opcode.addressMode != modeImmediate {
		switch s.rWidth {
		case R24:
			s.reg.updateFlag(flagN, value&(1<<23) != 0)
			s.reg.updateFlag(flagV, value&(1<<22) != 0)
		case R16:
			s.reg.updateFlag(flagN, value&(1<<15) != 0)
			s.reg.updateFlag(flagV, value&(1<<14) != 0)
		default:
			s.reg.updateFlag(flagN, value&(1<<7) != 0)
			s.reg.updateFlag(flagV, value&(1<<6) != 0)
		}
	}
}

func opTRB(s *State, line []uint8, opcode opcode) {
	value := resolveValue(s, line, opcode)
	a := s.reg.getA(s.rWidth)
	s.reg.updateFlag(flagZ, (value&a) == 0)
	resolveSetValue(s, line, opcode, value&^a)
}

func opTSB(s *State, line []uint8, opcode opcode) {
	value := resolveValue(s, line, opcode)
	a := s.reg.getA(s.rWidth)
	s.reg.updateFlag(flagZ, (value&a) == 0)
	resolveSetValue(s, line, opcode, value|a)
}

func buildOpCompare(reg int) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := resolveValue(s, line, opcode)
		reference := s.reg.getRegister(s.rWidth, reg)
		s.reg.updateFlagZN(s.rWidth, reference - value)
		s.reg.updateFlag(flagC, reference >= value)
	}
}

func operationAnd(a uint32, b uint32) uint32 { return a & b }
func operationOr(a uint32, b uint32) uint32  { return a | b }
func operationXor(a uint32, b uint32) uint32 { return a ^ b }

func buildOpLogic(operation func(uint32, uint32) uint32) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		value := resolveValue(s, line, opcode)
		result := operation(value, s.reg.getA(s.rWidth))
		s.reg.setA(s.rWidth, result)
		s.reg.updateFlagZN(s.rWidth, result)
	}
}

func opADC(s *State, line []uint8, opcode opcode) {
	value := resolveValue(s, line, opcode)
	aValue := s.reg.getA(s.rWidth)
	carry := s.reg.getFlagBit(flagC)

	total := aValue + value + uint32(carry)
	var signedTotal int32
	switch s.rWidth {
	case R24:
		signedTotal = int32(aValue) + int32(value) + int32(carry)
	case R16:
		signedTotal = int32(int16(aValue)) + int32(int16(value)) + int32(carry)
	default:
		signedTotal = int32(int16(int8(aValue)) + int16(int8(value)) + int16(carry))
	}

	var truncated uint32
	switch s.rWidth {
	case R24: truncated = total & 0x0FFFFFF
	case R16: truncated = total & 0x0FFFF
	default: truncated = total & 0x0FF
	}

	if s.reg.getFlag(flagD) {
		totalBcdLo := uint(aValue&0x0f) + uint(value&0x0f) + uint(carry)
		totalBcdHi := uint(aValue>>4) + uint(value>>4)
		if totalBcdLo >= 10 {
			totalBcdLo -= 10
			totalBcdHi++
		}
		totalBcdHiPrenormalised := uint8(totalBcdHi & 0xf)
		newCarry := false
		if totalBcdHi >= 10 {
			totalBcdHi -= 10
			newCarry = true
		}
		totalBcd := uint8(totalBcdHi)<<4 + (uint8(totalBcdLo) & 0xf)
		s.reg.setA(R08, uint32(totalBcd))
		s.reg.updateFlag(flagC, newCarry)
		s.reg.updateFlag(flagV, (uint8(value)>>7 == uint8(aValue)>>7) &&
			(uint8(value)>>7 != uint8(totalBcdHiPrenormalised)>>3))
	} else {
		s.reg.setA(s.rWidth, truncated)
		switch s.rWidth {
		case R24:
			s.reg.updateFlag(flagC, total > 0x0FFFFFF)
			s.reg.updateFlag(flagV, signedTotal < -8388608 || signedTotal > 8388607)
		case R16:
			s.reg.updateFlag(flagC, total > 0x0FFFF)
			s.reg.updateFlag(flagV, signedTotal < -32768 || signedTotal > 32767)
		default:
			s.reg.updateFlag(flagC, total > 0x0FF)
			s.reg.updateFlag(flagV, signedTotal < -128 || signedTotal > 127)
		}
		// Effectively the same as the less clear:
		// s.reg.updateFlag(flagV, (value>>7 == aValue>>7) && (value>>7 != truncated>>7))
		// See http://www.6502.org/tutorials/vflag.html
	}

	// ZN flags behave for BCD as if the operation was binary?
	s.reg.updateFlagZN(s.rWidth, truncated)
}

func opADCAlt(s *State, line []uint8, opcode opcode) {
	opADC(s, line, opcode)
	if s.reg.getFlag(flagD) {
		s.extraCycleBCD = true
	}

	// The Z and N flags on BCD are fixed in 65c02.
	s.reg.updateFlagZN(s.rWidth, s.reg.getA(s.rWidth))
}

func opSBC(s *State, line []uint8, opcode opcode) {
	value := resolveValue(s, line, opcode)
	aValue := s.reg.getA(s.rWidth)
	carry := s.reg.getFlagBit(flagC)

	var total uint32
	var signedTotal int32
	var truncated uint32
	switch s.rWidth {
	case R24:
		total = 0x1000000 + aValue - value + uint32(carry) - 1
		signedTotal = int32(aValue) - int32(value) + int32(carry) - 1
		truncated = total & 0x0FFFFFF
	case R16:
		total = 0x10000 + aValue - value + uint32(carry) - 1
		signedTotal = int32(int16(aValue)) - int32(int16(value)) + int32(carry) - 1
		truncated = total & 0x0FFFF
	default:
		total = 0x100 + aValue - value + uint32(carry) - 1
		signedTotal = int32(int8(aValue)) - int32(int8(value)) + int32(carry) - 1
		truncated = total & 0x0FF
	}

	if s.reg.getFlag(flagD) {
		totalBcdLo := int(aValue&0x0f) - int(value&0x0f) + int(carry) - 1
		totalBcdHi := int(aValue>>4) - int(value>>4)
		if totalBcdLo < 0 {
			totalBcdLo += 10
			totalBcdHi--
		}
		newCarry := true
		if totalBcdHi < 0 {
			totalBcdHi += 10
			newCarry = false
		}
		totalBcd := uint8(totalBcdHi)<<4 + (uint8(totalBcdLo) & 0xf)
		s.reg.setA(R08, uint32(totalBcd))
		s.reg.updateFlag(flagC, newCarry)
	} else {
		s.reg.setA(s.rWidth, truncated)
		switch s.rWidth {
		case R24:
			s.reg.updateFlag(flagC, total > 0x0FFFFFF)
		case R16:
			s.reg.updateFlag(flagC, total > 0x0FFFF)
		default:
			s.reg.updateFlag(flagC, total > 0x0FF)
		}
	}

	// ZNV flags behave for SBC as if the operation was binary
	s.reg.updateFlagZN(s.rWidth, truncated)
	switch s.rWidth {
	case R24:
		s.reg.updateFlag(flagV, signedTotal < -8388608 || signedTotal > 8388607)
	case R16:
		s.reg.updateFlag(flagV, signedTotal < -32768 || signedTotal > 32767)
	default:
		s.reg.updateFlag(flagV, signedTotal < -128 || signedTotal > 127)
	}
}

func opSBCAlt(s *State, line []uint8, opcode opcode) {
	opSBC(s, line, opcode)
	if s.reg.getFlag(flagD) {
		s.extraCycleBCD = true
	}
	// The Z and N flags on BCD are fixed in 65c02.
	s.reg.updateFlagZN(s.rWidth, s.reg.getA(s.rWidth))
}

const stackAddress uint32 = 0x0100

func pushByte(s *State, value uint8) {
	var adresss uint32
	if s.sWidth == R08 {
		adresss = stackAddress + s.reg.getSP(s.sWidth)
	} else {
		adresss = s.reg.getSP(s.sWidth)
	}
	s.mem.Poke(adresss, value)
	s.reg.setSP(s.sWidth, s.reg.getSP(s.sWidth) - 1)
}

func pullByte(s *State) uint8 {
	s.reg.setSP(s.sWidth, s.reg.getSP(s.sWidth) + 1)
	var adresss uint32
	if s.sWidth == R08 {
		adresss = stackAddress + s.reg.getSP(s.sWidth)
	} else {
		adresss = s.reg.getSP(s.sWidth)
	}
	return s.mem.Peek(adresss)
}

func pushWord(s *State, value uint16) {
	pushByte(s, uint8(value>>8))
	pushByte(s, uint8(value))
}

func pullWord(s *State) uint16 {
	return uint16(pullByte(s)) +
		(uint16(pullByte(s)) << 8)

}

func push24Bits(s *State, value uint32) {
	pushByte(s, uint8(value>>16))
	pushByte(s, uint8(value>>8))
	pushByte(s, uint8(value))
}

func pull24Bits(s *State) uint32 {
	return uint32(pullByte(s)) |
		(uint32(pullByte(s)) << 8) |
		(uint32(pullByte(s)) << 16)

}

func buildOpPull(regDst int) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		var value uint32
		switch s.rWidth {
		case R24:
			value = pull24Bits(s)
		case R16:
			value = uint32(pullWord(s))
		default:
			value = uint32(pullByte(s))
		}
		s.reg.setRegister(s.rWidth, regDst, value)
		if regDst == regP {
			s.reg.updateFlag5B()
		} else {
			s.reg.updateFlagZN(s.rWidth, value)
		}
	}
}

func buildOpPush(regSrc int) opFunc {
	return func(s *State, line []uint8, opcode opcode) {
		if regSrc == regP {
			value := uint8(s.reg.getRegister(s.rWidth, regSrc))
			value |= flagB + flag5
			pushByte(s, value)
		} else {
			value := s.reg.getRegister(s.rWidth, regSrc)
			switch s.rWidth {
			case R24: push24Bits(s, value)
			case R16: pushWord(s, uint16(value))
			default: pushByte(s, uint8(value))
			}
		}
	}
}

func opJMP(s *State, line []uint8, opcode opcode) {
	address := resolveAddress(s, line, opcode)
	s.reg.setPC(address)
}

func opNOP(s *State, line []uint8, opcode opcode) {}

func opHALT(s *State, line []uint8, opcode opcode) {
	pc := s.reg.getPC()

	// 24T8 BACKWARD COMPATIBILITY - when BRK at PC=$0000 roll back to $FFFF
	if (s.abWidth == AB16) && (pc == 0x000000) {
		pc = 0x00ffff;
	} else {
		pc -= 1
	}
	s.reg.setPC(pc)
}

func opJSR(s *State, line []uint8, opcode opcode) {
	switch s.abWidth {
	case AB24: push24Bits(s, s.reg.getPC()-1)
	default: pushWord(s, uint16(s.reg.getPC()-1))
	}
	address := resolveAddress(s, line, opcode)
	s.reg.setPC(address)
}

func opRTI(s *State, line []uint8, opcode opcode) {
	s.reg.setP(pullByte(s))
	s.reg.updateFlag5B()
	switch s.abWidth {
	case AB24: s.reg.setPC(uint32(pull24Bits(s)))
	default: s.reg.setPC(uint32(pullWord(s)))
	}
}

func opRTS(s *State, line []uint8, opcode opcode) {
	switch s.abWidth {
	case AB24: s.reg.setPC(uint32(pull24Bits(s) + 1))
	default: s.reg.setPC(uint32(pullWord(s) + 1))
	}
}

func opBRK(s *State, line []uint8, opcode opcode) {
	switch s.abWidth {
	case AB24: push24Bits(s, s.reg.getPC()+1)
	default: pushWord(s, uint16(s.reg.getPC()+1))
	}
	pushByte(s, s.reg.getP()|(flagB+flag5))
	s.reg.setFlag(flagI)
	switch s.abWidth {
	case AB24: s.reg.setPC(get24Bits(s.mem, vector24Break))
	default: s.reg.setPC(uint32(getWord(s.mem, vectorBreak)))
	}
}

func opBRKAlt(s *State, line []uint8, opcode opcode) {
	opBRK(s, line, opcode)
	/*
		The only difference in the BRK instruction on the 65C02 and the 6502
		is that the 65C02 clears the D (decimal) flag on the 65C02, whereas
		the D flag is not affected on the 6502.
	*/
	s.reg.clearFlag(flagD)
}

func opSTZ(s *State, line []uint8, opcode opcode) {
	resolveSetValue(s, line, opcode, 0)
}

// New opcode in 65C24T8 to return capabilities of the CPU
func opCPU(s *State, line []uint8, opcode opcode) {
	s.reg.setA(R24, 0x650200 | AB24 | R24 | N_THREADS)
}

// New opcode in 65C24T8 to switch to 24-bit address mode
func opA24(s *State, line []uint8, opcode opcode) {
	s.abWidth = AB24;
}

// New opcode in 65C24T8 to switch to 16-bit register mode
func opR16(s *State, line []uint8, opcode opcode) {
	s.rWidth = R16;
}

// New opcode in 65C24T8 to switch to 24-bit register mode
func opR24(s *State, line []uint8, opcode opcode) {
	s.rWidth = R24;
}

// New opcode in 65C24T8 to switch to 16-bit register mode
func opW16(s *State, line []uint8, opcode opcode) {
	s.abWidth = AB24;
	s.rWidth = R16;
}

// New opcode in 65C24T8 to switch to 24-bit register mode
func opW24(s *State, line []uint8, opcode opcode) {
	s.abWidth = AB24;
	s.rWidth = R24;
}

// New opcode in 65C24T8 to set the width of the stack register
func opSWS(s *State, line []uint8, opcode opcode) {
	s.sWidth = s.rWidth;
}
