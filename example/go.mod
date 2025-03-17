module github.com/modfin/smtpx/example

go 1.24.1

require (
	github.com/modfin/smtpx v0.0.0
	github.com/modfin/smtpx/middleware v0.0.0
	golang.org/x/crypto v0.36.0
)

require (
	blitiri.com.ar/go/spf v1.5.1 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/text v0.23.0 // indirect
)

replace (
	github.com/modfin/smtpx => ..
	github.com/modfin/smtpx/middleware => ../middleware
)
