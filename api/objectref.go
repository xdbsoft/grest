package api

import (
	"strings"
)

type ObjectRef []string

func (c ObjectRef) String() string {
	return strings.Join(c, "/")
}

func (o ObjectRef) IsDocument() bool {
	return len(o) > 0 && len(o)%2 == 0
}

func (o ObjectRef) ID() string {
	return o[len(o)-1]
}

func (d ObjectRef) Collection() ObjectRef {
	return ObjectRef(d[:len(d)-1])
}
