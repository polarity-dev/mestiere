// Local preview of mestiere.io: builds into dist/, serves it like CloudFront does
// (clean URLs, trailing-slash redirects, 404.html) and rebuilds on every change.
//
//   npm run dev            -> http://127.0.0.1:8080
//   PORT=3000 npm run dev
import { createReadStream, existsSync, statSync, watch } from 'node:fs';
import { createServer } from 'node:http';
import { extname, join, normalize } from 'node:path';
import { fileURLToPath } from 'node:url';

const ROOT = fileURLToPath(new URL('.', import.meta.url));
const DIST = join(ROOT, 'dist');
const PORT = Number(process.env.PORT || 8080);

const TYPES = {
  '.html': 'text/html; charset=utf-8',
  '.css': 'text/css; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.md': 'text/markdown; charset=utf-8',
  '.txt': 'text/plain; charset=utf-8',
  '.xml': 'application/xml; charset=utf-8',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
  '.woff2': 'font/woff2',
  '.webmanifest': 'application/manifest+json',
};

// Fresh import on every build so template and build.mjs edits apply without a restart.
async function rebuild() {
  const started = Date.now();
  try {
    const { build } = await import(`./build.mjs?t=${started}`);
    const { pages } = build();
    console.log(`rebuilt ${pages} pages in ${Date.now() - started} ms`);
  } catch (err) {
    console.error(`build failed: ${err.message}`);
  }
}

await rebuild();

let timer;
for (const dir of ['content', 'templates', 'static']) {
  watch(join(ROOT, dir), { recursive: true }, () => {
    clearTimeout(timer);
    timer = setTimeout(rebuild, 100);
  });
}
watch(join(ROOT, 'build.mjs'), () => { clearTimeout(timer); timer = setTimeout(rebuild, 100); });

createServer((req, res) => {
  const url = new URL(req.url, 'http://localhost');
  let path = decodeURIComponent(url.pathname);
  // Same language rules as the CloudFront function (infra/site.yaml)
  const lang = url.searchParams.get('lang');
  if (lang === 'en' || lang === 'it') {
    url.searchParams.delete('lang');
    res.writeHead(302, {
      location: path + (url.search || ''),
      'set-cookie': `mestiere_lang=${lang}; Path=/; Max-Age=31536000; SameSite=Lax`,
      'cache-control': 'no-store',
    }).end();
    return;
  }
  const isPage = path.endsWith('/') || !extname(path);
  if (isPage && !path.startsWith('/it/') && path !== '/it') {
    const cookie = (req.headers.cookie || '').match(/(?:^|;\s*)mestiere_lang=(en|it)/);
    const preferred = cookie ? cookie[1] : (req.headers['accept-language'] || '').trim().slice(0, 2).toLowerCase();
    if (preferred === 'it') {
      res.writeHead(302, { location: '/it' + path + url.search, 'cache-control': 'no-store', vary: 'Accept-Language, Cookie' }).end();
      return;
    }
  }
  const file = normalize(join(DIST, path));
  if (!file.startsWith(DIST)) {
    res.writeHead(400).end();
    return;
  }
  // Same rule as the CloudFront function: /docs -> 301 /docs/
  if (!path.endsWith('/') && !extname(path) && existsSync(file) && statSync(file).isDirectory()) {
    res.writeHead(301, { location: path + '/' + url.search }).end();
    return;
  }
  let target = path.endsWith('/') ? join(file, 'index.html') : file;
  let status = 200;
  if (!existsSync(target) || statSync(target).isDirectory()) {
    target = join(DIST, '404.html');
    status = 404;
  }
  res.writeHead(status, {
    'content-type': TYPES[extname(target)] || 'application/octet-stream',
    'cache-control': 'no-store',
  });
  createReadStream(target).pipe(res);
}).listen(PORT, '127.0.0.1', () => {
  console.log(`mestiere.io preview on http://127.0.0.1:${PORT}`);
});
