package db

import "fmt"

type InvalidHeight struct {
	put    uint64
	expect uint64
}

func (i InvalidHeight) Error() string {
	return fmt.Sprintf("invalid height detect while putting blocks to db %d, expect %d",
		i.put, i.expect)
}

type InternalError struct {
	error
}

func (i InternalError) Error() string {
	return "db internal error"
}

type NotFound struct {
}

func (n NotFound) Error() string {
	return "db not found"
}
