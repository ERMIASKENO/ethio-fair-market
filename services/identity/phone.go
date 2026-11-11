package main

import (
	"errors"
	"strings"
)

// NormalizePhone converts Ethiopian mobile numbers to E.164 (+2519XXXXXXXX / +2517XXXXXXXX).
// Accepted inputs: 0911234567, 911234567, 251911234567, +251 91 123 4567, 0711234567 (Safaricom).
func NormalizePhone(in string) (string, error) {
	var b strings.Builder
	for _, r := range in {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		} else if r != ' ' && r != '-' && r != '+' && r != '(' && r != ')' {
			return "", errors.New("phone number contains invalid characters")
		}
	}
	d := b.String()
	switch {
	case strings.HasPrefix(d, "251") && len(d) == 12:
		d = d[3:]
	case strings.HasPrefix(d, "0") && len(d) == 10:
		d = d[1:]
	}
	if len(d) != 9 || (d[0] != '9' && d[0] != '7') {
		return "", errors.New("enter an Ethiopian mobile number, e.g. 0911 234 567")
	}
	return "+251" + d, nil
}
