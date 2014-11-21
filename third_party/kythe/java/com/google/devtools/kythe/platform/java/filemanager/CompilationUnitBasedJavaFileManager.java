package com.google.devtools.kythe.platform.java.filemanager;

import com.google.devtools.kythe.platform.shared.FileDataProvider;
import com.google.devtools.kythe.proto.Analysis.CompilationUnit;
import com.google.devtools.kythe.proto.Analysis.JavaArguments;

import java.util.HashSet;
import java.util.Set;

import javax.tools.StandardJavaFileManager;
import javax.tools.StandardLocation;

/**
 * Makes it easier for our analysis to provide files in different ways than on the local file system
 * e.g. from BigTables or .index files. All users of this class have to do is provide a
 * {@link FileDataProvider} that will feed the actual content in as a future.
 */
public class CompilationUnitBasedJavaFileManager extends JavaFileStoreBasedFileManager {

  /**
   * paths searched for .class files, can be relative or absolute, but must match path as named by
   * the extractor.
   */
  private final Set<String> classpath = new HashSet<>();

  /**
   * paths searched for .java files, can be relative or absolute, but must match path as named by
   * the extractor.
   */
  private final Set<String> sourcepath = new HashSet<>();

  public CompilationUnitBasedJavaFileManager(FileDataProvider contentProvider,
      CompilationUnit unit, StandardJavaFileManager fileManager, String encoding) {
    super(new CompilationUnitBasedJavaFileStore(unit, contentProvider, encoding), fileManager);
    JavaArguments javaArguments = unit.getJavaArguments();
    classpath.add("");
    classpath.addAll(javaArguments.getClasspathList());
    sourcepath.add("");
    sourcepath.addAll(javaArguments.getSourcepathList());
  }

  @Override
  protected Set<String> getSearchPaths(Location location) {
    Set<String> dirsToLookIn = new HashSet<>();
    if (location == StandardLocation.CLASS_PATH) {
      dirsToLookIn = classpath;
    } else if (location == StandardLocation.SOURCE_PATH) {
      dirsToLookIn = sourcepath;
    }
    return dirsToLookIn;
  }
}
