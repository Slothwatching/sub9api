# 限时进食文献证据看板：已跑通的例子

这是 slothcode 论文教程配套项目，使用原生 HTML/CSS/JavaScript，无框架、软件包或 CDN。5 篇论文的来源与获取信息审阅于 2026-10-26，属于一次问题检索示范，不是完整系统综述或医疗建议。原始随机试验与综述存在重叠，不能作为五份独立证据计数。全文可读不等于可以上传给 AI。

## 零基础操作

1. 解压下载包，在 Codex 里选择这个文件夹作为本地工作区。不同版本按钮名称可能不同，重点是让 Codex 能读写这个文件夹。
2. 发送：“读取 AGENTS.md 和 README.md，检查当前文件夹和 Node.js 是否可用。不要安装软件、不要修改文件，告诉我下一步。”
3. 发送：“运行 node scripts/validate-review.js；通过后运行 node scripts/preview.js，把实际预览网址告诉我，并尝试打开浏览器。无法打开时我会复制网址手动打开。”
4. 对照论文原文核对结果，不要把结构校验通过当成文献事实核验通过。

如果自己操作终端，在这个解压文件夹内执行：

```sh
node scripts/validate-review.js
node scripts/preview.js
```

第二条命令会输出一个本机网址，端口自动选择，不与其他项目争抢固定端口。保持终端运行，复制网址到任意浏览器；结束按 Ctrl+C。不要直接双击 index.html，浏览器可能阻止读取 JSON。

Node.js 不存在时不需要盲目安装依赖：先让 Codex 报告环境。你仍可在教程网页查看完整例子，下载 JSON/README 阅读；本地执行步骤须在 Node.js 可用后继续。若当前助手没有联网工具，让它保留空骨架并给出人工检索清单，不得凭记忆填入论文。

## 用新问题复用

复制整个解压文件夹为一个新文件夹，不要覆盖旧项目。让 Codex 在新文件夹把 data/review.template.json 复制为 data/review.json，再填写 question 和 reviewed_at。更新 AGENTS.md 中的用途，保留来源、采购禁令与不编造规则。论文数组为空，不要复制本例结论到新问题。

在同一聊天按教程七条主线提示词逐步操作。每一步完成后查看文件/表格和验收项，再继续下一步。插件可选：scite 不可用时用 PubMed、Crossref、出版商等实际可用的学术来源逐 DOI 核对；Article Galaxy 不可用时仅查出版商或开放获取数据库，GenAI 使用权写“未核实”。所有来源保留网址和说明。

## JSON 约定

顶层固定为 `{ "question": "研究问题", "reviewed_at": "YYYY-MM-DD", "papers": [] }`，空模板的问题/日期必须先填。骨架检查用 `node scripts/validate-review.js --allow-empty`，正式校验必须有论文。

每篇包含 title、doi、year、authors（按原文顺序的姓名数组）、study_design、verified_at、key_findings、citation_counts、editorial_statements、evidence_source_note、evidence_sources、full_text_access。期刊元数据使用 journal、volume、issue、pages；仅记录已核实内容，不从来源名称猜期刊名。示例完整结构可见 data/review.json。

- citation_counts 包含 supporting、contrasting、mentioning、source_note。计数未知写“未获取”，不能写 0；分类引用数不代表原论文结论的支持票数。
- editorial_statements 包含 correction、retraction、sources 和可选 source_note。correction 可为文字，或 `{ "date": "YYYY-MM-DD", "doi": "声明 DOI", "description": "更正内容" }`。未发现声明不等于确定没有。
- 每个来源为 `{ "name": "来源名称", "url": "真实 http/https 链接", "description": "支持何项发现及原文位置" }`。证据来源数组、编辑声明来源数组和获取来源数组均使用这一格式。
- full_text_access 包含 status、basis、sources、genai_usage_rights、verified_at、source_note，可选 read_url。status 只允许“开放获取”“免费全文可获取”“订阅获取”“未核实”。未知权限写“未核实”；不得把 GenAI 权限作为全文分类。read_url 仅记录已确认的全文地址。
- 未取得的资料写“未获取”或“未核实”；没有证据支持的结论写“未验证”。研究设计、真实 DOI、日期和来源等正式必填项未查齐时，留在待核实清单，不能伪造字段让校验通过。

## 验收

运行校验脚本后，按 review-checklist.md 人工核对 DOI、原文、获取与使用权。页面蓝色证据区和暖黄色获取区独立展示；“只看现在就能读”仅筛开放获取/免费全文，本例 3/5。订阅不表示你当前有机构访问权。分类汇总始终统计全部文献。

GB/T 7714 引文只用已记录的作者、题名、期刊、年份、卷期、页码与 DOI；缺字段标明不完整，正式提交前人工核对。作者缩写也需复核，尤其含复姓、姓名前缀或机构作者时。
