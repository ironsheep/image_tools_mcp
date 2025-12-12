---
name: Feature Request
about: Suggest a new tool or enhancement
title: '[FEATURE] '
labels: enhancement
assignees: ''
---

## Summary

A brief description of the feature or enhancement.

## Use Case

Describe the problem you're trying to solve or the workflow this would improve.

## Proposed Solution

Describe how you envision this feature working.

### Tool Interface (if proposing a new tool)

```json
{
  "name": "image_new_tool",
  "description": "What the tool does",
  "inputSchema": {
    "type": "object",
    "properties": {
      "path": {"type": "string", "description": "Path to image file"},
      "param1": {"type": "integer", "description": "Description"}
    },
    "required": ["path"]
  }
}
```

### Expected Output

```json
{
  "field1": "value",
  "field2": 123
}
```

## Alternatives Considered

Any alternative solutions or workarounds you've considered.

## Additional Context

- Related tools or features in other software
- Links to relevant documentation or examples
- Any other context
