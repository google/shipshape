package test;

import com.google.devtools.source.v1.SourceContextProto.CloudRepoSourceContext;
import com.google.devtools.source.v1.SourceContextProto.ProjectRepoId;
import com.google.devtools.source.v1.SourceContextProto.RepoId;
import com.google.devtools.source.v1.SourceContextProto.SourceContext;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.proto.ShipshapeRpcProto.ShipshapeRequest;
import com.google.shipshape.proto.ShipshapeRpcProto.ShipshapeResponse;

import java.io.BufferedOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.util.Arrays;
import java.util.List;

public class TestRequestViaStream {

  public static void main(String[] args) {

    if (args.length != 1) {
      System.err.println("usage: java TestRequestViaStream [absolute path to shipshape binary]");
      System.exit(1);
    }

    String cmd = args[0];

    ShipshapeRequest req = constructShipshapeRequest();
    log(String.format("Request to be sent: %s", req));
    byte[] input = req.toByteArray();

    log("Running cmd: " + cmd);

    Process p = null;
    int exitCode = 0;
    String errorMsg = "";
    StringBuffer stdout = new StringBuffer();
    StringBuffer stderr = new StringBuffer();
    try {
      p = Runtime.getRuntime().exec(cmd);
      InputStream stdoutStream = p.getInputStream();
      InputStream stderrStream = p.getErrorStream();
      StreamBufferHandler stdoutHandler = new StreamBufferHandler(stdout, stdoutStream, false);
      StreamBufferHandler stderrHandler = new StreamBufferHandler(stderr, stderrStream, true);
      stdoutHandler.start();
      stderrHandler.start();

      if (input != null) {
        BufferedOutputStream stdin = new BufferedOutputStream(p.getOutputStream());
        stdin.write(input, 0, input.length);
        stdin.flush();
        stdin.close();
      }

      exitCode = p.waitFor();
      log("Process finished");

      stdoutStream.close();
      stderrStream.close();

      ShipshapeResponse response = ShipshapeResponse.parseFrom(stdout.toString().getBytes());
      log(String.format("Response received: %s", response));

    } catch (Throwable e) {
      log(String.format("Running command failed: %s", e.getMessage()));
      p.destroy();
      errorMsg = e.getMessage();
    }
    if (exitCode != 0 || !stderr.toString().isEmpty()) {
      log(String.format("Command failed with exit code: %d, error: %s, stdout: %s, stderr: %s",
         exitCode, errorMsg, stdout.toString(), stderr.toString()));
    } else {
      log(String.format("Finished executing cmd, exit code: %d, stdout: %s, stderr: %s",
            exitCode, stdout.toString(), stderr.toString()));
    }
  }

  private static ShipshapeRequest constructShipshapeRequest() {
    List<String> categoryList = Arrays.asList("".split(","));
    SourceContext sourceContext = SourceContext.newBuilder()
        .setCloudRepo(CloudRepoSourceContext.newBuilder()
        .setRevisionId("master").setRepoId(RepoId.newBuilder()
          .setProjectRepoId(ProjectRepoId.newBuilder().setProjectId("quixotic-treat-519"))))
      .build();
    ShipshapeContext shipshapeContext = ShipshapeContext.newBuilder()
        .setSourceContext(sourceContext)
        .setRepoRoot("/tmp")
        .build();
    return ShipshapeRequest.newBuilder()
        .setShipshapeContext(shipshapeContext).setEvent("event")
        .addAllTriggeredCategory(categoryList).build();
  }

  private static void log(String msg) {
    System.out.println(msg);
  }

  private static class StreamBufferHandler extends Thread {
    private InputStream stream;
    private StringBuffer buffer;
    private boolean output;
    public StreamBufferHandler(StringBuffer buffer, InputStream stream, boolean output) {
      this.stream = stream;
      this.buffer = buffer;
      this.output = output;
    }
    public void run() {
      try {
        int next;
        while ((next = stream.read()) != -1) {
          if (output) {
            System.out.print((char) next);
          }
          buffer.append((char) next);
        }
      } catch (IOException e) {
        log(String.format("Problem reading from command stream, error: %s, data read so far: %s",
              e.getMessage(), buffer.toString()));
      }
    }
  }
}
