package main

import (
	"net/http"
)

func paramsParseProfileParams(r *http.Request) (*ProfileParams, error) {
	resStruct := &ProfileParams{}
	return resStruct, nil
}

func (s *ProfileParams) Validate() bool {

	if s.Login == "" {
		return false
	}

	return true
}

func paramsParseCreateParams(r *http.Request) (*CreateParams, error) {
	resStruct := &CreateParams{}
	return resStruct, nil
}

func (s *CreateParams) Validate() bool {

	if s.Login == "" {
		return false
	}

	if s.Status != "user" && s.Status != "moderator" && s.Status != "admin" {
		return false
	}


	if s.Age < 0 {
		return false
	}


	if s.Age > 128 {
		return false
	}

	return true
}

func paramsParseOtherCreateParams(r *http.Request) (*OtherCreateParams, error) {
	resStruct := &OtherCreateParams{}
	return resStruct, nil
}

func (s *OtherCreateParams) Validate() bool {

	if s.Username == "" {
		return false
	}

	if s.Class != "warrior" && s.Class != "sorcerer" && s.Class != "rouge" {
		return false
	}


	if s.Level < 1 {
		return false
	}


	if s.Level > 50 {
		return false
	}

	return true
}

