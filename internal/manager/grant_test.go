package manager

import (
	"encoding/json"
	"testing"
)

// A grant from the upstream manager is applied only as the answer to a Grant this home asked for,
// for that robot and app; and a grant never changes an app the robot has already.
func TestGrantOnlyWhenAsked(t *testing.T) {
	m := newManager(t, Config{AppsFile: catalog(t), UpstreamURL: "https://upstream.example"})
	const id = "stackchan-000000000001"
	m.mu.Lock()
	m.robots[id] = &Robot{ID: id, Owner: localOwner.Key}
	m.mu.Unlock()
	granted := func(body string) { m.homeHandle("Granted", json.RawMessage(body)) }
	app := func() *UpApp {
		m.mu.Lock()
		defer m.mu.Unlock()
		return m.robots[id].Upstream["raw"]
	}

	granted(`{"id":"w-0","robot":"` + id + `","app":"raw","name":"Raw data","url":"wss://other.example","token":"t0"}`)
	if a := app(); a != nil {
		t.Fatalf("a grant nobody asked for was applied: %+v", a)
	}

	m.mu.Lock()
	m.grants = map[string]upGrant{"w-1": {robot: id, app: "raw"}}
	m.mu.Unlock()
	granted(`{"id":"w-1","robot":"stackchan-000000000002","app":"raw","name":"Raw data","url":"wss://other.example","token":"t1"}`)
	granted(`{"id":"w-1","robot":"` + id + `","app":"pet","name":"Pet","url":"wss://other.example","token":"t1"}`)
	if a := app(); a != nil {
		t.Fatalf("an answer for another robot or app was applied: %+v", a)
	}
	granted(`{"id":"w-1","robot":"` + id + `","app":"raw","name":"Raw data","url":"wss://raw.example","token":"t1"}`)
	if a := app(); a == nil || a.URL != "wss://raw.example" || a.Token != "t1" {
		t.Fatalf("the grant asked for: %+v", a)
	}

	granted(`{"id":"w-1","robot":"` + id + `","app":"raw","name":"Raw data","url":"wss://other.example","token":"t2"}`)
	m.mu.Lock()
	m.grants = map[string]upGrant{"w-2": {robot: id, app: "raw"}}
	m.mu.Unlock()
	granted(`{"id":"w-2","robot":"` + id + `","app":"raw","name":"Raw data","url":"wss://other.example","token":"t2"}`)
	if a := app(); a.URL != "wss://raw.example" || a.Token != "t1" {
		t.Fatalf("a grant changed the app the robot has: %+v", a)
	}
}
