'use strict';

var path = require('path');

var entity = require('./entity.js');
var kythe_rules = require('./kythe_rules.js');
var rule = require('./rule.js');

// A property that, when specified on a cc_external_lib, requires
// additional flags to be passed to the linker. Takes a list of strings
// as its value.
var EXTRA_LINK_FLAGS_PROPERTY = 'cc_extra_link_flags';

var INCLUDE_PATH_PROPERTY = 'cc_include_path';
var SYSTEM_INCLUDE_PATH_PROPERTY = 'cc_system_include_path';

// Append these copts when compiling only this target. List of strings.
var LOCAL_COPTS_PROPERTY = 'cc_local_copts';

// Append these copts when compiling this target or any target that
// depends on it. Useful on cc_library or cc_external_lib. List of
// strings.
var EXPORTED_COPTS_PROPERTY = 'cc_exported_copts';

function CcLibrary(engine) {
  this.engine = engine;
}

CcLibrary.prototype = new rule.Rule();
CcLibrary.prototype.getNinjaBuilds = function(target) {
  var builds = [];

  var srcs = rule.getAllOutputsFor(target.inputsByKind['srcs'], 'build',
                                   rule.fileFilter('src_file', '.cc'));
  srcs.append(rule.getAllOutputsFor(target.inputsByKind['srcs'], 'build',
                                    rule.fileFilter('src_file', '.c')));
  var deps =
      rule.getAllOutputsRecursiveFor(
          target.inputs, 'build',
          rule.fileFilter('src_file', '.h')).concat(
              rule.getAllOutputsFor(target.inputsByKind['cc_libs'],
                                    'build', rule.fileFilter('cc_archive')));

  var exportedProperties = target.getProperty(EXPORTED_COPTS_PROPERTY) || [];
  if (target.getProperty(INCLUDE_PATH_PROPERTY)) {
    exportedProperties.push(target.getProperty(INCLUDE_PATH_PROPERTY));
  }

  var copts = getBaseCOpts(target);
  var coptsProp = target.getProperty('copts');
  var extractions = [];
  var objects = [];
  for (var i = 0; i < srcs.length; i++) {
    var srcPath = srcs[i].getPath();
    var lang = path.extname(srcPath) == '.cc' ? 'cpp' : 'c';
    var obj =
        target.getFileNode(path.join(target.getRoot('gen'),
                                     path.dirname(srcPath),
                                     path.basename(srcPath,
                                                   path.extname(srcPath)) + '.o'),
                           'cc_object');
    var opts = (lang == 'cpp' && coptsProp)
        ? copts.concat(coptsProp.value)
        : copts;
    var compile = {
      rule: lang + '_compile',
      inputs: [srcs[i]],
      outs: [obj],
      implicits: deps,
      vars: {
        copts: opts.join(' ')
      }
    };
    builds.push(compile);
    var kindex = target.getFileNode(target.getRoot('gen') +
        path.basename(srcs[i].getPath()) + '.c++.kindex', 'kindex');
    extractions.push(kythe_rules.cxxNinjaExtractor(target, compile, kindex));
    objects.push(obj);
  }
  builds.push({
    rule: 'archive',
    inputs: objects,
    outs: [target.getFileNode(target.getRoot('bin') + '.a', 'cc_archive')],
    properties: exportedProperties
  });

  return {
    BUILD: builds,
    EXTRACT: extractions
  };
};

function getBaseCOpts(target) {
  var copts =
      rule.getAllOutputsFor(target.inputsByKind['cc_libs'], 'build',
                            rule.propertyFilter(EXPORTED_COPTS_PROPERTY))
                                .map(function(p) { return p.value; })
                                .reduce(function(p, n) { return p.concat(n); }, []);
  var localCopts = target.getProperty(LOCAL_COPTS_PROPERTY);
  if (localCopts) {
    copts = copts.concat(localCopts.value);
  }
  var exportedCopts = target.getProperty(EXPORTED_COPTS_PROPERTY);
  if (exportedCopts) {
    copts = copts.concat(exportedCopts.value);
  }

  var includePaths =
      rule.getAllOutputsFor(target.inputsByKind['cc_libs'], 'build',
                            rule.propertyFilter(INCLUDE_PATH_PROPERTY));
  var systemIncludePaths =
      rule.getAllOutputsFor(target.inputsByKind['cc_libs'], 'build',
                            rule.propertyFilter(SYSTEM_INCLUDE_PATH_PROPERTY));
  copts.append(includePaths
      .map(function(p) { return '-I ' + p.value; }));
  copts.append(systemIncludePaths
      .map(function(p) { return '-ISYSTEM ' + p.value; }));

  copts.push('-I.');
  return copts;
}

function CcBinary(engine) {
  this.engine = engine;
}

CcBinary.prototype = new rule.Rule();
CcBinary.prototype.getExecutable = function(target) {
  return target.getFileNode(target.getRoot('bin'), 'cc_executable');
};
CcBinary.prototype.getNinjaBuilds = function(target) {
  var extraLinkFlags = rule.getAllOutputsRecursiveFor(
      target.inputsByKind['cc_libs'], 'build',
      rule.propertyFilter(EXTRA_LINK_FLAGS_PROPERTY));
  var flags = extraLinkFlags
      .map(function(p) { return p.value; })
      .reduce(function(p, n) { return p.concat(n); }, [])
      .join(' ');
  return [{
    rule: 'linker',
    inputs: rule.getAllOutputsRecursiveFor(
        target.inputsByKind['cc_libs'], 'build',
        rule.fileFilter('cc_archive')),
    outs: [this.getExecutable(target)],
    vars: {
      flags: flags
    }
  }];
};

function CcTest(engine) {
  this.engine = engine;
}

CcTest.prototype = new CcBinary();
CcTest.prototype.getNinjaBuilds = function(target) {
  var builds = {
    BUILD: CcBinary.prototype.getNinjaBuilds.call(this, target)
  };
  if (builds.BUILD.length != 1 || builds.BUILD[0].outs.length != 1) {
    throw 'ERROR: unexpected CcBinary ninja builds';
  }
  var logFile = target.getFileNode(target.getRoot('test') + '.log', 'test_log');
  builds.TEST = [{
    rule: 'run_test',
    inputs: builds.BUILD[0].outs,
    outs: [target.getFileNode(target.getRoot('test') + '.done', 'done_marker')],
    vars: {
      log: logFile.getPath()
    }
  }];
  return builds;
};

function CCExternalLib(engine) {
  this.engine = engine;
}

CCExternalLib.prototype = new rule.Rule();
CCExternalLib.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var inputs = rule.getAllOutputsFor(target.inputsByKind['srcs'], kind,
                                     rule.fileFilter('src_file', '.so'));
  inputs.append(rule.getAllOutputsFor(target.inputsByKind['srcs'], kind,
                                      rule.fileFilter('src_file', '.a')));

  var outputs = [];
  for (var i = 0; i < inputs.length; i++) {
    inputs[i].kind = 'cc_archive';
    outputs.push(inputs[i]);
  }

  var includePath = target.getProperty(INCLUDE_PATH_PROPERTY);
  if (includePath) {
    outputs.push(includePath);
  }

  var extraLinkFlags = target.getProperty(EXTRA_LINK_FLAGS_PROPERTY);
  if (extraLinkFlags) {
    outputs = outputs.concat(extraLinkFlags);
  }

  var exportedCopts = target.getProperty(EXPORTED_COPTS_PROPERTY);
  if (exportedCopts) {
    outputs = outputs.concat(exportedCopts);
  }

  return outputs;
};

exports.register = function(engine) {
  engine.addRule('cc_library', new CcLibrary(engine));
  engine.addRule('cc_binary', new CcBinary(engine));
  engine.addRule('cc_test', new CcTest(engine));
  engine.addRule('cc_external_lib', new CCExternalLib(engine));
};
