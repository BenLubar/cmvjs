// +build !js

package main

import (
	"os"

	"github.com/BenLubar/df2014/cp437"
	"github.com/nsf/termbox-go"
)

func initRenderer(list []*PlaylistEntry) error {
	err := termbox.Init()
	if err != nil {
		return err
	}

	go func() {
		for {
			e := termbox.PollEvent()
			switch e.Type {
			case termbox.EventError:
				termbox.Close()
				panic(e.Err)

			case termbox.EventInterrupt:
				return

			case termbox.EventResize:
				// ignore

			default:
				termbox.Close()
				os.Exit(0)
			}
		}
	}()

	return nil
}

func closeRenderer() error {
	termbox.Interrupt()
	termbox.Close()
	return nil
}

func beginMovie(e *PlaylistEntry, h *CMVHeader) error {
	return nil
}

var termboxColors = [...]termbox.Attribute{
	termbox.ColorBlack,
	termbox.ColorBlue,
	termbox.ColorGreen,
	termbox.ColorCyan,
	termbox.ColorRed,
	termbox.ColorMagenta,
	termbox.ColorYellow,
	termbox.ColorWhite,
}

func displayFrame(index, total int, f *CMVFrame) error {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)

	for y := 0; y < f.Height(); y++ {
		for x := 0; x < f.Width(); x++ {
			ch := cp437.Rune(f.Glyph(x, y))
			fg, bright := f.Fg(x, y)
			bg := f.Bg(x, y)
			if bright {
				termbox.SetCell(x, y, ch, termboxColors[fg]|termbox.AttrBold, termboxColors[bg])
			} else {
				termbox.SetCell(x, y, ch, termboxColors[fg], termboxColors[bg])
			}
		}
	}

	return termbox.Flush()
}
