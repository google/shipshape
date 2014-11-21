package server

import (
	"net/http"
	"reflect"
	"testing"

	spb "third_party/kythe/proto/storage_proto"
)

// Verify that an endpoint satisfies http.Handler.
var _ http.Handler = Endpoint(nil)

// testService is used to verify that registration works as expected.
type testService struct{}

func (testService) ProtoMethod(ctx Context, in *spb.Entry, out chan<- *spb.VName) error {
	return nil
}

func (testService) Unrelated(bool) {}

// Register an test service with the *Service, or fail.
func (s *Service) mustRegister(t *testing.T) {
	if err := s.Register(testService{}); err != nil {
		t.Fatalf("Registering test service failed: %v", err)
	}
}

func TestBadService(t *testing.T) {
	type badService struct{} // A type with no compatible methods.

	var s Service
	err := s.Register(badService{})
	if err == nil {
		t.Error("Registered bad service with no error")
	} else {
		t.Logf("Got expected registration error: %v", err)
	}
}

func TestDefaultServiceName(t *testing.T) {
	var s Service
	s.mustRegister(t)
	if got, want := s.Name, "testService"; got != want {
		t.Errorf("Service name: got %q, want %q", got, want)
	}
}

func TestCustomServiceName(t *testing.T) {
	const serviceName = "Custom"
	s := Service{Name: serviceName}
	s.mustRegister(t)
	if got := s.Name; got != serviceName {
		t.Errorf("Service name: got %q, want %q", got, serviceName)
	}
}

func TestServiceMethods(t *testing.T) {
	var s Service
	s.mustRegister(t)

	m := make(map[string]*Method)
	for _, method := range s.Methods {
		m[method.Name] = method
	}

	if meth := m["Unrelated"]; meth != nil {
		t.Errorf("Registered unrelated method: %+v", meth)
	}

	meth := m["ProtoMethod"]
	if meth == nil {
		t.Error("Missing registration for ProtoMethod")
	} else {
		if got, want := meth.input, reflect.TypeOf(new(spb.Entry)); got != want {
			t.Errorf("Method %q input type: got %q, want %q", meth.Name, got, want)
		}
		if got, want := meth.output, reflect.TypeOf(new(spb.VName)); got != want {
			t.Errorf("Method %q output type: got %q, want %q", meth.Name, got, want)
		}
	}
}

func TestResolver(t *testing.T) {
	const serviceName = "TheLegionWillConquerAll"
	s := Service{Name: serviceName}
	s.mustRegister(t)

	e := Endpoint{&s}

	tests := []struct {
		service, method string
		err             error
	}{
		// The good service methods should be found.
		{serviceName, "ProtoMethod", nil},

		// Correctly diagnose an invalid service name.
		{"", "whatever", ErrNoSuchService},
		{"bogus", "whatever", ErrNoSuchService},

		// Correctly diagnose an invalid method name.
		{serviceName, "", ErrNoSuchMethod},
		{serviceName, "NoSuchMethod", ErrNoSuchMethod},

		// The ServiceInfo/List method is special.
		{serverInfoService, listMethod, ErrNoSuchService},
	}
	for _, test := range tests {
		m, err := e.Resolve(test.service, test.method)
		if err != test.err {
			t.Errorf("Resolve(%q, %q) error: got %v, want %v", test.service, test.method, err, test.err)
		}
		if test.err == nil && m == nil {
			t.Errorf("Resolve(%q, %q): no method returned", test.service, test.method)
		}
		if m != nil {
			t.Logf("Found method %+v", m)
		}
	}
}

// TODO(fromberger): Test the ServeHTTP method of Endpoint.
