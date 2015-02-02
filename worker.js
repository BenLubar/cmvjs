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
		parseIndex: 0
	};
	movie.xhr.open('GET', path, true);
	movie.xhr.overrideMimeType('text/plain; charset=x-user-defined');
	movie.xhr.onprogress = cmvProgress.bind(movie);
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

function cmvProgress(e) {
	if (this.version === null && e.loaded >= 4 * 1) {
		this.version = uint32(this.xhr.responseText, 4 * 0);
		console.log(this.path + ' version: ' + this.version);
		if (this.version < 10000 || this.version > 10001) {
			throw this.path + ' unsupported cmv version ' + this.version;
		}
	}
	if (this.width === null && e.loaded >= 4 * 2) {
		this.width = uint32(this.xhr.responseText, 4 * 1);
		console.log(this.path + ' width: ' + this.width);
	}
	if (this.height === null && e.loaded >= 4 * 3) {
		this.height = uint32(this.xhr.responseText, 4 * 2);
		console.log(this.path + ' height: ' + this.height);
	}
	if (this.index === null && e.loaded >= 4 * 5) {
		if (this.version >= 10001) {
			// skip sound information for now.
			var i = 4 * 5 + uint32(this.xhr.responseText, 4 * 4) * 50 + 200 * 16 * 4;
			if (e.loaded >= i) {
				this.index = i;
				console.log(this.path + ' finished header');
			}
		} else {
			this.index = 4 * 4;
			console.log(this.path + ' finished header');
		}
	}
	while (this.index !== null && e.loaded >= this.index + 4) {
		var length = uint32(this.xhr.responseText, this.index);
		if (e.loaded >= this.index + 4 + length) {
			var compressed = new Uint8Array(length);
			for (var i = 0; i < length; i++) {
				compressed[i] = this.xhr.responseText.charCodeAt(this.index + 4 + i) & 0xFF;
			}
			var data = new Zlib.Inflate(compressed).decompress();
			this.index += 4 + length;
			this.toParse.push(data);
			console.log(this.path + ' decompressed: ' + length + ' -> ' + data.length);

			this.parseIndex = extractFrames(this.path, this.toParse, this.parseIndex, this.version, this.width, this.height);
		} else {
			break;
		}
	}
}

function extractFrames(path, toParse, index, version, width, height) {
	var length = width * height * 2;

	while (toParse.length) {
		var remaining = -index;
		toParse.forEach(function(data) {
			remaining += data.length;
		});
		if (remaining < length) {
			return index;
		}

		var frame = new Uint8Array(length);
		for (var i = 0; i < length; i++) {
			while (index >= toParse[0].length) {
				toParse.shift();
				index = 0;
			}
			frame[i] = toParse[0][index];
			index++;
		}
		postMessage({frame: {data: frame, width: width, height: height}, file: path});
	}
	return 0;
}

function uint32(data, off) {
	return ((data.charCodeAt(off + 0) & 0xFF) <<  0) +
	       ((data.charCodeAt(off + 1) & 0xFF) <<  8) +
	       ((data.charCodeAt(off + 2) & 0xFF) << 16) +
	       ((data.charCodeAt(off + 3) & 0xFF) << 24);
}
