var fs = require("fs");
var path = require("path");

Array.prototype.append = function(array) {
    this.push.apply(this, array);
};

String.prototype.startsWith = function(prefix) {
  return this.indexOf(prefix) === 0;
};

String.prototype.endsWith = function(end) {
  var index = this.indexOf(end);
  return index >= 0 && index == this.length - end.length;
};

exports.rmdirs = function(dir) {
  var files = fs.readdirSync(dir);
  for(var i = 0; i < files.length; i++) {
    var file = path.join(dir, files[i]);
    var stat = fs.statSync(file);
    if(stat.isDirectory()) {
      exports.rmdirs(file);
    } else {
      fs.unlinkSync(file);
    }
  }
  fs.rmdirSync(dir);
};

exports.findFiles = function(dir, pattern, results) {
  if (!results) {
    results = [];
  }
  var files = fs.readdirSync(dir);
  for(var i = 0; i < files.length; i++) {
    var file = path.join(dir, files[i]);
    var stat = fs.lstatSync(file);
    if(stat.isDirectory()) {
      exports.findFiles(file, pattern, results);
    } else if (pattern.exec(file)) {
      results.push(file);
    }
  }
  return results;
};

exports.parseVersion = function(version) {
  var s = version.split('.');
  if (s.length != 3) {
    console.log("ERROR parsing release_version: '" + version + "'");
    process.exit(1);
  }
  return {
    major: s[0],
    minor: s[1],
    patch: s[2]
  };
}
