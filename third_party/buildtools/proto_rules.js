var fs = require("fs");
var path = require("path");
var util = require("util");
var rule = require("./rule.js");
var entity = require("./entity.js");
var ccRules = require("./cc_rules.js");

function JavaProtoCompile(owner, srcs, jars) {
  this.init("JavaProtoCompile for " + owner.id);
  this.mnemonic = "JavaProtoCompile";
  this.owner = owner;
  this.tmpDir = new entity.Directory(
      path.join(owner.getRoot("gen"), "java"));
  owner.rule.engine.actions.addNode(this.tmpDir);
  this.tmpDir.addParent(this);
  this.outputJar = new entity.File(
      owner.getRoot("bin") + ".jar", "java_jar");
  owner.rule.engine.actions.addNode(this.outputJar);
  this.outputJar.addParent(this);
  this.srcs = srcs;
  this.addAllInputs(srcs);
  this.jars =jars;
  this.addAllInputs(jars);
  this.jdkPath = owner.getProperty("jdk_path");
  this.addParent(this.jdkPath);
  this.protocPath = owner.getProperty("protoc_path");
  this.addParent(this.protocPath);
  this.version = "31c2db0a-82b6-4cb2-b94f-4cb9d6552581";
  this.javacopts = owner.getProperty("javacopts");
  if (this.javacopts) {
    this.addParent(this.javacopts);
  }
}

JavaProtoCompile.prototype = new entity.Action();
JavaProtoCompile.prototype.createScript = function() {
  var cp = [];
  for (var i = 0; i < this.jars.length; i++) {
    cp.push(this.jars[i].getPath());
  }
  var args = [];
  if (cp.length > 0) {
    args.push("-cp");
    args.push(cp.join(":"));
  }
  if (this.javacopts) {
    args.push(this.javacopts.value);
  }
  return [
    "SRCS=" + this.tmpDir.getPath() +"srcs",
    "mkdir -p $SRCS",
    "CLASSES=" + this.tmpDir.getPath() + "/classes",
    "mkdir -p $CLASSES",
    this.protocPath.value + "/protoc --java_out=$SRCS " +
        this.srcs[0].getPath(), "find $SRCS -name '*.java'| xargs " +
        this.jdkPath.value + "bin/javac " + args.join(" ") +
        " -d $CLASSES", "mkdir -p " +
        path.dirname(this.outputJar.getPath()), this.jdkPath.value +
        "bin/jar -cf " + this.outputJar.getPath() + " -C $CLASSES ."
  ];
};


function CcProtoCompile(owner, src, includePaths, libs) {
  this.init("CcProtoCompile for " + owner.id);
  this.mnemonic = "CcProtoCompile";
  this.owner = owner;
  this.outputDir = new entity.Directory(
      path.join(owner.getRoot("gen"),"cxx"));
  owner.rule.engine.actions.addNode(this.outputDir);
  this.outputDir.addParent(this);
  this.src = src;
  this.addParent(this.src);
  this.includePaths = includePaths;
  this.addAllInputs(includePaths);
  this.libs = libs;
  this.addAllInputs(libs);
  this.outputArchive = new entity.File(
      owner.getRoot("bin") + ".a", "cc_archive");
  owner.rule.engine.actions.addNode(this.outputArchive);
  this.outputArchive.addParent(this);

  this.protocPath = owner.getProperty("protoc_path");
  this.addParent(this.protocPath);
  this.crosstool = owner.getProperty("crosstool_cxx_path");
  this.addParent(this.crosstool);

  this.version = "61adfeab-7514-4a4c-b60f-c6c3b04dc205";
}

CcProtoCompile.prototype = new entity.Action();
CcProtoCompile.prototype.createScript = function() {
  var includePaths = [];
  for (var i = 0; i < this.includePaths.length; i++) {
    var includePath = this.includePaths[i].value;
    if (includePath.indexOf('/') !== 0) {
      includePath = path.join(this.owner.rule.engine.campfireRoot,
           includePath);
    }
    includePaths.push("-I" + includePath);
  }

  var outputArchive = path.join(
      this.owner.rule.engine.campfireRoot,
      this.outputArchive.getPath());

  return [
    "OUTDIR=" + this.outputDir.getPath(),
    "mkdir -p $OUTDIR",
    "mkdir -p " + path.dirname(outputArchive),
    this.protocPath.value + "/protoc --cpp_out=$OUTDIR " +
        this.src.getPath(),
    "cd $OUTDIR",
    "for i in $(find -name '*.cc');do " + this.crosstool.value +
         " -c -I. " + includePaths.join(" ") +" $i -o $i.o; done;",
    "/usr/bin/ar cr " + outputArchive + " $(find -name '*.o')",
    "cd " + this.owner.rule.engine.campfireRoot,
  ];
};

var goPkgDir = "campfire-out/go/pkg/linux_amd64/";

function GoProtoCompile(owner, src, includePaths, libs) {
  this.init("GoProtoCompile for " + owner.id);
  this.mnemonic = "GoProtoCompile";
  this.owner = owner;
  this.outputDir =
      new entity.Directory(path.join(owner.getRoot("gen"), "go"));
  owner.rule.engine.actions.addNode(this.outputDir);
  this.outputDir.addParent(this);
  this.src = src;
  this.addParent(this.src);
  this.includePaths = includePaths;
  this.addAllInputs(includePaths);
  this.libs = libs;
  this.addAllInputs(libs);

  this.outputArchive = new entity.File(goPkgDir + owner.asPath() + ".a",
      "go_archive");
  owner.rule.engine.actions.addNode(this.outputArchive);
  this.outputArchive.addParent(this);

  this.protocPath = owner.getProperty("protoc_path");
  this.addParent(this.protocPath);
  this.goPath = owner.getProperty("go_path");
  this.addParent(this.goPath);
  this.version = "cb39dac6-b8ad-44c5-a341-535df750286f";
}

getProtoImportMappings = function(owner, output, seen) {
  if (!output) {
    results = [];
    this.getProtoImportMappings(owner, results, {});
    return results;
  }
  if (owner.inputsByKind["proto_libs"]) {
    for (var i=0; i < owner.inputsByKind["proto_libs"].length; i++) {
      var input = owner.inputsByKind["proto_libs"][i];
      if (!seen[input.id]) {
        seen[input.id] = true;
        if (input.rule.__proto__ == ProtoLibrary.prototype) {
          getProtoImportMappings(input, output, seen);
        }
      }
    }
  }
  var srcs = rule.getAllOutputsFor(owner.inputsByKind["srcs"], "build",
      rule.fileFilter("src_file", ".proto"));
  output.push("M" + srcs[0].getPath() +"=" + owner.asPath());
};

GoProtoCompile.prototype = new entity.Action();
GoProtoCompile.prototype.createScript = function() {
  var includePaths = [];
  for (var i = 0; i < this.includePaths.length; i++) {
    var includePath = this.includePaths[i].value;
    if (includePath.indexOf('/') !== 0) {
      includePath = path.join(this.owner.rule.engine.campfireRoot,
          includePath);
    }
    includePaths.push("-I " + includePath);
  }
  includePaths.push("-I " + path.join(
      this.owner.rule.engine.campfireRoot, goPkgDir));

  var protoImportMappings =
      getProtoImportMappings(this.owner).join(",");
  if (protoImportMappings !== "") {
    protoImportMappings = "," + protoImportMappings;
  }

  var outputArchive = path.join(this.owner.rule.engine.campfireRoot,
      this.outputArchive.getPath());
  return [
    "OUTDIR=" + this.outputDir.getPath(),
    "mkdir -p $OUTDIR",
    "mkdir -p " + path.dirname(this.outputArchive.getPath()),
        this.protocPath.value + "/protoc --plugin=" +
        this.protocPath.value + "/protoc-gen-go --go_out=import_path=" +
        this.owner.asPath() + protoImportMappings + ":$OUTDIR " +
        this.src.getPath(),
    "cd $OUTDIR",
    "for i in $(find -name '*.go'); do " + this.goPath.value +
         "/6g -p " + this.owner.asPath() +
         " -complete -D _$OUTDIR -I $OUTDIR " + includePaths.join(" ") +
         " -o $i.o $i 1>&2; done;",
    "rm -f " + outputArchive,
    "find -name '*.o'| xargs " + this.goPath.value + "/pack cv " +
        outputArchive + " 1>&2",
    "cd " + this.owner.rule.engine.campfireRoot
  ];
};

ProtoLibrary = function(engine) {
  this.engine = engine;
};

ProtoLibrary.prototype = new rule.Rule();
ProtoLibrary.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var srcs = rule.getAllOutputsFor(target.inputsByKind["srcs"], "build",
      rule.fileFilter("src_file", ".proto"));
  target.outs = [];
  if (target.properties.java_api && target.properties.java_api.value) {
    var jars = [];
    jars.append(rule.getAllOutputsFor(target.inputsByKind["jars"],
        "build", rule.fileFilter("java_jar")));
    jars.append(rule.getAllOutputsFor(target.inputsByKind["proto_libs"],
        "build", rule.fileFilter("java_jar")));
    var javaProtoCompile = new JavaProtoCompile(target, srcs, jars);
    target.rule.engine.actions.addNode(javaProtoCompile);
    target.outs.push(javaProtoCompile.outputJar);
  }
  if (target.properties.cc_api && target.properties.cc_api.value) {
    var includePaths = [];
    includePaths.append(rule.getAllOutputsFor(
        target.inputsByKind["cc_libs"], "build",
        rule.propertyFilter("cc_include_path")));
    includePaths.append(rule.getAllOutputsFor(
        target.inputsByKind["proto_libs"], "build",
        rule.propertyFilter("cc_include_path")));
    var libs = rule.getAllOutputsFor(target.inputsByKind["proto_libs"],
        "build", rule.fileFilter("cc_archive"));
    var protoCompile =
        new CcProtoCompile(target, srcs[0], includePaths, libs);
    target.rule.engine.actions.addNode(protoCompile);
    target.outs.push(protoCompile.outputArchive);
    var includePath = new entity.Property(target.id, "cc_include_path",
        protoCompile.outputDir.getPath());
    target.rule.engine.actions.addNode(includePath);
    includePath.addParent(protoCompile.outputArchive);
    target.outs.push(includePath);
  }
  if (target.properties.go_api && target.properties.go_api.value) {
    var includePaths = [];
    includePaths.append(rule.getAllOutputsFor(
        target.inputsByKind["go_pkgs"], "build",
        rule.propertyFilter("go_include_path")));
    includePaths.append(rule.getAllOutputsFor(
        target.inputsByKind["proto_libs"], "build",
        rule.propertyFilter("go_include_path")));
    var pkgs = [];
    pkgs.append(rule.getAllOutputsFor(
        target.inputsByKind["go_pkgs"], "build",
        rule.fileFilter("go_archive")));
    pkgs.append(rule.getAllOutputsFor(target.inputsByKind["proto_libs"],
        "build", rule.fileFilter("go_archive")));

    var protoCompile =
        new GoProtoCompile(target, srcs[0], includePaths, pkgs);
    target.rule.engine.actions.addNode(protoCompile);
    target.outs.push(protoCompile.outputArchive);

    var includePath = new entity.Property(target.id, "go_include_path",
        protoCompile.outputDir.getPath());
    target.rule.engine.actions.addNode(includePath);
    includePath.addParent(protoCompile.outputArchive);
    target.outs.push(includePath);
  }
  return target.outs;
};

exports.register = function(engine) {
  engine.addRule("proto_library", new ProtoLibrary(engine));
};
