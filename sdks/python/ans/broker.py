"""Identity Broker client for ephemeral credential provisioning."""

from typing import Dict, List, Optional
import time
from .client import ANSClient


class Scope:
    """Defines what a credential can access."""

    def __init__(
        self,
        resource: str,
        permissions: List[str],
        constraints: Optional[Dict[str, str]] = None,
    ):
        """
        Initialize a scope.
        
        Args:
            resource: Target resource (e.g., "s3://bucket/object", "vault://secret/data")
            permissions: Allowed operations (e.g., ["read"], ["write"], ["read", "write"])
            constraints: Additional restrictions (e.g., {"ip": "192.168.1.0/24"})
        """
        self.resource = resource
        self.permissions = permissions
        self.constraints = constraints or {}

    def to_dict(self) -> Dict:
        """Convert to dict for JSON serialization."""
        return {
            "resource": self.resource,
            "permissions": self.permissions,
            "constraints": self.constraints,
        }


class Credential:
    """An ephemeral, scoped credential."""

    def __init__(self, data: Dict):
        self.credential_id = data["credential_id"]
        self.agent_id = data["agent_id"]
        self.provider = data["provider"]
        self.type = data["type"]
        self.secret = data["secret"]
        self.metadata = data.get("metadata", {})
        self.scope = data["scope"]
        self.issued_at = data["issued_at"]
        self.expires_at = data["expires_at"]
        self.revoked = data.get("revoked", False)
        self.request_id = data.get("request_id", "")
        self.pre_receipt_id = data.get("pre_receipt_id", "")

    def __repr__(self) -> str:
        return (f"Credential(credential_id={self.credential_id!r}, "
                f"agent_id={self.agent_id!r}, provider={self.provider!r}, "
                f"type={self.type!r}, secret='[REDACTED]', "
                f"scope={self.scope!r}, issued_at={self.issued_at!r}, "
                f"expires_at={self.expires_at!r}, revoked={self.revoked})")

    def is_expired(self) -> bool:
        """Check if the credential has expired."""
        return time.time() > self.expires_at

    def is_active(self) -> bool:
        """Check if the credential is active (not expired and not revoked)."""
        return not self.is_expired() and not self.revoked


class IdentityBroker:
    """
    Client for the ANS Identity Broker.
    
    Provisions ephemeral, scoped credentials for zero-trust agent access.
    
    Example:
        from ans.broker import IdentityBroker, Scope
        
        broker = IdentityBroker(ans_client)
        
        # Provision a 60-second credential for S3 read access
        cred = broker.provision(
            provider="aws-iam",
            agent_id="ans_3vQb7uL6x9",
            action_type="file.read",
            scope=Scope(
                resource="s3://my-bucket/data.json",
                permissions=["read"]
            ),
            ttl_seconds=60,
            pre_receipt_id="abc123"
        )
        
        # Use the credential
        import boto3
        s3 = boto3.client(
            's3',
            aws_access_key_id=cred.secret,
            aws_secret_access_key=cred.metadata["secret_key"],
            aws_session_token=cred.metadata["session_token"]
        )
        
        # Revoke when done (or wait 60s for auto-expiry)
        broker.revoke(cred.credential_id)
    """

    def __init__(self, client: ANSClient):
        self.client = client

    def provision(
        self,
        provider: str,
        agent_id: str,
        action_type: str,
        scope: Scope,
        ttl_seconds: int = 60,
        pre_receipt_id: str = "",
        parent_agent_id: str = "",
    ) -> Credential:
        """
        Provision an ephemeral credential.
        
        Args:
            provider: Provider name ("vault", "aws-iam", "gcp-iam", etc.)
            agent_id: Agent requesting the credential
            action_type: Action type (e.g., "file.read", "http.post")
            scope: What the credential can access
            ttl_seconds: TTL in seconds (max 60)
            pre_receipt_id: Linked pre-action receipt ID
            parent_agent_id: Parent agent ID if sub-agent
        
        Returns:
            Credential object
        """
        ttl = min(ttl_seconds, 60)
        resp = self.client.token_request(
            agent_id=agent_id,
            resource=scope.resource,
            action=scope.permissions[0] if scope.permissions else "read",
            ttl_seconds=ttl,
            single_use=True,
        )
        data = {
            "credential_id": resp.get("token_id", ""),
            "agent_id": agent_id,
            "provider": provider,
            "type": resp.get("token_type", ""),
            "secret": resp.get("access_key", "") or resp.get("bearer_token", ""),
            "metadata": {
                "secret_key": resp.get("secret_key", ""),
                "session_token": resp.get("session_token", ""),
            },
            "scope": scope.to_dict(),
            "issued_at": 0,
            "expires_at": resp.get("expires_ns", 0) / 1e9 if resp.get("expires_ns") else 0,
        }
        return Credential(data)

    def revoke(self, credential_id: str) -> None:
        """
        Revoke a credential immediately.
        
        Args:
            credential_id: Credential ID to revoke
        """
        self.client.token_revoke(credential_id)

    def get(self, credential_id: str) -> Optional[Credential]:
        """
        Get a credential from the cache.
        
        Args:
            credential_id: Credential ID
        
        Returns:
            Credential object or None if not found
        """
        tokens = self.client.token_list()
        for t in tokens:
            if t.get("token_id") == credential_id:
                data = {
                    "credential_id": t["token_id"],
                    "agent_id": t.get("agent_id", ""),
                    "provider": t.get("provider", ""),
                    "type": t.get("token_type", ""),
                    "secret": "",
                    "metadata": {},
                    "scope": {"resource": t.get("resource", ""), "permissions": [t.get("action", "read")]},
                    "issued_at": t.get("created_ns", 0) / 1e9,
                    "expires_at": t.get("expires_ns", 0) / 1e9,
                    "revoked": t.get("state") != "active",
                }
                return Credential(data)
        return None

    def list_active(self) -> List[Credential]:
        """
        List all active (non-expired, non-revoked) credentials.
        
        Returns:
            List of Credential objects
        """
        tokens = self.client.token_list()
        result = []
        for t in tokens:
            if t.get("state") != "active":
                continue
            data = {
                "credential_id": t["token_id"],
                "agent_id": t.get("agent_id", ""),
                "provider": t.get("provider", ""),
                "type": t.get("token_type", ""),
                "secret": "",
                "metadata": {},
                "scope": {"resource": t.get("resource", ""), "permissions": [t.get("action", "read")]},
                "issued_at": t.get("created_ns", 0) / 1e9,
                "expires_at": t.get("expires_ns", 0) / 1e9,
            }
            result.append(Credential(data))
        return result


# Context manager for automatic credential provisioning and revocation
class ephemeral_credential:
    """
    Context manager for automatic credential lifecycle management.
    
    Example:
        from ans.broker import IdentityBroker, Scope, ephemeral_credential
        
        broker = IdentityBroker(ans_client)
        
        with ephemeral_credential(
            broker,
            provider="aws-iam",
            agent_id="ans_xyz",
            action_type="file.read",
            scope=Scope("s3://bucket/file", ["read"])
        ) as cred:
            # Use credential for 60 seconds
            s3_client = boto3.client('s3', aws_access_key_id=cred.secret, ...)
            data = s3_client.get_object(Bucket='bucket', Key='file')
        # Credential is automatically revoked here
    """

    def __init__(
        self,
        broker: IdentityBroker,
        provider: str,
        agent_id: str,
        action_type: str,
        scope: Scope,
        ttl_seconds: int = 60,
        pre_receipt_id: str = "",
    ):
        self.broker = broker
        self.provider = provider
        self.agent_id = agent_id
        self.action_type = action_type
        self.scope = scope
        self.ttl_seconds = ttl_seconds
        self.pre_receipt_id = pre_receipt_id
        self.credential = None

    def __enter__(self) -> Credential:
        """Provision the credential."""
        self.credential = self.broker.provision(
            provider=self.provider,
            agent_id=self.agent_id,
            action_type=self.action_type,
            scope=self.scope,
            ttl_seconds=self.ttl_seconds,
            pre_receipt_id=self.pre_receipt_id,
        )
        return self.credential

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Revoke the credential."""
        if self.credential:
            try:
                self.broker.revoke(self.credential.credential_id)
            except Exception:
                pass  # Best-effort revocation
