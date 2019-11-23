package api

import "time"

type Cacheable interface {
	GetLastModified() time.Time
}
