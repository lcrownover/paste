package server

type CreatePasteRequest struct {
	Content         string `json:"content"`
	LifetimeSeconds int64  `json:"lifetimeSeconds"`
}

type DeletePasteRequest struct {
	ID string `json:"id"`
}

type PasteResponse struct {
	ID              string `json:"id"`
	Content         string `json:"content"`
	LifetimeSeconds int64  `json:"lifetimeSeconds"`
}
