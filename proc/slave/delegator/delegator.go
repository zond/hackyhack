package delegator

import (
	"errors"
	"fmt"
	"reflect"
)

type Delegator struct {
	b reflect.Value
}

func New(b interface{}) *Delegator {
	return &Delegator{
		b: reflect.ValueOf(b),
	}
}

var ErrNoSuchMethod = errors.New("No such method")
var ErrWrongNumberOfParams = errors.New("Wrong number of params")
var ErrWrongNumberOfResults = errors.New("Wrong number of results")
var ErrParamsNotSlice = errors.New("Params is not a slice")
var ErrResultsNotPointerToSlice = errors.New("Results is not a pointer to a slice")

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

func (h *Delegator) verifySlice(
	slice interface{},
	wantedLen int,
	typeGen func(int) reflect.Type,
	errs *Errors,
	typeErrGen func(index int, got reflect.Type, want reflect.Type) error,
	sliceErr error,
	lenErr error,
	wantPtr bool,
	ptrErr error,
) []reflect.Value {
	result := make([]reflect.Value, wantedLen)
	if slice == nil {
		return result
	}

	sliceVal := reflect.ValueOf(slice)
	if wantPtr {
		if sliceVal.Kind() != reflect.Ptr {
			*errs = append(*errs, ptrErr)
			return result
		}
		sliceVal = sliceVal.Elem()
	}
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

func (h *Delegator) Call(methName string, params, results interface{}) error {
	methVal := h.b.MethodByName(methName)
	if !methVal.IsValid() {
		return fmt.Errorf("No method %q found", methName)
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
		false,
		nil,
	)

	callResults := h.verifySlice(
		results,
		methType.NumOut(),
		methType.Out,
		&errs,
		newTypeErrorGen("result"),
		ErrResultsNotPointerToSlice,
		ErrWrongNumberOfResults,
		true,
		ErrResultsNotPointerToSlice,
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
