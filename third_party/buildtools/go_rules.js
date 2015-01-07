'use strict';

var path = require('path');

var entity = require('./entity.js');
var rule = require('./rule.js');

exports.PACKAGE_DIR = 'campfire-out/go/pkg/linux_amd64/';

function GoLibrary(engine) {
  this.engine = engine;
}

GoLibrary.prototype = new rule.Rule();
GoLibrary.prototype.getNinjaBuilds = function(target) {
  var includePaths = rule.getAllOutputsFor(target.inputsByKind['go_pkgs'], 'build',
                                           rule.propertyFilter('go_include_path'))
                                               .map(function(p) { return p.value; });
  includePaths.push(exports.PACKAGE_DIR);
  var srcs = rule.getAllOutputsFor(target.inputsByKind['srcs'], 'build',
                                   rule.fileFilter('src_file', '.go'));
  var pkgs = rule.getAllOutputsFor(target.inputsByKind['go_pkgs'], 'build',
                                   rule.fileFilter('go_archive'));
  var pkgName = path.dirname(target.asPath());
  var outPath = exports.PACKAGE_DIR + pkgName;
  return {
    BUILD: [{
      rule: 'go_compile',
      inputs: srcs,
      implicits: pkgs,
      outs: [target.getFileNode(outPath + '.a', 'go_archive')],
      vars: {
        'package': pkgName,
        include: includePaths.join(' ')
      }
    }]
  };
};

function GoBinary(engine) {
  this.engine = engine;
}

GoBinary.prototype = new rule.Rule();
GoBinary.prototype.getExecutable = function(target) {
  return target.getFileNode(target.getRoot('bin'), 'go_executable');
};
GoBinary.prototype.getNinjaBuilds = function(target) {
  var includePaths =
      rule.getAllOutputsRecursiveFor(target.inputsByKind['go_pkgs'], 'build',
                                     rule.propertyFilter('go_include_path'))
                                         .map(function(p) { return p.value; });
  var recursiveIncludePaths =
      rule.getAllOutputsRecursiveFor(target.inputsByKind['go_pkgs'], 'build',
                                     rule.propertyFilter('go_include_path'))
                                         .map(function(p) { return p.value; });
  includePaths.push(exports.PACKAGE_DIR);
  recursiveIncludePaths.push(exports.PACKAGE_DIR);

  var ccLibs = ccLibs = rule.getAllOutputsRecursiveFor(
      (target.inputsByKind['go_pkgs'] || [])
          .concat(target.inputsByKind['cc_libs'] || []),
      'build', rule.fileFilter('cc_archive'));
  var extLDArgs = [];
  for (var j = 0; j < ccLibs.length; j++) {
    extLDArgs.push('-L' + path.dirname(ccLibs[j].getPath()));
    extLDArgs.push('-Wl,-rpath=' +
        path.dirname(ccLibs[j].getPath()));
  }

  var srcs = rule.getAllOutputsFor(target.inputsByKind['srcs'], 'build',
                                   rule.fileFilter('src_file', '.go'));
  var pkgs = rule.getAllOutputsFor(target.inputsByKind['go_pkgs'], 'build',
                                   rule.fileFilter('go_archive'));
  var archive = target.getFileNode(exports.PACKAGE_DIR + target.asPath() + '.a',
                                   'go_archive');

  return [{
    rule: 'go_compile',
    inputs: srcs,
    implicits: pkgs,
    outs: [archive],
    vars: {
      include: includePaths.join(' ')
    }
  },{
    rule: 'go_linker',
    inputs: [archive],
    implicits: pkgs,
    outs: [this.getExecutable(target)],
    vars: {
      include: recursiveIncludePaths.join(' '),
      extldflags: extLDArgs.join(' ')
    }
  }];
};

function getLibInputs(targets, kind, filter) {
  var inputs = targets
      .mapcat(function(t) { return t.inputsByKind[kind] || []; });
  return rule.getAllOutputsFor(inputs, 'build', filter);
}

function getRecursiveLibInputs(targets, kind, filter) {
  var inputs = targets
      .mapcat(function(t) { return t.inputsByKind[kind] || []; });
  return rule.getAllOutputsRecursiveFor(inputs, 'build', filter);
}

function GoTest(engine) {
  this.engine = engine;
}

GoTest.prototype = new rule.Rule();
GoTest.prototype.getNinjaBuilds = function(target) {
  // Dependent go_library inputs
  var lib = target.inputsByKind['go_lib'];
  if (!lib || lib.length != 1) {
    console.error('ERROR: missing go_lib for target ' + target.id);
    process.exit(1);
  }
  var libSrcInputs =
      getLibInputs(lib, 'srcs', rule.fileFilter('src_file', '.go'));
  var libPkgDeps = getLibInputs(lib, 'go_pkgs', rule.fileFilter('go_archive'));
  var libIncludePaths =
      getLibInputs(lib, 'go_pkgs', rule.propertyFilter('go_include_path'));
  var ccLibs = getRecursiveLibInputs(lib, 'go_pkgs',
                                     rule.fileFilter('cc_archive'));
  ccLibs.append(getRecursiveLibInputs(lib, 'cc_libs',
                                      rule.fileFilter('cc_archive')));

  // Test inputs
  var srcs = rule.getAllOutputsFor(target.inputsByKind['srcs'], 'build',
                                   rule.fileFilter('src_file', '.go'))
                                       .concat(libSrcInputs);
  var pkgs = rule.getAllOutputsFor(target.inputsByKind['go_pkgs'], 'build',
                                   rule.fileFilter('go_archive'))
                                       .concat(libPkgDeps);
  var includePaths =
      rule.getAllOutputsFor(target.inputsByKind['go_pkgs'], 'build',
                            rule.propertyFilter('go_include_path'))
                                .concat(libIncludePaths)
                                .map(function(p) { return p.value; });
  includePaths.push(exports.PACKAGE_DIR);
  var generator =
      rule.getExecutable(this.engine,
                         this.engine
                             .settings.properties['go_testmain_generator']);

  var testPkg = path.dirname(target.asPath()) + '_test';

  var testMainSrc = target.getFileNode(path.join(target.getRoot('gen'),
                                                 'testmain.go'),
                                       'go_object');
  var outPath = exports.PACKAGE_DIR + testPkg;
  var testArchive = target.getFileNode(outPath + '.a', 'go_archive');
  var testMainArchive =
      target.getFileNode(outPath + 'main.a', 'go_archive');

  var builds = {
    BUILD: [{
      rule: 'go_testmain',
      inputs: srcs,
      outs: [testMainSrc],
      implicits: [generator],
      vars: {
        'package': testPkg,
        generator: generator.getPath()
      }
    }]
  };
  builds.BUILD.push({
    rule: 'go_compile',
    inputs: srcs,
    implicits: pkgs,
    outs: [testArchive],
    vars: {
      'package': testPkg,
      include: includePaths.join(' ')
    }
  });
  builds.BUILD.push({
    rule: 'go_compile',
    inputs: [testMainSrc],
    implicits: [testArchive],
    outs: [testMainArchive],
    vars: {
      'package': 'main',
      include: includePaths.join(' ')
    }
  });

  var extLDArgs = [];
  for (var j = 0; j < ccLibs.length; j++) {
    extLDArgs.push('-L' + path.dirname(ccLibs[j].getPath()));
    extLDArgs.push('-Wl,-rpath=' +
        path.dirname(ccLibs[j].getPath()));
  }
  var recursiveIncludePaths =
      rule.getAllOutputsRecursiveFor(target.inputsByKind['go_pkgs'], 'build',
                                     rule.propertyFilter('go_include_path'));
  recursiveIncludePaths
      .append(getRecursiveLibInputs(lib, 'go_pkgs',
                                    rule.propertyFilter('go_include_path')));
  recursiveIncludePaths = recursiveIncludePaths
      .map(function(p) { return p.value; });
  recursiveIncludePaths.push(exports.PACKAGE_DIR);
  var testBinary = target.getFileNode(target.getRoot('bin'), 'go_executable');
  builds.BUILD.push({
    rule: 'go_linker',
    inputs: [testMainArchive],
    implicits: pkgs,
    outs: [testBinary],
    vars: {
      include: recursiveIncludePaths.join(' '),
      extldflags: extLDArgs.join(' ')
    }
  });
  var testLog = target.getFileNode(target.getRoot('test') + '.log', 'test_log');
  builds.TEST = [{
    rule: 'run_test',
    inputs: [testBinary],
    outs: [target.getFileNode(target.getRoot('test') + '.done', 'done_marker')],
    vars: {
      log: testLog.getPath()
    }
  }];
  if (target.getProperty('go_benchmark')) {
    var benchLog =
        target.getFileNode(target.getRoot('bench') + '.log', 'test_log');
    builds.BENCH = [{
      rule: 'run_test',
      inputs: [testBinary],
      outs: [target.getFileNode(target.getRoot('bench') + '.done',
                                'done_marker')],
      vars: {
        args: '-test.bench ".*"',
        log: benchLog.getPath(),
        pool: 'console' // Ensure benchmarks run serially
      }
    }];
  }

  return builds;
};

function GoExternalLib(engine) {
  this.engine = engine;
}

GoExternalLib.prototype = new rule.Rule();
GoExternalLib.prototype.getOutputsFor = function(target, kind) {
  if (target.outs) {
    return target.outs;
  }
  var inputs = rule.getAllOutputsFor(target.inputsByKind['srcs'],
                                     kind, rule.fileFilter('src_file', '.a'));

  var outputs = [];
  for (var i = 0; i < inputs.length; i++) {
    inputs[i].kind = 'go_archive';
    outputs.push(inputs[i]);
  }

  var includePath = target.getProperty('go_include_path');
  if (includePath) {
    outputs.push(includePath);
  }

  target.outs = outputs;
  return outputs;
};

exports.register = function(engine) {
  engine.addRule('go_library', new GoLibrary(engine));
  engine.addRule('go_test', new GoTest(engine));
  engine.addRule('go_binary', new GoBinary(engine));
  engine.addRule('go_external_lib', new GoExternalLib(engine));
};
