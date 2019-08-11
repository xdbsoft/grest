package grest

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xdbsoft/grest/api"
)

type testRequest struct {
	method              string
	url                 string
	body                string
	expectedCode        int
	expectedContentType string
	expectedBody        string
}

type testCase struct {
	collections    []api.CollectionDefinition
	data           map[string]map[string]api.Document
	requests       []testRequest
	checkDatastore func(data map[string]map[string]api.Document, t *testing.T)
}

func (c testCase) Run(t *testing.T) {

	colDefs := make(map[string]api.CollectionDefinition)
	for _, cd := range c.collections {
		colDefs[cd.Path.String()] = cd
	}

	mock := &mockedDataRepository{Data: c.data, Now: time.Date(2018, 8, 24, 5, 0, 0, 0, time.UTC)}

	s := server{
		Collections:    colDefs,
		Authenticator:  mockedAuthenticator{},
		DataRepository: mock,
		RuleChecker:    api.RuleChecker{},
	}

	for j, request := range c.requests {
		var b io.Reader
		if request.body != "" {
			b = bytes.NewBufferString(request.body)
		}

		req := httptest.NewRequest(request.method, request.url, b)
		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != request.expectedCode {
			t.Errorf("Request %d: Unexpected status code, expected %d, got %d", j, request.expectedCode, resp.StatusCode)
		}

		contentType := resp.Header.Get("Content-Type")
		if contentType != request.expectedContentType {
			t.Errorf("Request %d: Unexpected content type, expected %s, got %s", j, request.expectedContentType, contentType)
		}

		bodyString := string(body)
		if bodyString != request.expectedBody {
			t.Errorf("Request %d: Unexpected body, expected '%s', got '%s'", j, request.expectedBody, bodyString)
		}

		mock.Now = mock.Now.Add(1 * time.Hour)
	}

}

func TestServeHTTP_Get_Document(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:         "doc1",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"doc1","properties":{"k":"v"}}
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_BadRequests(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com",
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `empty path
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test//test2/doc",
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `empty item in path
`,
			},
			{
				method:              "GET2",
				url:                 "http://example.com/test/doc",
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `unsupported method
`,
			},
			{
				method:              "GET2",
				url:                 "http://example.com/test",
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `unsupported method
`,
			},
			{
				method:              "PUT",
				url:                 "http://example.com/test/doc",
				body:                `not json`,
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unable to decode JSON body: invalid character 'o' in literal null (expecting 'u')
`,
			},
			{
				method:              "POST",
				url:                 "http://example.com/test/doc",
				body:                `123`,
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unable to decode JSON body: json: cannot unmarshal number into Go value of type api.DocumentProperties
`,
			},
			{
				method:              "POST",
				url:                 "http://example.com/test",
				body:                `"invalid"`,
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unable to decode JSON body: json: cannot unmarshal string into Go value of type api.DocumentProperties
`,
			},
			{
				method:              "PUT",
				url:                 "http://example.com/test/doc",
				body:                `{"k":"v"}`,
				expectedCode:        400,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Invalid ID
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_Collection(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {
				"doc1": api.Document{
					ID:         "doc1",
					Properties: map[string]interface{}{"k": "v"},
				},
				"doc2": api.Document{
					ID:         "doc2",
					Properties: map[string]interface{}{"k": "a"},
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"test","features":[{"id":"doc1","properties":{"k":"v"}},{"id":"doc2","properties":{"k":"a"}}]}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test?limit=10",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"test","features":[{"id":"doc1","properties":{"k":"v"}},{"id":"doc2","properties":{"k":"a"}}]}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test?limit=1&orderBy=k",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"test","features":[{"id":"doc2","properties":{"k":"a"}}]}
`,
			},
		},
	}

	c.Run(t)
}
func TestServeHTTP_Get_Print(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:         "doc1",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1?print=pretty",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{
  "id": "doc1",
  "properties": {
    "k": "v"
  }
}
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_NotFound(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {"doc0": api.Document{
				ID:         "doc0",
				Properties: map[string]interface{}{"k": "v"},
			}},
			"test2": {"doc1": api.Document{
				ID:         "doc1",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        404,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Data not found
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test2",
				expectedCode:        404,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Data not found
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test2/doc1",
				expectedCode:        404,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Data not found
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_InvalidAuth(t *testing.T) {

	c := testCase{
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1?auth=abcd",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test?auth=abcd",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1?auth=abcd||",
				expectedCode:        404,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Data not found
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_PutGet(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{},
		requests: []testRequest{
			{
				method:              "PUT",
				url:                 "http://example.com/test/doc1",
				body:                `{"id":"doc1","properties":{"k":"v"}}`,
				expectedCode:        204,
				expectedContentType: "",
				expectedBody:        "",
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"doc1","creationDate":"2018-08-24T05:00:00Z","lastModificationDate":"2018-08-24T05:00:00Z","properties":{"k":"v"}}
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_PostGet_Collection(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{},
		requests: []testRequest{
			{
				method:              "POST",
				url:                 "http://example.com/test",
				body:                `{"k":"v"}`,
				expectedCode:        202,
				expectedContentType: "application/json",
				expectedBody: `{"id":"ID_1","creationDate":"2018-08-24T05:00:00Z","lastModificationDate":"2018-08-24T05:00:00Z","properties":{"k":"v"}}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/ID_1",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"ID_1","creationDate":"2018-08-24T05:00:00Z","lastModificationDate":"2018-08-24T05:00:00Z","properties":{"k":"v"}}
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_PutPostGet(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{},
		requests: []testRequest{
			{
				method:              "PUT",
				url:                 "http://example.com/test/doc1",
				body:                `{"id":"doc1","properties":{"k":"v"}}`,
				expectedCode:        204,
				expectedContentType: "",
				expectedBody:        "",
			},
			{
				method:              "POST",
				url:                 "http://example.com/test/doc1",
				body:                `{"k":"v2","x":123}`,
				expectedCode:        204,
				expectedContentType: "",
				expectedBody:        "",
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"doc1","creationDate":"2018-08-24T05:00:00Z","lastModificationDate":"2018-08-24T06:00:00Z","properties":{"k":"v2","x":123}}
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_IncorrectRule(t *testing.T) {

	c := testCase{
		data: map[string]map[string]api.Document{
			"test": {"101": api.Document{
				ID:         "101",
				Properties: map[string]interface{}{"k": "v"},
			}, "099": api.Document{
				ID:         "099",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
				Rules: []api.Rule{
					{
						Path: "test/{doc}",
						Allow: []api.Allow{
							{
								Methods: []api.Method{"READ"},
								If:      `path.doc > '100`,
							},
						},
					},
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/099",
				expectedCode:        500,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Internal server error
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_RuleOnPath(t *testing.T) {

	c := testCase{
		data: map[string]map[string]api.Document{
			"test": {"101": api.Document{
				ID:         "101",
				Properties: map[string]interface{}{"k": "v"},
			}, "099": api.Document{
				ID:         "099",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
				Rules: []api.Rule{
					{
						Path: "test/{doc}",
						Allow: []api.Allow{
							{
								Methods: []api.Method{"READ"},
								If:      `path.doc > '100'`,
							},
						},
					},
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/101",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"101","properties":{"k":"v"}}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/099",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
		},
	}

	c.Run(t)
}
func TestServeHTTP_Get_RuleOnUser(t *testing.T) {

	c := testCase{
		data: map[string]map[string]api.Document{
			"test": {"abcd": api.Document{
				ID:         "abcd",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
				Rules: []api.Rule{
					{
						Path: "test/{userId}",
						Allow: []api.Allow{
							{
								Methods: []api.Method{"READ"},
								If:      `path.userId == user.id`,
							},
						},
					},
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/abcd?auth=abcd||",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"abcd","properties":{"k":"v"}}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/abcd",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Delete_Document(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:         "doc1",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "DELETE",
				url:                 "http://example.com/test/doc1",
				expectedCode:        204,
				expectedContentType: "",
				expectedBody:        "",
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        404,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Data not found
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Delete_Collection(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:         "doc1",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "DELETE",
				url:                 "http://example.com/test",
				expectedCode:        204,
				expectedContentType: "",
				expectedBody:        "",
			},
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        404,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Data not found
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_NotAutorized(t *testing.T) {

	c := testCase{
		collections: []api.CollectionDefinition{
			{
				Path: api.CollectionRef{"test"},
				Rules: []api.Rule{
					{
						Path: "test",
						Allow: []api.Allow{
							{
								Methods: []api.Method{"READ", "WRITE", "DELETE"},
								If:      `"doc1" != "doc1"`,
							},
						},
					},
				},
			},
		},
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:         "doc1",
				Properties: map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "POST",
				url:                 "http://example.com/test",
				body:                `{"k":"v"}`,
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "PUT",
				url:                 "http://example.com/test/doc1",
				body:                `{"id":"doc1","properties":{"k":"v"}}`,
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "POST",
				url:                 "http://example.com/test/doc1",
				body:                `{"k":"v"}`,
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "DELETE",
				url:                 "http://example.com/test/doc1",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
			{
				method:              "DELETE",
				url:                 "http://example.com/test",
				expectedCode:        401,
				expectedContentType: "text/plain; charset=utf-8",
				expectedBody: `Unauthorized
`,
			},
		},
	}

	c.Run(t)
}
