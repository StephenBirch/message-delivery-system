package types

type ListResponse struct {
	IDs []uint64
}

type SendingMessage struct {
	Recipients string
	Data       []byte
}
