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
	"net/http"
	"os"
	"strings"
	"time"
)

var movieDir = flag.String("dir", "/home/ben/df_linux/data/movies", "cmv directory")

func html(a asset) asset { return a }
func png(a asset) asset  { return a }
func js(a asset) asset   { return a }

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
	http.Handle("/main.js", mainjs)
	http.Handle("/worker.js", workerjs)
	http.Handle("/zlib.min.js", zlibjs)
	http.Handle("/curses_800x600.png", cursespng)

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

		type dirEnt struct {
			Name string
			Mod  time.Time
			Size int64
		}

		var data []dirEnt

		for _, fi := range info {
			data = append(data, dirEnt{
				Name: fi.Name(),
				Mod:  fi.ModTime(),
				Size: fi.Size(),
			})
		}

		_ = json.NewEncoder(w).Encode(&data)
	})

	// 3 = c, 13 = m, 22 = v
	if err := http.ListenAndServe(":31322", nil); err != nil {
		panic(err)
	}
}
