package discover

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"github.com/996BC/996.Blockchain/utils"
)

type Address struct {
	IP   net.IP
	Port int32
}

func NewAddress(ipstr string, port int32) *Address {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return nil
	}
	return &Address{
		IP:   ip,
		Port: port,
	}
}

func UnmarshalAddress(data io.Reader) (*Address, error) {
	result := &Address{}

	var ipLen uint8
	if err := binary.Read(data, binary.BigEndian, &ipLen); err != nil {
		return nil, err
	}

	ipBuf := make([]byte, ipLen)
	if err := binary.Read(data, binary.BigEndian, ipBuf); err != nil {
		return nil, err
	}
	if err := result.IP.UnmarshalText(ipBuf); err != nil {
		return nil, err
	}
	if err := binary.Read(data, binary.BigEndian, &result.Port); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *Address) Marshal() []byte {
	result := new(bytes.Buffer)

	ipBytes, _ := a.IP.MarshalText()
	ipLen := utils.Uint8Len(ipBytes)
	binary.Write(result, binary.BigEndian, ipLen)
	binary.Write(result, binary.BigEndian, ipBytes)

	binary.Write(result, binary.BigEndian, a.Port)
	return result.Bytes()
}

func (a *Address) String() string {
	return fmt.Sprintf("%v:%d", a.IP, a.Port)
}
