var fs = require("fs");
var path = require("path");
var util = require("util");
var rule = require("./rule.js");
var entity = require("./entity.js");

var pkgDir = "campfire-out/go/pkg/linux_amd64/";
GoCompile = function(owner, srcs, pkgs, includePaths, isBinary,
    suffix) {
  var outName = path.dirname(owner.asPath());
  if (suffix) {
    outName += "_" + suffix;
  } else if (!isBinary) {
    var base = path.basename(outName);
    if (base != owner.name) {
      console.log("Only one go_library can be defined per directory " +
          "and it should be named after the directory. " +
          "Move '" + owner.id + " to its own directory. Note: " +
          "go_binary and go_test are exempt from this rule.");
      process.exit(1);
    }
  }
  this.init("GoCompile for " + owner.id +"#" + outName);
  this.mnemonic = "GoCompile";
  this.owner = owner;
  this.dir = path.dirname(srcs[0].getPath());
  this.pkgName = isBinary ? "main" : this.dir;
  this.srcs = srcs;
  this.addAllInputs(srcs);
  this.pkgs = pkgs;
  this.addAllInputs(pkgs);
  this.includePaths = includePaths;
  this.addAllInputs(includePaths);

  this.version = "6616eed2-3e68-457e-ab8b-3830b879b6a9";
  var outPath = path.join(owner.getRoot("gen"), outName + ".o");
  this.outputFile = new entity.File(outPath, "go_object");
  owner.rule.engine.actions.addNode(this.outputFile);
  this.outputFile.addParent(this);

  this.goPath = owner.getProperty("go_path");
  this.addParent(this.goPath);

  this.goopts = owner.getProperty("goopts");
  if (this.goopts) {
    this.addParent(this.goopts);
  }
};

GoCompile.prototype = new entity.ProcessAction();
GoCompile.prototype.commandBuilder = function() {
  return this.goPath.value +"/6g";
};

GoCompile.prototype.argsBuilder = function() {
  var args = ["-p", this.pkgName, "-complete", "-o",
      this.outputFile.getPath(),
              "-D", "_" + this.dir ];
  for (var i = 0; i < this.includePaths.length; i++) {
    args.push("-I");
    args.push(this.includePaths[i].value);
  }
  args.push("-I");
  args.push(pkgDir);
  if (this.goopts) {
    args.push(this.goopts.value);
  }
  for (var j = 0; j < this.srcs.length; j++) {
    args.push(this.srcs[j].getPath());
  }
  args.push("1>&2");
  return args;
};

GoCompile.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.outputFile.getPath()));
  return lines;
};

GoPacker = function(owner, object, suffix) {
  var pkgName = path.dirname(owner.asPath());
  if (suffix) {
    pkgName += "_" + suffix;
  }
  this.init("GoPacker for " + owner.id + "#" + pkgName);
  this.mnemonic = "GoPacker";
  this.owner = owner;
  this.object = object;
  this.addParent(object);
  this.version = "9c72d821-fded-45c2-ba81-8633a8987c06";
  this.archive = new entity.File(pkgDir + pkgName + ".a", "go_archive");
  owner.rule.engine.actions.addNode(this.archive);
  this.archive.addParent(this);
  this.goPath = owner.getProperty("go_path");
  this.addParent(this.goPath);
};

GoPacker.prototype = new entity.ProcessAction();
GoPacker.prototype.commandBuilder = function() {
  return this.goPath.value +"/pack";
};

GoPacker.prototype.argsBuilder = function() {
  return ["cv", this.archive.getPath(), this.object.getPath(), "1>&2"];
};

GoPacker.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.archive.getPath()));
  lines.unshift("rm -f " + this.archive.getPath());
  return lines;
};

GoLinker = function(owner, object, libs, suffix) {
  this.init("GoLinker for " + owner.id);
  this.mnemonic = "GoLinker";
  this.owner = owner;
  this.object = object;
  this.addParent(object);
  this.libs = libs;
  this.addAllInputs(libs);
  this.includePaths = rule.getAllOutputsRecursiveFor(
      owner.inputsByKind["go_pkgs"], "build",
      rule.propertyFilter("go_include_path"));
  this.addAllInputs(this.includePaths);
  this.ccLibs = rule.getAllOutputsRecursiveFor(
      (owner.inputsByKind["go_pkgs"] || []).concat(
      owner.inputsByKind["cc_libs"] || []), "build",
      rule.fileFilter("cc_archive"));
  this.addAllInputs(this.ccLibs);
  this.version = "1d8115b2-d7ec-41df-bf48-46e299975c3e";
  var binPath = owner.getRoot("bin");
  if (suffix) {
    binPath = binPath + "_" + suffix;
  }
  this.output = new entity.File(binPath, "go_executable");
  owner.rule.engine.actions.addNode(this.output);
  this.output.addParent(this);

  this.goPath = owner.getProperty("go_path");
  this.addParent(this.goPath);
};

GoLinker.prototype = new entity.ProcessAction();
GoLinker.prototype.commandBuilder = function() {
  return this.goPath.value +"/6l";
};

GoLinker.prototype.argsBuilder = function() {
  var args = ["-L", pkgDir];
  for (var i = 0; i < this.includePaths.length; i++) {
    args.push("-L " + this.includePaths[i].value);
  }
  if (this.ccLibs.length > 0) {
    var extLDArgs = [];
    for (var j = 0; j < this.ccLibs.length; j++) {
      extLDArgs.push("-L" + path.dirname(this.ccLibs[j].getPath()));
      extLDArgs.push("-Wl,-rpath=" +
          path.dirname(this.ccLibs[j].getPath()));
    }
    args.push("--extldflags");
    args.push("'" + extLDArgs.join(" ") + "'");
  }
  args.push("-o");
  args.push(this.output.getPath());
  args.push(this.object.getPath());
  args.push("1>&2");
  return args;
};

GoLinker.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.output.getPath()));
  return lines;
};

GoTestMain = function(owner, srcs) {
  this.init("GoTestMain for " + owner.id);

  var generatorName = owner.engine.settings.properties["go_testmain_generator"];
  if (!generatorName || generatorName == "") {
    console.log("Missing go_testmain_generator configuration!");
    process.exit(1);
  }
  entity.ToolAction.call(this, owner, "919509eb-0af4-49d1-bcfc-511d7182bdc0", generatorName);

  this.srcs = srcs;
  this.addAllInputs(srcs);

  this.outPath = path.join(owner.getRoot("gen"), "testmain.go");
  this.outputFile = new entity.File(this.outPath, "go_object");
  owner.rule.engine.actions.addNode(this.outputFile);
  this.outputFile.addParent(this);
};
GoTestMain.prototype = Object.create(entity.ToolAction.prototype);
GoTestMain.prototype.argsBuilder = function() {
  var args = [path.dirname(this.owner.asPath()) + "_test",
              this.outPath];
  for (var i = 0; i < this.srcs.length; i++) {
    args.push(this.srcs[i].getPath());
  }
  args.push("1>&2");
  return args;
};

GoTestMain.prototype.createScript = function() {
  var lines = entity.ToolAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.outputFile.getPath()));
  return lines;
};

GoTestExecute = function(owner, testExecutable) {
  this.init("GoTestExecute for " + owner.id);
  this.mnemonic = "GoTestExecute";
  this.version = "b765b088-1d65-4226-8423-07871fe58019";
  this.owner = owner;
  this.logFile = new entity.File(owner.getRoot("test") + ".testlog",
      "test_log");
  owner.rule.engine.actions.addNode(this.logFile);
  this.logFile.addParent(this);
  this.testExecutable = testExecutable;
  this.addParent(testExecutable);
};

GoTestExecute.prototype = new entity.ProcessAction();
GoTestExecute.prototype.commandBuilder = function() {
  return this.testExecutable.getPath();
};

GoTestExecute.prototype.argsBuilder = function() {
  return [];
};

GoLibrary = function(engine) {
  this.engine = engine;
};

GoLibrary.prototype = new rule.Rule();
GoLibrary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var srcs = rule.getAllOutputsFor(
      target.inputsByKind["srcs"], kind, rule.fileFilter("src_file",
      ".go"));
  var pkgs = rule.getAllOutputsFor(
      target.inputsByKind["go_pkgs"], kind,
      rule.fileFilter("go_archive"));
  var includePaths = rule.getAllOutputsFor(
      target.inputsByKind["go_pkgs"], kind,
      rule.propertyFilter("go_include_path"));

  var goCompile = new GoCompile(target, srcs, pkgs, includePaths);
  target.rule.engine.actions.addNode(goCompile);
  var packer = new GoPacker(target, goCompile.outputFile);
  target.outs = [packer.archive];

  if (kind == "test") {
    var testSrcs = rule.getAllOutputsFor(
        target.inputsByKind["go_tests"], kind,
        rule.fileFilter("src_file", ".go"));
    if (testSrcs.length > 0) {
      var testCompile = new GoCompile(target, srcs.concat(testSrcs),
          pkgs, includePaths, false, "test");
      target.rule.engine.actions.addNode(testCompile);
      var testPacker = new GoPacker(target, testCompile.outputFile,
          "test");

      var testMain = new GoTestMain(target, testSrcs);
      target.rule.engine.actions.addNode(testMain);
      var mainCompile = new GoCompile(target, [testMain.outputFile],
          [testPacker.archive], [], true, "testmain");
      target.rule.engine.actions.addNode(mainCompile);
      var linker = new GoLinker(target, mainCompile.outputFile,
          [testPacker.archive], "test");

      var testExecute = new GoTestExecute(target, linker.output);
      target.rule.engine.actions.addNode(testExecute);

      target.outs.push(testExecute.logFile);
    }
  }

  return target.outs;
};

GoBinary = function(engine) {
  this.engine = engine;
};

GoBinary.prototype = new rule.Rule();
GoBinary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var srcs = rule.getAllOutputsFor(target.inputsByKind["srcs"],
      kind, rule.fileFilter("src_file", ".go"));
  var pkgs = rule.getAllOutputsFor(target.inputsByKind["go_pkgs"],
      kind, rule.fileFilter("go_archive"));
  var includePaths = rule.getAllOutputsFor(
      target.inputsByKind["go_pkgs"], kind,
      rule.propertyFilter("go_include_path"));

  var goCompile = new GoCompile(target, srcs, pkgs, includePaths, true);
  target.rule.engine.actions.addNode(goCompile);
  var linker = new GoLinker(target, goCompile.outputFile, pkgs);
  target.outs = [linker.output];
  return target.outs;
};

GoExternalLib = function(engine) {
  this.engine = engine;
};

GoExternalLib.prototype = new rule.Rule();
GoExternalLib.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var inputs = rule.getAllOutputsFor(target.inputsByKind["srcs"],
      kind, rule.fileFilter("src_file", ".a"));

  var outputs = [];
  for (var i =0 ; i < inputs.length; i++) {
    outputs.push(new entity.File(inputs[i].getPath(), "go_archive"));
  }

  var includePath = target.getProperty("go_include_path");
  if (includePath) {
    outputs.push(includePath);
  }

  target.outs = outputs;
  return outputs;
};

exports.register = function(engine) {
  engine.addRule("go_library", new GoLibrary(engine));
  engine.addRule("go_binary", new GoBinary(engine));
  engine.addRule("go_external_lib", new GoExternalLib(engine));
};
