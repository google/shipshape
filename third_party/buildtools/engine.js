var fs = require("fs");
var path = require("path");
var util = require("util");
var graphs = require("./graphs");
var rule = require("./rule.js");
var entity = require("./entity.js");
var shared = require("./shared.js");
var query = require("./query.js");
var child_process = require('child_process');

var buildFileBaseName = "CAMPFIRE";

exports.Engine = function(settings, campfireRoot, relative) {
  this.rules = {};
  this.files = {};
  this.targets = new graphs.Graph();
  this.actions = new graphs.Graph();
  this.addRule("static_file", new rule.StaticFile(this));
  this.settings = settings;
  this.campfireRoot = campfireRoot;
  this.relative = relative;
  this.loadRules();
};

exports.Engine.prototype.loadRules = function() {
  if (this.settings.rules) {
    for (var i = 0; i < this.settings.rules.length; i++) {
      var ruleSet = require(this.settings.rules[i]);
      ruleSet.register(this);
    }
  }
};

exports.Engine.prototype.addRule = function(name, rule) {
  this.rules[name] = rule;
  rule.config_name = name;
};

exports.Engine.prototype.loadFile = function(file) {
  var existing = this.files[file];
  if (existing) {
    return existing;
  }
  var data = fs.readFileSync(file);
  var parsed = JSON.parse(data);
  var entries = {};
  var file_dict = { entries: entries,
                    packageName: path.dirname(file)
                  };
  for (var i = 0; i < parsed.length; i++) {
    var unresolved = parsed[i];
    entries[unresolved.name] = {
      unresolved : unresolved
    };
  }
  this.files[file] = file_dict;
  return file_dict;
};

exports.Engine.prototype.resolvePath = function(p) {
  if (p.startsWith("//")) {
    p = p.substring(2);
  } else if (this.relative) {
    p = path.join(this.relative, p);
  }
  return p === "" ? "." : p;
};

exports.Engine.prototype.resolveTargets = function(pattern) {
  if (pattern.endsWith(":all")) {
    return this.loadAllTargets(
        this.resolvePath(pattern.substring(0, pattern.length - 4) +
             "/" + buildFileBaseName));
  } else if (pattern.endsWith("/...") || pattern === "...") {
    var root = this.resolvePath(
        pattern.substring(0, pattern.length - 3));
    var buildFiles = shared.findFiles(root, /.*\/CAMPFIRE$/);
    var targets = [];
    for (var i = 0; i < buildFiles.length; i++) {
      targets.append(this.loadAllTargets(buildFiles[i]));
    }
    return targets;
  } else {
    return [this.resolveTarget(pattern)];
  }
};

exports.Engine.prototype.loadAllTargets = function(file) {
  var pkg = this.loadFile(file);
  var targets = [];
  for (var name in pkg.entries) {
    var entry = pkg.entries[name];
    if (entry.resolved) {
      targets.push(entry.resolved);
    } else {
      targets.push(this.loadTarget(pkg, entry.unresolved));
    }
  }
  return targets;
};

exports.Engine.prototype.resolveTarget = function(target, file, context) {
  if (!file && this.relative) {
    if (target.charAt(0) != '/' && target.charAt(0) != ':') {
      if (this.relative) {
        target = "//" + this.relative + "/" + target;
      } else {
        target = "//" + target;
      }
    } else if (target.charAt(0) == ':') {
      target = "//" + this.relative + target;
    }
  }
  if (target.charAt(0) == '/' && target.indexOf(":") == -1) {
    var sub = target.substring(target.lastIndexOf("/") + 1);
    target = target + ":" + sub;
  }
  while (true) {
    var varOpen = target.indexOf("$(");
    if (varOpen < 0) {
      break;
    }
    var varClose = target.indexOf(")");
    if (varClose < 0) {
      break;
    }
    var pre = target.substring(0, varOpen);
    var post = target.substring(varClose + 1);
    var varName = target.substring(varOpen + 2, varClose);
    target = pre + this.settings.properties[varName] + post;
  }
  var entry;
  var name;
  if (target.indexOf(":") === 0) {
     name = target.substring(1);
     entry = file.entries[name];
  } else if (target.indexOf("//") === 0) {
    var loaded = this.targets.getNode(target);
    if (loaded) {
      return loaded;
    }
    var prefixStrippedTarget = target.substring(2);
    var targetNameIndex = prefixStrippedTarget.indexOf(":");
    if (targetNameIndex < 0) {
      contextLog(context, "ERROR: invalid target: " + target);
      process.exit(1);
    }
    var packageDirectory =
        prefixStrippedTarget.substring(0, targetNameIndex);
    var buildFile = path.join(packageDirectory, buildFileBaseName);
    if (!fs.existsSync(buildFile)) {
      var absPath = path.join(this.campfireRoot, packageDirectory);
      contextLog(context, 'ERROR: ' + buildFileBaseName + " file not found in '" + absPath + "'");
      if (!fs.existsSync(packageDirectory)) {
        console.log("  The '" + packageDirectory + "' package directory does not exist!");
      }
      process.exit(1);
    }
    var file = this.loadFile(buildFile);
    name = prefixStrippedTarget.substring(targetNameIndex + 1);
    entry = file.entries[name];
  } else {
    var filePath = target;
    if (filePath.charAt(0) != '/' && file) {
      filePath = path.join(file.packageName, target);
    }
    var loaded = this.targets.getNode(filePath);
    if (loaded) {
      return loaded;
    }
    if (fs.existsSync(filePath)) {
      return this.rules["static_file"].createTarget(name, filePath);
    } else {
      contextLog(context, "ERROR: missing file: " + target);
      process.exit(1);
    }
  }
  if (entry === undefined) {
    contextLog(context, "ERROR: missing target: " + target);
    process.exit(1);
  }
  if (entry.resolved) {
    return entry.resolved;
  }
  return this.loadTarget(file, entry.unresolved);
};

function contextLog(context, msg) {
  console.log(msg);
  if (context) {
    console.log("  context: " + context);
  }
}

exports.Engine.prototype.loadTarget = function(file, config) {
  var rule = this.rules[config.kind];
  if (rule === undefined) {
    console.log("Missing rule kind: " + config.kind);
    process.exit(1);
  }
  var targetId = "//" + file.packageName + ":" + config.name;
  var loaded = this.targets.getNode(targetId);
  if (loaded) {
    return loaded;
  }
  var root = getRoot(targetId);
  var allowedRoots = this.settings["allowed_dependencies"][root];
  var resolvedInputsByKind = {};
  for (var inputKind in config.inputs) {
    var inputs = config.inputs[inputKind];
    var resolvedInputs = resolvedInputsByKind[inputKind] = [];
    for (var i = 0; i < inputs.length; i++) {
      var resolvedInput = this.resolveTarget(inputs[i], file, targetId);
      var resolvedId = resolvedInput.id;
      if (allowedRoots && resolvedId.startsWith("//")) {
        var inputRoot = getRoot(resolvedId);
        if (inputRoot != root && !allowedRoots[inputRoot]) {
          console.log("ERROR: //" + root +
              " is not allowed to depend on //" + inputRoot +
              " as per .campfire_settings");
          process.exit(1);
        }
      }
      resolvedInputs.push(resolvedInput);
    }
  }
  var resolvedProperties = {};
  var configuration = this.settings.properties.configuration;
  var properties = [];
  for (var p in config.properties) {
    if (config.properties.hasOwnProperty(p)) {
      properties.push(p);
    }
  }
  properties = properties.sort();
  for (var j = 0; j < properties.length; ++j) {
    // Support configuration-regex:property keys.
    var property = properties[j];
    var lastColon = property.lastIndexOf(':');
    var resolvedProperty = undefined;
    if (lastColon < 0) {
      // This has no regex prefix.
      resolvedProperty = new entity.Property(
         targetId, property, config.properties[property]);
    } else {
      var regex = new RegExp(property.substr(0, lastColon));
      if (regex.test(configuration)) {
        var justProperty = property.substr(lastColon + 1);
        resolvedProperty = new entity.Property(
           targetId, justProperty, config.properties[property]);
        // We've no further use for the regex.
        property = justProperty;
      }
    }
    if (resolvedProperty) {
      if (property in resolvedProperties) {
        // Always merge by concat. Since we sorted the keys up above,
        // we'll merge deterministically on every run.
        resolvedProperties[property].value =
            resolvedProperties[property].value.concat(
                resolvedProperty.value);
      } else {
        resolvedProperties[property] = resolvedProperty;
      }
    }
  }
  var json = file.entries[config.name].unresolved;
  return rule.createTarget(config.name, targetId, resolvedInputsByKind,
                           resolvedProperties, json);
};

// getRoot assumes that the input starts with '//some_root/'
function getRoot(id) {
  var path = id.substring(2);
  var index = path.indexOf("/");
  return path.substring(0, index);
}

function checkCycles(graph, name) {
  /* TODO(jvg): Fix stronglyConnectedComponents.
  var sccs = graph.stronglyConnectedComponents();
  if (sccs.length > 0) {
    console.log(name + " graph has " + sccs.length + " cycle(s):")
    for (var i = 0; i < sccs.length; i++) {
      var scc = sccs[i];
        console.log(i +":");
      for (var j = 0; j < scc.length; j++) {
        console.log("\t" + scc[j].id);
        console.log("\tinputs");
        for (var k = 0; k < scc[j].inputs.length; k++) {
          console.log("\t\t" + scc[j].inputs[k].id);
        }
        console.log("\toutputs");
        for (var k = 0; k < scc[j].outputs.length; k++) {
          console.log("\t\t" + scc[j].outputs[k].id);
        }
      }
    }
    return false;
  }
  */
  return true;
}

exports.Engine.prototype.validateTargets = function () {
  return checkCycles(this.targets, "Targets");
};

exports.Engine.prototype.validateActions = function () {
  return checkCycles(this.actions, "Actions");
};

exports.Engine.prototype.query = function(q) {
  global.query_engine = this;
  var evalResults = query.queryEval(q);
  if (!evalResults) {
    console.log("Invalid query: " + q);
    process.exit(1);
  }
  evalResults = evalResults.sort(
      function(a,b) { return a.id > b.id; } );
  for (var i = 0; i < evalResults.length; ++i) {
    evalResults[i] = resolvedEntry(evalResults[i]);
  }
  if (this.settings.properties.print_names) {
    for (var i = 0; i < evalResults.length; i++) {
      var result = evalResults[i];
      console.log(result.name ? result.name : result);
    }
  } else {
    console.log(JSON.stringify(evalResults, undefined, 2));
  }
};

function resolvedEntry(entry) {
  if (!entry.json) {
    return entry.id ? entry.id : entry;
  }
  var json = {};
  for (key in entry.json) {
    if (key === "name") {
      json.name = entry.id;
    } else if (key === "inputs") {
      json.inputs = {};
      for (kind in entry.json.inputs) {
        json.inputs[kind] = entry.inputsByKind[kind].map(function(i) { return i.id; });
      }
    } else {
      json[key] = entry.json[key];
    }
  }
  return json;
}

exports.Engine.prototype.run = function (kind, targets, threads,
    callback) {
  if (!this.validateTargets()) {
    console.log("Target graph validation failed");
    process.exit(1);
  }

  var files = rule.getAllOutputsFor(targets, kind);
  if (!this.validateActions()) {
    console.log("Target graph validation failed");
    process.exit(1);
  }

  var state;
  if (fs.existsSync(".campfire_state")) {
    state = JSON.parse(fs.readFileSync(".campfire_state"));
  } else {
    state = {};
  }
  var queue = [];
  var needed = {};
  findRootsAndNeededNodes(files, queue, needed);
  console.log("Analysis complete");
  if (kind == "print_action") {
    this.printAllActions(queue);
  } else {
    var runner =
        new Runner(threads, this, queue, needed, state, callback);
    runner.processQueue();
  }
};

exports.Engine.prototype.printAllActions = function(queue) {
  var uniqueCommands = {};

  var activeMnemonics = undefined;
  var mnemonics = this.settings.properties["mnemonics"];
  if (mnemonics) {
    var split = mnemonics.split(",");
    activeMnemonics = {};
    for (var i = 0; i < split.length; i++) {
      activeMnemonics[split[i]] = true;
    }
  }

  while (queue.length > 0) {
    var entry = queue.shift();
    var cmd = entry.createFullCommand();
    if (!activeMnemonics || activeMnemonics[cmd.mnemonic]) {
      if (cmd.id) {
        // Remove duplicates
        uniqueCommands[cmd.id] = cmd;
      }
    }
    for (var i = 0; i < entry.outputs.length; i++) {
      var output = entry.outputs[i];
      queue.push(output);
    }
  }
  // Extract the unique objects from the map.
  var commands = [];
  for (var id in uniqueCommands) {
    var uc = uniqueCommands[id];
    uc.cwd = this.campfireRoot;
    commands.push(uc);
  }
  // Save the action commands to a JSON file.
  var actionsDir = "campfire-out/.print_action/";
  mkdirs(actionsDir);
  var serializedCommands = JSON.stringify(commands);
  var actionPath = "campfire-out/.print_action/actions.json";
  fs.writeFileSync(actionPath, serializedCommands);
  console.log("Actions written to: " + actionPath);
};

function findRootsAndNeededNodes(targets, roots, visited) {
  for (var i = 0; i <  targets.length; i++) {
    var target = targets[i];
    if (visited[target.id] === true) {
      continue;
    } else {
      visited[target.id] = true;
    }

    if (target.inputs.length > 0) {
      findRootsAndNeededNodes(target.inputs, roots, visited);
    } else {
      roots.push(target);
    }
  }
}

function Runner(threads, engine, queue, needed, state, callback) {
  this.threads = threads;
  this.engine = engine;
  this.queue = queue;
  this.queued = {};
  this.notready = [];
  this.needed = needed;
  this.state = state;
  this.processed = {};
  this.running = 0;
  this.torun = 0;
  this.processes = [];
  this.loggedAnsi = false;
  for (var need in needed) {
    this.torun++;
  }
  this.progress = 1;
  this.totalqueued = this.torun;
  this.processing = false;
  this.callback = callback;
}

Runner.prototype.schedule = function(nodes) {
 var allNodes = [];
 for (var i = 0; i <  this.notready.length; i++) {
   allNodes.push(this.notready[i]);
 }
 this.notready = [];
 if (nodes) {
   for (var j = 0; j <  nodes.length; j++) {
     allNodes.push(nodes[j]);
   }
 }
 this.trySchedule(allNodes);
};

Runner.prototype.trySchedule = function(nodes) {
  for (var i=0; i < nodes.length; i++) {
    var node = nodes[i];
    var skip = false;
    if (this.needed[node.id] === true) {
      for (var j =0; j < node.inputs.length; j++) {
        if (this.processed[node.inputs[j].id] !== true) {
          skip = true;
          break;
        }
      }
    }
    if (skip) {
      this.notready.push(node);
    } else {
      if (!this.queued[node.id]) {
        this.queue.push(node);
        this.queued[node.id] = true;
      }
    }
  }
};

Runner.prototype.getState = function(entry) {
  return this.state[entry.id];
};

Runner.prototype.setState = function(entry, state) {
  this.state[entry.id] = state;
};

Runner.prototype.close = function(exitCode) {
  var serialized = JSON.stringify(this.state);
  fs.writeFileSync(".campfire_state", serialized);
  console.log("Campfire completed with " + (
      exitCode === 0 ? "success." : "failure."));
  if (this.callback) {
    this.callback(exitCode);
  }
  process.exit(exitCode);
};

mkdirs = function(dir) {
  if (path === "" || dir == "/" || dir == "." || dir == "..") {
    return;
  }
  if (!fs.existsSync(dir)) {
    mkdirs(path.dirname(dir));
    fs.mkdirSync(dir);
  }
};

Runner.prototype.logForThread = function(thread, message) {
  var ansiSetting = this.engine.settings.properties["ansi"];
  if (!ansiSetting || ansiSetting == "false") {
    if (message === "") { return; }
    console.log(message);
  } else if (ansiSetting === "progress") {
    if (message === "") { return; }
    var progress = "[" + this.progress + "/" + this.totalqueued + "]";
    process.stdout.write("\033[1A\033[34m");
    process.stdout.write(progress + "\033[39;49m " + message);
    process.stdout.write("\033[K\n");
  } else {
    if (!this.loggedAnsi) {
      this.loggedAnsi = true;
      for (var i = 0; i < this.threads; i++) {
        var padded = (i + 100 + "").slice(1);
        console.log("\033[34mThread " + padded + ":\033[39;49m");
      }
    }
    var back = this.threads - thread + 1;
    var id = (thread + 100 + "").slice(1);
    process.stdout.write("\033[" + back + "A\033[34mThread " +
        id + ":\033[39;49m " + message + "\033[0K");
    process.stdout.write("\033[" + back + "B\033[0G");
  }
};

Runner.prototype.processQueue = function() {
  this.processing = true;
  var parent = this;
  if (this.queue.length === 0 && this.running === 0 &&
      this.torun === 0) {
    this.close(0);
  }
  while(this.queue.length > 0) {
    var entry = this.queue.shift();
    delete this.queued[entry.id];
    if (this.needed[entry.id] !== true) {
      continue;
    }
    this.progress++;
    var lines = entry.createScript();
    if (this.engine.settings.properties.debug) {
      lines.unshift("set -x");
    }
    if (!entry.isUpToDate(this.getState(entry)) && lines.length > 0) {
      if (lines === undefined) {
        console.log("ERROR: " + entry.id + " was not meant to be" +
            " scheduled, yet it was not up-to-date.");
        process.exit(1);
      }
      var scriptDir = "campfire-out/.scripts/" +
          entry.getUpToDateMarker();
      mkdirs(scriptDir);
      var scriptPath =  scriptDir + "/run.sh";
      lines.unshift("#!/bin/sh -e", "echo $0 for " + entry.id);
      fs.writeFileSync(scriptPath, lines.join("\n"), { mode: 0755 });
      this.needed[entry.id] = undefined;
      var thread = this.running++;
      this.logForThread(thread, "Running " + entry.id);
      var logsRoot = "campfire-out/.logs/" +
          entry.getUpToDateMarker() + "/";
      mkdirs(logsRoot);
      var stdoutPath = logsRoot +"STDOUT";
      if (entry.logFile) {
        mkdirs(path.dirname(entry.logFile.getPath()));
        stdoutPath = entry.logFile.getPath();
      }
      var stdout = fs.createWriteStream(stdoutPath);
      var stderrPath = logsRoot +"STDERR";
      var stderr = fs.createWriteStream(stderrPath);
      // Asynchronous closure, ensure all variables are scoped correctly
      // by wrapping in function.
      (function(runner, data) {
        var cmd = child_process.spawn(data.scriptPath).on(
            'close', function (code) {
          data.parent.running--;
          data.stdout.end();
          data.stderr.end();
          if (code !== 0) {
            console.log("");
            console.log("ERROR: " + data.entry.id +
                " failed with exit code " + code + ":");
            console.log(fs.readFileSync(data.stderrPath).toString());
            console.log("");
            console.log("See " + data.stdoutPath +
                " for more details.");
            data.parent.close(code);
          } else {
            data.parent.logForThread(data.thread, "");
          }
          data.parent.setState(data.entry,
              data.entry.getState(data.entry));
          data.parent.processed[data.entry.id] = true;

          data.parent.schedule(data.entry.outputs);
          data.parent.torun--;
          if (!data.parent.processing) {
            data.parent.processQueue();
          }
        });
        cmd.stdout.pipe(stdout);
        cmd.stderr.pipe(stderr);
     })(this, { entry: entry,
                stdout: stdout,
                stderr: stderr,
                stderrPath: stderrPath,
                stdoutPath: stdoutPath,
                scriptPath: scriptPath,
                parent: parent,
                thread: thread });
    } else {
      if (!this.processed[entry.id]) {
        this.torun--;
      }
      this.processed[entry.id] = true;
      this.schedule(entry.outputs);
    }
    if (this.threads - this.running <= 0 || this.queue.length === 0) {
      break;
    }
  }
  this.processing = false;
  if (this.torun === 0) {
    this.close(0);
  }
};
