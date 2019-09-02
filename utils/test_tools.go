package utils

import (
	"bytes"
	"fmt"
	"math/big"
	"net"
)

func TCheckString(prefix string, expect string, result string) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckBytes(prefix string, expect []byte, result []byte) error {
	if !bytes.Equal(expect, result) {
		return fmt.Errorf("%s check failed:expect %X, result %X",
			prefix, expect, result)
	}
	return nil
}

func TCheckAddr(prefix string, expect net.Addr, result net.Addr) error {
	if expect.String() != result.String() {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckIP(prefix string, expect net.IP, result net.IP) error {
	if !expect.Equal(result) {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckInt(prefix string, expect int, result int) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckInt32(prefix string, expect int32, result int32) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckInt64(prefix string, expect int64, result int64) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckUint8(prefix string, expect uint8, result uint8) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckUint16(prefix string, expect uint16, result uint16) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckUint32(prefix string, expect uint32, result uint32) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckUint64(prefix string, expect uint64, result uint64) error {
	if expect != result {
		return errorf(prefix, expect, result)
	}
	return nil
}

func TCheckBigInt(prefix string, expect *big.Int, result *big.Int) error {
	if expect.Cmp(result) != 0 {
		return errorf(prefix, expect, result)
	}
	return nil
}

func errorf(prefix string, expect interface{}, result interface{}) error {
	return fmt.Errorf("%s check failed:expect %v, result %v",
		prefix, expect, result)
}
