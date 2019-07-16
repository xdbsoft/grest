package api

import (
	"time"

	"github.com/rs/xid"
)

//Document represents a document in a collection
type Document struct {
	ID                   string                 `json:"id"`
	CreationDate         *time.Time             `json:"creationDate,omitempty"`
	LastModificationDate *time.Time             `json:"lastModificationDate,omitempty"`
	Properties           map[string]interface{} `json:"properties"`
}

//DocumentProperties represents the properties of the document
type DocumentProperties map[string]interface{}

//NextID generates a pseudo-random ID that could be used when creating a document
func NextID() string {
	return xid.New().String()
}

//Collection represents a list of documents
type Collection struct {
	ID       string     `json:"id"`
	Features []Document `json:"features"`
}
