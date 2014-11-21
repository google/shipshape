package com.google.devtools.kythe.extractors.maven;

import com.google.devtools.kythe.common.FormattingLogger;

import org.apache.maven.model.Repository;
import org.apache.maven.model.building.ModelSource;
import org.apache.maven.model.resolution.ModelResolver;
import org.apache.maven.model.resolution.UnresolvableModelException;

import java.io.IOException;
import java.io.InputStream;
import java.net.URL;

/**
 * This class resolves the maven coordinates of a POM file using a {@link ArtifactRepository} to
 * obtain it's contents.
 **/
public class MavenModelResolver implements ModelResolver {
  private static final FormattingLogger logger = FormattingLogger.getLogger(
      MavenModelResolver.class);

  final private ArtifactRepository repo;

  /**
   * Simple implementation of the {@link ModelSource} interface suitable to be returned from the
   * {@link resolveModel} method.
   **/
  private static class MavenModelSource implements ModelSource {
    private final InputStream stream;
    private final String location;

    public MavenModelSource(InputStream stream, String location) {
      this.stream = stream;
      this.location = location;
    }

    @Override
    public InputStream getInputStream() {
      logger.infofmt("Accessing the stream");
      return stream;
    }

    @Override
    public String getLocation() {
      return location;
    }
  }

  /**
   * Initializes the resolver with the {@link ArtifactRepository} to use for resolving the Maven
   * coordinates.
   **/
  public MavenModelResolver(ArtifactRepository repo) {
    this.repo = repo;
  }

  /**
   * Adds a new repostitory to the list of repositories to use to resolve Maven coordinates. Note
   * that because this resolver is only used during the POM phase we don't actually need to store
   * the given repository, but we need to implement the call to satisfy the interface.
   **/
  @Override
  public void addRepository(Repository repository) {
    logger.infofmt("Name: %s Url: %s\n", repository.getName(), repository.getUrl());
  }

  /**
   * Returns a new copy of the resolver, because this class doesn't have any modifiable state
   * returning itself as the new copy is sufficient.
   **/
  @Override
  public ModelResolver newCopy() {
    return this;
  }

  @Override
  public ModelSource resolveModel(String groupId, String artifactId, String version)
      throws UnresolvableModelException {
    String pomLocation = repo.getPomURL(groupId, artifactId, version);
    logger.infofmt("Resolving Model GroupId:%s ArtifactId:%s Version:%s URI:%s\n",
        groupId, artifactId, version, pomLocation);

    try {
      URL url = new URL(pomLocation);
      InputStream stream = url.openStream();
      return new MavenModelSource(stream, pomLocation);
    } catch (IOException ex) {
      throw new UnresolvableModelException("Can't resolve pom",
          groupId, artifactId, version, ex);
    }
  }
}
