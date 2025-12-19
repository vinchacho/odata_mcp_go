# LLM Compatibility Documentation Design

**Status**: Approved
**Author**: Claude + Vincent
**Date**: 2025-12-19

---

## Overview

Create comprehensive documentation for using the OData MCP Bridge with multiple LLM platforms and IDE integrations.

---

## Scope

**Current Scope (Option C):**
- Claude Desktop (reference)
- IDE tools: Cline, Roo Code, Cursor, Windsurf
- Chat platforms: ChatGPT, GitHub Copilot

**Future Vision (Option D - Out of Scope):**
- Remote deployment (Docker, cloud hosting)
- Reverse proxy configurations
- Authentication for exposed endpoints
- Multi-tenant scenarios

---

## Document Structure

```
docs/
  LLM_COMPATIBILITY.md          # Master overview (300-400 words)
  IDE_INTEGRATION.md            # Stdio-based platforms (~800 words)
  CHAT_PLATFORM_INTEGRATION.md  # HTTP-based + Copilot (~600 words)
```

### LLM_COMPATIBILITY.md

Quick reference landing page:
- Compatibility matrix (platform × transport × status)
- "Which doc do I need?" decision tree
- Links to detailed guides
- Future vision callout (remote deployment)

### IDE_INTEGRATION.md

Stdio-based platforms:
- Claude Desktop (reference, brief - already documented elsewhere)
- Cline (full guide)
- Roo Code (full guide)
- Cursor (full guide)
- Windsurf (full guide)
- Common troubleshooting section (shared across all)

### CHAT_PLATFORM_INTEGRATION.md

HTTP transport required:
- GitHub Copilot (stdio via agent mode)
- ChatGPT (HTTP transport + Custom GPT setup)
- Transport configuration section
- Security considerations for exposed endpoints

---

## Platform Classification

| Platform | Transport | Status | Notes |
|----------|-----------|--------|-------|
| Claude Desktop | stdio | ✅ Stable | Reference implementation |
| Cline | stdio | ✅ Stable | Well-documented, widely used |
| Roo Code | stdio | ✅ Stable | Similar to Cline |
| Cursor | stdio | ✅ Stable | Native MCP support |
| Windsurf | stdio | ✅ Stable | Native MCP support |
| GitHub Copilot | stdio | ✅ Stable | MCP via agent mode |
| ChatGPT | http | ✅ Stable | Requires HTTP endpoint + Custom GPT |

**Key distinction:** Transport requirement (stdio vs HTTP), not maturity level.

---

## Per-Platform Content Template

Each platform section follows this structure:

```markdown
## [Platform Name]

**Transport:** stdio | http | streamable-http
**Status:** ✅ Stable

### Prerequisites
- Platform version requirements
- OData MCP Bridge binary location

### Configuration
[Copy-paste ready config snippet]

### Common Use Cases
1. [Use case with example prompt]
2. [Use case with example prompt]

### Platform-Specific Notes
- Any quirks or limitations
- Recommended flags for this platform

### Troubleshooting
| Problem | Solution |
|---------|----------|
| [Issue] | [Fix] |
```

---

## Platform-Specific Additions

| Platform | Special Sections |
|----------|------------------|
| Claude Desktop | Reference to existing README config |
| Cline | `.cline/mcp.json` location, workspace vs global |
| Roo Code | Similar to Cline, note any differences |
| Cursor | `~/.cursor/mcp.json` or settings UI |
| Windsurf | Config location, any version caveats |
| GitHub Copilot | Agent mode setup, `@mcp` usage |
| ChatGPT | Custom GPT creation, HTTP endpoint exposure, security warnings |

---

## Content Depth

**Practical guide level (Option B):**
- Prerequisites
- Configuration (copy-paste ready)
- 2-3 common use cases with examples
- Platform-specific troubleshooting (3-5 items)

Not comprehensive (no architecture diagrams or edge cases) - can add depth later based on feedback.

---

## Acceptance Criteria

| Document | Criteria |
|----------|----------|
| LLM_COMPATIBILITY.md | Compatibility matrix accurate, links work, decision tree clear |
| IDE_INTEGRATION.md | All 5 platforms have working config snippets, troubleshooting covers top 3 issues each |
| CHAT_PLATFORM_INTEGRATION.md | HTTP transport setup clear, ChatGPT Custom GPT steps complete, security warnings present |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Platform config locations change | Medium | Low | Note version tested, link to official docs |
| ChatGPT Custom GPT process changes | Medium | Medium | Focus on HTTP transport (our side), link to OpenAI docs for GPT creation |
| Missing platform quirks | Medium | Low | Start with known issues, invite community contributions |

---

## Files to Create

1. `docs/LLM_COMPATIBILITY.md` - Master overview
2. `docs/IDE_INTEGRATION.md` - Cline, Roo Code, Cursor, Windsurf, Claude Desktop
3. `docs/CHAT_PLATFORM_INTEGRATION.md` - ChatGPT, GitHub Copilot

## Files to Update

1. `CLAUDE.md` - Add new docs to documentation index
