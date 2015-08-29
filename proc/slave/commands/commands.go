package commands

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

func Sprintf(f string, i ...interface{}) string {
	return fmt.Sprintf(f, i...)
}

var whitespaceReg = regexp.MustCompile("\\s+")

func SplitWhitespace(s string) []string {
	return whitespaceReg.Split(s, -1)
}

func Capitalize(s string) string {
	return strings.ToUpper(string([]rune(s)[0:1])) + s[1:]
}

type Handler struct {
	b reflect.Value
}

func New(b interface{}) *Handler {
	return &Handler{
		b: reflect.ValueOf(b),
	}
}

var ErrNoSuchMethod = errors.New("No such method")
var ErrWrongNumberOfParams = errors.New("Wrong number of params")
var ErrWrongNumberOfResults = errors.New("Wrong number of results")
var ErrParamsNotSlice = errors.New("Params is not a slice")
var ErrResultsNotSlice = errors.New("Results is not a slice")

type TypeError struct {
	Kind  string
	Index int
	Got   reflect.Type
	Want  reflect.Type
}

func (t *TypeError) Error() string {
	return fmt.Sprintf("Bad %v type %v; got %v, want %v", t.Kind, t.Index, t.Got, t.Want)
}

func newTypeErrorGen(kind string) func(index int, got, want reflect.Type) error {
	return func(index int, got, want reflect.Type) error {
		return &TypeError{
			Kind:  kind,
			Index: index,
			Got:   got,
			Want:  want,
		}
	}
}

type Errors []error

func (e Errors) Error() string {
	return fmt.Sprintf("%+v", []error(e))
}

func (h *Handler) verifySlice(
	slice interface{},
	wantedLen int,
	typeGen func(int) reflect.Type,
	errs *Errors,
	typeErrGen func(index int, got reflect.Type, want reflect.Type) error,
	sliceErr error,
	lenErr error,
) []reflect.Value {
	result := make([]reflect.Value, wantedLen)
	sliceVal := reflect.ValueOf(slice)
	if sliceVal.Kind() == reflect.Slice {
		if sliceVal.Len() == wantedLen {
			for i := 0; i < wantedLen; i++ {
				val := sliceVal.Index(i)
				if val.Type().AssignableTo(typeGen(i)) {
					result[i] = val
				} else {
					*errs = append(*errs, typeErrGen(i, val.Type(), typeGen(i)))
				}
			}
		} else {
			*errs = append(*errs, lenErr)
		}
	} else {
		*errs = append(*errs, sliceErr)
	}
	return result
}

func (h *Handler) Call(methName string, params, results interface{}) error {
	methVal := h.b.MethodByName(methName)
	if !methVal.IsValid() {
		return ErrNoSuchMethod
	}
	methType := methVal.Type()

	var errs Errors

	callParams := h.verifySlice(
		params,
		methType.NumIn(),
		methType.In,
		&errs,
		newTypeErrorGen("parameter"),
		ErrParamsNotSlice,
		ErrWrongNumberOfParams,
	)

	callResults := h.verifySlice(
		results,
		methType.NumOut(),
		methType.Out,
		&errs,
		newTypeErrorGen("result"),
		ErrResultsNotSlice,
		ErrWrongNumberOfResults,
	)

	if errs != nil {
		return errs
	}

	actualResults := methVal.Call(callParams)

	for index := range actualResults {
		callResults[index].Set(actualResults[index])
	}

	return nil
}
