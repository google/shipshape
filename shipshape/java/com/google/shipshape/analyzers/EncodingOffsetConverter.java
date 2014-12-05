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

import static java.nio.charset.StandardCharsets.ISO_8859_1;
import static java.nio.charset.StandardCharsets.UTF_8;

import com.google.common.base.Preconditions;
import com.google.common.base.Utf8;

import java.io.UnsupportedEncodingException;
import java.nio.charset.Charset;
import java.util.Arrays;
import java.util.List;

/**
 * Converts a UTF-16 code unit offset to a byte offset.  Useful to convert javac's internal
 * string offsets into byte offsets that can be used without knowing what encoding the file is in.
 *
 * <p>Currently supports only UTF-8 and ISO-8859-1, which seem to be the only encodings used for
 * Java source files in google3.
 */
public class EncodingOffsetConverter {

  private final CharSequence fileContent;
  private final Charset encoding;

  private static final List<Charset> SUPPORTED_ENCODINGS = Arrays.asList(
      UTF_8, ISO_8859_1);

  public EncodingOffsetConverter(CharSequence fileContent, Charset encoding)
      throws UnsupportedEncodingException {
    this.fileContent = Preconditions.checkNotNull(fileContent);
    this.encoding = Preconditions.checkNotNull(encoding);
    if (!SUPPORTED_ENCODINGS.contains(encoding)) {
      throw new UnsupportedEncodingException("Unsupported encoding: " + encoding);
    }
  }

  /**
   * Converts a UTF-16 code unit index to a byte index, based on the source provided when this
   * {@link EncodingOffsetConverter} was constructed.
   *
   * <p>Note that each call to this method may take O(n) time, where n is the length of the
   * fileContent.  Do not call in a tight loop.
   *
   * @param utf16CodeUnitIndex An index into the source string in terms of UTF-16 code units
   * @return An index into the source string in terms of bytes
   */
  public int toByteIndex(int utf16CodeUnitIndex) {
    if (utf16CodeUnitIndex < 0 || utf16CodeUnitIndex >= fileContent.length()) {
      throw new IllegalArgumentException("index out of bounds: " + utf16CodeUnitIndex + ", length "
          + fileContent.length());
    }
    if (encoding.equals(UTF_8)) {
      return Utf8.encodedLength(fileContent.subSequence(0, utf16CodeUnitIndex));
    } else if (encoding.equals(ISO_8859_1)) {
      // ISO-8859-1 is a fixed 8-bit encoding, and it was incorporated as the first 256 code
      // points for Unicode.  UTF-16 encodes code points U+0000 to U+D7FF and U+E000 to U+FFFF
      // as single 16-bit code units, so it is OK to just return the UTF-16 code unit index.
      return utf16CodeUnitIndex;
    } else {
      throw new IllegalStateException("Unsupported encoding: " + encoding);
    }
  }
}
