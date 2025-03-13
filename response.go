package brevx

import "fmt"

// Response represents a response to an SMTP connection after receiving DATA.
// The String method should return an SMTP message ready to send back to the
// connection, for example `250 OK: Message received`.
type Response interface {
	fmt.Stringer
	Code() int
	Class() int
}

type ResponseErr interface {
	Response
	error
}

type result struct {
	code int
	str  string
}

func (r result) String() string {
	var clazz string
	switch r.code / 100 {
	case 2:
		clazz = "OK"
	case 4:
		clazz = "Temporary failure"
	case 5:
		clazz = "Permanent failure"
	}
	return fmt.Sprintf("%d %s: %s", r.code, clazz, r.str)
}

func (r result) Code() int {
	return r.code
}
func (r result) Class() int {
	return r.code / 100
}

func NewResponse(code int, comment string) Response {
	return result{code: code, str: comment}
}

type resultErr struct {
	Response
	err error
}

func (e resultErr) Error() string {
	return e.err.Error()
}

func WrapResponse(res Response, err error) ResponseErr {
	return resultErr{Response: res, err: err}
}
