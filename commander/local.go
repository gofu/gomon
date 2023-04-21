package commander

import "os/exec"

type Local struct{}

func (Local) Run(cmd Cmd) error {
	c := exec.Command(cmd.Path, cmd.Args...)
	c.Env = cmd.Env
	c.Dir = cmd.Dir
	c.Stdin = cmd.Stdin
	c.Stdout = cmd.Stdout
	c.Stderr = cmd.Stderr
	return c.Run()
}
