package apiv2

// Host returns the host resource of the series if any
func (m *MetricSeries) Host() string {
	var host string
	for _, r := range m.Resources {
		if r.Type == "host" {
			host = r.Name
			break
		}
	}

	return host
}
