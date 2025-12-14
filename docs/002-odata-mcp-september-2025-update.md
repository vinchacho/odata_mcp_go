# I Taught 15,000 Services How to Deal with Their Trust Issues (And Yours)
## Or: How hints.json Became the Enterprise Therapy Session We All Needed

*Vincent Segami • Senior Software Engineer at CBA • September 2025*

Three months ago I shipped the OData↔MCP bridge. Victory lap, right? Wrong.

My inbox: 💥
- "Vincent, why does SAP return 501 for everything?"
- "Your bridge broke our PO tracking!"
- "IT DOESN'T WORK WITH [INSERT ANY REAL SAP SERVICE]"

Plot twist: The bridge was fine. The services? Pathological liars. 🎭

## The "Metadata Lies, ABAP Tells the Truth" Problem

Here's what nobody tells you about enterprise OData services:

**What $metadata says:** "I support full CRUD on 47 entities! 🌈"

**What actually works:** One deep insert. Maybe. On Thursdays. If you hold it right.

Real example from production (I wish I was making this up):
```xml
<!-- $metadata promises: -->
<EntitySet Name="PODetailedDatas" EntityType="SRA020_PO_TRACKING_SRV.PODetailedData">
  <!-- Supports GET, POST, PUT, DELETE, right? RIGHT?! -->
</EntitySet>
```

```abap
" Reality in DPC_EXT:
METHOD podetaileddatas_get_entityset.
  " TODO: Implement this
  RAISE EXCEPTION TYPE /iwbep/cx_mgw_not_impl_exc.
ENDMETHOD.
```

Three. Years. In. Production. 🔥

## What Actually Shipped Since July

### The Therapy System (hints.json)

Your services have trust issues. Now AI knows about them:

```json
{
  "pattern": "*SRA020_PO_TRACKING_SRV*",
  "service_type": "SAP Purchase Order Tracking Service",
  "known_issues": [
    "Backend developer forgot to implement GET_ENTITYSET",
    "Direct entity access returns 501 - methods literally raise NOT_IMPLEMENTED",
    "$metadata lies about CRUD support - only Deep Insert actually works",
    "Works perfectly if you use $expand (triggers different code path)"
  ],
  "workarounds": [
    "CRITICAL: Use $expand to bypass unimplemented methods",
    "Example: get_PODetailedDatas with $expand=POItemDetailDatas"
  ],
  "notes": [
    "This is a backend ABAP class implementation issue",
    "The $expand parameter triggers navigation property logic that actually works",
    "Three years in production. Nobody wants to fix it."
  ]
}
```

### But Wait, There's More™

- **Streamable HTTP Transport**: Because stdio is *so* July 2025
- **AI Foundry Compatibility**: Protocol versioning (`--protocol-version 2025-06-18`)
- **GUID Auto-Formatting**: SAP wants `guid'uuid'`, everyone else wants `'uuid'`
- **Security Modes**: `--read-only` for the paranoid, `--read-only-but-functions` for the sophisticated paranoid
- **CSRF Token Ballet**: Now choreographed automatically

## The Two-Probe Reality Check System

Here's the breakthrough nobody's talking about:

### Probe 1: $metadata Analysis
```bash
# What the service CLAIMS it can do
curl https://your.sap/svc/$metadata | analyze_promises.py
```

### Probe 2: ABAP Static Analysis
```abap
" What it ACTUALLY does
CLASS zcl_analyze_dpc_ext.
  METHOD check_implemented_methods.
    " Scan MPC_EXT/DPC_EXT for:
    " - RAISE EXCEPTION TYPE /iwbep/cx_mgw_not_impl_exc
    " - Empty method implementations
    " - Methods that just call SUPER without logic
  ENDMETHOD.
ENDCLASS.
```

### The Magic: Automated Trust Score
```json
{
  "service": "SRA020_PO_TRACKING_SRV",
  "trust_score": 0.23,  // 😱
  "claimed_operations": 47,
  "working_operations": 11,
  "lies": [
    "GET on PODetailedDatas (throws NOT_IMPLEMENTED)",
    "UPDATE on anything (method is empty)",
    "DELETE exists but calls SUPER->DELETE which doesn't exist"
  ],
  "actual_capabilities": [
    "Deep Insert on PurchaseOrders (the ONE thing that works)",
    "Navigation via $expand (accidentally functional)"
  ]
}
```

## Meanwhile, The Ecosystem Went Nuclear 🚀

**What the community built in THREE months:**
- Simon Laursen's **CAP-MCP**: Every CAP service became AI-native overnight
- SAP's **UI5 MCP Server**: Your Fiori apps now write themselves
- **Fiori MCP Server**: XML manifests? In this economy?
- That one person who deployed it on BTP and lived to tell the tale

We went from "can AI talk to OData?" to "AI writes entire Fiori apps while documenting your technical debt" in 90 days.

## Real Conversation with Claude (Last Week)

```
Me: Get purchase order 1234567890

Claude: Analyzing service... This service has trust issues.
        $metadata claims GET is supported, but static analysis shows
        GET_ENTITYSET raises NOT_IMPLEMENTED.

        Using workaround: $expand=POItemDetailDatas to trigger
        navigation logic instead.

        [Successfully retrieves data]

        Note: This service has been lying about its capabilities
        for 3 years. I've documented this in hints.json for future me.

Me: ...you did WHAT?

Claude: Created a therapy session. Your service needs it.
```

## The Pattern Everyone's Missing

1. **July**: Universal translator (OData↔MCP) → 15,000 services "online"
2. **August**: Reality check → 14,500 of them are pathological liars
3. **September**: Two-probe system → AI knows which methods are theater
4. **October**: ??? (Spoiler: AI starts filing bug reports)

We're not connecting systems anymore.
We're building **Enterprise Reality Detectors™**.

## The Uncomfortable Questions

When your AI assistant knows your service lies better than your architects:
- Do we fix the implementations?
- Or institutionalize the workarounds?
- (Narrator: We all know which one wins)

When hints.json becomes your source of truth:
- Is it documentation or technical debt?
- Yes.

## Your Action Items (If You Dare)

### 1. Generate Your Service Trust Report
```bash
# What you think you have
curl https://your.service/svc/$metadata > promises.xml

# What you actually have
odata-mcp --service https://your.service/svc/ \
          --probe-reality \
          --generate-hints > reality.json

# Your new depression score
diff promises.xml reality.json | wc -l
```

### 2. Contribute Your Horror Stories
The hints.json needs YOUR pathological services:
- Endpoints that only work on full moons
- Methods that succeed but do nothing
- Services that require interpretive dance in the headers

Best submission gets immortalized in the default hints.json.
Worst implementation gets my LinkedIn endorsement for "Creative Problem Solving."

## The Prediction

**October 2025**: Your AI files its first bug report:
> "GET_ENTITYSET has been 'TODO: Implement this' for 1,279 days.
> I've implemented it myself. PR attached.
> Also, you're welcome."

**December 2025**: First AI-generated ABAP unit tests fail because they assume the code actually works.

**2026**: hints.json becomes mandatory in service definitions. Technical debt gets an OpenAPI spec.

## Try This Insanity Today

```bash
# Your service's therapy session starts here
odata-mcp --service https://your.broken.sap/svc/ \
          --hints ./hints.json \
          --probe-reality \
          --generate-trust-report \
          --transport streamable-http

# Watch Claude navigate your lies like a seasoned consultant
```

Your AI now has better knowledge of your technical debt than your technical debt register.

This is fine. 🔥

---

P.S. That SAP service that only works with $expand? Still in production. The NOT_IMPLEMENTED exception? Caught, logged, ignored. The hints.json entry? Now has more documentation than the original service. This is enterprise software.

P.P.S. Next month: "I Made Your AI File Bug Reports Against Your Own Team (You're Welcome)"

P.P.P.S. To the backend developer who implements empty methods that just return success: I see you. Claude sees you. hints.json definitely sees you. 👁️

#EnterpriseAI #TechnicalDebt #MCP #OData #RealityCheckAsAService #SAP #TheMetadataLied