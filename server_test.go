package grest

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/xdbsoft/grest/api"
	"github.com/xdbsoft/grest/rules"
)

type testRequest struct {
	method              string
	url                 string
	headers             map[string]string
	body                string
	expectedCode        int
	expectedHeaders     map[string]string
	expectedContentType string
	expectedBody        string
}

type testCase struct {
	rules    []rules.Rule
	data     map[string]map[string]api.Document
	requests []testRequest
}

func (c testCase) Run(t *testing.T) {

	t.Helper()

	mock := &mockedDataRepository{Data: c.data, Now: time.Date(2018, 8, 24, 5, 0, 0, 0, time.UTC)}

	s := server{
		Authenticator:  mockedAuthenticator{},
		DataRepository: mock,
		RuleChecker:    rules.NewChecker(c.rules),
	}

	for j, request := range c.requests {
		var b io.Reader
		if request.body != "" {
			b = bytes.NewBufferString(request.body)
		}

		req := httptest.NewRequest(request.method, request.url, b)
		if request.headers != nil {
			for key, value := range request.headers {
				req.Header.Add(key, value)
			}
		}

		w := httptest.NewRecorder()

		s.ServeHTTP(w, req)

		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		if resp.StatusCode != request.expectedCode {
			t.Errorf("Request %d: Unexpected status code, expected %d, got %d", j, request.expectedCode, resp.StatusCode)
		}

		if request.expectedHeaders != nil {
			for key, expectedValue := range request.expectedHeaders {
				value := resp.Header.Get(key)
				if value != expectedValue {
					t.Errorf("Request %d: Unexpected header %s, expected %s, got %s", j, key, expectedValue, value)
				}
			}
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

var aDate = time.Date(2008, 8, 30, 15, 25, 0, 0, time.UTC)

func allowAll(path string) []rules.Rule {
	return []rules.Rule{
		{
			Path: path,
		},
	}
}

func TestServeHTTP_Get_Document(t *testing.T) {

	c := testCase{
		rules: allowAll("test/{docId}"),
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:                   "doc1",
				CreationDate:         aDate,
				LastModificationDate: aDate,
				Properties:           map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/doc1",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedHeaders:     map[string]string{"ETag": `"f21bd9d57dc248f0be1d1e0e4ad3a15796eb8f03"`, "Last-Modified": "Sat, 30 Aug 2008 15:25:00 GMT"},
				expectedBody: `{"id":"doc1","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"v"}}
`,
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_Document_NotModified(t *testing.T) {

	c := testCase{
		rules: allowAll("test/{docId}"),
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:                   "doc1",
				CreationDate:         aDate,
				LastModificationDate: aDate,
				Properties:           map[string]interface{}{"k": "v"},
			}},
		},
		requests: []testRequest{
			{
				method:          "GET",
				url:             "http://example.com/test/doc1",
				headers:         map[string]string{"If-None-Match": `"f21bd9d57dc248f0be1d1e0e4ad3a15796eb8f03"`},
				expectedCode:    304,
				expectedHeaders: map[string]string{"ETag": `"f21bd9d57dc248f0be1d1e0e4ad3a15796eb8f03"`},
			},
			{
				method:          "GET",
				url:             "http://example.com/test/doc1",
				headers:         map[string]string{"If-Modified-Since": aDate.UTC().Format(http.TimeFormat)},
				expectedCode:    304,
				expectedHeaders: map[string]string{"ETag": `"f21bd9d57dc248f0be1d1e0e4ad3a15796eb8f03"`, "Last-Modified": "Sat, 30 Aug 2008 15:25:00 GMT"},
			},
		},
	}

	c.Run(t)
}

func TestServeHTTP_BadRequests(t *testing.T) {

	c := testCase{
		rules: allowAll("test/{docId}"),
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
		rules: allowAll("test/{docId}"),
		data: map[string]map[string]api.Document{
			"test": {
				"doc1": api.Document{
					ID:                   "doc1",
					CreationDate:         aDate,
					LastModificationDate: aDate,
					Properties:           map[string]interface{}{"k": "v"},
				},
				"doc2": api.Document{
					ID:                   "doc2",
					CreationDate:         aDate,
					LastModificationDate: aDate,
					Properties:           map[string]interface{}{"k": "a"},
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"test","features":[{"id":"doc1","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"v"}},{"id":"doc2","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"a"}}]}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test?limit=10",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"test","features":[{"id":"doc1","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"v"}},{"id":"doc2","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"a"}}]}
`,
			},
			{
				method:              "GET",
				url:                 "http://example.com/test?limit=1&orderBy=k",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"test","features":[{"id":"doc2","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"a"}}]}
`,
			},
		},
	}

	c.Run(t)
}
func TestServeHTTP_Get_Print(t *testing.T) {

	c := testCase{
		rules: allowAll("test/{docId}"),
		data: map[string]map[string]api.Document{
			"test": {"doc1": api.Document{
				ID:                   "doc1",
				CreationDate:         aDate,
				LastModificationDate: aDate,
				Properties:           map[string]interface{}{"k": "v"},
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
  "creationDate": "2008-08-30T15:25:00Z",
  "lastModificationDate": "2008-08-30T15:25:00Z",
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
		rules: allowAll("test/{docId}"),
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
		},
	}

	c.Run(t)
}

func TestServeHTTP_Get_InvalidAuth(t *testing.T) {

	c := testCase{
		rules: allowAll("test/{docId}"),
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
		rules: allowAll("test/{docId}"),
		data:  map[string]map[string]api.Document{},
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
		rules: allowAll("test/{docId}"),
		data:  map[string]map[string]api.Document{},
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
		rules: allowAll("test/{docId}"),
		data:  map[string]map[string]api.Document{},
		requests: []testRequest{
			{
				method:              "PUT",
				url:                 "http://example.com/test/doc1",
				body:                `{"id":"doc1","properties":{"k":"v","u":"x"}}`,
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
				expectedBody: `{"id":"doc1","creationDate":"2018-08-24T05:00:00Z","lastModificationDate":"2018-08-24T06:00:00Z","properties":{"k":"v2","u":"x","x":123}}
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
		rules: []rules.Rule{
			{
				Path: "test/{doc}",
				Read: rules.Allow{
					IfPath: `path.doc > '100`,
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
				ID:                   "101",
				CreationDate:         aDate,
				LastModificationDate: aDate,
				Properties:           map[string]interface{}{"k": "v"},
			}, "099": api.Document{
				ID:                   "099",
				CreationDate:         aDate,
				LastModificationDate: aDate,
				Properties:           map[string]interface{}{"k": "v"},
			}},
		},
		rules: []rules.Rule{
			{
				Path: "test/{doc}",
				Read: rules.Allow{
					IfPath: `path.doc > '100'`,
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/101",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"101","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"v"}}
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
				ID:                   "abcd",
				CreationDate:         aDate,
				LastModificationDate: aDate,
				Properties:           map[string]interface{}{"k": "v"},
			}},
		},
		rules: []rules.Rule{
			{
				Path: "test/{userId}",
				Read: rules.Allow{
					IfPath: `path.userId == user.id`,
				},
			},
		},
		requests: []testRequest{
			{
				method:              "GET",
				url:                 "http://example.com/test/abcd?auth=abcd||",
				expectedCode:        200,
				expectedContentType: "application/json",
				expectedBody: `{"id":"abcd","creationDate":"2008-08-30T15:25:00Z","lastModificationDate":"2008-08-30T15:25:00Z","properties":{"k":"v"}}
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
		rules: allowAll("test/{docId}"),
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
		rules: allowAll("test/{docId}"),
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
		rules: []rules.Rule{
			{
				Path: "test/{docId}",
				Read: rules.Allow{
					IfPath: `"doc1" != "doc1"`,
				},
				Write: rules.Allow{
					IfPath: `"doc1" != "doc1"`,
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
