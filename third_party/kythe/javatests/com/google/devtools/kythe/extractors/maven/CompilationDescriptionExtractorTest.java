package com.google.devtools.kythe.extractors.maven;

import com.google.devtools.kythe.common.FormattingLogger;
import com.google.devtools.kythe.extractors.shared.CompilationDescription;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;
import com.google.devtools.kythe.proto.Analysis.FileData;

import org.apache.maven.model.Dependency;
import org.junit.Assert;
import org.junit.Test;

import java.io.File;
import java.nio.file.Paths;

/**
 * Test class for the {@link CompilationDescriptionExtractor}.
 **/
public class CompilationDescriptionExtractorTest {
  private static final FormattingLogger logger =
      FormattingLogger.getLogger(CompilationDescriptionExtractorTest.class);

  private static final String TESTS_ROOT =
      "third_party/kythe/javatests/com/google/devtools/kythe/extractors/maven/data/";

  private static final String REPOSITORY_ROOT = TESTS_ROOT + "my-app";
  private static final String POM_PATH = "pom.xml";

  private static final String ARTIFACT_REPOSITORY = TESTS_ROOT + "repository";

  private static final String SOURCE_PATH = "src/main/java/com/mycompany/app/App.java";
  private static final String SOURCE_HASH =
      "9ce6836d3492764f36198ff8d075ac8b7ad21b3465f845324f9dfd9cc7a52156";

  /**
   * Test that checks that the extractor works on a very simple project.
   **/
  @Test
  public void testCreatingCompilationUnit() throws Throwable {
    File pom = Paths.get(REPOSITORY_ROOT, POM_PATH).toFile();
    ArtifactRepository localRepo = ArtifactRepository.newLocalRepository(ARTIFACT_REPOSITORY);

    Project project = new Project(localRepo, pom);
    for (Dependency dep : project.getDependencies()) {
      logger.infofmt("Dependency: GroupId: %s ArtifactId: %s Version: %s\n",
          dep.getGroupId(), dep.getArtifactId(), dep.getVersion());
      logger.infofmt("Path: %s Scope: %s \n", dep.getSystemPath(), dep.getScope());
    }

    CompilationDescriptionExtractor extractor = new CompilationDescriptionExtractor(project,
        localRepo, Paths.get(REPOSITORY_ROOT));

    CompilationDescription description = extractor.extract();

    CompilationUnit unit = description.getCompilationUnit();
    Iterable<FileData> files = description.getFileContents();

    Assert.assertEquals(1, unit.getRequiredInputList().size());
    CompilationUnit.FileInput file = unit.getRequiredInput(0);
    Assert.assertEquals(SOURCE_PATH, file.getInfo().getPath());
    Assert.assertEquals(SOURCE_HASH, file.getInfo().getDigest());
  }
}
