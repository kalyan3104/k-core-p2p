package mock

import (
	"fmt"

	"github.com/kalyan3104/k-core/marshal"
)

var _ marshal.Marshalizer = (*ProtoMarshallerMock)(nil)

// ProtoMarshallerMock implements marshaling with protobuf
type ProtoMarshallerMock struct {
}

// Marshal does the actual serialization of an object
// The object to be serialized must implement the gogoProtoObj interface
func (pmm *ProtoMarshallerMock) Marshal(obj interface{}) ([]byte, error) {
	if msg, ok := obj.(marshal.GogoProtoObj); ok {
		return msg.Marshal()
	}
	return nil, fmt.Errorf("%T, %w", obj, marshal.ErrMarshallingProto)
}

// Unmarshal does the actual deserialization of an object
// The object to be deserialized must implement the gogoProtoObj interface
func (pmm *ProtoMarshallerMock) Unmarshal(obj interface{}, buff []byte) error {
	if msg, ok := obj.(marshal.GogoProtoObj); ok {
		msg.Reset()
		return msg.Unmarshal(buff)
	}

	return fmt.Errorf("%T, %w", obj, marshal.ErrUnmarshallingProto)
}

// IsInterfaceNil returns true if there is no value under the interface
func (pmm *ProtoMarshallerMock) IsInterfaceNil() bool {
	return pmm == nil
}
