"use strict";

importScripts('zlib.min.js');

onmessage = function(e) {
	loadCMV(e.data.file);
};

function loadCMV(path) {
	var xhr = new XMLHttpRequest();
	xhr.open('GET', path, true);
	xhr.overrideMimeType('text\/plain; charset=x-user-defined');

	var version = null;
	var width = null;
	var height = null;
	var index = null;
	var toParse = [];
	var parseIndex = 0;
	xhr.onprogress = function(e) {
		if (version === null && e.loaded >= 4) {
			version = uint32(xhr.responseText, 0);
			//console.log(path + ' version: ' + version);
			if (version < 10000 || version > 10001) {
				throw path + ' unsupported cmv version ' + version;
			}
		}
		if (width === null && e.loaded >= 4 + 4) {
			width = uint32(xhr.responseText, 4);
			//console.log(path + ' width: ' + width);
		}
		if (height === null && e.loaded >= 4 + 4 + 4) {
			height = uint32(xhr.responseText, 4 + 4);
			//console.log(path + ' height: ' + height);
		}
		if (index === null && e.loaded >= 4 + 4 + 4 + 4 + 4) {
			if (version >= 10001) {
				// skip sound information for now.
				var i = 4 + 4 + 4 + 4 + 4 + uint32(xhr.responseText, 4 + 4 + 4 + 4) * 50 + 200 * 16 * 4;
				if (e.loaded >= i) {
					index = i;
					//console.log(path + ' finished header');
				}
			} else {
				index = 4 + 4 + 4 + 4;
				//console.log(path + ' finished header');
			}
		}
		if (index !== null && e.loaded >= index + 4) {
			var length = uint32(xhr.responseText, index);
			if (e.loaded >= index + 4 + length) {
				var compressed = new Uint8Array(length);
				for (var i = 0; i < length; i++) {
					compressed[i] = xhr.responseText.charCodeAt(index + 4 + i) & 0xFF;
				}
				var data = new Zlib.Inflate(compressed).decompress();
				index += 4 + length;
				toParse.push(data);
				//console.log(path + ' decompressed: ' + length + ' -> ' + data.length);

				parseIndex = extractFrames(path, toParse, parseIndex, version, width, height);
			}
		}
	};
	xhr.send(null);
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
		postMessage({frame: parseFrame(frame, version, width, height), file: path});
	}
	return 0;
}

function parseFrame(data, version, width, height) {
	var frame = [];

	var mid = width * height;

	for (var x = 0; x < width; x++) {
		var off = x * height;
		var col = [];
		for (var y = 0; y < height; y++) {
			col.push([data[off + y] >> 4, data[off + y] & 0xF, data[mid + off + y] & 7, (data[mid + off + y] >> 3) & 7, data[mid + off + y] >> 6]);
		}
		frame.push(col);
	}
	return frame;
}

function uint32(data, off) {
	return ((data.charCodeAt(off + 0) & 0xFF) <<  0) +
	       ((data.charCodeAt(off + 1) & 0xFF) <<  8) +
	       ((data.charCodeAt(off + 2) & 0xFF) << 16) +
	       ((data.charCodeAt(off + 3) & 0xFF) << 24);
}