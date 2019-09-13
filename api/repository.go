package api

//Repository describes the interface that a datastore should implement
type Repository interface {
	Init() error

	Get(document ObjectRef) (Document, error)
	GetAll(collection ObjectRef, orderBy []string, limit int) ([]Document, error)
	Add(collection ObjectRef, payload DocumentProperties) (Document, error)
	Put(document ObjectRef, payload DocumentProperties) error
	Patch(document ObjectRef, payload DocumentProperties) error
	Delete(document ObjectRef) error
	DeleteCollection(collection ObjectRef) error
}
