var fs = require("fs");
var http = require("http");
var os = require("os");
var path = require("path");
var util = require("util");
var engine = require("./engine.js");
var shared = require("./shared.js");
var child_process = require('child_process');


function getNumberOfThreads(engine) {
  var setting = engine.settings.properties["threads"];
  if (setting) {
    return setting;
  } else {
    return os.cpus().length;
  }
}

function start(engine, kind, args, callback) {
  var targets = [];
  for (var i = 0; i < args.length; i++) {
    var resolved = engine.resolveTargets(args[i]);
    targets.append(resolved);
  }
  engine.run(kind, targets, getNumberOfThreads(engine), callback);
}

function startRegistry(engine, callback) {
 if (engine.settings.properties.convoy_bucket ||
   engine.settings.properties.start_registry == "false") {
   callback();
   return;
 }
 var bucket = engine.settings.properties.docker_gcs_bucket;
 var prefix = engine.settings.properties.docker_gcs_prefix || "/";
 var port = engine.settings.properties.docker_registry_port;
 var volume = engine.settings.properties.gcloud_config_volume;

 console.log("Starting docker registry for GCS...");
 child_process.exec(
     "docker run -d -e GCS_BUCKET=" + bucket +
         " -e STORAGE_PATH=" + prefix + " -p " + port + ":5000" +
         " --volumes-from " + volume  + " google/docker-registry",
     function (error, stdout, stderr) {
       if (error) {
         console.log('exec error: ' + error);
         console.log(stdout);
         console.log(stderr);
         process.exit(1);
       }
       engine.registryContainer = stdout.split('\n')[0];
       setTimeout(callback, 5000);
     });
}

function stopRegistry(engine, callback) {
 if (engine.settings.properties.convoy_bucket ||
   engine.settings.properties.start_registry == "false") {
   if (callback) {
     callback();
   }
   return;
 }
 console.log("Stopping docker registry for GCS...");
 child_process.exec(
     "docker stop " + engine.registryContainer);
 if (callback) {
   callback();
 }
}

function runContainerLogin() {
   child_process.spawn("docker", [ "run", "-t", "-i", "-name",
       "gcloud-config", "google/cloud-sdk", "gcloud", "auth",
       "login" ], { stdio: 'inherit' });
}

Command = function(run, help) {
 this.run = run;
 this.help = help;
};

function dedupLabels(labels) {
  var uniqueLabels = {};
  for (var k = 0; k < labels.length; k ++) {
    label = labels[k];
    var colon = label.indexOf(':');
    var lastSlash = label.lastIndexOf('/');
    if (lastSlash >= 0 && colon > lastSlash) {
      var name = label.substring(colon+1);
      var pkg = label.substring(lastSlash+1, colon);
      if (pkg == name) {
        label = label.substring(0, colon);
      }
    }
    uniqueLabels[label] = true;
  }
  return Object.keys(uniqueLabels);
}

function camper(input) {
  var parsed = JSON.parse(input);
  for (var j = 0; j < parsed.length; j++) {
    if (parsed[j].kind === "cc_external_lib") {
      // Order matters for these rules.
      continue;
    }
    for (var inputKind in parsed[j].inputs) {
      var labels = dedupLabels(parsed[j].inputs[inputKind]);
      labels.sort(function(a, b) {
        if (a.indexOf(":") === 0) {
          if (b.indexOf(":") === 0) {
            return a.localeCompare(b);
          } else {
            return -1;
          }
        } else if (b.indexOf(":") === 0) {
          return 1;
        }
        return a.localeCompare(b);
      });
      parsed[j].inputs[inputKind] = labels;
    }
  }
  return JSON.stringify(parsed, undefined, 2) + '\n';
}

exports.commands = {
  camper : new Command(function(engine, args) {
    if (args[0] === '-c') {
      // Check if a file is formatted correctly
      var input = fs.readFileSync(args[1]);
      var printed = camper(input);
      if (input != printed) {
        console.log('CAMPFIRE file is not formatted: ' + args[1]);
        process.exit(1);
      }
    } else {
      // Format each CAMPFIRE file in the repository
      var files = shared.findFiles(".", /.*\/CAMPFIRE$/);
      for (var i = 0; i < files.length; i++) {
        try {
          var input = fs.readFileSync(files[i]);
          var printed = camper(input);
          if (input != printed) {
            if (args.length === 0 || args[0] != "-n") {
              console.log("Rewriting " + files[i]);
              fs.writeFileSync(files[i], printed);
            } else {
              console.log("Would rewrite " + files[i]);
            }
          }
        } catch (e) {
          console.log("Camper error on file '" + files[i] + "': " + e);
          continue;
        }
      }
      console.log(process.env['USER'] + " is one happy camper");
    }
  }, "Formats CAMPFIRE files, use -n to see files to be rewritten" +
     " without applying the rewrite."),
  clean : new Command(function(engine, args) {
    if (fs.existsSync(".campfire_state")) {
      fs.unlinkSync(".campfire_state");
    }
    if (fs.existsSync("campfire-out")) {
      shared.rmdirs("campfire-out");
    }
  }, "Cleans campfire state and output"),
  build : new Command(function(engine, args) {
    start(engine, "build", args);
  }, "Builds provided targets"),
  deploy : new Command(function(engine, args) {
    startRegistry(engine, function() {
      start(engine, "deploy", args, function() {
        stopRegistry(engine);
      });
    });
  }, "Releases docker packages for provided targets"),
  "package" : new Command(function(engine, args) {
    startRegistry(engine, function() {
      start(engine, "package", args, function() {
        stopRegistry(engine);
      });
    });
  }, "Creates docker packages for provided targets"),
  pull : new Command(function(engine, args) {
    startRegistry(engine, function() {
      start(engine, "pull", args, function() {
        stopRegistry(engine);
      });
    });
  }, "Pulls existing docker packages for provided targets"),
  test : new Command(function(engine, args) {
    start(engine, "test", args);
  }, "Builds provided targets and runs the set of targets that are " +
      "tests"),
  query : new Command(function(engine, args) {
    engine.query(args[0]);
  }, "Queries build rule information from CAMPFIRE files"),
  print_action: new Command(function(engine, args) {
    var targets = [];
    for (var i = 0; i < args.length; i++) {
      var resolved = engine.resolveTargets(args[i]);
      targets.append(resolved);
    }
    engine.run("print_action", targets, 1);
  }, "Prints the actions that would be executed for the targets"),
  show_config: new Command(function(engine, args) {
    console.log(
        util.inspect(engine.settings.properties, {depth: undefined }));
  }, "Prints the configuration that will be used for the build."),
  gcloud_login: new Command(function(engine, args) {
    runContainerLogin();
  }, "Sets up campfire with credentials for gcloud, used for package " +
      "& deploy commands.")
};
