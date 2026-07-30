// Copyright 2023 National Technology & Engineering Solutions of Sandia, LLC (NTESS).
// Under the terms of Contract DE-NA0003525 with NTESS, the U.S. Government retains
// certain rights in this software.

package igorserver

import (
	"errors"
	"html/template"
	"testing"

	"github.com/stretchr/testify/assert"
)

// dedupeEmailList runs its input through common.Set, which silently discards the empty
// string and whitespace-only entries. A recipient list can therefore look populated to
// a len() check and still resolve to zero addresses.
func TestDedupeEmailListDropsUnusableAddresses(t *testing.T) {

	assert.Empty(t, dedupeEmailList([]string{""}),
		"a lone empty address should resolve to no recipients")
	assert.Empty(t, dedupeEmailList([]string{"   "}),
		"a whitespace-only address should resolve to no recipients")
	assert.Equal(t, []string{"a@b.com"}, dedupeEmailList([]string{"", "a@b.com", "a@b.com"}),
		"usable addresses should survive alongside dropped ones, deduped")
}

// A user with no email on file (igor-admin is seeded that way) yields a one-element
// recipient list holding the empty string. Before the fix that passed the len() guard,
// was emptied by dedupeEmailList, and reached gomail as a header with no addresses --
// which sends MAIL FROM and then DATA with no RCPT TO in between, drawing a
// "503 5.5.2 Need rcpt command" from the SMTP server.
func TestSendEmailRejectsUnusableRecipients(t *testing.T) {

	// point at a port nothing listens on so an unintended dial fails fast instead of hanging
	igor.Email.SmtpServer = "127.0.0.1"
	igor.Email.SmtpPort = 1
	tmpl := template.Must(template.New("test").Parse("test body"))

	cases := []struct {
		name    string
		toList  []string
		ccList  []string
		bccList []string
	}{
		{"empty owner address", []string{""}, nil, nil},
		{"whitespace owner address", []string{"  "}, nil, nil},
		{"unusable across all three headers", []string{""}, []string{" "}, []string{""}},
		{"no addresses at all", nil, nil, nil},
	}

	for _, c := range cases {
		err := sendEmail(tmpl, "test subject", c.toList, c.ccList, c.bccList, false, struct{}{})
		var noRcpt *NoEmailRecipientError
		assert.True(t, errors.As(err, &noRcpt),
			"%s: expected NoEmailRecipientError, got %v", c.name, err)
	}
}

// The guard must not swallow a message that does have somewhere to go. This one gets as
// far as the dialer and fails there, which is the correct place to fail.
func TestSendEmailAcceptsUsableRecipient(t *testing.T) {

	igor.Email.SmtpServer = "127.0.0.1"
	igor.Email.SmtpPort = 1
	tmpl := template.Must(template.New("test").Parse("test body"))

	err := sendEmail(tmpl, "test subject", []string{""}, []string{"someone@example.com"}, nil, false, struct{}{})
	var noRcpt *NoEmailRecipientError
	assert.False(t, errors.As(err, &noRcpt),
		"a usable Cc address should not trip the no-recipient guard, got %v", err)
}
