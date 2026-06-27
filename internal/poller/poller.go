package poller

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/client"
	"github.com/Linky-Link-Linky/Agent-Nervous-System/internal/model"
)

type Poller struct {
	client    client.Client
	baseMS    int
	paused    atomic.Bool
	forceCh   chan struct{}
	stopCh    chan struct{}
	stopOnce  sync.Once

	ChainCh    chan []*model.Receipt
	SnapshotCh chan []*model.Snapshot
	PolicyCh   chan []*model.Policy
	TokenCh    chan []*model.Token
	MCPCh      chan *MCPState
	DaemonCh   chan *model.DaemonStatus
	ErrorCh    chan error
}

type MCPState struct {
	Status *model.MCPStatus
	Log    []*model.MCPLogEntry
}

func New(c client.Client, baseRefreshMS int) *Poller {
	if baseRefreshMS < 100 {
		baseRefreshMS = 100
	}
	return &Poller{
		client:    c,
		baseMS:    baseRefreshMS,
		forceCh:   make(chan struct{}, 1),
		stopCh:    make(chan struct{}),
		ChainCh:   make(chan []*model.Receipt, 4),
		SnapshotCh: make(chan []*model.Snapshot, 4),
		PolicyCh:  make(chan []*model.Policy, 4),
		TokenCh:   make(chan []*model.Token, 4),
		MCPCh:     make(chan *MCPState, 4),
		DaemonCh:  make(chan *model.DaemonStatus, 4),
		ErrorCh:   make(chan error, 8),
	}
}

func (p *Poller) Start() {
	go p.pollDaemon(1)
	go p.pollChain(2)
	go p.pollSnapshots(5)
	go p.pollPolicies(5)
	go p.pollTokens(1)
	go p.pollMCP(2)
}

func (p *Poller) Stop() {
	p.stopOnce.Do(func() { close(p.stopCh) })
}

func (p *Poller) Pause() {
	p.paused.Store(true)
}

func (p *Poller) Resume() {
	p.paused.Store(false)
	p.ForceRefresh()
}

func (p *Poller) ForceRefresh() {
	select {
	case p.forceCh <- struct{}{}:
	default:
	}
}

func (p *Poller) shouldRun() bool {
	return !p.paused.Load()
}

func (p *Poller) sleep(baseInterval int, ch chan struct{}) bool {
	d := time.Duration(baseInterval) * time.Duration(p.baseMS) * time.Millisecond
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-p.forceCh:
		return true
	case <-p.stopCh:
		return false
	}
}

func sendOrDrop[T any](ch chan T, val T) {
	select {
	case ch <- val:
	default:
	}
}

func (p *Poller) pollDaemon(intervalMS int) {
	_ = intervalMS
	for {
		if !p.sleep(1000, nil) {
			return
		}
		if !p.shouldRun() {
			continue
		}
		if s, err := p.client.DaemonStatus(); err == nil {
			sendOrDrop(p.DaemonCh, s)
		}
	}
}

func (p *Poller) pollChain(intervalMS int) {
	for {
		if !p.sleep(intervalMS, nil) {
			return
		}
		if !p.shouldRun() {
			continue
		}
		if r, err := p.client.ListReceipts(50, ""); err == nil {
			sendOrDrop(p.ChainCh, r)
		}
	}
}

func (p *Poller) pollSnapshots(intervalMS int) {
	for {
		if !p.sleep(intervalMS, nil) {
			return
		}
		if !p.shouldRun() {
			continue
		}
		if s, err := p.client.ListSnapshots("", 50); err == nil {
			sendOrDrop(p.SnapshotCh, s)
		}
	}
}

func (p *Poller) pollPolicies(intervalMS int) {
	for {
		if !p.sleep(intervalMS, nil) {
			return
		}
		if !p.shouldRun() {
			continue
		}
		if pl, err := p.client.ListPolicies(); err == nil {
			sendOrDrop(p.PolicyCh, pl)
		}
	}
}

func (p *Poller) pollTokens(intervalMS int) {
	for {
		if !p.sleep(intervalMS, nil) {
			return
		}
		if !p.shouldRun() {
			continue
		}
		if t, err := p.client.ListTokens(); err == nil {
			sendOrDrop(p.TokenCh, t)
		}
	}
}

func (p *Poller) pollMCP(intervalMS int) {
	for {
		if !p.sleep(intervalMS, nil) {
			return
		}
		if !p.shouldRun() {
			continue
		}
		status, err := p.client.MCPStatus()
		if err != nil {
			continue
		}
		log, err := p.client.MCPLog(20)
		if err != nil {
			continue
		}
		sendOrDrop(p.MCPCh, &MCPState{Status: status, Log: log})
	}
}
