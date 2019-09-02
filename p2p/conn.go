package p2p

import (
	"github.com/996BC/996.Blockchain/p2p/peer"
	"github.com/996BC/996.Blockchain/utils"
)

type recvHandler = func(peer string, protocolID uint8, data []byte)

type conn struct {
	node    *Node
	p       *peer.Peer
	conn    utils.TCPConn
	ec      codec
	handler recvHandler
	lm      *utils.LoopMode
}

func newConn(p *peer.Peer, nc utils.TCPConn, ec codec, handler recvHandler) *conn {
	c := &conn{
		p:       p,
		conn:    nc,
		ec:      ec,
		handler: handler,
		lm:      utils.NewLoop(1),
	}

	return c
}

func (c *conn) start() {
	go c.loop()
	c.lm.StartWorking()
}

func (c *conn) stop() {
	if c.lm.Stop() {
		c.conn.Disconnect()
	}
}

func (c *conn) loop() {
	c.lm.Add()
	defer c.lm.Done()

	recvC := c.conn.GetRecvChannel()
	for {
		select {
		case <-c.lm.D:
			return
		case pkt := <-recvC:
			var ok bool
			var payload []byte
			var protocolID uint8
			if ok, payload, protocolID = verifyTCPPacket(pkt); !ok {
				logger.Warnln("verify packet failed, close connection")
				go c.stop()
				break
			}

			plaintext, err := c.ec.decrypt(payload)
			if err != nil {
				logger.Warnln("decrypt packet failed, close connection")
				go c.stop()
				break
			}
			c.handler(c.p.ID, protocolID, plaintext)
		}
	}
}

func (c *conn) send(protocolID uint8, data []byte) {
	cipherText, err := c.ec.encrypt(data)
	if err != nil {
		logger.Warn("encrypt payload failed, close connection")
		go c.stop()
		return
	}

	pkt := buildTCPPacket(cipherText, protocolID)
	c.conn.Send(pkt)
}
