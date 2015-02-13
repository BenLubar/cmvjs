//go:generate -command asset go run asset.go
//go:generate asset -var indexhtml index.html
//go:generate asset -var mainjs main.js
//go:generate asset -var workerjs worker.js
//go:generate asset -var zlibjs zlib.min.js
//go:generate asset -var cursespng curses_800x600.png

package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BenLubar/df2014"
	"github.com/davecheney/fswatch"
)

var movieDir = flag.String("dir", "/home/ben/df_linux/data/movies", "cmv directory")

func html(a asset) asset { http.Handle("/"+a.Name, a); return a }
func png(a asset) asset  { http.Handle("/"+a.Name, a); return a }
func js(a asset) asset   { http.Handle("/"+a.Name, a); return a }

type dirEnt struct {
	Name   string
	Mod    time.Time
	Size   int64
	Width  int `json:",omitempty"`
	Height int `json:",omitempty"`
	Frames int `json:",omitempty"`
}
type dirEnts []dirEnt

func (d dirEnts) Len() int           { return len(d) }
func (d dirEnts) Swap(i, j int)      { d[i], d[j] = d[j], d[i] }
func (d dirEnts) Less(i, j int) bool { return d[i].Mod.Before(d[j].Mod) }

func main() {
	movies := http.FileServer(http.Dir(*movieDir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			if strings.HasSuffix(r.URL.Path, ".cmv") {
				movies.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
			return
		}
		indexhtml.ServeHTTP(w, r)
	})

	var mtx sync.RWMutex
	deCache := make(map[dirEnt]dirEnt)

	http.HandleFunc("/movies.json", func(w http.ResponseWriter, r *http.Request) {
		dir, err := os.Open(*movieDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer dir.Close()

		info, err := dir.Readdir(0)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var data dirEnts

		mtx.RLock()
		for _, fi := range info {
			de := dirEnt{
				Name: fi.Name(),
				Mod:  fi.ModTime(),
				Size: fi.Size(),
			}
			if cached, ok := deCache[de]; ok {
				de = cached
			}
			data = append(data, de)
		}
		mtx.RUnlock()

		sort.Sort(data)

		_ = json.NewEncoder(w).Encode(&data)
	})

	go func() {
		updateFile := func(fn string, isRemove bool) {
			// clean up
			mtx.Lock()
			for de := range deCache {
				if de.Name == fn {
					delete(deCache, de)
				}
			}
			mtx.Unlock()

			if isRemove {
				return
			}

			f, err := os.Open(filepath.Join(*movieDir, fn))
			if err != nil {
				log.Println(err)
				return
			}
			stat, err := f.Stat()
			if err != nil {
				log.Println(err)
				f.Close()
				return
			}
			// StreamCMV closes the file.
			cmv, err := df2014.StreamCMV(f, 10)
			if err != nil {
				log.Println(err)
				return
			}

			key := dirEnt{
				Name: fn,
				Mod:  stat.ModTime(),
				Size: stat.Size(),
			}
			value := key

			value.Width = int(cmv.Header.Columns)
			value.Height = int(cmv.Header.Rows)
			for range cmv.Frames {
				value.Frames++
			}

			mtx.Lock()
			deCache[key] = value
			mtx.Unlock()
		}

		w, err := fswatch.Watch(*movieDir)
		if err != nil {
			log.Println(err)
			return
		}
		defer w.Close()

		func() {
			dir, err := os.Open(*movieDir)
			if err != nil {
				log.Println(err)
				return
			}
			defer dir.Close()

			names, err := dir.Readdirnames(0)
			if err != nil {
				log.Println(err)
				return
			}

			for _, fn := range names {
				updateFile(fn, false)
			}
		}()

		for e := range w.C {
			updateFile(e.Target, e.IsRemove())
		}
	}()

	// 3 = c, 13 = m, 22 = v
	if err := http.ListenAndServe(":31322", nil); err != nil {
		log.Fatalln(err)
	}
}
