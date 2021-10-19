package distributions

//go:generate msgp -unexported -marshal=false -o=payload_msgp.go -tests=false

type distributionPayload struct {
	Sketch []byte
	Timestamp float64
	Name string
	Tags []string
}
