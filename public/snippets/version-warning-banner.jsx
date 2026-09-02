export const VersionWarningBanner = () => {
  const latestVersion = "v1.14";

  const [latestUrl, setLatestUrl] = useState(null);
  const [currentVersion, setCurrentVersion] = useState(null);
  const [isBeta, setIsBeta] = useState(false);

  // helper: convert "v1.12" → [1, 12]
  const parseVersion = (v) => v.replace("v", "").split(".").map(Number);

  // helper: compare versions
  const isGreaterVersion = (a, b) => {
    const [aMajor, aMinor] = parseVersion(a);
    const [bMajor, bMinor] = parseVersion(b);

    if (aMajor > bMajor) return true;
    if (aMajor === bMajor && aMinor > bMinor) return true;
    return false;
  };

  useEffect(() => {
    if (typeof window === "undefined") return;

    const { pathname, hash, search } = window.location;
    const match = pathname.match(/\/talos\/(v\d+\.\d+)\//);

    if (!match) return;

    const detectedVersion = match[1];

    if (detectedVersion === latestVersion) return;

    setCurrentVersion(detectedVersion);

    // check if it's newer (beta)
    if (isGreaterVersion(detectedVersion, latestVersion)) {
      setIsBeta(true);
    }

    const newPath = pathname.replace(
      `/talos/${detectedVersion}/`,
      `/talos/${latestVersion}/`
    );

    setLatestUrl(`${newPath}${search}${hash}`);
  }, []);

  if (!latestUrl || !currentVersion) return null;

  return (
    <div className="not-prose sticky top-6 z-50 my-6">
      <div className="border border-yellow-500/30 bg-yellow-500/10 px-4 py-3 rounded-xl">
        <div className="text-sm">
          {isBeta ? (
            <>
              ⚠️ You are viewing a <strong>beta version</strong> of Talos ({currentVersion}).
              This version may be unstable.
              <a
                href={latestUrl}
                className="ml-2 underline text-yellow-400 hover:text-yellow-300 font-medium"
              >
                View latest stable version {latestVersion} →
              </a>
            </>
          ) : (
            <>
              ⚠️ You are viewing an older version of Talos ({currentVersion}).
              <a
                href={latestUrl}
                className="ml-2 underline text-yellow-400 hover:text-yellow-300 font-medium"
              >
                View the latest version {latestVersion} →
              </a>
            </>
          )}
        </div>
      </div>
    </div>
  );
};
