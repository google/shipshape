package com.google.devtools.kythe.extractors.maven;

import com.google.devtools.kythe.common.FormattingLogger;

import org.apache.maven.model.Dependency;
import org.apache.maven.model.Model;
import org.apache.maven.model.building.DefaultModelBuilder;
import org.apache.maven.model.building.DefaultModelBuildingRequest;
import org.apache.maven.model.building.DefaultModelProcessor;
import org.apache.maven.model.building.ModelBuilder;
import org.apache.maven.model.building.ModelBuildingException;
import org.apache.maven.model.building.ModelBuildingRequest;
import org.apache.maven.model.inheritance.DefaultInheritanceAssembler;
import org.apache.maven.model.interpolation.StringSearchModelInterpolator;
import org.apache.maven.model.io.DefaultModelReader;
import org.apache.maven.model.management.DefaultDependencyManagementInjector;
import org.apache.maven.model.management.DefaultPluginManagementInjector;
import org.apache.maven.model.normalization.DefaultModelNormalizer;
import org.apache.maven.model.path.DefaultModelPathTranslator;
import org.apache.maven.model.path.DefaultModelUrlNormalizer;
import org.apache.maven.model.path.DefaultPathTranslator;
import org.apache.maven.model.path.DefaultUrlNormalizer;
import org.apache.maven.model.profile.DefaultProfileSelector;
import org.apache.maven.model.resolution.ModelResolver;
import org.apache.maven.model.superpom.DefaultSuperPomProvider;
import org.apache.maven.model.validation.DefaultModelValidator;

import java.io.File;
import java.util.List;

/**
 * This class represents a Maven project, takes care of all of the setup
 * necessary to load and process a POM file.
 */
public class Project {
  private static final FormattingLogger logger = FormattingLogger.getLogger(
      Project.class);

  private Model model;
  private final ArtifactRepository repo;

  /**
   * Initializes the new project parsing the given pom, and using the given
   * repository to resolve any references.
   *
   * @param repo the repository to use when resolving Maven coordinages.
   * @param pom the pom file to load.
   */
  public Project(ArtifactRepository repo, File pom) throws ProjectException {
    this.repo = repo;

    ModelBuilder builder = getModelBuilder();
    ModelResolver resolver = new MavenModelResolver(this.repo);

    ModelBuildingRequest request = new DefaultModelBuildingRequest()
        .setProcessPlugins(false)
        .setPomFile(pom)
        .setModelResolver(resolver)
        .setValidationLevel(ModelBuildingRequest.VALIDATION_LEVEL_MINIMAL);

    try {
      this.model = builder.build(request).getEffectiveModel();
    } catch (ModelBuildingException ex) {
      logger.severe(ex, "Failed to create the model");
      throw new ProjectException("Failed to create the model", ex);
    }
  }

  /**
   * @return the name of the project composed from the Maven coordinates.
   **/
  public String getName() {
    return model.getGroupId() + "_" + model.getArtifactId() + "_" + model.getVersion();
  }

  /**
   * @return the root directory of the sources for the project.
   **/
  public String getSourceDirectory() {
    return model.getBuild().getSourceDirectory();
  }

  /**
   * @return the list of dependencies for this project.
   **/
  public List<Dependency> getDependencies() {
    return model.getDependencies();
  }

  /**
   * @return the repository used to resolve the POM file for this Project.
   **/
  public ArtifactRepository getRepository() {
    return repo;
  }

  private ModelBuilder getModelBuilder() {
    DefaultModelProcessor modelProcessor = new DefaultModelProcessor();
    modelProcessor.setModelReader(new DefaultModelReader());

    DefaultModelProcessor superModelProcessor = new DefaultModelProcessor();
    superModelProcessor.setModelReader(new DefaultModelReader());

    DefaultSuperPomProvider superPomProvider = new DefaultSuperPomProvider();
    superPomProvider.setModelProcessor(superModelProcessor);

    StringSearchModelInterpolator interpolator = new StringSearchModelInterpolator();
    interpolator.setPathTranslator(new DefaultPathTranslator());

    DefaultModelUrlNormalizer modelUrlNormalizer = new DefaultModelUrlNormalizer();
    modelUrlNormalizer.setUrlNormalizer(new DefaultUrlNormalizer());

    DefaultModelPathTranslator modelPathTranslator = new DefaultModelPathTranslator();
    modelPathTranslator.setPathTranslator(new DefaultPathTranslator());

    DefaultModelBuilder builder = new DefaultModelBuilder();
    builder.setProfileSelector(new DefaultProfileSelector());
    builder.setModelProcessor(modelProcessor);
    builder.setModelValidator(new DefaultModelValidator());
    builder.setSuperPomProvider(superPomProvider);
    builder.setModelNormalizer(new DefaultModelNormalizer());
    builder.setInheritanceAssembler(new DefaultInheritanceAssembler());
    builder.setModelInterpolator(interpolator);
    builder.setModelPathTranslator(modelPathTranslator);
    builder.setModelUrlNormalizer(modelUrlNormalizer);
    builder.setPluginManagementInjector(new DefaultPluginManagementInjector());
    builder.setDependencyManagementInjector(new DefaultDependencyManagementInjector());

    return builder;
  }
}
