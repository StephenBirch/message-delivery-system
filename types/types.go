package types

// ListResponse is used to wrap IDs for json (un)Marshalling
type ListResponse struct {
	IDs []uint64
}

// SendingMessage is used to combine a recipients and the data to deliver
type SendingMessage struct {
	Recipients string
	Data       []byte
}
