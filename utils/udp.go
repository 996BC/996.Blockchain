package utils

import (
	"net"
	"time"
)

const (
	udpRecvQSize      = 1024
	udpRecvBufferSize = 1024
	udpRecvTimeout    = 2 * time.Second
	udpSendQSize      = 1024
)

type UDPPacket struct {
	Data []byte
	Addr *net.UDPAddr
}

type UDPServer interface {
	GetRecvChannel() <-chan *UDPPacket
	Send(packet *UDPPacket)
	Start() bool
	Stop()
}

func NewUDPServer(ip net.IP, port int) UDPServer {
	return &udpServer{
		ip:    ip,
		port:  port,
		recvQ: make(chan *UDPPacket, udpRecvQSize),
		sendQ: make(chan *UDPPacket, udpSendQSize),
		lm:    NewLoop(2),
	}
}

type udpServer struct {
	ip    net.IP
	port  int
	conn  *net.UDPConn
	recvQ chan *UDPPacket
	sendQ chan *UDPPacket
	lm    *LoopMode
}

func (u *udpServer) GetRecvChannel() <-chan *UDPPacket {
	return u.recvQ
}

func (u *udpServer) Send(packet *UDPPacket) {
	select {
	case u.sendQ <- packet:
	default:
		logger.Warnln("udp server sendQ is full, drop packet")
	}
}

func (u *udpServer) Start() bool {
	udpAddr := &net.UDPAddr{
		IP:   u.ip,
		Port: u.port,
	}
	var err error

	if u.conn, err = net.ListenUDP("udp", udpAddr); err != nil {
		logger.Warn("setup UDP server failed:%v\n", err)
		return false
	}

	go u.recv()
	go u.send()
	u.lm.StartWorking()
	return true
}

func (u *udpServer) Stop() {
	if u.lm.Stop() {
		u.conn.Close()
	}
}

func (u *udpServer) recv() {
	u.lm.Add()
	defer u.lm.Done()

	for {
		select {
		case <-u.lm.D:
			return
		default:
			packBuf := make([]byte, udpRecvBufferSize)
			u.conn.SetReadDeadline(time.Now().Add(udpRecvTimeout))
			n, Addr, err := u.conn.ReadFromUDP(packBuf)

			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					break
				}
				logger.Warn("udp server read err:%v\n", err)
				break
			}

			pkt := &UDPPacket{
				Data: packBuf[:n],
				Addr: Addr,
			}

			select {
			case u.recvQ <- pkt:
			default:
				logger.Warnln("udp server recvQ is full, drop packet")
			}

		}
	}
}

func (u *udpServer) send() {
	u.lm.Add()
	defer u.lm.Done()

	for {
		select {
		case <-u.lm.D:
			return
		case packet := <-u.sendQ:
			_, err := u.conn.WriteToUDP(packet.Data, packet.Addr)
			if err != nil {
				logger.Warn("udp server send to %v failed:%v, size:%d\n",
					packet.Addr, err, len(packet.Data))
			}
		}
	}
}
