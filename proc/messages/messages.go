package messages

import "fmt"

const (
	MethodGetContainer = "GetContainer"
	MethodGetContent   = "GetContent"
	MethodSendToClient = "SendToClient"
	MethodGetShortDesc = "GetShortDesc"
	MethodGetLongDesc  = "GetLongDesc"
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

type RequestHeader struct {
	Id     string
	Source string
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
