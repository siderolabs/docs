{/*
  GENERATED FILE — do not edit by hand.

  Regenerate with to-docs-page.py from the config-explorer dataset, which is itself derived
  from the Talos source at pkg/machinery/config/types/ (document registrations plus the
  `Deprecated:` markers on the v1alpha1 structs) and from the release tags that first
  contain each registration.

  Capped at v1.14: no document that ships after v1.14 appears here.
*/}

export const CONFIG_EVOLUTION = {
 "version": "v1.14",
 "refBase": "/talos/v1.14/reference/configuration",
 "v1alpha1Href": "/talos/v1.14/reference/configuration/v1alpha1/config",
 "docs": [
  {
   "kind": "BGPInstanceConfig",
   "group": "network",
   "since": "v1.14",
   "desc": "Native BGP via an embedded GoBGP instance — named instances that can advertise VIP/pod CIDRs and import routes between instances, no external speaker needed."
  },
  {
   "kind": "BlackholeRouteConfig",
   "group": "network",
   "since": "v1.13",
   "desc": "Install a blackhole route to silently drop traffic to a prefix."
  },
  {
   "kind": "BondConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Bond several links into one for link aggregation / failover, choosing a bonding mode."
  },
  {
   "kind": "BridgeConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Create a software L2 bridge and enslave links to it."
  },
  {
   "kind": "DHCPv4Config",
   "group": "network",
   "since": "v1.12",
   "desc": "Tune the DHCPv4 client per link. v1.14 adds `ignoreRoutes`; `skipHostnameRequest` avoids DHCP hostname override."
  },
  {
   "kind": "DHCPv6Config",
   "group": "network",
   "since": "v1.12",
   "desc": "Tune the DHCPv6 client per link."
  },
  {
   "kind": "DummyLinkConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Create a dummy interface (a stable local address holder / blackhole)."
  },
  {
   "kind": "EthernetConfig",
   "group": "network",
   "since": "v1.10",
   "desc": "Tune NIC hardware settings: ring buffers, channels, features (offloads), and WoL."
  },
  {
   "kind": "HCloudVIPConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Manage a Hetzner Cloud floating IP as a VIP via the cloud API."
  },
  {
   "kind": "HTTPProbeConfig",
   "group": "network",
   "since": "v1.14",
   "desc": "Define an HTTP reachability probe used by network readiness logic."
  },
  {
   "kind": "HostnameConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Set the node hostname (and domain)."
  },
  {
   "kind": "KubeSpanConfig",
   "group": "network",
   "since": "v1.13",
   "desc": "Enable/configure KubeSpan — Talos's automatic WireGuard mesh between all cluster nodes."
  },
  {
   "kind": "KubeSpanEndpointsConfig",
   "group": "network",
   "since": "v1.8",
   "desc": "Provide extra/override endpoints for KubeSpan peers (e.g. behind NAT)."
  },
  {
   "kind": "Layer2VIPConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Configure a shared/floating layer-2 VIP for HA control planes (gratuitous-ARP failover)."
  },
  {
   "kind": "LinkAliasConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Give a link a stable alias/selector so config can target it by predictable name rather than kernel name (eth0/enp3s0)."
  },
  {
   "kind": "LinkConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Configure a physical/logical network link: up/down, MTU, and whether it participates in DHCP."
  },
  {
   "kind": "NetworkDefaultActionConfig",
   "group": "network",
   "since": "v1.6",
   "desc": "Set the default ingress firewall policy (accept vs block) that the rules refine."
  },
  {
   "kind": "NetworkRuleConfig",
   "group": "network",
   "since": "v1.6",
   "desc": "Declare an ingress firewall rule (ports/protocols/sources) for the host."
  },
  {
   "kind": "ResolverConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Set DNS nameservers and search domains. v1.14 adds per-server protocol (Do53/DoT/DoH) + host DNS caching options; `domains` override DHCP search domains."
  },
  {
   "kind": "RoutingRuleConfig",
   "group": "network",
   "since": "v1.13",
   "desc": "Add policy routing rules (rule table selection by src/fwmark)."
  },
  {
   "kind": "StaticHostConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Add static host-to-IP mappings resolved locally."
  },
  {
   "kind": "TCPProbeConfig",
   "group": "network",
   "since": "v1.13",
   "desc": "Define a TCP reachability probe used by network readiness logic."
  },
  {
   "kind": "TimeSyncConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Configure NTP time servers."
  },
  {
   "kind": "VLANConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Create 802.1Q VLAN sub-interfaces on a parent link."
  },
  {
   "kind": "VRFConfig",
   "group": "network",
   "since": "v1.13",
   "desc": "Create a VRF (virtual routing & forwarding) device to isolate a routing table."
  },
  {
   "kind": "VethConfig",
   "group": "network",
   "since": "v1.14",
   "desc": "Create a veth (virtual ethernet) pair — two linked interfaces, typically to wire a namespace/bridge together."
  },
  {
   "kind": "WireguardConfig",
   "group": "network",
   "since": "v1.12",
   "desc": "Manually configure a WireGuard interface (peers, keys, allowed IPs) as a plain overlay — distinct from KubeSpan."
  },
  {
   "kind": "KubeAPIServerCAConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "The Kubernetes API server CA cert/key (extracted from the secrets bundle)."
  },
  {
   "kind": "KubeAPIServerConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "kube-apiserver settings: extra args, extra volumes, cert SANs, admission."
  },
  {
   "kind": "KubeAdmissionControlConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Admission controller / PodSecurity configuration for the API server."
  },
  {
   "kind": "KubeAggregatorCAConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "The front-proxy/aggregator CA used for the API aggregation layer."
  },
  {
   "kind": "KubeAuditPolicyConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "API server audit policy."
  },
  {
   "kind": "KubeAuthenticationConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "API server authentication config (e.g. structured auth / OIDC)."
  },
  {
   "kind": "KubeAuthorizerConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "API server authorization mode/config (e.g. structured authz)."
  },
  {
   "kind": "KubeClusterConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Cluster-wide Kubernetes base settings (name, dns domain, cluster-level toggles) extracted from v1alpha1 `.cluster`."
  },
  {
   "kind": "KubeControllerManagerConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "kube-controller-manager extra args and volumes."
  },
  {
   "kind": "KubeCoreDNSConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Cluster DNS (CoreDNS) settings, or disabling the managed deployment."
  },
  {
   "kind": "KubeCredentialProviderConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Kubelet image credential provider config for registry auth."
  },
  {
   "kind": "KubeEtcdEncryptionConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Encryption-at-rest config for Kubernetes secrets in etcd."
  },
  {
   "kind": "KubeExternalManifestConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Bootstrap manifests fetched from external URLs at cluster bootstrap."
  },
  {
   "kind": "KubeFlannelCNIConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Flannel CNI settings; `backendMTU` now auto-aligns to the KubeSpan MTU."
  },
  {
   "kind": "KubeInlineManifestConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Bootstrap manifests embedded inline, applied once at cluster bootstrap."
  },
  {
   "kind": "KubeNetworkConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Pod/service CIDRs and cluster networking (moved out of `.cluster.network`); supports per-node pod CIDR."
  },
  {
   "kind": "KubeNodeConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Per-node Kubernetes settings extracted from the v1alpha1 `.cluster`/`.machine` blocks."
  },
  {
   "kind": "KubePrismConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "KubePrism — the local load-balancing API endpoint on each node (replaces `.machine.features.kubePrism`)."
  },
  {
   "kind": "KubeProxyConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "kube-proxy settings (mode, extra args), or disabling it for eBPF CNIs."
  },
  {
   "kind": "KubeSchedulerConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "kube-scheduler extra args, config, and volumes."
  },
  {
   "kind": "KubeServiceAccountConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Service-account issuer/signing key configuration."
  },
  {
   "kind": "KubeStaticPodConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Define extra static pods to run on nodes (moved out of `.machine.pods`)."
  },
  {
   "kind": "KubeTalosAPIAccessConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Controls whether/which Talos API access is granted to workloads in Kubernetes (moved out of Kubernetes config)."
  },
  {
   "kind": "KubeletConfig",
   "group": "kubernetes",
   "since": "v1.14",
   "desc": "Kubelet configuration (replaces `.machine.kubelet`): extra args, node IP, feature gates, extra mounts."
  },
  {
   "kind": "ExistingVolumeConfig",
   "group": "block",
   "since": "v1.11",
   "desc": "Adopt an already-formatted volume (by selector) without wiping it, and mount it."
  },
  {
   "kind": "ExternalVolumeConfig",
   "group": "block",
   "since": "v1.13",
   "desc": "Reference storage provisioned/attached outside Talos (e.g. iSCSI/cloud disk) so it can be mounted."
  },
  {
   "kind": "FilesystemScrubConfig",
   "group": "block",
   "since": "v1.14",
   "desc": "Schedule periodic online filesystem scrubbing (xfs_scrub / btrfs scrub) on filesystems that support it (min interval 10s)."
  },
  {
   "kind": "FilesystemTrimConfig",
   "group": "block",
   "since": "v1.14",
   "desc": "Schedule periodic `fstrim` on SSD-backed volumes (default 7-day interval), with a per-volume `trim:` override."
  },
  {
   "kind": "RawVolumeConfig",
   "group": "block",
   "since": "v1.11",
   "desc": "Provision a raw (unformatted) partition to hand to a CSI driver or app that manages its own format."
  },
  {
   "kind": "SwapVolumeConfig",
   "group": "block",
   "since": "v1.11",
   "desc": "Provision a swap volume (optionally encrypted); pairs with Kubernetes swap support."
  },
  {
   "kind": "UserVolumeConfig",
   "group": "block",
   "since": "v1.10",
   "desc": "Carve a user-defined volume out of a selected disk (CEL selector), pick size and filesystem (ext4/xfs, btrfs in v1.14). Mounted under /var/mnt."
  },
  {
   "kind": "VolumeConfig",
   "group": "block",
   "since": "v1.8",
   "desc": "Customize built-in system volumes (EPHEMERAL, and per-service overrides for CRI/kubelet/etcd): disk selector, size, encryption. One-time provisioning for system volumes."
  },
  {
   "kind": "ZswapConfig",
   "group": "block",
   "since": "v1.11",
   "desc": "Enable zswap — a compressed RAM cache in front of swap devices."
  },
  {
   "kind": "LVMLogicalVolumeConfig",
   "group": "storage",
   "since": "v1.14",
   "desc": "Declaratively define logical volumes in a VG (linear/raid0/raid1/raid10). Reconcile is additive-only — shrinking errors out."
  },
  {
   "kind": "LVMVolumeGroupConfig",
   "group": "storage",
   "since": "v1.14",
   "desc": "Declaratively define an LVM volume group from matching physical volumes/disks."
  },
  {
   "kind": "RAIDArrayConfig",
   "group": "storage",
   "since": "v1.14",
   "desc": "Declaratively provision a Linux MD (software RAID) array from matching disks; Talos can boot from a RAID1 array."
  },
  {
   "kind": "CRIBaseRuntimeSpecConfig",
   "group": "cri",
   "since": "v1.14",
   "desc": "Set the containerd base OCI runtime spec applied to all containers."
  },
  {
   "kind": "CRICustomizationConfig",
   "group": "cri",
   "since": "v1.14",
   "desc": "Layer additional containerd/CRI config customizations (dedicated doc, replaces inline file tweaks)."
  },
  {
   "kind": "ImageCacheConfig",
   "group": "cri",
   "since": "v1.14",
   "desc": "Enable the local image cache (`local.enabled: true`) — the canonical replacement for `.machine.features.imageCache.localEnabled`."
  },
  {
   "kind": "RegistryAuthConfig",
   "group": "cri",
   "since": "v1.12",
   "desc": "Provide pull credentials for a registry."
  },
  {
   "kind": "RegistryMirrorConfig",
   "group": "cri",
   "since": "v1.12",
   "desc": "Point a registry (e.g. docker.io) at one or more mirror endpoints; replaces `.machine.registries.mirrors`."
  },
  {
   "kind": "RegistryTLSConfig",
   "group": "cri",
   "since": "v1.12",
   "desc": "Provide TLS settings (CA/insecure/client cert) for a registry."
  },
  {
   "kind": "EnvironmentConfig",
   "group": "runtime",
   "since": "v1.13",
   "desc": "Set machine-wide environment variables (replaces `.machine.env`)."
  },
  {
   "kind": "EtcFileConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Declaratively manage files under /etc (name/contents/mode)."
  },
  {
   "kind": "EventSinkConfig",
   "group": "runtime",
   "since": "v1.5",
   "desc": "Send Talos runtime events to an external gRPC sink."
  },
  {
   "kind": "KernelModuleConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Load kernel modules with options (multi-doc, replaces `.machine.kernel.modules`)."
  },
  {
   "kind": "KmsgLogConfig",
   "group": "runtime",
   "since": "v1.5",
   "desc": "Ship kernel/service logs to a remote sink (UDP/TCP json_lines)."
  },
  {
   "kind": "OOMConfig",
   "group": "runtime",
   "since": "v1.12",
   "desc": "Configure the userspace OOM handler: CEL trigger/ranking expressions for which cgroup to kill under memory pressure."
  },
  {
   "kind": "SecurityProfileConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Adjust machine security profile settings (e.g. seccomp/hardening toggles)."
  },
  {
   "kind": "SysctlConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Set kernel sysctls (replaces `.machine.sysctls`); config generate emits it as a multi-doc."
  },
  {
   "kind": "SysfsConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Set /sys attributes (replaces `.machine.sysfs`)."
  },
  {
   "kind": "UdevRulesConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Add udev rules (additive to the legacy `.machine.udev` field)."
  },
  {
   "kind": "UnattendedInstallConfig",
   "group": "runtime",
   "since": "v1.14",
   "desc": "Drive fully automated, hands-off installation (no interactive/maintenance step)."
  },
  {
   "kind": "WatchdogTimerConfig",
   "group": "runtime",
   "since": "v1.7",
   "desc": "Arm a hardware watchdog so a wedged node auto-reboots."
  },
  {
   "kind": "DiscoveryIdentityConfig",
   "group": "cluster",
   "since": "v1.14",
   "desc": "Holds the cluster ID and shared secret (extracted from v1alpha1) that identify the cluster to discovery."
  },
  {
   "kind": "DiscoveryServiceConfig",
   "group": "cluster",
   "since": "v1.14",
   "desc": "Configure the external discovery service registry endpoint used for node discovery."
  },
  {
   "kind": "ImageVerificationConfig",
   "group": "security",
   "since": "v1.13",
   "desc": "Machine-wide container image signature verification policy."
  },
  {
   "kind": "TrustedRootsConfig",
   "group": "security",
   "since": "v1.8",
   "desc": "Add extra trusted CA root certificates machine-wide (for corporate TLS interception or private registries)."
  },
  {
   "kind": "PCIDriverRebindConfig",
   "group": "hardware",
   "since": "v1.10",
   "desc": "Rebind a PCI device to a different kernel driver (e.g. hand a NIC/GPU to vfio-pci for passthrough)."
  },
  {
   "kind": "ExtensionServiceConfig",
   "group": "extensions",
   "since": "v1.7",
   "desc": "Provide runtime config (env, files, args) to an installed extension service."
  },
  {
   "kind": "SideroLinkConfig",
   "group": "siderolink",
   "since": "v1.5",
   "desc": "Configure the SideroLink point-to-point WireGuard management overlay (used by Omni)."
  }
 ],
 "tree": {
  "machine": [
   {
    "path": ".machine.type",
    "field": "type",
    "type": "string",
    "desc": "Defines the role of the machine within the cluster. **Control Plane** Control Plane node type designates the node as a control plane member."
   },
   {
    "path": ".machine.token",
    "field": "token",
    "type": "string",
    "desc": "The `token` is used by a machine to join the PKI of the cluster. Using this token, a machine will create a certificate signing request (CSR), and request a certificate that will be used as its' identity."
   },
   {
    "path": ".machine.ca",
    "field": "ca",
    "type": "*x509.PEMEncodedCertificateAndKey",
    "desc": "The root certificate authority of the PKI. It is composed of a base64 encoded `crt` and `key`."
   },
   {
    "path": ".machine.acceptedCAs",
    "field": "acceptedCAs",
    "type": "[]*x509.PEMEncodedCertificate",
    "desc": "The certificates issued by certificate authorities are accepted in addition to issuing 'ca'. It is composed of a base64 encoded `crt``."
   },
   {
    "path": ".machine.certSANs",
    "field": "certSANs",
    "type": "[]string",
    "desc": "Extra certificate subject alternative names for the machine's certificate. By default, all non-loopback interface IPs are automatically added to the certificate's SANs."
   },
   {
    "path": ".machine.controlPlane",
    "field": "controlPlane",
    "type": "*MachineControlPlaneConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeControllerManagerConfig",
     "KubeSchedulerConfig"
    ],
    "note": "Deprecated: Use `KubeControllerManagerConfig`/`KubeSchedulerConfig` instead.",
    "children": [
     {
      "path": ".machine.controlPlane.controllerManager",
      "field": "controllerManager",
      "type": "*MachineControllerManagerConfig",
      "deprecated": true,
      "replacedBy": [
       "KubeControllerManagerConfig"
      ],
      "note": "Deprecated: Use `KubeControllerManagerConfig` instead.",
      "children": [
       {
        "path": ".machine.controlPlane.controllerManager.disabled",
        "field": "disabled",
        "type": "*bool",
        "desc": "Disable kube-controller-manager on the node."
       }
      ]
     },
     {
      "path": ".machine.controlPlane.scheduler",
      "field": "scheduler",
      "type": "*MachineSchedulerConfig",
      "deprecated": true,
      "replacedBy": [
       "KubeSchedulerConfig"
      ],
      "note": "Deprecated: Use `KubeSchedulerConfig` instead.",
      "children": [
       {
        "path": ".machine.controlPlane.scheduler.disabled",
        "field": "disabled",
        "type": "*bool",
        "desc": "Disable kube-scheduler on the node."
       }
      ]
     }
    ]
   },
   {
    "path": ".machine.kubelet",
    "field": "kubelet",
    "type": "*KubeletConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeletConfig"
    ],
    "note": "Deprecated: Use `KubeletConfig` instead.",
    "children": [
     {
      "path": ".machine.kubelet.image",
      "field": "image",
      "type": "string",
      "deprecated": true,
      "replacedBy": [
       "KubeletConfig"
      ],
      "note": "Deprecated: use `KubeletConfig` instead."
     },
     {
      "path": ".machine.kubelet.clusterDNS",
      "field": "clusterDNS",
      "type": "[]string",
      "deprecated": true,
      "replacedBy": [
       "KubeletConfig"
      ],
      "note": "Deprecated: use `KubeletConfig` instead."
     },
     {
      "path": ".machine.kubelet.extraArgs",
      "field": "extraArgs",
      "type": "meta.Args",
      "deprecated": true,
      "replacedBy": [
       "KubeletConfig"
      ],
      "note": "Deprecated: use `KubeletConfig` instead."
     },
     {
      "path": ".machine.kubelet.extraMounts",
      "field": "extraMounts",
      "type": "[]ExtraMount",
      "deprecated": true,
      "note": "Deprecated: removed in multi-doc config.",
      "children": [
       {
        "path": ".machine.kubelet.extraMounts.destination",
        "field": "destination",
        "type": "string",
        "desc": "Destination is the absolute path where the mount will be placed in the container."
       },
       {
        "path": ".machine.kubelet.extraMounts.type",
        "field": "type",
        "type": "string",
        "desc": "Type specifies the mount kind."
       },
       {
        "path": ".machine.kubelet.extraMounts.source",
        "field": "source",
        "type": "string",
        "desc": "Source specifies the source path of the mount."
       },
       {
        "path": ".machine.kubelet.extraMounts.options",
        "field": "options",
        "type": "[]string",
        "desc": "Options are fstab style mount options."
       },
       {
        "path": ".machine.kubelet.extraMounts.uidMappings",
        "field": "uidMappings",
        "type": "[]LinuxIDMapping",
        "desc": "UID/GID mappings used for changing file owners w/o calling chown, fs should support it. Every mount point could have its own mapping."
       },
       {
        "path": ".machine.kubelet.extraMounts.gidMappings",
        "field": "gidMappings",
        "type": "[]LinuxIDMapping",
        "desc": "UID/GID mappings used for changing file owners w/o calling chown, fs should support it. Every mount point could have its own mapping."
       }
      ]
     },
     {
      "path": ".machine.kubelet.extraConfig",
      "field": "extraConfig",
      "type": "meta.Unstructured",
      "deprecated": true,
      "replacedBy": [
       "KubeletConfig"
      ],
      "note": "Deprecated: use `KubeletConfig` instead."
     },
     {
      "path": ".machine.kubelet.credentialProviderConfig",
      "field": "credentialProviderConfig",
      "type": "meta.Unstructured",
      "desc": "The `KubeletCredentialProviderConfig` field is used to provide kubelet credential configuration."
     },
     {
      "path": ".machine.kubelet.defaultRuntimeSeccompProfileEnabled",
      "field": "defaultRuntimeSeccompProfileEnabled",
      "type": "*bool",
      "deprecated": true,
      "replacedBy": [
       "KubeletConfig"
      ],
      "note": "Deprecated: use `KubeletConfig` instead."
     },
     {
      "path": ".machine.kubelet.registerWithFQDN",
      "field": "registerWithFQDN",
      "type": "*bool",
      "deprecated": true,
      "replacedBy": [
       "KubeNodeConfig"
      ],
      "note": "Deprecated: use `KubeNodeConfig` instead."
     },
     {
      "path": ".machine.kubelet.nodeIP",
      "field": "nodeIP",
      "type": "*KubeletNodeIPConfig",
      "deprecated": true,
      "replacedBy": [
       "KubeNodeConfig"
      ],
      "note": "Deprecated: use `KubeNodeConfig` instead.",
      "children": [
       {
        "path": ".machine.kubelet.nodeIP.validSubnets",
        "field": "validSubnets",
        "type": "[]string",
        "desc": "The `validSubnets` field configures the networks to pick kubelet node IP from. For dual stack configuration, there should be two subnets: one for IPv4, another for IPv6."
       }
      ]
     },
     {
      "path": ".machine.kubelet.skipNodeRegistration",
      "field": "skipNodeRegistration",
      "type": "*bool",
      "deprecated": true,
      "replacedBy": [
       "KubeNodeConfig"
      ],
      "note": "Deprecated: use `KubeNodeConfig` instead."
     },
     {
      "path": ".machine.kubelet.disableManifestsDirectory",
      "field": "disableManifestsDirectory",
      "type": "*bool",
      "deprecated": true,
      "note": "Deprecated: locked to true in multi-doc config."
     }
    ]
   },
   {
    "path": ".machine.pods",
    "field": "pods",
    "type": "[]meta.Unstructured",
    "deprecated": true,
    "replacedBy": [
     "KubeStaticPodConfig"
    ],
    "note": "Deprecated: Use `KubeStaticPodConfig` instead."
   },
   {
    "path": ".machine.network",
    "field": "network",
    "type": "*NetworkConfig",
    "deprecated": true,
    "replacedBy": [
     "HostnameConfig",
     "KubeSpanConfig",
     "LinkConfig",
     "ResolverConfig",
     "StaticHostConfig"
    ],
    "note": "Deprecated: All fields within NetworkConfig are deprecated. Use multi-document network config types instead: HostnameConfig, NetworkDeviceConfig, ResolverConfig, StaticHostConfig, KubeSpanConfig.",
    "desc": "HostnameConfig, NetworkDeviceConfig, ResolverConfig, StaticHostConfig, KubeSpanConfig.",
    "children": [
     {
      "path": ".machine.network.hostname",
      "field": "hostname",
      "type": "string",
      "deprecated": true,
      "replacedBy": [
       "HostnameConfig"
      ],
      "note": "Deprecated: use `HostnameConfig` instead."
     },
     {
      "path": ".machine.network.interfaces",
      "field": "interfaces",
      "type": "NetworkDeviceList",
      "deprecated": true,
      "note": "Deprecated: use multi-doc network config."
     },
     {
      "path": ".machine.network.nameservers",
      "field": "nameservers",
      "type": "[]string",
      "deprecated": true,
      "replacedBy": [
       "ResolverConfig"
      ],
      "note": "Deprecated: Use `ResolverConfig` instead."
     },
     {
      "path": ".machine.network.searchDomains",
      "field": "searchDomains",
      "type": "[]string",
      "deprecated": true,
      "replacedBy": [
       "ResolverConfig"
      ],
      "note": "Deprecated: Use `ResolverConfig` instead."
     },
     {
      "path": ".machine.network.extraHostEntries",
      "field": "extraHostEntries",
      "type": "[]*ExtraHost",
      "deprecated": true,
      "replacedBy": [
       "StaticHostConfig"
      ],
      "note": "Deprecated: Use `StaticHostConfig` instead.",
      "children": [
       {
        "path": ".machine.network.extraHostEntries.ip",
        "field": "ip",
        "type": "string",
        "desc": "The IP of the host."
       },
       {
        "path": ".machine.network.extraHostEntries.aliases",
        "field": "aliases",
        "type": "[]string",
        "desc": "The host alias."
       }
      ]
     },
     {
      "path": ".machine.network.kubespan",
      "field": "kubespan",
      "type": "*NetworkKubeSpan",
      "deprecated": true,
      "replacedBy": [
       "KubeSpanConfig"
      ],
      "note": "Deprecated: Use `KubeSpanConfig` document instead.",
      "children": [
       {
        "path": ".machine.network.kubespan.enabled",
        "field": "enabled",
        "type": "*bool",
        "desc": "Enable the KubeSpan feature. Cluster discovery should be enabled with .cluster.discovery.enabled for KubeSpan to be enabled."
       },
       {
        "path": ".machine.network.kubespan.advertiseKubernetesNetworks",
        "field": "advertiseKubernetesNetworks",
        "type": "*bool",
        "desc": "Control whether Kubernetes pod CIDRs are announced over KubeSpan from the node. If disabled, CNI handles encapsulating pod-to-pod traffic into some node-to-node tunnel, and KubeSpan handles the node-to-node traffic."
       },
       {
        "path": ".machine.network.kubespan.allowDownPeerBypass",
        "field": "allowDownPeerBypass",
        "type": "*bool",
        "desc": "Skip sending traffic via KubeSpan if the peer connection state is not up."
       },
       {
        "path": ".machine.network.kubespan.harvestExtraEndpoints",
        "field": "harvestExtraEndpoints",
        "type": "*bool",
        "desc": "KubeSpan can collect and publish extra endpoints for each member of the cluster based on Wireguard endpoint information for each peer."
       },
       {
        "path": ".machine.network.kubespan.mtu",
        "field": "mtu",
        "type": "*uint32",
        "desc": "KubeSpan link MTU size. Default value is 1420."
       },
       {
        "path": ".machine.network.kubespan.filters",
        "field": "filters",
        "type": "*KubeSpanFilters",
        "desc": "KubeSpan advanced filtering of network addresses . Settings in this section are optional, and settings apply only to the node."
       }
      ]
     },
     {
      "path": ".machine.network.disableSearchDomain",
      "field": "disableSearchDomain",
      "type": "*bool",
      "deprecated": true,
      "replacedBy": [
       "ResolverConfig"
      ],
      "note": "Deprecated: Use `ResolverConfig` instead."
     }
    ]
   },
   {
    "path": ".machine.disks",
    "field": "disks",
    "type": "[]*MachineDisk",
    "deprecated": true,
    "replacedBy": [
     "UserVolumeConfig"
    ],
    "note": "Deprecated: Use 'UserVolumeConfig' instead.",
    "children": [
     {
      "path": ".machine.disks.device",
      "field": "device",
      "type": "string",
      "desc": "The name of the disk to use."
     },
     {
      "path": ".machine.disks.partitions",
      "field": "partitions",
      "type": "[]*DiskPartition",
      "desc": "A list of partitions to create on the disk.",
      "children": [
       {
        "path": ".machine.disks.partitions.size",
        "field": "size",
        "type": "DiskSize",
        "desc": "The size of partition: either bytes or human readable representation. If `size:` is omitted, the partition is sized to occupy the full disk."
       },
       {
        "path": ".machine.disks.partitions.mountpoint",
        "field": "mountpoint",
        "type": "string",
        "desc": "Where to mount the partition."
       }
      ]
     }
    ]
   },
   {
    "path": ".machine.install",
    "field": "install",
    "type": "*InstallConfig",
    "deprecated": true,
    "replacedBy": [
     "UnattendedInstallConfig"
    ],
    "note": "Deprecated: Use the 'UnattendedInstall' multi-document config instead.",
    "children": [
     {
      "path": ".machine.install.disk",
      "field": "disk",
      "type": "string",
      "desc": "The disk used for installations."
     },
     {
      "path": ".machine.install.diskSelector",
      "field": "diskSelector",
      "type": "*InstallDiskSelector",
      "desc": "Look up disk using disk attributes like model, size, serial and others. Always has priority over `disk`.",
      "children": [
       {
        "path": ".machine.install.diskSelector.size",
        "field": "size",
        "type": "*InstallDiskSizeMatcher",
        "desc": "Disk size. schema: type: string…"
       },
       {
        "path": ".machine.install.diskSelector.name",
        "field": "name",
        "type": "string",
        "desc": "Disk name `/sys/block/<dev>/device/name`."
       },
       {
        "path": ".machine.install.diskSelector.model",
        "field": "model",
        "type": "string",
        "desc": "Disk model `/sys/block/<dev>/device/model`."
       },
       {
        "path": ".machine.install.diskSelector.serial",
        "field": "serial",
        "type": "string",
        "desc": "Disk serial number `/sys/block/<dev>/serial`."
       },
       {
        "path": ".machine.install.diskSelector.modalias",
        "field": "modalias",
        "type": "string",
        "desc": "Disk modalias `/sys/block/<dev>/device/modalias`."
       },
       {
        "path": ".machine.install.diskSelector.uuid",
        "field": "uuid",
        "type": "string",
        "desc": "Disk UUID `/sys/block/<dev>/uuid`."
       },
       {
        "path": ".machine.install.diskSelector.wwid",
        "field": "wwid",
        "type": "string",
        "desc": "Disk WWID `/sys/block/<dev>/wwid`."
       },
       {
        "path": ".machine.install.diskSelector.type",
        "field": "type",
        "type": "InstallDiskType",
        "desc": "Disk Type. values…"
       },
       {
        "path": ".machine.install.diskSelector.busPath",
        "field": "busPath",
        "type": "string",
        "desc": "Disk bus path."
       }
      ]
     },
     {
      "path": ".machine.install.extraKernelArgs",
      "field": "extraKernelArgs",
      "type": "[]string",
      "deprecated": true,
      "note": "Deprecated: Use Image Factory/imager instead to build a proper installer."
     },
     {
      "path": ".machine.install.image",
      "field": "image",
      "type": "string",
      "desc": "Allows for supplying the image used to perform the installation. Image reference for each Talos release can be found on [GitHub releases page](https://github.com/siderolabs/talos/releases)."
     },
     {
      "path": ".machine.install.extensions",
      "field": "extensions",
      "type": "[]InstallExtensionConfig",
      "deprecated": true,
      "note": "Deprecated: Use custom `InstallImage` instead.",
      "children": [
       {
        "path": ".machine.install.extensions.image",
        "field": "image",
        "type": "string",
        "desc": "System extension image."
       }
      ]
     },
     {
      "path": ".machine.install.bootloader",
      "field": "bootloader",
      "type": "*bool",
      "deprecated": true,
      "note": "Deprecated: It never worked."
     },
     {
      "path": ".machine.install.wipe",
      "field": "wipe",
      "type": "*bool",
      "desc": "Indicates if the installation disk should be wiped at installation time. Defaults to `true`."
     },
     {
      "path": ".machine.install.legacyBIOSSupport",
      "field": "legacyBIOSSupport",
      "type": "*bool",
      "desc": "Indicates if MBR partition should be marked as bootable (active). Should be enabled only for the systems with legacy BIOS that doesn't support GPT partitioning scheme."
     },
     {
      "path": ".machine.install.grubUseUKICmdline",
      "field": "grubUseUKICmdline",
      "type": "*bool",
      "desc": "Indicates if legacy GRUB bootloader should use kernel cmdline from the UKI instead of building it on the host. This changes the way cmdline is managed with GRUB bootloader to be more consistent with UKI/systemd-boot."
     }
    ]
   },
   {
    "path": ".machine.files",
    "field": "files",
    "type": "[]*MachineFile",
    "deprecated": true,
    "replacedBy": [
     "CRICustomizationConfig",
     "EtcFileConfig"
    ],
    "note": "Deprecated: Use dedicated configuration documents such as EtcFileConfig and CRICustomizationConfig instead.",
    "children": [
     {
      "path": ".machine.files.content",
      "field": "content",
      "type": "string",
      "desc": "The contents of the file."
     },
     {
      "path": ".machine.files.permissions",
      "field": "permissions",
      "type": "FileMode",
      "desc": "The file's permissions in octal. schema: type: integer…"
     },
     {
      "path": ".machine.files.path",
      "field": "path",
      "type": "string",
      "desc": "The path of the file."
     },
     {
      "path": ".machine.files.op",
      "field": "op",
      "type": "string",
      "desc": "The operation to use values…"
     }
    ]
   },
   {
    "path": ".machine.env",
    "field": "env",
    "type": "Env",
    "deprecated": true,
    "replacedBy": [
     "EnvironmentConfig"
    ],
    "note": "Deprecated: Use 'EnvironmentConfig' instead."
   },
   {
    "path": ".machine.time",
    "field": "time",
    "type": "*TimeConfig",
    "deprecated": true,
    "replacedBy": [
     "TimeSyncConfig"
    ],
    "note": "Deprecated: Use 'TimeSyncConfig' instead.",
    "children": [
     {
      "path": ".machine.time.disabled",
      "field": "disabled",
      "type": "*bool",
      "desc": "Indicates if the time service is disabled for the machine. Defaults to `false`."
     },
     {
      "path": ".machine.time.servers",
      "field": "servers",
      "type": "[]string",
      "desc": "Specifies time (NTP) servers to use for setting the system time. Defaults to `time.cloudflare.com`."
     },
     {
      "path": ".machine.time.bootTimeout",
      "field": "bootTimeout",
      "type": "time.Duration",
      "desc": "Specifies the timeout when the node time is considered to be in sync unlocking the boot sequence. NTP sync will be still running in the background."
     }
    ]
   },
   {
    "path": ".machine.sysctls",
    "field": "sysctls",
    "type": "map[string]string",
    "deprecated": true,
    "replacedBy": [
     "SysctlConfig"
    ],
    "note": "Deprecated: Use 'SysctlConfig' instead."
   },
   {
    "path": ".machine.sysfs",
    "field": "sysfs",
    "type": "map[string]string",
    "deprecated": true,
    "replacedBy": [
     "SysfsConfig"
    ],
    "note": "Deprecated: Use 'SysfsConfig' instead."
   },
   {
    "path": ".machine.registries",
    "field": "registries",
    "type": "RegistriesConfig",
    "deprecated": true,
    "replacedBy": [
     "RegistryAuthConfig",
     "RegistryMirrorConfig",
     "RegistryTLSConfig"
    ],
    "note": "Deprecated: Use `Registry*Config` instead.",
    "children": [
     {
      "path": ".machine.registries.mirrors",
      "field": "mirrors",
      "type": "map[string]*RegistryMirrorConfig",
      "desc": "Specifies mirror configuration for each registry host namespace. This setting allows to configure local pull-through caching registires, air-gapped installations, etc."
     },
     {
      "path": ".machine.registries.config",
      "field": "config",
      "type": "map[string]*RegistryConfig",
      "desc": "Specifies TLS & auth configuration for HTTPS image registries. Mutual TLS can be enabled with 'clientIdentity' option. The full hostname and port (if not using a default port 443) should be used as the key."
     }
    ]
   },
   {
    "path": ".machine.systemDiskEncryption",
    "field": "systemDiskEncryption",
    "type": "*SystemDiskEncryptionConfig",
    "deprecated": true,
    "replacedBy": [
     "VolumeConfig"
    ],
    "note": "Deprecated: Use `VolumeConfig` instead.",
    "children": [
     {
      "path": ".machine.systemDiskEncryption.state",
      "field": "state",
      "type": "*EncryptionConfig",
      "desc": "State partition encryption.",
      "children": [
       {
        "path": ".machine.systemDiskEncryption.state.provider",
        "field": "provider",
        "type": "string",
        "desc": "Encryption provider to use for the encryption."
       },
       {
        "path": ".machine.systemDiskEncryption.state.keys",
        "field": "keys",
        "type": "[]*EncryptionKey",
        "desc": "Defines the encryption keys generation and storage method."
       },
       {
        "path": ".machine.systemDiskEncryption.state.cipher",
        "field": "cipher",
        "type": "string",
        "desc": "Cipher kind to use for the encryption. Depends on the encryption provider."
       },
       {
        "path": ".machine.systemDiskEncryption.state.keySize",
        "field": "keySize",
        "type": "uint",
        "desc": "Defines the encryption key length."
       },
       {
        "path": ".machine.systemDiskEncryption.state.blockSize",
        "field": "blockSize",
        "type": "uint64",
        "desc": "Defines the encryption sector size."
       },
       {
        "path": ".machine.systemDiskEncryption.state.options",
        "field": "options",
        "type": "[]string",
        "desc": "Additional --perf parameters for the LUKS2 encryption. values: []string{\"no_read_workqueue\",\"no_write_workqueue\"}…"
       }
      ]
     },
     {
      "path": ".machine.systemDiskEncryption.ephemeral",
      "field": "ephemeral",
      "type": "*EncryptionConfig",
      "desc": "Ephemeral partition encryption.",
      "children": [
       {
        "path": ".machine.systemDiskEncryption.ephemeral.provider",
        "field": "provider",
        "type": "string",
        "desc": "Encryption provider to use for the encryption."
       },
       {
        "path": ".machine.systemDiskEncryption.ephemeral.keys",
        "field": "keys",
        "type": "[]*EncryptionKey",
        "desc": "Defines the encryption keys generation and storage method."
       },
       {
        "path": ".machine.systemDiskEncryption.ephemeral.cipher",
        "field": "cipher",
        "type": "string",
        "desc": "Cipher kind to use for the encryption. Depends on the encryption provider."
       },
       {
        "path": ".machine.systemDiskEncryption.ephemeral.keySize",
        "field": "keySize",
        "type": "uint",
        "desc": "Defines the encryption key length."
       },
       {
        "path": ".machine.systemDiskEncryption.ephemeral.blockSize",
        "field": "blockSize",
        "type": "uint64",
        "desc": "Defines the encryption sector size."
       },
       {
        "path": ".machine.systemDiskEncryption.ephemeral.options",
        "field": "options",
        "type": "[]string",
        "desc": "Additional --perf parameters for the LUKS2 encryption. values: []string{\"no_read_workqueue\",\"no_write_workqueue\"}…"
       }
      ]
     }
    ]
   },
   {
    "path": ".machine.features",
    "field": "features",
    "type": "*FeaturesConfig",
    "desc": "Features describe individual Talos features that can be switched on or off.",
    "children": [
     {
      "path": ".machine.features.rbac",
      "field": "rbac",
      "type": "*bool"
     },
     {
      "path": ".machine.features.stableHostname",
      "field": "stableHostname",
      "type": "*bool",
      "deprecated": true,
      "replacedBy": [
       "HostnameConfig"
      ],
      "note": "Deprecated: use HostConfig instead."
     },
     {
      "path": ".machine.features.kubernetesTalosAPIAccess",
      "field": "kubernetesTalosAPIAccess",
      "type": "*KubernetesTalosAPIAccessConfig",
      "deprecated": true,
      "replacedBy": [
       "KubeTalosAPIAccessConfig"
      ],
      "note": "Deprecated: use KubeTalosAPIAccessConfig instead.",
      "children": [
       {
        "path": ".machine.features.kubernetesTalosAPIAccess.enabled",
        "field": "enabled",
        "type": "*bool",
        "desc": "Enable Talos API access from Kubernetes pods."
       },
       {
        "path": ".machine.features.kubernetesTalosAPIAccess.allowedRoles",
        "field": "allowedRoles",
        "type": "[]string",
        "desc": "The list of Talos API roles which can be granted for access from Kubernetes pods. Empty list means that no roles can be granted, so access is blocked."
       },
       {
        "path": ".machine.features.kubernetesTalosAPIAccess.allowedKubernetesNamespaces",
        "field": "allowedKubernetesNamespaces",
        "type": "[]string",
        "desc": "The list of Kubernetes namespaces Talos API access is available from."
       }
      ]
     },
     {
      "path": ".machine.features.apidCheckExtKeyUsage",
      "field": "apidCheckExtKeyUsage",
      "type": "*bool"
     },
     {
      "path": ".machine.features.diskQuotaSupport",
      "field": "diskQuotaSupport",
      "type": "*bool",
      "desc": "Enable XFS project quota support for EPHEMERAL partition and user disks. Also enables kubelet tracking of ephemeral disk usage in the kubelet via quota."
     },
     {
      "path": ".machine.features.kubePrism",
      "field": "kubePrism",
      "type": "*KubePrism",
      "deprecated": true,
      "replacedBy": [
       "KubePrismConfig"
      ],
      "note": "Deprecated: Use KubePrismConfig document instead.",
      "children": [
       {
        "path": ".machine.features.kubePrism.enabled",
        "field": "enabled",
        "type": "*bool",
        "desc": "Enable KubePrism support - will start local load balancing proxy."
       },
       {
        "path": ".machine.features.kubePrism.port",
        "field": "port",
        "type": "int",
        "desc": "KubePrism port."
       }
      ]
     },
     {
      "path": ".machine.features.hostDNS",
      "field": "hostDNS",
      "type": "*HostDNSConfig",
      "deprecated": true,
      "replacedBy": [
       "ResolverConfig"
      ],
      "note": "Deprecated: Use ResolverConfig document instead.",
      "children": [
       {
        "path": ".machine.features.hostDNS.enabled",
        "field": "enabled",
        "type": "*bool",
        "desc": "Enable host DNS caching resolver."
       },
       {
        "path": ".machine.features.hostDNS.forwardKubeDNSToHost",
        "field": "forwardKubeDNSToHost",
        "type": "*bool",
        "desc": "Use the host DNS resolver as upstream for Kubernetes CoreDNS pods. When enabled, CoreDNS pods use host DNS server as the upstream DNS (instead of using configured upstream DNS resolvers directly)."
       },
       {
        "path": ".machine.features.hostDNS.resolveMemberNames",
        "field": "resolveMemberNames",
        "type": "*bool",
        "desc": "Resolve member hostnames using the host DNS resolver. When enabled, cluster member hostnames and node names are resolved using the host DNS resolver. This requires service discovery to be enabled."
       }
      ]
     },
     {
      "path": ".machine.features.imageCache",
      "field": "imageCache",
      "type": "*ImageCacheConfig",
      "deprecated": true,
      "replacedBy": [
       "ImageCacheConfig"
      ],
      "note": "Deprecated: Use ImageCacheConfig document instead.",
      "children": [
       {
        "path": ".machine.features.imageCache.localEnabled",
        "field": "localEnabled",
        "type": "*bool",
        "desc": "Enable local image cache."
       }
      ]
     },
     {
      "path": ".machine.features.nodeAddressSortAlgorithm",
      "field": "nodeAddressSortAlgorithm",
      "type": "string",
      "desc": "Select the node address sort algorithm. The 'v1' algorithm sorts addresses by the address itself. The 'v2' algorithm prefers more specific prefixes. If unset, defaults to 'v1'."
     }
    ]
   },
   {
    "path": ".machine.udev",
    "field": "udev",
    "type": "*UdevConfig",
    "deprecated": true,
    "replacedBy": [
     "UdevRulesConfig"
    ],
    "note": "Deprecated: Use `UdevRulesConfig` instead.",
    "desc": "Configures the udev system.",
    "children": [
     {
      "path": ".machine.udev.rules",
      "field": "rules",
      "type": "[]string",
      "deprecated": true,
      "replacedBy": [
       "UdevRulesConfig"
      ],
      "note": "Deprecated: Use `UdevRulesConfig` instead.",
      "desc": "List of udev rules to apply to the udev system…"
     }
    ]
   },
   {
    "path": ".machine.logging",
    "field": "logging",
    "type": "*LoggingConfig",
    "desc": "Configures the logging system.",
    "children": [
     {
      "path": ".machine.logging.destinations",
      "field": "destinations",
      "type": "[]LoggingDestination",
      "desc": "Logging destination.",
      "children": [
       {
        "path": ".machine.logging.destinations.endpoint",
        "field": "endpoint",
        "type": "*Endpoint",
        "desc": "Where to send logs. Supported protocols are \"tcp\" and \"udp\"."
       },
       {
        "path": ".machine.logging.destinations.format",
        "field": "format",
        "type": "string",
        "desc": "Logs format. values…"
       },
       {
        "path": ".machine.logging.destinations.extraTags",
        "field": "extraTags",
        "type": "map[string]string",
        "desc": "Extra tags (key-value) pairs to attach to every log message sent."
       }
      ]
     }
    ]
   },
   {
    "path": ".machine.kernel",
    "field": "kernel",
    "type": "*KernelConfig",
    "deprecated": true,
    "replacedBy": [
     "KernelModuleConfig"
    ],
    "note": "Deprecated: Use 'KernelModuleConfig' instead.",
    "children": [
     {
      "path": ".machine.kernel.modules",
      "field": "modules",
      "type": "[]*KernelModuleConfig",
      "deprecated": true,
      "replacedBy": [
       "KernelModuleConfig"
      ],
      "note": "Deprecated: Use multi-doc `KernelModuleConfig` instead.",
      "desc": "Kernel modules to load.",
      "children": [
       {
        "path": ".machine.kernel.modules.name",
        "field": "name",
        "type": "string",
        "deprecated": true,
        "replacedBy": [
         "KernelModuleConfig"
        ],
        "note": "Deprecated: Use multi-doc `KernelModuleConfig` instead.",
        "desc": "Module name."
       },
       {
        "path": ".machine.kernel.modules.parameters",
        "field": "parameters",
        "type": "[]string",
        "deprecated": true,
        "replacedBy": [
         "KernelModuleConfig"
        ],
        "note": "Deprecated: Use multi-doc `KernelModuleConfig` instead.",
        "desc": "Module parameters, changes applied after reboot."
       }
      ]
     }
    ]
   },
   {
    "path": ".machine.seccompProfiles",
    "field": "seccompProfiles",
    "type": "[]*MachineSeccompProfile",
    "desc": "Configures the seccomp profiles for the machine.",
    "children": [
     {
      "path": ".machine.seccompProfiles.name",
      "field": "name",
      "type": "string",
      "desc": "The `name` field is used to provide the file name of the seccomp profile."
     },
     {
      "path": ".machine.seccompProfiles.value",
      "field": "value",
      "type": "meta.Unstructured",
      "desc": "The `value` field is used to provide the seccomp profile. schema…"
     }
    ]
   },
   {
    "path": ".machine.baseRuntimeSpecOverrides",
    "field": "baseRuntimeSpecOverrides",
    "type": "meta.Unstructured",
    "deprecated": true,
    "replacedBy": [
     "CRIBaseRuntimeSpecConfig"
    ],
    "note": "Deprecated: Use the CRIBaseRuntimeSpecConfig configuration document instead."
   },
   {
    "path": ".machine.nodeLabels",
    "field": "nodeLabels",
    "type": "map[string]string",
    "deprecated": true,
    "replacedBy": [
     "KubeNodeConfig"
    ],
    "note": "Deprecated: use `KubeNodeConfig` instead."
   },
   {
    "path": ".machine.nodeAnnotations",
    "field": "nodeAnnotations",
    "type": "map[string]string",
    "deprecated": true,
    "replacedBy": [
     "KubeNodeConfig"
    ],
    "note": "Deprecated: use `KubeNodeConfig` instead."
   },
   {
    "path": ".machine.nodeTaints",
    "field": "nodeTaints",
    "type": "map[string]string",
    "deprecated": true,
    "replacedBy": [
     "KubeNodeConfig"
    ],
    "note": "Deprecated: use `KubeNodeConfig` instead."
   }
  ],
  "cluster": [
   {
    "path": ".cluster.id",
    "field": "id",
    "type": "string",
    "deprecated": true,
    "replacedBy": [
     "DiscoveryIdentityConfig"
    ],
    "note": "Deprecated: Use 'DiscoveryIdentityConfig' document instead."
   },
   {
    "path": ".cluster.secret",
    "field": "secret",
    "type": "string",
    "deprecated": true,
    "replacedBy": [
     "DiscoveryIdentityConfig"
    ],
    "note": "Deprecated: Use 'DiscoveryIdentityConfig' document instead."
   },
   {
    "path": ".cluster.controlPlane",
    "field": "controlPlane",
    "type": "*ControlPlaneConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeClusterConfig"
    ],
    "note": "Deprecated: Use `KubeClusterConfig` instead.",
    "children": [
     {
      "path": ".cluster.controlPlane.endpoint",
      "field": "endpoint",
      "type": "*Endpoint",
      "desc": "Endpoint is the canonical controlplane endpoint, which can be an IP address or a DNS hostname. It is single-valued, and may optionally include a port number."
     },
     {
      "path": ".cluster.controlPlane.localAPIServerPort",
      "field": "localAPIServerPort",
      "type": "int",
      "deprecated": true,
      "replacedBy": [
       "KubeAPIServerConfig"
      ],
      "note": "Deprecated: Use `KubeAPIServerConfig` instead."
     }
    ]
   },
   {
    "path": ".cluster.clusterName",
    "field": "clusterName",
    "type": "string",
    "deprecated": true,
    "replacedBy": [
     "KubeClusterConfig"
    ],
    "note": "Deprecated: Use `KubeClusterConfig` instead."
   },
   {
    "path": ".cluster.network",
    "field": "network",
    "type": "*ClusterNetworkConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeFlannelCNIConfig",
     "KubeNetworkConfig"
    ],
    "note": "Deprecated: Use `KubeNetworkConfig` and `KubeFlannelCNIConfig` instead.",
    "children": [
     {
      "path": ".cluster.network.cni",
      "field": "cni",
      "type": "*CNIConfig",
      "desc": "The CNI used. Composed of \"name\" and \"urls\". The \"name\" key supports the following options: \"flannel\", \"custom\", and \"none\". \"flannel\" uses Talos-managed Flannel CNI, and that's the default option.",
      "children": [
       {
        "path": ".cluster.network.cni.name",
        "field": "name",
        "type": "string",
        "desc": "Name of CNI to use. values…"
       },
       {
        "path": ".cluster.network.cni.urls",
        "field": "urls",
        "type": "[]string",
        "desc": "URLs containing manifests to apply for the CNI. Should be present for \"custom\", must be empty for \"flannel\" and \"none\"."
       },
       {
        "path": ".cluster.network.cni.flannel",
        "field": "flannel",
        "type": "*FlannelCNIConfig",
        "desc": "Flannel configuration options."
       }
      ]
     },
     {
      "path": ".cluster.network.dnsDomain",
      "field": "dnsDomain",
      "type": "string",
      "desc": "The domain used by Kubernetes DNS. The default is `cluster.local`…"
     },
     {
      "path": ".cluster.network.podSubnets",
      "field": "podSubnets",
      "type": "[]string",
      "desc": "The pod subnet CIDR. []string{\"10.244.0.0/16\"}…"
     },
     {
      "path": ".cluster.network.serviceSubnets",
      "field": "serviceSubnets",
      "type": "[]string",
      "desc": "The service subnet CIDR. []string{\"10.96.0.0/12\"}…"
     }
    ]
   },
   {
    "path": ".cluster.token",
    "field": "token",
    "type": "string",
    "desc": "The [bootstrap token](https://kubernetes.io/docs/reference/access-authn-authz/bootstrap-tokens/) used to join the cluster."
   },
   {
    "path": ".cluster.aescbcEncryptionSecret",
    "field": "aescbcEncryptionSecret",
    "type": "string",
    "deprecated": true,
    "replacedBy": [
     "KubeEtcdEncryptionConfig"
    ],
    "note": "Deprecated: Use `KubeEtcdEncryptionConfig` instead."
   },
   {
    "path": ".cluster.secretboxEncryptionSecret",
    "field": "secretboxEncryptionSecret",
    "type": "string",
    "deprecated": true,
    "replacedBy": [
     "KubeEtcdEncryptionConfig"
    ],
    "note": "Deprecated: Use `KubeEtcdEncryptionConfig` instead."
   },
   {
    "path": ".cluster.ca",
    "field": "ca",
    "type": "*x509.PEMEncodedCertificateAndKey",
    "deprecated": true,
    "replacedBy": [
     "KubeAPIServerCAConfig"
    ],
    "note": "Deprecated: Use `KubeAPIServerCAConfig` instead."
   },
   {
    "path": ".cluster.acceptedCAs",
    "field": "acceptedCAs",
    "type": "[]*x509.PEMEncodedCertificate",
    "deprecated": true,
    "replacedBy": [
     "KubeAPIServerCAConfig"
    ],
    "note": "Deprecated: Use `KubeAPIServerCAConfig` instead."
   },
   {
    "path": ".cluster.aggregatorCA",
    "field": "aggregatorCA",
    "type": "*x509.PEMEncodedCertificateAndKey",
    "deprecated": true,
    "replacedBy": [
     "KubeAggregatorCAConfig"
    ],
    "note": "Deprecated: Use `KubeAPIServerAggregatorCAConfig` instead."
   },
   {
    "path": ".cluster.serviceAccount",
    "field": "serviceAccount",
    "type": "*x509.PEMEncodedKey",
    "deprecated": true,
    "replacedBy": [
     "KubeServiceAccountConfig"
    ],
    "note": "Deprecated: Use `KubeServiceAccountConfig` instead."
   },
   {
    "path": ".cluster.apiServer",
    "field": "apiServer",
    "type": "*APIServerConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeAPIServerConfig"
    ],
    "note": "Deprecated: Use `KubeAPIServerConfig` instead.",
    "children": [
     {
      "path": ".cluster.apiServer.image",
      "field": "image",
      "type": "string",
      "desc": "The container image used in the API server manifest."
     },
     {
      "path": ".cluster.apiServer.extraArgs",
      "field": "extraArgs",
      "type": "meta.Args",
      "desc": "Extra arguments to supply to the API server. schema…"
     },
     {
      "path": ".cluster.apiServer.extraVolumes",
      "field": "extraVolumes",
      "type": "[]VolumeMountConfig",
      "desc": "Extra volumes to mount to the API server static pod.",
      "children": [
       {
        "path": ".cluster.apiServer.extraVolumes.hostPath",
        "field": "hostPath",
        "type": "string",
        "desc": "Path on the host."
       },
       {
        "path": ".cluster.apiServer.extraVolumes.mountPath",
        "field": "mountPath",
        "type": "string",
        "desc": "Path in the container."
       },
       {
        "path": ".cluster.apiServer.extraVolumes.readonly",
        "field": "readonly",
        "type": "bool",
        "desc": "Mount the volume read only."
       }
      ]
     },
     {
      "path": ".cluster.apiServer.env",
      "field": "env",
      "type": "Env",
      "desc": "The `env` field allows for the addition of environment variables for the control plane component."
     },
     {
      "path": ".cluster.apiServer.certSANs",
      "field": "certSANs",
      "type": "[]string",
      "desc": "Extra certificate subject alternative names for the API server's certificate."
     },
     {
      "path": ".cluster.apiServer.disablePodSecurityPolicy",
      "field": "disablePodSecurityPolicy",
      "type": "*bool"
     },
     {
      "path": ".cluster.apiServer.admissionControl",
      "field": "admissionControl",
      "type": "AdmissionPluginConfigList",
      "deprecated": true,
      "replacedBy": [
       "KubeAdmissionControlConfig"
      ],
      "note": "Deprecated: Use `KubeAdmissionControlConfig` instead."
     },
     {
      "path": ".cluster.apiServer.auditPolicy",
      "field": "auditPolicy",
      "type": "meta.Unstructured",
      "deprecated": true,
      "replacedBy": [
       "KubeAuditPolicyConfig"
      ],
      "note": "Deprecated: Use `KubeAuditPolicyConfig` instead."
     },
     {
      "path": ".cluster.apiServer.resources",
      "field": "resources",
      "type": "*ResourcesConfig",
      "desc": "Configure the API server resources. schema…"
     },
     {
      "path": ".cluster.apiServer.authorizationConfig",
      "field": "authorizationConfig",
      "type": "AuthorizationConfigAuthorizerConfigList",
      "deprecated": true,
      "replacedBy": [
       "KubeAuthorizerConfig"
      ],
      "note": "Deprecated: Use `KubeAuthorizerConfig` instead."
     }
    ]
   },
   {
    "path": ".cluster.controllerManager",
    "field": "controllerManager",
    "type": "*ControllerManagerConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeControllerManagerConfig"
    ],
    "note": "Deprecated: Use `KubeControllerManagerConfig` instead.",
    "children": [
     {
      "path": ".cluster.controllerManager.image",
      "field": "image",
      "type": "string",
      "desc": "The container image used in the controller manager manifest."
     },
     {
      "path": ".cluster.controllerManager.extraArgs",
      "field": "extraArgs",
      "type": "meta.Args",
      "desc": "Extra arguments to supply to the controller manager. schema…"
     },
     {
      "path": ".cluster.controllerManager.extraVolumes",
      "field": "extraVolumes",
      "type": "[]VolumeMountConfig",
      "desc": "Extra volumes to mount to the controller manager static pod.",
      "children": [
       {
        "path": ".cluster.controllerManager.extraVolumes.hostPath",
        "field": "hostPath",
        "type": "string",
        "desc": "Path on the host."
       },
       {
        "path": ".cluster.controllerManager.extraVolumes.mountPath",
        "field": "mountPath",
        "type": "string",
        "desc": "Path in the container."
       },
       {
        "path": ".cluster.controllerManager.extraVolumes.readonly",
        "field": "readonly",
        "type": "bool",
        "desc": "Mount the volume read only."
       }
      ]
     },
     {
      "path": ".cluster.controllerManager.env",
      "field": "env",
      "type": "Env",
      "desc": "The `env` field allows for the addition of environment variables for the control plane component."
     },
     {
      "path": ".cluster.controllerManager.resources",
      "field": "resources",
      "type": "*ResourcesConfig",
      "desc": "Configure the controller manager resources. schema…"
     }
    ]
   },
   {
    "path": ".cluster.proxy",
    "field": "proxy",
    "type": "*ProxyConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeProxyConfig"
    ],
    "note": "Deprecated: use `KubeProxyConfig` instead.",
    "children": [
     {
      "path": ".cluster.proxy.disabled",
      "field": "disabled",
      "type": "*bool",
      "desc": "Disable kube-proxy deployment on cluster bootstrap."
     },
     {
      "path": ".cluster.proxy.image",
      "field": "image",
      "type": "string",
      "desc": "The container image used in the kube-proxy manifest."
     },
     {
      "path": ".cluster.proxy.mode",
      "field": "mode",
      "type": "string",
      "desc": "proxy mode of kube-proxy. The default is 'iptables'."
     },
     {
      "path": ".cluster.proxy.extraArgs",
      "field": "extraArgs",
      "type": "meta.Args",
      "desc": "Extra arguments to supply to kube-proxy. schema…"
     }
    ]
   },
   {
    "path": ".cluster.scheduler",
    "field": "scheduler",
    "type": "*SchedulerConfig",
    "deprecated": true,
    "replacedBy": [
     "KubeSchedulerConfig"
    ],
    "note": "Deprecated: Use `KubeSchedulerConfig` instead.",
    "children": [
     {
      "path": ".cluster.scheduler.image",
      "field": "image",
      "type": "string",
      "desc": "The container image used in the scheduler manifest."
     },
     {
      "path": ".cluster.scheduler.extraArgs",
      "field": "extraArgs",
      "type": "meta.Args",
      "desc": "Extra arguments to supply to the scheduler. schema…"
     },
     {
      "path": ".cluster.scheduler.extraVolumes",
      "field": "extraVolumes",
      "type": "[]VolumeMountConfig",
      "desc": "Extra volumes to mount to the scheduler static pod.",
      "children": [
       {
        "path": ".cluster.scheduler.extraVolumes.hostPath",
        "field": "hostPath",
        "type": "string",
        "desc": "Path on the host."
       },
       {
        "path": ".cluster.scheduler.extraVolumes.mountPath",
        "field": "mountPath",
        "type": "string",
        "desc": "Path in the container."
       },
       {
        "path": ".cluster.scheduler.extraVolumes.readonly",
        "field": "readonly",
        "type": "bool",
        "desc": "Mount the volume read only."
       }
      ]
     },
     {
      "path": ".cluster.scheduler.env",
      "field": "env",
      "type": "Env",
      "desc": "The `env` field allows for the addition of environment variables for the control plane component."
     },
     {
      "path": ".cluster.scheduler.resources",
      "field": "resources",
      "type": "*ResourcesConfig",
      "desc": "Configure the scheduler resources. schema…"
     },
     {
      "path": ".cluster.scheduler.config",
      "field": "config",
      "type": "meta.Unstructured",
      "desc": "Specify custom kube-scheduler configuration. schema…"
     }
    ]
   },
   {
    "path": ".cluster.discovery",
    "field": "discovery",
    "type": "*ClusterDiscoveryConfig",
    "deprecated": true,
    "replacedBy": [
     "DiscoveryServiceConfig"
    ],
    "note": "Deprecated: Use 'DiscoveryServiceConfig' instead",
    "children": [
     {
      "path": ".cluster.discovery.enabled",
      "field": "enabled",
      "type": "*bool",
      "deprecated": true,
      "replacedBy": [
       "DiscoveryServiceConfig"
      ],
      "note": "Deprecated: Use 'DiscoveryServiceConfig' document instead.",
      "desc": "Enable the cluster membership discovery feature. Cluster discovery is based on individual registries which are configured under the registries field."
     },
     {
      "path": ".cluster.discovery.registries",
      "field": "registries",
      "type": "DiscoveryRegistriesConfig",
      "deprecated": true,
      "replacedBy": [
       "DiscoveryServiceConfig"
      ],
      "note": "Deprecated: Use 'DiscoveryServiceConfig' document instead.",
      "desc": "Configure registries used for cluster member discovery.",
      "children": [
       {
        "path": ".cluster.discovery.registries.kubernetes",
        "field": "kubernetes",
        "type": "RegistryKubernetesConfig",
        "deprecated": true,
        "replacedBy": [
         "DiscoveryServiceConfig"
        ],
        "note": "Deprecated: Use 'DiscoveryServiceConfig' document instead.",
        "desc": "Kubernetes registry uses Kubernetes API server to discover cluster members and stores additional information as annotations on the Node resources."
       },
       {
        "path": ".cluster.discovery.registries.service",
        "field": "service",
        "type": "RegistryServiceConfig",
        "deprecated": true,
        "replacedBy": [
         "DiscoveryServiceConfig"
        ],
        "note": "Deprecated: Use 'DiscoveryServiceConfig' document instead.",
        "desc": "Service registry is using an external service to push and pull information about cluster members."
       }
      ]
     }
    ]
   },
   {
    "path": ".cluster.etcd",
    "field": "etcd",
    "type": "*EtcdConfig",
    "desc": "Etcd specific configuration options.",
    "children": [
     {
      "path": ".cluster.etcd.image",
      "field": "image",
      "type": "string",
      "desc": "The container image used to create the etcd service."
     },
     {
      "path": ".cluster.etcd.ca",
      "field": "ca",
      "type": "*x509.PEMEncodedCertificateAndKey",
      "desc": "The `ca` is the root certificate authority of the PKI. It is composed of a base64 encoded `crt` and `key`."
     },
     {
      "path": ".cluster.etcd.extraArgs",
      "field": "extraArgs",
      "type": "meta.Args",
      "desc": "Extra arguments to supply to etcd. Note that the following args are not allowed: meta.Args{ \"initial-cluster\": meta.NewArgValue(\"https://1.2.3.4:2380\", nil), \"advertise-client-urls\":…"
     },
     {
      "path": ".cluster.etcd.subnet",
      "field": "subnet",
      "type": "string",
      "deprecated": true,
      "note": "Deprecated: use EtcdAdvertistedSubnets"
     },
     {
      "path": ".cluster.etcd.advertisedSubnets",
      "field": "advertisedSubnets",
      "type": "[]string",
      "desc": "The `advertisedSubnets` field configures the networks to pick etcd advertised IP from. IPs can be excluded from the list by using negative match with `!`, e.g `!10.0.0.0/8`."
     },
     {
      "path": ".cluster.etcd.listenSubnets",
      "field": "listenSubnets",
      "type": "[]string",
      "desc": "The `listenSubnets` field configures the networks for the etcd to listen for peer and client connections. If `listenSubnets` is not set, but `advertisedSubnets` is set, `listenSubnets` defaults to `advertisedSubnets`."
     }
    ]
   },
   {
    "path": ".cluster.coreDNS",
    "field": "coreDNS",
    "type": "*CoreDNS",
    "deprecated": true,
    "replacedBy": [
     "KubeCoreDNSConfig"
    ],
    "note": "Deprecated: Use `KubeCoreDNSConfig` instead.",
    "children": [
     {
      "path": ".cluster.coreDNS.disabled",
      "field": "disabled",
      "type": "*bool",
      "desc": "Disable coredns deployment on cluster bootstrap."
     },
     {
      "path": ".cluster.coreDNS.image",
      "field": "image",
      "type": "string",
      "desc": "The `image` field is an override to the default coredns image."
     }
    ]
   },
   {
    "path": ".cluster.externalCloudProvider",
    "field": "externalCloudProvider",
    "type": "*ExternalCloudProviderConfig",
    "desc": "External cloud provider configuration.",
    "children": [
     {
      "path": ".cluster.externalCloudProvider.enabled",
      "field": "enabled",
      "type": "*bool",
      "desc": "Enable external cloud provider. values…"
     },
     {
      "path": ".cluster.externalCloudProvider.manifests",
      "field": "manifests",
      "type": "[]string",
      "desc": "A list of urls that point to additional manifests for an external cloud provider. These will get automatically deployed as part of the bootstrap."
     }
    ]
   },
   {
    "path": ".cluster.extraManifests",
    "field": "extraManifests",
    "type": "[]string",
    "deprecated": true,
    "replacedBy": [
     "KubeExternalManifestConfig"
    ],
    "note": "Deprecated: Use `KubeExternalManifestConfig` instead."
   },
   {
    "path": ".cluster.extraManifestHeaders",
    "field": "extraManifestHeaders",
    "type": "map[string]string",
    "deprecated": true,
    "replacedBy": [
     "KubeExternalManifestConfig"
    ],
    "note": "Deprecated: Use `KubeExternalManifestConfig` instead."
   },
   {
    "path": ".cluster.inlineManifests",
    "field": "inlineManifests",
    "type": "ClusterInlineManifests",
    "deprecated": true,
    "replacedBy": [
     "KubeInlineManifestConfig"
    ],
    "note": "Deprecated: Use `KubeInlineManifestConfig` instead."
   },
   {
    "path": ".cluster.adminKubeconfig",
    "field": "adminKubeconfig",
    "type": "*AdminKubeconfigConfig",
    "desc": "Settings for admin kubeconfig generation. Certificate lifetime can be configured.",
    "children": [
     {
      "path": ".cluster.adminKubeconfig.certLifetime",
      "field": "certLifetime",
      "type": "time.Duration",
      "desc": "Admin kubeconfig certificate lifetime (default is 1 year). Field format accepts any Go time.Duration format ('1h' for one hour, '10m' for ten minutes)."
     }
    ]
   },
   {
    "path": ".cluster.allowSchedulingOnMasters",
    "field": "allowSchedulingOnMasters",
    "type": "*bool",
    "deprecated": true,
    "replacedBy": [
     "KubeNodeConfig"
    ],
    "note": "Deprecated: use `KubeNodeConfig` instead."
   },
   {
    "path": ".cluster.allowSchedulingOnControlPlanes",
    "field": "allowSchedulingOnControlPlanes",
    "type": "*bool",
    "deprecated": true,
    "replacedBy": [
     "KubeNodeConfig"
    ],
    "note": "Deprecated: use `KubeNodeConfig` instead."
   }
  ]
 }
};
