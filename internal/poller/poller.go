package poller

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/client"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
)

type Channels struct {
	Chain  chan []*model.Receipt
	Snaps  chan []*model.Snapshot
	Policy chan []*model.Policy
	Token  chan []*model.Token
	MCP    chan *model.MCPStatus
	MCPLog chan []*model.MCPLogEntry
	Daemon chan *model.DaemonStatus
	Errors chan error
}

type Poller struct {
	client client.Client
	C      Channels
	paused atomic.Bool
	stop   chan struct{}
	wg     sync.WaitGroup
}

func New(c client.Client) *Poller {
	return &Poller{
		client: c,
		C: Channels{
			Chain:  make(chan []*model.Receipt, 1),
			Snaps:  make(chan []*model.Snapshot, 1),
			Policy: make(chan []*model.Policy, 1),
			Token:  make(chan []*model.Token, 1),
			MCP:    make(chan *model.MCPStatus, 1),
			MCPLog: make(chan []*model.MCPLogEntry, 1),
			Daemon: make(chan *model.DaemonStatus, 1),
			Errors: make(chan error, 64),
		},
		stop: make(chan struct{}),
	}
}

func (p *Poller) Start() {
	p.poll("daemon", 1*time.Second, func() (any, error) { return p.client.DaemonStatus() }, p.C.Daemon)
	p.poll("chain", 2*time.Second, func() (any, error) { return p.client.ListReceipts(50, "") }, p.C.Chain)
	p.poll("mcp", 2*time.Second, func() (any, error) { return p.client.MCPStatus() }, p.C.MCP)
	p.poll("mcp_log", 2*time.Second, func() (any, error) { return p.client.MCPLog(20, "", false) }, p.C.MCPLog)
	p.poll("token", 1*time.Second, func() (any, error) { return p.client.ListTokens() }, p.C.Token)
	p.poll("snaps", 5*time.Second, func() (any, error) { return p.client.ListSnapshots("", 20) }, p.C.Snaps)
	p.poll("policy", 5*time.Second, func() (any, error) { return p.client.ListPolicies() }, p.C.Policy)
}

func (p *Poller) poll(name string, interval time.Duration, fn func() (any, error), ch any) {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-t.C:
				if p.paused.Load() {
					continue
				}
				result, err := fn()
				if err != nil {
					select {
					case p.C.Errors <- err:
					default:
					}
					continue
				}
				switch c := ch.(type) {
				case chan *model.DaemonStatus:
					if v, ok := result.(*model.DaemonStatus); ok {
						select {
						case c <- v:
						default:
						}
					}
				case chan []*model.Receipt:
					if v, ok := result.([]*model.Receipt); ok {
						select {
						case c <- v:
						default:
						}
					}
				case chan *model.MCPStatus:
					if v, ok := result.(*model.MCPStatus); ok {
						select {
						case c <- v:
						default:
						}
					}
				case chan []*model.MCPLogEntry:
					if v, ok := result.([]*model.MCPLogEntry); ok {
						select {
						case c <- v:
						default:
						}
					}
				case chan []*model.Token:
					if v, ok := result.([]*model.Token); ok {
						select {
						case c <- v:
						default:
						}
					}
				case chan []*model.Snapshot:
					if v, ok := result.([]*model.Snapshot); ok {
						select {
						case c <- v:
						default:
						}
					}
				case chan []*model.Policy:
					if v, ok := result.([]*model.Policy); ok {
						select {
						case c <- v:
						default:
						}
					}
				}
			}
		}
	}()
}

func (p *Poller) Stop() {
	close(p.stop)
	p.wg.Wait()
}

func (p *Poller) Pause() {
	p.paused.Store(true)
}

func (p *Poller) Resume() {
	p.paused.Store(false)
}

func (p *Poller) ForceRefresh() {
	// drain channels
	drain := func(ch any) {
		switch c := ch.(type) {
		case chan *model.DaemonStatus:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		case chan []*model.Receipt:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		case chan *model.MCPStatus:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		case chan []*model.MCPLogEntry:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		case chan []*model.Token:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		case chan []*model.Snapshot:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		case chan []*model.Policy:
			for {
				select {
				case <-c:
				default:
					return
				}
			}
		}
	}
	drain(p.C.Daemon)
	drain(p.C.Chain)
	drain(p.C.MCP)
	drain(p.C.MCPLog)
	drain(p.C.Token)
	drain(p.C.Snaps)
	drain(p.C.Policy)
	// drain pending errors too
	for {
		select {
		case <-p.C.Errors:
		default:
			goto afterDrain
		}
	}
afterDrain:
	// immediate poll (non-blocking, just one shot per subsystem)
	go func() {
		if v, err := p.client.DaemonStatus(); err == nil {
			select {
			case p.C.Daemon <- v:
			default:
			}
		}
	}()
	go func() {
		if v, err := p.client.ListReceipts(50, ""); err == nil {
			select {
			case p.C.Chain <- v:
			default:
			}
		}
	}()
	go func() {
		if v, err := p.client.MCPStatus(); err == nil {
			select {
			case p.C.MCP <- v:
			default:
			}
		}
	}()
	go func() {
		if v, err := p.client.ListTokens(); err == nil {
			select {
			case p.C.Token <- v:
			default:
			}
		}
	}()
	go func() {
		if v, err := p.client.ListSnapshots("", 20); err == nil {
			select {
			case p.C.Snaps <- v:
			default:
			}
		}
	}()
	go func() {
		if v, err := p.client.ListPolicies(); err == nil {
			select {
			case p.C.Policy <- v:
			default:
			}
		}
	}()
}
