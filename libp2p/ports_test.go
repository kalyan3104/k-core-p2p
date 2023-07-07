package libp2p_test

import (
	"errors"
	"fmt"
	"net"
	"testing"

	p2p "github.com/kalyan3104/k-core-p2p"
	"github.com/kalyan3104/k-core-p2p/libp2p"
	"github.com/stretchr/testify/assert"
)

func TestGetPort_InvalidStringShouldErr(t *testing.T) {
	t.Parallel()

	port, err := libp2p.GetPort("NaN", libp2p.CheckFreePort)

	assert.Equal(t, 0, port)
	assert.True(t, errors.Is(err, p2p.ErrInvalidPortsRangeString))
}

func TestGetPort_InvalidPortNumberShouldErr(t *testing.T) {
	t.Parallel()

	port, err := libp2p.GetPort("-1", libp2p.CheckFreePort)
	assert.Equal(t, 0, port)
	assert.True(t, errors.Is(err, p2p.ErrInvalidPortValue))
}

func TestGetPort_SinglePortShouldWork(t *testing.T) {
	t.Parallel()

	port, err := libp2p.GetPort("0", libp2p.CheckFreePort)
	assert.Equal(t, 0, port)
	assert.Nil(t, err)

	p := 3638
	port, err = libp2p.GetPort(fmt.Sprintf("%d", p), libp2p.CheckFreePort)
	assert.Equal(t, p, port)
	assert.Nil(t, err)
}

func TestCheckFreePort_InvalidStartingPortShouldErr(t *testing.T) {
	t.Parallel()

	port, err := libp2p.GetPort("NaN-10000", libp2p.CheckFreePort)
	assert.Equal(t, 0, port)
	assert.Equal(t, p2p.ErrInvalidStartingPortValue, err)

	port, err = libp2p.GetPort("1024-10000", libp2p.CheckFreePort)
	assert.Equal(t, 0, port)
	assert.True(t, errors.Is(err, p2p.ErrInvalidValue))
}

func TestCheckFreePort_InvalidEndingPortShouldErr(t *testing.T) {
	t.Parallel()

	port, err := libp2p.GetPort("10000-NaN", libp2p.CheckFreePort)
	assert.Equal(t, 0, port)
	assert.Equal(t, p2p.ErrInvalidEndingPortValue, err)
}

func TestGetPort_EndPortLargerThanSendPort(t *testing.T) {
	t.Parallel()

	port, err := libp2p.GetPort("10000-9999", libp2p.CheckFreePort)
	assert.Equal(t, 0, port)
	assert.Equal(t, p2p.ErrEndPortIsSmallerThanStartPort, err)
}

func TestGetPort_RangeOfOneShouldWork(t *testing.T) {
	t.Parallel()

	port := 5000
	numCall := 0
	handler := func(p int) error {
		if p != port {
			assert.Fail(t, fmt.Sprintf("should have been %d", port))
		}
		numCall++
		return nil
	}

	result, err := libp2p.GetPort(fmt.Sprintf("%d-%d", port, port), handler)
	assert.Nil(t, err)
	assert.Equal(t, port, result)
}

func TestGetPort_RangeOccupiedShouldErrorShouldWork(t *testing.T) {
	t.Parallel()

	portStart := 5000
	portEnd := 10000
	portsTried := make(map[int]struct{})
	expectedErr := errors.New("expected error")
	handler := func(p int) error {
		portsTried[p] = struct{}{}
		return expectedErr
	}

	result, err := libp2p.GetPort(fmt.Sprintf("%d-%d", portStart, portEnd), handler)

	assert.True(t, errors.Is(err, p2p.ErrNoFreePortInRange))
	assert.Equal(t, portEnd-portStart+1, len(portsTried))
	assert.Equal(t, 0, result)
}

func TestCheckFreePort_PortZeroAlwaysWorks(t *testing.T) {
	err := libp2p.CheckFreePort(0)

	assert.Nil(t, err)
}

func TestCheckFreePort_InvalidPortShouldErr(t *testing.T) {
	err := libp2p.CheckFreePort(-1)

	assert.NotNil(t, err)
}

func TestCheckFreePort_OccupiedPortShouldErr(t *testing.T) {
	// 1. get a free port from OS, open a TCP listner
	// 2. get the allocated port
	// 3. test if that port is occupied
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		assert.Fail(t, err.Error())
		return
	}

	port := l.Addr().(*net.TCPAddr).Port

	fmt.Printf("testing port %d\n", port)
	err = libp2p.CheckFreePort(port)
	assert.NotNil(t, err)

	_ = l.Close()
}
