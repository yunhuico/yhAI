// Package smtp handles mail sending via smtp,
// including composing of mail receivers, content, attachments, etc.
//
// Implementation references:
//
// * Standard for ARPA Internet Text Messages https://www.rfc-editor.org/rfc/rfc822
// * MIME (Multipurpose Internet Mail Extensions) https://datatracker.ietf.org/doc/html/rfc1341
// * Multipurpose Internet Mail Extensions (MIME) Part One: Format of Internet Message Bodies https://www.rfc-editor.org/rfc/rfc2045
// * Multipurpose Internet Mail Extensions (MIME) Part Two: Media Types https://www.rfc-editor.org/rfc/rfc2046
// * Multipurpose Internet Mail Extensions (MIME) Part Five: Conformance Criteria and Examples https://www.rfc-editor.org/rfc/rfc2049
// * MIME (Multipurpose Internet Mail Extensions) Part Three: Message Header Extensions for Non-ASCII Text https://www.rfc-editor.org/rfc/rfc2047
// * Internet Message Format https://www.rfc-editor.org/rfc/rfc5322.html
package smtp
