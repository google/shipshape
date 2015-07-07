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

import com.google.common.base.Splitter;
import com.google.common.collect.ImmutableList;
import com.google.shipshape.proto.ShipshapeContextProto.Stage;

import org.kohsuke.stapler.DataBoundConstructor;

import hudson.Extension;
import hudson.FilePath;
import hudson.Launcher;
import hudson.model.BuildListener;
import hudson.model.AbstractBuild;
import hudson.model.AbstractProject;
import hudson.tasks.BuildStepDescriptor;
import hudson.tasks.Builder;

import java.io.IOException;
import java.util.List;
import java.util.LinkedList;

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
public class AnalysisRunner extends Builder {

  // Configure how we split up a comma-separated list of Shipshape categories.
  private static final Splitter CATEGORY_PARSER =
      Splitter.on(",").trimResults().omitEmptyStrings();

  // Plugin parameters (needs to be public final):
  // Comma-separated list of categories to run.
  public final String categories;
  /** The shipshape CLI command.  */
  public final String command;
  /** Comma-separated Docker image URIs for third-party analyzers.  */
  public final String analyzerImages;
  // Whether to use verbose output
  public final boolean verbose;
  // Custom docker socket, e.g., tcp://localhost:4243.
  public final String socket;
  // Stage to run analyses for
  public final Stage stage;

  /**
   * Fields in config.jelly must match the parameter names in the
   * "DataBoundConstructor" and public final fields (above).
   *
   * @param categories the categories to run on.
   * @param verbose whether to use verbose output
   * @param socket use the TCP socket to communicate with docker.
   */
  @DataBoundConstructor
  public AnalysisRunner(
      final String categories,
      final Stage stage,
      final boolean verbose,
      final String socket,
      final String command,
      final String analyzerImages) {
    this.categories = categories;
    this.verbose = verbose;
    this.socket = socket;
    this.stage = stage;
    this.command = command;
    this.analyzerImages = analyzerImages;
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
      final BuildListener listener) throws InterruptedException, IOException {

    List<String> cmd = new LinkedList<String>();
    cmd.add(command == null ? "shipshape" : command);
    if (verbose) {
      cmd.add("--stderrthreshold=INFO");
    }
    if (categories != null && !categories.trim().isEmpty()) {
      cmd.add("--categories=" + categories);
    }
    if (analyzerImages != null && !analyzerImages.trim().isEmpty()) {
      cmd.add("--analyzer_images=" + analyzerImages.trim());
    }
    cmd.add("--json_output=shipshape-findings.json");
    cmd.add("--inside_docker=true");
    cmd.add("."); // Analyze the workspace (the working directory).
    launcher.launch()
        .cmds(cmd)
        .stdout(listener)
        .pwd(build.getProject().getWorkspace())
        .join();
    return true;
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
    @Override
    public boolean isApplicable(final Class<? extends AbstractProject> aClass) {
      return true;
    }

    public Stage[] stages() {
      return new Stage[]{Stage.PRE_BUILD, Stage.POST_BUILD};
    }

    /**
     * @return This name is used in the configuration screen.
     */
    @Override
    public String getDisplayName() {
      return "Shipshape Plugin";
    }
  }
}
