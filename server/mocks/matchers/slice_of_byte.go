package matchers

import (
	"reflect"

	"github.com/petergtz/pegomock"
)

func AnySliceOfByte() []byte {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*([]byte))(nil)).Elem()))
	var nullValue []byte
	return nullValue
}

func EqSliceOfByte(value []byte) []byte {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue []byte
	return nullValue
}
