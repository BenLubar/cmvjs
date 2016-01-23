// +build js

package main

import (
	"sync"

	"github.com/gopherjs/gopherjs/js"
)

var colors = [2][8][3]int{
	{
		{0, 0, 0},
		{0, 0, 128},
		{0, 128, 0},
		{0, 128, 128},
		{128, 0, 0},
		{128, 0, 128},
		{128, 128, 0},
		{192, 192, 192},
	},
	{
		{128, 128, 128},
		{0, 0, 255},
		{0, 255, 0},
		{0, 255, 255},
		{255, 0, 0},
		{255, 0, 255},
		{255, 255, 0},
		{255, 255, 255},
	},
}

var tileset struct {
	Width  int
	Height int
	Tiles  [256]*js.Object
}

var container *js.Object
var canvas *js.Object
var playlist *js.Object
var progress *js.Object
var seekLock = sync.NewCond(new(sync.Mutex))
var seeking bool

func initRenderer(list []*PlaylistEntry) error {
	img := js.Global.Get("Image").New()
	img.Set("src", "curses_800x600.png")
	ch := make(chan struct{})
	img.Set("onload", func() {
		go func() {
			canvas := js.Global.Get("document").Call("createElement", "canvas")
			canvas.Set("width", img.Get("width"))
			tileset.Width = img.Get("width").Int() / 16
			canvas.Set("height", img.Get("height"))
			tileset.Height = img.Get("height").Int() / 16
			ctx := canvas.Call("getContext", "2d")
			ctx.Call("drawImage", img, 0, 0)
			i := 0
			for y := 0; y < 16; y++ {
				for x := 0; x < 16; x++ {
					tileset.Tiles[i] = ctx.Call("getImageData", x*tileset.Width, y*tileset.Height, tileset.Width, tileset.Height)
					i++
				}
			}
			ch <- struct{}{}
		}()
	})
	<-ch

	container = js.Global.Get("document").Call("createElement", "div")
	js.Global.Get("document").Get("body").Call("appendChild", container)
	canvas = js.Global.Get("document").Call("createElement", "canvas")
	container.Call("appendChild", canvas)
	container.Call("appendChild", js.Global.Get("document").Call("createElement", "br"))
	playlist = js.Global.Get("document").Call("createElement", "select")
	container.Call("appendChild", playlist)
	for i, e := range list {
		entry := js.Global.Get("document").Call("createElement", "option")
		entry.Set("value", i)
		entry.Set("textContent", e.Name)
		entry.Call("setAttribute", "data-name", e.Name)
		playlist.Call("appendChild", entry)
	}
	playlist.Call("addEventListener", "change", func() {
		e := list[playlist.Get("value").Int()]
		go func() {
			seek <- SeekInfo{e, 0}
		}()
	}, false)
	progress = js.Global.Get("document").Call("createElement", "input")
	progress.Set("type", "range")
	container.Call("appendChild", progress)
	progress.Call("addEventListener", "mousedown", func() {
		go func() {
			seekLock.L.Lock()
			seeking = true
			seekLock.L.Unlock()
		}()
	}, false)
	progress.Call("addEventListener", "mouseup", func() {
		go func() {
			seekLock.L.Lock()
			seeking = false
			seekLock.Broadcast()
			seekLock.L.Unlock()
		}()
	}, false)
	progress.Call("addEventListener", "change", func() {
		e := list[playlist.Get("value").Int()]
		v := progress.Get("value").Int() / 200
		go func() {
			seek <- SeekInfo{e, v}
		}()
	}, false)

	return nil
}

func closeRenderer() error {
	js.Global.Get("document").Get("body").Call("removeChild", container)
	return nil
}

var ctx *js.Object
var imageData *js.Object

func beginMovie(e *PlaylistEntry, h *CMVHeader) error {
	canvas.Set("width", tileset.Width*int(h.Width))
	container.Get("style").Set("width", canvas.Get("width").String()+"px")
	canvas.Set("height", tileset.Height*int(h.Height))
	ctx = canvas.Call("getContext", "2d")
	imageData = ctx.Call("createImageData", canvas.Get("width"), canvas.Get("height"))
	playlist.Call("querySelector", "[data-name=\""+e.Name+"\"]").Set("selected", true)
	progress.Set("disabled", true)
	progress.Set("min", 0)
	progress.Set("max", 1)
	progress.Set("value", 0)
	return nil
}

func displayFrame(index, total int, f *CMVFrame) error {
	seekLock.L.Lock()
	if seeking {
		for seeking {
			seekLock.Wait()
		}
		seekLock.L.Unlock()
		return nil
	}
	seekLock.L.Unlock()

	progress.Set("max", total-1)
	progress.Set("value", index)
	progress.Set("disabled", false)

	for tx := 0; tx < f.Width(); tx++ {
		for ty := 0; ty < f.Height(); ty++ {
			t := tileset.Tiles[f.Glyph(tx, ty)]
			fg := colors[f.Color(tx, ty)>>6][f.Color(tx, ty)&7]
			bg := colors[0][f.Bg(tx, ty)]

			for x := 0; x < tileset.Width; x++ {
				for y := 0; y < tileset.Height; y++ {
					off := (x + y*tileset.Width) * 4
					r, g, b, a := t.Get("data").Index(off+0).Int(), t.Get("data").Index(off+1).Int(), t.Get("data").Index(off+2).Int(), t.Get("data").Index(off+3).Int()
					if r == 255 && g == 0 && b == 255 && a == 255 {
						r, g, b, a = 0, 0, 0, 0
					}
					off = ((x + tx*tileset.Width) + (y+ty*tileset.Height)*imageData.Get("width").Int()) * 4
					imageData.Get("data").SetIndex(off+0, (r*a*fg[0]/255+(255-a)*bg[0])/255)
					imageData.Get("data").SetIndex(off+1, (g*a*fg[1]/255+(255-a)*bg[1])/255)
					imageData.Get("data").SetIndex(off+2, (b*a*fg[2]/255+(255-a)*bg[2])/255)
					imageData.Get("data").SetIndex(off+3, 255)
				}
			}
		}
	}

	ctx.Call("putImageData", imageData, 0, 0)

	return nil
}
