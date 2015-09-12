package messages

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/gedex/inflector"
	"github.com/zond/hackyhack/lang"
)

const (
	VoidResource = "0"
)

type EventType int

const (
	EventTypeRequest EventType = iota
)

const (
	MethodGetContainer = "GetContainer"
	MethodGetContent   = "GetContent"
	MethodSendToClient = "SendToClient"
	MethodGetShortDesc = "GetShortDesc"
	MethodGetLongDesc  = "GetLongDesc"
	MethodSubscribe    = "Subscribe"
)

type BlobType int

const (
	BlobTypeRequest BlobType = iota
	BlobTypeResponse
	BlobTypeConstruct
	BlobTypeDestruct
)

type ErrorCode int

const (
	ErrorCodeUnknown ErrorCode = iota
	ErrorCodeNoSuchMethod
	ErrorCodeMethodMismatch
	ErrorCodeNoSuchResource
	ErrorCodeJSONDecodeParameters
	ErrorCodeJSONEncodeParameters
	ErrorCodeJSONDecodeResult
	ErrorCodeJSONEncodeResult
	ErrorCodeProxyFailed
	ErrorCodeSendToClient
	ErrorCodeDatabase
	ErrorCodeRegexp
)

type Error struct {
	Message string
	Code    ErrorCode
}

func (e *Error) ToErr() error {
	if e == nil {
		return nil
	}
	return fmt.Errorf("%v: %v", e.Message, e.Code)
}

func FromErr(err error) *Error {
	return &Error{Message: err.Error()}
}

type ShortDescs []*ShortDesc

func (sd ShortDescs) Enumerate() string {
	uniques := ShortDescs{}
	nonUniques := map[ShortDesc]int{}
	for _, desc := range sd {
		if desc.Unique || desc.Name {
			uniques = append(uniques, desc)
		} else {
			nonUniques[*desc] = nonUniques[*desc] + 1
		}
	}

	result := []string{}
	for desc, count := range nonUniques {
		if count == 1 {
			result = append(result, desc.IndefArticlize())
		} else {
			result = append(result, fmt.Sprintf("%v %v", count, desc.Pluralize()))
		}
	}
	for _, desc := range uniques {
		result = append(result, desc.IndefArticlize())
	}

	buf := &bytes.Buffer{}
	for i := 0; i < len(result); i++ {
		fmt.Fprint(buf, result[i])
		if i < len(result)-2 {
			fmt.Fprint(buf, ", ")
		} else if i < len(result)-1 {
			fmt.Fprint(buf, ", and ")
		}
	}
	return buf.String()
}

type ShortDesc struct {
	Value string
	// Unique items will not be grouped together ("3 apples") or get the indefinite article ("an apple") but
	// will always get a definite article ("the apple").
	Unique bool
	// Names will never get an article at all, i.e. not "a percy" or "the percy" but "percy".
	Name bool
}

func (sd *ShortDesc) DefArticlize() string {
	if sd.Name {
		return sd.Value
	}
	return fmt.Sprintf("the %v", sd.Value)
}

func (sd *ShortDesc) IndefArticlize() string {
	if sd.Name {
		return sd.Value
	}
	if sd.Unique {
		return fmt.Sprintf("the %v", sd.Value)
	}
	return fmt.Sprintf("%v %v", lang.Art(sd.Value), sd.Value)
}

func (sd *ShortDesc) Pluralize() string {
	return inflector.Pluralize(sd.Value)
}

type Subscription struct {
	VerbReg     string
	MethReg     string
	HandlerName string
}

type Event struct {
	Type     EventType
	Metadata map[string]string
	Request  *Request
}

type Verb struct {
	SecondPerson string
	ThirdPerson  string
	Intransitive bool
}

func (v *Verb) Matches(r *regexp.Regexp) bool {
	return r.MatchString(v.SecondPerson) || r.MatchString(v.ThirdPerson)
}

type RequestHeader struct {
	Id     string
	Source string
	Verb   *Verb
}

type Request struct {
	Header     RequestHeader
	Resource   string
	Method     string
	Parameters string
}

type ResponseHeader struct {
	Id    string
	Error *Error `json:",omitempty"`
}

type Response struct {
	Header ResponseHeader
	Result string
}

type Deconstruct struct {
	Id            string
	Resource      string
	Deconstructed bool
}

type Blob struct {
	Type      BlobType
	Request   *Request     `json:",omitempty"`
	Response  *Response    `json:",omitempty"`
	Construct *Deconstruct `json:",omitempty"`
	Destruct  *Deconstruct `json:",omitempty"`
}
