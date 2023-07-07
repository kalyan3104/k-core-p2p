package discovery_test

import (
	"testing"

	"github.com/kalyan3104/k-core-p2p/libp2p/discovery"
	"github.com/kalyan3104/k-core/core/check"
	"github.com/stretchr/testify/assert"
)

func TestNilDiscoverer(t *testing.T) {
	t.Parallel()

	nd := discovery.NewNilDiscoverer()

	assert.False(t, check.IfNil(nd))
	assert.Equal(t, discovery.NullName, nd.Name())
	assert.Nil(t, nd.Bootstrap())
}
