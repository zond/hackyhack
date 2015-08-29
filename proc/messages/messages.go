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
	ErrorCodeNoSuchMethod ErrorCode = iota
	ErrorCodeNoSuchResource
	ErrorCodeJSONDecodeParameters
	ErrorCodeJSONEncodeResult
)

type Error struct {
	Message string
	Code    ErrorCode
}

func (e Error) Error() string {
	return fmt.Sprintf("%v: %v", e.Message, e.Code)
}

type RequestHeader struct {
	RequestId  string
	ResourceId string
	Method     string
}

type ResponseHeader struct {
	RequestId  string
	ResourceId string
	Error      *Error `json:",omitempty"`
}

type Request struct {
	Header     RequestHeader
	Parameters string
}

type Response struct {
	Header ResponseHeader
	Result string
}

type Construct struct {
	RequestId   string
	ResourceId  string
	Constructed bool
}

type Destruct struct {
	RequestId  string
	ResourceId string
	Destroyed  bool
}

type Blob struct {
	Type      BlobType
	Request   *Request   `json:",omitempty"`
	Response  *Response  `json:",omitempty"`
	Construct *Construct `json:",omitempty"`
	Destruct  *Destruct  `json:",omitempty"`
}
