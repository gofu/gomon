package gomon

type ServerConfig struct {
	Addr          string
	PProfURL      string
	Local, Remote EnvConfig
}

type EnvConfig struct {
	Root, GoRoot, GoPath string
}

func (c EnvConfig) WithDefaults(conf EnvConfig) EnvConfig {
	if len(c.Root) == 0 {
		c.Root = conf.Root
	}
	if len(c.GoRoot) == 0 {
		c.GoRoot = conf.GoRoot
	}
	if len(c.GoPath) == 0 {
		c.GoPath = conf.GoPath
	}
	return c
}
