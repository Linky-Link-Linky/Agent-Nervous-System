package poller

import (
    "testing"
    "time"
    "github.com/Linky-Link-Linky/Agent-Nervous-System/internal/client"
)

func TestPoller_Pause(t *testing.T) {
    c := client.NewMock()
    p := New(c)
    p.Start()
    time.Sleep(100 * time.Millisecond)
    p.Pause()
    // drain any queued messages
    drain := func() {
        for { select { case <-p.C.Daemon: default: return } }
    }
    drain()
    // after pause, no new message should arrive within 500ms
    select {
    case <-p.C.Daemon:
        t.Fatal("received message while paused")
    case <-time.After(500 * time.Millisecond):
    }
    p.Stop()
}

func TestPoller_Resume(t *testing.T) {
    c := client.NewMock()
    p := New(c)
    p.Start()
    p.Pause()
    time.Sleep(50 * time.Millisecond)
    p.Resume()
    // should get a message within daemon interval + margin
    select {
    case <-p.C.Daemon:
    case <-time.After(3 * time.Second):
        t.Fatal("no message received after resume")
    }
    p.Stop()
}

func TestPoller_Stop(t *testing.T) {
    c := client.NewMock()
    p := New(c)
    p.Start()
    p.Stop()
    // should not panic
    done := make(chan struct{})
    go func() {
        p.wg.Wait()
        close(done)
    }()
    select {
    case <-done:
    case <-time.After(3 * time.Second):
        t.Fatal("goroutines did not exit")
    }
}
