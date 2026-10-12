# i41eu-manager

A home's own Stackchan manager (no sign-in: the computer it runs on is the owner, phones sign in at
a robot): robots, the app catalog, setup over USB, tokens per robot per app, `robot-auth` for the
apps, and the link up to an upstream manager (e.g. m.i41.eu). See [README.md](README.md). Design: i41eu-arch
`docs/stackchan-sites.md`.

- Go only: the standard library, gorilla/websocket, go.bug.st/serial (`i41eu-usb`), and q41eu-raw's `statestore` (Postgres through pgx).
- Apps (e.g. q41eu-raw) call `POST /api/robot-auth`; change it together with their client (`internal/server/manager.go` in q41eu-raw).
- The setup page talks to the robot with the USB setup protocol of the firmware (`usb_setup.h` in i41eu-embody).
- Before committing, run `gofmt -l .`, `go vet ./...` and `go test -race ./...`.
