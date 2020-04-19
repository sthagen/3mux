package main

import (
	"time"

	"github.com/aaronjanse/3mux/ecma48"
)

type inputState struct {
	mouseDown bool
}

const demoMode = false

var demoTextTimer *time.Timer = nil
var demoTextDuration = 1000 * time.Millisecond

// handleInput puts the input through a series of switches and seive functions.
// When something acts on the event, we stop passing it downstream
func handleInput(human string, obj ecma48.Output) {
	defer func() {
		if config.statusBar {
			debug(root.serialize())
		}
	}()

	if demoMode {
		renderer.DemoText = human

		if demoTextTimer == nil {
			demoTextTimer = time.NewTimer(demoTextDuration)
		} else {
			demoTextTimer.Stop()
			demoTextTimer.Reset(demoTextDuration)
		}

		go func() {
			<-demoTextTimer.C
			renderer.DemoText = ""
		}()
	}

	if seiveTmuxEvents(human, obj) {
		return
	}

	if seiveResizeEvents(human, obj) {
		return
	}

	if seiveMouseEvents(human, obj) {
		return
	}

	if seiveConfigEvents(human) {
		return
	}

	// log.Printf("%q %+v", obj.Raw, obj.Parsed)
	t := getSelection().getContainer().(*Pane)

	switch x := obj.Parsed.(type) {
	case ecma48.CursorMovement:
		switch x.Direction {
		case ecma48.Up:
			t.handleStdin("\x1bOA")
		case ecma48.Down:
			t.handleStdin("\x1bOB")
		case ecma48.Right:
			t.handleStdin("\x1bOC")
		case ecma48.Left:
			t.handleStdin("\x1bOD")
		}
	default:
		// if we didn't find anything special, just pass the raw data to the selected terminal
		t.handleStdin(string(obj.Raw))
	}

}

var tmuxMode = false

func seiveTmuxEvents(human string, obj ecma48.Output) bool {
	if human == "Ctrl+B" {
		tmuxMode = true
		return true
	}

	if tmuxMode {
		switch string(obj.Raw) {
		case "%":
			pane := getSelection().getContainer().(*Pane)

			parent, _ := getSelection().getParent()
			parent.elements[parent.selectionIdx].contents = &Split{
				verticallyStacked: true,
				selectionIdx:      0,
				elements: []Node{Node{
					size:     1,
					contents: pane,
				}},
			}

			root.AddPane()
			root.simplify()
			root.refreshRenderRect()
		case "\"":
			pane := getSelection().getContainer().(*Pane)

			parent, _ := getSelection().getParent()
			parent.elements[parent.selectionIdx].contents = &Split{
				verticallyStacked: false,
				selectionIdx:      0,
				elements: []Node{Node{
					size:     1,
					contents: pane,
				}},
			}

			root.AddPane()
			root.simplify()
			root.refreshRenderRect()
		case "{":
			moveWindow(Left)
		case "}":
			moveWindow(Right)
		case "o": // next pane
			path := getSelection()
			oldTerm := path.getContainer().(*Pane)
			oldTerm.selected = false
			for {
				if len(path) == 1 {
					// select the first terminal
					for {
						done := false
						switch c := path.getContainer().(type) {
						case *Pane:
							done = true
						case *Split:
							c.selectionIdx = 0
							path = append(path, 0)
						}
						if done {
							break
							root.simplify()
						}
					}
					break
				}
				parent, _ := path.getParent()
				if parent.selectionIdx == len(parent.elements)-1 {
					path = path[:len(path)-1]
				} else {
					parent.selectionIdx++
					for {
						done := false
						switch c := path.getContainer().(type) {
						case *Pane:
							done = true
						case *Split:
							c.selectionIdx = 0
							path = append(path, 0)
						}
						if done {
							break
						}
					}
					break
				}
			}
			// select the new Term
			newTerm := getSelection().getContainer().(*Pane)
			newTerm.selected = true
			newTerm.vterm.RefreshCursor()
			root.refreshRenderRect()
		case ";": // prev pane
			path := getSelection()
			oldTerm := path.getContainer().(*Pane)
			oldTerm.selected = false
			for {
				if len(path) == 1 {
					// select the first terminal
					for {
						done := false
						switch c := path.getContainer().(type) {
						case *Pane:
							done = true
						case *Split:
							c.selectionIdx = len(c.elements) - 1
							path = append(path, 0)
						}
						if done {
							break
						}
					}
					break
				}
				parent, _ := path.getParent()
				if parent.selectionIdx == 0 {
					path = path[:len(path)-1]
				} else {
					parent.selectionIdx--
					for {
						done := false
						switch c := path.getContainer().(type) {
						case *Pane:
							done = true
						case *Split:
							c.selectionIdx = len(c.elements) - 1
							path = append(path, len(c.elements)-1)
						}
						if done {
							break
						}
					}
					break
				}
			}
			// select the new Term
			newTerm := getSelection().getContainer().(*Pane)
			newTerm.selected = true
			newTerm.vterm.RefreshCursor()
			root.refreshRenderRect()
		}
		tmuxMode = false
		return true
	}

	return false
}

func seiveResizeEvents(human string, obj ecma48.Output) bool {
	if resizeMode {
		switch human {
		case "Up", "k":
			resizeWindow(Up, 0.1)
		case "Down", "j":
			resizeWindow(Down, 0.1)
		case "Left", "h":
			resizeWindow(Left, 0.1)
		case "Right", "l":
			resizeWindow(Right, 0.1)
		default:
			resizeMode = false
		}
		return true
	}
	return false
}

var mouseDownPath Path
var mouseDownX, mouseDownY int

// seiveMouseEvents processes mouse events and returns true if the data should *not* be passed downstream
func seiveMouseEvents(human string, obj ecma48.Output) bool {
	switch ev := obj.Parsed.(type) {
	case ecma48.MouseDown:
		root.SelectAtCoords(ev.X, ev.Y)
		mouseDownX = ev.X
		mouseDownY = ev.Y
	case ecma48.MouseUp:
		root.DragBorder(mouseDownX, mouseDownY, ev.X, ev.Y)
	case ecma48.MouseDrag:
		// do nothing
	case ecma48.ScrollUp:
		t := getSelection().getContainer().(*Pane)
		t.vterm.ScrollbackDown()
	case ecma48.ScrollDown:
		t := getSelection().getContainer().(*Pane)
		t.vterm.ScrollbackUp()
	default:
		return false
	}

	return true
}
