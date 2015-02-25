var fs = require("fs");
var path = require("path");
var util = require("util");
var rule = require("./rule.js");
var entity = require("./entity.js");
var shared = require("./shared.js");

function DockerBuild(owner, srcs, data, deps, name) {
  this.init("DockerBuild for " + owner.id);
  this.mnemonic = "DockerBuild";
  this.owner = owner;
  this.addAllInputs(srcs);
  this.srcs = srcs;
  this.addAllInputs(data);
  this.data = data;
  this.addAllInputs(deps);
  this.deps = deps;
  this.name = name;
  this.addParent(this.name);
  this.outputDir = new entity.Directory(owner.getRoot("gen"));
  owner.rule.engine.actions.addNode(this.outputDir);
  this.outputDir.addParent(this);

  this.done = new entity.File(owner.getRoot("gen") + "/done",
      "done_marker");
  owner.rule.engine.actions.addNode(this.done);
  this.done.addParent(this);

  this.version = "d251b622-2657-4ae9-8210-171b39fcdb8e";
}
DockerBuild.prototype = new entity.Action();
DockerBuild.prototype.createScript = function() {
  var destDir = this.outputDir.getPath();
  var lines = [
      "mkdir -p " + destDir,
      "cp " + this.srcs[0].getPath() + " " + destDir
  ];
  for (var i=0; i < this.data.length; i++) {
      lines.push("cp -Lr --preserve=all " + this.data[i].getPath() + " " + destDir);
  }
  lines.push("pushd " + destDir +"/..");

  var name = this.name.value;
  lines.push("docker build -t " + name + " " + path.basename(destDir));
  var tag = this.owner.getProperty("docker_tag");
  if (tag && tag.value) {
    lines.push("docker tag " + name + ":latest " + name + ":" + tag.value);
  }
  lines.push("popd");
  lines.push("date +%s > " + this.done.getPath());
  return lines;
};

function DockerPush(owner, name, remoteName) {
  this.init("DockerPush for " + owner.id);
  this.mnemonic = "DockerPush";
  this.owner = owner;

  this.name = name;
  this.remoteName = remoteName;
  this.tags = [];
  this.addParent(this.name);

  var tag = owner.getProperty("docker_tag");
  if (tag && tag.value) {
    this.addParent(tag);
    this.tags.push(tag.value);
  }

  var releaseVersion = owner.getProperty("release_version");
  if (releaseVersion && releaseVersion.value) {
    this.addParent(releaseVersion);
    var semVer = shared.parseVersion(releaseVersion.value);
    this.tags.push(semVer.major);
    this.tags.push(semVer.major + "." + semVer.minor);
    this.tags.push(semVer.major + "." + semVer.minor + "." + semVer.patch);
  }

  this.logFile = new entity.File(owner.getRoot("gen") + ".push.log",
      "test_log");
  owner.rule.engine.actions.addNode(this.logFile);
  this.logFile.addParent(this);

  this.version = "169eaceb-bf0e-44f7-86b9-b8a865216aa9";
}

DockerPush.prototype = new entity.Action();
DockerPush.prototype.createScript = function() {
  if (this.tags.length == 0) {
    return dockerTagPush(this);
  }

  var lines = [];
  for (var i = 0; i < this.tags.length; i++) {
    lines = lines.concat(dockerTagPush(this, this.tags[i]));
  }
  return lines;
};

function dockerTagPush(target, tag) {

  var remote = target.remoteName;
  if (tag) {
    remote += ":" + tag;
  }
  var pushCommand = "docker push "
  var convoyServer = target.owner.getProperty("convoy_server");
  if (convoyServer) {
    pushCommand = "gcloud preview docker push ";
  }
  return ["docker tag " + target.name.value +":latest " + remote,
          pushCommand + remote];
}

function DockerSave(owner, name) {
  this.init("DockerSave for " + owner.id);
  this.mnemonic = "DockerSave";
  this.owner = owner;

  this.name = name;
  this.addParent(this.name);

  this.tarBall = new entity.File(owner.getRoot("bin") + ".tar.gz",
      "docker_tarball");
  owner.rule.engine.actions.addNode(this.tarBall);
  this.tarBall.addParent(this);

  this.version = "24e972a7-c1f6-444a-b42a-cca8feb2a043";
}
DockerSave.prototype = new entity.Action();
DockerSave.prototype.createScript = function() {
  return [
      "mkdir -p " + path.dirname(this.tarBall.getPath()),
      "docker save " + this.name.value + " | gzip -c > " +
          this.tarBall.getPath()
  ];
};

function DockerPull(owner, name, remoteName) {
  this.init("DockerPull for " + owner.id);
  this.mnemonic = "DockerPull";
  this.owner = owner;

  this.name = name;
  this.remoteName = remoteName;
  this.tags = [];
  this.addParent(this.name);

  var tag = owner.getProperty("docker_tag");
  if (tag && tag.value) {
    this.addParent(tag);
    this.tags.push(tag.value);
  }

  var releaseVersion = owner.getProperty("release_version");
  if (releaseVersion && releaseVersion.value) {
    this.addParent(releaseVersion);
    var semVer = shared.parseVersion(releaseVersion.value);
    this.tags.push(semVer.major);
    this.tags.push(semVer.major + "." + semVer.minor);
    this.tags.push(semVer.major + "." + semVer.minor + "." + semVer.patch);
  }

  this.logFile = new entity.File(owner.getRoot("gen") + ".pull.log",
      "test_log");
  owner.rule.engine.actions.addNode(this.logFile);
  this.logFile.addParent(this);

  this.version = "16583dac-8bfe-490b-8caa-fae1808a4efe";
}

DockerPull.prototype = new entity.Action();
DockerPull.prototype.createScript = function() {
  if (this.tags.length == 0) {
    return dockerTagPull(this);
  }

  var lines = [];
  for (var i = 0; i < this.tags.length; i++) {
    lines = lines.concat(dockerTagPull(this, this.tags[i]));
  }
  return lines;
};

function dockerTagPull(target, tag) {

  var remote = target.remoteName;
  if (tag) {
    remote += ":" + tag;
  }
  var local = target.name.value;
  if (tag) {
    local += ":" + tag
  }
  return [ "docker pull " + remote,
           "docker tag " + remote + " " + local];
}

DockerDeploy = function(engine) {
  this.engine = engine;
};

DockerDeploy.prototype = new rule.Rule();

DockerDeploy.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var name = target.getProperty("docker_name");
  if (!name) {
    name = new entity.Property(target.id, "docker_name", target.name);
  }

  if (kind != "pull" && kind != "package" && kind != "deploy") {
    target.outs = [];
    return target.outs;
  }

  if (kind == "pull") {
    target.outs = [];
    var convoyBucket = target.getProperty("convoy_bucket");
    var dockerRepository = target.getProperty("docker_repository");
    if (convoyBucket) {
      var tokens = name.value.split("/");
      var remoteName = convoyServer.value +
          "/" + convoyBucket.value + "/" + tokens[tokens.length-1];
      var dockerPull = new DockerPull(target, name, remoteName);
      target.outs.push(dockerPull.logFile);
    } else if (dockerRepository) {
      var dockerPull = new DockerPull(target, name,
          dockerRepository.value + "/" + name.value);
      target.outs.push(dockerPull.logFile);
    } else {
      console.log("Need at least --convoy_bucket or " +
          "--docker_repository to be able to pull docker images.");
      process.exit(1);
    }
    return target.outs;
  }

  var srcs = rule.getAllOutputsFor(target.inputsByKind["srcs"], "build",
      rule.fileFilter("src_file", "/Dockerfile"));
  var data = rule.getAllOutputsFor(target.inputsByKind["data"], "build",
      rule.allFilesAndDirsFilter);
  var deps = rule.getAllOutputsFor(target.inputsByKind["deps"], kind,
      rule.allFilesFilter);

  var dockerBuild = new DockerBuild(target, srcs, data, deps, name);
  target.outs = [ dockerBuild.done ];

  if (kind == "deploy") {
    var convoyBucket = target.getProperty("convoy_bucket");
    var convoyServer = target.getProperty("convoy_server");
    var dockerRepository = target.getProperty("docker_repository");
    if (convoyBucket) {
      var tokens = name.value.split("/");
      var remoteName = convoyServer.value + "/" +
          convoyBucket.value + "/" + tokens[tokens.length-1];
      var dockerPush = new DockerPush(target, name, remoteName);
      dockerPush.addParent(dockerBuild.done);
      target.outs.push(dockerPush.logFile);
    } else if (dockerRepository) {
      var dockerPush = new DockerPush(target, name,
          dockerRepository.value + "/" + name.value);
      dockerPush.addParent(dockerBuild.done);
      target.outs.push(dockerPush.logFile);
    } else {
      var dockerSave = new DockerSave(target, name);
      dockerSave.addParent(dockerBuild.done);
      target.outs.push(dockerSave.tarBall);
    }
  }
  return target.outs;
};

exports.register = function(engine) {
  engine.addRule("docker_deploy", new DockerDeploy(engine));
};
