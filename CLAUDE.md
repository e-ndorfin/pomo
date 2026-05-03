# Build

Always build the executable to `./pomo`:

```
go build -o ./pomo ./...
```

# Database

`started_at` in the sessions table stores the actual session start time (`time.Now() - elapsed` at recording time), not the wall-clock time of the `CreateSession` call. Historical records (before this fix) store the end time as `started_at`.
