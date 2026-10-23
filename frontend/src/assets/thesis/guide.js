
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
  if(navigator.clipboard&&navigator.clipboard.writeText){navigator.clipboard.writeText(t).then(ok,function(){fb();ok();});}else{fb();ok();}
  function fb(){var ta=document.createElement('textarea');ta.value=t;document.body.appendChild(ta);ta.select();try{document.execCommand('copy');}catch(e){/* Clipboard fallback is best effort. */}document.body.removeChild(ta);}
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
  var lines=['不想手动一篇篇翻文献？','不想自己整理引用表格？','让 Codex 找文献、查权限、做看板 —— 你只负责读懂和下判断。'];
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
    ['c-violet','› 只用 @scite 研究我的论文问题，最多 5 篇'],
    ['c-cyan','⟳ scite.search_literature("time-restricted eating" AND obesity …)'],
    ['c-dim','  534 条结果 → 自动收窄检索式…'],
    ['c-lime','✓ 找到 4 篇 RCT / 系统综述，均有真实 DOI'],
    ['c-cyan','⟳ ArticleGalaxy.check_access(4 DOIs)'],
    ['c-lime','✓ 2 篇开放获取 · 1 篇订阅 · 1 篇 GenAI 不可用'],
    ['c-amber','+ data/review.json'],
    ['c-amber','+ scripts/validate-review.js'],
    ['c-lime','✓ 校验通过：4/4 篇字段完整'],
    ['c-pink','★ 证据看板已生成 → index.html']
  ];
  var hl=$('#heroLog');
  function play(){hl.innerHTML='';log.forEach(function(l,i){setTimeout(function(){var d=document.createElement('div');d.className=l[0];d.textContent=l[1];hl.appendChild(d);},reduce?0:600+i*650);});}
  play();setInterval(play,reduce?1e9:12000);
})();

/* ---------- AI vs YOU cards ---------- */
(function(){
  var cards=[
    ['🔍','检索相关文献','ai','交给 AI','接入 scite 后，Codex 会自己构造检索式、反复调整，几分钟就能筛出核心文献。'],
    ['📊','整理引用数据表','ai','交给 AI','DOI、年份、研究设计、支持/反驳数，全部自动写进 JSON，零手抄。'],
    ['🔓','查能不能读全文','ai','交给 AI','Article Galaxy 一次查完开放获取、订阅、GenAI 使用权。'],
    ['🎨','做可视化看板','ai','交给 AI','HTML 网页、Excel、Word 都行，还能在页面上批注直接改。'],
    ['✅','核对每条引用','you','你来做','点开 DOI 看原文：作者、年份、结论是否和 AI 写的一致。'],
    ['📖','读懂核心文献','you','你来做','至少读摘要、方法和结论。答辩时老师追问的就是这些。'],
    ['⚖️','判断证据强弱','you','你来做','“支持 18、反驳 0”说明什么、算不算强证据，这个判断要你来下。'],
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
var FIELDS={
  med:'随机对照试验或系统综述',soc:'实证研究、准实验研究或元分析',eco:'实证研究、面板数据分析或元分析',
  eng:'实验研究、对比评测或系统综述',hum:'同行评审的专著章节、期刊论文或综述'
};
var cfg=store.get('ctg-cfg',null)||{title:$('#fTitle').value,q:$('#fQ').value,field:'med',n:5,cite:'GB/T 7714'};
function m(s){return '⟦'+s+'⟧';}
function P(){
  var c=cfg,t=c.title||'【论文题目】',q=c.q||'【研究问题】',ty=FIELDS[c.field],n=c.n;
  return [
  {k:'agents',name:'① 项目规则',text:
'在项目根目录创建 AGENTS.md，写入以下项目规则：\n\n【用途】为我的毕业论文《'+m(t)+'》搭建文献证据看板。\n\n【文献来源】\n- 文献发现、论文元数据、研究发现、引用语境（支持/反驳/提及）和编辑声明（更正、关注声明、撤稿），只使用 @scite。\n- 全文获取方式、订阅状态和 GenAI 使用权，只使用 @Article Galaxy。\n- 不要使用网页搜索，也不要使用以上两者以外的任何来源。\n- 只收录有真实 DOI 的已验证文献。\n\n【采购】不得购买、租用、订购或授权任何文章。\n\n【数据】如果 scite 没有返回支持某条结论的证据，标注为“未验证”，不要从其他来源补充。\n\n写完后把文件内容展示给我，然后停止。'},
  {k:'shell',name:'② 看板骨架',text:
'遵守 AGENTS.md，在本地创建一个“文献证据看板”的空白骨架：\n- index.html：页面结构\n- styles.css：页面样式\n- app.js：从 data/review.json 读取数据并渲染\n- data/review.json：目前为空，不要放示例论文\n\n要求：\n- 不安装任何软件包，不使用框架、外部库或 CDN。\n- 为 @scite 证据和 @Article Galaxy 获取权限分别预留独立区域。\n- 完成后告诉我改动了哪些文件，并在内置浏览器里打开页面。'},
  {k:'scite',name:'③ 找文献',text:
'只使用 @scite 研究这个问题：\n'+m(q)+'\n\n找出不超过 '+m(n)+' 篇直接相关的'+m(ty)+'。\n\n每篇返回：\n1. 标题、DOI、年份、作者、研究设计\n2. 与研究问题直接相关的发现\n3. scite 引用计数：支持 / 反驳 / 提及\n4. 任何更正、关注声明、撤稿或其他编辑声明\n5. 支撑该发现所用的 scite 证据\n\n不要使用网页搜索或 scite 以外的来源。\n这一步先不要修改任何文件。'},
  {k:'ag',name:'④ 查权限',text:
'只使用 @Article Galaxy 检查 scite 刚找到的这些 DOI。\n不要购买、租用、订购或授权任何内容。\n\n对每篇论文判断：开放获取 / 订阅获取 / 可购买 GenAI 使用权 / GenAI 使用权不可用。\n用一张简短的表格给出结果。'},
  {k:'json',name:'⑤ 写入数据',text:
'把已核实的结果写入 data/review.json，顶层结构：\n{ "question": 研究问题, "reviewed_at": 审阅日期, "papers": [ ... ] }\n\n- 同时保留 @scite 证据和 @Article Galaxy 获取信息。\n- scite 没有返回支持证据的，写“未验证”，不要用其他来源补充。\n\n再编写 scripts/validate-review.js，要求每篇论文必须包含：DOI、研究设计、核实日期、scite 来源说明、所用证据、引用计数，以及 Article Galaxy 来源说明和一个合法的获取类型分类。\n\n运行校验脚本，告诉我改动了哪些文件以及校验结果。'},
  {k:'board',name:'⑥ 生成看板',text:
'根据 data/review.json 更新证据看板，显示：\n- 研究问题和审阅日期\n- Article Galaxy 获取情况汇总（各类状态分别有几篇）\n- 每篇论文的 DOI、研究设计、关键发现\n- scite 引用计数：支持 / 反驳 / 提及\n- 任何编辑声明\n- 获取状态、GenAI 使用权状态和显示的价格\n- 每篇论文的 '+m(c.cite)+' 格式参考文献（只用已有字段）\n\n让 @scite 区块和 @Article Galaxy 区块在视觉上明显区分。\n不要添加来自其他来源的信息。\n完成后运行校验脚本，并打开更新后的页面。'},
  {k:'coll',name:'⑦ 存收藏夹',text:
'把 data/review.json 里的论文添加到一个新的 @scite 收藏夹（Collection），命名为“'+m(t.slice(0,16))+' review”。'},
  {k:'tpl',name:'⑧ 做成模板',text:
'把这个项目整理成可复用的模板：\n- 基于 data/review.json 的结构，创建空模板 data/review.template.json\n- 写 README.md，说明如何用一个新的研究问题复用这个流程\n- 写 review-checklist.md，列出每次复用都必须完成的检查步骤\n不要删除当前的 review.json。\n\n以后复用时只需要说：\n按照 README.md 和 review-checklist.md，为这个新问题生成证据看板：【论文第 X 章的子问题】'},
  {k:'draft',name:'⑨ 加分：提纲草稿',text:
'只根据 data/review.json 中已核实的文献，为论文《'+m(t)+'》的文献综述部分生成：\n1. 一份三级提纲\n2. 每个小节 150 字左右的要点草稿\n\n硬性要求：\n- 每一句事实性陈述后都标注来源 DOI；review.json 里没有的内容一律不写。\n- 有“反驳”引用的文献，必须在“研究争议”小节里体现。\n- 标注为“未验证”的内容不得使用。\n- 输出到 drafts/lit-review-outline.md，不要修改其他文件。\n\n这只是供我改写的草稿，不是终稿。'}
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
function syncForm(){
  $('#fTitle').value=cfg.title;$('#fQ').value=cfg.q;$('#fN').value=cfg.n;$('#fNv').textContent=cfg.n;$('#fCite').value=cfg.cite;
  $$('#fField .chip').forEach(function(ch){ch.classList.toggle('on',ch.dataset.v===cfg.field);});
}
function upd(){cfg.title=$('#fTitle').value.trim();cfg.q=$('#fQ').value.trim();cfg.n=+$('#fN').value;$('#fNv').textContent=cfg.n;cfg.cite=$('#fCite').value;store.set('ctg-cfg',cfg);renderGen();renderBoardQ();}
['#fTitle','#fQ','#fN','#fCite'].forEach(function(s){$(s).addEventListener('input',upd);});
$('#fField').addEventListener('click',function(e){var ch=e.target.closest('.chip');if(!ch)return;cfg.field=ch.dataset.v;syncForm();upd();});

/* ---------- levels ---------- */
var LV=[
 {t:'新建本地项目',goal:'给 Codex 一个专属工作区：一个空文件夹。之后它所有的读写都只发生在这里。',
  ops:['打开 ChatGPT 桌面应用，切到 <b>Codex</b>','点 <kbd>New chat</kbd> → 新建 <b>Project</b>','位置选 <kbd>Local</kbd>（不要选 Remote）','新建一个空文件夹，比如 <code>我的论文证据看板</code>','在项目下再点 <kbd>New chat</kbd>，确认对话框上方显示的是这个文件夹'],
  tip:'同时在访达（Finder）里打开这个文件夹，就能看到 Codex 实时新建的文件，很直观。',
  sim:[{s:'项目已创建：我的论文证据看板（Local）'},{a:'项目已就绪。当前文件夹为空，等待指令。'}],files:[]},
 {t:'安装文献插件',goal:'接入专业文献库，让 Codex 不再“凭记忆”引用文献。',
  ops:['左侧打开 <kbd>Plugins</kbd>','搜索 <b>scite</b> → 安装 → 登录授权 → 返回','搜索 <b>Article Galaxy</b> → 安装 → 登录授权','回到对话，输入 <code>@</code> 能看到这两个插件，说明安装成功'],
  tip:'全程不到 1 分钟。以后提示词里都用 @scite / @Article Galaxy 明确点名，不要指望它自己猜。',
  sim:[{tool:'Plugins › scite 已连接'},{tool:'Plugins › Article Galaxy 已连接'},{a:'现在可以调用 scite 检索文献、调用 Article Galaxy 查询全文权限了。'}],files:[]},
 {t:'写项目规则 AGENTS.md',p:0,goal:'给 Codex 定“家规”。这个文件它每次回答前都会读，比在聊天里反复叮嘱可靠得多。',
  ops:['复制下方提示词，粘贴到 Codex 并发送','等它写完，点开 <code>AGENTS.md</code>','切换“原文 / 预览”，确认来源、采购、数据三类规则都在'],
  tip:'为什么禁止网页搜索？智能体会“抓住一切能抓的东西”。只允许用有真实 DOI 的文献，能把编造引用的风险降到最低。',
  sim:[{tool:'write_file AGENTS.md'},{f:'AGENTS.md'},{a:'已创建 AGENTS.md，包含：用途、文献来源（仅 scite / Article Galaxy）、采购禁令、数据规则。已停止，等待下一步。'}]},
 {t:'搭看板骨架',p:1,goal:'先搭一个空网页框架，后面往里填数据。',
  ops:['发送提示词','观察右侧：Codex 依次创建 HTML / CSS / JS / JSON 文件','它会自动启动本地服务，并在 <kbd>Browser</kbd> 里打开预览','对样式不满意？发一张参考截图，说“照这个风格改”'],
  tip:'HTML 是网页结构，CSS 是颜色和字体，JS 负责交互。看不懂没关系，开一个 Side chat 问它。',
  sim:[{tool:'write_file index.html'},{f:'index.html'},{tool:'write_file styles.css'},{f:'styles.css'},{tool:'write_file app.js'},{f:'app.js'},{tool:'write_file data/review.json'},{f:'data/review.json'},{tool:'terminal › 启动本地预览服务'},{a:'骨架已完成，review.json 为空。页面已为 scite 和 Article Galaxy 预留独立区域，已在内置浏览器中打开。'}]},
 {t:'用 scite 找文献',p:2,goal:'最关键的一关：找到真实、相关、有引用信号的核心文献。',
  ops:['发送提示词（研究问题已经替换成你的）','展开工具调用，可以看到 Codex 自己写的布尔检索式','它会多次调整检索式，这是正常现象','看结果时重点看“支持 / 反驳”数，而不是总被引数'],
  tip:'这一关可以多聊几轮：“再找找反驳第 1 篇的研究”“排除 2015 年以前的”，直到确定核心文献清单。',
  sim:[{tool:'scite.search_literature(query: 你的研究问题 → 布尔检索式)'},{s:'返回 534 条 → 收窄检索式'},{tool:'scite.search_literature(加入研究设计限定)'},{tool:'scite.get_citation_context(4 DOIs)'},{a:'找到 4 篇相关文献，结论有分歧：1 篇发现额外减重，2 篇未发现，1 篇报告了代谢指标改善。\n\n论文 A：支持 18 · 反驳 0 · 提及 470\n论文 C：支持 12 · 反驳 3 · 提及 112\n\n按要求，未修改任何文件。'}]},
 {t:'查全文权限',p:3,goal:'弄清楚每篇文献：能不能读？能不能合规地交给 AI？',
  ops:['发送提示词','得到一张表：开放获取 / 订阅 / 可购买 GenAI 使用权 / 不可用','有的还会显示购买、租用价格（只看，不要买）'],
  tip:'订阅获取 ≠ 可以喂给 AI。“GenAI 使用权不可用”的文献，只能你自己读。',
  sim:[{tool:'ArticleGalaxy.check_access(4 DOIs)'},{a:'| 论文 | 状态 |\n| A | 订阅获取 |\n| B | GenAI 使用权不可用 |\n| C | 开放获取 |\n| D | 开放获取 |\n\n未进行任何购买。'}]},
 {t:'写入数据 + 自动校验',p:4,goal:'把结果存进 review.json。以后读文件就行，不用重复检索，省时又省钱。',
  ops:['发送提示词','Codex 会逐篇把文献写成结构化数据','它会写一个校验脚本并自动运行','看到“4/4 通过”就算通关'],
  tip:'从这一步开始，后续任务都只读文件，可以切到更便宜的模型。',
  sim:[{tool:'write_file data/review.json (4 papers)'},{tool:'write_file scripts/validate-review.js'},{f:'scripts/validate-review.js'},{tool:'terminal › node scripts/validate-review.js'},{a:'✓ 校验通过：4 篇论文的必填字段和获取类型分类全部合格。'}]},
 {t:'生成证据看板',p:5,goal:'把数据变成一眼就能看懂的可视化页面。',
  ops:['发送提示词，等待 2–5 分钟','可以实时看到它改写 HTML / CSS / JS','完成后在 <kbd>Browser</kbd> 里查看看板','往下滚动到「成品」部分，看看示意效果'],
  tip:'不一定要做网页：也可以说“导出成 Excel 阅读清单”或“生成 Word 版”，方便发给导师。',
  sim:[{tool:'read_file data/review.json'},{tool:'edit index.html'},{tool:'edit styles.css'},{tool:'edit app.js'},{tool:'terminal › node scripts/validate-review.js ✓'},{a:'看板已更新：研究问题、获取汇总（2 篇开放获取 · 1 篇订阅 · 1 篇不可用）、4 张论文卡片，scite 和 Article Galaxy 区块分色显示。',board:1}]},
 {t:'存进 scite 收藏夹',p:6,goal:'等上一关运行时并行做：把文献存到云端，以后有新的引用会自动提醒你。',
  ops:['在同一个项目里点 <kbd>New chat</kbd> 开一个新对话','发送提示词','把两个对话都 <kbd>Pin</kbd> 置顶，方便来回切换','到 scite 网站的 Collections 页面刷新，确认收藏夹已出现'],
  tip:'论文写作周期长，收藏夹里的论文有新引用时，scite 会发邮件提醒你，帮你跟上最新研究。',
  sim:[{s:'新对话 · 同一项目文件夹'},{tool:'read_file data/review.json'},{tool:'scite.create_collection(4 papers)'},{a:'已创建 scite 收藏夹，添加并匹配了 4 篇论文。'}]},
 {t:'在页面上批注修改',goal:'不用写提示词：直接在网页上圈出要改的地方。',
  ops:['在 <kbd>Browser</kbd> 中打开看板','点 <kbd>Annotate</kbd>','点选页面上的位置，写修改意见，比如“加一个‘现在就能读’的开关”','可以一次批注多处，还能附上截图','点 <kbd>Send</kbd>，所有批注会合并成一条消息发给 Codex'],
  tip:'其他好用的批注：“只显示反驳数大于 0 的论文”“每张卡片加一个已读/未读标记”“加一个复制参考文献的按钮”。',
  sim:[{u:'[批注 @ 获取汇总区] 加一个“Accessible now / 现在就能读”开关'},{tool:'edit app.js'},{tool:'edit index.html'},{a:'已添加开关：打开后只显示开放获取或订阅可读的论文，当前 3/4 篇可读。'}]},
 {t:'打包成模板',p:7,goal:'一次搭建，反复使用。论文每一章、每个子问题都能直接复用。',
  ops:['发送提示词','确认新增了 README.md、review-checklist.md 和 data/review.template.json','以后只需一句话：“按 README 为这个新问题生成证据看板”'],
  tip:'这个文件夹本身就是你的上下文：不用重新粘贴说明，也不用重新上传文件。可以放进网盘，和导师共享。',
  sim:[{tool:'write_file README.md'},{f:'README.md'},{tool:'write_file review-checklist.md'},{f:'review-checklist.md'},{tool:'write_file data/review.template.json'},{f:'data/review.template.json'},{a:'模板已完成。下次只需要提供一个新的研究问题即可。'}]},
 {t:'加分关：提纲 + 草稿',p:8,bonus:1,goal:'在核实过的证据上，让 Codex 起草综述提纲。每句话都可以溯源，方便你改写。',
  ops:['先确认学校和导师允许使用 AI 辅助写作','发送提示词','打开 <code>drafts/lit-review-outline.md</code>，逐条对照 DOI 检查','用你自己的话改写，并加入你的判断和取舍'],
  tip:'草稿的价值在于“不用从白纸开始”，而不是直接交上去。每句都标了 DOI，答辩时被问到也能马上找到出处。',
  sim:[{tool:'read_file data/review.json'},{tool:'write_file drafts/lit-review-outline.md'},{f:'drafts/lit-review-outline.md'},{a:'已生成三级提纲和各小节要点草稿，共 27 处 DOI 标注；“研究争议”小节引用了论文 C 的 3 条反驳证据。'}]}
];
var FILES_BY=[[],[],['AGENTS.md'],['index.html','styles.css','app.js','data/review.json'],[],[],['scripts/validate-review.js'],[],[],[],['README.md','review-checklist.md','data/review.template.json'],['drafts/lit-review-outline.md']];
var done=store.get('ctg-done',{}),cur=store.get('ctg-cur',0),running=0;
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
  $('#mDone').onclick=function(e){if(!done[cur]){done[cur]=1;store.set('ctg-done',done);var r=e.target.getBoundingClientRect();confetti(r.left+r.width/2,r.top);
      var n=Object.keys(done).length;toast(n>=LV.length?'🏆 全部通关！你已经搭好了自己的论文证据系统':'通关！'+xp);if(n>=LV.length)setTimeout(function(){confetti(innerWidth/2,innerHeight/3);},400);}
    renderLevels();renderMission(false);};
  if($('#mNext'))$('#mNext').onclick=function(){go(cur+1);};
  if(resetChat!==false){$('#chat').innerHTML='<div class="chat-empty">点上方「▶ 模拟运行」，看 Codex 在这一关会做什么</div>';renderTree(filesUpTo(cur));}
}
function go(i){if(running)return;cur=i;store.set('ctg-cur',cur);renderLevels();renderMission();}
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
      else if(ev.a){var d2=add('ai','');typeInto(d2,ev.a,function(){if(ev.board){d2.insertAdjacentHTML('beforeend','<div class="mini-board"><div>🟦 scite 证据区</div><div>🟪 Article Galaxy 权限区</div></div><div style="margin-top:8px"><a href="#board">↓ 查看完整看板效果</a></div>');}chat.scrollTop=chat.scrollHeight;});}
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
var PAPERS=[
  {id:'A',title:'示例论文 A：限时进食 + 热量限制的 12 个月随机对照试验',doi:'10.xxxx/demo.a',design:'随机对照试验',find:'与单纯热量限制相比，未观察到具有统计学意义的额外减重。',s:18,c:0,m:470,acc:'sub',gen:'no',notice:null},
  {id:'B',title:'示例论文 B：限时进食对体重影响的系统综述与荟萃分析',doi:'10.xxxx/demo.b',design:'系统综述',find:'限时进食组体重下降略多，但研究间异质性较高。',s:9,c:0,m:140,acc:'none',gen:'no',notice:null},
  {id:'C',title:'示例论文 C：早时段限时进食的随机对照试验',doi:'10.xxxx/demo.c',design:'随机对照试验',find:'早时段进食组减重更明显，部分后续研究的结果与之相反。',s:12,c:3,m:112,acc:'oa',gen:'na',notice:null},
  {id:'D',title:'示例论文 D：限时进食对代谢指标的影响',doi:'10.xxxx/demo.d',design:'随机对照试验',find:'减重差异不显著，但血压和胰岛素敏感性有所改善。',s:3,c:1,m:45,acc:'oa',gen:'na',notice:'2023-09-08 利益冲突声明更正'}
];
var ACC={oa:['开放获取','oa'],sub:['订阅获取','sub'],none:['无订阅','warn']},GEN={no:['GenAI 使用权不可用','no'],na:['GenAI 使用权：未返回','warn']};
function renderBoardQ(){$('#bQ').textContent=cfg.q||'【你的研究问题】';}
function renderBoard(){
  $('#bDate').textContent=new Date().toISOString().slice(0,10);renderBoardQ();
  var mx=Math.max.apply(null,PAPERS.map(function(p){return p.m;}));
  $('#papers').innerHTML=PAPERS.map(function(p){
    function bar(lbl,v,col,base){return '<div class="bar"><span>'+lbl+'</span><div class="tr"><div class="fl" data-w="'+Math.max(2,v/base*100)+'" style="background:'+col+'"></div></div><b>'+v+'</b></div>';}
    return '<div class="panel paper" data-acc="'+p.acc+'"><div class="doi">'+p.doi+'</div><h4>'+p.title+'</h4><span class="design">'+p.design+'</span>'+
      '<div class="zone s"><div class="zt">SCITE · 证据</div><div style="font-size:14px">'+p.find+'</div><div class="bars">'+
        bar('支持',p.s,'var(--lime)',20)+bar('反驳',p.c,'var(--red)',20)+bar('提及',p.m,'var(--dim)',mx)+'</div>'+
        (p.notice?'<div style="margin-top:8px"><span class="pill warn">⚠ '+p.notice+'</span></div>':'<div style="margin-top:8px;font-size:12px;color:var(--dim)">无编辑声明</div>')+'</div>'+
      '<div class="zone a"><div class="zt">ARTICLE GALAXY · 权限</div><span class="pill '+ACC[p.acc][1]+'">'+ACC[p.acc][0]+'</span><span class="pill '+GEN[p.gen][1]+'">'+GEN[p.gen][0]+'</span></div></div>';
  }).join('');
  var segs=[['开放获取',2,'#7dff9b'],['订阅获取',1,'#3ee6ff'],['GenAI 不可用',1,'#ff6b6b']],off=25,tot=4,svg='<circle cx="21" cy="21" r="15.9" fill="none" stroke="rgba(255,255,255,.08)" stroke-width="5"/>';
  segs.forEach(function(s){var pc=s[1]/tot*100;svg+='<circle cx="21" cy="21" r="15.9" fill="none" stroke="'+s[2]+'" stroke-width="5" stroke-dasharray="'+pc+' '+(100-pc)+'" stroke-dashoffset="'+off+'"/>';off-=pc;});
  svg+='<text x="21" y="24" text-anchor="middle" fill="#eef0ff" font-size="9" font-weight="700">4</text>';
  $('#donut').innerHTML=svg;
  $('#legend').innerHTML=segs.map(function(s){return '<div><i style="background:'+s[2]+'"></i>'+s[0]+' '+s[1]+'</div>';}).join('')+'<div style="color:var(--dim)"><i style="background:#555"></i>可购买 GenAI 0</div>';
  var bio=new IntersectionObserver(function(es){es.forEach(function(e){if(e.isIntersecting){$$('.fl',e.target).forEach(function(f){f.style.width=f.dataset.w+'%';});bio.unobserve(e.target);}});},{threshold:.3});
  $$('.paper').forEach(function(p){bio.observe(p);});
}
var tg=$('#accToggle');
function flipAcc(){var on=!tg.classList.contains('on');tg.classList.toggle('on',on);tg.setAttribute('aria-checked',on);
  $$('.paper').forEach(function(p){p.classList.toggle('hide',on&&!(p.dataset.acc==='oa'||p.dataset.acc==='sub'));});
  if(on)toast('3 / 4 篇现在就能读');}
tg.addEventListener('click',flipAcc);tg.addEventListener('keydown',function(e){if(e.key==='Enter'||e.key===' '){e.preventDefault();flipAcc();}});

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
syncForm();renderLevels();renderGen();renderMission();renderBoard();
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
