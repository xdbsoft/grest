package api

//Repository describes the interface that a datastore should implement
type Repository interface {
	Init() error

	Begin() (Transaction, error)
}

//Transaction describes the interface that a datastore transaction should implement
type Transaction interface {
	Get(document ObjectRef) (Document, error)
	GetAll(collection ObjectRef, orderBy []string) (Cursor, error)
	Add(collection ObjectRef, payload DocumentProperties) (Document, error)
	Put(document ObjectRef, payload DocumentProperties) error
	Patch(document ObjectRef, payload DocumentProperties) error
	Delete(document ObjectRef) error
	DeleteCollection(collection ObjectRef) error

	Commit() error
	Rollback() error
}

type Cursor interface {
	Fetch(count int) ([]Document, error)
	Close() error
}
