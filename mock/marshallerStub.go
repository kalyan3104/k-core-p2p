package mock

// MarshallerStub -
type MarshallerStub struct {
	MarshalCalled   func(obj interface{}) ([]byte, error)
	UnmarshalCalled func(obj interface{}, buff []byte) error
}

// Marshal -
func (ms *MarshallerStub) Marshal(obj interface{}) ([]byte, error) {
	return ms.MarshalCalled(obj)
}

// Unmarshal -
func (ms *MarshallerStub) Unmarshal(obj interface{}, buff []byte) error {
	return ms.UnmarshalCalled(obj, buff)
}

// IsInterfaceNil -
func (ms *MarshallerStub) IsInterfaceNil() bool {
	return ms == nil
}
