package com.google.shipshape.analyzers;

import static org.hamcrest.MatcherAssert.assertThat;

import com.google.common.collect.ImmutableList;
import com.google.common.collect.ImmutableMap;
import com.google.devtools.source.v1.SourceContextProto.SourceContext;
import com.google.shipshape.proto.NotesProto.Location;
import com.google.shipshape.proto.NotesProto.Note;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;
import com.google.shipshape.proto.TextRangeProto.TextRange;

import org.hamcrest.Matchers;
import org.junit.BeforeClass;
import org.junit.Test;

import java.io.File;
import java.util.Arrays;

public class CheckstyleUtilsTest {
  private static final String JAVA_FILE =
      "shipshape/javatests/com/google/shipshape/analyzers/testdata/Test.java";
  private static final String NON_JAVA_FILE =
      "shipshape/javatests/com/google/shipshape/analyzers/testdata/test.go";

  private static final String CHECKSTYLE_JAR = "third_party/checkstyle/checkstyle-6.11.2-all.jar";

  private static final String TEST_SRCDIR = System.getenv().get("TEST_SRCDIR");

  @BeforeClass
  public static void classSetUp() {
    CheckstyleUtils.checkstyleJar = new File(TEST_SRCDIR, CHECKSTYLE_JAR).toString();
  }

  @Test
  public void testGetJavaFiles() {
    ShipshapeContext context = makeContext(NON_JAVA_FILE, JAVA_FILE);
    ImmutableMap<String, String> expected = ImmutableMap.of(
        new File(TEST_SRCDIR, JAVA_FILE).toString(), JAVA_FILE);
    assertThat(CheckstyleUtils.getJavaFiles(context), Matchers.equalTo(expected));
  }

  @Test
  public void testAnalyze() throws Exception {
    ShipshapeContext context = makeContext(JAVA_FILE);
    ImmutableList<Note> notes = CheckstyleUtils.runCheckstyle(context, "checkstyle",
        CheckstyleUtils.getJavaFiles(context), "/google_checks.xml");
    assertThat(notes, Matchers.hasSize(2));
    Note expected1 = Note.newBuilder()
        .setCategory("checkstyle")
        .setSubcategory("modifier.ModifierOrderCheck")
        .setDescription("'private' modifier out of order with the JLS suggestions.")
        .setLocation(Location.newBuilder()
            .setSourceContext(SourceContext.newBuilder())
            .setPath(JAVA_FILE)
            .setRange(TextRange.newBuilder().setStartLine(5).setStartColumn(9)))
        .build();
    Note expected2 = Note.newBuilder()
        .setCategory("checkstyle")
        .setSubcategory("naming.MemberNameCheck")
        .setDescription("Member name 'foo_bar' must match pattern '^[a-z][a-z0-9][a-zA-Z0-9]*$'.")
        .setLocation(Location.newBuilder()
            .setSourceContext(SourceContext.newBuilder())
            .setPath(JAVA_FILE)
            .setRange(TextRange.newBuilder().setStartLine(8).setStartColumn(7)))
        .build();
    assertThat(notes, Matchers.containsInAnyOrder(expected1, expected2));
  }

  ShipshapeContext makeContext(String... files) {
    return ShipshapeContext.newBuilder()
        .setRepoRoot(TEST_SRCDIR)
        .addAllFilePath(Arrays.asList(files))
        .build();
  }
}
