module github.com/modfin/smtpx/middleware

go 1.24.1

require (
	blitiri.com.ar/go/spf v1.5.1
	github.com/emersion/go-msgauth v0.6.8
	github.com/modfin/smtpx v0.0.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.36.0
	golang.org/x/net v0.37.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/modfin/smtpx => ..
	github.com/modfin/smtpx/middleware => ../middleware
)
