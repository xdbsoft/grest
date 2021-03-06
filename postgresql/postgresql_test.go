package postgresql

import (
	"log"
	"os"
	"testing"

	"github.com/xdbsoft/grest/api"
)

const ConnectionString string = "user=nestor password=nestor dbname=nestor sslmode=disable"

func TestMain(m *testing.M) {
	r, err := New(ConnectionString)
	if err == nil {
		err = r.Init()
	}
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

func TestNew(t *testing.T) {

	r, err := New(ConnectionString)
	if err != nil {
		t.Error(err)
	}

	if r == nil {
		t.Error("Invalid repository")
	}
}

func TestPutPatchGetDeleteDocument(t *testing.T) {

	d := api.ObjectRef{"test", "doc1"}
	payload := make(api.DocumentProperties)
	payload["k"] = "v"
	payload["n"] = 123

	r, err := New(ConnectionString)
	if err != nil {
		t.Error(err)
	}

	tx, err := r.Begin()
	if err != nil {
		t.Error(err)
	}

	if err := tx.Put(d, payload); err != nil {
		t.Error(err)
	}

	res, err := tx.Get(d)
	if err != nil {
		t.Error(err)
	}

	if res.ID != "doc1" {
		t.Errorf("Invalid ID: got '%s', expected 'doc1'", res.ID)
	}

	if res.CreationDate != res.LastModificationDate {
		t.Error("Different creation and last modification dates", res.CreationDate, res.LastModificationDate)
	}

	v, ok := res.Properties["k"]
	if !ok {
		t.Errorf("Missing field 'k'")
	}
	if v != "v" {
		t.Errorf("Invalid field 'k': got '%s', expected 'v'", v)
	}

	v2, ok := res.Properties["n"]
	if !ok {
		t.Errorf("Missing field 'n'")
	}
	if v2 != 123. { //TODO: why float and not int? is it supported by json?
		t.Errorf("Invalid field 'n': got %v, expected 123", v2)
	}

	err = tx.Commit()
	if err != nil {
		t.Error(err)
	}

	tx, err = r.Begin()
	if err != nil {
		t.Error(err)
	}

	payload = make(api.DocumentProperties)
	payload["k2"] = "v2"
	payload["n"] = 125
	if err := tx.Patch(d, payload); err != nil {
		t.Error(err)
	}

	res, err = tx.Get(d)
	if err != nil {
		t.Error(err)
	}

	if res.ID != "doc1" {
		t.Errorf("Invalid ID: got '%s', expected 'doc1'", res.ID)
	}
	if !res.CreationDate.Before(res.LastModificationDate) {
		t.Error("Creation date expected before last modification date", res.CreationDate, res.LastModificationDate)
	}

	v, ok = res.Properties["k"]
	if !ok {
		t.Errorf("Missing field 'k'")
	}
	if v != "v" {
		t.Errorf("Invalid field 'k': got '%s', expected 'v'", v)
	}
	v, ok = res.Properties["k2"]
	if !ok {
		t.Errorf("Missing field 'k2'")
	}
	if v != "v2" {
		t.Errorf("Invalid field 'k2': got '%s', expected 'v2'", v)
	}

	v2, ok = res.Properties["n"]
	if !ok {
		t.Errorf("Missing field 'n'")
	}
	if v2 != 125. { //TODO: why float and not int? is it supported by json?
		t.Errorf("Invalid field 'n': got %v, expected 125", v2)
	}

	if err := tx.Delete(d); err != nil {
		t.Error(err)
	}

	res, err = tx.Get(d)
	if err == nil {
		t.Error("Document should not be found")
	}

	err = tx.Commit()
	if err != nil {
		t.Error(err)
	}
}

func TestAddGetDeleteCollection(t *testing.T) {

	payload := make(api.DocumentProperties)
	payload["k"] = "v"
	payload["n"] = 123
	payload2 := make(api.DocumentProperties)
	payload2["k"] = "v2"
	payload2["m"] = 125

	r, err := New(ConnectionString)
	if err != nil {
		t.Error(err)
	}

	tx, err := r.Begin()
	if err != nil {
		t.Error(err)
	}

	c := api.ObjectRef{"test"}
	d, err := tx.Add(c, payload)
	if err != nil {
		t.Error(err)
	}
	if len(d.ID) == 0 {
		t.Error("Document ID should be returned")
	}

	dref := api.ObjectRef{"test", d.ID}

	res, err := tx.Get(dref)
	if err != nil {
		t.Error(err)
	}

	if res.ID != d.ID {
		t.Errorf("Invalid ID: got '%s', expected '%s'", res.ID, d.ID)
	}

	v, ok := res.Properties["k"]
	if !ok {
		t.Errorf("Missing field 'k'")
	}
	if v != "v" {
		t.Errorf("Invalid field 'k': got '%s', expected 'v'", v)
	}

	v2, ok := res.Properties["n"]
	if !ok {
		t.Errorf("Missing field 'n'")
	}
	if v2 != 123. {
		t.Errorf("Invalid field 'n': got %v, expected 123", v2)
	}

	d2, err := tx.Add(c, payload)
	if err != nil {
		t.Error(err)
	}
	if len(d2.ID) == 0 {
		t.Error("Document 2 ID should be returned")
	}

	cu, err := tx.GetAll(c, nil)
	if err != nil {
		t.Error(err)
	}

	all, err := cu.Fetch(2)
	if err != nil {
		t.Error(err)
	}

	err = cu.Close()
	if err != nil {
		t.Error(err)
	}

	if len(all) != 2 {
		t.Errorf("Invalid list length, got %v, expected 2", len(all))
	}

	err = tx.DeleteCollection(c)
	if err != nil {
		t.Error(err)
	}

	cu2, err := tx.GetAll(c, nil)
	if err != nil {
		t.Error(err)
	}

	all, err = cu2.Fetch(2)
	if err != nil {
		t.Error(err)
	}

	err = cu2.Close()
	if err != nil {
		t.Error(err)
	}

	if len(all) != 0 {
		t.Errorf("Invalid list length, got %v, expected 0", len(all))
	}

	err = tx.Rollback()
	if err != nil {
		t.Error(err)
	}

}
