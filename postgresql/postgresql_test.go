package postgresql

import (
	"log"
	"testing"

	"github.com/xdbsoft/grest/api"
)

const ConnectionString string = "user=nestor password=nestor dbname=nestor sslmode=disable"

func init() {
	r, err := New(ConnectionString)
	if err != nil {
		err = r.Init()
	}
	if err != nil {
		log.Fatal(err)
	}
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

	d := api.DocumentRef{"test", "doc1"}
	payload := make(api.DocumentProperties)
	payload["k"] = "v"
	payload["n"] = 123

	r, err := New(ConnectionString)
	if err != nil {
		t.Error(err)
	}

	if err := r.Put(d, payload); err != nil {
		t.Error(err)
	}

	res, err := r.Get(d)
	if err != nil {
		t.Error(err)
	}

	if res.ID != "doc1" {
		t.Errorf("Invalid ID: got '%s', expected 'doc1'", res.ID)
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

	payload = make(api.DocumentProperties)
	payload["k2"] = "v2"
	payload["n"] = 125
	if err := r.Patch(d, payload); err != nil {
		t.Error(err)
	}

	res, err = r.Get(d)
	if err != nil {
		t.Error(err)
	}

	if res.ID != "doc1" {
		t.Errorf("Invalid ID: got '%s', expected 'doc1'", res.ID)
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

	if err := r.Delete(d); err != nil {
		t.Error(err)
	}

	res, err = r.Get(d)
	if err == nil {
		t.Error("Document should not be found")
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

	c := api.CollectionRef{"test"}
	d, err := r.Add(c, payload)
	if err != nil {
		t.Error(err)
	}
	if len(d.ID) == 0 {
		t.Error("Document ID should be returned")
	}

	dref := api.DocumentRef{"test", d.ID}

	res, err := r.Get(dref)
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

	d2, err := r.Add(c, payload)
	if err != nil {
		t.Error(err)
	}
	if len(d2.ID) == 0 {
		t.Error("Document 2 ID should be returned")
	}

	all, err := r.GetAll(c, nil, 2)
	if err != nil {
		t.Error(err)
	}
	if len(all) != 2 {
		t.Errorf("Invalid list length, got %v, expected 2", len(all))
	}

	err = r.DeleteCollection(c)
	if err != nil {
		t.Error(err)
	}

	all, err = r.GetAll(c, nil, 2)
	if err != nil {
		t.Error(err)
	}
	if len(all) != 0 {
		t.Errorf("Invalid list length, got %v, expected 0", len(all))
	}

}
