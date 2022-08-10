package iz6502

import "io/ioutil"

// Memory represents the addressable space of the processor
type Memory interface {
	Peek(address uint32) uint8
	Poke(address uint32, value uint8)

	// PeekCode can bu used to optimize the memory manager to requests with more
	// locality. It must return the same as a call to Peek()
	PeekCode(address uint32) uint8
}

func getWord(m Memory, address uint32) uint16 {
	address = address & 0x0FFFF
	addressP1 := (address + 1) & 0x0FFFF
	return uint16(m.Peek(address)) | (uint16(m.Peek(addressP1)) << 8)
}

func getWordNoCrossPage(m Memory, address uint32) uint16 {
	addressMSB := address + 1
	if address&0xff == 0xff {
		// We won't cross the page bounday for the MSB byte
		addressMSB -= 0x100
	}
	return uint16(m.Peek(address)) | (uint16(m.Peek(addressMSB)) << 8)
}

func getZeroPageWord(m Memory, address uint32) uint16 {
	address = address & 0x0FF
	addressP1 := (address + 1) & 0x0FF
	return uint16(m.Peek(address)) | (uint16(m.Peek(addressP1)) << 8)
}

func get24Bits(m Memory, address uint32) uint32 {
	return uint32(m.Peek(address)) | (uint32(m.Peek(address + 1)) << 8) | (uint32(m.Peek(address + 2)) << 16)
}

func getZeroPage24Bits(m Memory, address uint32) uint32 {
	address = address & 0x0FF
	addressP1 := (address + 1) & 0x0FF
	addressP2 := (address + 2) & 0x0FF
	return uint32(m.Peek(address)) | (uint32(m.Peek(addressP1)) << 8) | (uint32(m.Peek(addressP2)) << 16)
}

// FlatMemory puts RAM on the 64Kb addressable by the processor
type FlatMemory struct {
	data [65536]uint8
}

// Peek returns the data on the given address
func (m *FlatMemory) Peek(address uint32) uint8 {
	if int(address) >= len(m.data) { return 0xff }
	return m.data[address]
}

// PeekCode returns the data on the given address
func (m *FlatMemory) PeekCode(address uint32) uint8 {
	if int(address) >= len(m.data) { return 0xff }
	return m.data[address]
}

// Poke sets the data at the given address
func (m *FlatMemory) Poke(address uint32, value uint8) {
	if int(address) >= len(m.data) { return }
	m.data[address] = value
}

func (m *FlatMemory) loadBinary(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	for i, v := range bytes {
		m.Poke(uint32(i), uint8(v))
	}

	return nil
}

// FlatMemory puts RAM on the 64Kb addressable by the processor
type Flat256KMemory struct {
	data [262144]uint8
}

// Peek returns the data on the given address
func (m *Flat256KMemory) Peek(address uint32) uint8 {
	address &= 0x3FFFF
	return m.data[address]
}

// PeekCode returns the data on the given address
func (m *Flat256KMemory) PeekCode(address uint32) uint8 {
	address &= 0x3FFFF
	return m.data[address]
}

// Poke sets the data at the given address
func (m *Flat256KMemory) Poke(address uint32, value uint8) {
	address &= 0x3FFFF
	m.data[address] = value
}

func (m *Flat256KMemory) loadBinary(filename string) error {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	for i, v := range bytes {
		m.Poke(uint32(i), uint8(v))
	}

	return nil
}
