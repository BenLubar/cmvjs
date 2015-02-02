//go:generate -command asset go run asset.go
//go:generate asset -var indexhtml index.html
//go:generate asset -var mainjs main.js
//go:generate asset -var workerjs worker.js
//go:generate asset -var zlibjs zlib.min.js
//go:generate asset -var cursespng curses_800x600.png

package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const movieDir = "/home/ben/df_linux/data/movies/"

func html(a asset) asset { return a }
func png(a asset) asset  { return a }
func js(a asset) asset   { return a }

func main() {
	movies := http.FileServer(http.Dir(movieDir))
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

	http.HandleFunc("/last_record.cmv", func(w http.ResponseWriter, r *http.Request) {
		dir, err := os.Open(movieDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		files, err := dir.Readdir(0)

		dir.Close()

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var newest os.FileInfo
		for _, fi := range files {
			if fi.IsDir() {
				continue
			}
			if !strings.HasSuffix(fi.Name(), ".cmv") {
				continue
			}

			if newest == nil || fi.ModTime().After(newest.ModTime()) {
				newest = fi
			}
		}
		if newest == nil {
			http.Error(w, "no CMV found", http.StatusNotFound)
			return
		}

		f, err := os.Open(filepath.Join(dir.Name(), newest.Name()))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		noNewData := int(time.Since(newest.ModTime()) / time.Second)
		for {
			n, err := io.Copy(w, f)
			if err != nil {
				return
			}
			if n == 0 {
				noNewData++
				if noNewData > 30 {
					return
				}
			} else {
				noNewData = 0
			}
			time.Sleep(time.Second)
		}
	})

	// 3 = c, 13 = m, 22 = v
	if err := http.ListenAndServe(":31322", nil); err != nil {
		panic(err)
	}
}
