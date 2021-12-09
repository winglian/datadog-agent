package pipeline

// StatsPayload contains pipeline stats from multiple tracers. It is the
// payload used from the agent to the backend.
type StatsPayload struct {
	AgentHostname string
	AgentEnv      string
	AgentVersion  string
	Stats         []ClientStatsPayload
}

// ClientStatsPayload is the first layer of pipeline stats aggregation. It is also
// the payload sent by tracers to the agent.
type ClientStatsPayload struct {
	Env     string
	Version string
	Stats   []ClientStatsBucket
	TracerVersion string
	TracerLanguage string
}

// ClientStatsBucket is a time bucket containing pipeline stats points.
type ClientStatsBucket struct {
	Start    uint64 // bucket start in nanoseconds
	Duration uint64 // bucket duration in nanoseconds
	Stats    []ClientStatsPoint
}

// ClientStatsPoint are stats of a pipeline. Right now, only the latency from origin.
type ClientStatsPoint struct {
	Service               string
	ReceivingName string
	Hash          uint64
	ParentHash            uint64
	Sketch                []byte // ddsketch distribution of pipeline latencies encoded in protobuf
}
