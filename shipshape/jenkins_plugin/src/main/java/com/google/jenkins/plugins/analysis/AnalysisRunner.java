package com.google.jenkins.plugins.analysis;

import com.google.common.base.Joiner;
import com.google.common.base.Splitter;
import com.google.common.collect.Lists;
import com.google.devtools.source.v1.SourceContextProto.CloudRepoSourceContext;
import com.google.devtools.source.v1.SourceContextProto.GerritSourceContext;
import com.google.devtools.source.v1.SourceContextProto.ProjectRepoId;
import com.google.devtools.source.v1.SourceContextProto.RepoId;
import com.google.devtools.source.v1.SourceContextProto.SourceContext;
import com.google.protobuf.InvalidProtocolBufferException;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalysisFailure;
import com.google.shipshape.proto.ShipshapeRpcProto.AnalyzeResponse;
import com.google.shipshape.proto.ShipshapeRpcProto.ShipshapeRequest;
import com.google.shipshape.proto.ShipshapeRpcProto.ShipshapeResponse;

import hudson.Extension;
import hudson.FilePath;
import hudson.Launcher;
import hudson.model.AbstractBuild;
import hudson.model.AbstractProject;
import hudson.model.BuildListener;
import hudson.remoting.Callable;
import hudson.tasks.BuildStepDescriptor;
import hudson.tasks.Builder;

import jenkins.model.Jenkins;

import org.kohsuke.stapler.DataBoundConstructor;

import java.io.BufferedOutputStream;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.PrintStream;
import java.io.PrintWriter;
import java.io.Serializable;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.URL;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.List;

/**
 * A {@link Builder} for Shipshape.
 *
 * Takes parameters to determine project, revision, categories, host and socket (for docker).
 *
 * <p>
 * When a build is performed, the
 * {@link #perform(AbstractBuild, Launcher, BuildListener)} method will be
 * invoked.
 * </p>
 */
public class AnalysisRunner extends Builder implements Serializable {

  private static final long serialVersionUID = 1L;

  // Repo
  private static final String REPO_HOST = "container.cloud.google.com";
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

  private static final String DEFAULT_EVENT_NAME = "build";

  // The name of the output file, with a blub of analyzer output.
  private static final String RESULT_FILE_NAME = "analysis-log.txt";
  // The name of the shipshape log file. This file name may be used in integration tests.
  private static final String SHIPSHAPE_LOG_FILE_NAME = "shipshape-log.txt";
  // Encoding used for output files.
  private static final String FILE_ENCODING = "UTF-8";

  // Default parameter values going into the shipshape context when no other values
  // were given for the plugin.
  private static final String DEFAULT_REVISION = "master";

  // Default socket value used for docker communication. This is used as the default for
  // the socket plugin parameter when no other value is given.
  // For containers running inside a slave container this is the socket used (there is typically no
  // TCP socket available inside slave containers, but instead the plugin runs as root)
  // and no other value should be given.
  private static final String DEFAULT_DOCKER_SOCKET = "unix:///var/run/docker.sock";

  // Configuration used when retrying tasks, such as 'connecting to the docker registry' or
  // 'sending a request to ShippingContainer'.
  private static final int RETRY_MAX_ATTEMPTS = 10;
  private static final long RETRY_INIT_DELAY_MS = 20;
  private static final int RETRY_DELAY_MULTIPLIER = 3;

  // The port on localhost connecting to the local docker registry. This value is used to bring up
  // the registry and to communicate with it. Has not dependency to anything outside of the plugin.
  private static final int DOCKER_REGISTRY_PORT = 5000;

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
  // Pings the docker server and returns status code 200 on success.
  private static final String DOCKER_REST_API_PING = "v1/_ping";
  // Lists all containers (including non-running ones).
  private static final String DOCKER_REST_API_ALL_CONTAINERS = "containers/json?all=1";

  // Configure how we split up a comma-separated list of Shipshape categories.
  private static final Splitter CATEGORY_PARSER = Splitter.on(",").trimResults().omitEmptyStrings();

  // Plugin parameters (needs to be public final):
  // Comma-separated list of categories to run.
  public final String categories;
  // The host of the project if the repo type supports that.
  public final String host;
  // The name of the project in GCE.
  public final String project;
  // The revision in the repository.
  public final String revision;
  // Whether to use verbose output
  public final boolean verbose;
  // Custom docker socket, e.g., tcp://localhost:4243.
  public final String socket;

  /**
   * Fields in config.jelly must match the parameter names in the
   * "DataBoundConstructor" and public final fields (above).
   *
   * @param categories the categories to run on.
   * @param host the host of the repo.
   * @param project the name of the GCE project.
   * @param revision the revision in the repository.
   * @param verbose whether to use verbose output
   * @param socket use the TCP socket to communicate with docker.
   */
  @DataBoundConstructor
  public AnalysisRunner(
      final String categories,
      final String host,
      final String project,
      final String revision,
      final boolean verbose,
      final String socket
      ) {
    this.host = host;
    this.project = project;
    this.revision = revision;
    this.categories = categories;
    this.verbose = verbose;
    this.socket = socket;
  }

  /**
   * Perform this Shipshape operation.
   *
   * @param build The build
   * @param launcher Launcher
   * @param listener Listener
   * @return True if it worked, else false
   * @throws IOException Can be thrown if can't read files
   * @throws InterruptedException Can be thrown if build interrupted
   */
  @Override
  public final boolean perform(
      final AbstractBuild build,
      final Launcher launcher,
      final BuildListener listener
      ) throws InterruptedException, IOException {

    // Serializable values (strings, primitive types ...) needed on the Jenkins slave should be
    // moved to final variables here to be used in the anonymous Callable class below.
    final String host = this.host;
    final String project = (this.project == null || this.project.trim().equals(""))
        ? "" : this.project;
    final String revision = (this.revision == null || this.revision.trim().equals(""))
        ? DEFAULT_REVISION : this.revision;
    final String socket = (this.socket == null || this.socket.trim().equals(""))
        ? DEFAULT_DOCKER_SOCKET : this.socket;
    final List<String> categoryList = Lists.newArrayList(
        CATEGORY_PARSER.split(this.categories == null ? "" : this.categories));

    // TODO(emso): Add fail-fast check for .shipshape file.

    final FilePath workspace = build.getWorkspace();
    int actionableResults = 0;
    try {
      actionableResults = workspace.act(
        new Callable<Integer, Exception>() {
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
            // here to enable logging on the Jenkins slave.
            ShipshapeLogger logger = ShipshapeLogger.create(listener);
            logger.addStream(Paths.get(outputPath, SHIPSHAPE_LOG_FILE_NAME).toString(), true);

            // Logging parameter and path values for inspection.
            if (verbose) {
              logger.log("Verbose mode enabled");
              logger.log(String.format("Using workspace path: %s", workspacePath));
              logger.log(String.format("Using output path: %s", outputPath));
              logger.log(String.format("Using categories: %s", Joiner.on(", ").join(categoryList)));
              logger.log(String.format("Using host: %s", host));
              logger.log(String.format("Using project: %s", project));
              logger.log(String.format("Using revision: %s", revision));
              logger.log(String.format("Using socket: %s", socket));
            }

            getDockerAccessToken(logger);

            ShipshapeRequest req = constructShipshapeRequest(logger, workspace, categoryList,
                host, project, revision);
            logger.log(String.format("ShipshapeRequest: %s", req));
            ShipshapeResponse res = makeShippingContainerRequest(logger, socket, req, workspacePath,
                outputPath);
            if (verbose) {
              logger.log(String.format("ShipshapeResponse: %s", res));
            }
            int numRes = reportShipshapeResults(logger, res, outputPath);
            logger.closeStreams();
            return numRes;
          }
        });
    } catch (Exception e) {
      listener.getLogger().println(String.format("[Shipshape] Error: %s", e.getMessage()));
      e.printStackTrace(listener.getLogger());
      return false;
    }

    listener.getLogger().println("[Shipshape] Done");

    // TODO(ciera): use a configurable setting in the plugin to determine whether to fail
    // and on what kind of results.
    return actionableResults == 0;
  }

  /**
   * Get access token to the docker registry.
   *
   * This token lasts for about an hour.
   *
   * @param logger The logger to use.
   * @throws ShipshapeException If getting the access token fails.
   */
  private void getDockerAccessToken(ShipshapeLogger logger) throws ShipshapeException {
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
   * @param host The host in case of a gerrit change.
   * @param project The project of the repository.
   * @param revision The revision in the repository.
   * @throws ShipshapeException If reading of files from local workspace failed.
   */
  private ShipshapeRequest constructShipshapeRequest(ShipshapeLogger logger,
      FilePath workspacePath, List<String> categoryList, String host, String project,
      String revision) throws ShipshapeException {
    logger.log("Creating Shipshape request ...");
    SourceContext sourceContext = null;
    if (!(host == null || host.equals(""))) {
      sourceContext = SourceContext.newBuilder().setGerrit(
          GerritSourceContext.newBuilder()
              .setHost(host).setProject(project).setRevisionId(revision))
          .build();
    } else { // else assuming cloud repo
      sourceContext = sourceContext.newBuilder().setCloudRepo(CloudRepoSourceContext.newBuilder()
          .setRevisionId(revision).setRepoId(RepoId.newBuilder()
            .setProjectRepoId(ProjectRepoId.newBuilder().setProjectId(project))))
      .build();
    }
    ShipshapeContext shipshapeContext = ShipshapeContext.newBuilder()
        .setSourceContext(sourceContext)
        .setRepoRoot(JENKINS_WORKSPACE_MOUNT_POINT)
        .build();
    // TODO(emso): Find out and use the name of the job as the event name, for instance,
    // this could be 'canary', 'staging', or similar in a multistage pipeline.
    return ShipshapeRequest.newBuilder()
        .setShipshapeContext(shipshapeContext)
        .setEvent(DEFAULT_EVENT_NAME)
        .addAllTriggeredCategory(categoryList)
        .build();
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
  private ShipshapeResponse makeShippingContainerRequest(final ShipshapeLogger logger, final String socket,
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
      public String desc() {
        return "make Shipshape request";
      }
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
  private int reportShipshapeResults(ShipshapeLogger logger, ShipshapeResponse res,
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
   * Tears down the shipping_container container.
   *
   * @param logger The logger to use.
   * @throws ShipshapeException If tear down of shipping_container fails.
   */
  private void tearDownShippingContainer(ShipshapeLogger logger, String socket) throws ShipshapeException {
    stopDockerContainer(logger, socket, SHIPPING_CONTAINER_NAME);
    logger.log("Tore down shipping_container image");
  }

  // TODO(emso): Refactor out runTaskWithRetry and RetryTask into separate file.

  /**
   * Runs a  retry task.
   *
   * @param logger The logger to use.
   * @param task The task to retry on failure (i.e., task returns false or throws exception).
   * @return true if retry was successful.
   */
  private boolean runTaskWithRetry(ShipshapeLogger logger, RetryTask task) {
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
  private void maybeLog(ShipshapeLogger logger, String message) {
    if (verbose) {
      logger.log(message);
    }
  }

  /**
   * Stops a docker container.
   *
   * @param logger The logger to use.
   * @param containerName The name of the container to stop.
   * @throws ShipshapeException If stopping the container fails.
   */
  private void stopDockerContainer(ShipshapeLogger logger, String socket, String containerName)
      throws ShipshapeException {
    String[] cmd = {"docker", "-H", socket, "rm", "-f", containerName};
    runCommand(logger, cmd);
  }

  /**
   * Checks if the docker registry is running.
   *
   * @param logger The logger to use.
   * @throws ShipshapeException If the check fails.
   * @return true if the container is running, otherwise false.
   */
  private boolean isDockerRegistryRunning(ShipshapeLogger logger) throws ShipshapeException {
    String pingURL = String.format("http://localhost:%d/%s", DOCKER_REGISTRY_PORT,
        DOCKER_REST_API_PING);
    try {
      URL url = new URL(pingURL);
      return ((HttpURLConnection)url.openConnection()).getResponseCode()
        == HttpURLConnection.HTTP_OK;
    } catch (MalformedURLException e) {
      logger.log(String.format("Failed to connect to docker registry, url: %s, error: %s",
            pingURL, e.getMessage()));
      return false;
    } catch (IOException e) {
      logger.log(String.format("Failed to read from docker registry connection: %s",
            e.getMessage()));
      return false;
    }
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
  private boolean isDockerContainerInProcessList(ShipshapeLogger logger, String socket,
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
  private String runCommand(ShipshapeLogger logger, String[] cmd)
      throws ShipshapeException {
    if (verbose) {
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
    } else if (verbose) {
      logger.log(String.format("Finished executing cmd\n\tstdout:\n%s\n\tstderr:\n%s\n",
            stdout.toString().trim(), stderr.toString().trim()));
    }
    return stdout.toString();
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
  private ShipshapeResponse sendShipshapeRequestViaStream(ShipshapeLogger logger, String socket,
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

      ShipshapeResponse response = ShipshapeResponse.parseFrom(stdout.toByteArray());
      logger.log("Shipshape response received");
      if (verbose) {
        logger.log(String.format("\n\tstderr:\n%s\n", stderr.toString()));
      }
      return response;
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
   * Exception for ShippingContainer problems.
   */
  private class ShipshapeException extends Exception {
    public ShipshapeException(String message) {
      super(message);
    }
  }

  /**
   * Utility class for moving data from a stream to a buffer.
   */
  private class StreamBufferHandler extends Thread {
    private InputStream stream;
    private ByteArrayOutputStream buffer;
    private ShipshapeLogger logger;
    /**
     * Creates a handler moving data from a stream to a buffer.
     * @param logger The logger to use.
     * @param buffer Buffer to put input data in.
     * @param stream The stream to read from.
     */
    public StreamBufferHandler(ShipshapeLogger logger, ByteArrayOutputStream buffer,
        InputStream stream) {
      this.logger = logger;
      this.stream = stream;
      this.buffer = buffer;
    }
    public void run() {
      try {
        int next;
        while ((next = stream.read()) != -1) {
          buffer.write(next);
        }
      } catch (IOException e) {
        logger.log(String.format("Problem reading from stream, error: %s, data read so far: %s",
              e.getMessage(), buffer.toString()));
      }
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

  /**
   * Utility logger class outputting to one or more streams.
   */
  private static class ShipshapeLogger {
    private List<PrintStream> streamList;
    /**
     * Creates a logger handling outputting to multiple streams.
     */
    private ShipshapeLogger() {
      streamList = new ArrayList<PrintStream>();
    }
    /**
     * Creates Shipshape logger.
     *
     * @param listener The build listener used to log to the Jenkins console.
     * @return The logger.
     */
    private static ShipshapeLogger create(BuildListener listener) {
      ShipshapeLogger logger = new ShipshapeLogger();
      logger.streamList.add(listener.getLogger());
      return logger;
    }
    /**
     * Adds stream to logger.
     * @param filePath The path to the file to create stream to.
     * @param append Whether to append to the file stream.
     * @throws IOException If creation of stream fails.
     */
    public void addStream(String filePath, boolean append) throws IOException {
      File file = new File(filePath);
      file.createNewFile();
      streamList.add(new PrintStream(new FileOutputStream(file), append));
    }
    /**
     * Closes all streams, except the build listener print stream.
     * @throws IOException if close failed for a stream.
     */
    public void closeStreams() throws IOException {
      // Iterating over all streams except the first one which is the build listener logger stream,
      // which we don't want close.
      for (int i = 1; i < streamList.size(); i++) {
        streamList.get(i).close();
      }
    }
    /**
     * Flushes all streams.
     * @throws IOException if flush failed for a stream.
     */
    public void flushStreams() throws IOException {
      for (PrintStream stream : streamList) {
        stream.flush();
      }
    }
    /**
     * Logs a message to available streams.
     * @param msg The message to log.
     */
    public void log(String msg) {
      for (PrintStream stream : streamList) {
        stream.println(String.format("[Shipshape] %s", msg));
      }
    }
    /**
     * Logs the message of an exception and then its stack trace.
     * @param e The exception to log.
     */
    public void logWithTrace(Exception e) {
      for (PrintStream stream : streamList) {
        stream.println(String.format("[Shipshape] %s", e.getMessage()));
        e.printStackTrace(stream);
      }
    }
  }

  /**
   * Implementation of an extension point. Descriptor for this class.
   */
  @Extension
  public static final class DescriptorImpl extends BuildStepDescriptor<Builder> {

    /**
     * @param aClass Project to check whether this plugin can be used with.
     * @return Whether this descriptor is applicable - always true
     */
    public boolean isApplicable(final Class<? extends AbstractProject> aClass) {
      return true;
    }

    /**
     * @return This name is used in the configuration screen.
     */
    public String getDisplayName() {
      return "Google Analysis Plugin";
    }
  }
}
