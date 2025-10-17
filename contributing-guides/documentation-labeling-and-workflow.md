---
title: "Documentation labeling and workflow"
---

Use the `Doc_improvements` label to track questions, issues, or discussions relating to the docs on the [Talos repo](https://github.com/siderolabs/talos).

## When and how to handle `Doc_improvements` issues

If the question is answered in the comments or GitHub discussion and it requires changes to the docs, it may not always be realistic to keep the issue open, as users or engineers may close the issue once their question is resolved.

In that case:

1. Add the `Doc_improvements` label to the issue (open or closed) to indicate a documentation update is needed.
2. Open a new issue in the [docs repository](https://github.com/siderolabs/docs) describing what needs to be updated.
3. Once the docs issue is created, remove the `Doc_improvements` label from the original issue to avoid duplication. The label acts as a temporary pointer rather than a permanent historical marker.
4. Keep the issue open until a documentation pull request has been created and merged.

Even after the pull request that addresses the documentation issue has been created and merged, do not remove the label.

Keeping the label provides a clear historical record of all documentation updates.


### Answered issues that cannot be documented

Sometimes users raise issues that are resolved with a workaround but don’t need to be added to the docs. Most of the time, it’s because the user is really asking for a new feature or an improvement.

In those cases, switch the label from `doc_improvements` to `kind/feature` so the development team can decide whether the feature should be prioritized.
