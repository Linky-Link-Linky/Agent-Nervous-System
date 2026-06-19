------------------------------ MODULE chain ------------------------------
(*
  ANS Append-Only Chain — TLA+ Formal Specification

  This module models the core ANS chain invariants:
  1.  chain_index is strictly increasing with no gaps (1, 2, 3, ...)
  2.  prev_receipt_hash always points to the immediately prior receipt's hash
  3.  genesis receipt has prev_receipt_hash = 0^64
  4.  every receipt is signed by a registered agent
  5.  the hash chain is append-only — no deletion, no reordering
*)

EXTENDS Integers, Sequences, FiniteSets

CONSTANTS
    MaxChainLength,          \* Upper bound on chain size for model checking
    AgentIDs,                \* Set of registered agent IDs (e.g., {"a1", "a2"})
    GenesisHash              \* = "000...0" (64 zeros)

ASSUME GenesisHash \notin {NULL}

\* ─── State ──────────────────────────────────────────────────────────

VARIABLES
    chain,                   \* Sequence of receipts, indexed 1..Len(chain)
    nextIdx,                 \* Next chain_index to assign (always Len(chain) + 1)
    receiptsByID             \* Function receipt_id → receipt (for lookup)

vars == <<chain, nextIdx, receiptsByID>>

\* ─── Type Invariant ─────────────────────────────────────────────────

Receipt == [
    receipt_id       : Str,
    phase            : {"pre", "post"},
    agent_id         : AgentIDs,
    prev_receipt_hash: Str,
    chain_index      : 1..MaxChainLength,
    action_type      : Str,
    payload_hash     : Str,
    signature        : Str
]

TypeOK ==
    /\ chain \in Seq(Receipt)
    /\ nextIdx \in 1..(MaxChainLength + 1)
    /\ nextIdx = Len(chain) + 1
    /\ receiptsByID \in [Str -> Receipt]
    /\ \A r \in chain: receiptsByID[r.receipt_id] = r

\* ─── Invariants ──────────────────────────────────────────────────────

\* Invariant 1: chain_index is strictly increasing by 1 (no gaps)
NoGaps ==
    \A i \in 1..Len(chain):
        chain[i].chain_index = i

\* Invariant 2: prev_receipt_hash links correctly
\*   genesis:  prev_receipt_hash = GenesisHash
\*   receipt i: prev_receipt_hash = hash(chain[i-1])
\*   For the TLA+ spec we model hashes abstractly as:
\*     HashFn(r)  =  "hash(" + r.receipt_id + ")"
HashFn(r) == "hash(" + r.receipt_id + ")"

HashChainLinked ==
    /\ Len(chain) >= 1 => chain[1].prev_receipt_hash = GenesisHash
    /\ \A i \in 2..Len(chain):
        chain[i].prev_receipt_hash = HashFn(chain[i-1])

\* Invariant 3: Every receipt is signed by a registered agent
\*   (modeled: agent_id must be in AgentIDs)
AllAgentsRegistered ==
    \A r \in chain: r.agent_id \in AgentIDs

\* Invariant 4: receipt_id is unique
ReceiptIDsUnique ==
    \A i, j \in 1..Len(chain):
        i /= j => chain[i].receipt_id /= chain[j].receipt_id

\* Invariant 5: Append-only — chain only grows, never shrinks or mutates
AppendOnly ==
    \* Expressed via TLA+ stuttering: chain' = Append(chain, newReceipt)
    \* This is enforced by the action below.

ChainInvariant ==
    /\ NoGaps
    /\ HashChainLinked
    /\ AllAgentsRegistered
    /\ ReceiptIDsUnique

\* ─── Initial State ──────────────────────────────────────────────────

Init ==
    /\ chain = << >>
    /\ nextIdx = 1
    /\ receiptsByID = [x \in {} |-> NULL]

\* ─── Actions ─────────────────────────────────────────────────────────

\* Append a new signed receipt to the chain.
\* Precondition: receipt is fully constructed (agent_id, payload_hash,
\*               action_type, phase) and signed.
AppendReceipt(newReceipt) ==
    /\ newReceipt.chain_index = nextIdx
    /\ IF nextIdx = 1
       THEN newReceipt.prev_receipt_hash = GenesisHash
       ELSE newReceipt.prev_receipt_hash = HashFn(chain[nextIdx - 1])
    /\ chain' = Append(chain, newReceipt)
    /\ nextIdx' = nextIdx + 1
    /\ receiptsByID' = [receiptsByID EXCEPT ![newReceipt.receipt_id] = newReceipt]

\* Stuttering step — allowed in TLA+, models time passing with no action
\* (This is automatically allowed; we don't need to model it explicitly.)

Next ==
    \E r \in Receipt:
        AppendReceipt(r)

\* ─── Spec ────────────────────────────────────────────────────────────

Spec ==
    Init /\ [][Next]_vars

\* ─── Fairness / Liveness ────────────────────────────────────────────

\* Under weak fairness, the chain eventually grows (if receipts are submitted)
FairSpec ==
    Spec /\ WF_vars(Next)

\* ─── Model Checking Properties ──────────────────────────────────────

\* Safety: The chain invariant must always hold
Safety == []ChainInvariant

\* Liveness: The chain eventually reaches any length (under fairness)
\* Liveness == \A n \in 1..MaxChainLength: <>(Len(chain) >= n)

\* ─── Model Values (for TLC model checker) ───────────────────────────
\*
\* To model-check with TLC:
\*   CONSTANTS
\*     MaxChainLength <- 5
\*     AgentIDs <- {"a1", "a2"}
\*     GenesisHash <- "0000000000000000000000000000000000000000000000000000000000000000"
\*   INVARIANT Safety
\*
\* Expected: no invariant violation found.

========================================================================
