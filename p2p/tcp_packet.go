package p2p

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"

	"github.com/996BC/996.Blockchain/utils"
)

/*
+-------------+-----------+--------------+
|   Length    |    CRC    |    Protocol  |
+-------------+-----------+--------------+
|                Payload                 |
+----------------------------------------+

(bytes)
Length		4
CRC			4
Protocol	1
*/

const tcpHeaderSize = 9

func buildTCPPacket(payload []byte, protocolID uint8) []byte {
	length := utils.Uint32Len(payload)
	crc := crc32.ChecksumIEEE(payload)

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, length)
	binary.Write(buf, binary.BigEndian, crc)
	binary.Write(buf, binary.BigEndian, protocolID)
	buf.Write(payload)

	return buf.Bytes()
}

func splitTCPStream(received *bytes.Buffer) ([][]byte, error) {
	var length uint32
	var packets [][]byte

	for received.Len() > tcpHeaderSize {
		// peeker' reading has no effect on received
		peeker := bytes.NewReader(received.Bytes())
		binary.Read(peeker, binary.BigEndian, &length)

		packetLen := tcpHeaderSize + length
		if received.Len() < int(packetLen) {
			break
		}

		packet := make([]byte, packetLen)
		if _, err := received.Read(packet); err != nil {
			return nil, err
		}

		packets = append(packets, packet)
	}

	return packets, nil
}

func verifyTCPPacket(packet []byte) (bool, []byte, uint8) {
	var length uint32
	var crc uint32
	var protocolID uint8

	packetReader := bytes.NewReader(packet)
	binary.Read(packetReader, binary.BigEndian, &length)
	binary.Read(packetReader, binary.BigEndian, &crc)
	binary.Read(packetReader, binary.BigEndian, &protocolID)

	payload := make([]byte, length)
	packetReader.Read(payload)

	checkCrc := crc32.ChecksumIEEE(payload)
	if crc != checkCrc {
		return false, nil, 0
	}

	return true, payload, protocolID
}
