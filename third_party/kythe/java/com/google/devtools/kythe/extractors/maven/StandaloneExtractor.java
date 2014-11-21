package com.google.devtools.kythe.extractors.maven;

import com.google.devtools.kythe.extractors.shared.CompilationDescription;
import com.google.devtools.kythe.extractors.shared.IndexInfoUtils;

import com.beust.jcommander.JCommander;
import com.beust.jcommander.Parameter;

import java.io.File;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;

/** Standalone {@link CompilationDescription} extractor for Maven project. */
public class StandaloneExtractor {

  @Parameter(names = "--print", arity = 0,
      description = "Print CompilationUnit to standard output")
  private boolean printCompilation = false;

  @Parameter(names = "--help", help = true, description = "Print usage information")
  private boolean help;

  @Parameter(description = "<project-root> [out-root]")
  private List<String> positionalArgs = new ArrayList<String>();

  private final String repositoryPath;
  private final String outputRoot;

  public StandaloneExtractor(String[] args) {
    JCommander cmder = cmder = new JCommander(this, args);

    if (help) {
      usage(cmder, null);
    } else if (positionalArgs.size() == 0) {
      usage(cmder, "Missing <project-root> directory!");
    } else if (positionalArgs.size() > 2) {
      usage(cmder, "Too many positional arguments: " + positionalArgs);
    }

    repositoryPath = positionalArgs.get(0);
    outputRoot = positionalArgs.size() == 2 ? positionalArgs.get(1) : repositoryPath;
  }

  public void run() throws Throwable {
    File pom = Paths.get(repositoryPath, "pom.xml").toFile();

    // TODO(schroederc): Deal with non-default artifact repositories
    ArtifactRepository localRepo = ArtifactRepository.newLocalRepository();

    Project project = new Project(localRepo, pom);
    CompilationDescription description =
        new CompilationDescriptionExtractor(project, localRepo, Paths.get(repositoryPath))
        .extract();

    String indexFile = IndexInfoUtils.getIndexFilename(outputRoot,
        description.getCompilationUnit().getVName().getCorpus());

    if (printCompilation) {
      System.out.println("CompilationUnit:");
      System.out.println(description.getCompilationUnit());
    }

    System.out.printf("Writing compilation to:\n  %s\n", indexFile);
    IndexInfoUtils.writeIndexInfoToFile(description, indexFile);
  }

  public static void main(String[] args) throws Throwable {
    new StandaloneExtractor(args).run();
  }

  private static void usage(JCommander cmder, String msg) {
    if (msg != null) {
      System.err.println(msg);
    }
    cmder.usage();
    System.exit(1);
  }
}
