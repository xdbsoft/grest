package grest

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/xdbsoft/grest/api"
)

type mockedAuthenticator struct{}

func (a mockedAuthenticator) Authenticate(r *http.Request) (api.User, error) {
	formBearer := r.FormValue("auth")
	if len(formBearer) == 0 {
		return api.User{}, nil
	}

	tokens := strings.Split(formBearer, "|")
	if len(tokens) != 3 {
		return api.User{}, notAuthorizedError{}
	}

	return api.User{
		ID:    tokens[0],
		Name:  tokens[1],
		Email: tokens[2],
	}, nil
}

type notFound string

func (err notFound) IsNotFound() bool {
	return true
}
func (err notFound) Error() string {
	return string(err)
}

type mockedDataRepository struct {
	Data map[string]map[string]api.Document
	Now  time.Time
}

func (r *mockedDataRepository) Init() error {
	r.Data = make(map[string]map[string]api.Document)
	return nil
}

func (r *mockedDataRepository) Get(document api.ObjectRef) (api.Document, error) {

	c := document.Collection().String()
	col, found := r.Data[c]

	if !found {
		return api.Document{}, notFound("document not found")
	}

	doc, found := col[document.ID()]

	if !found {
		return api.Document{}, notFound("document not found")
	}

	return doc, nil
}

type SortDocuments struct {
	docs    []api.Document
	orderBy string
}

func (a SortDocuments) Len() int {
	return len(a.docs)
}
func (a SortDocuments) Less(i, j int) bool {

	if a.orderBy == "" || a.orderBy == "$id" {
		return a.docs[i].ID < a.docs[j].ID
	}

	switch vi := a.docs[i].Properties[a.orderBy].(type) {
	case string:
		vj := a.docs[j].Properties[a.orderBy].(string)
		return vi < vj
	}
	panic("unsupported type for sort")
}
func (a SortDocuments) Swap(i, j int) {
	a.docs[i], a.docs[j] = a.docs[j], a.docs[i]
}

func (r *mockedDataRepository) GetAll(c api.ObjectRef, orderBy []string, limit int) ([]api.Document, error) {

	col, found := r.Data[c.String()]

	if !found {
		return nil, notFound("collection not found")
	}

	var res []api.Document
	for _, d := range col {
		res = append(res, d)
	}

	orderByItem := ""
	if len(orderBy) > 0 {
		orderByItem = orderBy[0]
	}

	sort.Sort(SortDocuments{res, orderByItem})

	if len(res) > limit {
		res = res[:limit]
	}

	return res, nil
}

func (r *mockedDataRepository) Add(c api.ObjectRef, payload api.DocumentProperties) (api.Document, error) {

	col, found := r.Data[c.String()]

	if !found {
		col = make(map[string]api.Document)
	}

	id := fmt.Sprintf("ID_%d", len(col)+1)

	now := r.Now
	col[id] = api.Document{
		ID:                   id,
		CreationDate:         now,
		LastModificationDate: now,
		Properties:           payload,
	}

	r.Data[c.String()] = col

	return col[id], nil
}

func (r *mockedDataRepository) Put(document api.ObjectRef, payload api.DocumentProperties) error {

	c := document.Collection().String()
	col, found := r.Data[c]

	if !found {
		col = make(map[string]api.Document)
	}

	now := r.Now
	col[document.ID()] = api.Document{
		ID:                   document.ID(),
		CreationDate:         now,
		LastModificationDate: now,
		Properties:           payload,
	}

	r.Data[c] = col

	return nil
}
func (r *mockedDataRepository) Patch(document api.ObjectRef, payload api.DocumentProperties) error {

	c := document.Collection().String()
	col, found := r.Data[c]

	if !found {
		col = make(map[string]api.Document)
	}

	d := col[document.ID()]

	now := r.Now
	d.LastModificationDate = now

	for k, v := range payload {
		d.Properties[k] = v
	}

	col[document.ID()] = d
	r.Data[c] = col

	return nil
}
func (r *mockedDataRepository) Delete(document api.ObjectRef) error {

	c := document.Collection().String()
	col, found := r.Data[c]

	if found {
		delete(col, document.ID())
	}

	r.Data[c] = col

	return nil
}
func (r *mockedDataRepository) DeleteCollection(collection api.ObjectRef) error {

	delete(r.Data, collection.String())

	return nil
}
