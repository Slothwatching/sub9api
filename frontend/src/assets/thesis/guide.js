
(function(){
"use strict";
var reduce = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
function $(s,r){return (r||document).querySelector(s);}
function $$(s,r){return Array.prototype.slice.call((r||document).querySelectorAll(s));}
function esc(s){return String(s).replace(/[&<>"]/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c];});}
var store = {
  get:function(k,d){try{var v=localStorage.getItem(k);return v==null?d:JSON.parse(v);}catch(e){return d;}},
  set:function(k,v){try{localStorage.setItem(k,JSON.stringify(v));}catch(e){/* Clipboard fallback is best effort. */}}
};
function toast(t){var el=$('#toast');el.textContent=t;el.classList.add('on');clearTimeout(toast._t);toast._t=setTimeout(function(){el.classList.remove('on');},1800);}
function copyText(t,btn){
  var ok=function(){toast('✓ 已复制，去 Codex 里粘贴吧');if(btn){var o=btn.textContent;btn.textContent='已复制';setTimeout(function(){btn.textContent=o;},1400);}};
  function fallback(){var ta=document.createElement('textarea');ta.value=t;document.body.appendChild(ta);ta.select();var copied=false;try{copied=document.execCommand('copy');}catch(e){copied=false;}document.body.removeChild(ta);if(copied)ok();else toast('复制未成功，请选中提示词手动复制。');}
  if(navigator.clipboard&&navigator.clipboard.writeText)navigator.clipboard.writeText(t).then(ok,fallback);else fallback();
}

/* ---------- network background ---------- */
(function(){
  var c=$('#net'),x=c.getContext('2d'),W,H,pts=[],dpr=Math.min(window.devicePixelRatio||1,2);
  function size(){W=innerWidth;H=innerHeight;c.width=W*dpr;c.height=H*dpr;c.style.width=W+'px';c.style.height=H+'px';x.setTransform(dpr,0,0,dpr,0,0);
    var n=Math.min(70,Math.floor(W*H/22000));pts=[];for(var i=0;i<n;i++)pts.push({x:Math.random()*W,y:Math.random()*H,vx:(Math.random()-.5)*.3,vy:(Math.random()-.5)*.3,r:Math.random()*1.6+.6});}
  var mx=-999,my=-999;addEventListener('mousemove',function(e){mx=e.clientX;my=e.clientY;});
  function draw(){x.clearRect(0,0,W,H);
    for(var i=0;i<pts.length;i++){var p=pts[i];p.x+=p.vx;p.y+=p.vy;if(p.x<0||p.x>W)p.vx*=-1;if(p.y<0||p.y>H)p.vy*=-1;
      for(var j=i+1;j<pts.length;j++){var q=pts[j],d=Math.hypot(p.x-q.x,p.y-q.y);if(d<140){x.strokeStyle='rgba(140,160,255,'+(.14*(1-d/140))+')';x.lineWidth=1;x.beginPath();x.moveTo(p.x,p.y);x.lineTo(q.x,q.y);x.stroke();}}
      var dm=Math.hypot(p.x-mx,p.y-my);if(dm<180){x.strokeStyle='rgba(62,230,255,'+(.35*(1-dm/180))+')';x.beginPath();x.moveTo(p.x,p.y);x.lineTo(mx,my);x.stroke();}
      x.fillStyle='rgba(180,190,255,.55)';x.beginPath();x.arc(p.x,p.y,p.r,0,6.3);x.fill();}
    if(!reduce)requestAnimationFrame(draw);}
  addEventListener('resize',size);size();draw();
})();

/* ---------- scroll rail + reveal ---------- */
addEventListener('scroll',function(){var h=document.documentElement;$('#rail').style.width=(h.scrollTop/(h.scrollHeight-h.clientHeight)*100)+'%';},{passive:true});
var io=new IntersectionObserver(function(es){es.forEach(function(e){if(e.isIntersecting){e.target.classList.add('vis');io.unobserve(e.target);}});},{threshold:.12});
$$('.reveal').forEach(function(el){io.observe(el);});

/* ---------- hero typing ---------- */
(function(){
  var lines=['不想手动一篇篇翻文献？','不想自己整理引用表格？','跟着一个真实例子做：找不到插件，也能用公开学术来源继续。'];
  var el=$('#typed'),li=0,ci=0,txt='';
  function tick(){
    if(li>=lines.length){return;}
    var l=lines[li];
    if(ci<=l.length){el.innerHTML=esc(txt+l.slice(0,ci))+'<span class="cur"></span>';ci++;setTimeout(tick,reduce?0:55);}
    else{txt+=l+(li<lines.length-1?'\n':'');li++;ci=0;el.style.whiteSpace='pre-line';setTimeout(tick,reduce?0:500);}
  }
  tick();
  // counters
  $$('[data-count]').forEach(function(el){var n=+el.dataset.count,s=performance.now();
    (function f(t){var k=Math.min(1,(t-s)/1400);el.textContent=Math.round(n*(1-Math.pow(1-k,3)));if(k<1)requestAnimationFrame(f);})(s);});
  // hero log
  var log=[
    ['c-violet','› 以下是已跑通流程的回放，不会调用你的插件'],
    ['c-cyan','⟳ scite 不可用 → 出版商 / PubMed / PMC 核对'],
    ['c-lime','✓ 5 篇真实 DOI，保留结果不一致的研究'],
    ['c-cyan','⟳ Article Galaxy 不可用 → 单独核实全文获取'],
    ['c-lime','✓ 1 篇开放获取 · 2 篇免费全文 · 2 篇订阅'],
    ['c-amber','GenAI 使用权：全部未核实'],
    ['c-amber','+ data/review.json / scripts/preview.js'],
    ['c-lime','✓ 结构校验通过 ≠ 原文核验通过'],
    ['c-pink','★ 下载完整项目，或按七条主线提示词操作']
  ];
  var hl=$('#heroLog');
  function play(){hl.innerHTML='';log.forEach(function(l,i){setTimeout(function(){var d=document.createElement('div');d.className=l[0];d.textContent=l[1];hl.appendChild(d);},reduce?0:600+i*650);});}
  play();setInterval(play,reduce?1e9:12000);
})();

/* ---------- AI vs YOU cards ---------- */
(function(){
  var cards=[
    ['🔍','检索相关文献','ai','交给 AI','用实际可用的工具查出版商和学术数据库；scite 可用时优先用，不可用就记录替代来源。'],
    ['📊','整理引用数据表','ai','交给 AI','DOI、年份、研究设计、支持/反驳数，全部自动写进 JSON，零手抄。'],
    ['🔓','查能不能读全文','ai','交给 AI','优先用 Article Galaxy；不可用就核对出版商/开放库，只确认可读状态，GenAI 权限写未核实。'],
    ['🎨','做可视化看板','ai','交给 AI','HTML 网页、Excel、Word 都行，还能在页面上批注直接改。'],
    ['✅','核对每条引用','you','你来做','点开 DOI 看原文：作者、年份、结论是否和 AI 写的一致。'],
    ['📖','读懂核心文献','you','你来做','至少读摘要、方法和结论。答辩时老师追问的就是这些。'],
    ['⚖️','判断证据强弱','you','你来做','看研究设计、对照和结果。分类引用数不是结论的支持票数，不能代替证据质量判断。'],
    ['✍️','写出你的论证','you','你来做','AI 可以给提纲和草稿，但论点、取舍和最终文字要你来定。']
  ];
  var g=$('#splitGrid'),seen={};
  cards.forEach(function(c,i){
    var d=document.createElement('div');d.className='flip';d.tabIndex=0;
    d.innerHTML='<div class="flip-in"><div class="face front"><div class="emo">'+c[0]+'</div><h4>'+c[1]+'</h4><p>谁来做？</p><div class="hint">点击翻开 ↻</div></div>'+
      '<div class="face back"><div class="verdict '+c[2]+'">'+(c[2]==='ai'?'🤖 ':'🧑‍🎓 ')+c[3]+'</div><p style="margin-top:8px">'+c[4]+'</p></div></div>';
    function f(){d.classList.toggle('on');seen[i]=1;var n=Object.keys(seen).length;$('#splitScore').textContent='已翻开 '+n+' / 8 张'+(n===8?' · 🎉 分工搞清楚了，往下走':'');if(n===8&&!seen.done){seen.done=1;confetti(innerWidth/2,innerHeight/2);}}
    d.addEventListener('click',f);d.addEventListener('keydown',function(e){if(e.key==='Enter'||e.key===' '){e.preventDefault();f();}});
    g.appendChild(d);
  });
})();

/* ---------- prompt generator ---------- */
// D108: one worked example; no untested discipline/format combinations.
var cfg={title:'限时进食对肥胖人群减重效果的影响研究',q:'对于肥胖成年人，在热量限制的基础上加入限时进食，是否能提高减重效果？',n:5,cite:'GB/T 7714'};
function m(s){return '⟦'+s+'⟧';}
function P(){
  var root='请只在当前选定的论文证据看板文件夹工作。';
  var schema='顶层固定为 {"question":"研究问题","reviewed_at":"YYYY-MM-DD","papers":[]}。每篇字段统一为 title、doi、year、authors（姓名数组）、study_design、verified_at、key_findings、citation_counts（supporting/contrasting/mentioning/source_note）、editorial_statements（correction/retraction/sources/source_note）、evidence_source_note、evidence_sources、full_text_access。来源数组每项用 name、url、description（说明支撑内容及原文位置）。full_text_access 包含 status、basis、sources、genai_usage_rights、verified_at、source_note，可选 read_url。status 只允许“开放获取”“免费全文可获取”“订阅获取”“未核实”。期刊字段 journal、volume、issue、pages 仅填已核实值。';
  return [
    {k:'agents',name:'① 项目规则',text:root+'先检查当前文件夹、可调用的学术检索/联网工具及 Node.js 是否可用；不要安装软件。创建 AGENTS.md：\n【用途】为毕业论文《'+m(cfg.title)+'》搭建文献证据看板。\n【来源】优先使用实际可调用的 scite 查文献/引用/编辑声明，Article Galaxy 查全文与 GenAI 权限；不可用时，用出版商、PubMed、Crossref、PMC 等可访问学术来源逐 DOI 核实，并记录实际来源。不凭记忆编造论文、DOI、工具调用或计数。没有任何联网工具时保留空骨架，给人工检索清单，不假装检索完成。\n【采购】禁止购买、租用、订购或授权文章。\n【数据】未知写“未获取”或“未核实”，结论无证据写“未验证”；分类引用数不代表研究结论票数。\n展示规则和实际环境检查结果，然后停止。'},
    {k:'shell',name:'② 空白骨架',text:root+'遵守 AGENTS.md，创建 index.html、styles.css、app.js、data/review.json 的空骨架，不放示例论文。'+schema+'\napp.js 从第一步就读取 papers 数组，分别预留文献证据和全文获取区。创建只用 Node.js 内置模块的 scripts/preview.js：监听 127.0.0.1、自动选空闲端口并打印网址。不要安装软件包，不用框架/外部库/CDN。Node.js 可用则启动服务并尝试打开浏览器；否则说明缺失条件，不能声称已预览。报告新增文件后停止。'},
    {k:'scite',name:'③ 核实文献',text:root+'研究问题：'+m(cfg.q)+'\n找不超过 5 篇直接相关的随机对照试验或系统综述。优先实际可用的 scite；插件缺失即用出版商/PubMed/Crossref/PMC 等学术来源，不用记忆补论文。逐个打开 DOI 核对标题、作者、年份和设计。保留支持与不支持额外减重的结果，区分组内和组间差异。每篇返回标题、DOI、年份、作者、期刊/卷期/页码（取得才填）、研究设计、关键发现、支撑发现的原文段落或表格及来源链接、支持/反驳/提及计数、编辑声明和核实日期。未知计数写“未获取”，编辑状态无法确认写“未核实”；未发现声明不等于确定不存在。原文不可得时注明仅核摘要。没有联网工具则报告未验证，不编造候选文献。先用表格返回，不修改文件。'},
    {k:'ag',name:'④ 全文与权限',text:root+'检查上一条已核实的 DOI，优先实际可用的 Article Galaxy。不可用则仅根据出版商和开放获取数据库核实。全文状态分别用“开放获取”“免费全文可获取”“订阅获取”“未核实”；订阅不等于我有访问权。另列 GenAI 使用权，拿不到写“未核实”，不把免费阅读或开放许可当专项权限。记录判断依据、来源链接、可读全文地址和实际核实日期。禁止购买、租用、订购或授权。简表返回，不修改文件。'},
    {k:'json',name:'⑤ 写入并校验',text:root+'仅将本聊天已经核实的结果写入 data/review.json，不新增论文或推断结论。'+schema+'\n未知字段保留“未获取”/“未核实”，无证据结论写“未验证”。同时更新 app.js 兼容这个结构，避免换 schema 后页面报错。编写 scripts/validate-review.js，检查各项必填字段、DOI 格式/重复、日期真实性、来源说明及 URL、引用计数（非负整数或未获取）和合法获取分类。空论文只能通过 --allow-empty 骨架检查，不能报研究完成。运行 node scripts/validate-review.js，说明它仅校验结构、不证明 DOI 真实或原文核实通过。报告修改文件和真实校验结果后停止。'},
    {k:'board',name:'⑥ 看板与预览',text:root+'只根据 data/review.json 更新页面：研究问题/审阅日期、各全文状态篇数，每篇标题/DOI/作者/研究设计/关键发现/引用计数/编辑声明/证据来源与链接/获取状态/GenAI 使用权/核实日期。证据和获取区分色；加入“只看现在就能读”开关，仅筛开放获取和免费全文，两块同步，订阅不自动算可读。生成 GB/T 7714 参考文献，仅用已有字段；缺期刊/卷期/页码必须标明不完整。运行校验、启动 scripts/preview.js 并打开实际网址；浏览器无法打开时提供网址让我手动打开。实测开关前后数量、空数据/加载失败提示、窄屏阅读。报告真实结果后停止。'},
    {k:'coll',name:'可选：收藏',text:root+'读取 data/review.json。如果实际可调用 scite 收藏夹功能，将已核实论文添加至新收藏夹“限时进食 review”，报告实际成功和失败条目。不购买或订阅服务；若功能不可用，改为在本地创建 reading-list.md，逐项列出标题、DOI 与已有来源链接。不要承诺邮件提醒，也不要修改 review.json。'},
    {k:'tpl',name:'⑦ 复用模板',text:root+'按当前 papers 结构创建 data/review.template.json（question/reviewed_at 留空、papers 为 []），写 README.md 和 review-checklist.md。说明复制到新文件夹再换问题，不覆盖现有 review.json；写明 node scripts/validate-review.js 和 node scripts/preview.js、Node 缺失与浏览器不能打开时的处理、插件降级和人工逐 DOI/原文复核。校验通过不是事实核验；未知权限不冒认可用。完成后列新增文件并停止。'},
    {k:'draft',name:'可选：提纲',text:root+'先读取学校公开 AI 使用规定或我提供的要求；查不到标未核实，不推定允许。只根据 data/review.json 中已核实发现，为《'+m(cfg.title)+'》整理三级提纲和待我改写的要点，事实陈述对应 DOI 和证据位置。不写标为未验证的内容，不把分类反驳计数直接当研究争议；比较原始研究真实结果及局限，避免综述/试验重复计数。不获取或上传未确认使用权的全文。保存 drafts/lit-review-outline.md，供我对照原文审阅，不作为可直接提交的终稿。'}
  ];
}

var genIdx=0;
function hl(s){return esc(s).replace(/⟦([\s\S]*?)⟧/g,'<mark>$1</mark>');}
function plain(s){return s.replace(/⟦|⟧/g,'');}
function renderGen(){
  var ps=P();
  $('#genTabs').innerHTML=ps.map(function(p,i){return '<button type="button" class="tab'+(i===genIdx?' on':'')+'" data-i="'+i+'">'+p.name+'</button>';}).join('');
  $('#genPre').innerHTML=hl(ps[genIdx].text);
  renderMission(false);
}
$('#genTabs').addEventListener('click',function(e){var b=e.target.closest('.tab');if(!b)return;genIdx=+b.dataset.i;renderGen();});
$('#genCopy').addEventListener('click',function(){copyText(plain(P()[genIdx].text),this);});
// The form is intentionally a fixed, tested example. Prompts are copyable as-is.
function syncForm(){}

/* ---------- levels ---------- */
var LV=[
 {t:'新建本地项目',goal:'选一个空文件夹，让 Codex 能在其中读写文件。下载完整项目也可以直接查看结果。',ops:['打开你已能使用的 Codex 本地聊天；若尚未接通，先看本站“接入教程”','选一个专用空文件夹，例如“论文证据看板演示”，确认聊天能访问它','发送“请告诉我当前工作目录，不要修改文件”，核对确为你选的文件夹'],tip:'界面名称随版本变化，不必寻找完全相同的按钮。不要把项目放在已有论文唯一原件所在目录。',sim:[{s:'流程回放：不会在你的电脑创建文件'},{a:'实际验收：Codex 回报的工作目录与你选择的文件夹一致。'}]},
 {t:'检查工具（插件可跳过）',goal:'先确认现有能力，不把找不到插件当成失败。',ops:['发送“检查你当前实际可调用的联网/学术检索工具和 Node.js；只报告，不安装软件”','scite / Article Galaxy 可用就优先用；找不到就直接采用公开学术检索','没有任何联网工具时，停在待人工核实清单，不能生成假论文'],tip:'输入 @ 只是选择已有工具，不会自动安装或解锁服务。这个例子是在两个插件均不可用时跑通的。',sim:[{s:'本例实际路径：两个插件均不可调用'},{a:'改用出版商、PubMed、PMC 核对 DOI 和结果；GenAI 权限保留未核实。无需安装插件。'}]},
 {t:'写项目规则',p:0,goal:'把来源、禁止采购和未知信息处理写清楚。',ops:['复制提示词①，粘贴到同一个 Codex 聊天并发送','打开 AGENTS.md，核对没有“只许用缺失插件”的排他规则','检查环境报告；Node 缺失时先告知，不自动安装'],tip:'本例问题已填好，整条复制即可。之后每一步沿用同一聊天与文件夹。',sim:[{tool:'回放：创建 AGENTS.md'},{f:'AGENTS.md'},{a:'验收：来源允许降级，禁止采购，未知不编造；环境能力如实说明。'}]},
 {t:'搭看板骨架',p:1,goal:'从一开始统一 papers 数据结构，避免后续写入后页面报错。',ops:['发送提示词②','检查 index.html、styles.css、app.js、data/review.json 和 scripts/preview.js','确认 papers 为 []，页面显示空状态；HTTP 预览成功才进入下一步'],tip:'不要双击 HTML。让 Codex 运行预览脚本，把打印的网址复制进浏览器也能看。',sim:[{tool:'回放：创建空页面、papers 数组及预览脚本'},{f:'index.html'},{f:'styles.css'},{f:'app.js'},{f:'data/review.json'},{f:'scripts/preview.js'},{a:'验收：空网页没有示例论文；预览脚本只监听本机，自动选择空闲端口。'}]},
 {t:'找文献并核实 DOI',p:2,goal:'找到直接比较“热量限制＋限时进食”和“单纯热量限制”的真实研究。',ops:['发送提示词③，先看表格，不写入文件','逐 DOI 打开出版商页，核对题名、作者、设计和原文结果','确认未知计数写未获取；对照是否匹配研究问题，不能只看标题'],tip:'本例收录 3 项随机试验与 2 篇综述，其中一项试验和综述包含超重者。综述与试验重叠，不能作为独立证据累加。',sim:[{tool:'回放：出版商 / PubMed / PMC 交叉核对'},{a:'本例得到 5 篇：短期试验可见额外减重，12 个月试验未达显著优势；分类引用数未获取。先保留表格给你核对。'}]},
 {t:'查全文与使用权限',p:3,goal:'把能否免费读与能否交给 AI 分开记录。',ops:['发送提示词④','确认全文四种分类与 GenAI 权限是两列','不要采购；订阅不等于你当前已获访问权'],tip:'本例：开放获取 1、免费全文 2、订阅 2；GenAI 全部未核实。数字是这次记录，不保证以后永远不变。',sim:[{tool:'回放：核对出版商及 Europe PMC'},{a:'5 篇全文状态：1 开放、2 免费全文、2 订阅。GenAI 使用权均未核实，未采购任何内容。'}]},
 {t:'写入数据 + 结构校验',p:4,goal:'把已核实表格写入统一 JSON，防止数据和页面脱节。',ops:['发送提示词⑤','检查真实 DOI、设计、日期和来源，再运行校验','看到 5 篇结构通过后，仍需人工核对 DOI/原文，不把脚本当事实审查'],tip:'来源打不开或原文不可得就写明核查范围。不能为了通过校验给未知字段编答案。',sim:[{tool:'回放：写入 data/review.json'},{tool:'回放：node scripts/validate-review.js'},{f:'scripts/validate-review.js'},{a:'结构校验通过：5 篇。它不证明 DOI 真实性、原文结论或使用权限。'}]},
 {t:'生成证据看板',p:5,goal:'看清研究证据与全文获取，并实际打开页面检查。',ops:['发送提示词⑥','让 Codex 运行预览脚本，打开输出的实际网址','核对总数 5、可读筛选 3；逐项对照 JSON 与页面'],tip:'下方“成品”直接读取本例五篇 JSON。你的本地页面只会展示自己项目的数据，不会自动同步到本站。',sim:[{tool:'回放：更新 HTML / CSS / JS'},{tool:'回放：校验并启动本地预览'},{a:'验收：全部 5 篇、免费可读 3 篇，两块区域同步筛选，参考文献只用已有字段。',board:1}]},
 {t:'可选：收藏或本地清单',p:6,goal:'保存入口方便重读；这一步不影响主线完成。',ops:['没有 scite 收藏夹功能就跳过，或发送提示词生成本地 reading-list.md','功能确实可用时再创建收藏夹，核对成功条目','不承诺提醒功能，也不为此订阅服务'],tip:'不要为这一步另开会同时改同一数据文件的聊天。保留一份清晰的本地清单已经足够。',sim:[{s:'可跳过，不影响本地看板'},{a:'插件不可用时，改为生成带真实 DOI 和已有来源的本地阅读清单。'}]},
 {t:'检查与修改页面',goal:'用普通中文描述问题，不必寻找特定批注按钮。',ops:['查看本地看板，试开关并核对 5→3→5','直接发送“仅调整样式，不改数据；把两块区域区分清楚”','如有批注功能可用；没有就文字说明，修改后重新校验'],tip:'原文事实不一致要改数据并记录核实来源；纯样式问题只改页面。不要让样式修改补造研究信息。',sim:[{u:'只检查展示：开关开启仅显示开放获取或免费全文，订阅不自动可读。'},{a:'实测筛选应为 3/5，关闭恢复 5/5；整体获取汇总仍统计全部 5 篇。'}]},
 {t:'打包成模板',p:7,goal:'保留本次记录，在新文件夹复用，避免覆盖已有成果。',ops:['发送提示词⑦','检查空模板、README 与人工审阅清单','复制项目到新文件夹后再换研究问题，不覆盖当前 review.json'],tip:'完整下载包内已经带有这三个文件及可直接运行的脚本。空模板还要填问题/日期并重新检索，不能直接算完成。',sim:[{tool:'回放：创建空模板、README、核对清单'},{f:'README.md'},{f:'review-checklist.md'},{f:'data/review.template.json'},{a:'主线完成：保留当前数据，按 README 在副本复用。'}]},
 {t:'加分关：提纲与要点',p:8,bonus:1,goal:'在学校允许的范围内，基于核实证据整理供你改写的提纲。',ops:['先读学校公开规定或你提供的要求；无法确定就标未核实','发送可选提示词，不上传未确认使用权的全文','逐句对照 DOI 与原文证据，自行读懂和改写'],tip:'研究争议来自研究的真实结果和局限，不来自“反驳引用数”投票。草稿不能直接当终稿提交。',sim:[{tool:'回放：只读已核实的 review.json'},{a:'按已核实发现列提纲与证据位置，标明待人工审阅，不补写数据中没有的事实。'}]}
];
var FILES_BY=[[],[],['AGENTS.md'],['index.html','styles.css','app.js','data/review.json','scripts/preview.js'],[],[],['scripts/validate-review.js'],[],[],[],['README.md','review-checklist.md','data/review.template.json'],[]];

var done=store.get('ctg-d108-done',{}),cur=store.get('ctg-d108-cur',0),running=0;
if(!Number.isInteger(cur)||cur<0||cur>=LV.length)cur=0;
function filesUpTo(i){var a=[];for(var k=0;k<i;k++)a=a.concat(FILES_BY[k]);return a;}
function renderLevels(){
  $('#levels').innerHTML=LV.map(function(l,i){return '<button type="button" class="lv'+(i===cur?' on':'')+(done[i]?' done':'')+(l.bonus?' bonus':'')+'" data-i="'+i+'"><span class="n">'+(done[i]?'✓':(l.bonus?'★':i+1))+'</span><span class="t">'+l.t+'</span></button>';}).join('');
}
function renderTree(list,fresh){
  $('#tree').innerHTML=list.length?list.map(function(f){return '<div class="f'+(fresh&&fresh.indexOf(f)>-1?' new':'')+'">'+(f.indexOf('/')>-1?'&nbsp;&nbsp;📄 ':'📄 ')+esc(f)+'</div>';}).join(''):'<div class="empty">（空文件夹）</div>';
}
function renderMission(resetChat){
  var l=LV[cur],ps=P(),xp=l.bonus?'+50 XP':'+'+(10+cur*5)+' XP';
  var h='<div class="mission-top"><span class="lvtag">'+(l.bonus?'BONUS':'LEVEL '+(cur+1)+' / 11')+'</span><h3>'+l.t+'</h3><span class="xp">'+xp+'</span></div>'+
    '<p class="goal">🎯 '+l.goal+'</p><ul class="ops">'+l.ops.map(function(o,i){return '<li><span class="k">'+(i+1)+'</span><span>'+o+'</span></li>';}).join('')+'</ul>'+
    (l.tip?'<div class="tip">💡 '+l.tip+'</div>':'');
  if(l.p!=null){h+='<div class="prompt-box"><div class="lbl">PROMPT · 已按你的论文信息定制</div><button class="btn sm ghost copy-float" type="button" id="mCopy">复制</button><pre>'+hl(ps[l.p].text)+'</pre></div>';}
  h+='<div class="actions"><button class="btn sm" type="button" id="mRun">▶ 模拟运行</button><button class="btn sm ghost" type="button" id="mDone">'+(done[cur]?'✓ 已通关':'✔ 我在 Codex 里做完了，通关')+'</button>'+
    (cur<LV.length-1?'<button class="btn sm ghost" type="button" id="mNext">下一关 →</button>':'')+'</div>';
  $('#mission').innerHTML=h;
  if(l.p!=null)$('#mCopy').onclick=function(){copyText(plain(P()[l.p].text),this);};
  $('#mRun').onclick=runSim;
  $('#mDone').onclick=function(e){if(!done[cur]){done[cur]=1;store.set('ctg-d108-done',done);var r=e.target.getBoundingClientRect();confetti(r.left+r.width/2,r.top);
      var n=Object.keys(done).length;toast(n>=LV.length?'🏆 已勾选全部验收项；请确认你确实在本地完成了操作':'通关！'+xp);if(n>=LV.length)setTimeout(function(){confetti(innerWidth/2,innerHeight/3);},400);}
    renderLevels();renderMission(false);};
  if($('#mNext'))$('#mNext').onclick=function(){go(cur+1);};
  if(resetChat!==false){$('#chat').innerHTML='<div class="chat-empty">点上方「▶ 模拟运行」，看 Codex 在这一关会做什么</div>';renderTree(filesUpTo(cur));}
}
function go(i){if(running)return;cur=i;store.set('ctg-d108-cur',cur);renderLevels();renderMission();}
$('#levels').addEventListener('click',function(e){var b=e.target.closest('.lv');if(b)go(+b.dataset.i);});
function runSim(){
  if(running)return;running=1;
  var l=LV[cur],chat=$('#chat'),files=filesUpTo(cur),ps=P(),delay=0,step=reduce?50:750;
  chat.innerHTML='';renderTree(files);
  function add(cls,html){var d=document.createElement('div');d.className='msg '+cls;d.innerHTML=html;chat.appendChild(d);chat.scrollTop=chat.scrollHeight;return d;}
  if(l.p!=null){var first=plain(ps[l.p].text).split('\n').filter(Boolean).slice(0,3).join('\n');setTimeout(function(){add('user',esc(first)+'<span class="c-dim"> …</span>');},delay);delay+=step;}
  l.sim.forEach(function(ev){
    setTimeout(function(){
      if(ev.s)add('sys','— '+esc(ev.s)+' —');
      else if(ev.u)add('user',esc(ev.u));
      else if(ev.tool){var d=add('tool','<span class="spin"></span>'+esc(ev.tool));setTimeout(function(){d.classList.add('ok');},reduce?0:600);}
      else if(ev.f){files=files.concat([ev.f]);renderTree(files,[ev.f]);}
      else if(ev.a){var d2=add('ai','');typeInto(d2,ev.a,function(){if(ev.board){d2.insertAdjacentHTML('beforeend','<div class="mini-board"><div>🟦 文献证据区</div><div>🟪 全文获取区</div></div><div style="margin-top:8px"><a href="#board">↓ 查看完整看板效果</a></div>');}chat.scrollTop=chat.scrollHeight;});}
    },delay);
    delay+=ev.a?step*1.5:(ev.f?step*.5:step);
  });
  setTimeout(function(){running=0;},delay+(reduce?0:1500));
}
function typeInto(el,text,cb){
  if(reduce){el.textContent=text;el.style.whiteSpace='pre-line';cb&&cb();return;}
  var i=0;el.style.whiteSpace='pre-line';
  (function t(){i+=3;el.textContent=text.slice(0,i);$('#chat').scrollTop=1e6;if(i<text.length)setTimeout(t,16);else cb&&cb();})();
}

/* ---------- evidence board ---------- */
var PAPERS=[], boardReview=null;
function renderBoardQ(){$('#bQ').textContent=boardReview?boardReview.question:cfg.q;}
function renderBoard(){
  $('#bDate').textContent=boardReview.reviewed_at;renderBoardQ();
  $('#papers').innerHTML=PAPERS.map(function(p,i){
    var a=p.full_text_access,cs=p.citation_counts,ed=p.editorial_statements;
    var readable=a.status==='开放获取'||a.status==='免费全文可获取';
    var correction=typeof ed.correction==='object'?ed.correction.description+'（'+ed.correction.date+'）':ed.correction;
    var source=p.evidence_sources[0];
    var author=p.authors.slice(0,3).map(function(name){var parts=name.trim().split(/\s+/),last=parts.pop().toUpperCase();return last+' '+parts.map(function(part){return part.split('-').map(function(piece){return piece[0].toUpperCase();}).join('');}).join('');}).join(', ')+(p.authors.length>3?', et al':'');
    return '<article class="panel paper" data-readable="'+readable+'"><div class="doi">'+esc(p.doi)+'</div><h4>['+(i+1)+'] '+esc(p.title)+'</h4><span class="design">'+esc(p.study_design)+'</span>'+
      '<div class="zone s"><div class="zt">文献证据 · 已记录来源</div><div style="font-size:14px">'+esc(p.key_findings)+'</div>'+
      '<p style="font-size:12px">分类引用计数：支持 '+esc(cs.supporting)+' · 反驳 '+esc(cs.contrasting)+' · 提及 '+esc(cs.mentioning)+'</p>'+
      '<p style="font-size:12px">更正：'+esc(correction)+'<br>撤稿：'+esc(ed.retraction)+'</p>'+
      '<a href="'+esc(source.url)+'" target="_blank" rel="noopener noreferrer">查看证据来源 ↗</a><p style="font-size:12px">'+esc(source.description)+'</p></div>'+
      '<div class="zone a"><div class="zt">全文获取 · 与 GenAI 权限分开</div><span class="pill '+(readable?'oa':'warn')+'">'+esc(a.status)+'</span><span class="pill warn">GenAI 使用权：'+esc(a.genai_usage_rights)+'</span>'+
      '<p style="font-size:12px">'+esc(a.basis)+'</p>'+(readable&&a.read_url?'<a href="'+esc(a.read_url)+'" target="_blank" rel="noopener noreferrer">打开已记录全文 ↗</a>':'')+'</div>'+
      '<details style="font-size:12px;margin-top:12px"><summary>作者、参考文献与核实日期</summary><p>'+esc(p.authors.join('；'))+'</p><p>'+esc(author+'. '+p.title+'[J]. '+p.journal+', '+p.year+', '+p.volume+'('+p.issue+'): '+p.pages+'. DOI: '+p.doi+'.')+'</p><p>记录核实日期：'+esc(p.verified_at)+'；GB/T 7714 格式引文仅使用已记录字段，提交前人工复核。</p></details></article>';
  }).join('');
  var statuses=[['开放获取','#7dff9b'],['免费全文可获取','#9b6bff'],['订阅获取','#3ee6ff'],['未核实','#ffc35a']],off=25,tot=PAPERS.length;
  var svg='<circle cx="21" cy="21" r="15.9" fill="none" stroke="rgba(255,255,255,.08)" stroke-width="5"/>';
  $('#legend').innerHTML=statuses.map(function(s){var count=PAPERS.filter(function(p){return p.full_text_access.status===s[0];}).length;var pc=tot?count/tot*100:0;svg+='<circle cx="21" cy="21" r="15.9" fill="none" stroke="'+s[1]+'" stroke-width="5" stroke-dasharray="'+pc+' '+(100-pc)+'" stroke-dashoffset="'+off+'"/>';off-=pc;return '<div><i style="background:'+s[1]+'"></i>'+s[0]+' '+count+'</div>';}).join('');
  $('#donut').innerHTML=svg+'<text x="21" y="24" text-anchor="middle" fill="#eef0ff" font-size="9" font-weight="700">'+tot+'</text>';
  applyAccessible();
}
var tg=$('#accToggle');
function applyAccessible(){var on=tg.getAttribute('aria-checked')==='true';var count=0;$$('.paper').forEach(function(p){var hide=on&&p.dataset.readable!=='true';p.hidden=hide;if(!hide)count++;});$('#boardCount').textContent='当前显示 '+count+' / '+PAPERS.length+' 篇；获取汇总统计全部文献。';}
function flipAcc(){var on=tg.getAttribute('aria-checked')!=='true';tg.classList.toggle('on',on);tg.setAttribute('aria-checked',String(on));applyAccessible();}
tg.addEventListener('click',flipAcc);tg.addEventListener('keydown',function(e){if(e.key==='Enter'||e.key===' '){e.preventDefault();flipAcc();}});
async function loadBoard(){
  try{var response=await fetch('/guides/thesis-example/data/review.json',{cache:'no-store'});if(!response.ok)throw new Error('HTTP '+response.status);var review=await response.json();if(!review||!Array.isArray(review.papers))throw new Error('数据格式错误');boardReview=review;PAPERS=review.papers;renderBoard();}
  catch(error){$('#bDate').textContent='未获取';$('#papers').textContent='例子暂未加载成功，请刷新页面，或下载完整项目阅读 data/review.json。'+error.message;$('#boardCount').textContent='数据未加载，不显示虚构论文或计数。';}
}

/* ---------- confetti ---------- */
var cf=$('#confetti'),cx=cf.getContext('2d'),parts=[],cfOn=0;
function confetti(x,y){
  if(reduce)return;
  cf.width=innerWidth;cf.height=innerHeight;
  var cols=['#3ee6ff','#9b6bff','#ff5fa8','#7dff9b','#ffc35a'];
  for(var i=0;i<120;i++){var a=Math.random()*Math.PI*2,v=Math.random()*9+3;parts.push({x:x,y:y,vx:Math.cos(a)*v,vy:Math.sin(a)*v-6,g:.28,s:Math.random()*6+3,r:Math.random()*6,vr:(Math.random()-.5)*.4,c:cols[i%5],life:90+Math.random()*40});}
  if(!cfOn){cfOn=1;loop();}
  function loop(){cx.clearRect(0,0,cf.width,cf.height);
    parts=parts.filter(function(p){return p.life-->0;});
    parts.forEach(function(p){p.vy+=p.g;p.vx*=.985;p.x+=p.vx;p.y+=p.vy;p.r+=p.vr;cx.save();cx.translate(p.x,p.y);cx.rotate(p.r);cx.globalAlpha=Math.min(1,p.life/40);cx.fillStyle=p.c;cx.fillRect(-p.s/2,-p.s/4,p.s,p.s/2);cx.restore();});
    if(parts.length)requestAnimationFrame(loop);else{cfOn=0;cx.clearRect(0,0,cf.width,cf.height);}}
}

/* ---------- init ---------- */
syncForm();renderLevels();renderGen();renderMission();loadBoard();
})();


// Sub9API D99 integration: chapter and fragment navigation only.
// D98: chapter navigation only. The original guide owns all content and effects.
(function () {
  'use strict'
  var ids = ['overview', 'split', 'gen', 'sim', 'board', 'traps', 'cheat']
  var previous = ''
  var queued = false
  // about:srcdoc has a "null" location.origin, while its base URL and the
  // same-origin parent retain the real site origin.
  var parentOrigin = new URL(document.baseURI).origin
  // srcdoc inherits the parent URL as its base. Keep fragment links inside
  // the guide rather than requesting the main app inside the frame.
  document.addEventListener('click', function (event) {
    var link = event.target.closest('a[href^="#"]')
    if (!link) return
    var target = document.getElementById(link.getAttribute('href').slice(1))
    if (!target) return
    event.preventDefault()
    target.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start' })
  })
  function section(id) {
    return id === 'overview' ? document.querySelector('.hero') : document.getElementById(id)
  }
  function report() {
    queued = false
    var current = 'overview'
    ids.forEach(function (id) {
      var el = section(id)
      if (el && el.getBoundingClientRect().top <= 160) current = id
    })
    if (current !== previous) {
      previous = current
      window.parent.postMessage({ type: 'sloth-thesis:section', section: current }, parentOrigin)
    }
  }
  window.addEventListener('message', function (event) {
    if (event.origin !== parentOrigin || event.source !== window.parent) return
    if (!event.data || event.data.type !== 'sloth-thesis:navigate' || ids.indexOf(event.data.section) < 0) return
    var el = section(event.data.section)
    if (el) el.scrollIntoView({ behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'auto' : 'smooth', block: 'start' })
  })
  window.addEventListener('scroll', function () {
    if (!queued) { queued = true; requestAnimationFrame(report) }
  }, { passive: true })
  report()
})()
