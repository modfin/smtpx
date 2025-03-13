package middleware

import (
	"blitiri.com.ar/go/spf"
	"bytes"
	"fmt"
	"github.com/crholm/brevx"
	"github.com/crholm/brevx/envelope"
	"github.com/crholm/brevx/middleware/authres"
	"github.com/crholm/brevx/middleware/authres/dkim"
	"github.com/crholm/brevx/middleware/authres/dmarc"
	"github.com/crholm/brevx/utils"
	"golang.org/x/net/publicsuffix"
	"net"
	"net/mail"
	"strings"
)

func AddAuthenticationResult(hostname string) brevx.Middleware {
	return func(next brevx.HandlerFunc) brevx.HandlerFunc {
		return func(e *envelope.Envelope) brevx.Response {

			spfres := spfCheck(e)
			dkims := dkimCheck(e)
			dmarc := dmarcCheck(e, spfres, dkims)

			res := []authres.Result{spfres, dmarc}
			for _, d := range dkims {
				res = append(res, d)
			}

			val := authres.Format(hostname, res)
			_ = e.AddHeader("Authentication-Result", val)

			return next(e)
		}
	}
}

func dmarcCheck(e *envelope.Envelope, spf *authres.SPFResult, dkims []*authres.DKIMResult) *authres.DMARCResult {
	domain := utils.DomainOfEmail(e.MailFrom)

	var errResult = func(reason string) *authres.DMARCResult {
		return &authres.DMARCResult{
			Value:  authres.ResultNone,
			Reason: reason,
			From:   domain,
		}
	}
	headers, err := e.Headers()
	if err != nil {
		fmt.Println(err)
		return errResult("Unable to get headers")
	}

	fromHeader := headers.Get("From")
	from, err := mail.ParseAddress(fromHeader)

	if err != nil || len(fromHeader) == 0 {
		return errResult("No From header")
	}
	fromDomain := utils.DomainOfEmail(from)

	r, err := dmarc.Lookup(fromDomain)

	if err != nil {
		return errResult("DMARC lookup failed")
	}

	var val authres.ResultValue = authres.ResultFail
	reasons := []string{}

	spfAligned := false
	dkimAligned := false

	spfFrom, err := mail.ParseAddress(spf.From)
	if err != nil {
		return errResult("Unable to parse spf from")
	}
	spfDomain := utils.DomainOfEmail(spfFrom)

	if spf.Value == authres.ResultPass {
		if r.SPFAlignment == dmarc.AlignmentStrict && spfDomain == fromDomain {
			spfAligned = true
		}
		if r.SPFAlignment == dmarc.AlignmentRelaxed {

			if strings.HasSuffix(spfDomain, fromDomain) {
				spfAligned = true
			}
			spfOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(spfDomain)
			if err != nil {
				return errResult("Unable to get spf org domain")
			}
			fromOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(fromDomain)
			if err != nil {
				return errResult("Unable to get from org domain")
			}
			if spfOrgDomain == fromOrgDomain {
				spfAligned = true
			}

		}
	}

	if dkims != nil {
		for _, dkim := range dkims {
			if dkim.Value == authres.ResultPass {
				if r.DKIMAlignment == dmarc.AlignmentStrict && dkim.Domain == fromDomain {
					dkimAligned = true
					break // Found a passing, aligned DKIM signature
				}
				if r.DKIMAlignment == dmarc.AlignmentRelaxed {
					dkimOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(dkim.Domain)
					if err != nil {
						return errResult("Unable to get dkim org domain")
					}
					fromOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(fromDomain)
					if err != nil {
						return errResult("Unable to get from org domain")
					}
					if dkimOrgDomain == fromOrgDomain {
						dkimAligned = true
						break // Found a passing, aligned DKIM signature
					}
				}
			}
		}
	}

	if spfAligned || dkimAligned {
		val = authres.ResultPass
	}

	if val != authres.ResultFail {
		reasons = append(reasons, "DMARC check failed")
		if spf == nil || spf.Value != authres.ResultPass {
			reasons = append(reasons, fmt.Sprintf("SPF failed or not aligned: %s", spf.Reason))
		}
		if !dkimAligned {
			reasons = append(reasons, "DKIM failed or not aligned")
		}
	}

	if r.Policy == dmarc.PolicyNone {
		val = authres.ResultNone
		reasons = []string{"DMARC policy is none"}
	}

	res := &authres.DMARCResult{
		Value:  val,
		Reason: strings.Join(reasons, ", "),
		From:   domain,
	}
	return res
}

func dkimCheck(e *envelope.Envelope) []*authres.DKIMResult {

	buf := bytes.NewBuffer(e.Data.Bytes())
	verifications, err := dkim.Verify(buf)

	if err != nil {
		return nil
	}

	var res []*authres.DKIMResult

	for _, v := range verifications {

		fmt.Println(v.HeaderKeys)
		var val authres.ResultValue = authres.ResultFail
		if v.Err == nil {
			val = authres.ResultPass
		}

		reason := ""
		if v.Err != nil {
			reason = v.Err.Error()
		}

		res = append(res, &authres.DKIMResult{
			Value:      val,
			Reason:     reason,
			Domain:     v.Domain,
			Identifier: v.Identifier,
			Selector:   v.Selector,
			Signature:  v.Signature,
		})
	}
	return res
}
func spfCheck(e *envelope.Envelope) *authres.SPFResult {

	var reason string
	var from = e.MailFrom.Address

	result, err := spf.CheckHostWithSender(
		net.ParseIP(e.RemoteAddr.String()),
		e.Helo,
		from,
	)
	if err != nil {
		reason = err.Error()
	}

	var val authres.ResultValue = authres.ResultNone
	switch result {
	case spf.None:
		val = authres.ResultNone
	case spf.Neutral:
		val = authres.ResultNeutral
	case spf.Pass:
		val = authres.ResultPass
	case spf.Fail:
		val = authres.ResultFail
	case spf.SoftFail:
		val = authres.ResultSoftFail
	case spf.TempError:
		val = authres.ResultTempError
	case spf.PermError:
		val = authres.ResultPermError
	}

	return &authres.SPFResult{
		Value:  val,
		Reason: reason,
		From:   from,
		Helo:   e.Helo,
	}
}
