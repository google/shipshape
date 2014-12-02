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

  // Default socket value used for docker communication. This is used as the default for
  // the socket plugin parameter when no other value is given.
  // For containers running inside a slave container this is the socket used (there is typically no
  // TCP socket available inside slave containers, but instead the plugin runs as root)
  // and no other value should be given.
  private static final String DEFAULT_DOCKER_SOCKET = "unix:///var/run/docker.sock";

  // Configure how we split up a comma-separated list of Shipshape categories.
  private static final Splitter CATEGORY_PARSER =
      Splitter.on(",").trimResults().omitEmptyStrings();

  // Plugin parameters (needs to be public final):
  // Comma-separated list of categories to run.
  public final String categories;
  // Whether to use verbose output
  public final boolean verbose;
  // Custom docker socket, e.g., tcp://localhost:4243.
  public final String socket;

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
      final boolean verbose,
      final String socket) {
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
      final BuildListener listener) throws InterruptedException, IOException {

    // Serializable values (strings, primitive types ...) needed on the Jenkins slave should be
    // moved to final variables here to be used in the anonymous Callable class below.
    final String socket = (this.socket == null || this.socket.trim().equals(""))
        ? DEFAULT_DOCKER_SOCKET : this.socket;
    final ImmutableList<String> categoryList = ImmutableList.copyOf(
        CATEGORY_PARSER.split(this.categories == null ? "" : this.categories));

    final FilePath workspace = build.getWorkspace();
    final String jobName = build.getProject().getDisplayName();
    final boolean isVerbose = verbose;
    int actionableResults = 0;
    try {
      actionableResults = workspace.act(
          new ShipshapeSlave(workspace, isVerbose, categoryList, socket, jobName, listener));
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

    /**
     * @return This name is used in the configuration screen.
     */
    @Override
    public String getDisplayName() {
      return "Google Analysis Plugin";
    }
  }
}
