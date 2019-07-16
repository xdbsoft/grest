package api

import (
	"strings"
)

type ObjectRef []string

func (c ObjectRef) String() string {
	return strings.Join(c, "/")
}

func (o ObjectRef) IsDocumentRef() bool {
	return len(o) > 0 && len(o)%2 == 0
}

//CollectionRef is the path for a collection or sub-collection
type CollectionRef ObjectRef

func (c CollectionRef) String() string {
	return strings.Join(c, "/")
}

func (c CollectionRef) ID() string {
	return c[len(c)-1]
}

func (c CollectionRef) IsRoot() bool {
	return len(c) == 1
}

func (c CollectionRef) Parent() DocumentRef {
	if c.IsRoot() {
		return nil
	}
	return DocumentRef(c[:len(c)-1])
}

type DocumentRef ObjectRef

func (d DocumentRef) String() string {
	return strings.Join(d, "/")
}

func (d DocumentRef) ID() string {
	return d[len(d)-1]
}

func (d DocumentRef) Collection() CollectionRef {
	return CollectionRef(d[:len(d)-1])
}
