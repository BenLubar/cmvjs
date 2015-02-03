"use strict";

importScripts('zlib.min.js');

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
		xhr:        new XMLHttpRequest(),
		path:       path,
		version:    null,
		width:      null,
		height:     null,
		index:      null,
		toParse:    [],
		parseIndex: 0,
		frame:      -1,
		position:   0,
		loaded:     0,
		done:       0
	};
	movie.xhr.open('GET', path, true);
	movie.xhr.overrideMimeType('text/plain; charset=x-user-defined');
	movie.xhr.onprogress = function(e) {
		movie.loaded = e.loaded;
		cmvProgress.call(movie);
	};
	movie.xhr.onload = function() {
		movie.done = 1;
		cmvProgress.call(movie);
	};
	movie.xhr.send(null);

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

function cmvProgress(forcePosition) {
	if (this.version === null && this.loaded >= 4 * 1) {
		this.version = uint32(this.xhr.responseText, 4 * 0);
		console.log(this.path + ' version: ' + this.version);
		if (this.version < 10000 || this.version > 10001) {
			throw this.path + ' unsupported cmv version ' + this.version;
		}
	}
	if (this.width === null && this.loaded >= 4 * 2) {
		this.width = uint32(this.xhr.responseText, 4 * 1);
		console.log(this.path + ' width: ' + this.width);
	}
	if (this.height === null && this.loaded >= 4 * 3) {
		this.height = uint32(this.xhr.responseText, 4 * 2);
		console.log(this.path + ' height: ' + this.height);
	}
	if (forcePosition && this.frame > this.position) {
		console.log(this.path + ' seeking: ' + this.position);
		this.index = null;
		this.frame = -1;
		this.toParse = [];
		this.parseIndex = 0;
	}
	if (this.index === null && this.loaded >= 4 * 5) {
		if (this.version >= 10001) {
			// skip sound information for now.
			var i = 4 * 5 + uint32(this.xhr.responseText, 4 * 4) * 50 + 200 * 16 * 4;
			if (this.loaded >= i) {
				this.index = i;
				console.log(this.path + ' finished header');
			}
		} else {
			this.index = 4 * 4;
			console.log(this.path + ' finished header');
		}
	}
	while (this.index !== null && this.loaded >= this.index + 4) {
		if (this.frame >= this.position + 180000 && this.done == 2) {
			break;
		}

		var length = uint32(this.xhr.responseText, this.index);
		if (this.loaded >= this.index + 4 + length) {
			var compressed = new Uint8Array(length);
			for (var i = 0; i < length; i++) {
				compressed[i] = this.xhr.responseText.charCodeAt(this.index + 4 + i) & 0xFF;
			}
			var data = new Zlib.Inflate(compressed).decompress();
			this.index += 4 + length;
			this.toParse.push(data);
			//console.log(this.path + ' decompressed: ' + length + ' -> ' + data.length);

			extractFrames.call(this);
		} else {
			break;
		}
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
	return ((data.charCodeAt(off + 0) & 0xFF) <<  0) +
	       ((data.charCodeAt(off + 1) & 0xFF) <<  8) +
	       ((data.charCodeAt(off + 2) & 0xFF) << 16) +
	       ((data.charCodeAt(off + 3) & 0xFF) << 24);
}
