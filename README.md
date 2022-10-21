# Zongzi

A UDP discovery protocol and cluster control system for Dragonboat.

- [x] Plug-n-play UDP Multicast Discovery Protocol
- [x] Global nodehost registry
- [x] Global shard registry
- [ ] Cluster controller
- [ ] Event system

Every nodehost receives a materialized view of the global cluster state.
NodeHostInfo is collected and replicated for every nodehost in the cluster.
