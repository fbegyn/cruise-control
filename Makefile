setup-test:
	ip l add dev test-01 type dummy
	ip l add dev test-02 type dummy

clean-test:
	ip l del dev test-01 type dummy
	ip l del dev test-02 type dummy

reset-test: clean-test setup-test

reset:
	tc qdisc del dev test-01 root
	tc qdisc del dev test-02 root

show-qd:
	tc qdisc show dev test-01
	tc qdisc show dev test-02

show-cl:
	tc class show dev test-01
	tc class show dev test-02

show: show-qd show-cl
