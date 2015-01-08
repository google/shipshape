// sh_rules allows shell scripts to be executed as tests.
//
// Each input of a sh_test rule of kind 'src' is run as part of a ShTestRun
// action. The first script to fail terminates the test.
//
// Every ShTestRun action generated for the same sh_test rule depends on every
// target of kind 'data' declared by that rule.
//
// Scripts are executed in the campfire root directory.
//
// See //buildtools/test/CAMPFIRE for an example use of sh_test.

var fs = require("fs");
var path = require("path");
var util = require("util");
var rule = require("./rule.js");
var entity = require("./entity.js");

function ShTestRun(owner, scripts, tools, data) {
  this.init("ShTestRun for " + owner.id);
  this.mnemonic = "ShTestRun";
  this.owner = owner;
  this.scripts = scripts;
  this.addAllInputs(scripts);
  this.data = data;
  this.addAllInputs(data);
  this.tools = tools;
  this.addAllInputs(tools);
  this.version = "56167318-4ca5-4faa-93d8-a8d372aa5505";
  this.logFile = new entity.File(owner.getRoot("test") + ".testlog",
      "test_log");
  owner.rule.engine.actions.addNode(this.logFile);
  this.logFile.addParent(this);
}
ShTestRun.prototype = new entity.Action;
ShTestRun.prototype.createScript = function() {
  return [
    "set -e",
    "cd " + this.owner.rule.engine.campfireRoot
  ].concat(this.scripts.map(function (script) {
    return script.getPath();
  }));
}

ShTest = function(engine) {
  this.engine = engine;
}
ShTest.prototype = new rule.Rule;
ShTest.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  target.outs = [];
  var srcs = rule.getAllOutputsFor(target.inputsByKind["srcs"], "build",
      rule.allFilesFilter);
  var data = rule.getAllOutputsFor(target.inputsByKind["data"], "build",
      rule.allFilesFilter);
  var tools = rule.getAllOutputsFor(target.inputsByKind["tools"], "build");
  var toolBinaries = tools.filter(function(out) {
    return out.kind.endsWith("_shell") || out.kind.endsWith("_executable");
  });
  if (kind == "test") {
    var shTestRun = new ShTestRun(target, srcs, toolBinaries, data);
    target.rule.engine.actions.addNode(shTestRun);
    target.outs.push(shTestRun.logFile);
  }
  return target.outs;
}

exports.register = function(engine) {
  engine.addRule("sh_test", new ShTest(engine));
}
