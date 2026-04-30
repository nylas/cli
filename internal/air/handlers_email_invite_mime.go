package air

import (
	"bytes"
	"encoding/base64"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
	"strings"
)

// inlineCalendarPart is an iCalendar (text/calendar) MIME part extracted
// from a raw RFC822 message. Gmail invites ship the ICS as an inline
// part of multipart/alternative; Nylas does not surface those in the
// attachments[] list, so we have to walk raw_mime ourselves.
type inlineCalendarPart struct {
	Body      string
	Filename  string
	ContentID string
	Method    string // upper-case METHOD parameter (REQUEST / CANCEL / REPLY)
}

// maxRawMIMEBytes caps how much of a raw MIME blob we'll walk. Nylas
// returns base64-decoded MIME, so 5 MB covers any normal invitation
// while keeping memory predictable when an attacker stitches a giant
// message.
const maxRawMIMEBytes = 5 * 1024 * 1024

// maxMIMEDepth caps multipart recursion. RFC 5322 imposes no limit;
// real invitations nest at most 2 levels (mixed → alternative → calendar).
const maxMIMEDepth = 8

// findInlineCalendarParts walks the raw RFC822/MIME message and returns
// every text/calendar part it finds, decoded. Returns nil on any parse
// failure — callers treat "can't parse" the same as "no invite present".
func findInlineCalendarParts(rawMIME string) []inlineCalendarPart {
	if rawMIME == "" || len(rawMIME) > maxRawMIMEBytes {
		return nil
	}
	msg, err := mail.ReadMessage(strings.NewReader(rawMIME))
	if err != nil {
		return nil
	}
	var out []inlineCalendarPart
	walkMIMEForCalendar(textproto.MIMEHeader(msg.Header), msg.Body, &out, 0)
	return out
}

// walkMIMEForCalendar recursively descends multipart bodies and appends
// any text/calendar leaf to out. Depth-capped to defend against
// pathological MIME nesting.
func walkMIMEForCalendar(header textproto.MIMEHeader, body io.Reader, out *[]inlineCalendarPart, depth int) {
	if depth > maxMIMEDepth {
		return
	}
	mediaType, params, err := mime.ParseMediaType(header.Get("Content-Type"))
	if err != nil {
		return
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return
		}
		mr := multipart.NewReader(body, boundary)
		for {
			part, err := mr.NextPart()
			if err != nil {
				return
			}
			walkMIMEForCalendar(part.Header, part, out, depth+1)
		}
	}

	if !strings.EqualFold(mediaType, "text/calendar") {
		return
	}

	raw, err := io.ReadAll(io.LimitReader(body, maxRawMIMEBytes+1))
	if err != nil || len(raw) > maxRawMIMEBytes {
		return
	}
	decoded := decodePartBody(raw, header.Get("Content-Transfer-Encoding"))

	filename := params["name"]
	if filename == "" {
		if cd := header.Get("Content-Disposition"); cd != "" {
			if _, dParams, err := mime.ParseMediaType(cd); err == nil {
				filename = dParams["filename"]
			}
		}
	}
	cid := strings.TrimSuffix(strings.TrimPrefix(header.Get("Content-ID"), "<"), ">")
	method := strings.ToUpper(strings.TrimSpace(params["method"]))

	*out = append(*out, inlineCalendarPart{
		Body:      string(decoded),
		Filename:  filename,
		ContentID: cid,
		Method:    method,
	})
}

// decodePartBody applies the Content-Transfer-Encoding to a part body.
// 7bit/8bit/binary pass through unchanged. base64 and quoted-printable
// failures fall back to the raw bytes so we still attempt parsing — the
// iCalendar parser will reject garbage cleanly downstream.
func decodePartBody(raw []byte, cte string) []byte {
	switch strings.ToLower(strings.TrimSpace(cte)) {
	case "base64":
		// MIME base64 may include CR/LF runs; strip whitespace so the
		// permissive decoders accept it. Try standard then raw to handle
		// senders that omit padding.
		clean := stripBase64Whitespace(raw)
		if dec, err := base64.StdEncoding.DecodeString(string(clean)); err == nil {
			return dec
		}
		if dec, err := base64.RawStdEncoding.DecodeString(string(clean)); err == nil {
			return dec
		}
		return raw
	case "quoted-printable":
		dec, err := io.ReadAll(quotedprintable.NewReader(bytes.NewReader(raw)))
		if err != nil {
			return raw
		}
		return dec
	default:
		return raw
	}
}

// stripBase64Whitespace removes CR, LF, tabs, and spaces from a base64
// blob so the decoders don't trip on standard MIME line wrapping.
func stripBase64Whitespace(b []byte) []byte {
	out := make([]byte, 0, len(b))
	for _, c := range b {
		if c == '\r' || c == '\n' || c == '\t' || c == ' ' {
			continue
		}
		out = append(out, c)
	}
	return out
}
