package yamux

import (
	"io"
	"net"
	"sync"
	"testing"
)

func BenchmarkPing(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	for i := 0; i < b.N; i++ {
		rtt, err := client.Ping()
		if err != nil {
			b.Fatalf("err: %v", err)
		}
		if rtt == 0 {
			b.Fatalf("bad: %v", rtt)
		}
	}
}

func BenchmarkAccept(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	go func() {
		for i := 0; i < b.N; i++ {
			stream, err := server.AcceptStream()
			if err != nil {
				return
			}
			stream.Close()
		}
	}()

	for i := 0; i < b.N; i++ {
		stream, err := client.Open()
		if err != nil {
			b.Fatalf("err: %v", err)
		}
		stream.Close()
	}
}

func BenchmarkSendRecv(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()

	sendBuf := make([]byte, 512)
	recvBuf := make([]byte, 512)

	doneCh := make(chan struct{})
	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			return
		}
		defer stream.Close()
		for i := 0; i < b.N; i++ {
			if _, err := stream.Read(recvBuf); err != nil {
				b.Fatalf("err: %v", err)
			}
		}
		close(doneCh)
	}()

	stream, err := client.Open()
	if err != nil {
		b.Fatalf("err: %v", err)
	}
	defer stream.Close()
	for i := 0; i < b.N; i++ {
		if _, err := stream.Write(sendBuf); err != nil {
			b.Fatalf("err: %v", err)
		}
	}
	<-doneCh
}

func BenchmarkSendRecvLarge(b *testing.B) {
	client, server := testClientServer()
	defer client.Close()
	defer server.Close()
	const sendSize = 512 * 1024 * 1024
	const recvSize = 4 * 1024

	sendBuf := make([]byte, sendSize)
	recvBuf := make([]byte, recvSize)

	b.ResetTimer()
	recvDone := make(chan struct{})

	go func() {
		stream, err := server.AcceptStream()
		if err != nil {
			return
		}
		defer stream.Close()
		for i := 0; i < b.N; i++ {
			for j := 0; j < sendSize/recvSize; j++ {
				if _, err := stream.Read(recvBuf); err != nil {
					b.Fatalf("err: %v", err)
				}
			}
		}
		close(recvDone)
	}()

	stream, err := client.Open()
	if err != nil {
		b.Fatalf("err: %v", err)
	}
	defer stream.Close()
	for i := 0; i < b.N; i++ {
		if _, err := stream.Write(sendBuf); err != nil {
			b.Fatalf("err: %v", err)
		}
	}
	<-recvDone
}

func BenchmarkConnYamux(b *testing.B) {
	cs, ss, err := getSmuxStreamPair()
	if err != nil {
		b.Fatal(err)
	}
	defer cs.Close()
	defer ss.Close()
	bench(b, cs, ss)
}

func BenchmarkConnTCP(b *testing.B) {
	cs, ss, err := getTCPConnectionPair()
	if err != nil {
		b.Fatal(err)
	}
	defer cs.Close()
	defer ss.Close()
	bench(b, cs, ss)
}

func getSmuxStreamPair() (*Stream, *Stream, error) {
	c1, c2, err := getTCPConnectionPair()
	if err != nil {
		return nil, nil, err
	}

	s, err := Server(c2, nil)
	if err != nil {
		return nil, nil, err
	}
	c, err := Client(c1, nil)
	if err != nil {
		return nil, nil, err
	}
	var ss *Stream
	done := make(chan error)
	go func() {
		var rerr error
		ss, rerr = s.AcceptStream()
		done <- rerr
		close(done)
	}()
	cs, err := c.OpenStream()
	if err != nil {
		return nil, nil, err
	}
	err = <-done
	if err != nil {
		return nil, nil, err
	}

	return cs, ss, nil
}

func getTCPConnectionPair() (net.Conn, net.Conn, error) {
	lst, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return nil, nil, err
	}
	defer lst.Close()

	var conn0 net.Conn
	var err0 error
	done := make(chan struct{})
	go func() {
		conn0, err0 = lst.Accept()
		close(done)
	}()

	conn1, err := net.Dial("tcp", lst.Addr().String())
	if err != nil {
		return nil, nil, err
	}

	<-done
	if err0 != nil {
		return nil, nil, err0
	}
	return conn0, conn1, nil
}

func bench(b *testing.B, rd io.Reader, wr io.Writer) {
	buf := make([]byte, 128*1024)
	buf2 := make([]byte, 128*1024)
	b.SetBytes(128 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		count := 0
		for {
			n, _ := rd.Read(buf2)
			count += n
			if count == 128*1024*b.N {
				return
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		wr.Write(buf)
	}
	wg.Wait()
}
