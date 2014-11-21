var fs = require("fs");
var path = require("path");
var util = require("util");
var rule = require("./rule.js");
var entity = require("./entity.js");

exports.Javac = function(owner, srcs, jars) {
  this.init("Javac for " + owner.id);
  this.mnemonic = "Javac";
  this.owner = owner;
  this.srcs = srcs;
  this.jars = jars;
  this.addAllInputs(srcs);
  this.addAllInputs(jars);
  this.version = "8e6c143c-ea67-43b9-99ba-bc28a8c60cc7";
  this.outputDir = new entity.Directory(owner.getRoot("gen"));
  owner.rule.engine.actions.addNode(this.outputDir);
  this.jdkPath = owner.getProperty("jdk_path");
  this.addParent(this.jdkPath);
  this.outputDir.addParent(this);
  this.javacopts = owner.getProperty("javacopts");
  if (this.javacopts) {
    this.addParent(this.javacopts);
  }
};

exports.Javac.prototype = new entity.ProcessAction();
exports.Javac.prototype.commandBuilder = function() {
  return this.jdkPath.value + "bin/javac";
};

exports.Javac.prototype.argsBuilder = function() {
  var args = [];
  for (var i = 0; i < this.srcs.length; i++) {
    args.push(this.srcs[i].getPath());
  }
  var cp = [];
  for (var j = 0; j < this.jars.length; j++) {
    cp.push(this.jars[j].getPath());
  }
  if (cp.length > 0) {
    args.push("-cp");
    args.push(cp.join(":"));
  }
  args.push("-d");
  args.push(this.outputDir.getPath());
  if (this.javacopts) {
    args.push(this.javacopts.value);
  }
  return args;
};

exports.Javac.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + this.outputDir.getPath());
  return lines;
};

exports.Jar = function(owner, root) {
  this.init("Jar for " + owner.id);
  this.mnemonic = "Jar";
  this.owner = owner;
  this.root = root;

  this.outputJar = new entity.File(owner.getRoot("bin") + ".jar",
      "java_jar");
  owner.rule.engine.actions.addNode(this.outputJar);
  this.addParent(root);
  this.outputJar.addParent(this);
  this.version = "8e6c143c-ea67-43b9-99ba-bc28a8c60cc7";
  this.jdkPath = owner.getProperty("jdk_path");
  this.addParent(this.jdkPath);
};

exports.Jar.prototype = new entity.ProcessAction();
exports.Jar.prototype.commandBuilder = function() {
  return this.jdkPath.value + "bin/jar";
};

exports.Jar.prototype.argsBuilder = function() {
  var args = [];
  args.push("-cf");
  args.push(this.outputJar.getPath());
  args.push("-C");
  args.push(this.root.getPath());
  args.push(".");
  return args;
};

exports.Jar.prototype.createScript = function() {
  var lines = entity.ProcessAction.prototype.createScript.call(this);
  lines.unshift("mkdir -p " + path.dirname(this.outputJar.getPath()));
  return lines;
};

function JavaShell(owner, jars) {
  this.init("JavaShell for " + owner.id);
  this.mnemonic = "JavaShell";
  this.owner = owner;
  this.shellScript = new entity.DependentFile(owner.getRoot("bin"), "java_shell", jars);
  owner.rule.engine.actions.addNode(this.shellScript);
  this.shellScript.addParent(this);
  this.mainClass = owner.properties.main_class;
  owner.rule.engine.actions.addNode(this.mainClass);
  this.addParent(this.mainClass);
  this.jars = jars;
  this.addAllInputs(jars);
  this.jdkPath = owner.getProperty("jdk_path");
  this.addParent(this.jdkPath);
  this.version = "479c7634-5e5a-48af-94c0-08d368b41d38";
}

JavaShell.prototype = new entity.Action();
JavaShell.prototype.createScript = function() {
  var cp = [];
  for (var i = 0 ; i < this.jars.length; i++) {
    cp.push(fs.realpathSync(this.jars[i].getPath()));
  }
  return [
      "mkdir -p " + path.dirname(this.shellScript.getPath()),
      "echo '#!/bin/sh' > " + this.shellScript.getPath(),
      "echo 'ROOT=`dirname $0`' >> " + this.shellScript.getPath(),
      "echo '" + this.jdkPath.value + "bin/java -cp " + cp.join(":") +
          " " + this.mainClass.value + " \"$@\"' >> " +
          this.shellScript.getPath(),
      "echo 'exit $?' >> " + this.shellScript.getPath(),
      "chmod +x " + this.shellScript.getPath()];
};

JUnit = function(owner, jars) {
  this.init("JUnit for " + owner.id);
  this.mnemonic = "JUnit";
  this.owner = owner;
  this.jars = jars;
  this.addAllInputs(jars);
  this.logFile = new entity.File(owner.getRoot("test") + ".testlog",
      "test_log");
  owner.rule.engine.actions.addNode(this.logFile);
  this.logFile.addParent(this);
  this.version = "3c33e468-d3b1-44bf-bf26-ecd31ab4aa56";

  this.testClass = owner.properties.test_class;
  owner.rule.engine.actions.addNode(this.testClass);
  this.addParent(this.testClass);
  this.jdkPath = owner.getProperty("jdk_path");
  this.addParent(this.jdkPath);
  this.junitPath = owner.getProperty("junit_path");
  this.addParent(this.junitPath);
};

JUnit.prototype = new entity.ProcessAction();
JUnit.prototype.commandBuilder = function() {
  return this.jdkPath.value + "bin/java";
};

JUnit.prototype.argsBuilder = function() {
  var args = [];
  args.push("-cp");
  var cp = [this.junitPath.value];
  for (var i = 0; i < this.jars.length; i++) {
    cp.push(this.jars[i].getPath());
  }
  args.push(cp.join(":"));
  args.push("org.junit.runner.JUnitCore");
  args.push(this.testClass.value);
  return args;
};

function MakeDeployJar(owner, jars) {
  this.init("MakeDeployJar for " + owner.id);
  this.mnemonic = "MakeDeployJar";
  this.owner = owner;
  this.addAllInputs(jars);
  this.jars = jars;
  this.deployJar = new entity.File(owner.getRoot('bin') + '.jar',
                                   'java_deploy_jar');
  owner.rule.engine.actions.addNode(this.deployJar);
  this.deployJar.addParent(this);
  this.mainClass = owner.getProperty('main_class');
  this.addParent(this.mainClass);
  this.version = "33bda264-287f-4d70-b65b-3b6770335ad7";
  this.jdkPath = owner.getProperty("jdk_path");
  this.addParent(this.jdkPath);
}

MakeDeployJar.prototype = new entity.Action();
MakeDeployJar.prototype.createScript = function() {
  var jars = [];
  for (var i = 0 ; i < this.jars.length; i++) {
    jars.push(fs.realpathSync(this.jars[i].getPath()));
  }
  var jdkPath = this.jdkPath.value;
  if (jdkPath.indexOf('/') !== 0) {
      jdkPath = path.join(this.owner.rule.engine.campfireRoot, jdkPath);
  }
  return [
     "TMPDIR=" + this.owner.getRoot("gen"),
     "JARS=\"" + jars.join(" ") + "\"",
     "mkdir -p " + path.dirname(this.deployJar.getPath()),
     "DEPLOYJAR=`readlink -f \"" + this.deployJar.getPath() + "\"`",
     "if [ -f $DEPLOYJAR ];",
     "then",
     "  rm $DEPLOYJAR",
     "fi",
     "mkdir -p $TMPDIR",
     "cd $TMPDIR",
     "echo Main-Class: " + this.mainClass.value + " > manifest.txt",
     "for i in $JARS; do " + jdkPath + "bin/jar -xf $i; done;",
     jdkPath + "bin/jar -cfm $DEPLOYJAR manifest.txt ."
  ];
};

JavaLibrary = function(engine) {
  this.engine = engine;
};

JavaLibrary.prototype = new rule.Rule();
JavaLibrary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var srcs = rule.getAllOutputsFor(target.inputsByKind["srcs"], "build",
      rule.fileFilter("src_file", ".java"));
  var jars = rule.getAllOutputsFor(target.inputsByKind["jars"], "build",
      rule.fileFilter("java_jar"));
  var javac = new exports.Javac(target, srcs, jars);
  target.rule.engine.actions.addNode(javac);
  var jar = new exports.Jar(target, javac.outputDir);
  target.rule.engine.actions.addNode(jar);
  target.outs = [jar.outputJar];
  return target.outs;
};

JavaBinary = function(engine) {
  this.engine = engine;
};

JavaBinary.prototype = new JavaLibrary();
JavaBinary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  JavaLibrary.prototype.getOutputsFor.call(this, target, kind);
  var jars = rule.getAllOutputsRecursiveFor(target.inputsByKind["jars"],
      "build", rule.fileFilter("java_jar"));
  jars.push(target.outs[0]);
  var javaShell = new JavaShell(target, jars, "build", ".jar");
  target.rule.engine.actions.addNode(javaShell);
  target.outs.push(javaShell.shellScript);
  return target.outs;
};

JavaTest = function(engine) {
  this.engine = engine;
};

JavaTest.prototype = new JavaLibrary();
JavaTest.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  JavaLibrary.prototype.getOutputsFor.call(this, target, kind);
  if (kind == "test") {
    var jars = rule.getAllOutputsRecursiveFor(
        target.inputsByKind["jars"], "build",
        rule.fileFilter("java_jar"));
    jars.push(target.outs[0]);
    var junit = new JUnit(target, jars);
    target.rule.engine.actions.addNode(junit);
    target.outs.push(junit.logFile);
  }
  return target.outs;
};

JavaDeployJar = function(engine) {
  this.engine = engine;
};

JavaDeployJar.prototype = new rule.Rule();
JavaDeployJar.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var jars = rule.getAllOutputsRecursiveFor(target.inputsByKind["jars"],
       "build", rule.fileFilter("java_jar"));
  var makeDeployJar = new MakeDeployJar(target, jars);
  target.rule.engine.actions.addNode(makeDeployJar);
  target.outs = [makeDeployJar.deployJar];
  return target.outs;
};

JavaExternalJar = function(engine) {
  this.engine = engine;
};

JavaExternalJar.prototype = new rule.Rule();
JavaExternalJar.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var inputs = rule.getAllOutputsFor(target.inputsByKind["srcs"],
      "build", rule.fileFilter("src_file", ".jar"));

  target.outs = [];
  for (var i = 0 ; i < inputs.length; i++) {
    target.outs.push(new entity.File(inputs[i].getPath(), "java_jar"));
  }
  return target.outs;
};

exports.register = function(engine) {
  engine.addRule("java_library", new JavaLibrary(engine));
  engine.addRule("java_binary", new JavaBinary(engine));
  engine.addRule("java_test", new JavaTest(engine));
  engine.addRule("java_deploy_jar", new JavaDeployJar(engine));
  engine.addRule("java_external_jar", new JavaExternalJar(engine));
};
