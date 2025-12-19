# SolMan Hub Architecture - Documentation Report

**Date:** 2025-12-01
**Report ID:** 104
**Category:** Analysis & Research
**Status:** Complete

## Summary

The `docs/SOLMAN_HUB_ARCHITECTURE.md` file proposes using SAP Solution Manager as a central hub for AI/MCP access to the SAP landscape, rather than connecting directly to individual systems.

## Document Overview

| Attribute | Value |
|-----------|-------|
| Title | SAP Solution Manager as Central Hub Architecture |
| Created | December 2025 |
| Status | Future Exploration |
| Format | Architecture proposal |

## Architecture Concept

### Hub/Spoke Model

```
AI Agent (Claude)
      ↓ MCP Protocol
odata_mcp_go
      ↓ OData v2/v4 + Basic Auth
SAP SOLUTION MANAGER (HUB)
      ↓ RFC/HTTP (internal SAP)
   ┌──┴──┬──────┐
  ECC  S/4HANA  BW
(Spokes)
```

### Key Principle

Instead of multiple AI-to-SAP connections, use Solution Manager as a single entry point that provides access to all managed systems.

## Advantages

| Advantage | Description |
|-----------|-------------|
| Single Entry Point | One URL, one credential set, one firewall rule |
| Centralized Auth | Manage one service account instead of N systems |
| Landscape Visibility | SolMan already knows about all managed systems |
| Existing Infrastructure | No new servers - SolMan is already deployed |
| Security Boundary | AI never directly touches production ERP |
| Audit Trail | All access logged in one place |
| Network Simplicity | Only SolMan needs network access to AI |

## Challenges

| Challenge | Mitigation |
|-----------|------------|
| Single Point of Failure | SolMan typically has HA setup |
| Performance Bottleneck | SolMan is designed for this |
| OData Service Availability | May need custom OData services |
| Indirect Data Access | Use SolMan's existing integrations |
| Version Constraints | Check SolMan version capabilities |
| Licensing | Verify with SAP |

## Project Compatibility

The odata_mcp_go project already supports this use case:

| Feature | SolMan Relevance |
|---------|------------------|
| Basic Auth | SolMan supports this |
| CSRF Token Handling | Required for SAP systems |
| OData v2 + v4 | SolMan has both |
| SAP Hints System | Can add SolMan-specific hints |
| Read-Only Mode | Safe for production SolMan |
| Entity Filtering | Expose only relevant services |

## Configuration Examples

### Single Service Approach

```json
{
  "mcpServers": {
    "sap-solman": {
      "command": "/usr/local/bin/odata-mcp",
      "args": [
        "--service", "https://solman.company.com/sap/opu/odata/sap/SPECIFIC_SERVICE/",
        "--tool-shrink",
        "--read-only"
      ],
      "env": {
        "ODATA_USERNAME": "MCP_SERVICE_USER",
        "ODATA_PASSWORD": "secret"
      }
    }
  }
}
```

### Multiple Services Approach

```json
{
  "mcpServers": {
    "solman-lmdb": { ... },
    "solman-charm": { ... }
  }
}
```

## Potential Enhancements

| Enhancement | Value | Effort |
|-------------|-------|--------|
| Service Discovery Tool | Auto-list OData services on SolMan | Medium |
| Multi-Service Mode | Connect to base URL, expose all | Large |
| SolMan-Specific Hints | Pre-built guidance | Small |
| Landscape Context | Include managed system info | Medium |

## Open Questions

1. Which SolMan OData services should be exposed?
2. Is read-only access sufficient?
3. What SolMan version is deployed?
4. Should data from managed systems be accessed via SolMan?
5. Licensing implications for API access?

## Next Steps (Checklist)

- [ ] Inventory available OData services in Solution Manager
- [ ] Define use cases for AI access
- [ ] Test connectivity with odata_mcp_go
- [ ] Create SolMan-specific hints if needed
- [ ] Document security and access control requirements

## Use Case Scenarios

### Scenario 1: Landscape Overview

AI queries SolMan's LMDB (Landscape Management Database) to understand:
- What systems exist in the landscape
- System versions and components
- Change history

### Scenario 2: Change Management

AI queries SolMan's ChaRM (Change Request Management) to:
- List open change requests
- Check transport status
- Query approval workflows

### Scenario 3: Incident Management

AI queries SolMan's incident data to:
- Report open incidents
- Correlate incidents with recent changes
- Suggest known solutions

## Relevance to Project

This document serves as:

1. **Architecture exploration** - Hub/spoke model for enterprise deployments
2. **Future roadmap item** - Multi-service mode enhancement
3. **Enterprise selling point** - Addresses security and centralization concerns
4. **Implementation guide** - Configuration examples for SolMan

## Relationship to Other Docs

| Doc | Relationship |
|-----|-------------|
| COMPETITIVE_ANALYSIS.md | Multi-service support identified as v2.0 feature |
| 003-e2e-documentation.md | Network architecture section aligns |
| hints.json | Could include SolMan-specific patterns |

## File Reference

- Source: [docs/SOLMAN_HUB_ARCHITECTURE.md](docs/SOLMAN_HUB_ARCHITECTURE.md)
