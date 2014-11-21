var child_process = require("child_process");

// An object that makes it easy to call conduit APIs from node.
exports.Conduit = function(server) {
  this.server = server;
}

/**
 * Calls Phabricator Conduit APIs using arc.
 * We use arc as it does authentication for us and we need to run
 * it on this server anyway for running 'arc patch --diff'.
 * method -> string of conduit method to call
 * details -> javascript object to serialize as json to call to conduit
 * callback -> called after 'arc' call completes (success & failure)
 *   signature: function(error, details)
 */
exports.Conduit.prototype._call = function(method, details, callback) {
  var stdout = [];
  var stderr = [];

  var appendStdout = function(response) {
     stdout.push(response);
  };

  var appendStderr = function(response) {
     stderr.push(response);
  };

  child = child_process.spawn("arc", [ "call-conduit",
      "--conduit-uri=" + this.server, method ]);
  // register callbacks to child process to capture stdout & stderr
  child.stdout.on('data', appendStdout);
  child.stdout.on('end', appendStdout);
  child.stderr.on('data', appendStderr);
  child.stderr.on('end', appendStderr);

  child.on('close',
      function(code) {
        if (code !== 0) {
          callback("call to arc failed: " + stderr.join(""),
              // errors are reported as json to stdout as well.
              JSON.parse(stdout.join("")));
        } else {
          callback(undefined, JSON.parse(stdout.join("")));
        }
      });

  child.stdin.write(JSON.stringify(details));
  child.stdin.end();
};

// Functions below map 1:1 to phabricator conduit API
// as can be seen on /conduit/ on the phabricator UI.

exports.Conduit.prototype.differentialGetCommitPaths =
    function(revisionID, callback){
  var details = {
      revision_id: revisionID
  };
  this._call("differential.getcommitpaths", details, callback);
};

exports.Conduit.prototype.differentialCreateComment =
    function(revisionID, message, action, silent, attachInlines,
    callback) {
  var details = {
      revision_id: revisionID,
      message: message,
      action: action,
      silent: silent,
      attach_inlines: attachInlines
  };
  this._call("differential.createcomment", details, callback);
};

exports.Conduit.prototype.differentialCreateInline = function(
    revisionID, diffID, filePath, isNewFile, lineNumber, lineLength,
    content, callback) {
  var details = {
      revisionID: revisionID,
      diffID: diffID,
      filePath: filePath,
      isNewFile: isNewFile,
      lineNumber: lineNumber,
      lineLength: lineLength,
      content: content
  };
  this._call("differential.createinline", details, callback);
};

exports.Conduit.prototype.harbormasterSendMessage = function(
    buildTargetPHID, type, callback) {
  var details = {
      buildTargetPHID: buildTargetPHID,
      type: type
  };
  this._call("harbormaster.sendmessage", details, callback);
};
