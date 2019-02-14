package main

import "fmt"

type textPages struct {
	pages [4]textPage
}

type textPage struct {
	dirty bool
	data  [256]uint8
}

func (p *textPage) peek(address uint8) uint8 {
	return p.data[address]
}

func (p *textPage) poke(address uint8, value uint8) {
	p.data[address] = value
	// Note: we could avoid setting dirty on the 16 blocks of 8 hidden bytes
	p.dirty = true
}

func (p *textPage) getData() *[256]uint8 {
	return &p.data
}

func textMemoryByteToString(value uint8) string {
	value = value & 0x7F
	if value < ' ' {
		return " "
	}

	return string(value)
}

func textMemoryByteToStringHex(value uint8) string {
	return fmt.Sprintf("%02x ", value)
}

func (tp *textPages) dump() {
	// See "Understand the Apple II", page 5-10
	// http://www.applelogic.org/files/UNDERSTANDINGTHEAII.pdf

	fmt.Println("------------------------------------------")

	var i, j, h uint8
	// Top, middle and botton screen
	for i = 0; i < 120; i = i + 40 {
		// Memory pages
		for _, p := range tp.pages {
			// The two half pages
			for _, h = range []uint8{0, 128} {
				line := "|"
				for j = i + h; j < i+h+40; j++ {
					line += textMemoryByteToString(p.peek(j))
				}
				fmt.Println(line + "|")
			}
		}
	}

	fmt.Println("------------------------------------------")

}

func (tp *textPages) dumpIfDirty() {
	dirty := false
	for i := 0; i < 4; i++ {
		if tp.pages[i].dirty {
			dirty = true
			tp.pages[i].dirty = false
		}
	}

	if !dirty {
		return
	}
	tp.dump()
}

func (tp *textPages) charAddress(column uint8, line uint8) (page uint8, address uint8) {
	page = (line / 3) % 4
	address = column + (line/8)*40 + (line%2)*128
	return
}

func (tp *textPages) read(column uint8, line uint8) uint8 {
	page, address := tp.charAddress(column, line)
	return tp.pages[page].peek(address)
}

func (tp *textPages) write(column uint8, line uint8, value uint8) {
	page, address := tp.charAddress(column, line)
	tp.pages[page].poke(address, value)
}
