package com.google.devtools.kythe.extractors.maven;

import java.io.File;
import java.nio.file.Paths;

/**
 * This class abstracts the notion of a repository of Maven artifacts (sources, jars, etc) and
 * implements of the necessary logic to create the URLs to download them.
 **/
public class ArtifactRepository {
  // The path to the root of the repository
  private final String repoRootPath;

  public ArtifactRepository(String repoRootPath) {
    this.repoRootPath = repoRootPath;
  }

  public String getBasePath() {
    return repoRootPath;
  }

  /**
   * Returns the URL to the POM file pointed to by the given Maven coordinates in the repository
   * represented by this instance.
   **/
  public String getPomURL(String groupId, String artifactId, String version) {
    return String.format("%s/%s",
        getArtifactVersionURI(groupId, artifactId, version), getPomName(artifactId, version));
  }

  /**
   * Returns the URL to the jar pointed to by the given Maven coordinates in the repository
   * represented by this instance.
   **/
  public String getJarURL(String groupId, String artifactId, String version) {
    return String.format("%s/%s",
        getArtifactVersionURI(groupId, artifactId, version), getJarName(artifactId, version));
  }

  /**
   * Returns the URL to the sources jar pointed to by the given Maven coordinates in the repository
   * represented by this instance.
   **/
  public String getSourcesJarURL(String groupId, String artifactId, String version) {
    return String.format("%s/%s",
        getArtifactVersionURI(groupId, artifactId, version),
        getSourcesJarName(artifactId, version));
  }

  /**
   * Creates and initializes an instance of ArtifactRepository that represents the local repository
   * of Maven artifacts that Maven keeps in the users machine. Allows for access to the local
   * Artifacts for maximum speed.
   **/
  public static ArtifactRepository newLocalRepository() {
    File repository = Paths.get(System.getProperty("user.home"), ".m2/repository").toFile();
    if (!repository.exists() || !repository.isDirectory()) {
      return null;
    }
    return new ArtifactRepository("file://" + repository.getAbsolutePath());
  }

  /**
   * Creates and initializes an instance of ArtifactRepository that points to the given path, useful
   * for testing.
   **/
  public static ArtifactRepository newLocalRepository(String repoRootPath) {
    File repoRootPathFile = new File(repoRootPath);
    return new ArtifactRepository("file://" + repoRootPathFile.getAbsolutePath());
  }

  private String getGroupIdURI(String groupID) {
    return groupID.replace(".", "/");
  }

  private String getArtifactVersionURI(String groupId, String artifactId, String version) {
    return String.format("%s/%s/%s/%s",
        repoRootPath, getGroupIdURI(groupId), artifactId, version);
  }

  private String getPomName(String artifactId, String version) {
    return String.format("%s-%s.pom", artifactId, version);
  }

  private String getJarName(String artifactId, String version) {
    return String.format("%s-%s.jar", artifactId, version);
  }

  private String getSourcesJarName(String artifactId, String version) {
    return String.format("%s-%s-sources.jar", artifactId, version);
  }
}
