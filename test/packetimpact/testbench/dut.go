// Copyright 2020 The gVisor Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testbench

import (
	"context"
	"encoding/binary"
	"flag"
	"net"
	"strconv"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	grpcpb "gvisor.dev/gvisor/test/packetimpact/proto/stub_go_grpc_proto"
	pb "gvisor.dev/gvisor/test/packetimpact/proto/stub_go_proto"
)

var stubIP = flag.String("stub_ip", "", "ip address to listen to for UDP commands")
var stubPort = flag.Int("stub_port", 40000, "port to listen to for UDP commands")
var rpcTimeout = flag.Duration("rpc_timeout", 100*time.Millisecond, "gRPC timeout")
var rpcKeepalive = flag.Duration("rpc_keepalive", 10*time.Second, "gRPC keepalive")

// DUT communicates with the DUT to force it to make POSIX calls.
type DUT struct {
	t    *testing.T
	stub grpcpb.PosixClient
}

// NewDUT creates a new connection with the DUT over gRPC.
func NewDUT(t *testing.T) DUT {
	flag.Parse()
	stubAddress := *stubIP + ":" + strconv.Itoa(*stubPort)
	conn, err := grpc.Dial(stubAddress, grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{Timeout: *rpcKeepalive}))
	if err != nil {
		t.Fatalf("failed to grpc.Dial(%s): %s", stubAddress, err)
	}
	stub := grpcpb.NewPosixClient(conn)
	return DUT{
		t:    t,
		stub: stub,
	}
}

// SocketWithErrno calls socket on the DUT and returns the fd and errno.
func (dut *DUT) SocketWithErrno(domain, typ, proto int32) (int32, error) {
	req := pb.SocketRequest{
		Domain:   domain,
		Type:     typ,
		Protocol: proto,
	}
	ctx := context.Background()
	resp, err := dut.stub.Socket(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call Socket: %s", err)
	}
	return resp.GetFd(), syscall.Errno(resp.GetErrno_())
}

// Socket calls socket on the DUT and returns the file descriptor. If socket
// fails on the DUT, the test ends.
func (dut *DUT) Socket(domain, typ, proto int32) int32 {
	fd, err := dut.SocketWithErrno(domain, typ, proto)
	if fd < 0 {
		dut.t.Fatalf("failed to create socket: %s", err)
	}
	return fd
}

func (dut *DUT) sockaddrToProto(sa unix.Sockaddr) *pb.Sockaddr {
	switch s := sa.(type) {
	case *unix.SockaddrInet4:
		return &pb.Sockaddr{
			Sockaddr: &pb.Sockaddr_In{
				In: &pb.SockaddrIn{
					Family: unix.AF_INET,
					Port:   uint32(s.Port),
					Addr:   s.Addr[:],
				},
			},
		}
	case *unix.SockaddrInet6:
		return &pb.Sockaddr{
			Sockaddr: &pb.Sockaddr_In6{
				In6: &pb.SockaddrIn6{
					Family:   unix.AF_INET6,
					Port:     uint32(s.Port),
					Flowinfo: 0,
					ScopeId:  s.ZoneId,
					Addr:     s.Addr[:],
				},
			},
		}
	}
	dut.t.Fatalf("can't parse Sockaddr: %+v", sa)
	return nil
}

func (dut *DUT) protoToSockaddr(sa *pb.Sockaddr) unix.Sockaddr {
	switch s := sa.Sockaddr.(type) {
	case *pb.Sockaddr_In:
		ret := unix.SockaddrInet4{
			Port: int(s.In.GetPort()),
		}
		copy(ret.Addr[:], s.In.GetAddr())
		return &ret
	case *pb.Sockaddr_In6:
		ret := unix.SockaddrInet6{
			Port:   int(s.In6.GetPort()),
			ZoneId: s.In6.GetScopeId(),
		}
		copy(ret.Addr[:], s.In6.GetAddr())
	}
	dut.t.Fatalf("can't parse Sockaddr: %+v", sa)
	return nil
}

// BindWithErrno calls bind on the DUT.
func (dut *DUT) BindWithErrno(fd int32, sa unix.Sockaddr) (int32, error) {
	req := pb.BindRequest{
		Sockfd: fd,
		Addr:   dut.sockaddrToProto(sa),
	}
	ctx := context.Background()
	resp, err := dut.stub.Bind(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call Bind: %s", err)
	}
	return resp.GetRet(), syscall.Errno(resp.GetErrno_())
}

// Bind calls bind on the DUT and causes a fatal test failure if it doesn't succeed.
func (dut *DUT) Bind(fd int32, sa unix.Sockaddr) {
	ret, err := dut.BindWithErrno(fd, sa)
	if ret != 0 {
		dut.t.Fatalf("failed to bind socket: %s", err)
	}
}

// GetSockNameWithErrno calls getsockname on the DUT.
func (dut *DUT) GetSockNameWithErrno(sockfd int32) (int32, unix.Sockaddr, error) {
	req := pb.GetSockNameRequest{
		Sockfd: sockfd,
	}
	ctx := context.Background()
	resp, err := dut.stub.GetSockName(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call Bind: %s", err)
	}
	return resp.GetRet(), dut.protoToSockaddr(resp.GetAddr()), syscall.Errno(resp.GetErrno_())
}

// GetSockName calls getsockname on the DUT and causes a fatal test failure if
// it doens't succeed.
func (dut *DUT) GetSockName(sockfd int32) unix.Sockaddr {
	ret, sa, err := dut.GetSockNameWithErrno(sockfd)
	if ret != 0 {
		dut.t.Fatalf("failed to getsockname: %s", err)
	}
	return sa
}

// ListenWithErrno calls listen on the DUT.
func (dut *DUT) ListenWithErrno(sockfd, backlog int32) (int32, error) {
	req := pb.ListenRequest{
		Sockfd:  sockfd,
		Backlog: backlog,
	}
	ctx, cancel := context.WithTimeout(context.Background(), *rpcTimeout)
	defer cancel()
	resp, err := dut.stub.Listen(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call Listen: %s", err)
	}
	return resp.GetRet(), syscall.Errno(resp.GetErrno_())
}

// Listen calls listen on the DUT and causes a fatal test failure if it doesn't
// succeed.
func (dut *DUT) Listen(sockfd, backlog int32) {
	ret, err := dut.ListenWithErrno(sockfd, backlog)
	if ret != 0 {
		dut.t.Fatalf("failed to listen: %s", err)
	}
}

// AcceptWithErrno calls accept on the DUT.
func (dut *DUT) AcceptWithErrno(sockfd int32) (int32, unix.Sockaddr, error) {
	req := pb.AcceptRequest{
		Sockfd: sockfd,
	}
	ctx, cancel := context.WithTimeout(context.Background(), *rpcTimeout)
	defer cancel()
	resp, err := dut.stub.Accept(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call Accept: %s", err)
	}
	return resp.GetFd(), dut.protoToSockaddr(resp.GetAddr()), syscall.Errno(resp.GetErrno_())
}

// Accept calls accept on the DUT and causes a fatal test failure if it doesn't
// succeed.
func (dut *DUT) Accept(sockfd int32) (int32, unix.Sockaddr) {
	fd, sa, err := dut.AcceptWithErrno(sockfd)
	if fd < 0 {
		dut.t.Fatalf("failed to accept: %s", err)
	}
	return fd, sa
}

// SetSockOptWithErrno calls setsockopt on the DUT.
func (dut *DUT) SetSockOptWithErrno(sockfd, level, optname int32, optval []byte) (int32, error) {
	req := pb.SetSockOptRequest{
		Sockfd:  sockfd,
		Level:   level,
		Optname: optname,
		Optval:  optval,
	}
	ctx, cancel := context.WithTimeout(context.Background(), *rpcTimeout)
	defer cancel()
	resp, err := dut.stub.SetSockOpt(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call SetSockOpt: %s", err)
	}
	return resp.GetRet(), syscall.Errno(resp.GetErrno_())
}

// SetSockOpt calls setsockopt on the DUT and causes a fatal test failure if it
// doesn't succeed.
func (dut *DUT) SetSockOpt(sockfd, level, optname int32, optval []byte) {
	ret, err := dut.SetSockOptWithErrno(sockfd, level, optname, optval)
	if ret != 0 {
		dut.t.Fatalf("failed to SetSockOpt: %s", err)
	}
}

// SetSockOptTimevalWithErrno calls setsockopt with the timeval converted to
// bytes.
func (dut *DUT) SetSockOptTimevalWithErrno(fd, level, opt int, tv *unix.Timeval) (int32, error) {
	var optval [16]byte
	binary.LittleEndian.PutUint64(optval[0:8], uint64(tv.Sec))
	binary.LittleEndian.PutUint64(optval[8:16], uint64(tv.Usec))
	return dut.SetSockOptWithErrno(int32(fd), int32(level), int32(opt), optval[:])
}

// SetSockOptTimeval calls setsockopt on the DUT and causes a fatal test failure
// if it doesn't succeed.
func (dut *DUT) SetSockOptTimeval(fd, level, opt int, tv *unix.Timeval) {
	ret, err := dut.SetSockOptTimevalWithErrno(fd, level, opt, tv)
	if ret != 0 {
		dut.t.Fatalf("failed to SetSockOptTimeval: %s", err)
	}
}

// CloseWithErrno calls close on the DUT.
func (dut *DUT) CloseWithErrno(fd int32) (int32, error) {
	req := pb.CloseRequest{
		Fd: fd,
	}
	ctx, cancel := context.WithTimeout(context.Background(), *rpcTimeout)
	defer cancel()
	resp, err := dut.stub.Close(ctx, &req)
	if err != nil {
		dut.t.Fatalf("failed to call Close: %s", err)
	}
	return resp.GetRet(), syscall.Errno(resp.GetErrno_())
}

// Close calls close on the DUT and causes a fatal test failure if it doesn't
// succeed.
func (dut *DUT) Close(fd int32) {
	ret, err := dut.CloseWithErrno(fd)
	if ret != 0 {
		dut.t.Fatalf("failed to close: %s", err)
	}
}

// CreateListener makes a new TCP connection.  If it fails, the test ends.
func (dut *DUT) CreateListener(typ, proto, backlog int32) (int32, uint16) {
	addr := net.ParseIP(*remoteIP)
	var fd int32
	if addr.To4() != nil {
		fd = dut.Socket(unix.AF_INET, typ, proto)
		sa := unix.SockaddrInet4{}
		copy(sa.Addr[:], addr.To4())
		dut.Bind(fd, &sa)
	} else if addr.To16() != nil {
		fd = dut.Socket(unix.AF_INET6, typ, proto)
		sa := unix.SockaddrInet6{}
		copy(sa.Addr[:], addr.To16())
		dut.Bind(fd, &sa)
	} else {
		dut.t.Fatal("unknown ip addr type for remoteIP")
	}
	sa := dut.GetSockName(fd)
	var port int
	switch s := sa.(type) {
	case *unix.SockaddrInet4:
		port = s.Port
	case *unix.SockaddrInet6:
		port = s.Port
	default:
		dut.t.Fatalf("unknown sockaddr type from getsockname: %t", sa)
	}
	dut.Listen(fd, backlog)
	return fd, uint16(port)
}
