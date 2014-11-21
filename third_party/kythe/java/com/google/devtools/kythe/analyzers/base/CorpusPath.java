package com.google.devtools.kythe.analyzers.base;

import com.google.devtools.kythe.proto.Storage.VName;

/** Path within a particular corpus and corpus root. */
public final class CorpusPath {
  private final String corpus, root, path;
  public CorpusPath(String corpus, String root, String path) {
    this.corpus = corpus;
    this.root = root;
    this.path = path;
  }

  /**
   * Returns a new {@link CorpusPath} equivalent to the corpus/path/root subset of the given
   * {@link VName}.
   */
  public static CorpusPath fromVName(VName vname) {
    return new CorpusPath(
        vname.getCorpus(),
        vname.getRoot(),
        vname.getPath());
  }

  public String getCorpus() { return corpus; }
  public String getRoot() { return root; }
  public String getPath() { return path; }
}
