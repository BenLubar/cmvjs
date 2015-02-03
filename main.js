var CMV = function() {
	"use strict";

	var worker = new Worker('worker.js');
	var movies = {};
	worker.onmessage = function(e) {
		var movie = movies[e.data.file];
		if (!movie) {
			console.log('no movie for ' + e.data.file);
			return;
		}
		var frame = e.data.frame;
		if (frame) {
			var prev = movie.frames[frame.index - 1];
			for (var i = 0; prev != null && i < prev.data.length; i++) {
				if (frame.data[i] != prev.data[i]) {
					prev = null;
				}
			}
			if (prev != null) {
				frame.data = prev.data;
			}
			movie.frames[frame.index] = frame;
			movie.loaded = Math.max(movie.loaded, frame.index);
			movie.notify.forEach(function(f) {
				f(frame, movie);
			});
		}
		if ('loaded' in e.data) {
			movie.loaded = Math.max(movie.loaded, e.data.loaded);
		}
		if ('done' in e.data) {
			movie.done = e.data.done;
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
		movie = {
			loaded: -1,
			frames: [],
			notify: [callback],
			path: path,
			seek: function(tick) {
				movie.frames = [];
				worker.postMessage({
					file: path,
					mode: 'position',
					position: tick
				});
			}
		};
		movies[path] = movie;
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

	var defaultColors = [
		[
			[  0,   0,   0],
			[  0,   0, 128],
			[  0, 128,   0],
			[  0, 128, 128],
			[128,   0,   0],
			[128,   0, 128],
			[128, 128,   0],
			[192, 192, 192]
		],
		[
			[128, 128, 128],
			[  0,   0, 255],
			[  0, 255,   0],
			[  0, 255, 255],
			[255,   0,   0],
			[255,   0, 255],
			[255, 255,   0],
			[255, 255, 255]
		]
	];

	function TileSet(path, colors) {
		this.colors = colors || defaultColors;
		this.image = new Image();
		this.image.src = path || 'curses_800x600.png';
		this.image.onload = function() {
			var canvas = document.createElement('canvas');
			var width = this.width = (canvas.width = this.image.width) / 16;
			var height = this.height = (canvas.height = this.image.height) / 16;
			var tiles = this.tiles = [];
			var ctx = canvas.getContext('2d');
			ctx.drawImage(this.image, 0, 0);
			for (var y = 0; y < 16; y++) {
				for (var x = 0; x < 16; x++) {
					tiles.push(ctx.getImageData(x * width, y * height, width, height));
				}
			}
			this.loaded = true;
			if (this.onload) {
				this.onload();
			}
		}.bind(this);
	}

	var pauseButtonText = '▮▮', playButtonText = '▶';

	function Renderer(tileset) {
		var canvas = document.createElement('canvas');
		canvas.className = 'cmv-canvas';
		var ctx = null;

		var slider = document.createElement('input');
		slider.type = 'range';
		slider.min = slider.max = slider.value = 0;
		slider.disabled = true;
		slider.className = 'cmv-time-slider';
		slider.oninput = slider.onchange = function() {
			if (!rendering) {
				this.seek(+slider.value);
			}
		}.bind(this);
		slider.onmousedown = function() {
			mousedown = true;
		};
		slider.onmouseup = function() {
			mousedown = false;
		};

		var msPerFrame = 20;
		function formatTime(t) {
			var negative = '';
			if (t < 0) {
				t = -t;
				negative = '-';
			}
			var seconds = Math.floor(t * 10) / 10 % 60, minutes = Math.floor(t / 60);
			if (Math.floor(seconds) == seconds) {
				seconds += '.0';
			}
			if (seconds < 10) {
				seconds = '0' + seconds;
			}
			seconds = String(seconds).substring(0, 4);
			return negative + minutes + ':' + seconds;
		};

		var timeDisplay = document.createElement('span');
		timeDisplay.className = 'cmv-time-display';

		var renderTick = null;
		var currentFrame = -1;

		var rendering = false;
		var mousedown = false;
		var dirty = false;
		var paused = false;

		var pauseButton = document.createElement('button');
		pauseButton.className = 'cmv-pause-button cmv-pause-button-pause';
		pauseButton.innerHTML = pauseButtonText;
		pauseButton.onclick = function() {
			paused = !paused;
			dirty = true;
			if (paused) {
				pauseButton.className = 'cmv-pause-button cmv-pause-button-play';
				pauseButton.innerHTML = playButtonText;
			} else {
				pauseButton.className = 'cmv-pause-button cmv-pause-button-pause';
				pauseButton.innerHTML = pauseButtonText;
			}
		};
		var imageData = null;
		function renderFrame(frame) {
			var mid = frame.width * frame.height;

			for (var tx = 0; tx < frame.width; tx++) {
				var off1 = tx * frame.height;
				for (var ty = 0; ty < frame.height; ty++) {
					var off2 = off1 + ty;
					var off3 = off2 + mid;

					var t = tileset.tiles[frame.data[off2]];
					var fg = tileset.colors[frame.data[off3] >> 6][frame.data[off3] & 7];
					var bg = tileset.colors[0][(frame.data[off3] >> 3) & 7];

					for (var x = 0; x < tileset.width; x++) {
						for (var y = 0; y < tileset.height; y++) {
							var off = (x + y * tileset.width) * 4;
							var r = t.data[off + 0], g = t.data[off + 1], b = t.data[off + 2], a = t.data[off + 3];
							if (r == 255 && g == 0 && b == 255 && a == 255) {
								r = g = b = a = 0;
							}
							off = ((x + tx * tileset.width) + (y + ty * tileset.height) * imageData.width) * 4;
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

		var once = function(frame, movie) {
			if (!firstFrame) {
				firstFrame = frame;
				theMovie   = movie;
			}
			if (!tileset.loaded) {
				return;
			}

			canvas.width = tileset.width * frame.width;
			canvas.height = tileset.height * frame.height;
			ctx = canvas.getContext('2d');
			imageData = ctx.createImageData(canvas.width, canvas.height);
			slider.disabled = false;

			this.seek = function(tick) {
				currentFrame = tick;
				clearTimeout(renderTick);
				renderTick = setInterval(render.bind(this, movie), msPerFrame);
				dirty = true;
				if (movie.frames[tick]) {
					render.call(this, movie);
				} else {
					movie.seek(tick);
				}
			};
			this.seek(0);

			once = function(frame, movie) {};
		}.bind(this);

		function render(movie) {
			if (!(currentFrame in movie.frames)) {
				return;
			}

			rendering = true;

			if (!mousedown) {
				slider.value = currentFrame;
			}
			var add = 1;
			if (dirty) {
				var start = +new Date();
				renderFrame(movie.frames[currentFrame]);
				if (movie.done) {
					timeDisplay.innerHTML = formatTime(currentFrame * msPerFrame / 1000) + ' / ' + formatTime(movie.done * msPerFrame / 1000);
					slider.max = movie.done;
				} else {
					timeDisplay.innerHTML = formatTime((currentFrame - movie.loaded) * msPerFrame / 1000) + ' / LIVE';
					slider.max = movie.loaded;
				}
				dirty = false;
				add += Math.floor((new Date() - start) / msPerFrame);
			}

			if (!paused && !mousedown && movie.frames.length > currentFrame + 1) {
				currentFrame += add;
				currentFrame = Math.min(currentFrame, movie.frames.length - 1);
				dirty = true;
			} else if (!paused && !mousedown && currentFrame < movie.loaded) {
				this.seek(currentFrame + 1);
			}

			rendering = false;
		};

		var firstFrame = null, theMovie = null;
		if (!tileset.loaded) {
			var oldonload = tileset.onload;
			tileset.onload = function() {
				if (firstFrame) {
					once(firstFrame, theMovie);
				}

				if (oldonload) {
					oldonload.call(tileset);
				}
			}.bind(this);
		}

		this.callback = function(frame, movie) {
			once(frame, movie);
			if (movie.done) {
				slider.max = movie.done;
			} else {
				slider.max = movie.loaded;
			}
			dirty = true;
		}.bind(this);

		this.element = document.createElement('div');
		this.element.className = 'cmv-container';
		this.element.appendChild(canvas);
		this.element.appendChild(pauseButton);
		this.element.appendChild(timeDisplay);
		this.element.appendChild(slider);

		this.dispose = function() {
			clearTimeout(renderTick);
			renderTick = null;
			this.element.removeChild(canvas);
			this.element.removeChild(pauseButton);
			this.element.removeChild(timeDisplay);
			this.element.removeChild(slider);
			this.callback = function() {};
			this.dispose = function() {};
		};
	}

	return {
		start:    startStream,
		stop:     stopStream,
		TileSet:  TileSet,
		Renderer: Renderer
	};
}();
