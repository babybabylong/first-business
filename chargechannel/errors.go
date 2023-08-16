package chargechannel

import "errors"

var (
	ErrNotSupported = errors.New(`不支持的操作`)
)

func IsNotSupported(err error) bool {
	return errors.As(err, &ErrNotSupported)
}
