package authres

type msgauthTest struct {
	value      string
	identifier string
	results    []Result
}

var crlf = "\r\n"

var msgauthTests = []msgauthTest{
	{
		value: "example.org;" + crlf +
			"  none",
		identifier: "example.org",
		results:    nil,
	},
	{
		value: "example.com;" + crlf +
			"  dkim=none ",
		identifier: "example.com",
		results: []Result{
			&DKIMResult{Value: ResultNone},
		},
	},
	{
		value: "example.com;" + crlf +
			"  spf=pass smtp.mailfrom=example.net",
		identifier: "example.com",
		results: []Result{
			&SPFResult{Value: ResultPass, From: "example.net"},
		},
	},
	{
		value: "example.com;" + crlf +
			"  spf=fail reason=bad smtp.mailfrom=example.net",
		identifier: "example.com",
		results: []Result{
			&SPFResult{Value: ResultFail, Reason: "bad", From: "example.net"},
		},
	},
	{
		value: "example.com;" + crlf +
			"  auth=pass smtp.auth=sender@example.com;" + crlf +
			"  spf=pass smtp.mailfrom=example.com",
		identifier: "example.com",
		results: []Result{
			&AuthResult{Value: ResultPass, Auth: "sender@example.com"},
			&SPFResult{Value: ResultPass, From: "example.com"},
		},
	},
	{
		value: "example.com;" + crlf +
			"  sender-id=pass header.from=example.com",
		identifier: "example.com",
		results: []Result{
			&SenderIDResult{Value: ResultPass, HeaderKey: "from", HeaderValue: "example.com"},
		},
	},
	{
		value: "example.com;" + crlf +
			"  sender-id=hardfail header.from=example.com;" + crlf +
			"  dkim=pass header.i=sender@example.com",
		identifier: "example.com",
		results: []Result{
			&SenderIDResult{Value: ResultHardFail, HeaderKey: "from", HeaderValue: "example.com"},
			&DKIMResult{Value: ResultPass, Identifier: "sender@example.com"},
		},
	},
	{
		value: "example.com;" + crlf +
			"  auth=pass smtp.auth=sender@example.com;" + crlf +
			"  spf=hardfail smtp.mailfrom=example.com",
		identifier: "example.com",
		results: []Result{
			&AuthResult{Value: ResultPass, Auth: "sender@example.com"},
			&SPFResult{Value: ResultHardFail, From: "example.com"},
		},
	},
	{
		value: "example.com;" + crlf +
			"  dkim=pass header.i=@mail-router.example.net;" + crlf +
			"  dkim=fail header.i=@newyork.example.com",
		identifier: "example.com",
		results: []Result{
			&DKIMResult{Value: ResultPass, Identifier: "@mail-router.example.net"},
			&DKIMResult{Value: ResultFail, Identifier: "@newyork.example.com"},
		},
	},
}
