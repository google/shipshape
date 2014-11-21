var fs = require("fs");
var path = require("path");
var util = require("util");
var rule = require("./rule.js");
var entity = require("./entity.js");

// A property that, when specified on a cc_external_lib, requires
// additional flags to be passed to the linker. Takes a list of strings
// as its value.
var extraLinkFlagsProperty = "cc_extra_link_flags";

var includePathProperty = "cc_include_path";
var systemIncludePathProperty = "cc_system_include_path";

// Append these copts when compiling only this target. List of strings.
var localCoptsProperty = "cc_local_copts";

// Append these copts when compiling this target or any target that
// depends on it. Useful on cc_library or cc_external_lib. List of
// strings.
var exportedCoptsProperty = "cc_exported_copts";

// owner: This action's owning target.
// src: The source file to compile.
// includePaths: an array of properties of strings.
//               Added as inputs to the CppCompile action and used as
//               -i include search paths.
// systemIncludePaths: an array of properties of strings.
//                     Added as inputs to the CppCompile action and
//                     used as -isystem include search paths.
// hdrs: an array of additional inputs to this action; affects only
//       the computation of dependencies.
// importedCopts: an array of properties of arrays of strings.
//                These are added as inputs to the CppCompile action
//                and are flattened and added to the copts passed to
//                the compiler. Intended to support pulling in exported
//                copts from dependencies.
CppCompile = function(owner, src, includePaths, systemIncludePaths,
    hdrs, importedCopts) {
  var srcPath = src.getPath();
  this.init("CppCompile for " + owner.id +"#" +
      path.relative(path.dirname(owner.asPath()), srcPath));
  this.owner = owner;
  this.mnemonic = "CppCompile";
  this.src = src;
  this.addParent(src);
  this.includePaths = includePaths;
  this.addAllInputs(includePaths);
  this.language = srcPath.substring(srcPath.length - 2) == ".c" ? "cc" : "cxx";

  this.systemIncludePaths = systemIncludePaths;
  this.addAllInputs(systemIncludePaths);

  this.hdrs = hdrs;
  this.addAllInputs(hdrs);

  this.version = "ed518841-b35e-42a9-962e-2283e2634070";
  var outPath = path.join(owner.getRoot("gen"),
      path.dirname(srcPath),
      path.basename(srcPath, this.language == "cxx" ? ".cc" : ".c") + ".o");
  this.outputFile = new entity.File(outPath, "cc_object");
  owner.rule.engine.actions.addNode(this.outputFile);
  this.outputFile.addParent(this);

  this.crosstool = owner.getProperty("crosstool_" + this.language + "_path");
  this.addParent(this.crosstool);

  if (this.language == "cxx") {
    this.copts = owner.getProperty("copts");
    if (this.copts) {
      this.addParent(this.copts);
    }
  }

  this.importedCopts = importedCopts;
  this.addAllInputs(importedCopts);

  this.localCopts = owner.getProperty(localCoptsProperty);
  if (this.localCopts) {
    this.addParent(this.localCopts);
  }

  this.exportedCopts = owner.getProperty(exportedCoptsProperty);
  if (this.exportedCopts) {
    this.addParent(this.exportedCopts);
  }
};

CppCompile.prototype = new entity.ProcessAction();
CppCompile.prototype.commandBuilder = function() {
  return this.crosstool.value;
};

CppCompile.prototype.argsBuilder = function() {
  var args = [];
  args.push("-c");
  args.push(this.src.getPath());
  args.push("-o");
  args.push(this.outputFile.getPath());
  for (i = 0; i < this.includePaths.length; i++) {
    args.push("-I" + this.includePaths[i].value);
  }
  for (i = 0; i < this.systemIncludePaths.length; i++) {
    args.push("-ISYSTEM" + this.systemIncludePaths[i].value);
  }
  args.push("-I.");
  if (this.copts) {
    args = args.concat(this.copts.value);
  }
  if (this.importedCopts) {
    args = args.concat(this.importedCopts // property<string[]>[]
        .map(function (p) { return p.value; }) // string[][]
        .reduce(function (p, n) { return p.concat(n); }, []));
        // string[]
  }
  if (this.exportedCopts) {
    args = args.concat(this.exportedCopts.value);
  }
  if (this.localCopts) {
    args = args.concat(this.localCopts.value);
  }

  return args;
};

CppCompile.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.outputFile.getPath()));
  return lines;
};

Archiver = function(owner, objects) {
  this.init("Archiver for " + owner.id);
  this.mnemonic = "Archiver";
  this.owner = owner;
  this.objects = objects;
  this.addAllInputs(objects);
  this.version = "1e0ed4b6-bd9e-4943-90ce-a6ba329f9f15";
  this.archive = new entity.File(owner.getRoot("bin") + ".a",
      "cc_archive");
  owner.rule.engine.actions.addNode(this.archive);
  this.archive.addParent(this);
};

Archiver.prototype = new entity.ProcessAction();
Archiver.prototype.commandBuilder = function() {
  return "/usr/bin/ar";
};

Archiver.prototype.argsBuilder = function() {
  var args = [];
  args.push("cr");
  args.push(this.archive.getPath());
  for (var i = 0; i < this.objects.length; i++) {
    args.push(this.objects[i].getPath());
  }
  return args;
};

Archiver.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.archive.getPath()));
  return lines;
};

Linker = function(owner, libs, extraFlags) {
  this.init("Linker for " + owner.id);
  this.mnemonic = "Linker";
  this.owner = owner;
  this.libs = libs;
  this.addAllInputs(libs);
  this.extraFlags = extraFlags;
  this.version = "784663ba-1feb-4e61-a1a8-2180e6b91dea";
  this.output = new entity.File(owner.getRoot("bin"), "cc_executable");
  owner.rule.engine.actions.addNode(this.output);
  this.output.addParent(this);

  this.crosstool = owner.getProperty("crosstool_cxx_path");
  this.addParent(this.crosstool);
};

Linker.prototype = new entity.ProcessAction();
Linker.prototype.commandBuilder = function() {
  return this.crosstool.value;
};

Linker.prototype.argsBuilder = function() {
  var args = [];
  args.push("-o");
  args.push(this.output.getPath());
  args.push("-pthread");
  for (var i = 0; i < this.libs.length; i++) {
    args.push(this.libs[i].getPath());
  }
  // HACK HACK HACK: By adding the libs twice to the command line we can
  // help eliminate the library topological ordering issue, if a
  // reference is not found in the first group it will be most likely
  // found in the second group. If the chain of dependency is longer
  // than 2 then this will fail and manual lib ordering will be
  // required.
  for (var j = 0; j < this.libs.length; j++) {
    args.push(this.libs[j].getPath());
  }
  for (var k = 0; k < this.extraFlags.length; k++) {
    // It may be possible to detect linker flag conflicts here, or at
    // least to deduplicate them. This may be out of scope for this
    // tool.
    args.append(this.extraFlags[k].value);
  }
  return args;
};

Linker.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.output.getPath()));
  return lines;
};

CcTestRun = function(owner, binary) {
  this.init("CcTestRun for " + owner.id);
  this.mnemonic = "CcTestRun";
  this.owner = owner;
  this.binary = binary;
  this.addAllInputs([binary]);
  this.version = "9d0711d6-6a74-4edf-bf7d-339846a9c175";
  this.logFile = new entity.File(owner.getRoot("test") + ".testlog",
      "test_log");
  owner.rule.engine.actions.addNode(this.logFile);
  this.logFile.addParent(this);
};

CcTestRun.prototype = new entity.Action();

CcTestRun.prototype.createScript = function() {
  return [
    "cd " + this.owner.rule.engine.campfireRoot,
    this.binary.getPath()
  ];
};

CcLibrary = function(engine) {
  this.engine = engine;
};

CcLibrary.prototype = new rule.Rule();
CcLibrary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var srcs = rule.getAllOutputsFor(target.inputsByKind["srcs"],
      "build", rule.fileFilter("src_file", ".cc"));
  srcs.append(rule.getAllOutputsFor(target.inputsByKind["srcs"],
      "build", rule.fileFilter("src_file", ".c")));
  var hdrs = rule.getAllOutputsRecursiveFor(target.inputs, "build",
      rule.fileFilter("src_file", ".h"));

  var importedCopts = rule.getAllOutputsFor(
      target.inputsByKind["cc_libs"], "build",
      rule.propertyFilter(exportedCoptsProperty));
  var includePaths = rule.getAllOutputsFor(
      target.inputsByKind["cc_libs"], "build",
      rule.propertyFilter(includePathProperty));
  var systemIncludePaths = rule.getAllOutputsFor(
      target.inputsByKind["cc_libs"], "build",
      rule.propertyFilter(systemIncludePathProperty));
  var objects = [];
  for (var i = 0; i < srcs.length; i++) {
    var cppCompile = new CppCompile(target, srcs[i], includePaths,
        systemIncludePaths, hdrs, importedCopts);
    target.rule.engine.actions.addNode(cppCompile);
    objects.push(cppCompile.outputFile);
  }
  var archiver = new Archiver(target, objects);
  target.outs = [archiver.archive];

  var includePath = target.getProperty(includePathProperty);
  if (includePath) {
    target.outs.push(includePath);
  }

  var exportedCopts = target.getProperty(exportedCoptsProperty);
  if (exportedCopts) {
    target.outs = target.outs.concat(exportedCopts);
  }

  return target.outs;
};

CcBinary = function(engine) {
  this.engine = engine;
};

CcBinary.prototype = new rule.Rule();
CcBinary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var libs = rule.getAllOutputsRecursiveFor(
      target.inputsByKind["cc_libs"], "build",
      rule.fileFilter("cc_archive"));
  var extraLinkFlags = rule.getAllOutputsRecursiveFor(
      target.inputsByKind["cc_libs"], "build",
      rule.propertyFilter(extraLinkFlagsProperty));

  var linker = new Linker(target, libs, extraLinkFlags);
  target.outs = [linker.output];
  return target.outs;
};

CcTest = function(engine) {
  this.engine = engine;
};

CcTest.prototype = new CcBinary();
CcTest.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  CcBinary.prototype.getOutputsFor.call(this, target, kind);
  if (kind == "test") {
    var executable = target.outs[0];
    var ccTestRun = new CcTestRun(target, executable);
    target.rule.engine.actions.addNode(ccTestRun);
    target.outs.push(ccTestRun.logFile);
  }
  return target.outs;
};

CCExternalLib = function(engine) {
  this.engine = engine;
};

CCExternalLib.prototype = new rule.Rule();
CCExternalLib.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var inputs = [];
  inputs.append(rule.getAllOutputsFor(target.inputsByKind["srcs"],
      "build", rule.fileFilter("src_file", ".so")));
  inputs.append(rule.getAllOutputsFor(target.inputsByKind["srcs"],
      "build", rule.fileFilter("src_file", ".a")));

  var outputs = [];
  for (var i = 0 ; i < inputs.length; i++) {
    outputs.push(new entity.File(inputs[i].getPath(), "cc_archive"));
  }

  var includePath = target.getProperty(includePathProperty);
  if (includePath) {
    outputs.push(includePath);
  }

  var extraLinkFlags = target.getProperty(extraLinkFlagsProperty);
  if (extraLinkFlags) {
      outputs = outputs.concat(extraLinkFlags);
  }

  var exportedCopts = target.getProperty(exportedCoptsProperty);
  if (exportedCopts) {
    outputs = outputs.concat(exportedCopts);
  }

  return outputs;
};

exports.register = function(engine) {
  engine.addRule("cc_library", new CcLibrary(engine));
  engine.addRule("cc_binary", new CcBinary(engine));
  engine.addRule("cc_test", new CcTest(engine));
  engine.addRule("cc_external_lib", new CCExternalLib(engine));
};
