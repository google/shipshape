var fs = require('fs');
var crypto = require('crypto');
var graphs = require('./graphs.js');
var path = require("path");
var util = require("util");
exports.Entity = function(id) {
};

exports.Entity.prototype = new graphs.Node();
exports.Entity.prototype.isUpToDate = function(state) {
  var newState = this.getState();
  return newState === state;
};

exports.Entity.prototype.getState = function() {
  var shasum = crypto.createHash('sha1');
  shasum.update(this.getUpToDateMarker());
  for (var i = 0; i < this.outputs.length; i++) {
    shasum.update(this.outputs[i].getUpToDateMarker());
  }
  return shasum.digest("hex");
};

exports.Entity.prototype.createScript = function() {
  return undefined;
};

exports.Entity.prototype.createFullCommand = function() {
  return {};
};

exports.Entity.prototype.getFileInputs = function() {
  var files = [];
  for (var i = 0; i < this.inputs.length; i++) {
    if (this.inputs[i].__proto__ == exports.File.prototype) {
      files.push(this.inputs[i].getPath());
    }
  }
  return files;
}

function digestFile(shasum, file) {
    var content = fs.readFileSync(file);
    shasum.update(content);
}

function digestDirectory(shasum, dir) {
  var files = fs.readdirSync(dir);
  for (var i = 0; i < files.length; i++) {
    var file = path.join(dir, files[i]);
    var stats = fs.statSync(file);
    if (stats.isDirectory()) {
      digestDirectory(shasum,file);
    } else if (stats.isFile()) {
      digestFile(shasum, file);
    }
  }
}

exports.File = function(id, kind){
  this.init(id);
  this.kind = kind;
};

exports.File.prototype = new exports.Entity();
exports.File.prototype.createScript = function() {
  return [];
};

exports.File.prototype.isUpToDate = function(state) {
  return fs.existsSync(this.getPath());
};

exports.File.prototype.getPath = function() {
  return this.id;
};

exports.File.prototype.getUpToDateMarker = function() {
  var path = this.getPath();
  if (!fs.existsSync(path)) {
    return "";
  } else {
    var shasum = crypto.createHash('sha1');
    digestFile(shasum, path);
    return shasum.digest("hex");
  }
};

exports.getFileNode = function(id, kind, graph) {
  var loaded = graph.getNode(id);
  if (loaded) {
    return loaded;
  }
  var file = new exports.File(id, "src_file");
  graph.addNode(file);
  return file;
};

exports.DependentFile = function(id, kind, deps){
  exports.File.call(this, id, kind);
  this.deps = deps;
};
exports.DependentFile.prototype = new exports.File();
exports.DependentFile.prototype.getUpToDateMarker = function() {
  var shasum = crypto.createHash('sha1');
  shasum.update(exports.File.prototype.getUpToDateMarker.call(this));
  for (var i = 0; i < this.deps.length; i++) {
    shasum.update(this.deps[i].getUpToDateMarker());
  }
  return shasum.digest('hex');
};

exports.Directory = function(id) {
  this.init(id);
};

exports.Directory.prototype = new exports.Entity();
exports.Directory.prototype.isUpToDate = function(state) {
  return fs.existsSync(this.getPath());
};

exports.Directory.prototype.getPath = function() {
  return this.id;
};

exports.Directory.prototype.getUpToDateMarker = function() {
 var path = this.getPath();
  if (!fs.existsSync(path)) {
    return "";
  } else {
    var shasum = crypto.createHash('sha1');
    digestDirectory(shasum, path);
    return shasum.digest("hex");
  }
};

exports.getDirectoryNode = function(id, graph) {
  var loaded = graph.getNode(id);
  if (loaded) {
    return loaded;
  }
  var dir = new exports.Directory(id);
  graph.addNode(dir);
  return dir;
};

exports.Property = function(ownerId, key, value){
  this.init(ownerId +":" + key);
  this.key = key;
  this.value = value;
};

exports.Property.prototype = new exports.Entity();
exports.Property.prototype.isUpToDate = function(state) {
  return state === this.value;
};

exports.Property.prototype.getState = function() {
  return this.value;
};

exports.Property.prototype.getUpToDateMarker = function() {
  var shasum = crypto.createHash('sha1');
  if (this.value) {
    if (this.value instanceof Array) {
      for (var i = 0; i < this.value.length; ++i) {
        shasum.update(this.value[i]);
      }
    } else {
      shasum.update(this.value);
    }
  }
  return shasum.digest("hex");
};

exports.Property.prototype.createScript = function() {
  return [];
};

exports.Action = function(version, mnemonic) {
  this.version = version;
  this.mnemonic = mnemonic;
};

exports.Action.prototype = new exports.Entity();
exports.Action.prototype.getUpToDateMarker = function() {
  var shasum = crypto.createHash('sha1');
  shasum.update(this.version.toString());
  for (var i = 0; i < this.inputs.length; i++) {
    shasum.update(this.inputs[i].getUpToDateMarker());
  }
  return shasum.digest("hex");
};

exports.Action.prototype.addAllInputs = function(inputs) {
  for (var i = 0; i < inputs.length; i++) {
    this.addParent(inputs[i]);
  }
};

exports.ProcessAction = function(owner, version, mnemonic) {
  this.owner = owner;
  this.version = version;
  this.mnemonic = mnemonic;
};

exports.ProcessAction.prototype = new exports.Action();
exports.ProcessAction.prototype.commandBuilder = function() {
};

exports.ProcessAction.prototype.argsBuilder = function() {
};

exports.ProcessAction.prototype.createScript = function() {
  var lines = [];
  var parent = this;
  var cmd = [];
  cmd.push(this.commandBuilder());
  var args = this.argsBuilder();
  for (var i = 0; i < args.length; i++) {
    cmd.push(args[i]);
  }
  lines.push(cmd.join(" "));
  lines.push("exit $?");
  return lines;
};

exports.ProcessAction.prototype.createFullCommand = function() {
  return {
    id: this.id,
    tool: this.commandBuilder(),
    args: this.argsBuilder(),
    inputs: this.getFileInputs(),
    mnemonic: this.mnemonic
  };
};

// Action that runs a built binary target
exports.ToolAction = function(owner, version, toolTarget) {
  exports.ProcessAction.call(this, owner, version, "Tool");
  if (toolTarget) {
    this.binary = this.addTool(toolTarget);
  }
};

exports.ToolAction.prototype = new exports.ProcessAction();
exports.ToolAction.prototype.addTool = function(toolTarget) {
  var target = typeof(toolTarget) == 'object'
      ? toolTarget
      : this.owner.engine.resolveTarget(toolTarget);
  var toolOuts = target.rule.getOutputsFor(target, "build");
  var knownBinaries = toolOuts.filter(function(out) {
    return out.kind.endsWith("_shell") || out.kind.endsWith("_executable");
  });
  if (knownBinaries.length != 1) {
    console.log("Invalid ToolAction target '" + toolTarget + "'; has unknown binary outputs!");
    console.log(toolOuts);
    process.exit(1);
  }
  this.addAllInputs(knownBinaries);
  return "./" + knownBinaries[0].id;
};
exports.ToolAction.prototype.commandBuilder = function() {
  return this.binary;
};
