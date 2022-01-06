// +build windows,npm

package http

type noOpMonitor struct{}

// NewNoOpMonitor creates a monitor which always returns empty information
func NewNoOpMonitor() Monitor {
	return &noOpMonitor{}
}

func (*noOpMonitor) Start() {}

func (*noOpMonitor) GetHTTPStats() map[Key]RequestStats {
	return nil
}

func (*noOpMonitor) GetStats() (map[string]int64, error) {
	return nil, nil
}

func (*noOpMonitor) Stop() error {
	return nil
}
