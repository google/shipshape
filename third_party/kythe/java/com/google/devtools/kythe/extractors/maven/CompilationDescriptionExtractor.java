package com.google.devtools.kythe.extractors.maven;

import com.google.devtools.kythe.common.FormattingLogger;
import com.google.devtools.kythe.extractors.java.JavaCompilationUnitExtractor;
import com.google.devtools.kythe.extractors.shared.CompilationDescription;
import com.google.devtools.kythe.extractors.shared.ExtractionException;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;

import org.apache.maven.model.Dependency;

import java.io.File;
import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.ArrayList;
import java.util.List;

/**
 * This class implements an object that will extract {@link CompilationDescription} instances from a
 * Maven compilation.
 **/
public class CompilationDescriptionExtractor {
  private static final FormattingLogger logger = FormattingLogger.getLogger(
      CompilationDescriptionExtractor.class);

  private final Project project;
  private final ArtifactRepository repo;
  private final Path repositoryRoot;

  /**
   * Initializes the extractor with the {@link Projec} from which to extract the description and the
   * {@link ArtifactRepository} to use to resolve the dependencies.
   **/
  public CompilationDescriptionExtractor(Project project, ArtifactRepository repo,
      Path repositoryRoot) {
    this.project = project;
    this.repo = repo;
    this.repositoryRoot = repositoryRoot;
  }

  /**
   * @return the {@link Project} used to extract the information.
   **/
  public Project getProject() { return project; }

  /**
   * @return the {@link ArtifactRepository} used to resolve the dependencies.
   **/
  public ArtifactRepository getRepository() { return repo; }

  /**
   * Extracts a {@link CompilationDescription} from the {@link Project}
   **/
  public CompilationDescription extract() throws CompilationDescriptionExtractorException {
    try {
      List<String> sources = new ArrayList<String>();
      processSourceRoot(project.getSourceDirectory(), sources);

      List<String> jars = new ArrayList<String>();
      for (Dependency dep : project.getDependencies()) {
        processDependency(dep, jars);
      }

      return makeCompilationDescription(sources, jars);
    } catch (ExtractionException | IOException ex) {
      throw new CompilationDescriptionExtractorException("Failed to extract compilation unit", ex);
    }
  }

  private void processSourceRoot(String root, List<String> sources) {
    File rootDirectory = new File(root);
    Path rootPath = Paths.get(root);
    if (!rootDirectory.isDirectory()) {
      return;
    }
    processFileList(rootPath, rootDirectory.listFiles(), sources);
  }

  private void processFileList(Path rootPath, File[] files, List<String> sources) {
    for (File f : files) {
      if (f.isDirectory()) {
        processFileList(rootPath, f.listFiles(), sources);
      } else if (f.isFile() && isJavaFile(f)) {
        logger.infofmt("Adding source file: %s", f.getAbsolutePath());
        sources.add(f.getAbsolutePath());
      }
    }
  }

  private boolean isJavaFile(File f) {
    String name = f.getName();
    return name.endsWith(".java");
  }

  // Retrieves the dependency from the artifact repository.
  private void processDependency(Dependency dependency, List<String> jars)
      throws CompilationDescriptionExtractorException {
    logger.infofmt("Processing dependency GroupId: %s, ArtifactId: %s, Version: %s Scope: %s",
        dependency.getGroupId(), dependency.getArtifactId(),
        dependency.getVersion(), dependency.getScope());
    if (dependency.getScope() != "compile" && dependency.getScope() != "provided") {
      return;
    }

    String dependencyURL = repo.getJarURL(dependency.getGroupId(), dependency.getArtifactId(),
        dependency.getVersion());
    try {
      File dependencyFile = new File(new URI(dependencyURL));
      logger.infofmt("Adding jar at: %s", dependencyFile.getAbsolutePath());
      if (!dependencyFile.exists() || !dependencyFile.isFile()) {
        throw new CompilationDescriptionExtractorException(
            String.format("Invalid dependency found %s", dependencyFile.getAbsolutePath()));
      }
      jars.add(dependencyFile.getAbsolutePath());
    } catch (URISyntaxException ex) {
      throw new CompilationDescriptionExtractorException(
          String.format("Invalid URI for dependency: %s", dependencyURL));
    }
  }

  /**
   * Returns a {@link CompilationDescription} that has been filled by passing the sources and
   * required jars through the extractor to get to the minimal set of .class files necessary to
   * build the contained {@link CompilationUnit}.
   **/
  private CompilationDescription makeCompilationDescription(List<String> sources, List<String> jars)
      throws ExtractionException, IOException {
    JavaCompilationUnitExtractor extractor = new JavaCompilationUnitExtractor(project.getName(),
        repositoryRoot.toAbsolutePath().toString());

    ArrayList<String> empty = new ArrayList<String>();
    ArrayList<String> outputOptions = new ArrayList<String>();
    outputOptions.add("-d");
    outputOptions.add(Files.createTempDirectory(null).toAbsolutePath().toString());

    CompilationDescription description = extractor.extract(
        "maven", sources, jars, empty, empty, empty, outputOptions, "");

    return description;
  }
}
