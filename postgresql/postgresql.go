package postgresql

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	//we expect to depend on specific behaviour of github.com/lib/pq
	_ "github.com/lib/pq"
	"github.com/xdbsoft/grest/api"
)

func New(connStr string) (api.Repository, error) {

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "unable to connect")
	}

	return &repository{
		db: db,
	}, nil
}

type repository struct {
	db *sql.DB
}

type notFound string

func (err notFound) IsNotFound() bool {
	return true
}
func (err notFound) Error() string {
	return string(err)
}

func (r *repository) Init() error {
	// Check if tables exists, if not create them
	rows, err := r.db.Query("SELECT to_regclass('t_document')")
	if err != nil {
		return errors.Wrap(err, "Select query for t_document failed")
	}
	tablesFound := false
	for rows.Next() {
		var found string
		rows.Scan(&found)
		tablesFound = len(found) > 0
	}
	if !tablesFound {
		if _, err := r.db.Exec(`CREATE TABLE t_document (
			collection text NOT NULL,
			id         character varying(126) NOT NULL,
			content    jsonb,
			created    timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated    timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT t_document_pkey PRIMARY KEY (collection, id)
		)`); err != nil {
			return errors.Wrap(err, "CREATE TABLE t_document failed")
		}
	}
	return nil
}

func (r *repository) Get(d api.DocumentRef) (api.Document, error) {

	rows, err := r.db.Query("SELECT content, created, updated FROM t_document WHERE collection=$1 AND id=$2", d.Collection().String(), d.ID())
	if err != nil {
		return api.Document{}, errors.Wrap(err, "Select query failed")
	}

	if !rows.Next() {
		return api.Document{}, notFound("document not found")
	}

	var s []byte
	var created, updated time.Time
	if err := rows.Scan(&s, &created, &updated); err != nil {
		return api.Document{}, errors.Wrap(err, "DB retrieval failed")
	}

	content := make(api.DocumentProperties)
	if err := json.Unmarshal(s, &content); err != nil {
		return api.Document{}, errors.Wrap(err, "DB decoding failed")
	}

	return api.Document{
		ID:                   d.ID(),
		CreationDate:         &created,
		LastModificationDate: &updated,
		Properties:           content,
	}, nil
}

func (r *repository) GetAll(c api.CollectionRef, orderBy []string, limit int) ([]api.Document, error) {

	rows, err := r.db.Query("SELECT id, created, updated, content FROM t_document WHERE collection=$1 ORDER BY id LIMIT $2", c.String(), limit)
	if err != nil {
		return nil, errors.Wrap(err, "DB query failed")
	}

	var result []api.Document
	for rows.Next() {
		var id string
		var b []byte
		var created, updated time.Time
		if err := rows.Scan(&id, &created, &updated, &b); err != nil {
			return nil, errors.Wrap(err, "DB retrieval failed")
		}

		content := make(api.DocumentProperties)
		if err := json.Unmarshal(b, &content); err != nil {
			return nil, errors.Wrap(err, "DB decoding failed")
		}

		result = append(result, api.Document{
			ID:                   id,
			CreationDate:         &created,
			LastModificationDate: &updated,
			Properties:           content,
		})
	}

	return result, nil
}

func (r *repository) Add(c api.CollectionRef, payload api.DocumentProperties) (api.Document, error) {

	id := api.NextID()

	b, err := json.Marshal(&payload)

	if err != nil {
		return api.Document{}, errors.Wrap(err, "unable to encode payload")
	}

	if _, err := r.db.Exec("INSERT INTO t_document (collection, id, content) VALUES ($1,$2,$3)", c.String(), id, &b); err != nil {
		return api.Document{}, errors.Wrap(err, "unable to insert document")
	}

	d := api.Document{
		ID:         id,
		Properties: payload,
	}

	return d, nil
}

func (r *repository) Put(d api.DocumentRef, payload api.DocumentProperties) error {

	b, err := json.Marshal(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to encode payload")
	}

	if _, err := r.db.Exec("INSERT INTO t_document (collection, id, content) VALUES ($1,$2,$3) ON CONFLICT(collection,id) DO UPDATE SET content=$3,updated=CURRENT_TIMESTAMP", d.Collection().String(), d.ID(), &b); err != nil {
		return errors.Wrap(err, "unable to insert or update document")
	}

	return nil
}
func (r *repository) Patch(d api.DocumentRef, payload api.DocumentProperties) error {

	b, err := json.Marshal(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to encode payload")
	}

	if _, err := r.db.Exec("UPDATE t_document SET content = (content || $1),updated=CURRENT_TIMESTAMP WHERE collection=$2 AND id=$3", &b, d.Collection().String(), d.ID()); err != nil {
		return errors.Wrap(err, "unable to update document")
	}

	return nil
}

func (r *repository) Delete(d api.DocumentRef) error {

	if _, err := r.db.Exec("DELETE FROM t_document where collection=$1 and id=$2", d.Collection().String(), d.ID()); err != nil {
		return errors.Wrap(err, "unable to delete document")
	}

	return nil
}

func (r *repository) DeleteCollection(c api.CollectionRef) error {

	if _, err := r.db.Exec("DELETE FROM t_document where collection=$1", c.String()); err != nil {
		return errors.Wrap(err, "unable to delete collection")
	}

	return nil
}
