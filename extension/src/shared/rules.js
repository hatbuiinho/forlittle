export function domainFromUrl(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return "";
  }
}

function matches(rule, domain) {
  if (!domain) {
    return false;
  }

  if (rule.pattern_type === "domain_exact") {
    return domain === rule.pattern_value;
  }

  if (rule.pattern_type === "domain_suffix") {
    return domain === rule.pattern_value || domain.endsWith(`.${rule.pattern_value}`);
  }

  return false;
}

export function evaluatePolicy(policy, domain) {
  const rules = policy.rules || [];
  const allowRule = rules.find((rule) => rule.action === "allow" && matches(rule, domain));
  if (allowRule) {
    return { action: "allowed_whitelist", blocked: false };
  }

  const blockRule = rules.find((rule) => rule.action === "block" && matches(rule, domain));
  if (blockRule) {
    return { action: "blocked_blacklist", blocked: true };
  }

  return { action: "allowed", blocked: false };
}
