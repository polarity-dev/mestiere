// 404 page only, loaded in <head> without defer so it runs before the first paint.
// CloudFront serves the same 404.html for every missing path, so the file carries both languages
// and this picks the Italian one under /it/. Without JavaScript the English page shows.
if (/^\/it(\/|$)/.test(location.pathname)) {
  document.documentElement.classList.add('is-it');
  document.documentElement.lang = 'it';
  document.title = 'Pagina non trovata · mestiere';
}
