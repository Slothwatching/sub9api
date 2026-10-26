(async function () {
  const fs = await import('node:fs');
  const path = await import('node:path');
  const http = await import('node:http');
  const root = path.resolve(path.dirname(process.argv[1]), '..');
  const types = { '.html': 'text/html; charset=utf-8', '.css': 'text/css; charset=utf-8', '.js': 'application/javascript; charset=utf-8', '.json': 'application/json; charset=utf-8', '.md': 'text/plain; charset=utf-8' };
  const server = http.createServer((req, res) => {
    let name;
    try { name = decodeURIComponent(new URL(req.url, 'http://localhost').pathname); }
    catch { res.writeHead(400); res.end('地址格式不正确'); return; }
    const file = path.resolve(root, '.' + (name === '/' ? '/index.html' : name));
    if (!file.startsWith(root + path.sep)) { res.writeHead(403); res.end(); return; }
    fs.readFile(file, (error, data) => {
      if (error) { res.writeHead(404); res.end('文件未找到'); return; }
      res.setHeader('Content-Type', types[path.extname(file)] || 'application/octet-stream');
      res.setHeader('Cache-Control', 'no-store');
      res.end(data);
    });
  });
  // OS chooses an unused port; students do not need to resolve port collisions.
  server.listen(0, '127.0.0.1', () => {
    console.log(`预览已启动：http://127.0.0.1:${server.address().port}/`);
    console.log('在浏览器打开上面的网址；保持终端运行，结束按 Ctrl+C。');
  });
  server.on('error', error => { console.error('预览启动失败：' + error.message); process.exitCode = 1; });
})();
