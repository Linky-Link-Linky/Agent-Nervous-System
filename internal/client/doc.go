// Package client implements the ANS daemon protocol client.
// SocketClient communicates with the daemon over Unix domain sockets (or
// Windows named pipes) using a length-prefixed JSON wire format. The Client
// interface defines all 72 RPC methods for daemon interaction.
package client
