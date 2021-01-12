# cruise control

Cruise control is a piece of software that makes Linux TC easy! It comes with sane defaults where you
can just apply the included `iptables` rule set and define your internet speed in the configuration and it
will act as a cruise control system for your system.

The goal for this is to have an easy system that can be used to control the QoS settings
enforced by linux TC in an easy an accessible way.

The current statue of the project is: proof of concept

Currently for an small subset of classes and qdiscs this project seems to work
and the concept of it seems realistic. An improvement would be possible by
improving the "tree replace" code. Currently if a difference is detected, the
entire tree is replaced, while in reality only a subset of the tree can be
replaced.

For the moment, this project will remain as is, more implementation on TC filters
is first required before they can be implemented, so the focus is there.

## Configuration

The included config file `config.toml` is currently only used for testing
purposes.

## goals

- [x] apply a set of TC settings based on a configuration file
- [ ] include sane default iptables
- [ ] include sane default configuration
- [ ] allow the speed setting to be controlled through some means (HTTP call, API, ... TBD)
