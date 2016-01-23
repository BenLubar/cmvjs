// +build !js

package main

import (
	"os"
	"path/filepath"
)

var base = func() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}()

func getPlaylist(base string) ([]*PlaylistEntry, error) {
	dir, err := os.Open(base)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	files, err := dir.Readdir(0)
	if err != nil {
		return nil, err
	}

	var list []*PlaylistEntry
	for _, fi := range files {
		t := "file"
		if fi.IsDir() {
			t = "dir"
		}
		list = append(list, &PlaylistEntry{
			Name: fi.Name(),
			Type: t,
			Mod:  NginxTime{fi.ModTime()},
			Size: fi.Size(),
		})
	}

	return list, nil
}

func getReaderAt(base string, e *PlaylistEntry) (ReaderAtCloser, error) {
	return os.Open(filepath.Join(base, e.Name))
}
