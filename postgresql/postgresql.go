package postgresql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

type transaction struct {
	tx *sql.Tx
}

type cursor struct {
	name string
	tx   *sql.Tx
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
	defer rows.Close()
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
			created    timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated    timestamp with time zone NOT NULL DEFAULT CURRENT_TIMESTAMP,
			content    jsonb,
			CONSTRAINT t_document_pkey PRIMARY KEY (collection, id)
		)`); err != nil {
			return errors.Wrap(err, "CREATE TABLE t_document failed")
		}
	}
	return nil
}

func (r *repository) Begin() (api.Transaction, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}

	return &transaction{tx: tx}, nil
}

func (tx *transaction) Commit() error {
	return tx.tx.Commit()
}

func (tx *transaction) Rollback() error {
	return tx.tx.Rollback()
}

func (tx *transaction) Get(d api.ObjectRef) (api.Document, error) {

	rows, err := tx.tx.Query("SELECT content, created, updated FROM t_document WHERE collection=$1 AND id=$2", d.Collection().String(), d.ID())
	if err != nil {
		return api.Document{}, errors.Wrap(err, "Select query failed")
	}
	defer rows.Close()

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
		CreationDate:         created,
		LastModificationDate: updated,
		Properties:           content,
	}, nil
}

func (tx *transaction) GetAll(c api.ObjectRef, orderBy []string) (api.Cursor, error) {

	cursorName := api.NextID()

	orderByString := "id"
	if len(orderBy) > 0 {
		mappedOrderBy := make([]string, len(orderBy))
		for i := range orderBy {
			items := strings.Split(orderBy[i], ".")
			if len(items) == 1 {
				switch items[0] {
				case "id":
					mappedOrderBy[i] = "id"
				case "creationDate":
					mappedOrderBy[i] = "created"
				case "lastModificationDate":
					mappedOrderBy[i] = "updated"
				default:
					return nil, errors.New("Unknown item in order by clause: " + orderBy[i])
				}
			} else {
				return nil, errors.New("Unknown item in order by clause: " + orderBy[i])
			}
		}
		orderByString = strings.Join(mappedOrderBy, ",")
	}

	_, err := tx.tx.Exec("DECLARE "+cursorName+" CURSOR FOR SELECT id, created, updated, content FROM t_document WHERE collection=$1 ORDER BY "+orderByString, c.String())
	if err != nil {
		return nil, errors.Wrap(err, "DB query failed")
	}

	return &cursor{
		name: cursorName,
		tx:   tx.tx,
	}, nil
}

func (c *cursor) Close() error {
	_, err := c.tx.Exec("CLOSE " + c.name)
	if err != nil {
		return errors.Wrap(err, "DB query failed")
	}
	return nil
}

func (c *cursor) Fetch(count int) ([]api.Document, error) {

	rows, err := c.tx.Query(fmt.Sprintf("FETCH FORWARD %d FROM %s", count, c.name))
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
			CreationDate:         created,
			LastModificationDate: updated,
			Properties:           content,
		})
	}

	return result, nil
}

func (tx *transaction) Add(c api.ObjectRef, payload api.DocumentProperties) (api.Document, error) {

	id := api.NextID()

	b, err := json.Marshal(&payload)
	if err != nil {
		return api.Document{}, errors.Wrap(err, "unable to encode payload")
	}

	row := tx.tx.QueryRow("INSERT INTO t_document (collection, id, content) VALUES ($1,$2,$3) RETURNING CURRENT_TIMESTAMP", c.String(), id, &b)

	var t time.Time
	if err := row.Scan(&t); err != nil {
		return api.Document{}, errors.Wrap(err, "unable to insert document")
	}

	d := api.Document{
		ID:                   id,
		CreationDate:         t,
		LastModificationDate: t,
		Properties:           payload,
	}

	return d, nil
}

func (tx *transaction) Put(d api.ObjectRef, payload api.DocumentProperties) error {

	b, err := json.Marshal(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to encode payload")
	}

	if _, err := tx.tx.Exec("INSERT INTO t_document (collection, id, content) VALUES ($1,$2,$3) ON CONFLICT(collection,id) DO UPDATE SET content=$3,updated=CURRENT_TIMESTAMP", d.Collection().String(), d.ID(), &b); err != nil {
		return errors.Wrap(err, "unable to insert or update document")
	}

	return nil
}
func (tx *transaction) Patch(d api.ObjectRef, payload api.DocumentProperties) error {

	b, err := json.Marshal(&payload)
	if err != nil {
		return errors.Wrap(err, "unable to encode payload")
	}

	if _, err := tx.tx.Exec("UPDATE t_document SET content = (content || $1),updated=CURRENT_TIMESTAMP WHERE collection=$2 AND id=$3", &b, d.Collection().String(), d.ID()); err != nil {
		return errors.Wrap(err, "unable to update document")
	}

	return nil
}

func (tx *transaction) Delete(d api.ObjectRef) error {

	if _, err := tx.tx.Exec("DELETE FROM t_document where collection=$1 and id=$2", d.Collection().String(), d.ID()); err != nil {
		return errors.Wrap(err, "unable to delete document")
	}

	return nil
}

func (tx *transaction) DeleteCollection(c api.ObjectRef) error {

	if _, err := tx.tx.Exec("DELETE FROM t_document where collection=$1", c.String()); err != nil {
		return errors.Wrap(err, "unable to delete collection")
	}

	return nil
}
