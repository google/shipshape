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

// Implements a simple JSON formatter for C++ objects. The library doesn't use
// any kind of reflection to serialize the objects, instead it provides a set of
// objects that help creating a valid JSON string. The caller can provide
// functors to help serializing nested objects.
//
// General operation when serializing an object will look like this:
//   MyClass obj;
//   JSONObjectSerializer serializer;
//   serializer.BeginSerialization();
//   serializer.WriteProperty("name", "something");
//   serializer.WriteObject("obj", obj, [](const MyClass& obj,
//     JSONObjectSerializer* sub_serializer) {
//       sub_serializer.WriteProperty("prop1", obj.prop1);
//       sub_serializer.WriteProperty("prop2", obj.prop2);
//     });
//   serializer.EndSerialization();
//   string serialized = serializer.str();
//
// The BeginSerialization and EndSerialization calls will ensure that the object
// is correctly formatted, adding the initial {, and } as necessary. Note that
// the sub_serializers passed when serializing object properties have their
// BeginSerialization and EndSerialization method automatically called. After
// End is called then the caller can use the str() method to retrieve the JSON
// formatted string.
//
// There's also a JSONArraySerializer that is to be used to serialize a
// collection that can be enumerated by using iterators. A helper function makes
// this process much easier, as a collection can be serialized as follows:
//   vector<MyClass> objs;
//   string serialized = SerializeArrayToJSON(objs, [](const MyClass& obj,
//     JSONObjectSerializer* serializer) { ... };
// Where the functor will be called for each of the elements in the collection.
#ifndef KYTHE_RPC_CXX_JSON_HELPER_H_
#define KYTHE_RPC_CXX_JSON_HELPER_H_

#include <string>
#include <utility>

namespace json {

// This forward declarion for the template allows us to have a semi-circular
// dependency between JSONObjectSerializer and
// JSONArraySerializer. JSONObjectSerializer uses this template to serialize
// array properties instead of directly referencing the JSONArraySerializer
// class.
template <typename T, typename TFunc>
std::string SerializeArrayToJSON(const T& array, TFunc func);

// Simple JSON serializer for C++ classes, it doesn't use reflection, it offers
// up methods to write to an output string the various JSON constructs supported
// by this class. This class does not keep the objects in a tree as its only
// storage is an internal string stream that keeps the currently formatted
// string.
// BeginSerialization must be called before any of the Write* methods to ensure
// that the object is in the right state to receive properties and that the
// resulting JSON syntax is correct. EndSerialization must be called before the
// str() method is called to obtain the JSON string, this ensures that the
// syntax of the produced string is correct and that the object is correctly
// closed.
// Because BeginSerialization resets the state of the serializer completely it
// is possible to reuse the serializer instances as long as BeginSerialization
// is always called when serializing a new object.
class JSONObjectSerializer {
 public:
  // Begins the serialization process by re-setting the instance's state and
  // starting the JSON formatted string.
  void BeginSerialization() {
    *this = {};

    str_ += "{";
  }

  // Writes a property |name| with the given scalar |value|.
  // TODO: Add overloads for more value types.
  void WriteProperty(const std::string& name, const std::string& value) {
    EnsureSeparator();

    str_ += "\"" + name + "\": \"" + value + "\"";
  }

  // Writes a property |name| which value is an object. The caller must provide
  // |func| so object can be serialized. The |func| function is expected to
  // accept a const reference to T and a pointer to a JSONObjectSeriliazer in
  // which to serialize the sub-object. |func| does *not* need to call
  // BeginSerialization, nor it needs to call EndSerialization.
  template <typename T, typename TFunc>
  void WriteObject(const std::string& name, T&& obj, TFunc func) {
    EnsureSeparator();

    JSONObjectSerializer serializer;
    serializer.BeginSerialization();
    func(std::forward<T>(obj), &serializer);
    serializer.EndSerialization();

    str_ += "\"" + name + "\": " + serializer.str();
  }

  // Writes a property |name| which value is a collection, it is expected that
  // |array| can be enumerated in the STL way. |func| will be called for each of
  // the elements of the collection and it is expected to accept a const
  // reference to each elment as well as a pointer to JSONObjectSerializer to be
  // used to serialize that element in the same manner as WriteObject.
  template <typename T, typename TFunc>
  void WriteArray(const std::string& name, const T& array, TFunc func) {
    EnsureSeparator();

    str_ += "\"" + name + "\": " + SerializeArrayToJSON(array, func);
  }

  // Finishes the serialization of the object ensuring that the produced JSON is
  // valid. This method *must* be called before calling str() to retrieve the
  // string or the string will not be valid JSON. After this method is called
  // only str() can be called to retrieve the string and BeginSerialization to
  // serialize a different object, any other method is unsupported.
  void EndSerialization() { str_ += "}"; }

  // Retrieves the serialized JSON string, only valid after EndSerialization has
  // been called.
  std::string str() const { return str_; }

 private:
  void EnsureSeparator() {
    str_ += separator_;
    separator_ = ",\n";
  }

  std::string separator_;
  std::string str_;
};

// Simple serializer for C++ collections that produces a valid JSON array. This
// class uses JSONObjectSerializer to serialize each of the elements in the
// collection, providing the right wrapping syntax to make a valid JSON array.
// As with JSONObjectSerializer BeginSerialization *must* be called before any
// objects are added to array and EndSerialization *must* be called before the
// formatted string is retrieved by using str().
class JSONArraySerializer {
 public:
  // Begins the serialization process re-setting the instance to initial
  // formatting state.
  void BeginSerialization() {
    *this = {};

    str_ += "[\n";
  }

  // Writes a whole JSON object to the formatted array, the |element| and a
  // JSONObjectSerializer for it will be passed to the |func| functor in the
  // same way as JSONObjectSerializer does. |func| does not need to call
  // BeginSerialization nor it needs to call EndSerialization.
  template <typename T, typename TFunc>
  void WriteObject(T&& element, TFunc func) {
    EnsureSeparator();

    JSONObjectSerializer serializer;
    serializer.BeginSerialization();
    func(std::forward<T>(element), &serializer);
    serializer.EndSerialization();

    str_ += serializer.str();
  }

  // Finishes the serialization process for the array, ensuring that the
  // serializaed array object has the right syntax. This method *must* be called
  // before str() is called to retrieve the formatted string.
  void EndSerialization() { str_ += "\n]"; }

  std::string str() const { return str_; }

 private:
  void EnsureSeparator() {
    str_ += separator_;
    separator_ = ",\n";
  }

  std::string separator_;
  std::string str_;
};

// Helper for serializing the |array| collection by using |func| to serialize
// each of the elements in the collection. This function template will then
// return the resulting JSON string for the array.
template <typename T, typename TFunc>
std::string SerializeArrayToJSON(const T& array, TFunc func) {
  JSONArraySerializer serializer;
  serializer.BeginSerialization();
  for (const auto& element : array) {
    serializer.WriteObject(element, func);
  }
  serializer.EndSerialization();
  return serializer.str();
}
}  // namespace json

#endif  // KYTHE_RPC_CXX_JSON_HELPER_H_
