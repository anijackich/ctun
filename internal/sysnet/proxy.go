package sysnet

type Proxy struct {
	Host   string
	Port   int
	Bypass []string
}

var (
	LocalBypass = []string{"127.0.0.1", "::1", "localhost"}

	LANBypass = []string{"*.local", "169.254/16", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
)

func NewProxy(statePath string) *Manager[Proxy] {
	return &Manager[Proxy]{statePath: statePath, backend: newProxyBackend(execRunner)}
}
