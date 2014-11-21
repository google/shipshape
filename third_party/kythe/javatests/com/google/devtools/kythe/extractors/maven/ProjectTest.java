package com.google.devtools.kythe.extractors.maven;

import com.google.devtools.kythe.common.FormattingLogger;

import org.apache.maven.model.Dependency;
import org.junit.Assert;
import org.junit.Test;

import java.io.File;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.List;

/**
 * Test class for the {@link Project} class.
 **/
public class ProjectTest {
  private static final FormattingLogger logger = FormattingLogger.getLogger(ProjectTest.class);

  private static final String TESTS_ROOT =
      "third_party/kythe/javatests/com/google/devtools/kythe/extractors/maven/data/";

  private static final String REPOSITORY_ROOT = TESTS_ROOT + "my-app";
  private static final String POM_PATH = "pom.xml";

  private static final String ARTIFACT_REPOSITORY = TESTS_ROOT + "repository";

  /**
   * Test method that verifies that a very simple project can be processed by the {@link Project}
   * class.
   **/
  @Test
  public void testLoadingPom() throws ProjectException {
    File pom = Paths.get(REPOSITORY_ROOT, POM_PATH).toFile();
    ArtifactRepository localRepo = ArtifactRepository.newLocalRepository(ARTIFACT_REPOSITORY);

    Project project = new Project(localRepo, pom);
    String sources = project.getSourceDirectory();
    List<Dependency> dependencies = project.getDependencies();

    Path cwd = Paths.get(System.getProperty("user.dir"));
    Path testRelative = cwd.relativize(Paths.get(project.getSourceDirectory()));

    Assert.assertEquals(REPOSITORY_ROOT + "/src/main/java", testRelative.toString());
    Assert.assertEquals(1, dependencies.size());
  }
}
