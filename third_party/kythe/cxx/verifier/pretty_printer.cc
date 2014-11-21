#include <bitset>

#include "pretty_printer.h"

namespace kythe {
namespace verifier {

PrettyPrinter::~PrettyPrinter() {}

void StringPrettyPrinter::Print(const std::string &string) { data_ << string; }

void StringPrettyPrinter::Print(const char *string) { data_ << string; }

void StringPrettyPrinter::Print(const void *ptr) { data_ << ptr; }

void FileHandlePrettyPrinter::Print(const std::string &string) {
  fprintf(file_, "%s", string.c_str());
}

void FileHandlePrettyPrinter::Print(const char *string) {
  fprintf(file_, "%s", string);
}

void FileHandlePrettyPrinter::Print(const void *ptr) {
  fprintf(file_, "0x%016llx", reinterpret_cast<unsigned long long>(ptr));
}

void QuoteEscapingPrettyPrinter::Print(const std::string &string) {
  Print(string.c_str());
}

void QuoteEscapingPrettyPrinter::Print(const char *string) {
  char buf[2];
  buf[1] = 0;
  while ((buf[0] = *string++)) {
    if (buf[0] == '\"') {
      wrapped_.Print("\\\"");
    } else if (buf[0] == '\n') {
      wrapped_.Print("\\n");
    } else if (buf[0] == '\'') {
      wrapped_.Print("\\\'");
    } else {
      wrapped_.Print(buf);
    }
  }
}

void QuoteEscapingPrettyPrinter::Print(const void *ptr) { wrapped_.Print(ptr); }

void HtmlEscapingPrettyPrinter::Print(const std::string &string) {
  Print(string.c_str());
}

void HtmlEscapingPrettyPrinter::Print(const char *string) {
  char buf[2];
  buf[1] = 0;
  while ((buf[0] = *string++)) {
    if (buf[0] == '\"') {
      wrapped_.Print("&quot;");
    } else if (buf[0] == '&') {
      wrapped_.Print("&amp;");
    } else if (buf[0] == '<') {
      wrapped_.Print("&lt;");
    } else if (buf[0] == '>') {
      wrapped_.Print("&gt;");
    } else {
      wrapped_.Print(buf);
    }
  }
}

void HtmlEscapingPrettyPrinter::Print(const void *ptr) { wrapped_.Print(ptr); }

}  // namespace verifier
}  // namespace kythe
