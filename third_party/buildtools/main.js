var util = require("util");
var fs = require("fs");
var engine = require("./engine.js");
var rule = require("./rule.js");
var commands = require("./commands.js");
var path = require("path");

function findRoot(dir) {
  if (fs.existsSync(path.join(dir, ".campfire_settings"))) {
    return dir;
  }
  if (dir == "/") {
    return undefined;
  }
  return findRoot(path.dirname(dir));
}
var cwd = process.cwd();
var campfireRoot = findRoot(cwd);

if (!campfireRoot) {
  console.log("Unable to locate .campfire_settings in any parent" +
      " directory of current working directory");
  process.exit(1);
}
var relative = cwd.substring(campfireRoot.length + 1);
var lockfile = path.join(campfireRoot, ".campfire_lock");

function inheritConfiguration(allConfigurations, inheritFrom,
    configObject) {
  if (!(inheritFrom in allConfigurations)) {
    console.log("No such configuration '" + inheritFrom + "'");
    process.exit(1);
  }
  var parentConfig = allConfigurations[inheritFrom];
  if ('@inherit' in parentConfig) {
    inheritConfiguration(allConfigurations, parentConfig['@inherit'],
        configObject);
  }
  for (var key in parentConfig) {
    if (key == '@inherit' || !parentConfig.hasOwnProperty(key)) {
      continue;
    } else if (key.charAt(0) == '+') {
      var actualKey = key.substr(1);
      if (actualKey in configObject) {
        configObject[actualKey] =
            configObject[actualKey].concat(parentConfig[key]);
      } else {
        configObject[actualKey] = parentConfig[key];
      }
    } else {
      configObject[key] = parentConfig[key];
    }
  }
}

function runCampfire() {
  process.chdir(campfireRoot);
  var data = fs.readFileSync(".campfire_settings");
  var settings = JSON.parse(data);

  var e = new engine.Engine(settings, campfireRoot, relative);
  if (!e.settings.properties) {
    e.settings.properties = {};
  }
  if (!e.settings.configurations) {
    e.settings.configurations = {};
  }
  if (!e.settings.configurations.base) {
    e.settings.configurations.base = {};
  }
  var cmdlineConfig = {};
  var targets = [];
  var command = process.argv[2];
  for (var i =3; i < process.argv.length; i++) {
    var value = process.argv[i];
    if (value.indexOf("--") === 0) {
      var rest = value.substring(2);
      var eqIndex = value.indexOf('=');
      if (eqIndex == -1) {
        cmdlineConfig[rest] = true;
      } else {
        var key = rest.substring(0, eqIndex - 2);
        var begin = rest.substring(eqIndex - 1);
        cmdlineConfig[key] = begin;
      }
      continue;
    }
    targets.push(value);
  }
  if (!('configuration' in cmdlineConfig)) {
    if ('configuration' in e.settings) {
      cmdlineConfig['configuration'] = e.settings['configuration'];
    } else {
      console.log("No default configuration and no configuration " +
          "specified.");
      console.log("Specify a configuration with ");
      console.log("     --configuration='foo' or ");
      console.log("     'configuration': 'foo' in the top level of");
      console.log("     your .campfire_settings file.");
      process.exit(1);
    }
  }
  cmdlineConfig['@inherit'] = cmdlineConfig['configuration'];
  e.settings.configurations['@cmdline'] = cmdlineConfig;
  inheritConfiguration(e.settings.configurations,
      '@cmdline', e.settings.properties);
  if (!command || command == 'help') {
    console.log("usage: campfire command, arguments");
    console.log("");
    console.log("the following commands are available:");
    for (var availableCommand in commands.commands) {
      console.log("\t" + availableCommand+":\t" +
          commands.commands[availableCommand].help);
    }
  } else {
    var resolvedCommand = commands.commands[command];
    if (!resolvedCommand) {
      console.log("Unknown command: " + command);
      process.exit(1);
    }
    resolvedCommand.run(e, targets);
  }
}

function acquireLock(lastPID, lastOrphaned) {
  var lock = undefined;
  var pid = new Buffer(process.pid+'');
  var conflictingPID;
  try {
    lock = fs.openSync(lockfile, "wx");
    var written = 0;
    while (written < pid.length) {
      written +=
          fs.writeSync(lock, pid, written, pid.length-written, 0);
    }
    fs.closeSync(lock);
  } catch (err) {
    if (err.code != "EEXIST") {
      console.log("Error creating lock file:");
      console.log(err);
      process.exit(2);
    }
    conflictingPID = fs.readFileSync(lockfile).toString();
  }
  if (!lock) {
    var orphaned = false;
    try {
      process.kill(conflictingPID, 0);
    } catch (err) {
      orphaned = true;
    }
    if (lastPID !== conflictingPID || lastOrphaned !== orphaned) {
      if (orphaned) {
        console.log("Stray lock file detected; please remove '" +
            lockfile + "'");
      } else {
        console.log("Waiting for Campfire lock [held by " +
        conflictingPID + "]");
      }
    }
    setTimeout(acquireLock, 250, conflictingPID, orphaned);
    return;
  }

  // Remove lock on exit.
  process.on('exit', function(code) {
    fs.unlink(lockfile);
  });
  process.on('SIGINT', function() {
    console.log("Caught interrupt signal");
    fs.unlink(lockfile);
    process.exit(130);
  });

  runCampfire();
}

acquireLock();
