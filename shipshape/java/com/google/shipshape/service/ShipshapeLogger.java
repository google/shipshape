package com.google.shipshape.service;

import com.google.common.base.Strings;
import com.google.devtools.kythe.common.FormattingLogger;
import com.google.shipshape.proto.ShipshapeContextProto.ShipshapeContext;

import javax.annotation.Nullable;

/**
 * Wrapper class around the formatting logger so that we have consistently formatted logs and always
 * log relevant information from the {@link ShipshapeContext}.
 */
public class ShipshapeLogger {
  private FormattingLogger logger;

  public ShipshapeLogger(Class<?> cls) {
    logger = new FormattingLogger(cls);
  }

  public ShipshapeLogger(FormattingLogger logger) {
    this.logger = logger;
  }

  public static ShipshapeLogger getLogger(Class<?> cls) {
    return new ShipshapeLogger(cls);
  }

  public FormattingLogger getFormattingLogger() {
    return logger;
  }

  public void info(String message, ShipshapeContext context) {
    logger.infofmt(getLogMessageWithContext(message, context, null));
  }

  public void info(String message, ShipshapeContext context, String category) {
    logger.infofmt(getLogMessageWithContext(message, context, category));
  }

  public void warning(String message, ShipshapeContext context) {
    logger.warningfmt(getLogMessageWithContext(message, context, null));
  }

  public void warning(String message, ShipshapeContext context, String category) {
    logger.warningfmt(getLogMessageWithContext(message, context, category));
  }

  public void warning(Throwable e, String message, ShipshapeContext context) {
    logger.warningfmt(e, getLogMessageWithContext(message, context, null));
  }

  public void warning(Throwable e, String message, ShipshapeContext context,
      String category) {
    logger.warningfmt(e, getLogMessageWithContext(message, context, category));
  }

  public void severe(String message, ShipshapeContext context) {
    logger.severefmt(getLogMessageWithContext(message, context, null));
  }

  public void severe(String message, ShipshapeContext context, String category) {
    logger.severefmt(getLogMessageWithContext(message, context, category));
  }

  public void severe(Throwable e, String message, ShipshapeContext context) {
    logger.severefmt(e, getLogMessageWithContext(message, context, null));
  }

  public void severe(Throwable e, String message, ShipshapeContext context, String category) {
    logger.severefmt(e, getLogMessageWithContext(message, context, category));
  }

  /**
    * Appends key information from the context and analyzerName (if non-null) to the log message.
    */
  private static String getLogMessageWithContext(String message, ShipshapeContext context,
      @Nullable String category) {
    // TODO(jvg): add details from ShipshapeContext
    if (Strings.isNullOrEmpty(category)) {
      return message;
    } else {
      return String.format("Analyzer %s: %s", category, message);
    }
  }
}
