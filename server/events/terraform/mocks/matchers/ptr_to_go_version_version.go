package matchers

import (
	"reflect"

	go_version "github.com/hashicorp/go-version"
	"github.com/petergtz/pegomock"
)

func AnyPtrToGoVersionVersion() *go_version.Version {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(*go_version.Version))(nil)).Elem()))
	var nullValue *go_version.Version
	return nullValue
}

func EqPtrToGoVersionVersion(value *go_version.Version) *go_version.Version {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue *go_version.Version
	return nullValue
}
