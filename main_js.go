// +build js

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gopherjs/gopherjs/js"
)

var base = js.Global.Get("location").Get("hash").String()[1:]

func getPlaylist(base string) ([]*PlaylistEntry, error) {
	resp, err := http.Get(base)
	for err != nil {
		js.Global.Get("console").Call("error", base+": "+err.Error())
		time.Sleep(time.Second)
		resp, err = http.Get(base)
	}
	defer resp.Body.Close()

	var list []*PlaylistEntry

	err = json.NewDecoder(resp.Body).Decode(&list)
	if err != nil {
		return nil, err
	}
	cache := js.Global.Get("localStorage").Get("cmvjs-cache-" + base)
	if cache != js.Undefined {
		cache = js.Global.Get("JSON").Call("parse", cache)
		for _, e := range list {
			ce := cache.Get(e.Name)
			if ce != js.Undefined {
				err = json.Unmarshal([]byte(ce.Get("o").String()), &e.offset)
				if err != nil {
					return nil, err
				}
				err = json.Unmarshal([]byte(ce.Get("f").String()), &e.frames)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	return list, nil
}

func getReaderAt(base string, e *PlaylistEntry) (ReaderAtCloser, error) {
	return httpReaderAt(base + e.Name), nil
}

type httpReaderAt string

func (path httpReaderAt) ReadAt(b []byte, off int64) (n int, err error) {
	req, err := http.NewRequest("GET", string(path), nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", off, off+int64(len(b))-1))
	resp, err := http.DefaultClient.Do(req)
	for err != nil {
		js.Global.Get("console").Call("error", string(path)+": "+err.Error())
		time.Sleep(time.Second)
		resp, err = http.DefaultClient.Do(req)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		_, err = io.CopyN(ioutil.Discard, resp.Body, off)
		if err != nil {
			return 0, err
		}
		return io.ReadFull(resp.Body, b)
	}
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		return 0, io.EOF
	}
	if resp.StatusCode != http.StatusPartialContent {
		return 0, fmt.Errorf("http: %s", resp.Status)
	}

	return io.ReadFull(resp.Body, b)
}

func (path httpReaderAt) Close() error {
	return nil
}

func updateCache(base string, e *PlaylistEntry) {
	cache := js.Global.Get("localStorage").Get("cmvjs-cache-" + base)
	if cache != js.Undefined {
		cache = js.Global.Get("JSON").Call("parse", cache)
	} else {
		cache = js.Global.Get("Object").New()
	}
	ce := js.Global.Get("Object").New()
	b, _ := json.Marshal(&e.offset)
	ce.Set("o", string(b))
	b, _ = json.Marshal(&e.frames)
	ce.Set("f", string(b))
	cache.Set(e.Name, ce)
	js.Global.Get("localStorage").Set("cmvjs-cache-"+base, js.Global.Get("JSON").Call("stringify", cache))
}
