var CMV = function() {
	"use strict";

	var worker = new Worker('worker.js');
	var movies = {};
	worker.onmessage = function(e) {
		var frame = e.data.frame;
		if (frame) {
			var movie = movies[e.data.file];
			if (!movie) {
				console.log('no movie for ' + e.data.file);
				return;
			}
			movie.frames.push(frame);
			movie.notify.forEach(function(f) {
				f(frame, movie);
			});
		}
	}

	function startStream(path, callback) {
		var movie = movies[path];
		if (movie) {
			console.log('stream already started for ' + path);
			movie.notify.push(callback);
			movie.frames.forEach(function(frame) {
				callback(frame, movie);
			});
			return;
		}
		console.log('starting stream for ' + path);
		movies[path] = {frames: [], notify: [callback]};
		worker.postMessage({file: path, mode: 'start'});
	}

	function stopStream(path, callback) {
		var movie = movies[path];
		if (!movie) {
			console.log('no stream for ' + path);
			return;
		}

		movie.notify = movie.notify.filter(function(f) {
			return f != callback;
		});

		if (!movie.notify.length) {
			console.log('stopping stream for ' + path);
			worker.postMessage({file: path, mode: 'stop'});
			delete movies[path];
		} else {
			console.log('stream for ' + path + ' has ' + movie.notify.length + ' remaining subscriptions');
		}
	}

	var colors = [
		[
			[0, 0, 0],
			[0, 0, 128],
			[0, 128, 0],
			[0, 128, 128],
			[128, 0, 0],
			[128, 0, 128],
			[128, 128, 0],
			[192, 192, 192]
		],
		[
			[128, 128, 128],
			[0, 0, 255],
			[0, 255, 0],
			[0, 255, 255],
			[255, 0, 0],
			[255, 0, 255],
			[255, 255, 0],
			[255, 255, 255]
		]
	];

	var tileset = new Image();
	tileset.src = 'curses_800x600.png';
	tileset.onload = function() {
		var tileWidth = tileset.width / 16;
		var tileHeight = tileset.height / 16;
		var tiles = function() {
			var canvas = document.createElement('canvas');
			canvas.width = tileset.width;
			canvas.height = tileset.height;
			var ctx = canvas.getContext('2d');
			ctx.drawImage(tileset, 0, 0);
			var tiles = [];
			for (var y = 0; y < 16; y++) {
				for (var x = 0; x < 16; x++) {
					tiles.push(ctx.getImageData(x * tileWidth, y * tileHeight, tileWidth, tileHeight));
				}
			}
			return tiles;
		}();

		var rendering = false;
		var canvas = document.createElement('canvas');
		var ctx = null;
		var slider = document.createElement('input');
		slider.type = 'range';
		slider.min = slider.max = slider.value = 0;
		slider.disabled = true;
		slider.oninput = slider.onchange = function() {
			if (!rendering) {
				seek(+slider.value);
			}
		};
		var msPerFrame = 20;
		var timeDisplay = document.createElement('span');
		var renderTick = null;
		var currentFrame = -1;
		var seek = null;
		var mousedown = false;
		slider.onmousedown = function() {
			mousedown = true;
		};
		slider.onmouseup = function() {
			mousedown = false;
		}
		var dirty = false;
		var paused = false;
		var pauseButton = document.createElement('button');
		pauseButton.innerHTML = '▮▮';
		pauseButton.onclick = function() {
			paused = !paused;
			dirty = true;
			if (paused) {
				pauseButton.innerHTML = '▶';
			} else {
				pauseButton.innerHTML = '▮▮';
			}
		};

		var imageData = null;
		var renderFrame = function(frame) {
			var mid = frame.width * frame.height;

			for (var tx = 0; tx < frame.width; tx++) {
				var off1 = tx * frame.height;
				for (var ty = 0; ty < frame.height; ty++) {
					var off2 = off1 + ty;
					var off3 = off2 + mid;

					var t = tiles[frame.data[off2]];
					var fg = colors[frame.data[off3] >> 6][frame.data[off3] & 7];
					var bg = colors[0][(frame.data[off3] >> 3) & 7];

					for (var x = 0; x < tileWidth; x++) {
						for (var y = 0; y < tileHeight; y++) {
							var off = (x + y * tileWidth) * 4;
							var r = t.data[off + 0], g = t.data[off + 1], b = t.data[off + 2], a = t.data[off + 3];
							if (r == 255 && g == 0 && b == 255 && a == 255) {
								r = g = b = a = 0;
							}
							off = ((x + tx * tileWidth) + (y + ty * tileHeight) * imageData.width) * 4;
							imageData.data[off + 0] = (r * a * fg[0] / 255 + (255 - a) * bg[0]) / 255;
							imageData.data[off + 1] = (g * a * fg[1] / 255 + (255 - a) * bg[1]) / 255;
							imageData.data[off + 2] = (b * a * fg[2] / 255 + (255 - a) * bg[2]) / 255;
							imageData.data[off + 3] = 255;
						}
					}
				}
			}

			ctx.putImageData(imageData, 0, 0);
		};

		var formatTime = function(t) {
			var seconds = Math.floor(t * 10) / 10 % 60, minutes = Math.floor(t / 60);
			if (Math.floor(seconds) == seconds) {
				seconds += '.0';
			}
			if (seconds < 10) {
				seconds = '0' + seconds;
			}
			seconds = String(seconds).substring(0, 4);
			return minutes + ':' + seconds;
		};

		var render = function(movie) {
			rendering = true;

			if (!mousedown) {
				slider.value = currentFrame;
			}
			if (dirty) {
				renderFrame(movie.frames[currentFrame]);
				timeDisplay.innerHTML = formatTime(currentFrame * msPerFrame / 1000) + ' / ' + formatTime((movie.frames.length - 1) * msPerFrame / 1000);
				dirty = false;
			}

			if (!paused && !mousedown && movie.frames.length > currentFrame + 1) {
				currentFrame++;
				dirty = true;
			}

			rendering = false;
		};

		var once = function(frame, movie) {
			canvas.width = tileWidth * frame.width;
			canvas.height = tileHeight * frame.height;
			ctx = canvas.getContext('2d');
			imageData = ctx.createImageData(canvas.width, canvas.height);
			slider.disabled = false;

			seek = function(tick) {
				currentFrame = tick;
				clearTimeout(renderTick);
				renderTick = setInterval(render.bind(null, movie), msPerFrame);
				dirty = true;
				render(movie);
			};
			seek(0);

			once = function(frame, movie) {};
		};

		startStream('last_record.cmv', function(frame, movie) {
			once(frame, movie);
			slider.max = movie.frames.length - 1;
			dirty = true;
		});

		document.body.appendChild(canvas);
		document.body.appendChild(pauseButton);
		document.body.appendChild(timeDisplay);
		document.body.appendChild(slider);
	};

	return {
		start: startStream,
		stop: stopStream
	};
}();