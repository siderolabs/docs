---
title: "SideroLabs documentation contribution guide"
---

Welcome to the official SideroLabs documentation, we are excited that you want to contribute!

This guide walks you through how to make changes to the SideroLabs Docs, which power the documentation for Talos, Omni, and the shared Kubernetes Guides both projects rely on.

Whether you’re fixing a typo or adding a brand-new page, this document explains how to get started, how the docs are structured, and how to make sure your contribution is clear, consistent, and high-quality.

## Where docs live

All SideroLabs documentation is housed in a single repository, organized under the `public/` folder:

```txt

public/

 ├── talos/               → Versioned Talos docs

 ├── omni/                → Unversioned Omni docs

 └── kubernetes-guides/   → Unversioned Kubernetes guides

```

The documentation is written in MDX format. You can learn more about[ MDX syntax from the Mintlify documentation](https://mintlify.com/docs) and the [official MDX website](https://mdxjs.com/docs/). 


## Docs GitHub issues

The [Docs repository issues](https://github.com/siderolabs/docs/issues) page lists known gaps or inaccuracies in the documentation. These issues are open for anyone to contribute to, whether you’re part of the SideroLabs team or a member of the community.

Sometimes, users also share doc-related feedback or questions directly in the [Talos GitHub repository](https://github.com/siderolabs/talos), usually under the [Docs_improvement label](https://github.com/siderolabs/talos/issues?q=is%3Aissue%20state%3Aopen%20label%3ADoc_improvements).

Issues with the `Docs_improvement` label might be resolved in the Talos repo or migrated to the docs repo for further updates.

To understand how a `Docs_improvement` issue moves from open to merged, check out the [Documentation Labeling and Workflow guide](./contributing-guides/documentation-labeling-and-workflow.md).


## Writing and style standards

Before you start writing or making any contributions, please review our [Style Guide](https://www.notion.so/siderolabs/Documentation-style-guide-24eb1211badf80ecafc2c87635329719?source=copy_link). It explains how we write, format, and structure documentation at SideroLabs.

Following the style guide helps keep every contribution consistent, clear, and approachable, no matter who wrote it.

## Make a PR

When you're ready to contribute, head over to our [How to contribute to the SideroLabs documentation guide](./contributing-guides/contribute-to-the-siderolabs-docs.md). It walks you through, step by step, how to create a pull request and submit your documentation changes.