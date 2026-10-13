package manager

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A phone signed in at a robot is an owner, but what only the computer this manager runs on may
// do stays there: setting a robot up (every token of the robot comes back) and the link to the
// upstream manager (it is the computer's, and outlasts a phone's sign-in).
func TestPhoneCannotSetUpOrLink(t *testing.T) {
	m := newManager(t, Config{AppsFile: catalog(t), UpstreamURL: "https://upstream.example"})
	h := m.Handler()
	phone := strings.Repeat("7", 64)
	m.mu.Lock()
	m.phones = map[string]time.Time{phone: time.Now()}
	m.linkSt = linkState{Token: "link-token", Home: "h-1", Account: "owner@example", Mode: "read-only", Since: time.Now()}
	m.mu.Unlock()

	fromPhone := func(method, path, body string) (int, string) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.RemoteAddr, r.Host = "192.168.1.20:5000", "192.168.1.10:8790"
		r.Header.Set("Origin", "http://192.168.1.10:8790")
		r.Header.Set("Content-Type", "application/json")
		r.AddCookie(&http.Cookie{Name: sessionCookie, Value: phone})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		b, _ := io.ReadAll(w.Result().Body)
		return w.Code, string(b)
	}
	if code, body := fromPhone("GET", "/api/me", ""); code != http.StatusOK || !strings.Contains(body, `"phone":true`) {
		t.Fatalf("the phone is not signed in: %d %s", code, body)
	}
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/api/my/robots/stackchan-000000000001/setup", `{"apps":["raw"]}`},
		{"POST", "/api/link/start", `{"mode":"full","return":"http://192.168.1.10:8790"}`},
		{"GET", "/link/done?code=c&state=s", ""},
		{"POST", "/api/link/settings", `{"mode":"full","share_local":true}`},
		{"POST", "/api/link/unlink", ""},
	} {
		if code, body := fromPhone(c.method, c.path, c.body); code != http.StatusForbidden {
			t.Errorf("%s %s from a phone: %d %s, want 403", c.method, c.path, code, body)
		}
	}
	m.mu.Lock()
	st, robots := m.linkSt, len(m.robots)
	m.mu.Unlock()
	if st.Token != "link-token" || st.Mode != "read-only" || st.ShareLocal || st.Pending != "" || robots != 0 {
		t.Fatalf("a phone changed the link or the robots: %+v, %d robots", st, robots)
	}

	// This computer may; a second link over an existing one is refused (unlink first).
	if code, body := call(t, h, "POST", "/api/link/start", `{"mode":"full","return":"http://localhost:8790"}`, ""); code != http.StatusConflict {
		t.Fatalf("link start while linked: %d %s, want 409", code, body)
	}
	if code, body := call(t, h, "POST", "/api/link/settings", `{"mode":"full"}`, ""); code != http.StatusNoContent {
		t.Fatalf("settings from this computer: %d %s", code, body)
	}
	if code, body := call(t, h, "POST", "/api/link/unlink", "", ""); code != http.StatusNoContent {
		t.Fatalf("unlink from this computer: %d %s", code, body)
	}
	if code, body := call(t, h, "POST", "/api/link/start", `{"mode":"full","return":"http://localhost:8790"}`, ""); code != http.StatusOK || !strings.Contains(body, "/link/approve?") {
		t.Fatalf("link start from this computer: %d %s", code, body)
	}
}
