# clock architecture

## Point

The system under test must never read the wall clock. It receives a
`Clock` at construction and asks that clock for the time and for sleeps.
Production passes `Real`. Tests pass `Fake`, which lets the test own time:
a sleep of one second costs nothing, and the test can inspect every sleep
the system asked for. This package carries pattern P2 in `docs/options.md`.

## Shape

One interface, two implementations, one owner each.

```
                     +----------------------------+
                     |  clock.Clock (interface)   |
                     |  Now()   time.Time         |
                     |  Sleep(d time.Duration)    |
                     +-------------+--------------+
                                   |
                 +-----------------+-----------------+
                 |                                   |
     +-----------v-----------+           +-----------v-----------+
     |  clock.Real           |           |  clock.Fake           |
     |  Now   -> time.Now()  |           |  now    time.Time     |
     |  Sleep -> time.Sleep  |           |  Sleeps []Duration    |
     +-----------+-----------+           |  Advance(d)           |
                 |                       +-----------+-----------+
                 |                                   |
        held by cmd/retrydemo                 held by the test
                 |                                   |
                 +--------------+   +----------------+
                                |   |
                        +-------v---v-------+
                        |  retry.Retrier    |   the system under test
                        |  clock clock.Clock|   sees only the interface
                        +-------------------+
```

The retrier holds the interface. It cannot tell which implementation it
got, so the same code runs in production and in the simulator.

## How it is used

The same retry run, once against each clock. The op fails twice and then
succeeds. Jitter is fixed at half the backoff to keep the numbers round.

```
  wall time                Real clock              Fake clock
  ---------   -------------------------------   ------------------------
  t=0         op() fails                        op() fails
              Sleep(100ms) blocks ......        Sleep(100ms) returns now
  t=100ms     op() fails                          now += 100ms
              Sleep(200ms) blocks ......          Sleeps = [100ms]
  t=300ms     op() succeeds                     op() fails
                                                Sleep(200ms) returns now
                                                  now += 200ms
                                                  Sleeps = [100ms 200ms]
                                                op() succeeds
  ---------   -------------------------------   ------------------------
  cost        300ms of real waiting             0ms, and a recorded log
```

Time inside an operation counts too. When a test calls `Advance` from
inside the op, the retrier sees that time pass through `Now`, exactly as
it would see a slow RPC in production.

```
   test                       Fake                       retrier
   ----                       ----                       -------
    |                          |                            |
    |  NewFake(epoch)          |                            |
    |------------------------->|                            |
    |                          |   Now()                    |
    |                          |<---------------------------|  deadline = now + Timeout
    |                          |                            |
    |                          |            op() runs       |
    |  Advance(600ms)          |                            |
    |------------------------->|  now += 600ms              |
    |                          |                            |
    |                          |   Now()                    |
    |                          |<---------------------------|  would the sleep end
    |                          |                            |  after the deadline?
    |                          |   Sleep(d) or give up      |
    |                          |<---------------------------|
    |  read clk.Sleeps         |                            |
    |------------------------->|                            |
```

Wiring in code:

```go
r := retry.New(policy, clock.Real{}, rng)          // production
r := retry.New(policy, clock.NewFake(epoch), rng)  // test
```

## What it proves

With `Fake` in place a test can check, without waiting and with exact
numbers. `projects/retry/invariants.md` owns the full list with IDs and test
names. In short:

- The sequence of sleeps matches the backoff schedule.
- Every sleep stays inside its jitter bounds.
- The clock never ends up past the deadline, so the retrier never sleeps
  through it.
- Time spent inside the operation counts against the deadline.

It also makes the seeded test practical. One seed drives thousands of
simulated runs in milliseconds, and a failure prints the seed to replay.

It does not prove that `Real` behaves. `time.Sleep` can oversleep, and the
retrier makes no promise about wall-clock accuracy. The demo in
`cmd/retrydemo` is the only place the real clock runs.

## Open items

- No timers. `Fake` only knows `Now`, `Sleep`, and `Advance`. The token
  bucket in project 2 needs only `Now`. Timers arrive with the scheduler
  in project 5 and later.
- `Sleep` on `Fake` moves shared time from inside the system under test.
  That works with one actor. Once several actors share a clock, the
  simulator must own time and wake actors by timer instead.
- `Sleeps` is a spy for the retrier. When `sim/trace` exists, sleeps
  become trace events and this field goes away.
