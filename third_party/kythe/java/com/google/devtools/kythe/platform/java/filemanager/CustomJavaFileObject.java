// Copyright 2011 Google Inc. All Rights Reserved.

package com.google.devtools.kythe.platform.java.filemanager;

import com.google.devtools.kythe.platform.shared.FileDataProvider;

import javax.lang.model.element.Modifier;
import javax.lang.model.element.NestingKind;
import javax.tools.JavaFileObject;

/**
 * JavaFileObject that can be provided a body at a future time.
 */
public class CustomJavaFileObject extends CustomFileObject implements JavaFileObject {

  private final String className;
  private Kind kind;

  public CustomJavaFileObject(FileDataProvider contentProvider,
      String path, String digest, String className, Kind kind, String encoding) {
    super(contentProvider, path, digest, encoding);
    this.className = className;
    this.kind = kind;
  }

  @Override
  public Kind getKind() {
    return kind;
  }

  @Override
  public boolean isNameCompatible(String simpleName, Kind kind) {
    return getName().equals(simpleName + kind.extension);
  }

  @Override
  public NestingKind getNestingKind() {
    return null;
  }
  @Override
  public Modifier getAccessLevel() {
    return null;
  }

  public String getClassName() {
    return className;
  }

  @Override
  public String toString() {
    return "CustomJavaFileObject [className=" + className + ", kind=" + kind + "]";
  }
}
