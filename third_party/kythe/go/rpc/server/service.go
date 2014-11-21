// Package server provides an interface for defining KRPC services.
//
// Example: Defining a handler.
//
//   type EchoService struct{}
//   func (EchoService) Echo(ctx krpc.Context, in string, out chan<- string) error {
//     out <- in
//     return nil
//   }
//
// Example: Registering a handler with a service.
//
//   var s server.Service
//   s.Register(EchoService{}) // s.Name is now "EchoService"
//   http.Handle("/", server.Endpoint{&s})
//
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"reflect"
	"sort"
	"sync"

	"third_party/kythe/go/rpc/protocol"
)

// A Service represents a named collection of remotely-callable methods.
type Service struct {
	// Name is the name presented by the service to callers.
	Name string `json:"name"`

	// The set of methods registered with the service.
	Methods []*Method `json:"methods"`
}

// Method returns the *Method corresponding to the given name, or nil if there
// is no such method defined on the service.  Names are case-sensitive.
func (s *Service) Method(name string) *Method {
	for _, m := range s.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

// ErrNoSuchService indicates that a requested service was not found.
var ErrNoSuchService = errors.New("no such service")

// ErrNoSuchMethod indicates that a requested method was not found.
var ErrNoSuchMethod = errors.New("no such method")

// Register registers the methods on the given handler with the Service.  A
// handler is a value of a type T that exposes methods having the following
// general signatures:
//
//   // streaming method in which results are returned as they are available
//   func (T) Name(Context, Input, chan<- Output) error
//
//   // single-result, blocking method
//   func (T) Name(Context, Input) (Output, error)
//
// Methods not matching these signatures are ignored, as are any unexported
// methods even if they do match this signature.  The service wrapper will
// ensure that the output channel is closed once the method returns.
//
// It is an error if the handler has no methods defined; it is also an error if
// more than one method with the same name is registered.
func (s *Service) Register(handler interface{}) error {
	name, ms := checkInterface(handler)
	if len(ms) == 0 {
		return fmt.Errorf("no service methods defined on %v", handler)
	}

	// If the service hasn't already got a name assigned, use the name of the
	// receiver type.
	if s.Name == "" {
		s.Name = name
	}

	// Check that all the methods to be registered have unique names.
	for _, old := range s.Methods {
		for _, m := range ms {
			if m.Name == old.Name {
				return fmt.Errorf("duplicate method name: %q", m.Name)
			}
		}
	}

	s.Methods = append(s.Methods, ms...)
	return nil
}

// decode unpacks a slice of bytes into a value of the appropriate type.
func decode(in []byte, t reflect.Type) (interface{}, error) {
	var (
		v   interface{}
		err error
	)
	if t.Kind() == reflect.Ptr {
		v = reflect.New(t.Elem()).Interface()
		if in != nil {
			err = json.Unmarshal(in, v)
		}
	} else {
		w := reflect.New(t)
		if in != nil {
			err = json.Unmarshal(in, w.Interface())
		}
		v = w.Elem().Interface()
	}
	return v, err
}

// A Method represents a single method on a service.
type Method struct {
	Name   string   `json:"name"` // The name of the method
	Params []string `json:"params"`
	Stream bool     `json:"stream,omitempty"`

	input  reflect.Type
	output reflect.Type

	rcvr reflect.Value // The method's receiver
	fun  reflect.Value // The method's function
}

// Invoke calls the method's handler with the given input.  Each encoded output
// from the method is passed to out, and the final result is returned.  Invoke
// will panic if it is unable to encode an output from the handler.
//
// Returns ErrNoSuchMethod if m == nil.
func (m *Method) Invoke(ctx Context, in []byte, out func([]byte)) error {
	if m == nil {
		return ErrNoSuchMethod
	}
	inValue, err := decode(in, m.input)
	if err != nil {
		return &protocol.Error{
			Code:    protocol.ErrorInvalidParams,
			Message: fmt.Sprintf("unable to decode params: %v", err),
		}
	}

	if m.Stream {
		outChan := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, m.output), 0)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				v, ok := outChan.Recv()
				if !ok {
					break
				}
				bits, err := json.Marshal(v.Interface())
				if err != nil {
					panic(err) // Not expected to occur
				}
				out(bits)
			}
		}()

		err = m.callStream(ctx, inValue, outChan)
		outChan.Close()
		wg.Wait()
		return err
	}

	// Non-streaming case
	v, err := m.call(ctx, inValue)
	if err != nil {
		return err
	}
	bits, err := json.Marshal(v)
	if err != nil {
		panic(err) // Not expected to occur
	}
	out(bits)
	return nil
}

func (m *Method) callStream(ctx Context, in interface{}, out reflect.Value) error {
	if r := m.fun.Call([]reflect.Value{
		m.rcvr,
		reflect.ValueOf(ctx),
		reflect.ValueOf(in), out,
	})[0].Interface(); r != nil {
		return r.(error)
	}
	return nil
}

func (m *Method) call(ctx Context, in interface{}) (interface{}, error) {
	res := m.fun.Call([]reflect.Value{
		m.rcvr,
		reflect.ValueOf(ctx),
		reflect.ValueOf(in),
	})
	if len(res) != 2 {
		panic("unexpected number of call results")
	}
	if err := res[1].Interface(); err != nil {
		return nil, err.(error)
	}
	return res[0].Interface(), nil
}

// checkInterface extracts the name of t, along with a slice of all the methods
// on t that are suitable for use as RPC handlers.
func checkInterface(t interface{}) (string, []*Method) {
	var methods []*Method
	pType := reflect.TypeOf(t)
	for i := 0; i < pType.NumMethod(); i++ {
		m, err := checkMethod(t, pType.Method(i))
		if err != nil {
			continue
		}
		methods = append(methods, m)
	}
	return pType.Name(), methods
}

// Some common types used in checking method signatures.  This indirection is
// necessary because reflect can't infer an interface type from a nil.
var (
	errType = reflect.TypeOf((*error)(nil)).Elem()
	ctxType = reflect.TypeOf((*Context)(nil)).Elem()
)

func structOrStructPtr(t reflect.Type) bool {
	return t.Kind() == reflect.Struct || (t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct)
}

// checkMethod verifies whether the specified method has a valid RPC handler
// signature:
//   func(R, Context, IN, chan<- OUT) error
//   func(R, Context, IN) (OUT, error)
//
// If so, returns a filled-in *Method; otherwise returns an error.
func checkMethod(rcvr interface{}, m reflect.Method) (*Method, error) {
	if !ast.IsExported(m.Name) {
		return nil, errors.New("method is not exported")
	}
	t := m.Type

	if t.Kind() != reflect.Func || t.NumIn() < 3 || t.In(1) != ctxType || !structOrStructPtr(t.In(2)) {
		return nil, errors.New("invalid method signature")
	}

	var (
		stream bool
		out    reflect.Type
	)
	switch {
	case t.NumIn() == 4 && t.NumOut() == 1 && t.Out(0) == errType:
		// Signature: (receiver, Context, input, chan output) => error
		out = t.In(3)
		if out.Kind() != reflect.Chan || out.ChanDir()&reflect.SendDir == 0 {
			return nil, errors.New("invalid output type")
		}
		stream = true
		out = out.Elem()
	case t.NumIn() == 3 && t.NumOut() == 2 && t.Out(1) == errType:
		// Signature: (receiver, Context, input) => (output, error)
		out = t.Out(0)
	}

	in := t.In(2)
	if in.Kind() == reflect.Ptr {
		in = in.Elem()
	}

	params := []string{}
	for i := 0; i < in.NumField(); i++ {
		params = append(params, in.Field(i).Name)
	}
	sort.Strings(params)

	return &Method{
		Name:   m.Name,
		Params: params,
		Stream: stream,

		input:  t.In(2),
		output: out,

		rcvr: reflect.ValueOf(rcvr),
		fun:  m.Func,
	}, nil
}

// A Context provides access to a virtual collection of request-specific
// key/value metadata for an RPC call.
type Context interface {
	// Get fetches the value associated with key, or "" if none is defined.
	Get(key string) string

	// Set associates key with the given value in the context, replacing any
	// previously-defined value for that key.
	Set(key, value string)

	// Del removes any value associated with key in the context.
	Del(key string)
}

// A Map implements the Context interface on a string-to-string map.
type Map map[string]string

// Get returns the value associated with a given key with the default value
// being the empty string.
func (m Map) Get(key string) string { return m[key] }

// Set overrides the value associated with a given key.
func (m Map) Set(key, value string) { m[key] = value }

// Del removes the value associated with a given key.
func (m Map) Del(key string) { delete(m, key) }
