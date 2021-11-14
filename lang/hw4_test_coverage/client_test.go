package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

const xmlFilePath = "./dataset.xml"
const authToken = "AHGDS-PTRBLQRYUI-ZLPQG-DAWWL"
const unexistedSiteUrl = "http://this-site-does-not-exist.example.com"

type UserToken struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

// код писать тут
func SearchServer(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("AccessToken")
	if authorization != authToken {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"error": "Not authorized"}`)
		return
	}
	query := r.FormValue("query")
	orderField := r.FormValue("order_field")
	if orderField == "" {
		orderField = "name"
	}
	if orderField != "name" && orderField != "age" && orderField != "id" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "`+ErrorBadOrderField+`"}`)
		return
	}
	orderByNotParsed := r.FormValue("order_by")
	var orderBy int
	if orderByNotParsed == "" {
		orderBy = OrderByAsIs
	} else {
		var err error
		orderBy, err = strconv.Atoi(orderByNotParsed)
		if err != nil || orderBy != OrderByAsc && orderBy != OrderByDesc && orderBy != OrderByAsIs {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `{"error": "OrderBy is invalid"}`)
			return
		}
	}
	limitNotParsed := r.FormValue("limit")
	limit, err := strconv.Atoi(limitNotParsed)
	if err != nil || limit < 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "Limit is invalid"}`)
		return
	}
	offsetNotParsed := r.FormValue("offset")
	offset, err := strconv.Atoi(offsetNotParsed)
	if err != nil || offset < 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error": "Offset in invalid"}`)
		return
	}

	xmlData, err := ioutil.ReadFile(xmlFilePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	users := parseXmlUsers(xmlData, query)

	if len(users) == 0 {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `[]`)
		return
	}

	if orderBy != OrderByAsIs {
		switch orderField {
		case "id":
			sort.SliceStable(users, func(i, j int) bool {
				if orderBy == OrderByAsc {
					return users[i].Id > users[j].Id
				} else {
					return users[i].Id < users[j].Id
				}
			})
		case "name":
			sort.SliceStable(users, func(i, j int) bool {
				if orderBy == OrderByAsc {
					return users[i].Name > users[j].Name
				} else {
					return users[i].Name < users[j].Name
				}
			})
		case "age":
			sort.SliceStable(users, func(i, j int) bool {
				if orderBy == OrderByAsc {
					return users[i].Age > users[j].Age
				} else {
					return users[i].Age < users[j].Age
				}
			})
		}
	}

	if len(users) < offset+1 {
		users = []User{}
	} else if len(users) < offset+limit {
		users = users[offset:]
	} else {
		users = users[offset : offset+limit]
	}

	resultData, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, string(resultData))
}

func parseXmlUsers(data []byte, query string) (users []User) {
	input := bytes.NewReader(data)
	decoder := xml.NewDecoder(input)
	record := UserToken{}
	for {
		tok, tokenErr := decoder.Token()
		if tokenErr != nil && tokenErr != io.EOF {
			fmt.Println("error happend", tokenErr)
			break
		} else if tokenErr == io.EOF {
			break
		}
		if tok == nil {
			fmt.Println("t is nil break")
		}
		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "row" {
				if err := decoder.DecodeElement(&record, &tok); err != nil {
					fmt.Println("error happend", err)
					continue
				}
				userName := record.FirstName + " " + record.LastName
				if query != "" && !strings.Contains(userName, query) && !strings.Contains(record.About, query) {
					continue
				}
				user := User{
					Id:     record.Id,
					Name:   userName,
					Age:    record.Age,
					About:  record.About,
					Gender: record.Gender,
				}
				users = append(users, user)
			}
		}
	}
	return users
}

func SearchServerTimeout(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `[]`)
}

func SearchServerFatal(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func SearchServerInvalidError(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `{"error":`)
}

func SearchServerInvalidBody(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `[{"name":`)
}

type TestCase struct {
	Req    SearchRequest
	Result *SearchResponse
	Error  error
}

func TestFindUsersSuccess(t *testing.T) {
	cases := []TestCase{
		{
			Req: SearchRequest{
				Offset:     1,
				Limit:      1,
				OrderField: "age",
				OrderBy:    OrderByAsc,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						Id:     32,
						Name:   "Christy Knapp",
						Age:    40,
						About:  "Incididunt culpa dolore laborum cupidatat consequat. Aliquip cupidatat pariatur sit consectetur laboris labore anim labore. Est sint ut ipsum dolor ipsum nisi tempor in tempor aliqua. Aliquip labore cillum est consequat anim officia non reprehenderit ex duis elit. Amet aliqua eu ad velit incididunt ad ut magna. Culpa dolore qui anim consequat commodo aute.\n",
						Gender: "female",
					},
				},
				NextPage: true,
			},
		},
		{
			Req: SearchRequest{
				Offset:     0,
				Limit:      10,
				Query:      "voluptate co",
				OrderField: "id",
				OrderBy:    OrderByDesc,
			},
			Result: &SearchResponse{
				Users: []User{
					{
						Id:     0,
						Name:   "Boyd Wolf",
						Age:    22,
						About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
						Gender: "male",
					},
					{
						Id:     1,
						Name:   "Hilda Mayer",
						Age:    21,
						About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
						Gender: "female",
					},
				},
				NextPage: false,
			},
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{authToken, ts.URL}

	for caseNum, item := range cases {
		result, err := client.FindUsers(item.Req)

		if err != nil {
			t.Errorf("[%d] unexpected error: %#v", caseNum, err)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestFindUsersParamsErrors(t *testing.T) {
	cases := []TestCase{
		{
			Req: SearchRequest{
				Offset:     1,
				Limit:      -1,
				OrderField: "age",
				OrderBy:    OrderByAsc,
			},
			Error: fmt.Errorf("limit must be > 0"),
		},
		{
			Req: SearchRequest{
				Offset:     -1,
				Limit:      20,
				OrderField: "age",
				OrderBy:    OrderByAsc,
			},
			Error: fmt.Errorf("offset must be > 0"),
		},
		{
			Req: SearchRequest{
				Offset:     1,
				Limit:      20,
				OrderField: "about",
				OrderBy:    OrderByAsc,
			},
			Error: fmt.Errorf("OrderField about invalid"),
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{authToken, ts.URL}

	for caseNum, item := range cases {
		result, err := client.FindUsers(item.Req)

		if err == nil || err.Error() != item.Error.Error() {
			t.Errorf("[%d] expected error %#v, got %#v", caseNum, item.Error, err)
		}
		if !reflect.DeepEqual(item.Result, result) {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.Result, result)
		}
	}
	ts.Close()
}

func TestFindUsersNotAuthorized(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{"aaa", ts.URL}
	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)

	if err == nil || err.Error() != "Bad AccessToken" {
		t.Errorf("expected error `Bad AccessToken`, got %#v", err)
	}
	ts.Close()
}

func TestFindUsersTimeOut(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServerTimeout))
	client := SearchClient{authToken, ts.URL}
	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)

	if err == nil || !strings.Contains(err.Error(), "timeout for") {
		t.Errorf("expected error `timeout for`, got %#v", err)
	}
	ts.Close()
}

func TestFindUsersInternalError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServerFatal))
	client := SearchClient{authToken, ts.URL}
	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)

	if err == nil || err.Error() != "SearchServer fatal error" {
		t.Errorf("expected error `SearchServer fatal error`, got %#v", err)
	}
	ts.Close()
}

func TestFindUsersInvalidError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServerInvalidError))
	client := SearchClient{authToken, ts.URL}
	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)

	if err == nil || !strings.Contains(err.Error(), "cant unpack error json") {
		t.Errorf("expected error `cant unpack error json`, got %#v", err)
	}
	ts.Close()
}

func TestFindUsersInvalidBody(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServerInvalidBody))
	client := SearchClient{authToken, ts.URL}
	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)

	if err == nil || !strings.Contains(err.Error(), "cant unpack result json") {
		t.Errorf("expected error `cant unpack result json`, got %#v", err)
	}
	ts.Close()
}

func TestFindUsersUnknownBadRequestError(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{authToken, ts.URL}
	req := SearchRequest{Limit: 10, OrderBy: 100500}
	_, err := client.FindUsers(req)

	if err == nil || !strings.Contains(err.Error(), "unknown bad request error") {
		t.Errorf("expected error `unknown bad request error`, got %#v", err)
	}
	ts.Close()
}

func TestFindUsersUnknownServerError(t *testing.T) {

	client := SearchClient{authToken, unexistedSiteUrl}
	req := SearchRequest{Limit: 10}
	_, err := client.FindUsers(req)

	if err == nil || !strings.Contains(err.Error(), "unknown error") {
		t.Errorf("expected error `unknown error`, got %#v", err)
	}
}

func TestFindUsersBigLimit(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	client := SearchClient{authToken, ts.URL}
	req := SearchRequest{Limit: 40}
	result, err := client.FindUsers(req)

	if err != nil {
		t.Errorf("unexpected error %#v", err)
	}
	if len(result.Users) != 25 {
		t.Errorf("expected 25 users, got %d", len(result.Users))
	}
	ts.Close()
}
