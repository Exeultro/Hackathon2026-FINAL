package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"time"

	"aba/internal/store"
)

// mediaKey signs video and thumbnail links: <video> and <img> load them without the caller's headers
var mediaKey []byte

const mediaLinkTTL = 24 * time.Hour

func mediaSig(lectureID string, exp int64) string {
	m := hmac.New(sha256.New, mediaKey)
	m.Write([]byte(lectureID + "." + strconv.FormatInt(exp, 10)))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

// signedMediaURL is valid for at least mediaLinkTTL; the expiry is rounded to the hour so the link stays cacheable
func signedMediaURL(lectureID, kind string) string {
	exp := time.Now().Add(mediaLinkTTL + time.Hour).Truncate(time.Hour).Unix()
	return PublicURL + "/api/v1/lectures/" + lectureID + "/" + kind +
		"?exp=" + strconv.FormatInt(exp, 10) + "&sig=" + mediaSig(lectureID, exp)
}

// mediaAllowed: shared lectures are open; a private one needs its owner's credentials or a valid signed link
func mediaAllowed(r *http.Request, l *store.Lecture) bool {
	if store.Visible(l.Owner, viewer(r)) {
		return true
	}
	q := r.URL.Query()
	exp, err := strconv.ParseInt(q.Get("exp"), 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(q.Get("sig")), []byte(mediaSig(l.ID, exp)))
}
