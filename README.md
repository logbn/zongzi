# Zongzi

A UDP discovery protocol and cluster control system for Dragonboat.

- [x] Plug-n-play UDP Multicast Discovery Protocol
- [x] Global nodehost registry
- [x] Global shard registry
- [x] Cluster controller
- [x] Host controller

## Work in progress

Note that this repository represents a very crude proof of concept.  
Expect all aspects of this API to change.

There are basically no tests.  
A lot of refactoring remains to make this API stable and correct.  
Testing will commence in earnest after the API stabilizes.

The primary goal of this repository is to completely wrap dragonboat behind a facade that presents a simpler interface 
with a lot of the complex tasks handled automatically with good defaults. It will be opinionated by design and is may
not support some Dragonboat features, even in its final form.
