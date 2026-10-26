// Works in a downloaded folder without package.json, and in an ES module project.
(async function () {
  const fs = await import('node:fs');
  const path = await import('node:path');
  const file = path.resolve(path.dirname(process.argv[1]), '../data/review.json');
  const errors = [];
  const statuses = ['开放获取', '免费全文可获取', '订阅获取', '未核实'];
  const text = v => typeof v === 'string' && v.trim().length > 0;
  const known = v => text(v) && !['未获取', '未核实', '未验证'].includes(v.trim());
  const date = v => {
    if (!text(v) || !/^\d{4}-\d{2}-\d{2}$/.test(v)) return false;
    const d = new Date(v + 'T00:00:00Z');
    return !Number.isNaN(d.getTime()) && d.toISOString().slice(0, 10) === v;
  };
  const url = v => { try { return /^https?:$/.test(new URL(v).protocol); } catch { return false; } };
  const sources = v => Array.isArray(v) && v.length > 0 && v.every(s => s && known(s.name) && known(s.description) && url(s.url));
  try {
    const data = JSON.parse(fs.readFileSync(file, 'utf8'));
    if (!data || !known(data.question)) errors.push('研究问题不能为空');
    if (!date(data?.reviewed_at)) errors.push('审阅日期必须是有效 YYYY-MM-DD');
    if (!Array.isArray(data?.papers)) throw new Error('papers 必须是数组');
    if (!data.papers.length && !process.argv.includes('--allow-empty')) errors.push('尚未收录论文；空骨架可加 --allow-empty 检查，不能作为研究完成');
    const seen = new Set();
    data.papers.forEach((p, i) => {
      const label = `第 ${i + 1} 篇`;
      const check = (condition, message) => { if (!condition) errors.push(`${label}：${message}`); };
      if (!p || typeof p !== 'object' || Array.isArray(p)) { errors.push(`${label}：条目必须为对象`); return; }
      check(known(p.title), '缺少标题');
      check(text(p.doi) && /^10\.\d{4,9}\/\S+$/i.test(p.doi), 'DOI 格式无效');
      const doi = String(p.doi).toLowerCase();
      check(!seen.has(doi), 'DOI 重复'); seen.add(doi);
      check(Number.isInteger(p.year) || p.year === '未获取', '年份须为整数或未获取');
      check(Array.isArray(p.authors) && p.authors.length > 0 && p.authors.every(text), '作者须为非空姓名数组');
      check(known(p.study_design), '缺少研究设计');
      check(date(p.verified_at), '核实日期无效');
      check(text(p.key_findings), '关键发现不能为空；无法验证时写未验证');
      check(known(p.evidence_source_note) && sources(p.evidence_sources), '缺少具体证据来源说明或有效来源');
      for (const kind of ['supporting', 'contrasting', 'mentioning']) {
        const count = p.citation_counts?.[kind];
        check((Number.isInteger(count) && count >= 0) || count === '未获取', `引用计数 ${kind} 须为非负整数或未获取`);
      }
      check(text(p.citation_counts?.source_note), '缺少引用计数来源说明');
      const editorial = p.editorial_statements;
      check(text(editorial?.retraction), '缺少撤稿核查结果');
      check(text(editorial?.correction) || (text(editorial?.correction?.description) && date(editorial?.correction?.date) && /^10\.\d{4,9}\/\S+$/i.test(editorial?.correction?.doi)), '更正声明字段无效');
      check(sources(editorial?.sources), '缺少编辑声明核查来源');
      const access = p.full_text_access;
      check(statuses.includes(access?.status), '获取分类不合法');
      check(known(access?.basis) && sources(access?.sources), '缺少获取依据或来源');
      check(date(access?.verified_at), '获取信息核实日期无效');
      check(text(access?.genai_usage_rights), 'GenAI 使用权未填；未知写未核实');
      check(text(access?.source_note), '缺少获取来源说明');
      if (access?.read_url != null) check(url(access.read_url), '全文链接须为 http/https');
    });
    if (errors.length) throw new Error(errors.join('\n'));
    console.log(`${data.papers.length ? '结构校验通过' : '空骨架检查通过（研究未完成）'}：${data.papers.length} 篇论文。`);
    console.log('此脚本不验证 DOI 真实性、原文结论或使用权限；仍须人工核对 DOI 和原文。');
  } catch (error) {
    console.error('校验失败：\n' + error.message);
    process.exitCode = 1;
  }
})();
