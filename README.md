# host-monitor
Expiremental tools for monitoring hosts on the network. These were developed for learning and experimenting with my network
and aren't intended for any use outside of that.

Spoiler `sniffer2` is the preferred tool.

## Tools

### arpmon
Periodically load the arp table and compute changes to hosts (offline, online, ip changes). Does not compile on macOS.

#### Usage
```bash
$ arpmon -i <interface>
```

#### Example Log
```log
2022/06/23 19:22:54 "level"=0 "msg"="current table" "mac"="..." "members"=["mac=(...) ip=(192.168.1.250) port=(0) lastSeen=(2022-06-23T19:22:54-07:00)"] "manufacturer"="Google"
2022/06/23 19:22:54 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.250) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:22:54.962546916 -0700 PDT m=+0.355961298)
```

### sniffer
Use [irai/packet](https://github.com/irai/packet) to monitor for DHCP broadcasts for hosts coming online or offline. Does not
compile on macOS. 

#### Usage
```bash
$ sniffer -i <interface>
```

#### Example Log
```log
022/06/23 19:22:54 "level"=0 "msg"="current table" "mac"="..." "members"=["mac=(...) ip=(192.168.1.250) port=(0) lastSeen=(2022-06-23T19:22:54-07:00)"] "manufacturer"="Google"
2022/06/23 19:22:54 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.250) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:22:54.962546916 -0700 PDT m=+0.355961298)
2022/06/23 19:23:09 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.14) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:23:09.967858563 -0700 PDT m=+15.361272424)
2022/06/23 19:23:24 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.95) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:23:24.977653195 -0700 PDT m=+30.371067004)
2022/06/23 19:23:39 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.174) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:23:39.967705921 -0700 PDT m=+45.361119678)
2022/06/23 19:23:54 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.214) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:23:54.978907095 -0700 PDT m=+60.372320904)
2022/06/23 19:24:09 change detected: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.13) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:24:09.96705628 -0700 PDT m=+75.360469881)
```


### sniffer2
Use [google/gopacket](https://github.com/google/gopacket) to monitor packets coming across an interface. The combination
of the learnings from `aprmon` and `sniffer`. Compiles for both macOS and ARM devices (testing on my Raspberry Pi 3). Combines
the host name option from DHCPv4 broadcasts with the traffic of all packets across the interface to track host activity and 
collect host names. 

Depending on the capabilities of your device you may be able to see DHCP broadcast replies as well (denoted by `dhcp(2)`).

#### Usage
```bash
$ sniffer2 -i <interface>
```

#### Example Log
```log
2022/06/23 19:06:35 host '' changed: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.65) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:06:35.433000921 -0700 PDT m=+1129.570742588)
2022/06/23 19:06:49 host '' changed: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.232) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:06:49.074255259 -0700 PDT m=+1143.211996822)
2022/06/23 19:07:18 host '' changed: change=(offline) online=(false) addr=(mac=(...) ip=(192.168.1.14) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:02:18.847651744 -0700 PDT m=+872.985393463)
2022/06/23 19:07:39 host '' changed: change=(offline) online=(false) addr=(mac=(...) ip=(192.168.1.167) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:02:37.528613261 -0700 PDT m=+891.666354876)
2022/06/23 19:10:31 dhcp(1) from ...(ip=0.0.0.0), hostname=(foo), manufacturer=(Apple)
2022/06/23 19:10:33 dhcp(1) from ...(ip=0.0.0.0), hostname=(foo), manufacturer=(Apple)
2022/06/23 19:10:36 dhcp(1) from ...(ip=0.0.0.0), hostname=(foo), manufacturer=(Apple)
2022/06/23 19:10:37 host 'foo' changed: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.95) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:10:37.977827023 -0700 PDT m=+1372.115568637)
2022/06/23 19:10:38 host '' changed: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.167) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:10:38.360982042 -0700 PDT m=+1372.498723709)
2022/06/23 19:10:38 host '' changed: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.203) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:10:38.389072738 -0700 PDT m=+1372.526814665)
2022/06/23 19:10:41 host '' changed: change=(online) online=(true) addr=(mac=(...) ip=(192.168.1.14) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:10:41.724847837 -0700 PDT m=+1375.862589400)
2022/06/23 19:10:44 host '' changed: change=(ip change) online=(true) addr=(mac=(...) ip=(192.168.1.31) port=(0)) previousAddr=(mac=(...) ip=(192.168.1.223) port=(0)) lastSeen=(2022-06-23 19:10:44.998192473 -0700 PDT m=+1379.135933931)
2022/06/23 19:11:35 host '' changed: change=(offline) online=(false) addr=(mac=(...) ip=(192.168.1.65) port=(0)) previousAddr=(<nil>) lastSeen=(2022-06-23 19:06:35.433000921 -0700 PDT m=+1129.570742588)
2022/06/23 19:13:06 dhcp(1) from ...(ip=0.0.0.0), hostname=(<unknown>), manufacturer=(Nintendo)
2022/06/23 19:13:06 dhcp(1) from ...(ip=0.0.0.0), hostname=(<unknown>), manufacturer=(Nintendo)
```

## Building the Tools
Build the tools using `make`. The default target builds the binaries for 32bit ARM, for my use with the Raspberry Pi 3. It can easily be changed to compile
for other platforms. Or you can run the commands with the Go tool, e.g. `go run ./cmd/sniffer2/`. 

The `go` command line tool is required to compile the binaries.
