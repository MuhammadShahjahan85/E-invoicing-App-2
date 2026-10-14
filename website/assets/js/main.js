/* ==========================================================================
   Veridian Partners — website behaviour (no libraries).
   ========================================================================== */
(function () {
  'use strict';

  var doc = document;
  var root = doc.documentElement;
  var body = doc.body;
  var BASE = body.getAttribute('data-base') || '';
  var reduceMotion = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  function $(sel, ctx) { return (ctx || doc).querySelector(sel); }
  function $$(sel, ctx) { return Array.prototype.slice.call((ctx || doc).querySelectorAll(sel)); }
  function on(el, ev, fn, opts) { if (el) el.addEventListener(ev, fn, opts || false); }
  function escapeHtml(s) { return String(s).replace(/[&<>"']/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]; }); }
  function store(key, val) { try { if (val === undefined) return localStorage.getItem(key); localStorage.setItem(key, val); } catch (e) { return null; } }

  /* ---------- Toast ---------- */
  var toastEl = $('[data-toast]');
  var toastTimer;
  function toast(msg) {
    if (!toastEl) return;
    toastEl.textContent = msg;
    toastEl.hidden = false;
    clearTimeout(toastTimer);
    toastTimer = setTimeout(function () { toastEl.hidden = true; }, 2600);
  }

  /* ---------- Theme ---------- */
  function currentTheme() {
    var t = root.getAttribute('data-theme');
    if (t) return t;
    return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  $$('[data-theme-toggle]').forEach(function (btn) {
    on(btn, 'click', function () {
      var next = currentTheme() === 'dark' ? 'light' : 'dark';
      root.setAttribute('data-theme', next);
      store('vp-theme', next);
      toast(next === 'dark' ? 'Dark theme on' : 'Light theme on');
    });
  });

  /* ---------- Header state ---------- */
  var header = $('[data-header]');
  var toTop = $('[data-to-top]');
  var ticking = false;
  function onScroll() {
    var y = window.pageYOffset || root.scrollTop;
    if (header) header.classList.toggle('is-scrolled', y > 40);
    if (toTop) toTop.classList.toggle('is-shown', y > 900);
    ticking = false;
  }
  on(window, 'scroll', function () { if (!ticking) { ticking = true; requestAnimationFrame(onScroll); } }, { passive: true });
  onScroll();
  on(toTop, 'click', function () { window.scrollTo({ top: 0, behavior: reduceMotion ? 'auto' : 'smooth' }); });

  /* ---------- Mega menus ---------- */
  var openMega = null;
  var megaTimer = null;
  function closeMega() {
    if (!openMega) return;
    openMega.panel.hidden = true;
    openMega.trigger.setAttribute('aria-expanded', 'false');
    header.classList.remove('is-mega-open');
    openMega = null;
  }
  function showMega(trigger) {
    var panel = $('[data-mega="' + trigger.getAttribute('data-mega-trigger') + '"]');
    if (!panel) return;
    if (openMega && openMega.panel === panel) return;
    closeMega();
    panel.hidden = false;
    trigger.setAttribute('aria-expanded', 'true');
    header.classList.add('is-mega-open');
    openMega = { panel: panel, trigger: trigger };
  }
  var hoverCapable = window.matchMedia && window.matchMedia('(hover: hover) and (pointer: fine)').matches;
  $$('[data-mega-trigger]').forEach(function (trigger) {
    var panel = $('[data-mega="' + trigger.getAttribute('data-mega-trigger') + '"]');
    if (hoverCapable) {
      on(trigger, 'mouseenter', function () { clearTimeout(megaTimer); megaTimer = setTimeout(function () { showMega(trigger); }, 90); });
      on(trigger, 'mouseleave', function () { clearTimeout(megaTimer); megaTimer = setTimeout(closeMega, 180); });
      on(panel, 'mouseenter', function () { clearTimeout(megaTimer); });
      on(panel, 'mouseleave', function () { clearTimeout(megaTimer); megaTimer = setTimeout(closeMega, 180); });
    } else {
      on(trigger, 'click', function (e) {
        if (trigger.getAttribute('aria-expanded') !== 'true') { e.preventDefault(); showMega(trigger); }
      });
    }
    on(trigger, 'keydown', function (e) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        showMega(trigger);
        var first = panel && panel.querySelector('a');
        if (first) first.focus();
      }
    });
  });
  on(doc, 'keydown', function (e) {
    if (e.key === 'Escape' && openMega) { var t = openMega.trigger; closeMega(); t.focus(); }
  });
  on(doc, 'click', function (e) {
    if (openMega && !header.contains(e.target)) closeMega();
  });
  on(doc, 'focusin', function (e) {
    if (openMega && !header.contains(e.target)) closeMega();
  });

  /* ---------- Mobile drawer ---------- */
  var drawer = $('[data-drawer]');
  var menuBtn = $('[data-menu-open]');
  function openDrawer() {
    if (!drawer) return;
    drawer.hidden = false;
    body.classList.add('is-locked');
    menuBtn && menuBtn.setAttribute('aria-expanded', 'true');
    var c = drawer.querySelector('[data-menu-close].icon-btn');
    c && c.focus();
  }
  function closeDrawer() {
    if (!drawer || drawer.hidden) return;
    drawer.hidden = true;
    body.classList.remove('is-locked');
    menuBtn && menuBtn.setAttribute('aria-expanded', 'false');
    menuBtn && menuBtn.focus();
  }
  on(menuBtn, 'click', openDrawer);
  $$('[data-menu-close]').forEach(function (b) { on(b, 'click', closeDrawer); });
  on(doc, 'keydown', function (e) { if (e.key === 'Escape') closeDrawer(); });
  on(window, 'resize', function () { if (window.innerWidth > 1220) closeDrawer(); });

  /* ---------- Search ---------- */
  var searchModal = $('[data-search]');
  var searchInput = $('[data-search-input]');
  var searchResults = $('[data-search-results]');
  var suggestHtml = searchResults ? searchResults.innerHTML : '';
  var index = null;
  var activeIdx = -1;

  function loadIndex() {
    if (index) return Promise.resolve(index);
    return fetch(BASE + '/search-index.json', { credentials: 'same-origin' })
      .then(function (r) { return r.json(); })
      .then(function (data) { index = data; return index; })
      .catch(function () { index = []; return index; });
  }
  function openSearch() {
    if (!searchModal) return;
    closeDrawer();
    searchModal.hidden = false;
    body.classList.add('is-locked');
    setTimeout(function () { searchInput && searchInput.focus(); }, 30);
    loadIndex();
  }
  function closeSearch() {
    if (!searchModal || searchModal.hidden) return;
    searchModal.hidden = true;
    body.classList.remove('is-locked');
  }
  $$('[data-search-open]').forEach(function (b) { on(b, 'click', openSearch); });
  $$('[data-search-close]').forEach(function (b) { on(b, 'click', closeSearch); });
  on(doc, 'keydown', function (e) {
    var tag = (e.target.tagName || '').toLowerCase();
    var typing = tag === 'input' || tag === 'textarea' || tag === 'select' || e.target.isContentEditable;
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') { e.preventDefault(); openSearch(); }
    else if (e.key === '/' && !typing) { e.preventDefault(); openSearch(); }
    else if (e.key === 'Escape') closeSearch();
  });

  function tokenize(q) { return q.toLowerCase().replace(/[^\w\s()-]/g, ' ').split(/\s+/).filter(function (t) { return t.length > 1; }); }
  function scoreItem(item, tokens) {
    var t = item.t.toLowerCase(), k = (item.k || '').toLowerCase(), d = (item.d || '').toLowerCase();
    var score = 0;
    for (var i = 0; i < tokens.length; i++) {
      var tok = tokens[i], s = 0;
      if (t.indexOf(tok) === 0) s += 14; else if ((' ' + t).indexOf(' ' + tok) > -1) s += 10; else if (t.indexOf(tok) > -1) s += 6;
      if (k.indexOf(tok) > -1) s += 4;
      if (d.indexOf(tok) > -1) s += 2;
      if (!s) return 0;
      score += s;
    }
    return score + (item.w || 0);
  }
  function highlight(text, tokens) {
    var out = escapeHtml(text);
    tokens.forEach(function (tok) {
      var re = new RegExp('(' + tok.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + ')', 'ig');
      out = out.replace(re, '<mark>$1</mark>');
    });
    return out;
  }
  function renderResults(q) {
    if (!searchResults) return;
    var tokens = tokenize(q);
    activeIdx = -1;
    if (!tokens.length) { searchResults.innerHTML = suggestHtml; return; }
    loadIndex().then(function (items) {
      var hits = items.map(function (it) { return { it: it, s: scoreItem(it, tokens) }; })
        .filter(function (h) { return h.s > 0; })
        .sort(function (a, b) { return b.s - a.s; })
        .slice(0, 9);
      if (!hits.length) {
        searchResults.innerHTML = '<div class="search-modal__empty">No results for “' + escapeHtml(q) + '”. Try another word, or <a href="' + BASE + '/contact">ask our team</a>.</div>';
        return;
      }
      searchResults.innerHTML = hits.map(function (h, i) {
        return '<a class="search-result" role="option" id="sr-' + i + '" href="' + escapeHtml(h.it.u) + '">' +
          '<span class="search-result__type">' + escapeHtml(h.it.y) + '</span>' +
          '<span class="search-result__body"><span class="search-result__title">' + highlight(h.it.t, tokens) + '</span>' +
          '<span class="search-result__text">' + escapeHtml(h.it.d || '') + '</span></span></a>';
      }).join('') + '<a class="search-result" href="' + BASE + '/search?q=' + encodeURIComponent(q) + '"><span class="search-result__type">All</span><span class="search-result__body"><span class="search-result__title">See all results for “' + escapeHtml(q) + '”</span></span></a>';
    });
  }
  var searchDebounce;
  on(searchInput, 'input', function () { clearTimeout(searchDebounce); var v = searchInput.value; searchDebounce = setTimeout(function () { renderResults(v); }, 80); });
  on(searchInput, 'keydown', function (e) {
    var links = $$('.search-result', searchResults);
    if (!links.length) return;
    if (e.key === 'ArrowDown' || e.key === 'ArrowUp') {
      e.preventDefault();
      activeIdx = e.key === 'ArrowDown' ? Math.min(activeIdx + 1, links.length - 1) : Math.max(activeIdx - 1, 0);
      links.forEach(function (l, i) { l.classList.toggle('is-active', i === activeIdx); });
      links[activeIdx].scrollIntoView({ block: 'nearest' });
    } else if (e.key === 'Enter' && activeIdx > -1) {
      e.preventDefault();
      window.location.href = links[activeIdx].href;
    }
  });

  /* ---------- Reveal on scroll & counters ---------- */
  function animateCount(el) {
    var target = parseFloat(el.getAttribute('data-count'));
    var decimals = (el.getAttribute('data-count').split('.')[1] || '').length;
    if (reduceMotion || isNaN(target)) { el.textContent = target.toLocaleString('en-US', { minimumFractionDigits: decimals }); return; }
    var start = null, dur = 1600;
    function step(ts) {
      if (!start) start = ts;
      var p = Math.min((ts - start) / dur, 1);
      var eased = 1 - Math.pow(1 - p, 3);
      el.textContent = (target * eased).toLocaleString('en-US', { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
      if (p < 1) requestAnimationFrame(step);
    }
    requestAnimationFrame(step);
  }
  var revealEls = $$('.reveal, [data-count], [data-animate]');
  if ('IntersectionObserver' in window) {
    var io = new IntersectionObserver(function (entries) {
      entries.forEach(function (en) {
        if (!en.isIntersecting) return;
        var el = en.target;
        el.classList.add('is-visible');
        if (el.hasAttribute('data-count')) animateCount(el);
        io.unobserve(el);
      });
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.08 });
    // Stagger siblings in grids.
    $$('.grid, .why, .steps, .offerings, .values, .feature-grid, .portal-grid, .gallery, .editions').forEach(function (g) {
      $$('.reveal', g).forEach(function (el, i) { el.style.setProperty('--delay', (i % 6) * 70 + 'ms'); });
    });
    revealEls.forEach(function (el) { io.observe(el); });
  } else {
    revealEls.forEach(function (el) { el.classList.add('is-visible'); if (el.hasAttribute('data-count')) animateCount(el); });
  }

  /* ---------- Forms ---------- */
  var EMAIL_RE = /^[^\s@]+@[^\s@]+\.[^\s@]{2,}$/;
  function setFieldError(input, msg) {
    var field = input.closest('.field, .check, .file-drop') || input.parentNode;
    field.classList.toggle('has-error', !!msg);
    var err = field.querySelector('.field__error');
    if (msg) {
      if (!err) { err = doc.createElement('span'); err.className = 'field__error'; field.appendChild(err); }
      err.textContent = msg;
      input.setAttribute('aria-invalid', 'true');
    } else {
      if (err) err.remove();
      input.removeAttribute('aria-invalid');
    }
  }
  function validateScope(scope) {
    var ok = true, firstBad = null;
    $$('input, select, textarea', scope).forEach(function (input) {
      if (input.type === 'hidden' || input.name === 'website' || input.disabled) return;
      var msg = '';
      var val = (input.value || '').trim();
      if (input.type === 'checkbox') {
        if (input.required && !input.checked) msg = 'Please tick this box to continue.';
      } else if (input.type === 'file') {
        if (input.required && !input.files.length) msg = 'Please attach your CV.';
        else if (input.files.length && input.files[0].size > 5 * 1024 * 1024) msg = 'The file is larger than 5 MB.';
      } else if (input.required && !val) {
        msg = 'This field is required.';
      } else if (val && input.type === 'email' && !EMAIL_RE.test(val)) {
        msg = 'Please enter a valid e-mail address.';
      } else if (val && input.type === 'tel' && val.replace(/\D/g, '').length < 7) {
        msg = 'Please enter a valid phone number.';
      }
      setFieldError(input, msg);
      if (msg) { ok = false; if (!firstBad) firstBad = input; }
    });
    // Required groups of checkboxes (data-required-group)
    $$('[data-required-group]', scope).forEach(function (group) {
      var checked = $$('input:checked', group).length > 0;
      var err = group.querySelector('.field__error');
      if (!checked) {
        ok = false;
        if (!err) { err = doc.createElement('span'); err.className = 'field__error'; group.appendChild(err); }
        err.textContent = group.getAttribute('data-required-group');
        if (!firstBad) firstBad = group.querySelector('input');
      } else if (err) err.remove();
    });
    if (firstBad) firstBad.focus({ preventScroll: false });
    return ok;
  }
  $$('.js-form').forEach(function (form) {
    form.setAttribute('novalidate', '');
    $$('input, select, textarea', form).forEach(function (input) {
      on(input, 'input', function () { if (input.getAttribute('aria-invalid')) setFieldError(input, ''); });
      on(input, 'change', function () { if (input.getAttribute('aria-invalid')) setFieldError(input, ''); });
    });
    on(form, 'submit', function (e) {
      e.preventDefault();
      var status = form.querySelector('.form__status');
      if (status) { status.className = 'form__status'; status.textContent = ''; }
      if (!validateScope(form)) return;
      var btn = form.querySelector('[type="submit"]');
      btn && btn.classList.add('is-loading');
      fetch(form.action, {
        method: 'POST',
        body: new FormData(form),
        headers: { 'X-Requested-With': 'fetch', 'Accept': 'application/json' },
        credentials: 'same-origin'
      }).then(function (r) {
        return r.json().catch(function () { return { ok: false, message: 'Something went wrong. Please try again, or e-mail us directly.' }; });
      }).then(function (res) {
        btn && btn.classList.remove('is-loading');
        if (res.ok) {
          if (form.classList.contains('newsletter')) {
            status.className = 'form__status is-success';
            status.textContent = res.message || 'Thank you for subscribing.';
            form.reset();
            return;
          }
          var wrap = form.closest('[data-form-wrap]') || form;
          wrap.innerHTML = '<div class="form-success" tabindex="-1"><div class="form-success__icon"><svg class="icon" aria-hidden="true"><use href="#i-check"/></svg></div>' +
            '<h3>' + escapeHtml(res.title || 'Thank you — message received.') + '</h3><p>' + escapeHtml(res.message || 'We will be in touch within one business day.') + '</p>' +
            '<a class="btn btn--ghost" href="' + BASE + '/knowledge-bank">Browse the Knowledge Bank</a></div>';
          var s = wrap.querySelector('.form-success');
          s && s.focus();
          if (window.gtag) window.gtag('event', 'generate_lead', { form: (form.querySelector('[name=form]') || {}).value });
        } else {
          if (res.errors) {
            Object.keys(res.errors).forEach(function (name) {
              var input = form.querySelector('[name="' + name + '"]');
              if (input) setFieldError(input, res.errors[name]);
            });
          }
          if (status) { status.className = 'form__status is-error'; status.textContent = res.message || 'Please check the form and try again.'; }
        }
      }).catch(function () {
        btn && btn.classList.remove('is-loading');
        if (status) { status.className = 'form__status is-error'; status.textContent = 'We could not send your message. Please check your connection and try again.'; }
      });
    });
  });

  /* File input label */
  $$('.file-drop').forEach(function (drop) {
    var input = drop.querySelector('input[type=file]');
    var label = drop.querySelector('[data-file-label]');
    on(input, 'change', function () { if (label && input.files.length) label.textContent = input.files[0].name; });
    ['dragenter', 'dragover'].forEach(function (ev) { on(drop, ev, function () { drop.classList.add('is-over'); }); });
    ['dragleave', 'drop'].forEach(function (ev) { on(drop, ev, function () { drop.classList.remove('is-over'); }); });
  });

  /* ---------- Wizard ---------- */
  $$('[data-wizard]').forEach(function (wiz) {
    var steps = $$('.wizard__step', wiz);
    var tabs = $$('.wizard__tab', wiz);
    var cur = 0;
    function show(i) {
      cur = i;
      steps.forEach(function (s, j) { s.classList.toggle('is-current', j === i); });
      tabs.forEach(function (t, j) { t.classList.toggle('is-active', j === i); t.classList.toggle('is-done', j < i); });
      var top = wiz.getBoundingClientRect().top + window.pageYOffset - 110;
      if (window.pageYOffset > top) window.scrollTo({ top: top, behavior: reduceMotion ? 'auto' : 'smooth' });
    }
    $$('[data-wizard-next]', wiz).forEach(function (b) {
      on(b, 'click', function () { if (validateScope(steps[cur])) show(Math.min(cur + 1, steps.length - 1)); });
    });
    $$('[data-wizard-prev]', wiz).forEach(function (b) { on(b, 'click', function () { show(Math.max(cur - 1, 0)); }); });
    show(0);
  });

  /* ---------- Knowledge bank filter ---------- */
  $$('[data-filter-list]').forEach(function (wrap) {
    var list = $(wrap.getAttribute('data-filter-list'));
    if (!list) return;
    var items = $$('[data-category]', list);
    var chips = $$('[data-filter]', wrap);
    var input = wrap.querySelector('[data-filter-search]');
    var empty = $(wrap.getAttribute('data-filter-empty'));
    var count = wrap.querySelector('[data-filter-count]');
    var active = 'all';
    var params = new URLSearchParams(window.location.search);
    if (params.get('cat')) active = params.get('cat');
    if (params.get('q') && input) input.value = params.get('q');
    function apply() {
      var q = input ? input.value.trim().toLowerCase() : '';
      var shown = 0;
      items.forEach(function (it) {
        var okCat = active === 'all' || it.getAttribute('data-category') === active;
        var okQ = !q || (it.getAttribute('data-search') || it.textContent.toLowerCase()).indexOf(q) > -1;
        var vis = okCat && okQ;
        it.hidden = !vis;
        if (vis) { shown++; it.classList.add('is-visible'); }
      });
      chips.forEach(function (c) { c.setAttribute('aria-pressed', c.getAttribute('data-filter') === active ? 'true' : 'false'); });
      if (empty) empty.hidden = shown > 0;
      if (count) count.textContent = shown + (shown === 1 ? ' article' : ' articles');
    }
    chips.forEach(function (c) { on(c, 'click', function () { active = c.getAttribute('data-filter'); apply(); }); });
    on(input, 'input', apply);
    apply();
  });

  /* Glossary filter */
  var gInput = $('[data-glossary-search]');
  on(gInput, 'input', function () {
    var q = gInput.value.trim().toLowerCase();
    var any = false;
    $$('.glossary-group').forEach(function (g) {
      var vis = 0;
      $$('[data-term]', g).forEach(function (t) {
        var ok = !q || t.textContent.toLowerCase().indexOf(q) > -1;
        t.hidden = !ok;
        if (ok) vis++;
      });
      g.hidden = vis === 0;
      if (vis) any = true;
    });
    var empty = $('[data-glossary-empty]');
    if (empty) empty.hidden = any;
  });

  /* ---------- Article: TOC, progress, copy, print ---------- */
  var prose = $('[data-prose]');
  if (prose) {
    var heads = $$('h2, h3', prose);
    var tocTargets = $$('[data-toc]');
    if (heads.length > 2 && tocTargets.length) {
      var html = '<ol>';
      heads.forEach(function (h, i) {
        if (!h.id) h.id = h.textContent.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '') || 'section-' + i;
        html += '<li' + (h.tagName === 'H3' ? ' class="toc__sub"' : '') + '><a href="#' + h.id + '">' + escapeHtml(h.textContent) + '</a></li>';
      });
      html += '</ol>';
      tocTargets.forEach(function (t) { t.innerHTML = html; });
      var tocLinks = $$('[data-toc] a');
      if ('IntersectionObserver' in window) {
        var tio = new IntersectionObserver(function (entries) {
          entries.forEach(function (en) {
            if (en.isIntersecting) {
              tocLinks.forEach(function (a) { a.classList.toggle('is-active', a.getAttribute('href') === '#' + en.target.id); });
            }
          });
        }, { rootMargin: '-20% 0px -70% 0px' });
        heads.forEach(function (h) { tio.observe(h); });
      }
    } else {
      $$('[data-toc-wrap]').forEach(function (w) { w.hidden = true; });
    }
    var bar = $('[data-progress]');
    if (bar) {
      var updateBar = function () {
        var rect = prose.getBoundingClientRect();
        var total = prose.offsetHeight - window.innerHeight * 0.6;
        var p = Math.min(Math.max(-rect.top / Math.max(total, 1), 0), 1);
        bar.style.transform = 'scaleX(' + p + ')';
      };
      on(window, 'scroll', function () { requestAnimationFrame(updateBar); }, { passive: true });
      updateBar();
    }
  }
  $$('.js-copy').forEach(function (b) {
    on(b, 'click', function () {
      var text = b.getAttribute('data-copy');
      if (navigator.clipboard) navigator.clipboard.writeText(text).then(function () { toast('Link copied'); }, function () { toast(text); });
      else toast(text);
    });
  });
  $$('.js-print').forEach(function (b) { on(b, 'click', function () { window.print(); }); });

  /* ---------- Tabs ---------- */
  $$('[data-tabs]').forEach(function (tabsEl) {
    var tabs = $$('[role="tab"]', tabsEl);
    function select(tab) {
      tabs.forEach(function (t) {
        var sel = t === tab;
        t.setAttribute('aria-selected', sel ? 'true' : 'false');
        t.tabIndex = sel ? 0 : -1;
        var panel = doc.getElementById(t.getAttribute('aria-controls'));
        if (panel) panel.hidden = !sel;
      });
    }
    tabs.forEach(function (t, i) {
      on(t, 'click', function () { select(t); });
      on(t, 'keydown', function (e) {
        var n = null;
        if (e.key === 'ArrowRight') n = tabs[(i + 1) % tabs.length];
        if (e.key === 'ArrowLeft') n = tabs[(i - 1 + tabs.length) % tabs.length];
        if (n) { e.preventDefault(); n.focus(); select(n); }
      });
    });
  });

  /* ---------- Lightbox ---------- */
  var shots = $$('[data-lightbox]');
  if (shots.length) {
    var lb = null, lbIdx = 0;
    function lbShow(i) {
      lbIdx = (i + shots.length) % shots.length;
      var s = shots[lbIdx];
      lb.querySelector('img').src = s.getAttribute('data-lightbox');
      lb.querySelector('img').alt = s.getAttribute('data-caption') || '';
      lb.querySelector('p').textContent = s.getAttribute('data-caption') || '';
    }
    function lbClose() { if (lb) { lb.remove(); lb = null; body.classList.remove('is-locked'); shots[lbIdx].focus(); } }
    shots.forEach(function (s, i) {
      on(s, 'click', function (e) {
        e.preventDefault();
        lb = doc.createElement('div');
        lb.className = 'lightbox';
        lb.setAttribute('role', 'dialog');
        lb.setAttribute('aria-modal', 'true');
        lb.innerHTML = '<button class="icon-btn lightbox__close" type="button" aria-label="Close"><svg class="icon"><use href="#i-x"/></svg></button>' +
          '<button class="icon-btn lightbox__nav lightbox__nav--prev" type="button" aria-label="Previous"><svg class="icon"><use href="#i-arrow-left"/></svg></button>' +
          '<img alt=""><p></p>' +
          '<button class="icon-btn lightbox__nav lightbox__nav--next" type="button" aria-label="Next"><svg class="icon"><use href="#i-arrow-right"/></svg></button>';
        body.appendChild(lb);
        body.classList.add('is-locked');
        lbShow(i);
        lb.querySelector('.lightbox__close').focus();
        on(lb, 'click', function (ev) { if (ev.target === lb) lbClose(); });
        on(lb.querySelector('.lightbox__close'), 'click', lbClose);
        on(lb.querySelector('.lightbox__nav--prev'), 'click', function () { lbShow(lbIdx - 1); });
        on(lb.querySelector('.lightbox__nav--next'), 'click', function () { lbShow(lbIdx + 1); });
      });
    });
    on(doc, 'keydown', function (e) {
      if (!lb) return;
      if (e.key === 'Escape') lbClose();
      if (e.key === 'ArrowRight') lbShow(lbIdx + 1);
      if (e.key === 'ArrowLeft') lbShow(lbIdx - 1);
    });
  }

  /* ---------- Number helpers ---------- */
  function fmt(n, dp) { return Number(n).toLocaleString('en-US', { minimumFractionDigits: dp || 0, maximumFractionDigits: dp || 0 }); }
  function rs(n) { return 'Rs ' + fmt(Math.round(n)); }
  function parseAmount(v) { var n = parseFloat(String(v).replace(/[^0-9.]/g, '')); return isNaN(n) ? 0 : n; }
  function moneyInput(input) {
    on(input, 'input', function () {
      var pos = input.value.length - input.selectionStart;
      var raw = input.value.replace(/[^0-9.]/g, '');
      var parts = raw.split('.');
      var intPart = parts[0] ? Number(parts[0]).toLocaleString('en-US') : '';
      input.value = parts.length > 1 ? intPart + '.' + parts[1].slice(0, 2) : intPart;
      var p = Math.max(input.value.length - pos, 0);
      try { input.setSelectionRange(p, p); } catch (e) { /* ignore */ }
    });
  }

  /* ---------- Salary tax calculator ---------- */
  var SLABS = {
    '2027': { label: 'Tax Year 2027 (2026-27)', surcharge: null, slabs: [
      [600000, 0, 0], [1200000, 0, 0.01], [2200000, 6000, 0.11], [3200000, 116000, 0.20],
      [4100000, 316000, 0.25], [5600000, 541000, 0.29], [7000000, 976000, 0.32], [Infinity, 1424000, 0.35]] },
    '2026': { label: 'Tax Year 2026 (2025-26)', surcharge: { above: 10000000, rate: 0.09 }, slabs: [
      [600000, 0, 0], [1200000, 0, 0.01], [2200000, 6000, 0.11], [3200000, 116000, 0.23],
      [4100000, 346000, 0.30], [Infinity, 616000, 0.35]] }
  };
  function salaryTax(annual, year) {
    var def = SLABS[year], prev = 0;
    for (var i = 0; i < def.slabs.length; i++) {
      var s = def.slabs[i];
      if (annual <= s[0]) {
        var tax = s[1] + (annual - prev) * s[2];
        var sur = def.surcharge && annual > def.surcharge.above ? tax * def.surcharge.rate : 0;
        return { tax: tax + sur, base: tax, surcharge: sur, slab: i, marginal: s[2] };
      }
      prev = s[0];
    }
    return { tax: 0, base: 0, surcharge: 0, slab: 0, marginal: 0 };
  }
  var calc = $('[data-salary-calc]');
  if (calc) {
    var inp = calc.querySelector('[name=salary]');
    moneyInput(inp);
    var slabTable = calc.querySelector('[data-slab-table]');
    function renderSlabs(year) {
      if (!slabTable) return;
      var def = SLABS[year], prev = 0, rows = '';
      def.slabs.forEach(function (s, i) {
        var range = s[0] === Infinity ? 'Above ' + fmt(prev) : (prev ? fmt(prev + 1) : '0') + ' – ' + fmt(s[0]);
        var rule = s[2] === 0 ? '0%' : (s[1] ? 'Rs ' + fmt(s[1]) + ' + ' : '') + Math.round(s[2] * 100) + '% above ' + fmt(prev);
        rows += '<tr data-slab="' + i + '"><td>' + range + '</td><td>' + rule + '</td></tr>';
        prev = s[0];
      });
      if (def.surcharge) rows += '<tr><td colspan="2" class="muted small">Plus a 9% surcharge on the tax where taxable income exceeds Rs 10,000,000.</td></tr>';
      slabTable.innerHTML = rows;
    }
    function run() {
      var year = (calc.querySelector('[name=year]:checked') || {}).value || '2027';
      var period = (calc.querySelector('[name=period]:checked') || {}).value || 'month';
      var amount = parseAmount(inp.value);
      var annual = period === 'month' ? amount * 12 : amount;
      var r = salaryTax(annual, year);
      var monthlyTax = r.tax / 12;
      var take = annual / 12 - monthlyTax;
      calc.querySelector('[data-out=monthly]').textContent = rs(monthlyTax);
      calc.querySelector('[data-out=annual]').textContent = rs(r.tax);
      calc.querySelector('[data-out=take]').textContent = rs(take);
      calc.querySelector('[data-out=effective]').textContent = annual ? fmt(r.tax / annual * 100, 2) + '%' : '0%';
      calc.querySelector('[data-out=marginal]').textContent = Math.round(r.marginal * 100) + '%' + (r.surcharge ? ' + surcharge' : '');
      calc.querySelector('[data-out=gross]').textContent = rs(annual / 12);
      var taxPct = annual ? r.tax / annual * 100 : 0;
      calc.querySelector('.split-bar__tax').style.width = taxPct + '%';
      calc.querySelector('.split-bar__take').style.width = (annual ? 100 - taxPct : 0) + '%';
      var other = year === '2027' ? '2026' : '2027';
      var o = salaryTax(annual, other);
      var diff = o.tax - r.tax;
      var cmp = calc.querySelector('[data-out=compare]');
      if (cmp) {
        if (!annual) cmp.textContent = '';
        else if (Math.abs(diff) < 1) cmp.textContent = 'Same tax as ' + SLABS[other].label + '.';
        else cmp.textContent = (diff > 0 ? 'Rs ' + fmt(diff) + ' a year less' : 'Rs ' + fmt(-diff) + ' a year more') + ' than under ' + SLABS[other].label + '.';
      }
      renderSlabs(year);
      var hit = slabTable && slabTable.querySelector('[data-slab="' + r.slab + '"]');
      if (hit && annual) hit.classList.add('is-hit');
    }
    $$('input', calc).forEach(function (el) { on(el, 'input', run); on(el, 'change', run); });
    run();
  }

  /* ---------- Sales tax calculator ---------- */
  var stc = $('[data-sales-calc]');
  if (stc) {
    var amt = stc.querySelector('[name=amount]');
    moneyInput(amt);
    var rateSel = stc.querySelector('[name=rate]');
    var custom = stc.querySelector('[name=custom_rate]');
    function runSt() {
      var a = parseAmount(amt.value);
      var rate = rateSel.value === 'custom' ? parseAmount(custom.value) : parseFloat(rateSel.value);
      custom.closest('.field').hidden = rateSel.value !== 'custom';
      var mode = (stc.querySelector('[name=mode]:checked') || {}).value || 'exclusive';
      var further = stc.querySelector('[name=further]').checked ? 4 : 0;
      var net = mode === 'exclusive' ? a : a / (1 + (rate + further) / 100);
      var tax = net * rate / 100, ft = net * further / 100;
      stc.querySelector('[data-out=net]').textContent = rs(net);
      stc.querySelector('[data-out=tax]').textContent = rs(tax);
      stc.querySelector('[data-out=further]').textContent = rs(ft);
      stc.querySelector('[data-out=total]').textContent = rs(net + tax + ft);
    }
    $$('input, select', stc).forEach(function (el) { on(el, 'input', runSt); on(el, 'change', runSt); });
    runSt();
  }

  /* ---------- Amount in words ---------- */
  var ONES = ['', 'One', 'Two', 'Three', 'Four', 'Five', 'Six', 'Seven', 'Eight', 'Nine', 'Ten', 'Eleven', 'Twelve', 'Thirteen', 'Fourteen', 'Fifteen', 'Sixteen', 'Seventeen', 'Eighteen', 'Nineteen'];
  var TENS = ['', '', 'Twenty', 'Thirty', 'Forty', 'Fifty', 'Sixty', 'Seventy', 'Eighty', 'Ninety'];
  function below100(n) { return n < 20 ? ONES[n] : TENS[Math.floor(n / 10)] + (n % 10 ? '-' + ONES[n % 10] : ''); }
  function below1000(n) { var h = Math.floor(n / 100), r = n % 100; return (h ? ONES[h] + ' Hundred' + (r ? ' ' : '') : '') + (r ? below100(r) : ''); }
  function wordsSouthAsian(n) {
    if (n === 0) return 'Zero';
    var parts = [], crore = Math.floor(n / 10000000);
    n %= 10000000;
    if (crore) parts.push((crore >= 1000 ? wordsSouthAsian(crore) : below1000(crore)) + ' Crore');
    var lakh = Math.floor(n / 100000); n %= 100000;
    if (lakh) parts.push(below100(lakh) + ' Lakh');
    var th = Math.floor(n / 1000); n %= 1000;
    if (th) parts.push(below100(th) + ' Thousand');
    if (n) parts.push(below1000(n));
    return parts.join(' ');
  }
  function wordsInternational(n) {
    if (n === 0) return 'Zero';
    var units = ['', ' Thousand', ' Million', ' Billion', ' Trillion'], parts = [], i = 0;
    while (n > 0 && i < units.length) { var c = n % 1000; if (c) parts.unshift(below1000(c) + units[i]); n = Math.floor(n / 1000); i++; }
    return parts.join(' ');
  }
  var aw = $('[data-words-calc]');
  if (aw) {
    var wIn = aw.querySelector('[name=words_amount]');
    moneyInput(wIn);
    var out = aw.querySelector('[data-out=words]');
    function runWords() {
      var v = parseAmount(wIn.value);
      if (!wIn.value.trim()) { out.textContent = 'Type an amount to see it in words.'; return; }
      if (v >= 1e15) { out.textContent = 'Please enter a smaller amount.'; return; }
      var sys = (aw.querySelector('[name=system]:checked') || {}).value || 'pk';
      var rupees = Math.floor(v), paisa = Math.round((v - rupees) * 100);
      var w = sys === 'pk' ? wordsSouthAsian(rupees) : wordsInternational(rupees);
      out.textContent = 'Rupees ' + w + (paisa ? ' and ' + below100(paisa) + ' Paisa' : '') + ' Only';
    }
    $$('input', aw).forEach(function (el) { on(el, 'input', runWords); on(el, 'change', runWords); });
    var copyW = aw.querySelector('[data-copy-words]');
    on(copyW, 'click', function () { if (navigator.clipboard) navigator.clipboard.writeText(out.textContent).then(function () { toast('Copied'); }); });
    runWords();
  }

  /* ---------- Deadlines: live countdowns & calendar files ---------- */
  function daysUntil(ymd) {
    var p = ymd.split('-');
    var due = new Date(+p[0], +p[1] - 1, +p[2]);
    var now = new Date();
    var today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    return Math.round((due - today) / 86400000);
  }
  $$('[data-due]').forEach(function (el) {
    var d = daysUntil(el.getAttribute('data-due'));
    var label = el.querySelector('.countdown') || (el.classList.contains('countdown') ? el : null);
    if (d < 0) { var row = el.closest('[data-deadline-row]'); if (row) row.hidden = true; return; }
    if (label) {
      label.textContent = d === 0 ? 'Due today' : d === 1 ? 'Tomorrow' : 'In ' + d + ' days';
      label.classList.toggle('is-today', d === 0);
      label.classList.toggle('is-soon', d > 0 && d <= 7);
    }
  });
  $$('[data-cal-filter]').forEach(function (wrap) {
    var chips = $$('[data-filter]', wrap);
    chips.forEach(function (c) {
      on(c, 'click', function () {
        var f = c.getAttribute('data-filter');
        chips.forEach(function (x) { x.setAttribute('aria-pressed', x === c ? 'true' : 'false'); });
        $$('[data-deadline-row]').forEach(function (row) {
          if (row.getAttribute('data-past') === '1') return;
          row.hidden = !(f === 'all' || row.getAttribute('data-type') === f);
        });
        $$('[data-month-group]').forEach(function (g) {
          g.hidden = !$$('[data-deadline-row]', g).some(function (r) { return !r.hidden; });
        });
      });
    });
  });
  function icsEscape(s) { return String(s).replace(/\\/g, '\\\\').replace(/;/g, '\\;').replace(/,/g, '\\,').replace(/\n/g, '\\n'); }
  function icsDate(ymd, add) { var p = ymd.split('-'); var d = new Date(Date.UTC(+p[0], +p[1] - 1, +p[2] + (add || 0))); return d.toISOString().slice(0, 10).replace(/-/g, ''); }
  $$('[data-ics]').forEach(function (b) {
    on(b, 'click', function () {
      var date = b.getAttribute('data-ics');
      var title = b.getAttribute('data-ics-title');
      var desc = (b.getAttribute('data-ics-desc') || '') + '\n\nFrom the Veridian Partners tax calendar: ' + window.location.href;
      var stamp = new Date().toISOString().replace(/[-:]/g, '').replace(/\.\d+/, '');
      var ics = ['BEGIN:VCALENDAR', 'VERSION:2.0', 'PRODID:-//Veridian Partners//Tax Calendar//EN', 'CALSCALE:GREGORIAN', 'METHOD:PUBLISH',
        'BEGIN:VEVENT', 'UID:' + date + '-' + title.replace(/\W+/g, '').toLowerCase() + '@veridian', 'DTSTAMP:' + stamp,
        'DTSTART;VALUE=DATE:' + icsDate(date), 'DTEND;VALUE=DATE:' + icsDate(date, 1),
        'SUMMARY:' + icsEscape(title), 'DESCRIPTION:' + icsEscape(desc),
        'BEGIN:VALARM', 'TRIGGER:-P3D', 'ACTION:DISPLAY', 'DESCRIPTION:' + icsEscape(title + ' in 3 days'), 'END:VALARM',
        'END:VEVENT', 'END:VCALENDAR'].join('\r\n');
      var blob = new Blob([ics], { type: 'text/calendar;charset=utf-8' });
      var a = doc.createElement('a');
      a.href = URL.createObjectURL(blob);
      a.download = title.replace(/[^\w]+/g, '-').toLowerCase() + '-' + date + '.ics';
      doc.body.appendChild(a); a.click(); a.remove();
      setTimeout(function () { URL.revokeObjectURL(a.href); }, 1000);
      toast('Calendar file downloaded');
    });
  });

  /* ---------- Cookie consent & analytics ---------- */
  var gaId = body.getAttribute('data-ga');
  function loadGA() {
    if (!gaId || window.gtag) return;
    var s = doc.createElement('script');
    s.async = true;
    s.src = 'https://www.googletagmanager.com/gtag/js?id=' + encodeURIComponent(gaId);
    doc.head.appendChild(s);
    window.dataLayer = window.dataLayer || [];
    window.gtag = function () { window.dataLayer.push(arguments); };
    window.gtag('js', new Date());
    window.gtag('config', gaId, { anonymize_ip: true });
  }
  var cookieBox = $('[data-cookie]');
  if (gaId) {
    var choice = store('vp-consent');
    if (choice === 'accept') loadGA();
    else if (!choice && cookieBox) cookieBox.hidden = false;
    $$('[data-cookie-choice]').forEach(function (b) {
      on(b, 'click', function () {
        var c = b.getAttribute('data-cookie-choice');
        store('vp-consent', c);
        cookieBox.hidden = true;
        if (c === 'accept') loadGA();
      });
    });
  }
  $$('[data-cookie-reset]').forEach(function (b) {
    on(b, 'click', function () { try { localStorage.removeItem('vp-consent'); } catch (e) { /* ignore */ } if (cookieBox) cookieBox.hidden = false; toast('Cookie preferences reset'); });
  });

  /* ---------- Preselect from query string (?service=) ---------- */
  var qs = new URLSearchParams(window.location.search);
  if (qs.get('service')) {
    $$('[data-preselect]').forEach(function (group) {
      $$('input', group).forEach(function (i) { if (i.value === qs.get('service') || i.getAttribute('data-slug') === qs.get('service')) i.checked = true; });
    });
  }
})();
