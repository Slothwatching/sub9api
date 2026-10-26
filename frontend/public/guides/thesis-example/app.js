"use strict";

const statusElement = document.getElementById("load-status");
const evidenceList = document.getElementById("evidence-list");
const accessList = document.getElementById("access-list");
const filter = document.getElementById("readable-only");
const accessStatuses = ["开放获取", "免费全文可获取", "订阅获取", "未核实"];
const readableStatuses = new Set(["开放获取", "免费全文可获取"]);
let papers = [];

function displayValue(value) {
  return value === undefined || value === null || value === "" ? "未获取" : String(value);
}
function element(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = displayValue(text);
  if (className) node.className = className;
  return node;
}
function field(details, label, value, className) {
  const description = element("dd", undefined, className);
  if (value instanceof Node) description.append(value);
  else description.textContent = displayValue(value);
  details.append(element("dt", label), description);
}
function safeLink(label, url) {
  try {
    const parsed = new URL(url);
    if (!["https:", "http:"].includes(parsed.protocol)) throw new Error("无效协议");
    const link = element("a", label);
    link.href = parsed.href;
    link.target = "_blank";
    link.rel = "noopener noreferrer";
    return link;
  } catch {
    return element("span", label);
  }
}
function sourceList(sources) {
  if (!Array.isArray(sources) || !sources.length) return element("p", "未获取");
  const list = element("ul", undefined, "sources");
  for (const source of sources) {
    const item = element("li");
    item.append(safeLink(source.name, source.url), element("p", source.description, "source-note"));
    list.append(item);
  }
  return list;
}
function doiLink(doi) {
  return safeLink(doi, `https://doi.org/${encodeURIComponent(doi)}`);
}
function authorForReference(name) {
  // Existing names use given-name-first order; preserve hyphenated given names as initials.
  const parts = name.trim().split(/\s+/);
  const surname = parts.pop().toUpperCase();
  const initials = parts.map(part => part.split("-").map(piece => piece[0]?.toUpperCase() || "").join(""));
  return surname + (initials.length ? " " + initials.join("") : "");
}
function reference(paper, number) {
  const authors = Array.isArray(paper.authors) ? paper.authors : [];
  const names = authors.slice(0, 3).map(authorForReference);
  if (authors.length > 3) names.push("et al");
  const publication = paper.journal ? `${paper.journal}, ${displayValue(paper.year)}${paper.volume ? ", " + paper.volume : ""}${paper.issue ? "(" + paper.issue + ")" : ""}${paper.pages ? ": " + paper.pages : ""}` : displayValue(paper.year);
  return `[${number}] ${names.join(", ") || "未获取"}. ${displayValue(paper.title)}[J]. ${publication}. DOI: ${displayValue(paper.doi)}.`;
}
function badge(status) {
  return element("span", status, `badge${readableStatuses.has(status) ? " readable" : ""}`);
}
function evidenceCard(paper, number) {
  const card = element("article", undefined, "card");
  card.id = `evidence-${number}`;
  card.append(element("h3", `[${number}] ${paper.title}`));
  const details = element("dl");
  field(details, "DOI", doiLink(paper.doi));
  field(details, "作者", paper.authors?.join("；"));
  field(details, "年份", paper.year);
  field(details, "研究设计", paper.study_design);
  field(details, "关键发现", paper.key_findings, "finding");
  const counts = paper.citation_counts || {};
  field(details, "引用计数", `支持：${displayValue(counts.supporting)} ／ 反驳：${displayValue(counts.contrasting)} ／ 提及：${displayValue(counts.mentioning)}`);
  field(details, "引用计数来源说明", counts.source_note);
  const editorial = paper.editorial_statements || {};
  const correction = editorial.correction;
  if (correction && typeof correction === "object") {
    const content = element("div");
    content.append(element("p", correction.description), element("p", `发布日期：${displayValue(correction.date)}`));
    if (correction.doi) content.append(doiLink(correction.doi));
    field(details, "编辑声明 · 更正", content);
  } else field(details, "编辑声明 · 更正", correction);
  field(details, "编辑声明 · 撤稿", editorial.retraction);
  if (editorial.source_note) field(details, "编辑声明核查说明", editorial.source_note);
  field(details, "编辑声明来源", sourceList(editorial.sources));
  field(details, "证据核实日期", paper.verified_at);
  field(details, "证据来源说明", paper.evidence_source_note);
  field(details, "证据来源", sourceList(paper.evidence_sources));
  field(details, "GB/T 7714 参考文献" + (paper.journal && paper.volume && paper.issue && paper.pages ? "" : "（字段不全）"), reference(paper, number), "reference");
  card.append(details);
  return card;
}
function accessCard(paper, number) {
  const access = paper.full_text_access || {};
  const card = element("article", undefined, "card");
  card.id = `access-${number}`;
  card.append(element("h3", `[${number}] ${paper.title}`), badge(access.status || "未核实"));
  const details = element("dl");
  field(details, "DOI", doiLink(paper.doi));
  if (access.read_url && readableStatuses.has(access.status)) field(details, "阅读全文", safeLink("打开已记录的全文来源", access.read_url));
  field(details, "获取依据", access.basis);
  field(details, "GenAI 使用权", access.genai_usage_rights || "未核实");
  field(details, "权限核实日期", access.verified_at);
  field(details, "获取来源", sourceList(access.sources));
  field(details, "获取来源说明", access.source_note);
  card.append(details);
  return card;
}
function renderSummary() {
  const summary = document.getElementById("access-summary");
  summary.replaceChildren();
  for (const status of accessStatuses) {
    const item = element("dl", undefined, "summary-item");
    const count = papers.filter(paper => paper.full_text_access?.status === status).length;
    item.append(element("dt", status), element("dd", `${count} 篇`));
    summary.append(item);
  }
}
function renderPapers() {
  evidenceList.replaceChildren();
  accessList.replaceChildren();
  let count = 0;
  papers.forEach((paper, index) => {
    if (filter.checked && !readableStatuses.has(paper.full_text_access?.status)) return;
    evidenceList.append(evidenceCard(paper, index + 1));
    accessList.append(accessCard(paper, index + 1));
    count++;
  });
  if (!count) {
    evidenceList.append(element("p", "暂无符合条件的文献证据。", "empty-state"));
    accessList.append(element("p", "暂无符合条件的全文获取记录。", "empty-state"));
  }
  statusElement.textContent = `当前显示 ${count} / ${papers.length} 篇论文${filter.checked ? "（只看现在就能读）" : ""}。`;
}
filter.addEventListener("change", renderPapers);

async function loadReview() {
  try {
    const response = await fetch("data/review.json");
    if (!response.ok) throw new Error(`读取失败（${response.status}）`);
    const review = await response.json();
    if (!review || !Array.isArray(review.papers) || !review.papers.every(paper => paper && typeof paper.title === "string" && typeof paper.doi === "string" && accessStatuses.includes(paper.full_text_access?.status))) {
      throw new Error("数据格式无效");
    }
    papers = review.papers;
    document.getElementById("research-question").textContent = displayValue(review.question);
    const date = document.getElementById("reviewed-at");
    date.textContent = displayValue(review.reviewed_at);
    if (/^\d{4}-\d{2}-\d{2}$/.test(review.reviewed_at)) date.dateTime = review.reviewed_at;
    renderSummary();
    renderPapers();
    filter.disabled = false;
  } catch (error) {
    evidenceList.replaceChildren();
    accessList.replaceChildren();
    document.getElementById("access-summary").replaceChildren();
    document.getElementById("research-question").textContent = "研究问题未获取";
    filter.disabled = true;
    statusElement.textContent = `无法加载看板数据：${error.message}。请通过 HTTP 服务打开页面，并检查 data/review.json。`;
  }
}
loadReview();
