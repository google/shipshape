var fs = require("fs");
var path = require("path");
var util = require("util");

var graphs = require("./graphs.js");
var entity = require("./entity.js");
var shared = require("./shared.js");

exports.Target = function(id, json) {
  this.init(id);
  this.json = json;
};

exports.Target.prototype = new graphs.Node();
exports.Target.prototype.asPath = function() {
  return this.id.substring(2).replace(":", "/");
};

exports.Target.prototype.getRoot = function(kind) {
  var out_label = this.engine.settings.properties["out_label"];
  if (out_label === '')
    return path.join("campfire-out/", kind, this.asPath());
  else
    return path.join("campfire-out/", out_label, kind, this.asPath());
};

exports.Target.prototype.baseName = function() {
  var index = this.id.indexOf(':');
  return this.id.substring(index + 1);
};

exports.Target.prototype.getProperty = function(name) {
  if (this.properties && this.properties[name]) {
    return this.properties[name];
  } else {
    if (this.engine.settings.properties &&
        this.engine.settings.properties[name]) {
      var prop = new entity.Property(this.id, name,
                                     this.engine.settings.properties[name]);
      var loaded = this.engine.actions.getNode(prop.id);
      if (loaded) {
        return loaded;
      }
      this.engine.actions.addNode(prop);
      return prop;
    }
  }
  return undefined;
};

exports.Rule = function(engine) {
  this.engine = engine;
};

exports.Rule.prototype.createTarget = function(name, id, inputsByKind,
    properties, json) {
  var node = new exports.Target(id, json);
  this.engine.targets.addNode(node);
  node.inputsByKind = inputsByKind;
  for (var kind in inputsByKind) {
    var inputs = inputsByKind[kind];
    for (var i = 0; i < inputs.length; i++) {
      node.addParent(inputs[i]);
    }
  }
  node.properties = properties;
  node.rule = this;
  node.engine = this.engine;
  node.name = name;
  return node;
};

exports.Rule.prototype.getOutputsFor = function(target, kind) {
 return [];
};

exports.StaticFile = function(engine) {
  this.engine = engine;
};

exports.StaticFile.prototype = new exports.Rule();
exports.StaticFile.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var fileName = target.id;
  if (fileName.indexOf("//") === 0) {
    fileName = fileName.substring(2).replace(":","/");
  }
  if (fs.lstatSync(fileName).isDirectory()) {
    target.outs = [entity.getDirectoryNode(fileName, target.rule.engine.actions)];
  } else {
    target.outs = [entity.getFileNode(fileName, "src_file", target.rule.engine.actions)];
  }
  return target.outs;
};

exports.getAllOutputsRecursiveFor = function(targets, kind, filter,
    results, visited) {
  if (!targets) {
    return [];
  }
  if (!results) {
    results = [];
    exports.getAllOutputsRecursiveFor(
        targets, kind, filter, results, {});
    return results;
  }
  for (var i = 0; i < targets.length; i++) {
    var target = targets[i];
    if (visited[target.id]) {
      continue;
    }
    visited[target.id] = true;
    var outputs = target.rule.getOutputsFor(target, kind);
    for (var j = 0; j < outputs.length; j++) {
      if (filter) {
        if (!filter(outputs[j])) {
          continue;
        }
      }
      results.push(outputs[j]);
    }
    exports.getAllOutputsRecursiveFor(target.inputs, kind, filter,
        results, visited);
  }
};

exports.getAllOutputsFor = function(targets, kind, filter) {
  var results = [];
  if (!targets) {
    return results;
  }
  for (var i = 0; i < targets.length; i++) {
    var target = targets[i];
    var outputs = target.rule.getOutputsFor(target, kind);
    for (var j = 0; j < outputs.length; j++) {
      if (filter) {
        if (!filter(outputs[j])) {
          continue;
        }
      }
      results.push(outputs[j]);
    }
  }
  return results;
};

exports.fileFilter = function(kind, extension) {
  return function(output) {
    if (output.__proto__ == entity.File.prototype &&
        output.kind == kind) {
      if (extension) {
        return entity.File.prototype &&
            output.getPath().endsWith(extension);
      }
      return true;
    }
    return false;
  };
};

exports.propertyFilter = function(name) {
  return function(output) {
        return output.__proto__ == entity.Property.prototype &&
            output.key == name;
  };
};

exports.allFilesFilter = function(output) {
  return output.__proto__ == entity.File.prototype;
};

exports.allFilesAndDirsFilter = function(output) {
  return exports.allFilesFilter(output) || output.__proto__ == entity.Directory.prototype;
};
