// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package main

var mainjs = js(asset.init(asset{Name: "main.js", Content: "" +
	"var CMV = function() {\n\t\"use strict\";\n\n\tvar worker = new Worker('worker.js');\n\tvar movies = {};\n\tworker.onmessage = function(e) {\n\t\tvar movie = movies[e.data.file];\n\t\tif (!movie) {\n\t\t\tconsole.log('no movie for ' + e.data.file);\n\t\t\treturn;\n\t\t}\n\t\tvar frame = e.data.frame;\n\t\tif (frame) {\n\t\t\tvar prev = movie.frames[frame.index - 1];\n\t\t\tfor (var i = 0; prev != null && i < prev.data.length; i++) {\n\t\t\t\tif (frame.data[i] != prev.data[i]) {\n\t\t\t\t\tprev = null;\n\t\t\t\t}\n\t\t\t}\n\t\t\tif (prev != null) {\n\t\t\t\tframe.data = prev.data;\n\t\t\t}\n\t\t\tmovie.frames[frame.index] = frame;\n\t\t\tmovie.loaded = Math.max(movie.loaded, frame.index);\n\t\t\tmovie.notify.forEach(function(f) {\n\t\t\t\tf(frame, movie);\n\t\t\t});\n\t\t}\n\t\tif ('loaded' in e.data) {\n\t\t\tmovie.loaded = Math.max(movie.loaded, e.data.loaded);\n\t\t}\n\t\tif ('done' in e.data) {\n\t\t\tmovie.done = e.data.done;\n\t\t}\n\t}\n\n\tfunction startStream(path, callback) {\n\t\tvar movie = movies[path];\n\t\tif (movie) {\n\t\t\tconsole.log('stream already started for ' + path);\n\t\t\tmovie.notify.push(callback);\n\t\t\tmovie.frames.forEach(function(frame) {\n\t\t\t\tcallback(frame, movie);\n\t\t\t});\n\t\t\treturn;\n\t\t}\n\t\tconsole.log('starting stream for ' + path);\n\t\tmovie = {\n\t\t\tloaded: -1,\n\t\t\tframes: [],\n\t\t\tnotify: [callback],\n\t\t\tpath: path,\n\t\t\tseek: function(tick, force) {\n\t\t\t\tif (force) {\n\t\t\t\t\tmovie.frames = [];\n\t\t\t\t}\n\t\t\t\tworker.postMessage({\n\t\t\t\t\tfile: path,\n\t\t\t\t\tmode: 'position',\n\t\t\t\t\tposition: tick,\n\t\t\t\t\tforce: force\n\t\t\t\t});\n\t\t\t}\n\t\t};\n\t\tmovies[path] = movie;\n\t\tworker.postMessage({file: path, mode: 'start'});\n\t}\n\n\tfunction stopStream(path, callback) {\n\t\tvar movie = movies[path];\n\t\tif (!movie) {\n\t\t\tconsole.log('no stream for ' + path);\n\t\t\treturn;\n\t\t}\n\n\t\tmovie.notify = movie.notify.filter(function(f) {\n\t\t\treturn f != callback;\n\t\t});\n\n\t\tif (!movie.notify.length) {\n\t\t\tconsole.log('stopping stream for ' + path);\n\t\t\tworker.postMessage({file: path, mode: 'stop'});\n\t\t\tdelete movies[path];\n\t\t} else {\n\t\t\tconsole.log('stream for ' + path + ' has ' + movie.notify.length + ' remaining subscriptions');\n\t\t}\n\t}\n\n\tvar defaultColors = [\n\t\t[\n\t\t\t[  0,   0,   0],\n\t\t\t[  0,   0, 128],\n\t\t\t[  0, 128,   0],\n\t\t\t[  0, 128, 128],\n\t\t\t[128,   0,   0],\n\t\t\t[128,   0, 128],\n\t\t\t[128, 128,   0],\n\t\t\t[192, 192, 192]\n\t\t],\n\t\t[\n\t\t\t[128, 128, 128],\n\t\t\t[  0,   0, 255],\n\t\t\t[  0, 255,   0],\n\t\t\t[  0, 255, 255],\n\t\t\t[255,   0,   0],\n\t\t\t[255,   0, 255],\n\t\t\t[255, 255,   0],\n\t\t\t[255, 255, 255]\n\t\t]\n\t];\n\n\tfunction TileSet(path, colors) {\n\t\tthis.colors = colors || defaultColors;\n\t\tthis.image = new Image();\n\t\tthis.image.src = path || 'curses_800x600.png';\n\t\tthis.image.onload = function() {\n\t\t\tvar canvas = document.createElement('canvas');\n\t\t\tvar width = this.width = (canvas.width = this.image.width) / 16;\n\t\t\tvar height = this.height = (canvas.height = this.image.height) / 16;\n\t\t\tvar tiles = this.tiles = [];\n\t\t\tvar ctx = canvas.getContext('2d');\n\t\t\tctx.drawImage(this.image, 0, 0);\n\t\t\tfor (var y = 0; y < 16; y++) {\n\t\t\t\tfor (var x = 0; x < 16; x++) {\n\t\t\t\t\ttiles.push(ctx.getImageData(x * width, y * height, width, height));\n\t\t\t\t}\n\t\t\t}\n\t\t\tthis.loaded = true;\n\t\t\tif (this.onload) {\n\t\t\t\tthis.onload();\n\t\t\t}\n\t\t}.bind(this);\n\t}\n\n\tvar pauseButtonText = '\u25ae\u25ae', playButtonText = '\u25b6';\n\n\tfunction Renderer(tileset) {\n\t\tvar canvas = document.createElement('canvas');\n\t\tcanvas.className = 'cmv-canvas';\n\t\tvar ctx = null;\n\n\t\tvar slider = document.createElement('input');\n\t\tslider.type = 'range';\n\t\tslider.min = slider.max = slider.value = 0;\n\t\tslider.disabled = true;\n\t\tslider.className = 'cmv-time-slider';\n\t\tslider.oninput = slider.onchange = function() {\n\t\t\tif (!rendering) {\n\t\t\t\tthis.seek(+slider.value);\n\t\t\t}\n\t\t}.bind(this);\n\t\tslider.onmousedown = function() {\n\t\t\tmousedown = true;\n\t\t};\n\t\tslider.onmouseup = function() {\n\t\t\tmousedown = false;\n\t\t};\n\n\t\tvar msPerFrame = 20;\n\t\tfunction formatTime(t) {\n\t\t\tvar negative = '';\n\t\t\tif (t < 0) {\n\t\t\t\tt = -t;\n\t\t\t\tnegative = '-';\n\t\t\t}\n\t\t\tvar seconds = Math.floor(t * 10) / 10 % 60, minutes = Math.floor(t / 60);\n\t\t\tif (Math.floor(seconds) == seconds) {\n\t\t\t\tseconds += '.0';\n\t\t\t}\n\t\t\tif (seconds < 10) {\n\t\t\t\tseconds = '0' + seconds;\n\t\t\t}\n\t\t\tseconds = String(seconds).substring(0, 4);\n\t\t\treturn negative + minutes + ':' + seconds;\n\t\t};\n\n\t\tvar timeDisplay = document.createElement('span');\n\t\ttimeDisplay.className = 'cmv-time-display';\n\n\t\tvar renderTick = null;\n\t\tvar currentFrame = -1;\n\n\t\tvar rendering = false;\n\t\tvar mousedown = false;\n\t\tvar dirty = false;\n\t\tvar paused = false;\n\n\t\tvar pauseButton = document.createElement('button');\n\t\tpauseButton.className = 'cmv-pause-button cmv-pause-button-pause';\n\t\tpauseButton.innerHTML = pauseButtonText;\n\t\tpauseButton.onclick = function() {\n\t\t\tpaused = !paused;\n\t\t\tdirty = true;\n\t\t\tif (paused) {\n\t\t\t\tpauseButton.className = 'cmv-pause-button cmv-pause-button-play';\n\t\t\t\tpauseButton.innerHTML = playButtonText;\n\t\t\t} else {\n\t\t\t\tpauseButton.className = 'cmv-pause-button cmv-pause-button-pause';\n\t\t\t\tpauseButton.innerHTML = pauseButtonText;\n\t\t\t}\n\t\t};\n\t\tvar imageData = null;\n\t\tfunction renderFrame(frame) {\n\t\t\tvar mid = frame.width * frame.height;\n\n\t\t\tfor (var tx = 0; tx < frame.width; tx++) {\n\t\t\t\tvar off1 = tx * frame.height;\n\t\t\t\tfor (var ty = 0; ty < frame.height; ty++) {\n\t\t\t\t\tvar off2 = off1 + ty;\n\t\t\t\t\tvar off3 = off2 + mid;\n\n\t\t\t\t\tvar t = tileset.tiles[frame.data[off2]];\n\t\t\t\t\tvar fg = tileset.colors[frame.data[off3] >> 6][frame.data[off3] & 7];\n\t\t\t\t\tvar bg = tileset.colors[0][(frame.data[off3] >> 3) & 7];\n\n\t\t\t\t\tfor (var x = 0; x < tileset.width; x++) {\n\t\t\t\t\t\tfor (var y = 0; y < tileset.height; y++) {\n\t\t\t\t\t\t\tvar off = (x + y * tileset.width) * 4;\n\t\t\t\t\t\t\tvar r = t.data[off + 0], g = t.data[off + 1], b = t.data[off + 2], a = t.data[off + 3];\n\t\t\t\t\t\t\tif (r == 255 && g == 0 && b == 255 && a == 255) {\n\t\t\t\t\t\t\t\tr = g = b = a = 0;\n\t\t\t\t\t\t\t}\n\t\t\t\t\t\t\toff = ((x + tx * tileset.width) + (y + ty * tileset.height) * imageData.width) * 4;\n\t\t\t\t\t\t\timageData.data[off + 0] = (r * a * fg[0] / 255 + (255 - a) * bg[0]) / 255;\n\t\t\t\t\t\t\timageData.data[off + 1] = (g * a * fg[1] / 255 + (255 - a) * bg[1]) / 255;\n\t\t\t\t\t\t\timageData.data[off + 2] = (b * a * fg[2] / 255 + (255 - a) * bg[2]) / 255;\n\t\t\t\t\t\t\timageData.data[off + 3] = 255;\n\t\t\t\t\t\t}\n\t\t\t\t\t}\n\t\t\t\t}\n\t\t\t}\n\n\t\t\tctx.putImageData(imageData, 0, 0);\n\t\t};\n\n\t\tvar once = function(frame, movie) {\n\t\t\tif (!firstFrame) {\n\t\t\t\tfirstFrame = frame;\n\t\t\t\ttheMovie   = movie;\n\t\t\t}\n\t\t\tif (!tileset.loaded) {\n\t\t\t\treturn;\n\t\t\t}\n\n\t\t\tcanvas.width = tileset.width * frame.width;\n\t\t\tcanvas.height = tileset.height * frame.height;\n\t\t\tctx = canvas.getContext('2d');\n\t\t\timageData = ctx.createImageData(canvas.width, canvas.height);\n\t\t\tslider.disabled = false;\n\n\t\t\tthis.seek = function(tick) {\n\t\t\t\tcurrentFrame = tick;\n\t\t\t\tclearTimeout(renderTick);\n\t\t\t\trenderTick = setInterval(render.bind(this, movie), msPerFrame);\n\t\t\t\tdirty = true;\n\t\t\t\tif (movie.frames[tick]) {\n\t\t\t\t\trender.call(this, movie);\n\t\t\t\t} else {\n\t\t\t\t\tmovie.seek(tick, true);\n\t\t\t\t}\n\t\t\t};\n\t\t\tthis.seek(0);\n\n\t\t\tonce = function(frame, movie) {};\n\t\t}.bind(this);\n\n\t\tfunction render(movie) {\n\t\t\tif (!(currentFrame in movie.frames)) {\n\t\t\t\treturn;\n\t\t\t}\n\n\t\t\trendering = true;\n\n\t\t\tif (!mousedown) {\n\t\t\t\tslider.value = currentFrame;\n\t\t\t}\n\t\t\tvar add = 1;\n\t\t\tif (dirty) {\n\t\t\t\tvar start = +new Date();\n\t\t\t\trenderFrame(movie.frames[currentFrame]);\n\t\t\t\tif (movie.done) {\n\t\t\t\t\ttimeDisplay.innerHTML = formatTime(currentFrame * msPerFrame / 1000) + ' / ' + formatTime(movie.done * msPerFrame / 1000);\n\t\t\t\t\tslider.max = movie.done;\n\t\t\t\t} else {\n\t\t\t\t\ttimeDisplay.innerHTML = formatTime((currentFrame - movie.loaded) * msPerFrame / 1000) + ' / LIVE';\n\t\t\t\t\tslider.max = movie.loaded;\n\t\t\t\t}\n\t\t\t\tdirty = false;\n\t\t\t\tmovie.seek(currentFrame, false);\n\t\t\t\tadd += Math.floor((new Date() - start) / msPerFrame);\n\t\t\t}\n\n\t\t\tif (!paused && !mousedown && movie.frames.length > currentFrame + 1) {\n\t\t\t\tcurrentFrame += add;\n\t\t\t\tcurrentFrame = Math.min(currentFrame, movie.frames.length - 1);\n\t\t\t\tdirty = true;\n\t\t\t} else if (!paused && !mousedown && currentFrame < movie.loaded) {\n\t\t\t\tthis.seek(currentFrame + 1);\n\t\t\t}\n\n\t\t\trendering = false;\n\t\t};\n\n\t\tvar firstFrame = null, theMovie = null;\n\t\tif (!tileset.loaded) {\n\t\t\tvar oldonload = tileset.onload;\n\t\t\ttileset.onload = function() {\n\t\t\t\tif (firstFrame) {\n\t\t\t\t\tonce(firstFrame, theMovie);\n\t\t\t\t}\n\n\t\t\t\tif (oldonload) {\n\t\t\t\t\toldonload.call(tileset);\n\t\t\t\t}\n\t\t\t}.bind(this);\n\t\t}\n\n\t\tthis.callback = function(frame, movie) {\n\t\t\tonce(frame, movie);\n\t\t\tif (movie.done) {\n\t\t\t\tslider.max = movie.done;\n\t\t\t} else {\n\t\t\t\tslider.max = movie.loaded;\n\t\t\t}\n\t\t\tdirty = true;\n\t\t}.bind(this);\n\n\t\tthis.element = document.createElement('div');\n\t\tthis.element.className = 'cmv-container';\n\t\tthis.element.appendChild(canvas);\n\t\tthis.element.appendChild(pauseButton);\n\t\tthis.element.appendChild(timeDisplay);\n\t\tthis.element.appendChild(slider);\n\n\t\tthis.dispose = function() {\n\t\t\tclearTimeout(renderTick);\n\t\t\trenderTick = null;\n\t\t\tthis.element.removeChild(canvas);\n\t\t\tthis.element.removeChild(pauseButton);\n\t\t\tthis.element.removeChild(timeDisplay);\n\t\t\tthis.element.removeChild(slider);\n\t\t\tthis.callback = function() {};\n\t\t\tthis.dispose = function() {};\n\t\t};\n\t}\n\n\treturn {\n\t\tstart:    startStream,\n\t\tstop:     stopStream,\n\t\tTileSet:  TileSet,\n\t\tRenderer: Renderer\n\t};\n}();\n" +
	""}))