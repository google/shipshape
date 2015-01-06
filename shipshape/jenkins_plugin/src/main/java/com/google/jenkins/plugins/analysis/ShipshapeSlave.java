/*
 * Copyright 2014 Google Inc. All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package com.google.jenkins.plugins.analysis;

import com.google.common.base.Joiner;
import com.google.common.collect.ImmutableList;
import com.google.protobuf.InvalidProtocolBufferException;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalysisFailure;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalyzeResponse;
import com.google.shipshape.proto.ShipshapeRpcProto.ShipshapeRequest;
import com.google.shipshape.proto.ShipshapeRpcProto.ShipshapeResponse;

import hudson.FilePath;
import hudson.model.BuildListener;
import hudson.remoting.Callable;

import java.io.BufferedOutputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.PrintWriter;
import java.io.Serializable;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.URL;
import java.nio.file.Paths;
import java.util.Arrays;
import java.util.List;

public class ShipshapeSlave implements Callable<Integer, Exception>, Serializable {
  private static final long serialVersionUID = 1L;

  // Repo
  private static final String REPO_HOST = "gcr.io"
  private static final String REPO_REGISTRY = "shipshape_registry";
  private static final String REPO = REPO_HOST + "/_b_" + REPO_REGISTRY;

  // The name of the shipping_container image, as found in the GCS bucket.
  private static final String SHIPPING_CONTAINER_IMAGE = "service";

  // The tag used to distinguish the released version of the shipping_container image.
  private static final String SHIPPING_CONTAINER_IMAGE_TAG = "prod";

  // Name of the docker container started by this plugin. This name is used to check if
  // a container is already running. Has no dependency to anything outside the plugin.
  private static final String SHIPPING_CONTAINER_NAME = "analysis_runner";

  // Mount points for volumes shared with the shipping_container image. These paths are used
  // inside the container to read from and write to files.
  private static final String JENKINS_OUTPUT_MOUNT_POINT = "/shipshape-output";
  private static final String JENKINS_WORKSPACE_MOUNT_POINT = "/shipshape-workspace";

  // The name of the output file, with a blub of analyzer output.
  private static final String RESULT_FILE_NAME = "analysis-log.txt";
  // The name of the shipshape log file. This file name may be used in integration tests.
  private static final String SHIPSHAPE_LOG_FILE_NAME = "shipshape-log.txt";
  // Encoding used for output files.
  private static final String FILE_ENCODING = "UTF-8";

  // Configuration used when retrying tasks, such as 'connecting to the docker registry' or
  // 'sending a request to ShippingContainer'.
  private static final int RETRY_MAX_ATTEMPTS = 10;
  private static final long RETRY_INIT_DELAY_MS = 20;
  private static final int RETRY_DELAY_MULTIPLIER = 3;

  // The port used to communicate to docker on localhost via a TCP socket. This form of
  // communication requires that docker is configured to communicate on that port on start up.
  private static final int DOCKER_TCP_PORT = 4243;

  // The port used to communicate with the Shipshape service. This is configured by the service on
  // start up and mapped to the same port value when the shipping_container container is brought up
  // from this plugin.
  private static final int SHIPSHAPE_SERVICE_PORT = 10007;

  // Docker Remote API
  // (See https://docs.docker.com/reference/api/docker_remote_api/ for documentation).
  // Provides an alternative to the docker command-line interface provided that there is a TCP
  // socket configured for docker.
  //
  // Lists all containers (including non-running ones).
  private static final String DOCKER_REST_API_ALL_CONTAINERS = "containers/json?all=1";
  
  private FilePath workspace;
  private boolean isVerbose;
  private ImmutableList<String> categoryList;
  private String socket;
  private String jobName;
  private BuildListener listener;
  private Stage stage;
  
  /**
   * @param workspace
   * @param isVerbose
   * @param categoryList
   * @param socket
   * @param jobName
   */
  public ShipshapeSlave(FilePath workspace, boolean isVerbose, ImmutableList<String> categoryList,
      String socket, String jobName, Stage stage, BuildListener listener) {
    super();
    this.workspace = workspace;
    this.isVerbose = isVerbose;
    this.categoryList = categoryList;
    this.socket = socket;
    this.jobName = jobName;
    this.stage = stage;
    this.listener = listener;
  }

  @Override
  public Integer call() throws Exception {
    // Anything inside this method (along with called methods) are executed on the Jenkins
    // slave on a master-slave configuration).

    // Fetching workspace path and output path here as they are laid out differently on a
    // Jenkins slave.
    String workspacePath = workspace.absolutize().toString();
    // Setting the output path to be the same as the workspace path in case only one 
    // directory is mounted.
    String outputPath = workspacePath;
    
    // The print streams used for logging are not serializable and need to be recreated
    // here to enable logging on the Jenkins slave. Don't close the build log stream,
    // others will be using it after us.
    ShipshapeStreamLogger logger = new ShipshapeStreamLogger();
    logger.addStream(listener.getLogger(), false);
    logger.addStream(Paths.get(outputPath, SHIPSHAPE_LOG_FILE_NAME).toString(), true);

    // Logging parameter and path values for inspection.
    if (isVerbose) {
      logger.log("Verbose mode enabled");
      logger.log(String.format("Using workspace path: %s", workspacePath));
      logger.log(String.format("Using output path: %s", outputPath));
      logger.log(String.format("Using categories: %s", Joiner.on(", ").join(categoryList)));
      logger.log(String.format("Using socket: %s", socket));
    }

    getDockerAccessToken(logger);

    ShipshapeRequest req = constructShipshapeRequest(logger, workspace, categoryList, jobName, stage);
    logger.log(String.format("ShipshapeRequest: %s", req));
    ShipshapeResponse res = makeShippingContainerRequest(logger, socket, req, workspacePath,
        outputPath);
    if (isVerbose) {
      logger.log(String.format("ShipshapeResponse: %s", res));
    }
    int numRes = reportShipshapeResults(logger, res, outputPath);
    logger.closeStreams();
    return numRes;
  }
  
  /**
   * Get access token to the docker registry.
   *
   * This token lasts for about an hour.
   *
   * @param logger The logger to use.
   * @throws ShipshapeException If getting the access token fails.
   */
  private void getDockerAccessToken(ShipshapeStreamLogger logger) throws ShipshapeException {
    // Get the access token
    String[] accessCmd = {
      "gcloud", "preview", "docker",
      String.format("--server=%s", REPO_HOST), "--authorize_only"
    };
    runCommand(logger, accessCmd);
  }

  /**
   * Constructs a Shipshape request.
   *
   * @param logger The logger to use.
   * @param workspacePath Path to workspace.
   * @param categoryList The categories to run.
   * @param stage 
   * @throws ShipshapeException If reading of files from local workspace failed.
   */
  private ShipshapeRequest constructShipshapeRequest(ShipshapeStreamLogger logger,
      FilePath workspacePath, List<String> categoryList, String jobName, Stage stage)
      throws ShipshapeException {
    logger.log("Creating Shipshape request ...");
    ShipshapeContext shipshapeContext = ShipshapeContext.newBuilder()
        .setRepoRoot(JENKINS_WORKSPACE_MOUNT_POINT)
        .build();
    return ShipshapeRequest.newBuilder()
        .setShipshapeContext(shipshapeContext)
        .setEvent(jobName)
        .addAllTriggeredCategory(categoryList)
        .setStage(stage)
        .build();
  }

  /**
   * Sends a Shipshape request via the input stream of a 'docker run' process and collects the
   * response from the output stream of the same process.
   *
   * Takes the paths to the output and workspace directories to share as volumes with the docker
   * container.
   *
   * @param logger The logger to use.
   * @param socket The socket to use for docker communication.
   * @param request The Shipshape request.
   * @param outputPath The path to output directory.
   * @param workspacePath The path to the workspace.
   * @return The Shipshape response.
   */
  private ShipshapeResponse sendShipshapeRequestViaStream(ShipshapeStreamLogger logger, String socket,
      ShipshapeRequest request, String outputPath, String workspacePath)
      throws ShipshapeException {
    String[] cmd = new String[] {
        "docker", "-H", socket, "run", "-i", "-a=stdin", "-a=stdout", "-a=stderr",
        String.format("-p=%d:%d", SHIPSHAPE_SERVICE_PORT, SHIPSHAPE_SERVICE_PORT),
        String.format("-v=%s:%s", outputPath, JENKINS_OUTPUT_MOUNT_POINT),
        String.format("-v=%s:%s", workspacePath, JENKINS_WORKSPACE_MOUNT_POINT),
        String.format("--name=%s", SHIPPING_CONTAINER_NAME),
        String.format("%s/%s:%s", REPO, SHIPPING_CONTAINER_IMAGE, SHIPPING_CONTAINER_IMAGE_TAG)};
    logger.log("Sending Shipshape request using cmd: " + Arrays.toString(cmd));
    Process p = null;
    int exitCode = 0;
    String errorMsg = "";
    ByteArrayOutputStream stdout = new ByteArrayOutputStream();
    ByteArrayOutputStream stderr = new ByteArrayOutputStream();
    try {
      p = Runtime.getRuntime().exec(cmd);
      InputStream stdoutStream = p.getInputStream();
      InputStream stderrStream = p.getErrorStream();
      StreamBufferHandler stdoutHandler = new StreamBufferHandler(logger, stdout, stdoutStream);
      StreamBufferHandler stderrHandler = new StreamBufferHandler(logger, stderr, stderrStream);
      stdoutHandler.start();
      stderrHandler.start();

      // Send request via stdin
      BufferedOutputStream stdin = new BufferedOutputStream(p.getOutputStream());
      byte[] stdinData = request.toByteArray();
      stdin.write(stdinData, 0, stdinData.length);
      stdin.flush();
      stdin.close();
      exitCode = p.waitFor();
      stdoutHandler.join();
      stderrHandler.join();

      if (exitCode == 0) {
        ShipshapeResponse response = ShipshapeResponse.parseFrom(stdout.toByteArray());
        logger.log("Shipshape response received");
        if (isVerbose) {
          logger.log(String.format("\n\tstderr:\n%s\n", stderr.toString()));
        }
        return response;
      }
    } catch (InvalidProtocolBufferException e) {
      logger.log(String.format("Failed to parse proto response, error: %s", e.getMessage()));
      errorMsg = e.getMessage();
    } catch (Throwable e) {
      logger.log(String.format("Sending Shipshape request failed: %s", e.getMessage()));
      p.destroy();
      errorMsg = e.getMessage();
    }
    throw new ShipshapeException(String.format(
       "Sending Shipshape request failed with exit code: %d, error: %s, stderr: %s",
       exitCode, errorMsg, stderr.toString()));
  }

  /**
   * Connects to shipping_container, sends request and returns the received response.
   *
   * This process includes the following steps:
   * 1) Pull down latest shipping_container image.
   * 2) Remove any running or stopped shipping_container container.
   * 3) Start shipping_container passing the request in via stdin.
   * 4) Collect the response via stdout.
   *
   * @param logger The logger to use.
   * @param request The shipshape request.
   * @param workspacePath Absolute workspace directory path.
   * @param outputPath Absolute build directory path.
   * @param socket The socket to use for docker communication.
   * @return The Shipshape response.
   * @throws ShipshapeException If the container fails to start.
   */
  private ShipshapeResponse makeShippingContainerRequest(
      final ShipshapeStreamLogger logger, final String socket,
      final ShipshapeRequest request, final String workspacePath, final String outputPath)
      throws ShipshapeException {
    // 1. Pull down latest version of shipping_container.
    // Instructs the local docker registry to get the most
    // recent version of shipping_container uploaded to the GCS bucket.
    String[] cmd = {
      "docker", "-H", socket, "pull",
      String.format("%s/%s:%s", REPO, SHIPPING_CONTAINER_IMAGE, SHIPPING_CONTAINER_IMAGE_TAG)};
    runCommand(logger, cmd);
    // 2., 3. and 4.: Remove running container, start container and send Shipshape request
    // via streams and collect result as Shipshape response
    class MakeRequestTask implements RetryTask {
      public ShipshapeResponse response;
      @Override
      public String desc() {
        return "make Shipshape request";
      }
      @Override
      public boolean run() throws Exception {
        if (isDockerContainerInProcessList(logger, socket, SHIPPING_CONTAINER_NAME)) {
          stopDockerContainer(logger, socket, SHIPPING_CONTAINER_NAME);
        }
        response = sendShipshapeRequestViaStream(logger, socket, request, outputPath,
            workspacePath);
        return true;
      }
    }
    MakeRequestTask task = new MakeRequestTask();
    if (!runTaskWithRetry(logger, task)) {
      throw new ShipshapeException("Failed to make Shipshape request");
    }
    return task.response;
  }

  /**
   * Reports a Shipshape analyzer response via the build logger and the analysis log file.
   *
   * @param logger The logger to use.
   * @param res The Shipshape response.
   * @param outputPath Absolute build directory path.
   * @throws ShipshapeException If writing to the result file fails.
   */
  private int reportShipshapeResults(ShipshapeStreamLogger logger, ShipshapeResponse res,
      String outputPath) throws ShipshapeException {
    try {
      PrintWriter writer = new PrintWriter(outputPath + "/" + RESULT_FILE_NAME, FILE_ENCODING);
      int nbrAnalysisResults = 0;
      int nbrAnalysisFailures = 0;
      int actionableResults = 0;
      StringBuffer resultBuffer = new StringBuffer();
      StringBuffer failureBuffer = new StringBuffer();
      for (AnalyzeResponse analyzeRes : res.getAnalyzeResponseList()) {
        for (Note note : analyzeRes.getNoteList()) {
          String line = "", col = "";

          if (note.getLocation().hasRange()) {
            if (note.getLocation().getRange().hasStartLine()) {
              line = ":" + note.getLocation().getRange().getStartLine();
            }
            if (note.getLocation().getRange().hasStartColumn()) {
              col = ":" + note.getLocation().getRange().getStartColumn();
            }
          }

          String noteMsg = String.format(
              "%s%s\n\t%s%s%s\n\t%s",
              note.getCategory(), (note.hasSubcategory() ? ":" + note.getSubcategory() : ""),
              note.getLocation().getPath(), line, col,
              note.getDescription());
          resultBuffer.append(noteMsg + "\n");
          nbrAnalysisResults++;
          if (note.getSeverity() != Note.Severity.OTHER) {
            actionableResults++;
          }
        }
        for (AnalysisFailure failure : analyzeRes.getFailureList()) {
          String failureMsg =
              String.format("Analyzer %s failed to run: %s",
                failure.getCategory(), failure.getFailureMessage());
          failureBuffer.append(failureMsg + "\n");
          nbrAnalysisFailures++;
        }
      }
      String resultSummary = String.format("Shipshape returned %d analysis result(s)",
          nbrAnalysisResults);
      if (nbrAnalysisResults > 0) {
        logger.log(resultSummary);
        logger.log(resultBuffer.toString());
        writer.println(resultSummary);
        writer.println(resultBuffer.toString());
      }
      if (nbrAnalysisFailures > 0) {
        String failureSummary = "Some analyses had problems: ";
        logger.log(failureSummary);
        logger.log(failureBuffer.toString());
        writer.println(failureSummary);
        writer.println(failureBuffer.toString());
      }
      writer.close();
      return actionableResults + nbrAnalysisFailures;
    } catch (IOException e)  {
      throw new ShipshapeException(String.format(
          "Failed to write to result file, output path: %s, result file: %s, error: %s",
          outputPath, RESULT_FILE_NAME, e.getMessage()));
    }
  }
  
  /**
   * Stops a docker container.
   *
   * @param logger The logger to use.
   * @param containerName The name of the container to stop.
   * @throws ShipshapeException If stopping the container fails.
   */
  private void stopDockerContainer(ShipshapeStreamLogger logger, String socket, String containerName)
      throws ShipshapeException {
    String[] cmd = {"docker", "-H", socket, "rm", "-f", containerName};
    runCommand(logger, cmd);
  }

  /**
   * Checks if a docker container is listed in docker process list.
   *
   * @param logger The logger to use.
   * @param socket The socket to use for docker communication.
   * @param containerName The name of the container to check for.
   * @throws ShipshapeException If the check fails.
   * @return true if the container is in process list, otherwise false.
   */
  private boolean isDockerContainerInProcessList(ShipshapeStreamLogger logger, String socket,
      String containerName) throws ShipshapeException {
    // Use the REST API if a TCP socket is used to communicate with docker
    if (socket.startsWith("tcp")) {
      String containersURL = String.format("http://localhost:%d/%s", DOCKER_TCP_PORT,
            DOCKER_REST_API_ALL_CONTAINERS);
      logger.log(String.format("Checking running containers via %s ..", containersURL));
      try {
        URL url = new URL(containersURL);
        HttpURLConnection conn = (HttpURLConnection) url.openConnection();
        int httpCode = conn.getResponseCode();
        if (httpCode == HttpURLConnection.HTTP_OK) {
          StringBuffer buffer = new StringBuffer();
          InputStream stream = conn.getInputStream();
          int next;
          while ((next = stream.read()) != -1) {
            buffer.append((char) next);
          }
          stream.close();
          return buffer.toString().contains(containerName);
        } else {
          throw new ShipshapeException(String.format(
              "Failed to connect to docker with URL connection (code=%d)", httpCode));
        }
      } catch (MalformedURLException e) {
        throw new ShipshapeException(String.format(
            "Failed to check if docker container (%s) is running using URL: %s, error: %s",
              SHIPPING_CONTAINER_NAME, containersURL, e.getMessage()));
      } catch (IOException e) {
        throw new ShipshapeException(String.format(
            "Failed to check if docker container (%s) is running: %s",
              SHIPPING_CONTAINER_NAME, e.getMessage()));
      }
    } else {
      // Assuming unix socket and using a docker command
      String[] cmd = {"docker", "-H", socket, "ps", "-a"};
      return runCommand(logger, cmd).contains(containerName);
    }
  }
  
  
  
  // TODO(emso): Refactor runCommand etc. to utility class?

  /**
   * Runs a shell command.
   *
   * @param logger The logger to use.
   * @param cmd The command to run.
   * @return content of stdout.
   * @throws ShipshapeException If the current thread is interrupted.
   */
  private String runCommand(ShipshapeStreamLogger logger, String[] cmd)
      throws ShipshapeException {
    if (isVerbose) {
      logger.log("Running cmd: " + Arrays.toString(cmd));
    }
    Process p = null;
    int exitCode = 0;
    String errorMsg = "";
    ByteArrayOutputStream stdout = new ByteArrayOutputStream();
    ByteArrayOutputStream stderr = new ByteArrayOutputStream();
    try {
      p = Runtime.getRuntime().exec(cmd, null, null);
      InputStream stdoutStream = p.getInputStream();
      InputStream stderrStream = p.getErrorStream();
      StreamBufferHandler stdoutHandler =
          new StreamBufferHandler(logger, stdout, stdoutStream);
      StreamBufferHandler stderrHandler =
          new StreamBufferHandler(logger, stderr, stderrStream);
      stdoutHandler.start();
      stderrHandler.start();
      exitCode = p.waitFor();
      stdoutHandler.join();
      stderrHandler.join();
    } catch (Throwable e) {
      logger.log(String.format("Running command failed: %s", e.getMessage()));
      p.destroy();
      errorMsg = e.getMessage();
    }
    if (exitCode != 0) {
      throw new ShipshapeException(String.format(
         "Command failed with exit code: %d, error: %s, stdout: %s, stderr: %s",
         exitCode, errorMsg, stdout.toString(), stderr.toString()));
    } else if (isVerbose) {
      logger.log(String.format("Finished executing cmd\n\tstdout:\n%s\n\tstderr:\n%s\n",
            stdout.toString().trim(), stderr.toString().trim()));
    }
    return stdout.toString();
  }

  // TODO(emso): Refactor out runTaskWithRetry and RetryTask into separate file.

  /**
   * Runs a  retry task.
   *
   * @param logger The logger to use.
   * @param task The task to retry on failure (i.e., task returns false or throws exception).
   * @return true if retry was successful.
   */
  private boolean runTaskWithRetry(ShipshapeStreamLogger logger, RetryTask task) {
    boolean success = false;
    long delay = RETRY_INIT_DELAY_MS;
    for (int i = 1; i <= RETRY_MAX_ATTEMPTS; i++) {
      maybeLog(logger, String.format("Running task [%s] (attempt:%d) ...", task.desc(), i));
      try {
        if (task.run()) {
          maybeLog(logger, String.format("Task [%s] succeeded.", task.desc()));
          success = true;
          break;
        }
        maybeLog(logger, String.format("Task [%s] failed with return false", task.desc()));
      } catch (Exception e) {
        maybeLog(logger, String.format("Task [%s] failed with exception: %s", task.desc(),
              e.getMessage()));
      }
      try {
        maybeLog(logger, String.format("Retrying task [%s] in %d ms ...", task.desc(), delay));
        Thread.sleep(delay);
        delay *= RETRY_DELAY_MULTIPLIER;
      } catch (InterruptedException e) {
        maybeLog(logger, String.format("Task [%s] was interrupted: %s", task.desc(),
              e.getMessage()));
      }
    }
    return success;
  }

  // TODO(ciera): Instead of using "maybeLog" and verbose checks everwhere, just
  // set these to log level fine and have verbose print out log level FINE.
  private void maybeLog(ShipshapeStreamLogger logger, String message) {
    if (isVerbose) {
      logger.log(message);
    }
  }

  /**
   * Utility interface for a task that should be retried on failure.
   */
  private interface RetryTask {
    /**
     * @return The description of the task.
     */
    public String desc();
    /**
     * @return False if the task fails.
     * @throws Exception If the task fails.
     */
    public boolean run() throws Exception;
  }

}
