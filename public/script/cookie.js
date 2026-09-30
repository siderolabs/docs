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

  // Apply the saved choice before Mintlify reads it.
  setConsent(/(?:^|;\s*)cookieyes-consent=[^;]*analytics:yes/.test(document.cookie));

  // Mintlify reads the value on page load, so reload when the choice changes.
  function onChange(granted) {
    if (setConsent(granted)) window.location.reload();
  }
  document.addEventListener('cookieyes_banner_load', function(event) {
    onChange(!!(event.detail && event.detail.categories && event.detail.categories.analytics));
  });
  document.addEventListener('cookieyes_consent_update', function(event) {
    onChange(!!(event.detail && event.detail.accepted && event.detail.accepted.indexOf('analytics') !== -1));
  });

  var script = document.createElement('script');
  script.id = 'cookieyes';
  script.type = 'text/javascript';
  script.src = 'https://cdn-cookieyes.com/client_data/ba228474d21603054ee6972b5faefc8d/script.js';
  document.head.insertBefore(script, document.head.firstChild);
})();
