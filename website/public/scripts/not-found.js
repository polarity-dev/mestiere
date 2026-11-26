if (/^\/it(\/|$)/.test(location.pathname)) {
  document.documentElement.classList.add("is-it")
  document.documentElement.lang = "it"
  addEventListener("DOMContentLoaded", () => {
    document.title = document.querySelector(".page-it")?.dataset.title ?? document.title
  })
}
