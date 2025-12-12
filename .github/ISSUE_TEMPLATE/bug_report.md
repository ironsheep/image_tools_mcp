---
name: Bug Report
about: Report a defect or unexpected behavior
title: '[BUG] '
labels: bug
assignees: ''
---

## Description

A clear description of what the bug is.

## Environment

- **Platform**: (e.g., Linux AMD64, macOS ARM64, Windows AMD64)
- **Binary version**: (output of `image-tools-mcp --version`)
- **MCP Client**: (e.g., Claude Desktop, Claude Code CLI)
- **Tesseract version** (if OCR-related): (output of `tesseract --version`)

## Steps to Reproduce

1.
2.
3.

## Expected Behavior

What you expected to happen.

## Actual Behavior

What actually happened.

## Tool Request/Response

If applicable, include the MCP request and response:

**Request:**
```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"tool_name","arguments":{...}}}
```

**Response:**
```json
{...}
```

## Additional Context

- Sample image (if applicable and shareable)
- Error messages or stack traces
- Any other relevant information
