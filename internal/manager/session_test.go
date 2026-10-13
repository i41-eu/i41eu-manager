package manager

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Session ids are made by the manager when a phone signs in: an id the browser comes with (one
// another page or service on this host may have set) never becomes a sign-in, a page visit sets no
// cookie, and an id the manager does not know is no one's.
func TestSessionMadeAtSignIn(t *testing.T) {
	m := newManager(t, Config{AppsFile: catalog(t)})
	h := m.Handler()
	const id = "stackchan-000000000001"
	newCode := func(code string) {
		m.mu.Lock()
		m.robots[id] = &Robot{ID: id, Owner: localOwner.Key}
		m.pageCodes = map[string]pageCode{id: {code: code, expires: time.Now().Add(time.Minute)}}
		m.mu.Unlock()
	}
	get := func(path, session string) *http.Response {
		r := httptest.NewRequest("GET", path, nil)
		r.RemoteAddr, r.Host = "192.168.1.20:5000", "192.168.1.10:8790"
		if session != "" {
			r.AddCookie(&http.Cookie{Name: sessionCookie, Value: session})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Result()
	}
	cookie := func(res *http.Response) string {
		for _, c := range res.Cookies() {
			if c.Name == sessionCookie {
				return c.Value
			}
		}
		return ""
	}
	signedIn := func(session string) bool {
		res := get("/api/me", session)
		var b strings.Builder
		buf := make([]byte, 4096)
		for {
			n, err := res.Body.Read(buf)
			b.Write(buf[:n])
			if err != nil {
				break
			}
		}
		return strings.Contains(b.String(), `"signed_in":true`)
	}

	if c := cookie(get("/", "")); c != "" {
		t.Fatalf("a page visit set a session cookie: %q", c)
	}
	chosen := strings.Repeat("ab", 32)
	if signedIn(chosen) {
		t.Fatal("an unknown session id is signed in")
	}

	newCode("code-1")
	res := get("/phone?code=code-1", chosen)
	first := cookie(res)
	if res.StatusCode != http.StatusSeeOther || first == "" || first == chosen {
		t.Fatalf("sign-in: %d, cookie %q (came with %q)", res.StatusCode, first, chosen)
	}
	if signedIn(chosen) || !signedIn(first) {
		t.Fatalf("after sign-in: the chosen id signed in %v, the new one %v", signedIn(chosen), signedIn(first))
	}

	newCode("code-2") // the same phone signs in again: a new id, the old one ends
	second := cookie(get("/phone?code=code-2", first))
	if second == "" || second == first || signedIn(first) || !signedIn(second) {
		t.Fatalf("signing in again: %q after %q (old signed in %v)", second, first, signedIn(first))
	}

	r := httptest.NewRequest("POST", "/auth/logout", nil)
	r.RemoteAddr, r.Host = "192.168.1.20:5000", "192.168.1.10:8790"
	r.Header.Set("Origin", "http://192.168.1.10:8790")
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: second})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || signedIn(second) {
		t.Fatalf("sign-out: %d, still signed in %v", w.Code, signedIn(second))
	}
	if c := w.Result().Cookies(); len(c) != 1 || c[0].MaxAge >= 0 {
		t.Fatalf("sign-out does not clear the cookie: %+v", c)
	}
}
