# SAP Solution Manager as Central Hub Architecture

*Created: December 2025*
*Status: Future Exploration*

This document outlines a proposed architecture using SAP Solution Manager as a central hub for AI/MCP access to the SAP landscape.

---

## Concept Overview

Instead of connecting the OData MCP Bridge directly to multiple SAP systems, use Solution Manager as a single entry point (hub) that provides access to all managed systems (spokes).

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           HUB/SPOKE ARCHITECTURE                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│                         ┌─────────────────┐                             │
│                         │   AI Agent      │                             │
│                         │   (Claude)      │                             │
│                         └────────┬────────┘                             │
│                                  │                                       │
│                                  │ MCP Protocol                          │
│                                  ▼                                       │
│                         ┌─────────────────┐                             │
│                         │  odata_mcp_go   │                             │
│                         │  (This Project) │                             │
│                         └────────┬────────┘                             │
│                                  │                                       │
│                                  │ OData v2/v4 + Basic Auth              │
│                                  ▼                                       │
│  ┌───────────────────────────────────────────────────────────────────┐  │
│  │                    SAP SOLUTION MANAGER (HUB)                      │  │
│  │                                                                    │  │
│  │   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐            │  │
│  │   │  OData Svc 1 │  │  OData Svc 2 │  │  OData Svc N │            │  │
│  │   └──────────────┘  └──────────────┘  └──────────────┘            │  │
│  │                                                                    │  │
│  │   ┌────────────────────────────────────────────────────────────┐  │  │
│  │   │              Managed System Connections (RFC)               │  │  │
│  │   └────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────┘  │
│                                  │                                       │
│                                  │ RFC/HTTP (internal SAP)               │
│                    ┌─────────────┼─────────────┐                        │
│                    ▼             ▼             ▼                        │
│             ┌──────────┐  ┌──────────┐  ┌──────────┐                   │
│             │  ECC/ERP │  │ S/4HANA  │  │   BW     │                   │
│             │  (Spoke) │  │  (Spoke) │  │  (Spoke) │                   │
│             └──────────┘  └──────────┘  └──────────┘                   │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Advantages

| Advantage | Description |
|-----------|-------------|
| **Single Entry Point** | One URL, one credential set, one firewall rule |
| **Centralized Auth** | Manage one service account instead of N systems |
| **Landscape Visibility** | SolMan already knows about all managed systems |
| **Existing Infrastructure** | No new servers needed - SolMan is already deployed |
| **Security Boundary** | AI never directly touches production ERP systems |
| **Audit Trail** | All access logged in one place |
| **Network Simplicity** | Only SolMan needs to be network-accessible to AI |

---

## Challenges

| Challenge | Description | Mitigation |
|-----------|-------------|------------|
| **Single Point of Failure** | If SolMan is down, no access to anything | SolMan typically has HA setup |
| **Performance Bottleneck** | All queries route through one system | SolMan is designed for this |
| **OData Service Availability** | Not all data is exposed via OData in SolMan | May need custom OData services |
| **Indirect Data Access** | Data from spokes may need RFC calls | Use SolMan's existing integrations |
| **Version Constraints** | SolMan OData capabilities vary by version | Check your SolMan version |
| **Licensing** | May need specific SolMan licenses for API access | Verify with SAP |

---

## Current Project Compatibility

The odata_mcp_go project already supports this use case:

| Feature | Relevance to SolMan Hub |
|---------|------------------------|
| Basic Auth | ✅ SolMan supports this |
| CSRF Token Handling | ✅ Required for SAP systems |
| OData v2 + v4 | ✅ SolMan has both |
| SAP Hints System | ✅ Can add SolMan-specific hints |
| Read-Only Mode | ✅ Safe for production SolMan |
| Entity Filtering | ✅ Expose only relevant services |

---

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
    "solman-lmdb": {
      "command": "odata-mcp",
      "args": [
        "--service", "https://solman.company.com/sap/opu/odata/sap/LMDB_SERVICE/",
        "--read-only"
      ],
      "env": {
        "ODATA_USERNAME": "MCP_SERVICE_USER",
        "ODATA_PASSWORD": "secret"
      }
    },
    "solman-charm": {
      "command": "odata-mcp",
      "args": [
        "--service", "https://solman.company.com/sap/opu/odata/sap/CHARM_SERVICE/",
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

---

## Potential Enhancements

Future features that would improve the SolMan hub use case:

| Enhancement | Value | Effort |
|-------------|-------|--------|
| **Service Discovery Tool** | Auto-list available OData services on SolMan | Medium |
| **Multi-Service Mode** | Connect to base URL, expose all services | Large |
| **SolMan-Specific Hints** | Pre-built guidance for common SolMan services | Small |
| **Landscape Context** | Include managed system info in responses | Medium |

---

## Open Questions

1. Which SolMan OData services should be exposed?
2. Is read-only access sufficient, or is write access needed?
3. What SolMan version is deployed? (Affects available OData services)
4. Should data from managed systems be accessed via SolMan, or just SolMan's own data?
5. What are the licensing implications for API access?

---

## Next Steps

- [ ] Inventory available OData services in Solution Manager
- [ ] Define use cases for AI access (read-only queries, reporting, etc.)
- [ ] Test connectivity with odata_mcp_go
- [ ] Create SolMan-specific hints if needed
- [ ] Document security and access control requirements
