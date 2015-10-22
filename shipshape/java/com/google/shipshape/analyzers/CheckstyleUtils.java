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

package com.google.shipshape.analyzers;

import com.google.common.base.Function;
import com.google.common.base.Predicate;
import com.google.common.collect.FluentIterable;
import com.google.common.collect.ImmutableList;
import com.google.common.collect.ImmutableMap;
import com.google.common.io.ByteStreams;
import com.google.shipshape.proto.NotesProto.Location;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.proto.TextRangeProto.TextRange;
import com.google.shipshape.service.AnalyzerException;

import org.xml.sax.Attributes;
import org.xml.sax.SAXException;
import org.xml.sax.helpers.DefaultHandler;

import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.util.concurrent.Callable;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import javax.xml.parsers.ParserConfigurationException;
import javax.xml.parsers.SAXParser;
import javax.xml.parsers.SAXParserFactory;

/**
 * A utility class of methods for writing analyzers that wrap Checkstyle.
 */
final class CheckstyleUtils {
  private static final ExecutorService threadpool = Executors.newCachedThreadPool();
  private static final SAXParserFactory saxParserFactory = SAXParserFactory.newInstance();
  private static final String checkstylePackageBase = "com.puppycrawl.tools.checkstyle.checks.";

  /**
   * Create a map of all the Java files in the context where the key is the absolute path
   * and the value is relative path.
   *
   * The reason to have this done as a separate step is to allow the caller to partition the
   * map to analyze different files with different configurations.
   *
   * @param context the Shipshape context.
   * @return a map of all Java files.
   */
  static ImmutableMap<String, String> getJavaFiles(final ShipshapeContext context) {
    return FluentIterable.from(context.getFilePathList())
        .filter(new Predicate<String>() {
          @Override
          public boolean apply(String path) {
            return path.endsWith(".java");
          }
        })
        .uniqueIndex(new Function<String, String>() {
          @Override
          public String apply(String path) {
            return new File(context.getRepoRoot(), path).getPath();
          }
        });
  }

  /**
   * Run Checkstyle against a set of Java files using a specified Checkstyle configuration file.
   *
   * @param context the Shipshape context.
   * @param category the category to report the problems as coming from.
   * @param javaFiles the files to analyze; the map keys are the absolute paths to the file while
   * the values the relative paths to the file from context.file_path. The correct map can be
   * created using {@link CheckstyleUtils#getJavaFiles(ShipshapeContext)}.
   * @param checkstyleConfig the Checkstyle configuration to use.
   * @return the list of problems found by Checkstyle.
   * @throws AnalyzerException
   */
  static ImmutableList<Note>  runCheckstyle(ShipshapeContext context, String category,
      ImmutableMap<String, String> javaFiles, String checkstyleConfig) throws AnalyzerException {
    ImmutableList<String> commandLine = new ImmutableList.Builder<String>()
        .add("java")
        .add("-jar", "/usr/local/bin/checkstyle-6.11.2-all.jar")
        .add("-c", checkstyleConfig)
        .add("-f", "xml")
        .addAll(javaFiles.keySet())
        .build(); 
    ProcessBuilder processBuilder = new ProcessBuilder(commandLine);
    processBuilder.redirectInput(new File("/dev/null"));
    processBuilder.redirectOutput(ProcessBuilder.Redirect.PIPE);
    processBuilder.redirectError(ProcessBuilder.Redirect.PIPE);
    Process process;
    try {
      process = processBuilder.start();
    } catch (IOException e) {
      throw new AnalyzerException(category, context,
          String.format("error starting command %s", commandLine), e);
    }
    byte[] stdout;
    byte[] stderr;
    try {
      // We need to read in the entire stream as pipes can get filled and block the process.
      Future<byte[]> stdoutFuture = threadpool.submit(new ReadAll(process.getInputStream()));
      Future<byte[]> stderrFuture = threadpool.submit(new ReadAll(process.getErrorStream()));
      process.waitFor();
      stdout = stdoutFuture.get();
      stderr = stderrFuture.get();
    } catch (InterruptedException | ExecutionException e) {
      throw new AnalyzerException(category, context,
          String.format("error waiting for command %s", commandLine), e);
    }
    // TODO(rsk): double check what a non-0 exit code means.
    if (process.exitValue() != 0 || stderr.length > 0) {
      throw new AnalyzerException(category, context,
          String.format("command %s failed with %d and stderr \"%s\"", commandLine,
            process.exitValue(), stderr));
    }

    return parseCheckstyleXml(context, category, javaFiles, stdout);
  }

  private static class ReadAll implements Callable<byte[]> {
    private final InputStream inputStream;

    public ReadAll(InputStream inputStream) {
      this.inputStream = inputStream;
    }

    @Override
    public byte[] call() throws IOException {
      return ByteStreams.toByteArray(inputStream);
    }
  }

  private static ImmutableList<Note> parseCheckstyleXml(ShipshapeContext context, String category,
      ImmutableMap<String, String> javaFiles, byte[] xml) throws AnalyzerException {
    CheckstyleHandler handler = new CheckstyleHandler(context, category, javaFiles);
    try {
      SAXParser saxParser = saxParserFactory.newSAXParser();
      saxParser.parse(new ByteArrayInputStream(xml), handler);
    } catch (ParserConfigurationException | SAXException | IOException e) {
      throw new AnalyzerException(category, context, "XML parsing error", e);
    }
    return handler.results();
  }

  private static class CheckstyleHandler extends DefaultHandler {
    private final ShipshapeContext context;
    private final String category;
    private final ImmutableMap<String, String> javaFiles;
    private final ImmutableList.Builder<Note> listBuilder = new ImmutableList.Builder<>();
    private String currentFile;

    public CheckstyleHandler(ShipshapeContext context, String category,
        ImmutableMap<String, String> javaFiles) {
      this.context = context;
      this.category = category;
      this.javaFiles = javaFiles;
    }

    public ImmutableList<Note> results() {
      return listBuilder.build();
    }

    @Override
    public void startElement(String unusedUri, String unusedLocalName, String qualifiedName,
        Attributes attributes) throws SAXException {
      switch (qualifiedName) {
        case "checkstyle":
          // Nothing of interest.
          break;
        case "file":
          currentFile = javaFiles.get(getRequiredAttribute(attributes, "name", qualifiedName));
          break;
        case "error":

          // TODO(rsk): should we do something with this (perhaps drop "ignore" and "info")?
          // String severity = getRequiredAttribute(attributes, "severity", qName);
          // switch (severity) {
          // case "ignore":
          // case "info":
          // case "warning":
          // case "error":
          //   break;
          // default:
          //   throw new SAXException(String.format("Unrecognized severity value %s.", severity));
          // }

          TextRange.Builder textRangeBuilder = TextRange.newBuilder()
              .setStartLine(Integer.parseInt(
                  getRequiredAttribute(attributes, "line", qualifiedName)));
          String columnText = attributes.getValue("column");
          if (columnText != null) {
            textRangeBuilder.setStartColumn(Integer.parseInt(columnText));
          }

          String source = getRequiredAttribute(attributes, "source", qualifiedName);
          if (source.startsWith(checkstylePackageBase)) {
            source = source.substring(checkstylePackageBase.length());
          }
          listBuilder.add(Note.newBuilder()
              .setCategory(category)
              .setSubcategory(source)
              .setDescription(getRequiredAttribute(attributes, "message", qualifiedName))
              .setLocation(Location.newBuilder()
                  .setSourceContext(context.getSourceContext())
                  .setPath(currentFile)
                  .setRange(textRangeBuilder))
              .build());
          break;
        case "exception":
          // TODO(rsk): determine data format. Note: XML handling of exceptions
          // appear to be broken.
          throw new SAXException("exception handling unimplemented.");
        default:
          throw new SAXException(String.format("Unrecognized element %s", qualifiedName));
      }
    }

    private String getRequiredAttribute(Attributes attributes, String attributeName,
            String elementName) throws SAXException {
      String value = attributes.getValue(attributeName);
      if (value == null) {
        throw new SAXException(String.format("Element file missing attribute name", elementName));
      }
      return value;
    }
  }
}
