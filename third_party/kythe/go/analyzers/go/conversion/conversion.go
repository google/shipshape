// Package conversion defines a conversion interface for proto messages.
package conversion

import "code.google.com/p/goprotobuf/proto"

// TODO(mataevs) - currently, this defines only an interface, conversion.
// Probably we will want to refactor the grok package and remove it.

// A Converter provides the ability to convert a proto message to
// a sequence of other proto messages. It is up to the implementing structure
// to define the logic for proto message conversion. If the conversion outputs
// a single proto message, a slice of one element should be returned.
// Returns an error if the conversion is not possible.
type Converter interface {
	Convert(proto.Message) ([]proto.Message, error)
}
