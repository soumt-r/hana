package stdimpl

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/soumt-r/hana/errs"
)

// The text is UTF-8: base64 and URL escaping work on its bytes.

var base64Shape = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)

func encodingBase64Encode(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	return base64.StdEncoding.EncodeToString([]byte(s)), nil
}

// encodingBase64Decode reads standard base64 (with padding, no line breaks)
// whose bytes are valid UTF-8 text.
func encodingBase64Decode(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	if len(s)%4 != 0 || !base64Shape.MatchString(s) {
		return nil, errs.New(errs.Base64Invalid)
	}
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil || !utf8.Valid(raw) {
		return nil, errs.New(errs.Base64Invalid)
	}
	return string(raw), nil
}

func isURLUnreserved(b byte) bool {
	return b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b == '_' || b == '.' || b == '~'
}

// encodingURLEncode escapes everything but letters, digits and - _ . ~ as
// %XX (uppercase), so a space is %20.
func encodingURLEncode(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	for _, b := range []byte(s) {
		if isURLUnreserved(b) {
			out.WriteByte(b)
		} else {
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String(), nil
}

func encodingURLDecode(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	raw := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '%' {
			raw = append(raw, s[i])
			continue
		}
		if i+2 >= len(s) {
			return nil, errs.New(errs.URLDecodeInvalid)
		}
		b, err := hex.DecodeString(s[i+1 : i+3])
		if err != nil {
			return nil, errs.New(errs.URLDecodeInvalid)
		}
		raw = append(raw, b[0])
		i += 2
	}
	if !utf8.Valid(raw) {
		return nil, errs.New(errs.URLDecodeInvalid)
	}
	return string(raw), nil
}

// hashSHA256 is the SHA-256 of the text's UTF-8 bytes as 64 lowercase hex digits.
func hashSHA256(args []interface{}) (interface{}, error) {
	if err := exactly(args, 1); err != nil {
		return nil, err
	}
	s, err := stringArg(args, 0)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:]), nil
}

// randomUUID is a random (version 4) UUID like 3b241101-e2bb-4255-8caf-4136c566a962.
func randomUUID(args []interface{}) (interface{}, error) {
	if err := exactly(args, 0); err != nil {
		return nil, err
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return nil, err
	}
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}
