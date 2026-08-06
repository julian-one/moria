# Graceful shutdown — a walkthrough

Two passes: first the idea like you're five, then the actual code so the
implementation in `cmd/serve.go` reads as no surprise.

## Part 1: Like you're five

Moria is a restaurant. Requests are customers. Kubernetes is the landlord.

**How closing time works today (the bad way):** the landlord knocks
("SIGTERM — time to close"). Our restaurant has no plan for this, so the
whole staff instantly vanishes. Customers mid-bite are thrown onto the
street with their plates (in-flight requests get their connections cut;
someone's login just dies with a network error).

**How it should work (graceful):** when the knock comes,

1. **Lock the front door.** No new customers get in (the listener stops
   accepting connections).
2. **Let seated customers finish eating.** Every request already being
   handled runs to completion and gets its response.
3. **Then turn off the lights and leave.** The process exits cleanly, code 0.
4. **The 15-minute rule.** If someone is *still* chewing after the grace
   window, we escort them out anyway — because the landlord doesn't wait
   forever: 30 seconds after the knock, Kubernetes bulldozes the building
   (SIGKILL). Our own deadline must be shorter than the bulldozer's.

And why timeouts on the door, even when we're open? Without them, a
prankster can stand in the doorway forever, holding it open and never
ordering (a "slowloris" attack: open a connection, send headers infinitely
slowly, tie up the server). A door that auto-closes after a few seconds
shrugs that off.

That's the whole feature: **stop taking new work, finish existing work,
leave before the bulldozer, and never let anyone hold the door open.**

## Part 2: Meet the code

What `runServe` in `cmd/serve.go` does today, trimmed to its skeleton:

```go
errCh := make(chan error, 1)

go func() { errCh <- http.ListenAndServe(":"+port, handler) }()

return <-errCh // block until the server dies
```

Three problems hide in these few lines:

- `http.ListenAndServe` is a shortcut that builds a server we never get a
  handle to — so there is nothing to call "please stop nicely" on.
- That hidden server has **no timeouts at all** (doorway prankster welcome).
- Nobody listens for SIGTERM, so the Go runtime's default applies: kill the
  process on the spot, mid-request.

The fix is four pieces, each small:

**1. `signal.NotifyContext` — the doorbell wire.** Converts an OS signal
into a context turning off. Everything already flows from `ctx` (the session
purge loop takes it too), so one wire shuts down the whole building:

```go
ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

**2. `http.Server` — own the server, set the timeouts.** Same handler,
but now we hold the struct, which gives us both the knobs and a `Shutdown`
method to call later:

```go
srv := &http.Server{
    Addr:              ":" + port,
    Handler:           handler,
    ReadHeaderTimeout: 5 * time.Second,  // anti-slowloris: send headers promptly or leave
    ReadTimeout:       10 * time.Second, // whole request must arrive within this
    WriteTimeout:      30 * time.Second, // response must finish (emails make some handlers slow)
    IdleTimeout:       2 * time.Minute,  // keep-alive connections can't squat forever
}
```

**3. `errgroup` — a supervisor for goroutines.** Replaces the hand-rolled
`errCh`. Start workers with `g.Go`, then `g.Wait()` blocks until all finish
and hands back the first error. Bonus: if any worker fails, the group's
context cancels, which the others can react to:

```go
g, gctx := errgroup.WithContext(ctx)
g.Go(func() error { /* run the server */ })
g.Go(func() error { /* wait for gctx, then shut down (next piece) */ })
return g.Wait()
```

**4. `srv.Shutdown` — the graceful close itself.** One more worker sits
quietly until the doorbell rings (`gctx.Done()` — SIGTERM arrived *or* the
server crashed), then tells the server to lock the door and wait for
eaters, giving up after 15 seconds (safely inside kubernetes' 30-second
SIGKILL deadline):

```go
g.Go(func() error {
    <-gctx.Done()
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()
    return srv.Shutdown(shutdownCtx)
})
```

One subtlety: after `Shutdown` is called, `ListenAndServe` returns
`http.ErrServerClosed`. That's the "we closed on purpose" receipt, not a
failure — the server workers must treat it as success or every clean
shutdown would report an error.

### A rollout, start to finish

```
kubectl rollout restart deployment moria
  → k8s sends SIGTERM to the pod
  → signal.NotifyContext cancels ctx
  → the server stops accepting; in-flight requests finish
  → the session purge loop sees ctx.Done() and exits
  → g.Wait() returns nil, process exits 0
  → k8s starts the new pod   (SIGKILL at +30s never fires)
```
