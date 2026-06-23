# Contributing to Agent Nervous System

## Development Setup

### Prerequisites
- Go 1.21+
- Python 3.10+ (for Python SDK)
- Node.js 18+ (for TypeScript SDK)

### Building

```bash
cd ans
go build ./...
```

### Testing

```bash
cd ans
go test ./...
```

### Running the linter

```bash
cd ans
go vet ./...
staticcheck ./...
gosec -quiet ./...
```

## Pull Request Process

1. Fork the repo and create your branch from `master`.
2. Run tests and linters before submitting.
3. Include a clear description of the change.
4. PRs require at least one maintainer review.

## Contributor License Agreement

### Individual Contributor License Agreement

Thank you for your interest in contributing to Agent Nervous System ("Project") owned by Linky-Link-Linky ("Company").

This Individual Contributor License Agreement ("Agreement") sets out the terms under which You agree to license Your Contributions to the Company.

#### 1. Definitions

"You" (or "Your") means the individual agreeing to this Agreement.

"Contribution" means any original work of authorship, including any modifications or additions to an existing work, that You intentionally submit to the Project for inclusion.

"Submit" means any form of electronic communication sent to the Company or its representatives for the purpose of discussing and improving the Project.

#### 2. Grant of Copyright License

You grant the Company and recipients of the Project a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable copyright license to reproduce, prepare derivative works of, publicly display, publicly perform, sublicense, and distribute Your Contributions and derivative works thereof.

#### 3. Grant of Patent License

You grant the Company and recipients of the Project a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable patent license to make, have made, use, offer to sell, sell, import, and otherwise transfer the Project, where such license applies only to patent claims licensable by You that are necessarily infringed by Your Contribution(s). If any entity institutes patent litigation against You alleging Your Contribution infringes, patent licenses granted under this Agreement for that Contribution shall terminate as of the date such litigation is filed.

#### 4. Representations

You represent that:

(a) Each Contribution is Your original work and You have the right to submit it;

(b) You are legally entitled to grant the licenses in this Agreement;

(c) Your Contributions do not include third-party content unless You have obtained all necessary permissions to include and license it under this Agreement; and

(d) To Your knowledge, Your Contributions do not violate any third-party intellectual property rights.

#### 5. No Support

You are not obligated to provide support for Your Contributions. You provide them on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

#### 6. Notification

You agree to notify the Company if You become aware of any facts that would make the above representations inaccurate.

#### 7. Miscellaneous

(a) **Entire Agreement.** This Agreement constitutes the entire agreement between the parties regarding its subject matter and supersedes all prior agreements and understandings, whether written or oral.

(b) **Governing Law.** This Agreement shall be governed by and construed in accordance with the laws governing the Project's license, without regard to conflict of law principles.

(c) **Severability.** If any provision of this Agreement is found to be unenforceable or invalid, that provision shall be limited or eliminated to the minimum extent necessary so that this Agreement shall otherwise remain in full force and effect.

(d) **No Waiver.** The failure of either party to enforce any right or provision of this Agreement shall not constitute a waiver of future enforcement of that right or provision.

(e) **Assignment.** You may not assign or transfer this Agreement, or any of Your rights or obligations hereunder, by operation of law or otherwise, without the prior written consent of the Company. The Company may freely assign this Agreement.

#### 8. Signature

When you submit your first pull request, comment on the PR with the following exact text to indicate your acceptance:

**I have read the CLA Document and I hereby sign the CLA**

The CLA automation will record your signature on the `cla-signatures` branch of the repository and update the PR status. Signing is required once per GitHub account before contributions can be merged.
