package protox

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protopath"
	"google.golang.org/protobuf/reflect/protorange"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// CloneProto is a generic typed version of proto.Clone from proto.
func CloneProto[T proto.Message](v T) T {
	return proto.Clone(v).(T)
}

// DiscardUnknownProto discards unknown fields in a proto message.
func DiscardUnknownProto(m proto.Message) error {
	return protorange.Range(m.ProtoReflect(), func(values protopath.Values) error {
		m, ok := values.Index(-1).Value.Interface().(protoreflect.Message)
		if ok && len(m.GetUnknown()) > 0 {
			m.SetUnknown(nil)
		}
		return nil
	})
}
