package main

import (
	"fmt"
	"encoding/json"
	"net/http"
	"errors"
	"strconv"
)


func paramsParseProfileParams(r *http.Request) (res *ProfileParams, err error) {
	res = &ProfileParams{}

	res.Login = r.FormValue("login")

	return res, err
}



func paramsParseCreateParams(r *http.Request) (res *CreateParams, err error) {
	res = &CreateParams{}

	res.Login = r.FormValue("login")
	res.Name = r.FormValue("full_name")
	res.Status = r.FormValue("status")
	res.Age, err = strconv.Atoi(r.FormValue("age"))
	if err != nil {
		return nil, errors.New("age must be int")
	}

	return res, err
}



func paramsParseOtherCreateParams(r *http.Request) (res *OtherCreateParams, err error) {
	res = &OtherCreateParams{}

	res.Username = r.FormValue("username")
	res.Name = r.FormValue("account_name")
	res.Class = r.FormValue("class")
	res.Level, err = strconv.Atoi(r.FormValue("level"))
	if err != nil {
		return nil, errors.New("level must be int")
	}

	return res, err
}



func (s *ProfileParams) Validate() (err error) {

	if s.Login == "" {
		return errors.New("login must be not empty")
	}

	return err
}

func (s *CreateParams) Validate() (err error) {

	if s.Login == "" {
		return errors.New("login must be not empty")
	}
	if len(s.Login) < 10 {
		return errors.New("login len must be >= 10")
	}
	if s.Status != "" && s.Status != "user" && s.Status != "moderator" && s.Status != "admin" {
		return errors.New("status must be one of [user, moderator, admin]")
	}
	if s.Age < 0 {
		return errors.New("age must be >= 0")
	}
	if s.Age > 128 {
		return errors.New("age must be <= 128")
	}

	return err
}

func (s *OtherCreateParams) Validate() (err error) {

	if s.Username == "" {
		return errors.New("username must be not empty")
	}
	if len(s.Username) < 3 {
		return errors.New("username len must be >= 3")
	}
	if s.Class != "" && s.Class != "warrior" && s.Class != "sorcerer" && s.Class != "rouge" {
		return errors.New("class must be one of [warrior, sorcerer, rouge]")
	}
	if s.Level < 1 {
		return errors.New("level must be >= 1")
	}
	if s.Level > 50 {
		return errors.New("level must be <= 50")
	}

	return err
}
func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/user/profile":
		params, err := paramsParseProfileParams(r)
		var errResp []byte
		if err != nil {
			errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(w, string(errResp), http.StatusBadRequest)
			return
		}

		err = params.Validate()
		if err != nil {
			errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(w, string(errResp), http.StatusBadRequest)
			return
		}

		fRes, err := h.Profile(r.Context(), *params)

		if err != nil {
			errApi, ok := err.(ApiError)
			if(ok) {
				errResp, _ = json.Marshal(map[string]interface{}{"error": errApi.Err.Error()})
				http.Error(w, string(errResp), errApi.HTTPStatus)
				return
			} else {
				errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
				http.Error(w, string(errResp), http.StatusInternalServerError)
				return
			}

		}
		result := map[string]interface{}{"response": fRes, "error": ""}
		resultMarhalled, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "{\"error\":\"\"}", http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, string(resultMarhalled))
	case "/user/create":
		if r.Method != "POST" {
			http.Error(w, "{\"error\": \"bad method\"}", http.StatusNotAcceptable)
			return
		}

		if r.Header.Get("X-Auth") != "100500" {
			http.Error(w, "{\"error\": \"unauthorized\"}", http.StatusForbidden)
			return
		}

		params, err := paramsParseCreateParams(r)
		var errResp []byte
		if err != nil {
			errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(w, string(errResp), http.StatusBadRequest)
			return
		}

		err = params.Validate()
		if err != nil {
			errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(w, string(errResp), http.StatusBadRequest)
			return
		}

		fRes, err := h.Create(r.Context(), *params)

		if err != nil {
			errApi, ok := err.(ApiError)
			if(ok) {
				errResp, _ = json.Marshal(map[string]interface{}{"error": errApi.Err.Error()})
				http.Error(w, string(errResp), errApi.HTTPStatus)
				return
			} else {
				errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
				http.Error(w, string(errResp), http.StatusInternalServerError)
				return
			}

		}
		result := map[string]interface{}{"response": fRes, "error": ""}
		resultMarhalled, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "{\"error\":\"\"}", http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, string(resultMarhalled))
	default:
		http.Error(w, "{\"error\": \"unknown method\"}", http.StatusNotFound)
	}

}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/user/create":
		if r.Method != "POST" {
			http.Error(w, "{\"error\": \"bad method\"}", http.StatusNotAcceptable)
			return
		}

		if r.Header.Get("X-Auth") != "100500" {
			http.Error(w, "{\"error\": \"unauthorized\"}", http.StatusForbidden)
			return
		}

		params, err := paramsParseOtherCreateParams(r)
		var errResp []byte
		if err != nil {
			errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(w, string(errResp), http.StatusBadRequest)
			return
		}

		err = params.Validate()
		if err != nil {
			errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
			http.Error(w, string(errResp), http.StatusBadRequest)
			return
		}

		fRes, err := h.Create(r.Context(), *params)

		if err != nil {
			errApi, ok := err.(ApiError)
			if(ok) {
				errResp, _ = json.Marshal(map[string]interface{}{"error": errApi.Err.Error()})
				http.Error(w, string(errResp), errApi.HTTPStatus)
				return
			} else {
				errResp, _ = json.Marshal(map[string]interface{}{"error": err.Error()})
				http.Error(w, string(errResp), http.StatusInternalServerError)
				return
			}

		}
		result := map[string]interface{}{"response": fRes, "error": ""}
		resultMarhalled, err := json.Marshal(result)
		if err != nil {
			http.Error(w, "{\"error\":\"\"}", http.StatusInternalServerError)
			return
		}

		fmt.Fprintln(w, string(resultMarhalled))
	default:
		http.Error(w, "{\"error\": \"unknown method\"}", http.StatusNotFound)
	}

}


