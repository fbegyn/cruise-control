# cruise control

Cruise control is a piece of software that makes Linux TC easy! It comes with sane defaults where you
can just apply the included `iptables` rule set and define your internet speed in the configuration and it
will act as a cruise control system for your system.

The goal for this is to have an easy system that can be used to control the QoS settings
enforced by linux TC in an easy an accessible way.

## goals

- [ ] apply a set of TC settings based on a configuration file
- [ ] include sane default iptables
- [ ] include sane default configuration
- [ ] allow the speed setting to be controlled through some means (HTTP call, API, ... TBD)
