package protocol

const (
	MessageTypeOpen        = iota
	MessageTypeClose       = iota
	MessageTypePing        = iota
	MessageTypePong        = iota
	MessageTypeEmpty       = iota
	MessageTypeUnKnown     = iota
	MessageTypeEmit        = iota
	MessageTypeAckRequest  = iota
	MessageTypeAckResponse = iota
)

type Message struct {
	Type   int
	AckId  int
	Method string
	Args   string
	Source string
}
