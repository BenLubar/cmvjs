// AUTOMATICALLY GENERATED FILE. DO NOT EDIT.

package main

var indexhtml = html(asset.init(asset{Name: "index.html", Content: "" +
	"<html>\n<head>\n<title>cmvjs</title>\n<style>\nhtml {\n\tbackground: #000;\n}\nbody {\n\tmargin: 0;\n}\n.cmv-container {\n\twidth: 800px;\n\tmargin: auto;\n}\n.cmv-container:after {\n\tcontent: '';\n\tclear: both;\n}\n.cmv-canvas, .cmv-time-slider, .cmv-pause-button, .cmv-time-display {\n\tdisplay: block;\n}\n.cmv-time-slider {\n\twidth: 610px;\n\theight: 22px;\n}\n.cmv-pause-button {\n\tfloat: left;\n\twidth: 40px;\n\theight: 26px;\n}\n.cmv-time-display {\n\tfloat: right;\n\twidth: 150px;\n\tcolor: #fff;\n\tfont: 11px/26px monospace;\n\ttext-align: center;\n}\n</style>\n</head>\n<body>\n<script src=\"main.js\"></script>\n<script>\nvar tileset = new CMV.TileSet();\nvar last_record = new CMV.Renderer(tileset);\nCMV.start('last_record.cmv', last_record.callback);\ndocument.body.appendChild(last_record.element);\n</script>\n</body>\n</html>\n" +
	""}))
