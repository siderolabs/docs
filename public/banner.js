(function () {
  function shouldCheckPath() {
    const path = window.location.pathname;

    return (
      path.startsWith("/omni/") ||
      path.startsWith("/kubernetes-guides/") ||
      path.startsWith("/talos/v1.12/")
    );
  }

  function deleteVersionBannerIfNeeded() {
    if (!shouldCheckPath()) return;

    const banner = document.getElementById("banner");
    if (!banner) return;

    const isVersionBanner =
      banner.textContent &&
      banner.textContent.includes(
        "You are viewing an older version of the docs."
      );

    if (isVersionBanner) {
      banner.remove();
    }
  }

  deleteVersionBannerIfNeeded();

  // Run after hydration
  window.addEventListener("load", deleteVersionBannerIfNeeded);

  // Handle SPA navigation (Next.js)
  const pushState = history.pushState;
  history.pushState = function () {
    pushState.apply(this, arguments);
    setTimeout(deleteVersionBannerIfNeeded, 0);
  };

  window.addEventListener("popstate", () => {
    setTimeout(deleteVersionBannerIfNeeded, 0);
  });

  // Watch for React re-mounting the banner
  const observer = new MutationObserver(deleteVersionBannerIfNeeded);
  observer.observe(document.body, {
    childList: true,
    subtree: true,
  });
})();

