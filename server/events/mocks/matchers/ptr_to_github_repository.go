package matchers

import (
	"reflect"

	github "github.com/google/go-github/github"
	"github.com/petergtz/pegomock"
)

func AnyPtrToGithubRepository() *github.Repository {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(*github.Repository))(nil)).Elem()))
	var nullValue *github.Repository
	return nullValue
}

func EqPtrToGithubRepository(value *github.Repository) *github.Repository {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue *github.Repository
	return nullValue
}
