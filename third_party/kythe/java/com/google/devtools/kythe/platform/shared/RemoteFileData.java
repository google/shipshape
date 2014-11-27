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

package com.google.devtools.kythe.platform.shared;

import static com.google.devtools.kythe.proto.FileDataServiceKrpc.newStub;

import com.google.common.util.concurrent.SettableFuture;
import com.google.devtools.kythe.platform.rpc.HttpTransport;
import com.google.devtools.kythe.platform.rpc.OutputChannel;
import com.google.devtools.kythe.proto.Analysis.FileData;
import com.google.devtools.kythe.proto.Analysis.FileInfo;
import com.google.devtools.kythe.proto.FileDataServiceKrpc.FileDataService;

import java.util.concurrent.Future;

/**
 * {@link FileDataProvider} that looks up file data using a {@link FileDataService}.
 */
public class RemoteFileData implements FileDataProvider {
  private final FileDataService service;

  public RemoteFileData(String address) {
    this.service = newStub(HttpTransport.parse(address));
  }

  @Override
  public Future<byte[]> startLookup(String path, String digest) {
    SettableFuture<byte[]> future = SettableFuture.create();
    service.getFileData(FileInfo.newBuilder()
        .setPath(path).setDigest(digest)
        .build(),
        new OutputChannel<FileData>() {
          @Override
          public void onValue(FileData data) {
            future.set(data.getContent().toByteArray());
          }

          @Override
          public void onError(Throwable t) {
            future.setException(t);
          }

          @Override
          public void onCompleted() {}
        });
    return future;
  }
}
