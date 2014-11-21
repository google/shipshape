//===-- CommandLineUtilsTest.cpp - Test for argument utilities -*- C++ -*--===//
//
//                     The LLVM Compiler Infrastructure
//
// This file is distributed under the University of Illinois Open Source
// License. See LICENSE.TXT for details.
//
//===----------------------------------------------------------------------===//
/// \file
/// \brief Contains tests for CommandLineUtils{.h,.cc}.
//===----------------------------------------------------------------------===//

// This file uses the Clang style conventions.

// TODO(zarko): "clang/Driver/CommandLineUtils.h"
#include "CommandLineUtils.h"
#include "gtest/gtest.h"

namespace {

using ::kythe::common::HasCxxInputInCommandLineOrArgs;
using ::kythe::common::AdjustClangArgsForAnalyze;
using ::kythe::common::AdjustClangArgsForSyntaxOnly;
using ::kythe::common::ClangArgsToGCCArgs;
using ::kythe::common::GCCArgsToClangArgs;
using ::kythe::common::DetermineDriverAction;
using ::kythe::common::DriverAction;
using ::kythe::common::ASSEMBLY;
using ::kythe::common::CXX_COMPILE;
using ::kythe::common::C_COMPILE;
using ::kythe::common::FORTRAN_COMPILE;
using ::kythe::common::GO_COMPILE;
using ::kythe::common::LINK;
using ::kythe::common::UNKNOWN;

TEST(HasCxxInputInCommandLineOrArgs, GoodInputs) {
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a.c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a.c", "b", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.c", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b", "c.c"}));

  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.C", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.c++", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.cc", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.cp", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.cpp", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.cxx", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.i", "c"}));
  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "a", "b.ii", "c"}));

  EXPECT_TRUE(HasCxxInputInCommandLineOrArgs({"-c", "base/timestamp.cc"}));
}

TEST(HasCxxInputInCommandLineOrArgs, BadInputs) {
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"", "", ""}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.o", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.a", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b", "c."}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.ccc", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.ccc.ccc"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.ccc+", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.cppp", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.CC", "c"}));
  EXPECT_FALSE(HasCxxInputInCommandLineOrArgs({"a", "b.xx", "c"}));
  EXPECT_FALSE(
      HasCxxInputInCommandLineOrArgs({"-Wl,@foo", "base/timestamp.cc"}));
  EXPECT_FALSE(
      HasCxxInputInCommandLineOrArgs({"base/timestamp.cc", "-Wl,@foo"}));
}

// TODO(zarko): Port additional tests.

} // namespace

int main(int argc, char **argv) {
  ::testing::InitGoogleTest(&argc, argv);
  int result = RUN_ALL_TESTS();
  return result;
}
