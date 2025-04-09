package net

import (
	"errors"
	"fmt"
	"io"
	"net/netip"
	"sync"
	"time"

	"github.com/Azure/iot-operations-sdks/go/wasm/internal/wasi/io/streams"
	instancenetwork "github.com/Azure/iot-operations-sdks/go/wasm/internal/wasi/sockets/instance-network"
	ipnamelookup "github.com/Azure/iot-operations-sdks/go/wasm/internal/wasi/sockets/ip-name-lookup"
	"github.com/Azure/iot-operations-sdks/go/wasm/internal/wasi/sockets/network"
	"github.com/Azure/iot-operations-sdks/go/wasm/internal/wasi/sockets/tcp"
	tcpcreatesocket "github.com/Azure/iot-operations-sdks/go/wasm/internal/wasi/sockets/tcp-create-socket"
	"go.bytecodealliance.org/cm"
	"tinygo.org/x/drivers/netdev"
)

type (
	wasiNetdev struct {
		network network.Network
		sockets map[int]*wasiSocket
		counter int
		freq    time.Duration
		mu      sync.RWMutex
	}

	wasiSocket struct {
		*tcp.TCPSocket
		in  *streams.InputStream
		out *streams.OutputStream
		fd  int
	}
)

func UseWASI() {
	netdev.UseNetdev(&wasiNetdev{
		network: instancenetwork.InstanceNetwork(),
		sockets: map[int]*wasiSocket{},
		freq:    time.Millisecond,
	})
}

func (n *wasiNetdev) GetHostByName(name string) (netip.Addr, error) {
	str, err := res(ipnamelookup.ResolveAddresses(n.network, name))
	if err != nil {
		return netip.Addr{}, err
	}
	opt, err := poll(n.freq, str.ResolveNextAddress)
	if err != nil {
		return netip.Addr{}, err
	}
	if opt.None() {
		return netip.Addr{}, netdev.ErrHostUnknown
	}
	addr := opt.Some().IPv4()
	if addr == nil {
		return netip.Addr{}, netdev.ErrFamilyNotSupported
	}
	return netip.AddrFrom4(*addr), nil
}

func (n *wasiNetdev) Addr() (netip.Addr, error) {
	return netip.Addr{}, netdev.ErrNotSupported
}

func (n *wasiNetdev) Socket(domain int, stype int, protocol int) (int, error) {
	if domain != netdev.AF_INET {
		return -1, netdev.ErrFamilyNotSupported
	}

	if stype != netdev.SOCK_STREAM {
		return -1, netdev.ErrProtocolNotSupported
	}

	if protocol != netdev.IPPROTO_TCP {
		return -1, netdev.ErrProtocolNotSupported
	}

	sock, err := res(
		tcpcreatesocket.CreateTCPSocket(network.IPAddressFamilyIPv4),
	)
	if err != nil {
		return -1, err
	}

	return n.add(&wasiSocket{TCPSocket: sock}), nil
}

func (n *wasiNetdev) Bind(sockfd int, ip netip.AddrPort) error {
	s, err := n.get(sockfd)
	if err != nil {
		return err
	}
	if err := void(res(s.StartBind(n.network, ap2sa(ip)))); err != nil {
		return err
	}
	return void(poll(n.freq, s.FinishBind))
}

func (n *wasiNetdev) Connect(sockfd int, host string, ip netip.AddrPort) error {
	s, err := n.get(sockfd)
	if err != nil {
		return err
	}
	if err := void(res(s.StartConnect(n.network, ap2sa(ip)))); err != nil {
		return err
	}
	tup, err := poll(n.freq, s.FinishConnect)
	if err != nil {
		return err
	}
	s.in = &tup.F0
	s.out = &tup.F1
	return nil
}

func (n *wasiNetdev) Listen(sockfd int, _ int) error {
	s, err := n.get(sockfd)
	if err != nil {
		return err
	}
	if err := void(res(s.StartListen())); err != nil {
		return err
	}
	return void(poll(n.freq, s.FinishListen))
}

func (n *wasiNetdev) Accept(sockfd int) (int, netip.AddrPort, error) {
	s, err := n.get(sockfd)
	if err != nil {
		return 0, netip.AddrPort{}, err
	}
	tup, err := res(s.Accept())
	if err != nil {
		return 0, netip.AddrPort{}, err
	}
	sa, err := res(tup.F0.LocalAddress())
	if err != nil {
		return 0, netip.AddrPort{}, err
	}
	ip4 := sa.IPv4()
	if ip4 == nil {
		return 0, netip.AddrPort{}, netdev.ErrFamilyNotSupported
	}
	return n.add(&wasiSocket{
		TCPSocket: &tup.F0,
		in:        &tup.F1,
		out:       &tup.F2,
	}), netip.AddrPortFrom(netip.AddrFrom4(ip4.Address), ip4.Port), nil
}

func (n *wasiNetdev) Send(
	sockfd int,
	buf []byte,
	_ int,
	_ time.Time,
) (int, error) {
	s, err := n.get(sockfd)
	if err != nil {
		return -1, err
	}
	if s.out == nil {
		return -1, io.EOF
	}
	if err := void(res(
		s.out.BlockingWriteAndFlush(cm.ToList(buf)),
	)); err != nil {
		return -1, err
	}
	return len(buf), nil
}

func (n *wasiNetdev) Recv(
	sockfd int,
	buf []byte,
	_ int,
	_ time.Time,
) (int, error) {
	s, err := n.get(sockfd)
	if err != nil {
		return -1, err
	}
	if s.in == nil {
		return -1, io.EOF
	}
	list, err := res(s.in.BlockingRead(uint64(len(buf))))
	if err != nil {
		return -1, err
	}
	copy(buf, list.Slice())
	return int(list.Len()), nil
}

func (n *wasiNetdev) Close(sockfd int) error {
	s, err := n.get(sockfd)
	if err != nil {
		return err
	}

	if s.in != nil || s.out != nil {
		if err := void(res(s.Shutdown(tcp.ShutdownTypeBoth))); err != nil {
			return err
		}
	}

	n.del(s)
	return nil
}

func (n *wasiNetdev) SetSockOpt(int, int, int, any) error {
	return netdev.ErrNotSupported
}

func (n *wasiNetdev) get(sockfd int) (*wasiSocket, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if socket, ok := n.sockets[sockfd]; ok {
		return socket, nil
	}
	return nil, netdev.ErrInvalidSocketFd
}

func (n *wasiNetdev) add(s *wasiSocket) int {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.counter++
	s.fd = n.counter
	n.sockets[n.counter] = s
	return n.counter
}

func (n *wasiNetdev) del(s *wasiSocket) {
	n.mu.Lock()
	defer n.mu.Unlock()

	delete(n.sockets, s.fd)
}

func res[S, V any, E fmt.Stringer](r cm.Result[S, V, E]) (*V, error) {
	if r.IsErr() {
		return nil, errors.New((*r.Err()).String())
	}
	return r.OK(), nil
}

func poll[S, V any](
	freq time.Duration,
	cb func() cm.Result[S, V, network.ErrorCode],
) (*V, error) {
	for {
		r := cb()
		if !r.IsErr() || *r.Err() != network.ErrorCodeWouldBlock {
			return res(r)
		}
		<-time.After(freq)
	}
}

func void(_ *struct{}, err error) error {
	return err
}

func ap2sa(ap netip.AddrPort) network.IPSocketAddress {
	return network.IPSocketAddressIPv4(network.IPv4SocketAddress{
		Address: network.IPv4Address(ap.Addr().As4()),
		Port:    ap.Port(),
	})
}
