"use strict";

importScripts('zlib.min.js');

self.console = self.console || {};
console.log = console.log || function(message) {};

onmessage = function(e) {
	switch (e.data.mode) {
	case 'start':
		startCMV(e.data.file);
		break;
	case 'stop':
		stopCMV(e.data.file);
		break;
	case 'position':
		var movie = movies[e.data.file];
		if (movie) {
			if (e.data.force) {
				console.log(movie.path + ' seek request: ' + movie.frame + ' -> ' + e.data.position);
			}
			movie.position = e.data.position;
			cmvProgress.call(movie, e.data.force);
		}
		break;
	default:
		throw 'unknown mode ' + e.data.mode;
	}
};

var movies = {};

function startCMV(path) {
	var movie = {
		data:       new Uint8Array(0).buffer,
		path:       path,
		version:    null,
		width:      null,
		height:     null,
		index:      null,
		toParse:    [],
		parseIndex: 0,
		frame:      -1,
		position:   0,
		done:       0,
		keyframes:  []
	};
	if (/^[^\/?#]+\.cmv$/.test(path)) {
		var xhr = new XMLHttpRequest();
		xhr.open('GET', 'movies.json', true);
		xhr.responseType = 'json';
		xhr.onload = function() {
			xhr.response.forEach(function(entry) {
				if (entry.Name === path) {
					var age = new Date(xhr.getResponseHeader('Date') || new Date) - new Date(entry.Mod);
					if ('Frames' in entry) {
						if (age > 60000) {
							postMessage({done: entry.Frames, file: path});
						} else {
							postMessage({loaded: entry.Frames, file: path});
						}
					}
				}
			});
		}
		xhr.send(null);
	}
	cmvRequest.call(movie);

	movies[path] = movie;
}

function stopCMV(path) {
	if (!(path in movies)) {
		console.log(path + ' already stopped');
		return;
	}
	movies[path].xhr.abort();
	delete movies[path];
}

function cmvRequest() {
	var len = this.data.byteLength;
	this.xhr = new XMLHttpRequest();
	this.xhr.open('GET', this.path, true);
	this.xhr.setRequestHeader('Range', 'bytes=' + len + '-' + (len + 1024 * 1024 - 1));
	this.xhr.responseType = 'arraybuffer';
	this.xhr.onload = function() {
		var buf = new Uint8Array(len + this.xhr.response.byteLength);
		buf.set(new Uint8Array(this.data), 0);
		buf.set(new Uint8Array(this.xhr.response), len);
		this.data = buf.buffer;

		var age = new Date(this.xhr.getResponseHeader('Date') || new Date) - new Date(this.xhr.getResponseHeader('Last-Modified'));

		if (this.xhr.response.byteLength === 1024 * 1024) {
			cmvRequest.call(this);
		} else if (age > 60000) {
			this.done = 1;
		} else {
			setTimeout(cmvRequest.bind(this), 10000);
		}

		cmvProgress.call(this);
	}.bind(this);
	this.xhr.send(null);
}

function cmvProgress(forcePosition) {
	if (this.version === null && this.data.byteLength >= 4 * 1) {
		this.version = uint32(this.data, 4 * 0);
		console.log(this.path + ' version: ' + this.version);
		if (this.version < 10000 || this.version > 10001) {
			throw this.path + ' unsupported cmv version ' + this.version;
		}
	}
	if (this.width === null && this.data.byteLength >= 4 * 2) {
		this.width = uint32(this.data, 4 * 1);
		console.log(this.path + ' width: ' + this.width);
	}
	if (this.height === null && this.data.byteLength >= 4 * 3) {
		this.height = uint32(this.data, 4 * 2);
		console.log(this.path + ' height: ' + this.height);
	}
	if (forcePosition && (this.frame >= this.position || Math.floor(this.position / 180000) != Math.floor(this.frame / 180000))) {
		var keyframe = Math.floor(this.position / 180000);
		console.log(this.path + ' seeking: ' + this.position + ' (using keyframe ' + keyframe + ')');
		this.index = this.keyframes[keyframe].index;
		this.frame = keyframe * 180000 - 1;
		this.toParse = [this.keyframes[keyframe].data];
		this.parseIndex = 0;
	}
	if (this.index === null && this.data.byteLength >= 4 * 5) {
		if (this.version >= 10001) {
			// skip sound information for now.
			var i = 4 * 5 + uint32(this.data, 4 * 4) * 50 + 200 * 16 * 4;
			if (this.data.byteLength >= i) {
				this.index = i;
				console.log(this.path + ' finished header');
			}
		} else {
			this.index = 4 * 4;
			console.log(this.path + ' finished header');
		}
	}
	while (this.index !== null && this.data.byteLength >= this.index + 4) {
		if (this.frame >= this.position + 180000 && this.done == 2) {
			break;
		}

		var length = uint32(this.data, this.index);
		if (this.data.byteLength >= this.index + 4 + length) {
			var compressed = new Uint8Array(this.data, this.index + 4, length);
			var data = new Zlib.Inflate(compressed).decompress();
			this.index += 4 + length;
			this.toParse.push(data);
			//console.log(this.path + ' decompressed: ' + length + ' -> ' + data.length);

			extractFrames.call(this);
		} else {
			break;
		}

		// allow event handling to run between iterations.
		setTimeout(cmvProgress.bind(this), 0);
		return;
	}
	if (this.done === 1) {
		this.done = 2;
		postMessage({file: this.path, done: this.frame});
	}
}

function extractFrames() {
	var length = this.width * this.height * 2;

	while (this.toParse.length) {
		if (this.frame >= this.position + 180000 && this.done == 2) {
			return;
		}

		var remaining = -this.parseIndex;
		this.toParse.forEach(function(data) {
			remaining += data.length;
		});
		if (remaining < length) {
			return;
		}

		if ((this.frame + 1) % 180000 == 0) {
			var keyframe = (this.frame + 1) / 180000;
			if (!(keyframe in this.keyframes)) {
				var parseIndex = this.parseIndex;
				var toParse = this.toParse.slice(0);

				var data = new Uint8Array(remaining);
				for (var i = 0; i < remaining; i++) {
					while (parseIndex >= toParse[0].length) {
						toParse.shift();
						parseIndex = 0;
					}
					data[i] = toParse[0][parseIndex];
					parseIndex++;
				}

				console.log(this.path + ' adding keyframe: ' + keyframe + ' (index = ' + this.index + ', ' + remaining + ' bytes)');
				this.keyframes[keyframe] = {
					data:  data,
					index: this.index,
				};
			}
		}

		var frame = new Uint8Array(length);
		for (var i = 0; i < length; i++) {
			while (this.parseIndex >= this.toParse[0].length) {
				this.toParse.shift();
				this.parseIndex = 0;
			}
			frame[i] = this.toParse[0][this.parseIndex];
			this.parseIndex++;
		}
		this.frame++;
		if (this.frame >= this.position && this.frame < this.position + 180000) {
			postMessage({frame: {
				data: frame,
				width: this.width,
				height: this.height,
				index: this.frame
			}, file: this.path});
		} else if (this.done == 0) {
			postMessage({loaded: this.frame, file: this.path});
		}
	}
	this.parseIndex = 0;
}

function uint32(data, off) {
	data = new Uint8Array(data, off, 4);
	return (data[0] <<  0) +
	       (data[1] <<  8) +
	       (data[2] << 16) +
	       (data[3] << 24);
}
