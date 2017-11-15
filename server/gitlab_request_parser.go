package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/lkysow/go-gitlab"
)

const secretHeader = "X-Gitlab-Token"

//go:generate pegomock generate --use-experimental-model-gen --package mocks -o mocks/mock_gitlab_request_parser.go GitlabRequestParser

// GitlabRequestParser parses and validates GitLab requests.
type GitlabRequestParser interface {
	// Validate returns the JSON payload of the request.
	// If secret is not empty, it checks that the request was signed
	// by secret and returns an error if it was not.
	// If secret is empty, it does not check if the request was signed.
	Validate(r *http.Request, secret []byte) ([]byte, error)
	Parse(r *http.Request, bytes []byte) (interface{}, error)
}

// DefaultGitlabRequestValidator validates GitLab requests.
type DefaultGitlabRequestValidator struct{}

// Validate returns the JSON payload of the request.
// If secret is not empty, it checks that the request was signed
// by secret and returns an error if it was not.
// If secret is empty, it does not check if the request was signed.
func (d *DefaultGitlabRequestValidator) Validate(r *http.Request, secret []byte) ([]byte, error) {
	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	headerSecret := r.Header.Get(secretHeader)
	secretStr := string(secret)
	if len(secret) != 0 && headerSecret != secretStr {
		return nil, fmt.Errorf("header %s=%s did not match expected secret", secretHeader, headerSecret)
	}
	return bytes, nil
}

func (d *DefaultGitlabRequestValidator) Parse(r *http.Request, bytes []byte) (interface{}, error) {
	const mergeEventHeader = "Merge Request Hook"
	const noteEventHeader = "Note Hook"

	switch r.Header.Get(gitlabHeader) {
	case mergeEventHeader:
		var m gitlab.MergeEvent
		if err := json.Unmarshal(bytes, &m); err != nil {
			return nil, err
		}
		return m, nil
	case noteEventHeader:
		var m gitlab.MergeCommentEvent
		if err := json.Unmarshal(bytes, &m); err != nil {
			return nil, err
		}
		return m, nil
	default:
		return nil, nil
	}
}
