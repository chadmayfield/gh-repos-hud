// Progressive enhancement: when JS is on, update #content in place instead of
// doing a full <meta refresh> page reload (no flicker). With JS off, the meta
// refresh in the page head remains the fallback.
(function () {
  "use strict";
  var script = document.currentScript;
  var interval = (parseInt(script.getAttribute("data-interval"), 10) || 30) * 1000;

  // JS is active: remove the meta refresh so we don't also full-reload.
  var meta = document.querySelector('meta[http-equiv="refresh"]');
  if (meta && meta.parentNode) meta.parentNode.removeChild(meta);

  // Adopt parsed nodes rather than assigning innerHTML: DOMParser already
  // neutralizes <script>, and node adoption avoids the string->HTML path.
  // (Source is our own loopback server; values are html/template-escaped.)
  function swap(selector, doc) {
    var fresh = doc.querySelector(selector);
    var cur = document.querySelector(selector);
    if (!fresh || !cur) return;
    var nodes = [];
    fresh.childNodes.forEach(function (n) { nodes.push(document.importNode(n, true)); });
    cur.replaceChildren.apply(cur, nodes);
  }

  async function tick() {
    try {
      var res = await fetch("/", { cache: "no-store" });
      var html = await res.text();
      var doc = new DOMParser().parseFromString(html, "text/html");
      swap("#content", doc);
      swap(".meta", doc);
    } catch (e) {
      /* transient; try again next tick */
    }
    setTimeout(tick, interval);
  }

  setTimeout(tick, interval);
})();
