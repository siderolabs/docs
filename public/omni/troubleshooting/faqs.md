---
title: "FAQs"
weight: 999
description: "Frequently Asked Questions about Omni."
---

import { version } from '/snippets/custom-variables.mdx';

## What do you need to self-host Omni?

Omni relies on what we call the "Sidero Stack" which consists of the following components:

* Container registry
* Image Factory
* Omni

There are also some supporting components you will need that aren't Sidero specific such as:

* Authentication
* TLS certificates
* Storage for logs and backups

## Is Omni open source?

Omni is licensed under the [Business Source License 1.1](https://github.com/siderolabs/omni/blob/main/LICENSE) which is not an open source license. Omni is a "source available" license which allows you to run Omni in non-production environments without a business agreement with Sidero Labs.

## Can Omni be run air gapped?

Yes, Omni and the rest of the Sidero Stack can be run without internet access.
To do so you will need to run each component within your environment.

Please see documents:

* [Deploy Image Factory on-prem](../self-hosted/run-image-factory-on-prem.mdx)
* [Deploy Omni on-prem](../self-hosted/run-omni-on-prem)

## Why do multiple Omni clusters appear as the same cluster in ArgoCD?

When using Omni-managed clusters behind the Omni kubeapi proxy, ArgoCD may treat multiple clusters as the same cluster if they share the same kubeapi server URL.

This happens because ArgoCD identifies clusters primarily by their `server` URL, not by their cluster name or credentials. Since Omni exposes clusters through a shared kubeapi proxy endpoint, different clusters can appear identical from ArgoCD’s perspective.

This is a limitation of ArgoCD and is not specific to Omni.

To ensure clusters are treated separately, each cluster must have a unique `server` value in ArgoCD. You can do this by:

- Appending a unique query parameter to the server URL (for example: `?cluster=cluster-a`)
- Exposing each cluster through a distinct API endpoint (via DNS, reverse proxy, or ingress)
- Running a separate ArgoCD instance per cluster

As long as the `server` URL is different, ArgoCD will treat the clusters as separate clusters.

## Should I enable embedded service discovery in Omni?

In most cases, **no**.

Omni can run an embedded discovery service for Talos clusters, but this mode has several limitations and is generally intended only for **air-gapped environments**.

### Limitations of embedded service discover in Omni

Embedded service discovery in Omni has several limitations compared to the public Talos Discovery Service, including:

- **Dependency on Omni availability**: If Omni becomes unavailable, the embedded discovery service is also unavailable.
  As a result, cluster discovery may fail and nodes might not rejoin the cluster after reboot.

- **No public IP awareness**: Embedded discovery cannot detect public IPs, which means it cannot assist with **KubeSpan NAT traversal**.

- **Tighter coupling with Omni**: Cluster discovery becomes dependent on Omni rather than running as an independent service.

### Recommendation

For most deployments, use the **public <a href={`../../talos/${version}/configure-your-talos-cluster/system-configuration/discovery`}>Talos Discovery Service</a>**.

For **air-gapped environments**, run a standalone discovery service alongside Omni instead of relying on the embedded discovery service.
