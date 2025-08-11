package storage

type redisPaste struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

type Paste struct {
	ID              string `json:"id"`
	Content         string `json:"content"`
	LifetimeSeconds int64  `json:"lifetimeSeconds"`
}
