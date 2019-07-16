package api

//Repository describes the interface that a datastore should implement
type Repository interface {
	Init() error

	Get(document DocumentRef) (Document, error)
	GetAll(collection CollectionRef, orderBy []string, limit int) ([]Document, error)
	Add(collection CollectionRef, payload DocumentProperties) (Document, error)
	Put(document DocumentRef, payload DocumentProperties) error
	Patch(document DocumentRef, payload DocumentProperties) error
	Delete(document DocumentRef) error
	DeleteCollection(collection CollectionRef) error
}
