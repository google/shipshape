'use strict';

var child_process = require('child_process');
var fs = require('fs');
var path = require('path');

var entity = require('./entity.js');
var graphs = require('./graphs');
var query = require('./query.js');
var rule = require('./rule.js');
var shared = require('./shared.js');

// Basename for a build specification file.
var BUILD_FILE_BASENAME = 'CAMPFIRE';

/**
 * Core runtime engine for campfire, handling analysis of CAMPFIRE files,
 * resolution of targets, and the generation of ninja build rules.
 */
exports.Engine = function(settings, campfireRoot, relative) {
  this.rules = {};
  this.files = {};
  this.targets = new graphs.Graph();
  this.entities = {};
  this.addRule('static_file', new rule.StaticFile(this));
  this.settings = settings;
  this.campfireRoot = campfireRoot;
  this.relative = relative;
  this.loadRules();
};

/**
 * Loads each of the rule files specified in the campfire_settings.
 */
exports.Engine.prototype.loadRules = function() {
  if (this.settings.rules) {
    for (var i = 0; i < this.settings.rules.length; i++) {
      var ruleSet = require(this.settings.rules[i]);
      ruleSet.register(this);
    }
  }
};

/**
 * Adds a given rule to the campfire runtime.
 */
exports.Engine.prototype.addRule = function(name, rule) {
  this.rules[name] = rule;
  rule.config_name = name;
};

/**
 * Reads a given build specification file and returns a map of its contained
 * targets (target name -> specification).  The reading of a file may be cached.
 */
exports.Engine.prototype.loadFile = function(file) {
  var existing = this.files[file];
  if (existing) {
    return existing;
  }
  var data = fs.readFileSync(file);
  var parsed = JSON.parse(data);
  var entries = {};
  var file_dict = {
    entries: entries,
    packageName: path.dirname(file)
  };
  for (var i = 0; i < parsed.length; i++) {
    var unresolved = parsed[i];
    entries[unresolved.name] = {
      unresolved: unresolved
    };
  }
  this.files[file] = file_dict;
  return file_dict;
};

/**
 * Returns the path {@code p} as a package path relative to the build root
 * (e.g. //kythe/go/storage -> kythe/go/storage).
 */
exports.Engine.prototype.resolvePath = function(p) {
  if (p.startsWith('//')) {
    p = p.substring(2);
  } else if (this.relative) {
    p = path.join(this.relative, p);
  }
  return p === '' ? '.' : p;
};

/**
 * Returns the list of targets specified by {@code pattern}. Normal targets such
 * as //kythe/go/storage or go/storage will resolve to a singleton array.  The
 * special ':all' suffix (e.g. //kythe/go/storage:all) will return all targets
 * in a given package.  The special '/...' suffix (e.g. //kythe/go/...) will
 * return all targets contained within a package as well as any subpackages.
 */
exports.Engine.prototype.resolveTargets = function(pattern) {
  if (pattern.endsWith(':all')) {
    return this.loadAllTargets(
        this.resolvePath(pattern.substring(0, pattern.length - 4) +
            '/' + BUILD_FILE_BASENAME));
  } else if (pattern.endsWith('/...') || pattern === '...') {
    var root = this.resolvePath(
        pattern.substring(0, pattern.length - 3));
    var buildFiles = this.findBuildFiles(root);
    var targets = [];
    for (var i = 0; i < buildFiles.length; i++) {
      targets.append(this.loadAllTargets(buildFiles[i]));
    }
    return targets;
  } else {
    return [this.resolveTarget(pattern)];
  }
};

/**
 * Reads a given build specification file and returns an array of its contained
 * targets definitions, each with fully resolved inputs and dependencies.
 */
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

/**
 * Given a target string specification, with a possible {@code file} environment
 * (see {@code loadFile}), returns its fully resolved target definition.
 * {@code file} is needed to resolve target dependencies and files relative to a
 * package.  {@code context} is used for error reporting when resolving a target
 * (see {@code contextError}).
 */
exports.Engine.prototype.resolveTarget = function(target, file, context) {
  if (!file && this.relative) {
    if (target.charAt(0) != '/' && target.charAt(0) != ':') {
      if (this.relative) {
        target = '//' + this.relative + '/' + target;
      } else {
        target = '//' + target;
      }
    } else if (target.charAt(0) == ':') {
      target = '//' + this.relative + target;
    }
  }
  if (target.charAt(0) == '/' && target.indexOf(':') == -1) {
    var sub = target.substring(target.lastIndexOf('/') + 1);
    target = target + ':' + sub;
  }
  while (true) {
    var varOpen = target.indexOf('$(');
    if (varOpen < 0) {
      break;
    }
    var varClose = target.indexOf(')');
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
  if (target.indexOf(':') === 0) {
    name = target.substring(1);
    entry = file.entries[name];
  } else if (target.indexOf('//') === 0) {
    var loaded = this.targets.getNode(target);
    if (loaded) {
      return loaded;
    }
    var prefixStrippedTarget = target.substring(2);
    var targetNameIndex = prefixStrippedTarget.indexOf(':');
    if (targetNameIndex < 0) {
      contextError(context, 'ERROR: invalid target: ' + target);
      process.exit(1);
    }
    var packageDirectory =
        prefixStrippedTarget.substring(0, targetNameIndex);
    var buildFile = path.join(packageDirectory, BUILD_FILE_BASENAME);
    if (!fs.existsSync(buildFile)) {
      var absPath = path.join(this.campfireRoot, packageDirectory);
      contextError(context, 'ERROR: ' +
          BUILD_FILE_BASENAME + " file not found in '" + absPath + "'");
      if (!fs.existsSync(packageDirectory)) {
        console.error("  The '" + packageDirectory + "' package directory does not exist!");
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
      return this.rules['static_file'].createTarget(name, filePath);
    } else {
      contextError(context, 'ERROR: missing file: ' + target);
      process.exit(1);
    }
  }
  if (entry === undefined) {
    contextError(context, 'ERROR: missing target: ' + target);
    process.exit(1);
  }
  if (entry.resolved) {
    return entry.resolved;
  }
  return this.loadTarget(file, entry.unresolved);
};

/**
 * Logs {@code msg} to the console, suffixed by a context line if a
 * {@code context} is given.
 */
function contextError(context, msg) {
  console.error(msg);
  if (context) {
    console.error('  context: ' + context);
  }
}

/**
 * Given a {@code file} environment (see {@code loadFile}) and a particular
 * unresolved target configuration within in, returns a fully-resolved target
 * configuration.
 */
exports.Engine.prototype.loadTarget = function(file, config) {
  var rule = this.rules[config.kind];
  var targetId = '//' + file.packageName + ':' + config.name;
  if (rule === undefined) {
    contextError(targetId, 'Missing rule kind: ' + config.kind);
    process.exit(1);
  }
  var loaded = this.targets.getNode(targetId);
  if (loaded) {
    return loaded;
  }
  var root = getRoot(targetId);
  var allowedRoots = this.settings['allowed_dependencies'][root];
  var resolvedInputsByKind = {};
  for (var inputKind in config.inputs) {
    var inputs = config.inputs[inputKind];
    var resolvedInputs = resolvedInputsByKind[inputKind] = [];
    for (var i = 0; i < inputs.length; i++) {
      var resolvedInput = this.resolveTarget(inputs[i], file, targetId);
      var resolvedId = resolvedInput.id;
      if (allowedRoots && resolvedId.startsWith('//')) {
        var inputRoot = getRoot(resolvedId);
        if (inputRoot != root && !allowedRoots[inputRoot]) {
          console.error('ERROR: //' + root +
              ' is not allowed to depend on //' + inputRoot +
              ' as per .campfire_settings');
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
  var index = path.indexOf('/');
  return path.substring(0, index);
}

/**
 * Entry-function for the query engine.  Runs the JS code {@code q} within the
 * query environment and logs the results to the console.
 */
exports.Engine.prototype.query = function(q) {
  global.query_engine = this;
  var evalResults = query.queryEval(q);
  if (!evalResults) {
    console.error('Invalid query: ' + q);
    process.exit(1);
  }
  evalResults = evalResults.sort(
      function(a, b) { return a.id > b.id; });
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
  for (var key in entry.json) {
    if (key === 'name') {
      json.name = entry.id;
    } else if (key === 'inputs') {
      json.inputs = {};
      for (var kind in entry.json.inputs) {
        json.inputs[kind] =
            entry.inputsByKind[kind].map(function(i) { return i.id; });
      }
    } else {
      json[key] = entry.json[key];
    }
  }
  return json;
}

/**
 * Entry-function for campfire commands that generate/execute ninja build rules.
 * The specified targets in {@code targetArgs} will be resolved, their required
 * build rules for the specified build {@code kind} (see {@code rule.kinds})
 * will be emitted to build.ninja, and possibly executed.
 */
exports.Engine.prototype.ninjaCommand = function(kind, targetArgs, execute,
                                                 callback) {
  var targets = [];
  for (var i = 0; i < targetArgs.length; i++) {
    targets.append(this.resolveTargets(targetArgs[i]));
  }
  var ids = targets
      .filter(function(t) { return t.rule.getNinjaBuilds; })
      .map(function(t) { return t.id; });
  if (targets.length == 0) {
    targets.append(this.resolveTargets('//...'));
  }
  this.convertToNinja(kind);
  if (execute) {
    var cmd = this.settings.properties['ninja_path'] || 'ninja';
    child_process.spawn(cmd, ids, {
      stdio: ['ignore', process.stderr, process.stderr]
    }).on('exit', function(code) {
      if (code !== 0) {
        process.exit(code);
      } else if (callback) {
        callback();
      }
    });
  }
};

/**
 * Gathers all build rules of the given {@code kind} (see {@code rule.kinds}) in
 * the engine's resolved targets and emits them to build.ninja.
 */
exports.Engine.prototype.convertToNinja = function(kind) {
  var ninjaPath = this.campfireRoot + '/build.ninja';
  fs.writeFileSync(ninjaPath, ninjaBuildHeader(this).join('\n') + '\n\n');
  var ninjaFile = fs.openSync(ninjaPath, 'a');
  for (var i = 0; i < this.targets.nodes.length; i++) {
    // NOTE: on each iteration of this loop, the number of targets may increase
    // as implicit rule dependencies (e.g. //buildtools:go_testmain_generator)
    // are resolved.

    var target = this.targets.nodes[i];
    if (target.rule.getBuilds) {
      writeBuilds(ninjaFile, target.rule.getBuilds(target, kind), target.id);
    }
  }
  fs.closeSync(ninjaFile);
};

function writeBuilds(ninjaFile, builds, phony) {
  if (phony) {
    builds = builds.concat([{
      rule: 'phony',
      inputs: builds
          .map(function(b) { return b.outs; })
          .reduce(function(p, n) { return p.concat(n); }, []),
      outs: [phony]
    }]);
  }
  for (var i = 0; i < builds.length; i++) {
    var str = ninjaBuild(builds[i]) + '\n';
    var buf = new Buffer(str);
    var leftToWrite = str.length;
    while (leftToWrite > 0) {
      leftToWrite -= fs.writeSync(ninjaFile, buf,
                                  str.length - leftToWrite, leftToWrite);
    }
  }
}

function mergeBuilds(builds, more) {
  for (var kind in more) {
    if (builds[kind]) {
      builds[kind].append(more[kind]);
    } else {
      builds[kind] = more[kind];
    }
  }
}

function ninjaBuild(b) {
  var outs = rule.getPaths(b.outs)
      .map(function(o) { return o.replace(':', '$:'); })
      .join(' ');
  var str = 'build ' + outs + ': ' + b.rule + ' ' + rule.getPaths(b.inputs).join(' ');
  if (b.implicits && b.implicits.length > 0) {
    str += ' | ' + rule.getPaths(b.implicits).join(' ');
  }
  if (b.ordered && b.ordered.length > 0) {
    str += ' || ' + rule.getPaths(b.ordered).join(' ');
  }
  str += '\n';
  for (var v in b.vars) {
    str += '  ' + v + ' = ' + b.vars[v] + '\n';
  }
  if (b.phony) {
    str += 'build ' + b.phony.replace(':', '$:') + ': phony ' + outs + '\n';
  }
  return str;
}

function ninjaBuildHeader(engine) {
  var vars = {
    'gotool': engine.settings.properties['go_path'],
    'jdkpath': engine.settings.properties['jdk_path'],
    'javac': '$jdkpath/bin/javac',
    'java': '$jdkpath/bin/java',
    'javajar': '$jdkpath/bin/jar',
    'protocpath': engine.settings.properties['protoc_path'],
    'asciidoc': engine.settings.properties['asciidoc_path'] || 'asciidoc',
    'cxxpath': engine.settings.properties['cxx_path'],
    'cpath': engine.settings.properties['cc_path']
  };

  var lines = [];
  for (var k in vars) {
    lines.push(k + ' = ' + vars[k]);
  }
  lines.push('subninja buildtools/rules.ninja');
  return lines;
}

var EXCLUDED_DIRECTORIES = {
  '.git': true,
  'campfire-out': true
};

/**
 * Returns the list of build specification files contained within {@code dir}
 * (or the campfire root, if {@code dir} is not given).
 */
exports.Engine.prototype.findBuildFiles = function(dir) {
  var results = [];
  var dirs = [dir || '.'];
  while (dirs.length > 0) {
    var dir = dirs.pop();
    if (EXCLUDED_DIRECTORIES[dir]) {
      continue;
    }
    var files = fs.readdirSync(dir);
    for (var i = 0; i < files.length; i++) {
      var file = path.join(dir, files[i]);
      var stat = fs.lstatSync(file);
      if (stat.isDirectory()) {
        dirs.push(file);
      } else if (files[i] === BUILD_FILE_BASENAME) {
        results.push(file);
      }
    }
  }
  return results;
};
