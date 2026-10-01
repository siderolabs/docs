(function() {
  // Mintlify only runs analytics (PostHog, GA4) when localStorage holds the
  // integrations.cookies key/value from common.yaml. Mirror the CookieYes
  // "Analytics" choice into it.
  var KEY = 'sidero-analytics-consent';

  // Returns true when analytics has to start or stop.
  function setConsent(granted) {
    var previous = localStorage.getItem(KEY);
    var value = granted ? 'granted' : 'denied';
    if (previous === value) return false;
    localStorage.setItem(KEY, value);
    return granted || previous === 'granted';
  }

  try {
    // Apply the saved choice. Mintlify may already have read a stale "granted"
    // (e.g. the CookieYes cookie expired), so reload if consent was withdrawn.
    var saved = /(?:^|;\s*)cookieyes-consent=[^;]*analytics:yes/.test(document.cookie);
    if (setConsent(saved) && !saved) window.location.reload();

    // Mintlify reads the value on page load, so reload when the choice changes.
    var onChange = function(granted) {
      if (setConsent(granted)) window.location.reload();
    };
    document.addEventListener('cookieyes_banner_load', function(event) {
      onChange(!!(event.detail && event.detail.categories && event.detail.categories.analytics));
    });
    document.addEventListener('cookieyes_consent_update', function(event) {
      onChange(!!(event.detail && event.detail.accepted && event.detail.accepted.indexOf('analytics') !== -1));
    });
  } catch (e) {
    // localStorage can be blocked; without it Mintlify keeps analytics off.
  }

  var script = document.createElement('script');
  script.id = 'cookieyes';
  script.type = 'text/javascript';
  script.src = 'https://cdn-cookieyes.com/client_data/ba228474d21603054ee6972b5faefc8d/script.js';
  document.head.insertBefore(script, document.head.firstChild);
})();
