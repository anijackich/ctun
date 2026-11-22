package sysnet

func NewDNS(statePath string) *Manager[[]string] {
	return &Manager[[]string]{statePath: statePath, backend: newDNSBackend(execRunner)}
}
