// Package poller implements a background polling loop that periodically
// fetches daemon state and pushes typed model values onto buffered channels
// for consumption by the TUI bridge. Supports start/stop/pause/resume and
// one-shot ForceRefresh for immediate data retrieval.
package poller
