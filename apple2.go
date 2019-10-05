package apple2

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ivanizag/apple2/core6502"
)

// Apple2 represents all the components and state of the emulated machine
type Apple2 struct {
	Name                string
	cpu                 *core6502.State
	mmu                 *memoryManager
	io                  *ioC0Page
	cg                  *CharacterGenerator
	cards               [8]card
	isApple2e           bool
	commandChannel      chan int
	cycleDurationNs     float64 // Current speed. Inverse of the cpu clock in Ghz
	isColor             bool
	fastMode            bool
	fastRequestsCounter int
}

const (
	// CpuClockMhz is the actual Apple II clock speed
	CpuClockMhz     = 14.318 / 14
	cpuClockEuroMhz = 14.238 / 14
)

const maxWaitDuration = 100 * time.Millisecond

// Run starts the Apple2 emulation
func (a *Apple2) Run() {
	// Start the processor
	a.cpu.Reset()
	referenceTime := time.Now()

	for {
		// Run a 6502 step
		a.cpu.ExecuteInstruction()

		// Execute meta commands
		commandsPending := true
		for commandsPending {
			select {
			case command := <-a.commandChannel:
				a.executeCommand(command)
			default:
				commandsPending = false
			}
		}

		if a.cycleDurationNs != 0 && a.fastRequestsCounter <= 0 {
			// Wait until next 6502 step has to run
			clockDuration := time.Since(referenceTime)
			simulatedDuration := time.Duration(float64(a.cpu.GetCycles()) * a.cycleDurationNs)
			waitDuration := simulatedDuration - clockDuration
			if waitDuration > maxWaitDuration || -waitDuration > maxWaitDuration {
				// We have to wait too long or are too much behind. Let's fast forward
				referenceTime = referenceTime.Add(-waitDuration)
				waitDuration = 0
			}
			if waitDuration > 0 {
				time.Sleep(waitDuration)
			}
		}
	}
}

const (
	// CommandToggleSpeed toggles cpu speed between full speed and actual Apple II speed
	CommandToggleSpeed = iota + 1
	// CommandToggleColor toggles between NTSC color TV and Green phospor monitor
	CommandToggleColor
	// CommandSaveState stores the state to file
	CommandSaveState
	// CommandLoadState reload the last state
	CommandLoadState
	// CommandDumpDebugInfo dumps usefull info
	CommandDumpDebugInfo
	// CommandNextCharGenPage cycles the CharGen page if several
	CommandNextCharGenPage
	// CommandToggleCPUTrace toggle tracing of CPU execution
	CommandToggleCPUTrace
)

// SendCommand enqueues a command to the emulator thread
func (a *Apple2) SendCommand(command int) {
	a.commandChannel <- command
}

func (a *Apple2) executeCommand(command int) {
	switch command {
	case CommandToggleSpeed:
		if a.cycleDurationNs == 0 {
			fmt.Println("Slow")
			a.cycleDurationNs = 1000.0 / CpuClockMhz
		} else {
			fmt.Println("Fast")
			a.cycleDurationNs = 0
		}
	case CommandToggleColor:
		a.isColor = !a.isColor
	case CommandSaveState:
		fmt.Println("Saving state")
		err := a.save("apple2.state")
		if err != nil {
			fmt.Printf("Error loadind state: %v.", err)
		}
	case CommandLoadState:
		fmt.Println("Loading state")
		err := a.load("apple2.state")
		if err != nil {
			fmt.Printf("Error loadind state: %v.", err)
		}
	case CommandDumpDebugInfo:
		a.dumpDebugInfo()
	case CommandNextCharGenPage:
		a.cg.nextPage()
		fmt.Printf("Chargen page %v\n", a.cg.page)
	case CommandToggleCPUTrace:
		a.cpu.SetTrace(!a.cpu.GetTrace())
	}
}

func (a *Apple2) requestFastMode() {
	// Note: if the fastMode is shorter than maxWaitDuration, there won't be any gain.
	if a.fastMode {
		a.fastRequestsCounter++
	}
}

func (a *Apple2) releaseFastMode() {
	if a.fastMode {
		a.fastRequestsCounter--
	}
}

type persistent interface {
	save(io.Writer) error
	load(io.Reader) error
}

func (a *Apple2) save(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	err = a.cpu.Save(w)
	if err != nil {
		return err
	}
	err = a.mmu.save(w)
	if err != nil {
		return err
	}
	err = a.io.save(w)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, a.isColor)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, a.fastMode)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, a.fastRequestsCounter)
	if err != nil {
		return err
	}

	for _, c := range a.cards {
		if c != nil {
			err = c.save(w)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Apple2) load(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	r := bufio.NewReader(f)

	err = a.cpu.Load(r)
	if err != nil {
		return err
	}
	err = a.mmu.load(r)
	if err != nil {
		return err
	}
	err = a.io.load(r)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &a.isColor)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &a.fastMode)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &a.fastRequestsCounter)
	if err != nil {
		return err
	}

	for _, c := range a.cards {
		if c != nil {
			err = c.load(r)
			if err != nil {
				return err
			}

		}
	}

	return nil
}

func (a *Apple2) dumpDebugInfo() {
	// See "Apple II Monitors Peeled"
	pageZeroSymbols := map[int]string{
		0x36: "CSWL",
		0x37: "CSWH",
		0x38: "KSWL",
		0x39: "KSWH",
	}

	fmt.Printf("Page zero values:\n")
	for _, k := range []int{0x36, 0x37, 0x38, 0x39} {
		d := a.mmu.physicalMainRAM.data[k]
		fmt.Printf("  %v(0x%x): 0x%02x\n", pageZeroSymbols[k], k, d)
	}
}
