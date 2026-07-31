{/*
  ConfigEvolution — interactive map of the v1alpha1 machine config against the
  multi-document config, pinned to the dataset's docs version.

  The component takes a generated dataset, e.g.

    import { ConfigEvolution } from "/snippets/config-evolution.jsx"
    import { CONFIG_EVOLUTION } from "/snippets/config-evolution-data-v1.14.jsx"

    <ConfigEvolution data={CONFIG_EVOLUTION} />

  The dataset is capped at its own docs version (data.version), so a page under
  /talos/v1.14/ never advertises a document that does not exist in 1.14. Clicking a
  document links straight to its reference page; clicking a v1alpha1 field opens an
  inline detail panel with the replacement document(s), since there is nowhere else
  for that information to link to.

  Styling note: the styles are scoped CSS shipped with the component rather than
  Tailwind classes. Mintlify rewrites Tailwind utilities only inside static
  className literals, so any class assembled in JavaScript — which is most of an
  interactive component — would silently resolve to nothing.
*/}

export const ConfigEvolution = ({ data }) => {
  const CSS = `
.cfg-root{
  --cfg-bg:#fff; --cfg-panel:#f8fafc; --cfg-border:#e4e4e7;
  --cfg-text:#27272a; --cfg-muted:#71717a; --cfg-accent:#e02a5f;
  --cfg-keep:#059669; --cfg-moved:#e11d48; --cfg-inherit:#d97706;
  --cfg-drop:#a1a1aa; --cfg-new:#7c3aed; --cfg-link:#0284c7;
  font-size:13px; line-height:1.55; color:var(--cfg-text); margin:1.5rem 0;
}
.dark .cfg-root{
  --cfg-bg:#18181b; --cfg-panel:#27272a; --cfg-border:#3f3f46;
  --cfg-text:#e4e4e7; --cfg-muted:#a1a1aa; --cfg-accent:#fb326e;
  --cfg-keep:#34d399; --cfg-moved:#fb7185; --cfg-inherit:#fbbf24;
  --cfg-drop:#a1a1aa; --cfg-new:#a78bfa; --cfg-link:#38bdf8;
}
.cfg-root code{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
.cfg-card{border:1px solid var(--cfg-border);border-radius:12px;background:var(--cfg-bg);padding:14px}
.cfg-chip{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--cfg-border);
  border-radius:8px;padding:4px 10px;font-size:12px;background:var(--cfg-bg);color:var(--cfg-text);cursor:pointer}
.cfg-chip:hover{border-color:var(--cfg-muted)}
.cfg-tools{display:flex;flex-wrap:wrap;align-items:center;gap:8px}
.cfg-search,.cfg-select{border:1px solid var(--cfg-border);border-radius:8px;padding:6px 10px;
  background:var(--cfg-bg);color:var(--cfg-text);font-size:13px}
.cfg-search{flex:1 1 220px;min-width:170px}
.cfg-search:focus,.cfg-select:focus{outline:none;border-color:var(--cfg-link)}
.cfg-panes{display:grid;gap:14px;margin-top:16px}
@media(min-width:1024px){.cfg-panes{grid-template-columns:1fr 1fr}}
.cfg-pane{border:1px solid var(--cfg-border);border-radius:12px;background:var(--cfg-bg);overflow:hidden}
.cfg-panehead{display:flex;justify-content:space-between;align-items:baseline;gap:8px;
  padding:8px 12px;border-bottom:1px solid var(--cfg-border);background:var(--cfg-panel);
  font-size:11px;font-weight:600;letter-spacing:.5px;text-transform:uppercase;color:var(--cfg-muted)}
.cfg-panehead .cfg-sub{font-weight:400;letter-spacing:0;text-transform:none}
.cfg-panebody{max-height:60vh;overflow:auto;padding:8px}
.cfg-sect{margin-bottom:10px}
.cfg-shead{padding:4px 6px;font-family:ui-monospace,monospace;font-size:12px;font-weight:700;
  color:var(--cfg-accent);cursor:pointer}
.cfg-row{display:flex;align-items:flex-start;gap:6px;padding:3px 6px;border-radius:6px;cursor:pointer}
.cfg-row:hover{background:var(--cfg-panel)}
.cfg-row.isnew{background:rgba(124,58,237,.10)}
.cfg-row.linked{background:rgba(2,132,199,.14)}
.cfg-row.sel{background:rgba(224,42,95,.10);box-shadow:inset 0 0 0 1px var(--cfg-accent)}
.cfg-bar{flex:0 0 3px;width:3px;min-height:16px;align-self:stretch;border-radius:2px}
.cfg-bar.keep{background:var(--cfg-keep)}
.cfg-bar.moved{background:var(--cfg-moved)}
.cfg-bar.inherited{background:var(--cfg-inherit)}
.cfg-bar.dropped{background:var(--cfg-drop)}
.cfg-caret{flex:0 0 12px;width:12px;font-size:10px;color:var(--cfg-muted)}
.cfg-label{display:flex;flex-wrap:wrap;align-items:center;gap:2px 6px}
.cfg-fname{font-family:ui-monospace,monospace;font-size:12.5px;word-break:break-all}
.cfg-fname.gone{color:var(--cfg-muted);text-decoration:line-through}
.cfg-fname.dim{color:var(--cfg-muted)}
.cfg-to{font-family:ui-monospace,monospace;font-size:11px;color:var(--cfg-link)}
.cfg-to.dim{color:var(--cfg-muted)}
.cfg-badge{border:1px solid var(--cfg-border);border-radius:20px;padding:0 6px;font-size:10px;
  color:var(--cfg-muted);white-space:nowrap}
.cfg-badge.new{border-color:var(--cfg-new);color:var(--cfg-new)}
.cfg-kids{margin-left:12px;padding-left:4px;border-left:1px dashed var(--cfg-border)}
.cfg-dgroup{margin-bottom:14px}
.cfg-dhead{margin-bottom:6px;padding:0 4px;font-size:10px;font-weight:600;letter-spacing:.6px;
  text-transform:uppercase;color:var(--cfg-muted)}
.cfg-dgrid{display:grid;gap:8px;grid-template-columns:repeat(auto-fill,minmax(160px,1fr))}
.cfg-doc{display:block;border:1px solid var(--cfg-border);border-left:3px solid var(--cfg-link);
  border-radius:8px;padding:6px 8px;background:var(--cfg-panel);color:inherit;text-decoration:none}
.cfg-doc:hover{border-color:var(--cfg-link)}
.cfg-doc.isnew{border-left-color:var(--cfg-new);background:rgba(124,58,237,.08)}
.cfg-doc.linked{box-shadow:inset 0 0 0 1px var(--cfg-link);background:rgba(2,132,199,.12)}
.cfg-dk{font-family:ui-monospace,monospace;font-size:11.5px;font-weight:600;word-break:break-all}
.cfg-dm{display:flex;flex-wrap:wrap;gap:6px;margin-top:3px;font-size:10px;color:var(--cfg-muted)}
.cfg-dm .new{color:var(--cfg-new)}
.cfg-detail{margin-top:14px;padding:10px 12px;border:1px solid var(--cfg-border);border-radius:12px;
  background:var(--cfg-panel)}
.cfg-drow{display:flex;justify-content:space-between;align-items:baseline;gap:12px;flex-wrap:wrap}
.cfg-title{font-family:ui-monospace,monospace;font-size:14px;font-weight:600;word-break:break-all}
.cfg-meta{font-size:11.5px;color:var(--cfg-muted)}
.cfg-x{margin-left:auto;background:none;border:none;font-size:18px;line-height:1;
  color:var(--cfg-muted);cursor:pointer}
.cfg-verdict{margin-top:6px}
.cfg-verdict.keep{color:var(--cfg-keep)}
.cfg-verdict.moved{color:var(--cfg-moved)}
.cfg-verdict.inherited{color:var(--cfg-inherit)}
.cfg-verdict.dropped{color:var(--cfg-drop)}
.cfg-p{margin-top:6px}
.cfg-note{margin-top:6px;padding-left:10px;border-left:2px solid var(--cfg-moved);
  font-style:italic;color:var(--cfg-muted)}
.cfg-sec{margin-top:9px}
.cfg-dtitle{display:flex;align-items:baseline;gap:10px;flex-wrap:wrap}
.cfg-h{margin-bottom:5px;font-size:10px;font-weight:600;letter-spacing:.6px;text-transform:uppercase;
  color:var(--cfg-muted)}
.cfg-pill{display:inline-flex;align-items:center;margin:0 6px 6px 0;padding:2px 9px;
  border:1px solid var(--cfg-border);border-radius:20px;font-family:ui-monospace,monospace;
  font-size:11px;color:var(--cfg-link);text-decoration:none}
.cfg-pill:hover{border-color:var(--cfg-link);background:rgba(2,132,199,.1)}
.cfg-pill .since{margin-left:4px;opacity:.7}
.cfg-empty{padding:24px;text-align:center;color:var(--cfg-muted)}
.cfg-foot{margin-top:12px;font-size:12px;color:var(--cfg-muted)}
.cfg-link{color:var(--cfg-accent);text-decoration:none;font-weight:500}
.cfg-link:hover{text-decoration:underline}
`;

  const GROUP_LABEL = {
    network: "Networking",
    kubernetes: "Kubernetes",
    block: "Block and volumes",
    storage: "Storage (LVM/RAID)",
    cri: "CRI and registries",
    runtime: "Runtime and OS",
    cluster: "Cluster and discovery",
    security: "Security",
    hardware: "Hardware",
    extensions: "Extensions",
    siderolink: "SideroLink",
  };
  const GROUP_ORDER = [
    "network", "kubernetes", "block", "storage", "cri",
    "runtime", "cluster", "security", "hardware", "extensions", "siderolink",
  ];

  const docs = data.docs;
  const tree = data.tree;
  const version = data.version;

  const [query, setQuery] = useState("");
  const [group, setGroup] = useState("");
  const [collapsed, setCollapsed] = useState({});
  const [selField, setSelField] = useState(null);

  // The detail panel sits under the two panes, so nudge it into view when a
  // selection is made rather than leaving it below the fold.
  useEffect(() => {
    if (!selField || typeof document === "undefined") return;
    const el = document.getElementById("cfg-detail");
    if (el && el.scrollIntoView) el.scrollIntoView({ block: "nearest", behavior: "smooth" });
  }, [selField]);

  const vnum = (v) => {
    const p = String(v).replace("v", "").split(".");
    return Number(p[0]) * 1000 + Number(p[1]);
  };
  const limit = vnum(version);

  const docByKind = {};
  docs.forEach((d) => {
    docByKind[d.kind] = d;
  });

  // Flatten the v1alpha1 tree and remember each node's parent, so a leaf can tell
  // whether an ancestor block was moved out from under it.
  const fields = [];
  const parentOf = {};
  const collect = (nodes, parent) => {
    (nodes || []).forEach((n) => {
      fields.push(n);
      parentOf[n.path] = parent;
      collect(n.children, n);
    });
  };
  collect(tree.machine, null);
  collect(tree.cluster, null);

  // Replacement documents that actually exist at the selected release. A field
  // deprecated in favour of a document that has not shipped yet is still the only
  // way to configure that setting, so it counts as "keep".
  const replacements = (n) =>
    (n.replacedBy || []).filter((k) => docByKind[k] && vnum(docByKind[k].since) <= limit);

  const ownStatus = (n) => {
    if (!n.deprecated) return "keep";
    if (replacements(n).length) return "moved";
    if (!(n.replacedBy || []).length) return "dropped";
    return "keep";
  };

  const movedAncestor = (n) => {
    let p = parentOf[n.path];
    while (p) {
      if (ownStatus(p) === "moved") return p;
      p = parentOf[p.path];
    }
    return null;
  };

  const statusOf = (n) => {
    const own = ownStatus(n);
    if (own !== "keep") return own;
    return movedAncestor(n) ? "inherited" : "keep";
  };

  // Did this field move out exactly at the selected release?
  const fieldIsNew = (n) => {
    const since = replacements(n).map((k) => vnum(docByKind[k].since));
    if (!since.length) return false;
    return Math.max.apply(null, since) === limit && Math.min.apply(null, since) === limit;
  };

  const fieldsReplacedBy = (kind) => fields.filter((n) => (n.replacedBy || []).indexOf(kind) !== -1);
  const refHref = (d) => data.refBase + "/" + d.group + "/" + d.kind.toLowerCase();

  const q = query.trim().toLowerCase();
  const hit = (s) => !q || String(s || "").toLowerCase().indexOf(q) !== -1;

  const fieldMatches = (n) => {
    if (!hit(n.path + " " + (n.desc || "") + " " + (n.replacedBy || []).join(" "))) return false;
    if (group && !replacements(n).some((k) => docByKind[k].group === group)) return false;
    return true;
  };
  const subtreeMatches = (n) => fieldMatches(n) || (n.children || []).some(subtreeMatches);

  const docMatches = (d) => {
    if (!hit(d.kind + " " + (d.desc || ""))) return false;
    if (group && d.group !== group) return false;
    return true;
  };

  // Selecting a v1alpha1 field cross-highlights the documents that replace it.
  const linkedDocs = {};
  if (selField) {
    const n = fields.filter((f) => f.path === selField)[0];
    if (n) replacements(n).forEach((k) => (linkedDocs[k] = true));
  }

  const selectField = (n) => setSelField(n.path);

  const toggle = (key) => {
    const next = Object.assign({}, collapsed);
    if (collapsed[key]) delete next[key];
    else next[key] = true;
    setCollapsed(next);
  };

  const renderRow = (n) => {
    const status = statusOf(n);
    const kids = (n.children || []).filter(subtreeMatches);
    const open = !collapsed[n.path];
    const isNew = fieldIsNew(n);
    const selected = selField === n.path;
    const repl = replacements(n);
    const parent = status === "inherited" ? movedAncestor(n) : null;
    const parentRepl = parent ? replacements(parent) : [];

    let rowClass = "cfg-row";
    if (selected) rowClass += " sel";
    else if (isNew) rowClass += " isnew";

    let nameClass = "cfg-fname";
    if (status === "moved" || status === "dropped") nameClass += " gone";
    else if (status === "inherited") nameClass += " dim";

    return (
      <div key={n.path}>
        <div className={rowClass} onClick={() => selectField(n)}>
          <span className={"cfg-bar " + status} />
          <span
            className="cfg-caret"
            onClick={(e) => {
              if (!kids.length) return;
              e.stopPropagation();
              toggle(n.path);
            }}
          >
            {kids.length ? (open ? "▾" : "▸") : ""}
          </span>
          <span className="cfg-label">
            <code className={nameClass}>{n.field}</code>
            {status === "moved" && repl.length ? (
              <span className="cfg-to">
                → {repl.slice(0, 2).join(", ")}
                {repl.length > 2 ? " +" + (repl.length - 2) : ""}
              </span>
            ) : null}
            {status === "inherited" && parentRepl.length ? (
              <span className="cfg-to dim">→ {parentRepl[0]}</span>
            ) : null}
            {status === "dropped" ? <span className="cfg-badge">removed</span> : null}
            {isNew ? <span className="cfg-badge new">new in {version}</span> : null}
          </span>
        </div>
        {kids.length && open ? <div className="cfg-kids">{kids.map(renderRow)}</div> : null}
      </div>
    );
  };

  const renderSection = (key, label) => {
    const nodes = (tree[key] || []).filter(subtreeMatches);
    if (!nodes.length) return null;
    const open = !collapsed[key];
    return (
      <div key={key} className="cfg-sect">
        <div className="cfg-shead" onClick={() => toggle(key)}>
          {open ? "▾" : "▸"} {label}
        </div>
        {open ? <div>{nodes.map(renderRow)}</div> : null}
      </div>
    );
  };

  const renderDoc = (d) => {
    const isNew = d.since === version;
    const replaces = fieldsReplacedBy(d.kind).length;

    let cls = "cfg-doc";
    if (linkedDocs[d.kind]) cls += " linked";
    else if (isNew) cls += " isnew";

    return (
      <a key={d.kind} className={cls} href={refHref(d)} title={"Open the " + d.kind + " reference"}>
        <div className="cfg-dk">{d.kind}</div>
        <div className="cfg-dm">
          <span>{d.since}</span>
          {isNew ? <span className="new">new</span> : null}
          {replaces ? <span>replaces {replaces}</span> : null}
        </div>
      </a>
    );
  };

  const docPill = (kind) => {
    const d = docByKind[kind];
    if (!d) return null;
    return (
      <a key={kind} className="cfg-pill" href={refHref(d)} title={"Open the " + kind + " reference"}>
        {kind} <span className="since">{d.since}</span>
      </a>
    );
  };

  const detail = () => {
    if (!selField) return null;
    const n = fields.filter((f) => f.path === selField)[0];
    if (!n) return null;

    const status = statusOf(n);
    const repl = replacements(n);
    const parent = status === "inherited" ? movedAncestor(n) : null;
    const parentRepl = parent ? replacements(parent) : [];

    return (
      <div className="cfg-detail" id="cfg-detail">
        <div className="cfg-drow">
          <div className="cfg-dtitle">
            <div className="cfg-title">{n.path}</div>
            <div className="cfg-meta">
              v1alpha1 field · <code>{n.type}</code>
            </div>
          </div>
          <button type="button" className="cfg-x" onClick={() => setSelField(null)} aria-label="Close">
            ×
          </button>
        </div>

        <p className={"cfg-verdict " + status}>
          {status === "moved"
            ? "Configured by a separate document as of " + version + "."
            : status === "inherited"
            ? "Moved out with its parent block " +
              parent.path +
              ", which a document replaces as of " +
              version +
              "."
            : status === "dropped"
            ? "Deprecated with no replacement document — this setting is gone."
            : "Still configured here. No replacement document exists."}
        </p>

        {n.desc ? <p className="cfg-p">{n.desc}</p> : null}
        {n.note ? <p className="cfg-note">{n.note}</p> : null}

        {repl.length ? (
          <div className="cfg-sec">
            <div className="cfg-h">Configure with</div>
            <div>{repl.map((k) => docPill(k))}</div>
          </div>
        ) : null}

        {parentRepl.length ? (
          <div className="cfg-sec">
            <div className="cfg-h">Configure with (via the parent block)</div>
            <div>{parentRepl.map((k) => docPill(k))}</div>
          </div>
        ) : null}
      </div>
    );
  };

  const visibleDocs = docs.filter(docMatches);
  const groupsWithDocs = GROUP_ORDER.filter((g) => docs.some((d) => d.group === g));
  const leftEmpty =
    !(tree.machine || []).some(subtreeMatches) && !(tree.cluster || []).some(subtreeMatches);
  const filtered = query || group || selField;

  return (
    <div className="cfg-root not-prose">
      <style dangerouslySetInnerHTML={{ __html: CSS }} />

      <div className="cfg-card">
        <div className="cfg-tools">
          <input
            type="search"
            className="cfg-search"
            value={query}
            placeholder="Search fields and documents…"
            onChange={(e) => setQuery(e.target.value)}
            aria-label="Search fields and documents"
          />
          <select
            className="cfg-select"
            value={group}
            onChange={(e) => setGroup(e.target.value)}
            aria-label="Filter by area"
          >
            <option value="">All areas</option>
            {groupsWithDocs.map((g) => (
              <option key={g} value={g}>
                {GROUP_LABEL[g] || g}
              </option>
            ))}
          </select>
          {filtered ? (
            <button
              type="button"
              className="cfg-chip"
              onClick={() => {
                setQuery("");
                setGroup("");
                setSelField(null);
              }}
            >
              Reset
            </button>
          ) : null}
        </div>
      </div>

      <div className="cfg-panes">
        <section className="cfg-pane">
          <div className="cfg-panehead">
            <span>← v1alpha1</span>
            <span className="cfg-sub">
              single <code>machine:</code>/<code>cluster:</code> document
            </span>
          </div>
          <div className="cfg-panebody">
            {leftEmpty ? (
              <p className="cfg-empty">No v1alpha1 fields match.</p>
            ) : (
              <div>
                {renderSection("machine", ".machine:")}
                {renderSection("cluster", ".cluster:")}
              </div>
            )}
          </div>
        </section>

        <section className="cfg-pane">
          <div className="cfg-panehead">
            <span>Config documents →</span>
            <span className="cfg-sub">
              one <code>kind:</code> per concern
            </span>
          </div>
          <div className="cfg-panebody">
            {visibleDocs.length ? (
              GROUP_ORDER.map((g) => {
                const items = visibleDocs
                  .filter((d) => d.group === g)
                  .sort((a, b) => vnum(a.since) - vnum(b.since) || a.kind.localeCompare(b.kind));
                if (!items.length) return null;
                return (
                  <div key={g} className="cfg-dgroup">
                    <div className="cfg-dhead">
                      {GROUP_LABEL[g] || g} · {items.length}
                    </div>
                    <div className="cfg-dgrid">{items.map(renderDoc)}</div>
                  </div>
                );
              })
            ) : (
              <p className="cfg-empty">No documents match.</p>
            )}
          </div>
        </section>
      </div>

      {detail()}

      <p className="cfg-foot">
        Click a v1alpha1 field to see which document replaces it. Click a document to open its
        reference page. Field counts cover the top three levels of the v1alpha1 schema — see the{" "}
        <a className="cfg-link" href={data.v1alpha1Href}>
          v1alpha1 reference
        </a>{" "}
        for every leaf.
      </p>
    </div>
  );
};
