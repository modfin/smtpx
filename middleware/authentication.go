package middleware

import (
	"blitiri.com.ar/go/spf"
	"bytes"
	"fmt"
	"github.com/modfin/smtpx"
	"github.com/modfin/smtpx/envelope"
	"github.com/modfin/smtpx/middleware/authres"
	"github.com/modfin/smtpx/middleware/authres/dkim"
	"github.com/modfin/smtpx/middleware/authres/dmarc"
	"github.com/modfin/smtpx/utils"
	"golang.org/x/net/publicsuffix"
	"log/slog"
	"net"
	"net/mail"
	"strings"
)

func AddAuthenticationResult(hostname string, logger *slog.Logger) smtpx.Middleware {
	log := func(s string) {
		if logger != nil {
			logger.Debug(s)
		}
	}
	return func(next smtpx.HandlerFunc) smtpx.HandlerFunc {
		return func(e *envelope.Envelope) smtpx.Response {

			spfres := spfCheck(e)
			dkims := dkimCheck(e)
			dmarc := dmarcCheck(e, spfres, dkims)

			res := []authres.Result{spfres, dmarc}
			for _, d := range dkims {
				res = append(res, d)
			}

			val := authres.Format(hostname, res)
			_ = e.PrependHeader("Authentication-Results", val)

			log("Authentication-Results: " + val)

			return next(e)
		}
	}
}

func dmarcCheck(e *envelope.Envelope, spf *authres.SPFResult, dkims []*authres.DKIMResult) *authres.DMARCResult {
	domain := utils.DomainOfEmail(e.MailFrom)

	// Helper function for error cases
	createErrorResult := func(reason string) *authres.DMARCResult {
		return &authres.DMARCResult{
			Value:  authres.ResultNone,
			Reason: reason,
			From:   domain,
		}
	}

	m, err := e.Mail()
	if err != nil {
		return createErrorResult("unable to get mail")
	}
	// Get headers
	headers, err := m.Headers()
	if err != nil {
		fmt.Println(err)
		return createErrorResult("unable to get headers")
	}

	// Parse From header
	fromHeader := headers.Get("From")
	from, err := mail.ParseAddress(fromHeader)
	if err != nil || len(fromHeader) == 0 {
		return createErrorResult("uo From header")
	}
	fromDomain := utils.DomainOfEmail(from)

	// Lookup DMARC record
	dmarcRecord, err := dmarc.Lookup(fromDomain)
	if err != nil {
		return createErrorResult("DMARC lookup failed")
	}

	// Parse SPF domain
	spfFrom, err := mail.ParseAddress(spf.From)
	if err != nil {
		return createErrorResult("Unable to parse spf from")
	}
	spfDomain := utils.DomainOfEmail(spfFrom)

	// Check SPF alignment
	spfAligned := checkSPFAlignment(spf, dmarcRecord, spfDomain, fromDomain)

	// Check DKIM alignment
	dkimAligned := checkDKIMAlignment(dkims, dmarcRecord, fromDomain)

	// Determine DMARC result
	result := determineDMARCResult(spfAligned, dkimAligned, spf, dmarcRecord)
	result.From = domain

	return result
}

// Check if SPF is aligned according to DMARC policy
func checkSPFAlignment(spf *authres.SPFResult, r *dmarc.Record, spfDomain, fromDomain string) bool {
	// If SPF didn't pass, it can't be aligned
	if spf.Value != authres.ResultPass {
		return false
	}

	// Strict alignment - domains must exactly match
	if r.SPFAlignment == dmarc.AlignmentStrict {
		return spfDomain == fromDomain
	}

	// Relaxed alignment - check for suffix or organizational domain match
	if r.SPFAlignment == dmarc.AlignmentRelaxed {
		// Check if spfDomain is a subdomain of fromDomain
		if strings.HasSuffix(spfDomain, fromDomain) {
			return true
		}

		// Check organizational domain match
		spfOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(spfDomain)
		if err != nil {
			return false
		}

		fromOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(fromDomain)
		if err != nil {
			return false
		}

		return spfOrgDomain == fromOrgDomain
	}

	return false
}

// Check if any DKIM signature is aligned according to DMARC policy
func checkDKIMAlignment(dkims []*authres.DKIMResult, r *dmarc.Record, fromDomain string) bool {
	if dkims == nil {
		return false
	}

	for _, dkim := range dkims {
		// DKIM must pass to be considered for alignment
		if dkim.Value != authres.ResultPass {
			continue
		}

		// Strict alignment - domains must exactly match
		if r.DKIMAlignment == dmarc.AlignmentStrict && dkim.Domain == fromDomain {
			return true
		}

		// Relaxed alignment - check organizational domain match
		if r.DKIMAlignment == dmarc.AlignmentRelaxed {
			dkimOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(dkim.Domain)
			if err != nil {
				continue
			}

			fromOrgDomain, err := publicsuffix.EffectiveTLDPlusOne(fromDomain)
			if err != nil {
				continue
			}

			if dkimOrgDomain == fromOrgDomain {
				return true
			}
		}
	}

	return false
}

// Determine the final DMARC result based on alignment results and policy
func determineDMARCResult(spfAligned, dkimAligned bool, spf *authres.SPFResult, r *dmarc.Record) *authres.DMARCResult {
	var value authres.ResultValue
	var reasons []string

	// If either SPF or DKIM is aligned, DMARC passes
	if spfAligned || dkimAligned {
		value = authres.ResultPass
		reasons = []string{"DMARC check passed"}
	} else {
		value = authres.ResultFail
		reasons = []string{"DMARC check failed"}

		// Add specific failure reasons
		if spf == nil || spf.Value != authres.ResultPass {
			reasons = append(reasons, fmt.Sprintf("SPF failed or not aligned: %s", spf.Reason))
		}

		if !dkimAligned {
			reasons = append(reasons, "DKIM failed or not aligned")
		}
	}

	// Override if policy is "none"
	if r.Policy == dmarc.PolicyNone {
		value = authres.ResultNone
		reasons = []string{"DMARC policy is none"}
	}

	return &authres.DMARCResult{
		Value:  value,
		Reason: strings.Join(reasons, ", "),
	}
}

func dkimCheck(e *envelope.Envelope) []*authres.DKIMResult {

	buf := bytes.NewBuffer(e.Data.Bytes())
	verifications, err := dkim.Verify(buf)

	if err != nil {
		return nil
	}

	var res []*authres.DKIMResult

	for _, v := range verifications {
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

	ip, _, _ := strings.Cut(e.RemoteAddr.String(), ":")

	result, err := spf.CheckHostWithSender(
		net.ParseIP(ip),
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
