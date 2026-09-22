/*
 * Homepage search wiring. Mintlify runs every .js file on every page, so this
 * binds only when the homepage search button exists.
 *
 * The button opens Mintlify's own search modal by forwarding the click to the
 * navbar search trigger (#search-bar-entry is a documented Mintlify hook),
 * with a synthesized Cmd/Ctrl-K as the fallback.
 */
(function () {
  function openSearch() {
    var entry =
      document.getElementById('search-bar-entry') ||
      document.getElementById('search-bar-entry-mobile');
    if (entry) {
      entry.click();
      return;
    }
    var mac = /Mac|iPhone|iPad/.test(navigator.platform);
    document.dispatchEvent(
      new KeyboardEvent('keydown', {
        key: 'k',
        metaKey: mac,
        ctrlKey: !mac,
        bubbles: true,
      })
    );
  }

  function bind() {
    var button = document.getElementById('home-search');
    if (button && !button.dataset.searchBound) {
      button.dataset.searchBound = 'true';
      button.addEventListener('click', openSearch);
    }
  }

  bind();
  // Mintlify is a client-side app: the homepage can mount after this script
  // runs, or on navigation back to it. Watch for the button (re)appearing.
  new MutationObserver(bind).observe(document.body, {
    childList: true,
    subtree: true,
  });
})();
