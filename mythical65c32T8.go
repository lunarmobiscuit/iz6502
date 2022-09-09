package iz6502

/*
For the diffrences with NMOS6502 and CMOS65C02 see:
	https://github.com/lunarmobiscuit/verilog-65C2424-fsm
	https://github.com/lunarmobiscuit/verilog-65C24T8-fsm
*/

const (
	AB16 = 0x00
	AB24 = 0x40
	AB32 = 0x80
	AB48 = 0xC0

	R08 = 0x00
	R16 = 0x10
	R24 = 0x20
	R32 = 0x30

	N_THREADS = 8
)

// NewMythical65c24T8 returns an initialized (mythical) 65c24T8
func NewMythical65c24T8(m Memory) *State {
	var s State
	s.mem = m

	s.abWidth = AB24
	s.abMaxWidth = AB24

	var opcodes [256]opcode
	add65c02NOPs(&opcodes)
	for i := 0; i < 256; i++ {
		opcodes[i] = opcodesNMOS6502[i]
		rockwell := ((i & 0x07) == 0x07) || ((i & 0x0f) == 0x0f)
		if (opcodes65c02Delta[i].cycles != 0) && (rockwell == false) {
			opcodes[i] = opcodes65c02Delta[i]
		}
		if opcodes65c24T8Delta[i].cycles != 0 {
			opcodes[i] = opcodes65c24T8Delta[i]
		}
	}
	s.opcodes = &opcodes
	return &s
}

func (s *State) AddressWidth() uint8 {
	return s.abWidth
}

func (s *State) AddressMaxWidth() uint8 {
	return s.abMaxWidth
}

func (s *State) RegisterWidth() uint8 {
	return s.rWidth
}

func (s *State) RegisterMaxWidth() uint8 {
	return s.rMaxWidth
}


var opcodes65c24T8Delta = [256]opcode{
	// New address modes
	0x5c: {"JSR", 3, 6, false, modeIndirect, opJSR},

	// Functional difference
	0x0F: {"CPU", 1, 2, false, modeImplicit, opCPU},
	0x4F: {"A24", 1, 2, true, modeImplicit, opA24},
	0x1F: {"R16", 1, 2, true, modeImplicit, opR16},
	0x2F: {"R24", 1, 2, true, modeImplicit, opR24},
	0x5F: {"W16", 1, 2, true, modeImplicit, opW16},
	0x6F: {"W24", 1, 2, true, modeImplicit, opW24},
	0xFC: {"SWS", 1, 2, false, modeImplicit, opSWS},
	0x0B: {"SL8", 1, 2, false, modeImplicit, opSL8},
	0x1B: {"SR8", 1, 2, false, modeImplicit, opSR8},
	0x2B: {"XSL", 1, 2, false, modeImplicit, opXSL},
	0x3B: {"YSL", 1, 2, false, modeImplicit, opYSL},
	0x7B: {"LDA", 1, 9, false, modeX, buildOpLoad(regA)},
	0x8B: {"STA", 1, 9, false, modeX, buildOpStore(regA)},
	0x9B: {"CMP", 1, 9, false, modeX, buildOpCompare(regA)},
	0xAB: {"LDA", 1, 9, false, modeXY, buildOpLoad(regA)},
	0xBB: {"STA", 1, 9, false, modeXY, buildOpStore(regA)},
	0xCB: {"CMP", 1, 9, false, modeXY, buildOpCompare(regA)},
	0xDB: {"ADX", 1, 2, false, modeImplicit, opADX},
	0xEB: {"ADY", 1, 2, false, modeImplicit, opADY},
	0xFB: {"AXY", 1, 2, false, modeImplicit, opAXY},
}
