package main

import (
	"net/http"
	"errors"
	"strconv"
	"context"
)

func paramsParseProfileParams(r *http.Request) (*ProfileParams, error) {
	resStruct := &ProfileParams{}
	resStruct.Login = r.FormValue("login")
	return resStruct, nil
}

func (s *ProfileParams) Validate() error {

	if s.Login == "" {
		return errors.New("login must be a empty")
	}

	return nil
}

func paramsParseCreateParams(r *http.Request) (*CreateParams, error) {
	resStruct := &CreateParams{}
	resStruct.Login = r.FormValue("login")
	resStruct.Name = r.FormValue("name")
	resStruct.Status = r.FormValue("status")

	var err error
	resStruct.Age, err = strconv.Atoi(r.FormValue("age"))
	if err != nil {
		return nil, errors.New("age must be int")
	}
	return resStruct, nil
}

func (s *CreateParams) Validate() error {

	if s.Login == "" {
		return errors.New("login must be a empty")
	}

	if s.Status != "user" && s.Status != "moderator" && s.Status != "admin" {
		return errors.New("status must be one of [user, moderator, admin]")
	}


	if s.Age < 0 {
		return errors.New("age must be >= 0")
	}

	return nil
}

func paramsParseOtherCreateParams(r *http.Request) (*OtherCreateParams, error) {
	resStruct := &OtherCreateParams{}
	resStruct.Username = r.FormValue("username")
	resStruct.Name = r.FormValue("name")
	resStruct.Class = r.FormValue("class")

	var err error
	resStruct.Level, err = strconv.Atoi(r.FormValue("level"))
	if err != nil {
		return nil, errors.New("level must be int")
	}
	return resStruct, nil
}

func (s *OtherCreateParams) Validate() error {

	if s.Username == "" {
		return errors.New("username must be a empty")
	}

	if s.Class != "warrior" && s.Class != "sorcerer" && s.Class != "rouge" {
		return errors.New("class must be one of [warrior, sorcerer, rouge]")
	}


	if s.Level < 1 {
		return errors.New("level must be >= 1")
	}

	return nil
}

func (h *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/user/profile":
	case "/user/create":
		if r.Method != "POST" {
			http.Error(w, "bad method", http.StatusNotAcceptable)
		}

		if r.Header.Get("X-Auth") != "100500" {
			http.Error(w, "unauthorized", http.StatusForbidden)
		}

		params, err := paramsParseCreateParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		err = params.Validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}

}

func (h *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.URL.Path {
	case "/user/create":
		if r.Method != "POST" {
			http.Error(w, "bad method", http.StatusNotAcceptable)
		}

		if r.Header.Get("X-Auth") != "100500" {
			http.Error(w, "unauthorized", http.StatusForbidden)
		}

		params, err := paramsParseOtherCreateParams(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

		err = params.Validate()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}

	default:
		http.Error(w, "unknown method", http.StatusNotFound)
	}

}

