package gui

import (
	"log"
	"runtime"
	"time"
)

// NewPageMenu xxx
func NewPageMenu(parent Window) *Menu {
	menu := NewMenu(parent)
	menu.addItem("About", func(name string) {
		log.Printf("This is the Palette Window System.")
		log.Printf("# of goroutines: %d", runtime.NumGoroutine())
	})
	menu.addItem("Move", func(name string) {
		log.Printf("Move callback")
	})
	menu.addItem("New Console", func(name string) {
		menu.startNewConsole()
	})
	menu.addItem("Delete", func(name string) {
		log.Printf("Delete callback")
	})
	menu.addItem("Tools ->", func(name string) {
		log.Printf("Tools callback")
	})
	menu.addItem("Misc  ->", func(name string) {
		log.Printf("Misc callback")
	})
	menu.addItem("Window ->", func(name string) {
		log.Printf("Misc callback")
	})
	return menu
}

func (menu *Menu) startNewConsole() {

	menu.OutChan <- SetCursorStyleCmd{sweepCursorStyle}
	// Take over the mouse to sweep out the console area

	newInput := PushWindowInput(menu)

	log.Printf("startNewConsole: go menu.sweep...()\n")

	go menu.sweepConsoleWindow(newInput)

}

func (menu *Menu) sweepConsoleWindow(input chan WinInput) {
	for {
		select {

		case inCmd := <-input:

			switch t := inCmd.(type) {
			case MouseCmd:
				switch t.Ddu {
				case MouseDown:
					log.Printf("!!!!!!!!!!!!!!!!!!sweepWindow: MouseDown pos=%v\n", t.Pos)
				case MouseDrag:
					log.Printf("!!!!!!!!!!!!!!!!!!sweepWindow: MouseDrag pos=%v\n", t.Pos)
				case MouseUp:
					log.Printf("!!!!!!!!!!!!!!!!!!sweepWindow: MouseUp pos=%v\n", t.Pos)
					PopWindowInput(menu)
				}
			}

		default:
			time.Sleep(time.Millisecond)
		}
	}

}
