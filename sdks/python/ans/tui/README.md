# ANS TUI — Interactive Terminal UI

## Install

```bash
pip install "ans-sdk[tui]"
```

## Prerequisites

The ANS daemon must be running:

```bash
ans start
```

## Launch

```bash
python -m ans.tui
```

Or using the installed script:

```bash
ans-tui
```

## Screens

```
┌────────────────────────────────────────────────────────────────┐
│  ANS — Agent Nervous System              receipt chain        │
├────────────────────────────────────────────────────────────────┤
│  ● CONNECTED  │  uptime: 1h23m  │  chain: 42  │  receipts:   │
├────────────────────────────────────────────────────────────────┤
│  Index │ Receipt ID │ Phase │ Agent │ Action │ Outcome │ Time  │
│  ──────┼────────────┼───────┼───────┼────────┼─────────┼────── │
│  1     │ aabbccdd   │ pre   │ ag_1  │ file.w │ allow   │ ...   │
│  2     │ 11223344   │ post  │ ag_1  │ file.w │ success │ ...   │
│  ...                                                          │
├────────────────────────────────────────────────────────────────┤
│  [r] Refresh  [a] Agents  [v] Verify  [q] Quit               │
└────────────────────────────────────────────────────────────────┘
```

## Key Bindings

| Key     | Action           |
|---------|------------------|
| `r`     | Refresh          |
| `a`     | Agents screen    |
| `c`     | Chain screen     |
| `v`     | Verify screen    |
| `q`     | Quit             |
| `Ctrl+T`| Toggle theme     |
