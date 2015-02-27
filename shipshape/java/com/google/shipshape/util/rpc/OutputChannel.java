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

package com.google.shipshape.util.rpc;

import com.google.common.base.Function;
import com.google.common.base.Optional;
import com.google.common.base.Preconditions;
import com.google.common.util.concurrent.Uninterruptibles;

import java.util.Iterator;
import java.util.NoSuchElementException;
import java.util.concurrent.BlockingQueue;
import java.util.concurrent.LinkedBlockingQueue;

/** A streaming channel for sending a {@link Server.Service.Method}'s outputs. */
public abstract class OutputChannel<T> {

  /** Received a value from the channel. */
  public abstract void onValue(T value);

  /**
   * Send the given value as an output.
   *
   * @deprecated use {@link onValue(T)}
   */
  @Deprecated
  public OutputChannel<T> send(T data) {
    onValue(data);
    return this;
  }

  /** Signal an error occurred and the end of the channel. */
  public abstract void onError(Throwable t);

  /** Signals the end of the channel. */
  public abstract void onCompleted();

  /**
   * Close the channel. Further sends will throw exceptions.
   *
   * @deprecated use {@link onCompleted()}
   */
  @Deprecated
  public void close() {
    onCompleted();
  }

  /**
   * Returns a wrapping {@link OutputChannel} that will apply the given function to a sent value
   * before sending it to {@code this} {@link OutputChannel}.
   */
  public <M> OutputChannel<M> transformingStream(Function<M, T> f) {
    return new TransformingOutputChannel<M, T>(this, f);
  }
}

/**
 * {@link OutputChannel} implemented with a blocking, unbuffered channel which can be retrieved
 * through the {@link #getOutputs()} method. The iterator returned will continue returning outputs
 * (or blocking until one is available) until the stream is closed. This {@link OutputChannel} is
 * thread-safe.
 */
class QueuedOutputChannel<T> extends OutputChannel<T> {
  private final BlockingQueue<Optional<T>> queue = new LinkedBlockingQueue<>();
  private boolean closed;
  private boolean outputsExposed;

  protected QueuedOutputChannel() {}

  @Override
  public synchronized void onValue(T data) {
    checkNotClosed();
    Uninterruptibles.putUninterruptibly(queue, Optional.of(data));
  }

  /** Returns an {@link Iterator} for the sent values. Cannot be called twice. */
  public synchronized Iterator<T> getOutputs() {
    Preconditions.checkState(!outputsExposed, "getOutputs should only be called once");
    outputsExposed = true;
    return new TakeUntilAbsentIterator();
  }

  @Override
  public synchronized void onError(Throwable t) {
    // TODO(schroederc): propagate error to receiver / do not use onCompleted
    onCompleted();
  }

  @Override
  public synchronized void onCompleted() {
    checkNotClosed();
    Uninterruptibles.putUninterruptibly(queue, Optional.<T>absent());
    closed = true;
  }

  private void checkNotClosed() {
    Preconditions.checkState(!closed, "OutputChannel has been closed.");
  }

  /**
   * Simple {@link Iterator} over the internal queue that stops once an absent {@link Optional}
   * object is reached. Not thread-safe.
   */
  private class TakeUntilAbsentIterator implements Iterator<T> {
    private Optional<T> next;

    @Override
    public boolean hasNext() {
      if (next == null) {
        next = Uninterruptibles.takeUninterruptibly(queue);
      }
      return next.isPresent();
    }

    @Override
    public T next() {
      Optional<T> val = next == null ? Uninterruptibles.takeUninterruptibly(queue) : next;
      if (!val.isPresent()) {
        throw new NoSuchElementException();
      }
      next = null;
      return val.get();
    }

    @Override
    public void remove() {
      throw new UnsupportedOperationException();
    }
  }
}

/** Implementing class for the {@link OutputChannel#transformingStream(Function)} method. */
class TransformingOutputChannel<M, T> extends OutputChannel<M> {
  private final OutputChannel<T> stream;
  private final Function<M, T> f;

  public TransformingOutputChannel(OutputChannel<T> stream, Function<M, T> f) {
    Preconditions.checkArgument(stream != null, "stream to transform must be non-null");
    Preconditions.checkArgument(f != null, "transforming function must be non-null");
    this.stream = stream;
    this.f = f;
  }

  @Override
  public void onValue(M data) {
    stream.onValue(f.apply(data));
  }

  @Override
  public void onError(Throwable t) {
    stream.onError(t);
  }

  @Override
  public void onCompleted() {
    stream.onCompleted();
  }
}
