export default {
  "experience": {
    "links": "Explore the service",
    "plans": {
      "title": "Choose a plan",
      "intro": "Choose a quota window for your usage pattern, then compare prices. All figures below come from products currently on sale.",
      "startTitle": "Connecting for the first time",
      "startText": "Check your client, model and connection method first. Start with a smaller tier that meets your needs, rather than assuming a fixed number of calls.",
      "dailyTitle": "Steady daily use",
      "dailyText": "Compare daily quotas alongside any weekly and monthly limits. Unused daily quota follows the plan rules; do not assume it rolls over.",
      "flexibleTitle": "Work comes in bursts",
      "flexibleText": "Consider a monthly-only quota pool for uneven usage. Plan validity, Key limits and service capacity still apply.",
      "find": "Find a plan for your estimated spend",
      "pattern": "Usage pattern",
      "steady": "Steady daily use",
      "bursty": "Bursts within a month",
      "dailySpend": "Estimated daily metered spend (USD)",
      "monthlySpend": "Estimated monthly metered spend (USD)",
      "currency": "Plan payment currency",
      "unlabelled": "Currency not specified",
      "result": "Consider {name}.",
      "noMatch": "No GPT plan meets this estimate. Adjust it or review the comparison below.",
      "method": "Matches GPT products within one currency, then compares price per service day. Daily estimates assume 7 days per week and 30 days per month. Metered spend includes applicable billing multipliers; check usage records. A recommendation is not a capacity guarantee.",
      "compare": "Compare all plans on sale",
      "columns": {
        "name": "Plan",
        "price": "Price",
        "duration": "Validity",
        "daily": "Daily USD quota",
        "weekly": "Weekly USD quota",
        "monthly": "Monthly USD quota",
        "rate": "Billing multiplier",
        "action": "Select"
      },
      "tableNote": "A dash means no limit is configured for that dimension, not an absence of other limits. Daily quota × 30 is not a freely pooled monthly allowance. All configured windows apply together. Check the model directory and connection guide for supported models.",
      "renewal": "Existing legacy plans retain purchased benefits until expiry. When the deferred renewal policy applies, a prepaid replacement waits for legacy benefits to end. Check the activation time shown when purchasing.",
      "rules": "Renewal and quota rules"
    },
    "help": {
      "title": "GPT / Codex help",
      "subtitle": "Find the next step for setup, troubleshooting and checking your charges.",
      "search": "Search help",
      "placeholder": "Try Codex, 401, 404, 429, cache or renewal",
      "category": "Documentation category",
      "categories": {
        "all": "All",
        "start": "Setup and protocols",
        "trouble": "Troubleshooting",
        "billing": "Billing and renewal"
      },
      "results": "{count} articles found",
      "noResults": "No matching articles. Try another keyword or category.",
      "permalink": "Link to this article",
      "sources": "References and scope",
      "sourceNote": "Configuration and caching references checked against official documentation on 2026-10-23. This site provides API access. Actual model access, features and charges depend on site configuration and usage records.",
      "codexReference": "Codex configuration reference",
      "cacheReference": "OpenAI caching guide",
      "actions": {
        "connect": "Open connection center",
        "configure": "Generate client configuration",
        "subscriptions": "View my subscriptions",
        "usage": "View usage and available error records",
        "estimate": "Open cost estimator",
        "orders": "View orders"
      }
    },
    "billing": {
      "title": "Billing guide and estimator",
      "subtitle": "Understand plan prices, metered API usage and actual charges using this site’s price cards.",
      "tokensTitle": "API usage",
      "tokensText": "Input, output and caching use separate prices. One client task can make many requests; reasoning and tool use may add usage.",
      "quotaTitle": "Plan quota",
      "quotaText": "Quota is a metered spending limit, not cash balance. Daily, weekly and monthly windows constrain usage separately, alongside Key limits and expiry.",
      "paymentTitle": "Payment amount",
      "paymentText": "A plan price does not buy the same numeric amount of API quota. Currency conversion, fees and final payment appear at checkout.",
      "auditTitle": "Check a charge step by step",
      "audit1": "Filter usage by the actual Key, model and time to find the request. A single task often creates multiple requests.",
      "audit2": "Expand or hover over token and cost details. Check uncached input, output, cache reads and cache writes separately; do not count cached tokens again as uncached input.",
      "audit3": "Check the group, effective multiplier, context tier, reasoning level and any time or peak multipliers. Today’s price card may differ from the price when a historical request ran.",
      "audit4": "Use the actual recorded charge as the source of truth. If it differs from your expectation, share the time, model, request ID (if present) and redacted error through the site support entry. Never share a full Key."
    },
    "calculator": {
      "loginRequired": "The administrator requires sign-in to view model prices. Sign in to estimate costs; the billing guide remains readable.",
      "inputCost": "Uncached input cost", "outputCost": "Output cost", "cacheReadCost": "Cache-read cost", "cacheWriteCost": "Cache-write cost", "rateHint": "Effective group / user billing multiplier: ×{rate}.",

      "title": "GPT text request cost estimator",
      "intro": "Estimate identical requests using this site’s prices from the model directory. Select the group and model you actually use.",
      "empty": "No GPT text price cards are available, or the model directory is disabled. You can still read the billing guide.",
      "failed": "Price cards could not be loaded. Estimates are unavailable.",
      "group": "Billing group",
      "model": "Model",
      "input": "Uncached input tokens",
      "output": "Output tokens",
      "cacheRead": "Cache-read tokens",
      "cacheWrite": "Cache-write tokens",
      "requests": "Identical request count",
      "writeDuration": "Cache-write duration",
      "effort": "Reasoning multiplier",
      "baseEffort": "Base multiplier ×1 (match your request)",
      "period": "Request pricing period",
      "standard": "Standard period ×1",
      "weekdays": "This time multiplier applies on weekdays only; select the standard period on weekends.",
      "peak": "Apply peak multiplier",
      "counting": "Enter each input token once: uncached input, cache reads and cache writes are separate. Their sum selects the context tier. Enter output using the billed token count.",
      "estimated": "Estimated total (USD)",
      "multiplier": "Combined multiplier",
      "tier": "Context tier: {tier}. The entire request uses this tier.",
      "formula": "Total = (uncached input + output + cache-read + cache-write costs) × group/user multiplier × reasoning multiplier × time multiplier × peak multiplier × request count. Line items above are for one request before multipliers.",
      "limitations": "This is a text-token estimate, not a quote or a recalculation of historical bills. Image, video, tool/search charges and priority service differences are excluded. Missing prices are never treated as free. Match time and reasoning settings to your request; actual token counts and charges come from usage records.",
      "errors": {
        "invalid": "Enter non-negative integer token counts and a positive integer request count, and check multiplier selections.",
        "unsupported": "This model or billing mode cannot be estimated here. Check prices and actual usage.",
        "missingPrice": "A required price or tier is missing for this usage. Check actual usage instead."
      }
    },
    "models": {
      "title": "GPT capabilities and connection",
      "intro": "Select a group and model to check declared catalog capabilities, connection protocols and recent observations. A directory entry does not mean every feature is callable with your Key.",
      "protocolHelp": "Choose a protocol",
      "capabilities": "Declared catalog capabilities",
      "max_input_tokens": "Catalog maximum input tokens",
      "max_output_tokens": "Catalog maximum output tokens",
      "supports_vision": "Image input",
      "supports_function_calling": "Tool calling",
      "catalogNote": "Fields come from the identified model’s pricing catalog and are reference information. Missing fields remain unverified. Catalog declarations do not verify end-to-end support or current access.",
      "protocols": "Clients and protocols",
      "chatClients": "Chat clients / SDKs with a custom OpenAI endpoint",
      "protocolNote": "These are this site’s OpenAI connection paths. Codex uses Responses; chat clients need the matching protocol. Model, tool and streaming compatibility still depend on the group and upstream configuration.",
      "specialized": "This is not a text-token model. Use the image or specialized endpoint instructions in the connection center instead of a text request.",
      "availability": "Directory and service status",
      "listed": "Directory status: included in the visible group. Access also depends on your Key’s group, quota, expiry and administrator access rules.",
      "monitorFailed": "Monitor data could not be loaded. Current status is unknown.",
      "unknownStatus": "No matching recent model probe is available. Sign in to see monitors shared with you. Missing observations do not mean the service is down.",
      "monitorNote": "Only monitors visible to your account with an exact group name and primary-model match are shown. Results older than 15 minutes are marked old. A probe observes one request; it does not guarantee the next request or represent global provider status.",
      "unknown": "Unverified",
      "yes": "Declared supported",
      "no": "Declared unsupported",
      "status": {
        "operational": "Last probe operational",
        "degraded": "Last probe degraded",
        "failed": "Last probe failed",
        "error": "Last probe error",
        "stale": "Old observation; current status unverified"
      }
    },
    "articles": {
      "quickstart": {
        "title": "First connection: where do I start?",
        "lead": "You need a valid site API Key and applicable balance or subscription. A site Key and an official ChatGPT membership are different credentials.",
        "step1": "Open the connection center. After signing in, select your valid Key and check its GPT group.",
        "step2": "Choose an administrator-provided endpoint and copy the configuration for your client and operating system. Do not mix site and official API Keys.",
        "step3": "Restart the client. After your own first call, match the Key, model and time in usage records to confirm the request used this site."
      },
      "codex": {
        "title": "Codex configuration and restart checks",
        "lead": "Codex uses the Responses protocol. The connection center retains both existing authentication modes and OS-specific snippets; choose one mode there.",
        "step1": "Open manual configuration, choose OpenAI / Codex, your OS and authentication mode. Use the generated endpoint and configuration instead of guessing a URL.",
        "step2": "In config.toml, match the top-level model_provider to its model_providers section and use responses for wire_api. For environment-key authentication, match the variable name to env_key.",
        "step3": "Save, fully quit and reopen Codex. Choose a model from your Key’s catalog. If the old provider is still active, check project configuration, stale environment variables and the environment inherited by your terminal."
      },
      "chat": {
        "title": "Connect a chat client or SDK",
        "lead": "Use a client that accepts a custom OpenAI endpoint and check whether it uses Chat Completions or Responses.",
        "step1": "Copy configuration for the selected Key from the connection center. A Base URL is not the full request path; avoid duplicating /v1.",
        "step2": "Use the model ID from the site directory. Renaming a client display label does not change the model ID it sends.",
        "step3": "Custom endpoint support does not guarantee every tool feature. For tool or streaming failures, check the protocol and the model’s upstream support."
      },
      "protocol": {
        "title": "Match protocol, model and endpoint",
        "lead": "Model name, wire protocol and client features are separate dimensions. A name alone does not establish compatibility.",
        "step1": "Codex uses Responses (/v1/responses); Chat Completions clients use /v1/chat/completions. Copy the Base URL configuration from the connection center.",
        "step2": "Check your group, model and declared capabilities in the model directory. Catalog fields are declarations; consult monitoring and actual requests for availability.",
        "step3": "A site GPT Key does not activate membership on the official ChatGPT website. Other protocols and platforms depend on services actually configured by the administrator."
      },
      "auth": {
        "title": "401 / 403: authentication or access",
        "lead": "Check the site Key, endpoint, expiry and group, then the authentication mode the client actually uses.",
        "step1": "On the Keys page, confirm the Key is active, not expired and in the right group. Do not send full Keys to support or include them in screenshots.",
        "step2": "Ensure the client sends the site Key to the site endpoint. Remove unintended stale environment overrides and restart the client.",
        "step3": "For access restrictions, check Key IP rules, model access rules and account status. Keep a redacted error and timestamp for investigation."
      },
      "quota": {
        "title": "Quota exhausted, low balance or expired subscription",
        "lead": "Quota windows, Key limits and balance are separate constraints. HTTP status codes vary by protocol and upstream; read the error details.",
        "step1": "Check subscription validity and daily/weekly/monthly usage, then the Key’s own quota and expiry on the Keys page.",
        "step2": "Check the Key’s billing group. Having balance does not automatically bypass subscription limits or purchase/switch plans.",
        "step3": "Use the reset or activation time shown in the UI. A deferred legacy renewal waits for current benefits to end; repeated payments will not make it activate immediately."
      },
      "model": {
        "title": "404 / model_not_found: endpoint or model mismatch",
        "lead": "A 404 can mean a bad path or an unavailable model. Read the response details to distinguish them.",
        "step1": "Compare the full path against the connection center. Avoid /v1/v1, putting /responses into a Base URL, or mixing protocols.",
        "step2": "Check the model ID actually sent and the current Key’s group. Directory visibility does not grant access to every model.",
        "step3": "If configuration is correct, check error records shared with users in the usage page. Note time, model, protocol and redacted error instead of repeatedly changing Keys."
      },
      "rate": {
        "title": "429: rate, concurrency or upstream capacity",
        "lead": "A 429 does not always mean your daily quota is exhausted. Concurrent requests or upstream limits can cause it.",
        "step1": "Reduce parallel tasks and automatic retries. Several clients sharing one Key can amplify traffic.",
        "step2": "Respect Retry-After when present. Otherwise use bounded backoff with jitter rather than an infinite retry loop.",
        "step3": "Check quota and errors to distinguish Key limits, site capacity and upstream limits. Do not buy another plan to fix an undiagnosed capacity issue."
      },
      "network": {
        "title": "Timeouts, connection failures, 5xx and interrupted streams",
        "lead": "First establish whether the request reached the site, then distinguish first-token wait, full generation time and disconnection.",
        "step1": "Check the endpoint and your network. Do not disable certificate verification or enter your Key into an unfamiliar speed-test site.",
        "step2": "First-token latency and total duration are different metrics. A long output can have high total duration without a slow first token. Monitoring is an observation at a particular time.",
        "step3": "Interrupted streams may already have billable output. Check request records before retrying long tasks to avoid additional charges. For persistent issues, share redacted errors and timestamps."
      },
      "billing": {
        "title": "Why did one task cost so much?",
        "lead": "One client action is not one API request. An agent can repeatedly send context, call tools and generate output.",
        "step1": "Filter the task’s time range and total the relevant requests. Conversation input grows over time, and output may include billable reasoning tokens.",
        "step2": "Check detailed prices, multipliers and context tiers. Peak and time-based pricing may also affect the charge.",
        "step3": "Use the estimator to understand current text-token costs. Historical records remain authoritative; image, video and tool fees are outside this estimate."
      },
      "cache": {
        "title": "Cache hits and cost savings",
        "lead": "Cache reads, writes and uncached input can have different prices. Repeated content does not guarantee a hit.",
        "step1": "Calculate uncached input, cache reads and cache writes separately, without counting cached tokens as uncached input again.",
        "step2": "Keep reusable instructions, tool definitions and context prefixes stable when your workflow permits. Eligibility and retention depend on the model and upstream.",
        "step3": "Measure savings using reported usage and actual cost. A cache-hit ratio is not a discount percentage. Compare total costs before and after context compaction when relevant."
      },
      "renewal": {
        "title": "Legacy renewals, scheduled activation and quota resets",
        "lead": "Existing legacy benefits remain usable. New purchases and renewals use current products. Deferred renewal and immediate upgrade are different rules.",
        "step1": "When prepaying, check the activation time. Orders subject to legacy succession show paid and scheduled, and do not immediately replace existing benefits.",
        "step2": "The purchased duration starts on actual activation. If multiple legacy plans remain protected, follow the activation arrangement shown in the UI.",
        "step3": "Daily/weekly/monthly resets are separate from subscription expiry. Daily quota multiplied by days is not a freely spendable pool. Check order status after payment before purchasing again."
      }
    }
  }
}
