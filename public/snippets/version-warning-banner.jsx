export const VersionWarningBanner = () => {

  // MOVE latestVersion inside component
  const latestVersion = "v1.12";

  const [latestUrl, setLatestUrl] = useState(null);
  const [currentVersion, setCurrentVersion] = useState(null);

  useEffect(() => {
    if (typeof window === "undefined") return;

    const { pathname, hash, search } = window.location;

    const match = pathname.match(/\/talos\/(v\d+\.\d+)\//);

    if (!match) return;

    const detectedVersion = match[1];

    if (detectedVersion === latestVersion) return;

    setCurrentVersion(detectedVersion);

    const newPath = pathname.replace(
      `/talos/${detectedVersion}/`,
      `/talos/${latestVersion}/`
    );

    setLatestUrl(`${newPath}${search}${hash}`);

  }, []);

  if (!latestUrl || !currentVersion) return null;

  return (
    <div className="not-prose sticky top-6 z-50 my-6">
      <div className="border border-red-500/30 bg-red-500/10 px-4 py-3 rounded-xl">

        <div className="text-sm">
          ⚠️ You are viewing an older version of Talos ({currentVersion}).
          <a
            href={latestUrl}
            className="ml-2 underline text-red-400 hover:text-red-300 font-medium"
          >
            View the latest version {latestVersion} →
          </a>
        </div>

      </div>
    </div>
  );
};
