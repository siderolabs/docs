---
title: Omni Support Matrix
---
The Sidero Labs managed version of Omni SaaS is updated regularly by our Operations team. For a list of the most recent updates, bug fixes and changes, please subscribe to the GitHub [Release notes.](https://github.com/siderolabs/omni/releases)

If you are running a self-hosted version of Omni licensed under the BSL, please regularly update to the latest release - we suggest at least monthly. Bug fixes will not be backported to older versions of Omni, so support that involves a bug fix will require an update.

## Talos Linux Versions Supported

Each version of Omni will support versions of Talos Linux where the first stable release of the minor version of Talos Linux was within 18 months of the Omni release date. For example, because Talos Linux 1.3.0 was released on 2022-12-15, the Omni version released on Jun 30th, 2024 will not support any version of 1.3.x, even though patch releases of 1.3 were made within the prior 18 months. However, all versions of Talos Linux Talos 1.4 and later (which was released on 2023-04-18) are supported.

## Current Minimum Talos Linux Support Within Omni

Talos Linux 1.4

<Info>
SecureBoot with Omni is only supported for Talos Linux versions of 1.7.0 or greater.
</Info>

Note that each version of Talos Linux generally supports up to five prior versions of Kubernetes at the time it was released. See [https://www.talos.dev/latest/introduction/support-matrix/](https://www.talos.dev/latest/introduction/support-matrix/) for details.

## Effect of non-supported Talos Linux versions

If you neglect to update Talos Linux to a version where the initial minor version was released within the prior 18 months, your cluster will be out of support, and you must upgrade in order to receive support.

Clusters running unsupported versions of Talos Linux or Kubernetes may have limited ability to manage their clusters within Omni - clusters may not be able to have new nodes provisioned to scale up the cluster, for example.

We encourage all users to take advantage of how easy Omni makes Talos Linux and Kubernetes upgrades, to ensure they are always running the latest security and bug fixes.