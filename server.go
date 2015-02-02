package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))

	http.HandleFunc("/last_record.cmv", func(w http.ResponseWriter, r *http.Request) {
		dir, err := os.Open("/home/ben/df_linux/data/movies/")
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
