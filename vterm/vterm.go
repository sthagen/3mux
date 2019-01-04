/*
Package vterm provides a layer of abstraction between a channel of incoming text (possibly containing ANSI escape codes, et al) and a channel of outbound Char's.

A Char is a character printed using a given cursor (which is stored alongside the Char).
*/
package vterm

import (
	"sync"

	"github.com/aaronduino/i3-tmux/cursor"
)

// Char represents one character in the terminal's grid
type Char struct {
	Rune rune

	Cursor cursor.Cursor
}

/*
VTerm acts as a virtual terminal emulator between a shell and the host terminal emulator

It both transforms an inbound stream of bytes into Char's and provides the option of dumping all the Char's that need to be rendered to display the currently visible terminal window from scratch.
*/
type VTerm struct {
	w, h int

	buffer      [][]Char
	bufferMutux *sync.Mutex

	cursor cursor.Cursor

	in  <-chan rune
	out chan<- Char

	storedCursorX, storedCursorY int

	blinker *Blinker
}

// NewVTerm returns a VTerm ready to be used by its exported methods
func NewVTerm(in <-chan rune, out chan<- Char) *VTerm {
	w := 30
	h := 30

	buffer := [][]Char{}
	for j := 0; j < h; j++ {
		row := []Char{}
		for i := 0; i < w; i++ {
			row = append(row, Char{
				Rune:   0,
				Cursor: cursor.Cursor{X: i, Y: j},
			})
		}
		buffer = append(buffer, row)
	}

	return &VTerm{
		w:           w,
		h:           h,
		buffer:      buffer,
		bufferMutux: &sync.Mutex{},
		cursor:      cursor.Cursor{X: 0, Y: 0},
		in:          in,
		out:         out,
		blinker:     newBlinker(),
	}
}

// Reshape safely updates a VTerm's width & height
func (v *VTerm) Reshape(w, h int) {
	v.bufferMutux.Lock()
	v.w = w
	v.h = h
	v.bufferMutux.Unlock()
}

// RedrawWindow draws the entire visible window from scratch, sending the Char's to the scheduler via the out channel
func (v *VTerm) RedrawWindow() {
	v.bufferMutux.Lock()

	verticalArea := v.h
	if v.h > len(v.buffer) {
		verticalArea = len(v.buffer)
	}

	for _, row := range v.buffer[len(v.buffer)-verticalArea:] {
		for _, char := range row {
			// truncate characters past the width
			if char.Cursor.X > v.w {
				break
			}

			if char.Rune != 0 {
				v.out <- char
			}
		}
	}

	v.bufferMutux.Unlock()
}