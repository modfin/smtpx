package responses

import (
	"fmt"
)

// class is a type for ClassSuccess, ClassTransientFailure and ClassPermanentFailure constants
type class int

// String implements Response for the class type
func (c class) String() string {
	return fmt.Sprintf("%c00", c)
}

// it looks like this ".5.4"
type subjectDetail string

// EnhancedStatus are the ones that look like 2.1.0
type EnhancedStatusCode struct {
	Class             class
	SubjectDetailCode subjectDetail
}

// String returns a string representation of EnhancedStatus
func (e EnhancedStatusCode) String() string {
	return fmt.Sprintf("%d%s", e.Class, e.SubjectDetailCode)
}

// response type for Stringer interface
type response struct {
	enhancedCode subjectDetail
	basicCode    int
	class        class

	// Comment is optional
	comment string
}

func (r *response) StatusCode() int {
	if r.basicCode == 0 {
		r.basicCode = getBasicStatusCode(EnhancedStatusCode{r.class, r.enhancedCode})
	}
	return r.basicCode
}

func (r *response) Class() int {
	return int(r.class)
}

// String returns a custom response as a string
func (r *response) String() string {
	if r.enhancedCode == "" {
		return r.comment
	}

	class := class(r.Class())
	basicCode := r.StatusCode()
	enhancedCode := EnhancedStatusCode{class, r.enhancedCode}
	comment := r.comment

	if len(comment) == 0 {
		comment = defaultTexts.m[enhancedCode]
	}
	if len(comment) == 0 {
		switch class {
		case 2:
			comment = "OK"
		case 4:
			comment = "Temporary failure."
		case 5:
			comment = "Permanent failure."
		}
	}

	str := fmt.Sprintf("%d %s %s", basicCode, enhancedCode.String(), comment)
	return str
}

// getBasicStatusCode gets the basic status code from codeMap, or fallback code if not mapped
func getBasicStatusCode(e EnhancedStatusCode) int {
	if val, ok := codeMap.m[e]; ok {
		return val
	}
	// Fallback if code is not defined
	return int(e.Class) * 100
}
