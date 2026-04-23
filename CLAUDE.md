# Build

Always build the executable to `./pomo`:

```
go build -o ./pomo ./...
```

# Test

Run the focused package tests with:

```
go test ./cmd ./config ./ui ./db
```

If the default Go cache is not writable in this environment, use:

```
GOCACHE=/tmp/pomo-go-build-cache go test ./cmd ./config ./ui ./db
```

# Database

`started_at` in the sessions table stores the actual session start time (`time.Now() - elapsed` at recording time), not the wall-clock time of the `CreateSession` call. Historical records (before this fix) store the end time as `started_at`.
