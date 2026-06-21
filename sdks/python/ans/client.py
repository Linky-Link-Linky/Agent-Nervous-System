"""ANS client — communicates with the ANS daemon over Unix socket or named pipe."""

import hashlib
import json
import os
import platform
import socket
import struct
import threading
from typing import Any, Dict, List, Optional

# Protocol constants (must match daemon/protocol.go)
MSG_SIGN_APPEND = 0x01
MSG_SIGN_APPEND_RESP = 0x02
MSG_VERIFY = 0x03
MSG_VERIFY_RESP = 0x04
MSG_QUERY = 0x05
MSG_QUERY_RESP = 0x06
MSG_REGISTER = 0x07
MSG_REGISTER_RESP = 0x08
MSG_STATUS = 0x09
MSG_STATUS_RESP = 0x0A
MSG_PING = 0x0B
MSG_PONG = 0x0C
MSG_SNAPSHOT = 0x0D
MSG_SNAPSHOT_RESP = 0x0E
MSG_RESTORE = 0x0F
MSG_RESTORE_RESP = 0x10
MSG_SNAPSHOT_LIST = 0x11
MSG_SNAPSHOT_LIST_RESP = 0x12
MSG_REGISTER_COMPENSATE = 0x13
MSG_REGISTER_COMPENSATE_RESP = 0x14
MSG_COMPENSATE = 0x15
MSG_COMPENSATE_RESP = 0x16
MSG_TOKEN_REQUEST = 0x21
MSG_TOKEN_RESP = 0x22
MSG_TOKEN_REVOKE = 0x23
MSG_TOKEN_REVOKE_RESP = 0x24
MSG_TOKEN_LIST = 0x25
MSG_TOKEN_LIST_RESP = 0x26
MSG_ERROR = 0xFF

MAX_FRAME_SIZE = 4 * 1024 * 1024  # 4 MB


class ANSError(Exception):
    """Base exception for ANS client errors."""
    pass


class ANSClient:
    """
    Client for the ANS daemon. Communicates over Unix socket (Linux/macOS)
    or named pipe (Windows).
    
    Example:
        client = ANSClient()
        resp = client.sign_append(
            agent_id="ans_3vQb7uL6x9",
            phase="pre",
            action_type="file.write",
            payload_hash="abc123...",
            payload_summary="Write config.json"
        )
        print(f"Receipt ID: {resp['receipt_id']}")
    """

    def __init__(self, agent_id: Optional[str] = None, socket_path: Optional[str] = None):
        """
        Initialize the ANS client.
        
        Args:
            agent_id: Default agent ID for all operations (also sets global default)
            socket_path: Path to the daemon socket. If None, uses platform default:
                - Linux/macOS: /tmp/ans.sock
                - Windows: //./pipe/ans
        """
        if socket_path is None:
            socket_path = os.environ.get("ANS_SOCKET_PATH")
        if socket_path is None:
            if platform.system() == "Windows":
                socket_path = r"\\.\pipe\ans"
            else:
                socket_path = "/tmp/ans.sock"
        self.socket_path = socket_path
        self._sock: Optional[socket.socket] = None
        self._lock = threading.RLock()
        if agent_id is not None:
            configure(agent_id=agent_id, socket_path=socket_path)

    def connect(self) -> None:
        """Connect to the ANS daemon."""
        with self._lock:
            if self._sock is not None:
                return

            if platform.system() == "Windows":
                # Named pipe on Windows
                try:
                    import pywintypes  # noqa: F401
                    import win32file
                except ImportError:
                    raise ANSError(
                        "pywin32 is required on Windows. Install with: pip install pywin32"
                    )

                try:
                    handle = win32file.CreateFile(
                        self.socket_path,
                        win32file.GENERIC_READ | win32file.GENERIC_WRITE,
                        0,
                        None,
                        win32file.OPEN_EXISTING,
                        0,
                        None,
                    )
                    # Wrap in a socket-like interface for uniform I/O
                    self._sock = _WindowsPipeSocket(handle)
                except pywintypes.error as e:
                    raise ANSError(f"Failed to connect to ANS daemon at {self.socket_path}: {e}")
            else:
                # Unix domain socket on Linux/macOS
                try:
                    sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
                    sock.settimeout(30)
                    sock.connect(self.socket_path)
                    self._sock = sock
                except (FileNotFoundError, ConnectionRefusedError, OSError) as e:
                    raise ANSError(
                        f"Failed to connect to ANS daemon at {self.socket_path}. "
                        f"Is the daemon running? (run 'ans start'): {e}"
                    )

    def close(self) -> None:
        """Close the connection to the daemon."""
        with self._lock:
            if self._sock is not None:
                self._sock.close()
                self._sock = None

    def __enter__(self):
        self.connect()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()

    def _write_frame(self, msg_type: int, body: bytes) -> None:
        """Write [4-byte size][1-byte type][body] to socket."""
        payload_len = 1 + len(body)
        if payload_len > MAX_FRAME_SIZE:
            raise ANSError(f"Frame too large: {payload_len} (max {MAX_FRAME_SIZE})")
        header = struct.pack(">I", payload_len)
        if self._sock is None:
            raise ANSError("Not connected")
        self._sock.sendall(header + bytes([msg_type]) + body)

    def _read_frame(self) -> tuple[int, bytes]:
        """Read one frame from socket. Returns (msg_type, body)."""
        header = self._recv_exactly(4)
        size = struct.unpack(">I", header)[0]
        if size == 0 or size > MAX_FRAME_SIZE:
            raise ANSError(f"Invalid frame size: {size}")
        payload = self._recv_exactly(size)
        return payload[0], payload[1:]

    def _recv_exactly(self, n: int) -> bytes:
        """Read exactly n bytes from socket."""
        if self._sock is None:
            raise ANSError("Not connected")
        buf = b""
        while len(buf) < n:
            chunk = self._sock.recv(n - len(buf))
            if not chunk:
                raise ANSError("Connection closed by daemon")
            buf += chunk
        return buf

    def _call(self, req_type: int, req_body: Dict[str, Any], resp_type: int) -> Dict[str, Any]:
        """Send request, read response. Raises ANSError on error frame."""
        with self._lock:
            self.connect()
            body = json.dumps(req_body).encode("utf-8")
            self._write_frame(req_type, body)

            msg_type, resp_body = self._read_frame()
            if msg_type == MSG_ERROR:
                try:
                    err = json.loads(resp_body)
                except json.JSONDecodeError:
                    raise ANSError("Daemon error (malformed response)")
                raise ANSError(f"Daemon error: {err.get('message', 'unknown')}")
            if msg_type != resp_type:
                raise ANSError(f"Unexpected response type: {msg_type} (expected {resp_type})")

            try:
                return json.loads(resp_body)
            except json.JSONDecodeError as e:
                raise ANSError(f"Invalid JSON response: {e}")

    def take_snapshot(
        self,
        agent_id: str,
        snap_type: str = "filesystem",
        paths: str = "",
    ) -> Dict[str, Any]:
        """
        Take a snapshot of the agent's workspace.
        
        Args:
            agent_id: Agent identifier
            snap_type: Snapshot type ("filesystem", "memory", "database")
            paths: Comma-separated paths to snapshot (empty = full workspace)
        
        Returns:
            Dict with keys: snapshot_id, chain_index, snap_type, size_bytes, hash
        """
        req = {"agent_id": agent_id, "snap_type": snap_type}
        if paths:
            req["paths"] = paths
        return self._call(MSG_SNAPSHOT, req, MSG_SNAPSHOT_RESP)

    def register_compensation(
        self,
        agent_id: str,
        receipt_id: str,
        action_type: str = "",
        reverse_action: str = "",
        reverse_cmd: str = "",
    ) -> bool:
        """
        Register a compensating action for a receipt.
        
        Args:
            agent_id: Agent identifier
            receipt_id: Receipt ID to compensate
            action_type: Action type
            reverse_action: Human-readable description of the reverse action
            reverse_cmd: Shell command or URL to execute for compensation
        
        Returns:
            True if registered successfully
        """
        req = {
            "agent_id": agent_id,
            "receipt_id": receipt_id,
            "action_type": action_type,
            "reverse_action": reverse_action,
            "reverse_cmd": reverse_cmd,
        }
        resp = self._call(MSG_REGISTER_COMPENSATE, req, MSG_REGISTER_COMPENSATE_RESP)
        return resp.get("success", False)

    def sign_append(
        self,
        agent_id: str,
        phase: str,
        action_type: str,
        payload_hash: str,
        payload_summary: str = "",
        policy_decision: str = "",
        auth_context: str = "",
        outcome: str = "",
        outcome_summary: str = "",
        duration_ms: int = 0,
        pre_receipt_id: str = "",
        parent_agent_id: str = "",
        auto_snapshot: bool = False,
    ) -> Dict[str, Any]:
        """
        Sign and append a receipt to the chain.
        
        Args:
            agent_id: Agent identifier (e.g., "ans_3vQb7uL6x9")
            phase: "pre" or "post"
            action_type: Action type (e.g., "file.write", "http.get")
            payload_hash: SHA-256 hex of the action payload
            payload_summary: Human-readable summary (max ~80 chars)
            policy_decision: "allow", "deny", or "allow_with_conditions" (pre only)
            auth_context: Authorizing context (pre only)
            outcome: "success", "failure", or "partial" (post only)
            outcome_summary: Summary of outcome (post only)
            duration_ms: Duration in milliseconds (post only)
            pre_receipt_id: Linked pre-action receipt ID (post only)
            parent_agent_id: Parent agent ID if this is a sub-agent
        
        Returns:
            Dict with keys: receipt_id, chain_index, chain_tip, signature
        """
        if auto_snapshot:
            snap_resp = self.take_snapshot(agent_id=agent_id, snap_type="filesystem")
            # The receipt will automatically get the snapshot_id from the daemon
            # via the MsgSnapshot flow; we just trigger it before sign_append
            _ = snap_resp  # snapshot_id will be embedded by daemon

        req = {
            "agent_id": agent_id,
            "phase": phase,
            "action_type": action_type,
            "payload_hash": payload_hash,
            "payload_summary": payload_summary,
            "policy_decision": policy_decision,
            "auth_context": auth_context,
            "outcome": outcome,
            "outcome_summary": outcome_summary,
            "duration_ms": duration_ms,
            "pre_receipt_id": pre_receipt_id,
            "parent_agent_id": parent_agent_id,
        }
        return self._call(MSG_SIGN_APPEND, req, MSG_SIGN_APPEND_RESP)

    def trace(self, action_type: str, auto_snapshot: bool = False, **kwargs):
        """
        Context manager for automatic pre/post tracing.
        
        Example:
            with client.trace("file.write"):
                deploy_config(path, content)
        
        Args:
            action_type: The action type being traced
            auto_snapshot: Whether to take a snapshot before the action
            **kwargs: Additional arguments passed to sign_append (agent_id, etc.)
        """
        return _TraceContext(self, action_type, auto_snapshot=auto_snapshot, **kwargs)

    def verify(self, receipt_id: str) -> Dict[str, Any]:
        """
        Verify a single receipt.
        
        Args:
            receipt_id: Receipt ID to verify
        
        Returns:
            Dict with keys: valid, receipt_id, agent_id, action_type, phase, etc.
        """
        req = {"receipt_id": receipt_id}
        return self._call(MSG_VERIFY, req, MSG_VERIFY_RESP)

    def query(
        self,
        agent_id: str = "",
        action_type: str = "",
        phase: str = "",
        since_ns: int = 0,
        limit: int = 100,
        offset: int = 0,
    ) -> List[Dict[str, Any]]:
        """
        Query receipts with filters.
        
        Args:
            agent_id: Filter by agent ID
            action_type: Filter by action type
            phase: Filter by phase ("pre" or "post")
            since_ns: Filter by timestamp (nanoseconds since epoch)
            limit: Max number of results
            offset: Pagination offset
        
        Returns:
            List of receipt dicts
        """
        req = {
            "agent_id": agent_id,
            "action_type": action_type,
            "phase": phase,
            "since_ns": since_ns,
            "limit": limit,
            "offset": offset,
        }
        resp = self._call(MSG_QUERY, req, MSG_QUERY_RESP)
        return resp.get("receipts", [])

    def register(
        self,
        name: str,
        version: str,
        owner: str = "",
        public_key_hex: str = "",
        metadata: Optional[Dict[str, str]] = None,
    ) -> str:
        """
        Register a new agent and get its ID.
        
        Args:
            name: Agent name
            version: Agent version
            owner: Owner/creator of the agent
            public_key_hex: Hex-encoded Ed25519 public key (optional, daemon generates if empty)
            metadata: Additional metadata
        
        Returns:
            Agent ID (e.g., "ans_3vQb7uL6x9")
        """
        req = {
            "name": name,
            "version": version,
            "owner": owner,
            "public_key_hex": public_key_hex,
            "metadata": metadata or {},
        }
        resp = self._call(MSG_REGISTER, req, MSG_REGISTER_RESP)
        return resp["agent_id"]

    def status(self) -> Dict[str, Any]:
        """
        Get daemon status.
        
        Returns:
            Dict with keys: uptime, chain_length, total_receipts, total_agents, db_size_bytes, etc.
        """
        return self._call(MSG_STATUS, {}, MSG_STATUS_RESP)

    def token_request(
        self,
        agent_id: str,
        resource: str,
        action: str = "read",
        ttl_seconds: int = 60,
        single_use: bool = True,
    ) -> Dict[str, Any]:
        """
        Request an ephemeral token from the identity broker.
        
        Args:
            agent_id: Agent identifier
            resource: Resource ARN or path
            action: Action (read, write, etc.)
            ttl_seconds: TTL in seconds (max 60)
            single_use: Whether the token is single-use
        """
        req = {
            "agent_id": agent_id,
            "resource": resource,
            "action": action,
            "ttl_seconds": ttl_seconds,
            "single_use": single_use,
        }
        return self._call(MSG_TOKEN_REQUEST, req, MSG_TOKEN_RESP)

    def token_revoke(self, token_id: str) -> bool:
        """
        Revoke a token immediately.
        
        Args:
            token_id: Token ID to revoke
        """
        req = {"token_id": token_id}
        resp = self._call(MSG_TOKEN_REVOKE, req, MSG_TOKEN_REVOKE_RESP)
        return resp.get("success", False)

    def token_list(self, agent_id: str = "") -> List[Dict[str, Any]]:
        """
        List active tokens.
        
        Args:
            agent_id: Filter by agent ID
        """
        req = {"agent_id": agent_id}
        resp = self._call(MSG_TOKEN_LIST, req, MSG_TOKEN_LIST_RESP)
        return resp.get("tokens", [])

    def ping(self) -> bool:
        """Ping the daemon. Returns True if daemon responds."""
        try:
            with self._lock:
                self.connect()
                self._write_frame(MSG_PING, b"")
                msg_type, _ = self._read_frame()
                return msg_type == MSG_PONG
        except Exception:
            return False


class _TraceContext:
    """Context manager for automatic pre/post tracing."""

    def __init__(self, client: ANSClient, action_type: str, auto_snapshot: bool = False, **kwargs):
        self.client = client
        self.action_type = action_type
        self.auto_snapshot = auto_snapshot
        if "agent_id" not in kwargs and _default_agent_id is not None:
            kwargs["agent_id"] = _default_agent_id
        self.kwargs = kwargs
        self.pre_receipt = None
        self._payload_hash = None

    def __enter__(self) -> Dict[str, Any]:
        """Send pre-action receipt."""
        ph = self.kwargs.pop("payload_hash", None) or hash_payload("")
        self._payload_hash = ph
        self.pre_receipt = self.client.sign_append(
            phase="pre",
            action_type=self.action_type,
            payload_hash=ph,
            auto_snapshot=self.auto_snapshot,
            **self.kwargs,
        )
        return self.pre_receipt

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Send post-action receipt."""
        outcome = "failure" if exc_type is not None else "success"
        ph = self._payload_hash or hash_payload(str(exc_val) if exc_val else "success")
        self.client.sign_append(
            phase="post",
            action_type=self.action_type,
            payload_hash=ph,
            pre_receipt_id=self.pre_receipt.get("receipt_id", ""),
            outcome=outcome,
            **self.kwargs,
        )


class _WindowsPipeSocket:
    """Wrapper for Windows named pipe that provides a socket-like interface."""

    def __init__(self, handle):
        self.handle = handle

    def sendall(self, data: bytes) -> None:
        import win32file

        win32file.WriteFile(self.handle, data)

    def recv(self, n: int) -> bytes:
        import win32file

        _, data = win32file.ReadFile(self.handle, n)
        return data

    def close(self) -> None:
        import win32file

        win32file.CloseHandle(self.handle)


# Global client instance for convenience
_global_client: Optional[ANSClient] = None
_default_agent_id: Optional[str] = None


def configure(agent_id: str, socket_path: Optional[str] = None) -> None:
    """
    Configure the global ANS client.
    
    Args:
        agent_id: Default agent ID for all operations
        socket_path: Path to daemon socket (uses platform default if None)
    """
    global _global_client, _default_agent_id
    _global_client = ANSClient(socket_path=socket_path)
    _default_agent_id = agent_id


def get_client() -> ANSClient:
    """Get the global ANS client. Raises if not configured."""
    if _global_client is None:
        raise ANSError("ANS client not configured. Call ans.configure() first.")
    return _global_client


def hash_payload(payload: Any) -> str:
    """
    Hash a payload to SHA-256 hex.
    
    Args:
        payload: Any JSON-serializable object
    
    Returns:
        SHA-256 hex digest
    """
    if isinstance(payload, str):
        data = payload.encode("utf-8")
    else:
        data = json.dumps(payload, sort_keys=True, default=str).encode("utf-8")
    return hashlib.sha256(data).hexdigest()
