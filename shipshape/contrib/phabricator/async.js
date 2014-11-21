// This file contains helpers to make async work in node easier.

/*
 * JobPool is a queue that has a set of runners associated with it
 * that will process entries in this pool in an async manner.
 * runner is the function called to process each entry
 *   its signature is: function(value, workerId, callback(error))
 * failureHandler is a function called when runner reports failure.
 *   its signature is: function(error, value)
 * max is the maximum amount of concurrent runners that can be active
 *   at a given time.
 */
exports.JobPool = function(runner, failureHandler, max) {
  this.runner = runner;
  this.failureHandler = failureHandler;
  this.queue = [];
  this.available = [];
  for (var i = 0; i < max; i++) {
    this.available.push(i);
  }
};

/*
 * Enqueues a value in the JobPool and starts processing if there are
 * any runners idle.
 */
exports.JobPool.prototype.enqueue = function(value) {
  this.queue.push(value);
  if (this.available.length > 0) {
    this._run(this.available.pop());
    this.running++;
  }
};

// Implementation logic of the JobPool
exports.JobPool.prototype._run = function(id) {
  var parent = this;
  if (this.queue.length == 0) {
    this.available.push(id);
    return;
  }
  var value = this.queue.pop();
  this.runner(value, id,
      function(error) {
        if (error !== undefined) {
          parent.failureHandler(error, value);
        }
        parent._run(id);
      });
};

/*
 * A helper to asynchrously process an array of items.
 * items is the array to process
 * handler is a function that processes each individual item
 *  its signature is: function(value, callback(error))
 * callback is a function that is called on completion/failure
 *  its signature is: function(error)
 */
exports.serialAsyncForEach = function(items, handler, callback) {
  if (items.length === 0) {
    callback();
    return;
  }
  var value = items.shift();
  handler(value, function(error) {
    if (error !== undefined) {
      callback(error);
      return;
    }
    exports.serialAsyncForEach(items, handler, callback);
  });
}

