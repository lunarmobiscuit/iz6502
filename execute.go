package iz6502

import (
	"encoding/binary"
	"fmt"
	"io"
)

// https://www.masswerk.at/6502/6502_instruction_set.html
// http://www.emulator101.com/reference/6502-reference.html
// https://www.csh.rit.edu/~moffitt/docs/6502.html#FLAGS
// https://ia800509.us.archive.org/18/items/Programming_the_6502/Programming_the_6502.pdf

const (
	maxInstructionSize = 4
)

// State represents the state of the simulated device
type State struct {
	opcodes *[256]opcode
	trace   bool

	reg    registers
	mem    Memory
	cycles uint64

	// 24T8 state for current and maximum address and register widths
	wasPrefix	bool
	abWidth 	uint8
	abMaxWidth 	uint8
	rWidth 		uint8
	rMaxWidth 	uint8
	sWidth 		uint8

	extraCycleCrossingBoundaries bool
	extraCycleBranchTaken        bool
	extraCycleBCD                bool
	lineCache                    []uint8
	// We cache the allocation of a line to avoid a malloc per instruction. To be used only
	// by ExecuteInstruction(). 2x speedup on the emulation!!
}

const (
	vectorNMI   uint32 = 0xfffa
	vectorReset uint32 = 0xfffc
	vectorBreak uint32 = 0xfffe

	vector24NMI   uint32 = 0xfffff7
	vector24Reset uint32 = 0xfffffa
	vector24Break uint32 = 0xfffffd
)

type opcode struct {
	name        string
	bytes       uint16
	cycles      int
	isPrefix	bool
	addressMode int
	action      opFunc
}

type opFunc func(s *State, line []uint8, opcode opcode)

func (s *State) executeLine(line []uint8) {
	opcode := s.opcodes[line[0]]
	if opcode.cycles == 0 {
		panic(fmt.Sprintf("Unknown opcode 0x%02x\n", line[0]))
	}

	// 24T8 if the previous instruction is not a prefix code, switch back to 16/8 mode
	if (s.wasPrefix == false) && (opcode.isPrefix == false)  {
		s.abWidth = AB16;
		s.rWidth = R08;
	}
	s.wasPrefix = opcode.isPrefix

	opcode.action(s, line, opcode)
}

// ExecuteInstruction transforms the state given after a single instruction is executed.
func (s *State) ExecuteInstruction() {
	pc := s.reg.getPC()
	opcodeID := s.mem.PeekCode(pc)
	opcode := s.opcodes[opcodeID]

	if opcode.cycles == 0 {
		panic(fmt.Sprintf("Unknown opcode 0x%02x\n", opcodeID))
	}

	// 24T8 if the previous instruction is not a prefix code, switch back to 16/8 mode
	if (s.wasPrefix == false) && (opcode.isPrefix == false)  {
		s.abWidth = AB16;
		s.rWidth = R08;
	}
	s.wasPrefix = opcode.isPrefix

	if s.lineCache == nil {
		s.lineCache = make([]uint8, maxInstructionSize)
	}
	nBytes := opcode.bytes
	// 24T8 - add one more byte when an opcode has an address or a long branch
	if (s.abWidth == AB24) && ((nBytes >= 3) || (opcode.addressMode == modeRelative)) {
		nBytes += 1
	}
	if (opcode.addressMode == modeImmediate) && (s.rWidth != R08) {  // 24T8 - add more bytes for long immediates
		switch s.rWidth {
		case R16: nBytes += 1
		case R24: nBytes += 2
		}
	}
	for i := uint16(0); i < nBytes; i++ {
		s.lineCache[i] = s.mem.PeekCode(pc)
		pc++

		// 24T8 BACKWARD COMPATIBILITY - roll around the PC from $FFFF to $0000 if in 16-bit address mode
		if (s.abWidth == AB16) && (pc == 0x010000) {
			pc = 0x000000;
		}
	}
	s.reg.setPC(pc)

	if s.trace {
		//fmt.Printf("%#06x %#02x\n", pc-uint32(opcode.bytes), opcodeID)
		fmt.Printf("%#06x %-13s: ", pc-uint32(nBytes), lineString(s, s.lineCache, opcode))
	}
	opcode.action(s, s.lineCache, opcode)
	s.cycles += uint64(opcode.cycles)

	// Extra cycles
	if s.extraCycleBranchTaken {
		s.cycles++
		s.extraCycleBranchTaken = false
	}
	if s.extraCycleCrossingBoundaries {
		s.cycles++
		s.extraCycleCrossingBoundaries = false
	}
	if s.extraCycleBCD {
		s.cycles++
		s.extraCycleBCD = false
	}

	if s.trace {
		fmt.Printf("%v, [%02x] <w%x/%x>\n", s.reg, s.lineCache[0:opcode.bytes], s.abWidth, s.rWidth)
	}
}

// Reset resets the processor. Moves the program counter to the vector in 0xfffc (24T8 or 0xffffc )
func (s *State) Reset() {
	var startAddress uint32

	// 24T8 uses 3-byte RST/IRQ/NMI vectors
	switch (s.abMaxWidth) {
		case AB24:
			s.abWidth = s.abMaxWidth
			startAddress = get24Bits(s.mem, vector24Reset)
		default:
			startAddress = uint32(getWord(s.mem, vectorReset))
	}
	s.cycles += 6
	s.reg.setPC(startAddress)
}

// GetCycles returns the count of CPU cycles since last reset.
func (s *State) GetCycles() uint64 {
	return s.cycles
}

// SetTrace activates tracing of the cpu execution
func (s *State) SetTrace(trace bool) {
	s.trace = trace
}

// GetTrace gets trhe tracing state of the cpu execution
func (s *State) GetTrace() bool {
	return s.trace
}

// SetMemory changes the memory provider
func (s *State) SetMemory(mem Memory) {
	s.mem = mem
}

// GetPCAndSP returns the current program counter and stack pointer. Used to trace MLI calls
func (s *State) GetPCAndSP() (uint32, uint32) {
	return s.reg.getPC(), s.reg.getSP(s.sWidth)
}

// GetCarryAndAcc returns the value of the carry flag and the accumulator. Used to trace MLI calls
func (s *State) GetCarryAndAcc() (bool, uint32) {
	return s.reg.getFlag(flagC), s.reg.getA(s.rWidth)
}

// GetAXYP returns the value of the A, X, Y and P registers
func (s *State) GetAXYP() (uint32, uint32, uint32, uint8) {
	return s.reg.getA(s.rWidth), s.reg.getX(s.rWidth), s.reg.getY(s.rWidth), s.reg.getP()
}

// SetAXYP changes the value of the A, X, Y and P registers
func (s *State) SetAXYP(regA uint32, regX uint32, regY uint32, regP uint8) {
	s.reg.setA(s.rWidth, regA)
	s.reg.setX(s.rWidth, regX)
	s.reg.setY(s.rWidth, regY)
	s.reg.setP(regP)
}

// SetPC changes the program counter, as a JMP instruction
func (s *State) SetPC(pc uint32) {
	s.reg.setPC(pc)
}

// Save saves the CPU state (registers and cycle counter)
func (s *State) Save(w io.Writer) error {
	err := binary.Write(w, binary.BigEndian, s.cycles)
	if err != nil {
		return err
	}
	binary.Write(w, binary.BigEndian, s.reg.data)
	if err != nil {
		return err
	}
	return nil
}

// Load loads the CPU state (registers and cycle counter)
func (s *State) Load(r io.Reader) error {
	err := binary.Read(r, binary.BigEndian, &s.cycles)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &s.reg.data)
	if err != nil {
		return err
	}
	return nil
}
