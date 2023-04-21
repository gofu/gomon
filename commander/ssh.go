package commander

import (
	"errors"
	"fmt"
	"github.com/gofu/gomon/sshpool"
)

type SSH struct {
	Pool *sshpool.Pool
	Hops []sshpool.Credentials
}

func (s SSH) Run(cmd Cmd) error {
	if s.Pool == nil {
		return fmt.Errorf("%T has nil %T", s, s.Pool)
	}
	return errors.New("not implemented")
}
