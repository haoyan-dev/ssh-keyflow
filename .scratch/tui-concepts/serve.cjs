// Throwaway local-only viewer; no dependencies or external requests.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const files = new Set(['prototype.html', 'prototype.js', 'prototype.css', 'README.md']);
const types = {'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.css':'text/css; charset=utf-8','.md':'text/plain; charset=utf-8'};
const port = Number(process.env.KEYFLOW_PROTOTYPE_PORT || 4318);
http.createServer((req,res) => {
  const name = new URL(req.url, 'http://localhost').pathname.slice(1) || 'prototype.html';
  if (!files.has(name)) { res.writeHead(404); res.end('Not found'); return; }
  fs.readFile(path.join(__dirname,name), (error,content) => {
    res.writeHead(error ? 404 : 200, {'Content-Type':types[path.extname(name)],'Cache-Control':'no-store'});
    res.end(error ? 'Not found' : content);
  });
}).listen(port,'127.0.0.1',() => console.log(`TUI prototype: http://127.0.0.1:${port}/prototype.html`));
