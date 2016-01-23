package main

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"time"
)

func main() {
	var prev []*PlaylistEntry
	for {
		list, err := getPlaylist(base)
		if err != nil {
			panic(err)
		}

		if len(list) == 0 {
			return
		}

		sort.Sort(playlistSort(list))

		start := 0
		if len(prev) != 0 {
			for i, e := range list {
				if e.Name == prev[len(prev)-1].Name {
					start = i + 1
					break
				}
			}
			if start == len(list) {
				break
			}
			for _, e := range list {
				for _, p := range prev {
					if e.Name == p.Name {
						e.offset, e.frames = p.offset, p.frames
					}
				}
			}
		}
		prev = list

		err = playMovies(base, list, start)
		if err != nil {
			panic(err)
		}
	}
}

func playMovies(base string, list []*PlaylistEntry, start int) error {
	err := initRenderer(list)
	if err != nil {
		return err
	}
	defer closeRenderer()

	seekStart := 0

	for j := start; j < len(list); j++ {
		e := list[j]
		if !strings.HasSuffix(e.Name, ".cmv") || e.Type != "file" {
			continue
		}
		r, err := getCMVReader(base, e)
		if err != nil {
			return err
		}
		if len(e.offset) > len(r.offset) {
			r.offset = e.offset
		}
		found := make(chan []int64, 1)
		go r.findOffsets(r.offset, found)
		err = beginMovie(e, &r.Header)
		if err != nil {
			return err
		}
		start := time.Now()
		t := time.NewTicker(r.Header.FrameTime())
		for i := seekStart; ; i++ {
			if frames, err := r.Frames(i); err == io.EOF {
				break
			} else if err != nil {
				return err
			} else {
				if e.frames < i*200+len(frames) {
					e.frames = i*200 + len(frames)
				}
			nextFrame:
				for fi, f := range frames {
					start = start.Add(r.Header.FrameTime())
					if time.Since(start) > r.Header.FrameTime() {
						continue
					}
					err = displayFrame(i*200+fi, e.frames, f)
					if err != nil {
						return err
					}
					select {
					case s := <-seek:
						seekStart = s.Block
						if s.Entry != e {
							for k := 0; k < len(list); k++ {
								if list[k] == s.Entry {
									j = k - 1
									goto next
								}
							}
							fmt.Println("could not find", s.Entry.Name)
							continue
						}
						i = seekStart - 1
						break nextFrame
					case <-t.C:
					case r.offset = <-found:
						last, err := r.Frames(len(r.offset) - 2)
						if err != nil {
							return err
						}
						e.frames = (len(r.offset)-2)*200 + len(last)
					}
				}
			}
		}
		seekStart = 0
	next:
		e.offset = r.offset
		t.Stop()
		r.Close()
	}
	return nil
}

type SeekInfo struct {
	Entry *PlaylistEntry
	Block int
}

var seek = make(chan SeekInfo)

type playlistSort []*PlaylistEntry

func (s playlistSort) Len() int           { return len(s) }
func (s playlistSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s playlistSort) Less(i, j int) bool { return s[i].Mod.Time.Before(s[j].Mod.Time) }

type PlaylistEntry struct {
	Name   string    `json:"name"`
	Type   string    `json:"type"`
	Mod    NginxTime `json:"mtime"`
	Size   int64     `json:"size"`
	offset []int64
	frames int
}

type NginxTime struct {
	time.Time
}

func (t NginxTime) MarshalJSON() ([]byte, error) {
	return []byte(t.Format(`"` + time.RFC1123 + `"`)), nil
}

func (t *NginxTime) UnmarshalJSON(data []byte) (err error) {
	t.Time, err = time.Parse(`"`+time.RFC1123+`"`, string(data))
	return
}

type ReaderAtCloser interface {
	io.ReaderAt
	io.Closer
}

type CMVReader struct {
	r      ReaderAtCloser
	Header CMVHeader
	Sounds *CMVSounds
	offset []int64
}

// CMVHeader is the CMV header.
type CMVHeader struct {
	// Version is either 10000 (0x2710) or 10001 (0x2701). The latter
	// includes Sounds.
	Version uint32
	// Width is the number of columns in each frame of the CMV.
	Width uint32
	// Height is the number of rows in each frame of the CMV.
	Height uint32
	// FrameTicks is the frame rate in hundredths of a second per frame.
	// It can be 0, in which case the default frame rate is used.
	FrameTicks uint32
}

// FrameTime converts FrameTicks to a duration. It uses a default of 50 fps if
// FrameTicks is zero.
func (h *CMVHeader) FrameTime() time.Duration {
	raw := h.FrameTicks
	if raw == 0 {
		raw = 2
	}
	return time.Duration(raw) * time.Second / 100
}

// CMVSounds is the CMV 10001 sounds header, used by the intro videos for Dwarf
// Fortress.
type CMVSounds struct {
	Files  []string
	Timing [200][16]uint32
}

func getCMVReader(base string, e *PlaylistEntry) (*CMVReader, error) {
	r, err := getReaderAt(base, e)
	if err != nil {
		return nil, err
	}

	cmv := &CMVReader{r: r}

	var offs int64
	size := int64(binary.Size(&cmv.Header))
	err = binary.Read(io.NewSectionReader(r, offs, size), binary.LittleEndian, &cmv.Header)
	if err != nil {
		r.Close()
		return nil, err
	}
	offs += size

	if cmv.Header.Version < 10000 || cmv.Header.Version > 10001 {
		r.Close()
		return nil, fmt.Errorf("cmv: unhandled version %d", cmv.Header.Version)
	}

	if cmv.Header.Version >= 10001 {
		cmv.Sounds = new(CMVSounds)

		var n uint32
		size = int64(binary.Size(&n))
		err = binary.Read(io.NewSectionReader(r, offs, size), binary.LittleEndian, &n)
		if err != nil {
			r.Close()
			return nil, err
		}
		offs += size

		cmv.Sounds.Files = make([]string, n)

		var buf [50]byte
		size = 50
		for i := range cmv.Sounds.Files {
			_, err = io.ReadFull(io.NewSectionReader(r, offs, size), buf[:])
			if err != nil {
				r.Close()
				return nil, err
			}
			offs += size
			cmv.Sounds.Files[i] = string(buf[:bytes.IndexByte(buf[:], 0)])
		}

		size = int64(binary.Size(&cmv.Sounds.Timing))
		err = binary.Read(io.NewSectionReader(r, offs, size), binary.LittleEndian, &cmv.Sounds.Timing)
		if err != nil {
			r.Close()
			return nil, err
		}
		offs += size
	}

	cmv.offset = []int64{offs}

	return cmv, nil
}

type CMVFrame struct {
	h *CMVHeader
	b []byte
}

func (f *CMVFrame) Width() int          { return int(f.h.Width) }
func (f *CMVFrame) Height() int         { return int(f.h.Height) }
func (f *CMVFrame) Glyph(x, y int) byte { return f.b[f.Height()*x+y] }
func (f *CMVFrame) Color(x, y int) byte { return f.b[f.Height()*x+y+f.Width()*f.Height()] }
func (f *CMVFrame) Fg(x, y int) (byte, bool) {
	c := f.Color(x, y)
	return c & 7, c&(1<<6) != 0
}
func (f *CMVFrame) Bg(x, y int) byte {
	return (f.Color(x, y) >> 3) & 7
}

func (cmv *CMVReader) Frames(index int) ([]*CMVFrame, error) {
	const size = 4
	for len(cmv.offset) < index+2 {
		var l uint32
		err := binary.Read(io.NewSectionReader(cmv.r, cmv.offset[len(cmv.offset)-1], size), binary.LittleEndian, &l)
		if err != nil {
			return nil, err
		}
		cmv.offset = append(cmv.offset, cmv.offset[len(cmv.offset)-1]+size+int64(l))
	}

	z, err := zlib.NewReader(io.NewSectionReader(cmv.r, cmv.offset[index]+size, cmv.offset[index+1]-cmv.offset[index]-size))
	if err != nil {
		return nil, err
	}
	defer z.Close()

	b, err := ioutil.ReadAll(z)
	if err != nil {
		return nil, err
	}

	frameSize := int(cmv.Header.Width) * int(cmv.Header.Height) * 2
	if len(b)%frameSize != 0 {
		return nil, io.ErrUnexpectedEOF
	}
	frames := make([]*CMVFrame, len(b)/frameSize)
	for i := range frames {
		frames[i] = &CMVFrame{
			h: &cmv.Header,
			b: b[frameSize*i : frameSize*(i+1)],
		}
	}
	return frames, nil
}

func (cmv *CMVReader) findOffsets(offset []int64, found chan<- []int64) {
	// make sure we aren't modifying the extra capacity of the original
	// offset slice.
	offset = append([]int64(nil), offset...)
	const size = 4
	for {
		var l uint32
		err := binary.Read(io.NewSectionReader(cmv.r, offset[len(offset)-1], size), binary.LittleEndian, &l)
		if err == io.EOF {
			found <- offset
			return
		}
		if err != nil {
			return
		}
		offset = append(offset, offset[len(offset)-1]+size+int64(l))
	}
}

func (cmv *CMVReader) Close() error {
	return cmv.r.Close()
}
