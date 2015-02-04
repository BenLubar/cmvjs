// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package main

var workerjs = js(asset.init(asset{Name: "worker.js", Content: "" +
	"\"use strict\";\n\nimportScripts('zlib.min.js');\n\nonmessage = function(e) {\n\tswitch (e.data.mode) {\n\tcase 'start':\n\t\tstartCMV(e.data.file);\n\t\tbreak;\n\tcase 'stop':\n\t\tstopCMV(e.data.file);\n\t\tbreak;\n\tcase 'position':\n\t\tvar movie = movies[e.data.file];\n\t\tif (movie) {\n\t\t\tif (e.data.force) {\n\t\t\t\tconsole.log(movie.path + ' seek request: ' + movie.frame + ' -> ' + e.data.position);\n\t\t\t}\n\t\t\tmovie.position = e.data.position;\n\t\t\tcmvProgress.call(movie, e.data.force);\n\t\t}\n\t\tbreak;\n\tdefault:\n\t\tthrow 'unknown mode ' + e.data.mode;\n\t}\n};\n\nvar movies = {};\n\nfunction startCMV(path) {\n\tvar movie = {\n\t\txhr:        new XMLHttpRequest(),\n\t\tpath:       path,\n\t\tversion:    null,\n\t\twidth:      null,\n\t\theight:     null,\n\t\tindex:      null,\n\t\ttoParse:    [],\n\t\tparseIndex: 0,\n\t\tframe:      -1,\n\t\tposition:   0,\n\t\tloaded:     0,\n\t\tdone:       0,\n\t\tkeyframes:  []\n\t};\n\tmovie.xhr.open('GET', path, true);\n\tmovie.xhr.overrideMimeType('text/plain; charset=x-user-defined');\n\tmovie.xhr.onprogress = function(e) {\n\t\tmovie.loaded = e.loaded;\n\t\tcmvProgress.call(movie);\n\t};\n\tmovie.xhr.onload = function() {\n\t\tmovie.done = 1;\n\t\tcmvProgress.call(movie);\n\t};\n\tmovie.xhr.send(null);\n\n\tmovies[path] = movie;\n}\n\nfunction stopCMV(path) {\n\tif (!(path in movies)) {\n\t\tconsole.log(path + ' already stopped');\n\t\treturn;\n\t}\n\tmovies[path].xhr.abort();\n\tdelete movies[path];\n}\n\nfunction cmvProgress(forcePosition) {\n\tif (this.version === null && this.loaded >= 4 * 1) {\n\t\tthis.version = uint32(this.xhr.responseText, 4 * 0);\n\t\tconsole.log(this.path + ' version: ' + this.version);\n\t\tif (this.version < 10000 || this.version > 10001) {\n\t\t\tthrow this.path + ' unsupported cmv version ' + this.version;\n\t\t}\n\t}\n\tif (this.width === null && this.loaded >= 4 * 2) {\n\t\tthis.width = uint32(this.xhr.responseText, 4 * 1);\n\t\tconsole.log(this.path + ' width: ' + this.width);\n\t}\n\tif (this.height === null && this.loaded >= 4 * 3) {\n\t\tthis.height = uint32(this.xhr.responseText, 4 * 2);\n\t\tconsole.log(this.path + ' height: ' + this.height);\n\t}\n\tif (forcePosition && (this.frame > this.position || Math.floor(this.position / 180000) != Math.floor(this.frame / 180000))) {\n\t\tvar keyframe = Math.floor(this.position / 180000);\n\t\tconsole.log(this.path + ' seeking: ' + this.position + ' (using keyframe ' + keyframe + ')');\n\t\tthis.index = this.keyframes[keyframe].index;\n\t\tthis.frame = keyframe * 180000 - 1;\n\t\tthis.toParse = [this.keyframes[keyframe].data];\n\t\tthis.parseIndex = 0;\n\t}\n\tif (this.index === null && this.loaded >= 4 * 5) {\n\t\tif (this.version >= 10001) {\n\t\t\t// skip sound information for now.\n\t\t\tvar i = 4 * 5 + uint32(this.xhr.responseText, 4 * 4) * 50 + 200 * 16 * 4;\n\t\t\tif (this.loaded >= i) {\n\t\t\t\tthis.index = i;\n\t\t\t\tconsole.log(this.path + ' finished header');\n\t\t\t}\n\t\t} else {\n\t\t\tthis.index = 4 * 4;\n\t\t\tconsole.log(this.path + ' finished header');\n\t\t}\n\t}\n\twhile (this.index !== null && this.loaded >= this.index + 4) {\n\t\tif (this.frame >= this.position + 180000 && this.done == 2) {\n\t\t\tbreak;\n\t\t}\n\n\t\tvar length = uint32(this.xhr.responseText, this.index);\n\t\tif (this.loaded >= this.index + 4 + length) {\n\t\t\tvar compressed = new Uint8Array(length);\n\t\t\tfor (var i = 0; i < length; i++) {\n\t\t\t\tcompressed[i] = this.xhr.responseText.charCodeAt(this.index + 4 + i) & 0xFF;\n\t\t\t}\n\t\t\tvar data = new Zlib.Inflate(compressed).decompress();\n\t\t\tthis.index += 4 + length;\n\t\t\tthis.toParse.push(data);\n\t\t\t//console.log(this.path + ' decompressed: ' + length + ' -> ' + data.length);\n\n\t\t\textractFrames.call(this);\n\t\t} else {\n\t\t\tbreak;\n\t\t}\n\n\t\t// allow event handling to run between iterations.\n\t\tsetTimeout(cmvProgress.bind(this), 0);\n\t\treturn;\n\t}\n\tif (this.done === 1) {\n\t\tthis.done = 2;\n\t\tpostMessage({file: this.path, done: this.frame});\n\t}\n}\n\nfunction extractFrames() {\n\tvar length = this.width * this.height * 2;\n\n\twhile (this.toParse.length) {\n\t\tif (this.frame >= this.position + 180000 && this.done == 2) {\n\t\t\treturn;\n\t\t}\n\n\t\tvar remaining = -this.parseIndex;\n\t\tthis.toParse.forEach(function(data) {\n\t\t\tremaining += data.length;\n\t\t});\n\t\tif (remaining < length) {\n\t\t\treturn;\n\t\t}\n\n\t\tif ((this.frame + 1) % 180000 == 0) {\n\t\t\tvar keyframe = (this.frame + 1) / 180000;\n\t\t\tif (!(keyframe in this.keyframes)) {\n\t\t\t\tvar parseIndex = this.parseIndex;\n\t\t\t\tvar toParse = this.toParse.slice(0);\n\n\t\t\t\tvar data = new Uint8Array(remaining);\n\t\t\t\tfor (var i = 0; i < remaining; i++) {\n\t\t\t\t\twhile (parseIndex >= toParse[0].length) {\n\t\t\t\t\t\ttoParse.shift();\n\t\t\t\t\t\tparseIndex = 0;\n\t\t\t\t\t}\n\t\t\t\t\tdata[i] = toParse[0][parseIndex];\n\t\t\t\t\tparseIndex++;\n\t\t\t\t}\n\n\t\t\t\tconsole.log(this.path + ' adding keyframe: ' + keyframe + ' (index = ' + this.index + ', ' + remaining + ' bytes)');\n\t\t\t\tthis.keyframes[keyframe] = {\n\t\t\t\t\tdata:  data,\n\t\t\t\t\tindex: this.index,\n\t\t\t\t};\n\t\t\t}\n\t\t}\n\n\t\tvar frame = new Uint8Array(length);\n\t\tfor (var i = 0; i < length; i++) {\n\t\t\twhile (this.parseIndex >= this.toParse[0].length) {\n\t\t\t\tthis.toParse.shift();\n\t\t\t\tthis.parseIndex = 0;\n\t\t\t}\n\t\t\tframe[i] = this.toParse[0][this.parseIndex];\n\t\t\tthis.parseIndex++;\n\t\t}\n\t\tthis.frame++;\n\t\tif (this.frame >= this.position && this.frame < this.position + 180000) {\n\t\t\tpostMessage({frame: {\n\t\t\t\tdata: frame,\n\t\t\t\twidth: this.width,\n\t\t\t\theight: this.height,\n\t\t\t\tindex: this.frame\n\t\t\t}, file: this.path});\n\t\t} else if (this.done == 0) {\n\t\t\tpostMessage({loaded: this.frame, file: this.path});\n\t\t}\n\t}\n\tthis.parseIndex = 0;\n}\n\nfunction uint32(data, off) {\n\treturn ((data.charCodeAt(off + 0) & 0xFF) <<  0) +\n\t       ((data.charCodeAt(off + 1) & 0xFF) <<  8) +\n\t       ((data.charCodeAt(off + 2) & 0xFF) << 16) +\n\t       ((data.charCodeAt(off + 3) & 0xFF) << 24);\n}\n" +
	""}))
