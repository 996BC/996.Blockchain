package utils

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	tcpConnQSize      = 128
	tcpListenTimeout  = 2 * time.Second
	tcpRecvQSize      = 1024
	tcpReadBufferSize = 2048
	tcpRecvTimeout    = 2 * time.Second
	tcpSendQSize      = 1024
)

type TCPServer interface {
	GetTCPAcceptConnChannel() <-chan TCPConn
	Addr() string
	Start() bool
	Stop()
}

func NewTCPServer(ip net.IP, port int) TCPServer {
	return &tcpServer{
		ip:         ip,
		port:       port,
		acceptConn: make(chan TCPConn, tcpConnQSize),
		lm:         NewLoop(1),
	}
}

type tcpServer struct {
	ip         net.IP
	port       int
	ln         *net.TCPListener
	acceptConn chan TCPConn
	lm         *LoopMode
}

func (s *tcpServer) GetTCPAcceptConnChannel() <-chan TCPConn {
	return s.acceptConn
}

func (s *tcpServer) Addr() string {
	return fmt.Sprintf("%s:%d", s.ip.String(), s.port)
}

func (s *tcpServer) Start() bool {
	lnAddr := &net.TCPAddr{
		IP:   s.ip,
		Port: s.port,
	}
	var err error
	if s.ln, err = net.ListenTCP("tcp", lnAddr); err != nil {
		logger.Warn("tcp server listen failed:%v\n", err)
		return false
	}

	go s.loop()
	s.lm.StartWorking()
	return true
}

func (s *tcpServer) Stop() {
	if s.lm.Stop() {
		s.ln.Close()
	}
}

func (s *tcpServer) loop() {
	s.lm.Add()
	defer s.lm.Done()

	for {
		select {
		case <-s.lm.D:
			return
		default:
			s.ln.SetDeadline(time.Now().Add(tcpListenTimeout))
			if conn, err := s.ln.AcceptTCP(); err == nil {
				select {
				case s.acceptConn <- newTCPConn(conn):
				default:
					logger.Warnln("tcp server listen accept queue full, drop connection")
					conn.Close()
				}
			}
		}
	}
}

type TCPConn interface {
	Send(data []byte)
	GetRecvChannel() <-chan []byte
	SetSplitFunc(func(received *bytes.Buffer) ([][]byte, error))
	SetDisconnectCb(func(addr net.Addr))
	RemoteAddr() net.Addr
	Disconnect()
}

func TCPConnectTo(ip net.IP, port int) (TCPConn, error) {
	targetAddr := &net.TCPAddr{
		IP:   ip,
		Port: port,
	}
	conn, err := net.DialTCP("tcp", nil, targetAddr)
	if err != nil {
		return nil, err
	}

	return newTCPConn(conn), nil
}

type tcpConn struct {
	conn         *net.TCPConn
	split        func(received *bytes.Buffer) ([][]byte, error)
	disconnectCb func(addr net.Addr)
	recvQ        chan []byte
	sendQ        chan []byte
	lm           *LoopMode
}

func newTCPConn(conn *net.TCPConn) TCPConn {
	result := &tcpConn{
		conn:  conn,
		recvQ: make(chan []byte, tcpRecvQSize),
		sendQ: make(chan []byte, tcpSendQSize),
		lm:    NewLoop(2),
	}
	result.start()

	return result
}

func (c *tcpConn) Send(data []byte) {
	c.sendQ <- data
}

func (c *tcpConn) GetRecvChannel() <-chan []byte {
	return c.recvQ
}

func (c *tcpConn) SetSplitFunc(f func(received *bytes.Buffer) ([][]byte, error)) {
	c.split = f
}

func (c *tcpConn) SetDisconnectCb(f func(addr net.Addr)) {
	c.disconnectCb = f
}

func (c *tcpConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *tcpConn) Disconnect() {
	c.stop()
}

func (c *tcpConn) start() {
	go c.recv()
	go c.send()
	c.lm.StartWorking()
}

func (c *tcpConn) stop() {
	if c.lm.Stop() {
		c.conn.Close()
		if c.disconnectCb != nil {
			c.disconnectCb(c.RemoteAddr())
		}
	}
}

func (c *tcpConn) recv() {
	c.lm.Add()
	defer c.lm.Done()

	buffer := new(bytes.Buffer)
	readBuf := make([]byte, tcpReadBufferSize)
	for {
		select {
		case <-c.lm.D:
			return
		default:
			c.conn.SetReadDeadline(time.Now().Add(tcpRecvTimeout))
			size, err := c.conn.Read(readBuf)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					break
				}
				if err == io.EOF {
					logger.Debug("connection closed by remote:%v\n", c.RemoteAddr())
				} else {
					logger.Warn("connection got unexpected err:%v\n", err)
				}
				go c.stop()
				return
			}

			buffer.Write(readBuf[:size])

			if c.split != nil {
				pkts, err := c.split(buffer)
				if err != nil {
					logger.Warn("tcp split packet err:%v\n", err)
					go c.stop()
					return
				}

				if pkts == nil {
					break
				}

				for _, pkt := range pkts {
					select {
					case c.recvQ <- pkt:
					default:
						logger.Warn("recvQ of %v is full, drop packet\n", c.RemoteAddr())
					}
				}
			}
		}
	}
}

func (c *tcpConn) send() {
	c.lm.Add()
	defer c.lm.Done()

	for {
		select {
		case <-c.lm.D:
			return
		case pkt := <-c.sendQ:
			_, err := c.conn.Write(pkt)
			if err != nil {
				logger.Warn("send to %v failed:%v , close connection\n",
					c.RemoteAddr(), err)
				go c.stop()
			}
		}
	}
}
