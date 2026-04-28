export function domainFromUrl(url) {
  try {
    return new URL(url).hostname;
  } catch {
    return "";
  }
}

const accentMap = new Map([
  ["àáãảạăằắẳẵặâầấẩẫậä", "a"],
  ["èéẻẽẹêềếểễệë", "e"],
  ["ìíỉĩịïî", "i"],
  ["òóỏõọôồốổỗộơờớởỡợö", "o"],
  ["ùúủũụưừứửữựüû", "u"],
  ["ýỳỹỵỷ", "y"],
  ["ñ", "n"],
  ["ç", "c"]
]);

function removeAccent(character) {
  if (character === "đ") {
    return "d";
  }

  for (const [from, to] of accentMap) {
    if (from.includes(character)) {
      return to;
    }
  }

  return character;
}

function normalizeText(value) {
  return String(value || "")
    .toLowerCase()
    .trim()
    .split("")
    .map(removeAccent)
    .join("")
    .replace(/[^a-z0-9]+/g, " ")
    .replace(/\s+/g, " ")
    .trim();
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

  if (rule.pattern_type === "domain_contains") {
    return domain.includes(rule.pattern_value);
  }

  return false;
}

function parseKeywordList(value) {
  try {
    const parsed = JSON.parse(value);
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.map(normalizeText).filter(Boolean);
  } catch {
    return [];
  }
}

function titleMatches(rule, title) {
  if (rule.pattern_type !== "title_contains_any") {
    return false;
  }

  const normalizedTitle = normalizeText(title);
  if (!normalizedTitle) {
    return false;
  }

  return parseKeywordList(rule.pattern_value).some((keyword) => normalizedTitle.includes(keyword));
}

export function evaluatePolicy(policy, domain) {
  const rules = policy.rules || [];
  const blockRule = rules.find((rule) => rule.action === "block" && matches(rule, domain));
  if (blockRule) {
    return { action: "blocked_blacklist", blocked: true };
  }

  if (policy.default_action === "block") {
    const allowRule = rules.find((rule) => rule.action === "allow" && matches(rule, domain));
    if (allowRule) {
      return { action: "allowed_whitelist", blocked: false };
    }

    return { action: "blocked_default", blocked: true };
  }

  return { action: "allowed", blocked: false };
}

export function evaluateTitlePolicy(policy, title) {
  const rules = policy.rules || [];
  const blockRule = rules.find((rule) => rule.action === "block" && titleMatches(rule, title));
  if (blockRule) {
    return { action: "blocked_title", blocked: true };
  }

  return { action: "allowed", blocked: false };
}
