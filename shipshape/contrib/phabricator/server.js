// base modules from node
var child_process = require('child_process');
var fs= require('fs');
var http = require('http');
var url = require('url');
var util = require('util');

// npm installed modules
// winston is a library for writing log messages
var winston = require('winston');

// modules from this application
var async = require('./async.js');
var conduit = require('./conduit.js');

/*
 * Processes a shipshape json output file and posts its messages
 * as inline comments to phabricator
 * jsonFile -> file shipshape wrote its output to
 * revisionID -> revisionID from phabricator
 * diffID -> diffID from phabricator
 * callback -> callback, signature: function(error)
 */
function processShipshapeJson(jsonFile, revisionID, diffID, callback) {
  // We filter the notes from shipshape to only care about notes
  // associated with files that are part of the code review.
  // TODO(jvg): modify shipshape CLI to allow passing these in to
  // shipshape to reduce analysis & processing time.
  conduitInstance.differentialGetCommitPaths(revisionID,
      function(error, result) {
        if (error !== undefined) {
          callback(error);
          return;
        }

        var files = {};
        for (var key in result) {
          files[result[key]] = 1;
        }
        var input = fs.readFileSync(jsonFile);
        var parsed = JSON.parse(input);
        if (!parsed.analyze_response) {
          callback("Invalid shipshape response file: " + jsonFile);
          return;
        }
        var relevant = [];
        for (var response in parsed.analyze_response) {
          var notes = response.note;
          if (notes !== undefined) {
            for (var note in notes) {
              if (note.location && note.location.path &&
                  files[note.location.path]) {
                relevant.push(note);
              }
            }
          }
        }
        if (relevant.length == 0) {
          callback();
          return;
        }

        // async post each note as an inline comment to phabricator.
        var postRelevant = function(item, callback) {
          conduitInstance.differentialCreateInline(revisionID,
              diffID, item.location.path, true,
              item.location.range.start_line, undefined,
              item.category + ": " + item.description, callback);
        };

        // run above function async over each relevant entry.
        async.serialAsyncForEach(relevant, postRelevant,
            function(error) {
              if (error !== undefined) {
                callback(error);
                return;
              }
              // inline comments won't be committed/shown until
              // a createComment is posted with attachInlines == true
              conduitInstance.differentialCreateComment(
                  revisionID, "", "comment", false /* silent */,
                      true /* attachInlines */, callback);
            });
      });
}

/*
 * Runs analysis job:
 * -clone/sync repo
 * -apply diff using "arc patch --diff"
 * -run shipshape CLI
 * -process shipshape results
 */
function analyze(workerID, revisionID, diffID, callback) {
  var jsonOutput = "/output/shipshape_" + diffID + ".json";
  var repoDir = "/repos/shipshape_repo_" + workerID;
  var mergedOut = [];

  var appendMerged = function(response) {
     mergedOut.push(response);
  };

  var child = child_process.spawn(analysisScript,  [repoDir, diffID,
      jsonOutput]);
  child.stdout.on('data', appendMerged);
  child.stdout.on('end', appendMerged);
  child.stderr.on('data', appendMerged);
  child.stderr.on('end', appendMerged);

  child.on('close',
      function(code) {
        if (code !== 0) {
          callback("\'" + analysisScript + "\' failed with exit code " +
              code + ", out: " + mergedOut.join(""));
        } else {
          processShipshapeJson(jsonOutput, revisionID, diffID,
              callback);
        }
      });
}

/*
 * Runs shipshape analysis for the provided job.
 * Upon success posts build success to phabricator
 */
function analysisRunner(value, workerID, callback) {
  analyze(workerID, value.revisionID, value.diffID, function(error) {
      if (error !== undefined) {
        callback(error);
        return;
      }
      conduitInstance.harbormasterSendMessage(value.PHID, "pass",
          callback);
    });
}

/*
 * Callback called for any failure in the analysis process.
 * Reports details of failure as comment to the phabricator review
 * and marks the build as failed.
 */
function analysisFailureHandler(error, value) {
  winston.error("Analysis of revision " + value.revisionID +
      ", diff " + value.diffID + " failed: " + error);

  conduitInstance.differentialCreateComment(
    value.revisionID, "Shipshape analysis failed: " + error, "comment",
    false, true, function(error) {
      if (error !== undefined) {
        winston.error("unable to report error message as comment to " +
            "Phabricator in analysis failure handler: " + error);
      }
      conduitInstance.harbormasterSendMessage(value.PHID, "fail",
        function(error) {
          if (error !== undefined) {
            winston.error("unable to report error build status to " +
                " Phabricator in analysis failure handler: " + error);
          }
        });
    });
}

function getEnvironmentVariableOrFail(name) {
  var value = process.env[name];
  if (value === undefined) {
    winston.error("Unable to read required environment variable '" +
        name +"'");
    process.exit(1);
  }
  return value;
}

function getEnvironmentVariableOrDefault(name, defaultValue) {
  var value = process.env[name];
  return value === undefined ? defaultValue : value;
}

// http(s) address of the phabricator instance to talk to
var server = getEnvironmentVariableOrFail("PHABRICATOR_SERVER");
// Port to have our web server listen to
var httpPort = getEnvironmentVariableOrFail("SHIPSHAPE_HTTP_PORT");
// Secret that should be part of the http request coming from
// phabricator, to ensure we only process valid requests.
var shipshapeSecret = getEnvironmentVariableOrFail("SHIPSHAPE_SECRET");
// How many analysis workers can run in parallel,
// (# phabricator builds handled at a time).
var maxAnalysisWorkers =
    getEnvironmentVariableOrDefault("ANALYSIS_WORKER_COUNT", 4);
// IP Address to bind our web server to
var httpAddress = getEnvironmentVariableOrDefault(
    "SHIPSHAPE_HTTP_ADDRESS", '0.0.0.0');
// Path to the analysis shell script used for syncing the repo
// and running shipshape
var analysisScript = getEnvironmentVariableOrDefault(
    "SHIPSHAPE_ANALYSIS_SCRIPT", '/shipshape/analyze.sh');
// pool used to async process analysis requests.
var analyzePool = new async.JobPool(analysisRunner,
    analysisFailureHandler, maxAnalysisWorkers);
// used to call phabricator's conduit APIs.
var conduitInstance = new conduit.Conduit(server);

/*
 * Start a webserver to listen for incoming request from phabricator
 * and process any valid request by enqueuing the build into the
 * analyzePool
 */
try {
  http.createServer(function (req, res) {
    var path = url.parse(req.url, true);
    winston.info('incoming request: %s from %s', req.url,
        req.socket.remoteAddress);
    if (!path.query || !path.query.phid || !path.query.rev ||
        !path.query.diff || !path.query.secret ||
        path.query.secret != shipshapeSecret) {
      winston.warn('invalid request: %s from %s', req.url,
          req.socket.remoteAddress);
      res.writeHead(500, {'Content-Type': 'text/plain'});
      res.end("Internal server error");
      return;
    }

    analyzePool.enqueue({
        PHID: path.query.phid,
        revisionID: path.query.rev,
        diffID: path.query.diff
    });

    res.writeHead(200, {'Content-Type': 'text/plain'});
    res.end("OK");
  }).listen(httpPort, httpAddress);
  winston.info("Server listening on " + httpAddress +":" + httpPort);
} catch (e) {
  winston.error("Error starting server: " + e);
  process.exit(1);
}
