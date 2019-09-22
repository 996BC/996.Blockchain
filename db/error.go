package db

import (
	"errors"
	"fmt"
)

type ErrInvalidHeight struct {
	put    uint64
	expect uint64
}

func (i ErrInvalidHeight) Error() string {
	return fmt.Sprintf("an invalid height detected while putting blocks to db %d, expect %d",
		i.put, i.expect)
}

var ErrInternal = errors.New("internal error")

var ErrNotFound = errors.New("not found")
