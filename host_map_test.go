package hostmonitor_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"net/netip"
	"testing"
	"time"

	hostmonitor "github.com/rickbau5/host-monitor"
	"github.com/stretchr/testify/require"
)

func mustMAC(t *testing.T, mac string) net.HardwareAddr {
	m, err := net.ParseMAC(mac)
	require.NoError(t, err, "invalid MAC")
	return m
}

func mustIP(t *testing.T, ip string) netip.Addr {
	i, err := netip.ParseAddr(ip)
	require.NoError(t, err, "invalid IP")
	return i
}

// drain is generic because I felt like it and performance isn't important for this case.
func drain[T any](c <-chan T, num int) ([]T, error) {
	var cs []T
	for i := 0; i < num; i++ {
		read := func() error {
			ctx, done := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer done()

			select {
			case ac, ok := <-c:
				if !ok {
					if collected := len(cs); collected+1 < num {
						return fmt.Errorf("expected %d but only received %d", num, collected)
					}
				}

				cs = append(cs, ac)
			case <-ctx.Done():
				return fmt.Errorf("did not read %d items, only %d before context error: %w", num, len(cs), ctx.Err())
			}

			return nil
		}

		if err := read(); err != nil {
			return cs, err
		}
	}

	// we know we read the correct amount at this point
	return cs, nil
}

func TestHostMap_ResetAndLoad(t *testing.T) {
	testMAC1 := mustMAC(t, "1A:1A:1A:1A:1A:1A")
	hm := hostmonitor.NewHostMap()

	hm.UpdateAddresses([]hostmonitor.Addr{
		{
			MAC: testMAC1,
			IP:  mustIP(t, "192.168.1.2"),
		},
	})

	_, err := drain(hm.Notifications(), 1)
	require.NoError(t, err)

	hm.ResetAndLoad([]hostmonitor.Addr{
		{
			MAC: testMAC1,
			IP:  mustIP(t, "192.168.1.2"),
		},
	})

	notifications, err := drain(hm.Notifications(), 1)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))

	// update a pre loaded mac, no change
	hm.UpdateAddresses([]hostmonitor.Addr{
		{
			MAC: testMAC1,
			IP:  mustIP(t, "192.168.1.2"),
		},
	})

	notifications, err = drain(hm.Notifications(), 1)
	require.Error(t, err)
	require.True(t, errors.Is(err, context.DeadlineExceeded))

	// change a mac
	hm.UpdateAddresses([]hostmonitor.Addr{
		{
			MAC: testMAC1,
			IP:  mustIP(t, "192.168.1.3"),
		},
	})

	notifications, err = drain(hm.Notifications(), 1)
	require.NoError(t, err)

	expected := hostmonitor.Change{
		ChangeType: hostmonitor.IPChange,
		Addr: hostmonitor.Addr{
			MAC: testMAC1,
			IP:  mustIP(t, "192.168.1.3"),
		},
		Online: true,
		PreviousAddr: &hostmonitor.Addr{
			MAC: testMAC1,
			IP:  mustIP(t, "192.168.1.2"),
		},
	}

	actual := notifications[0]
	actual.LastSeen = time.Time{}
	assert.Equal(t, actual, expected)
}

func TestHostMap_UpdateAddresses(t *testing.T) {
	testMAC1 := mustMAC(t, "1A:1A:1A:1A:1A:1A")
	testMAC2 := mustMAC(t, "2B:2B:2B:2B:2B:2B")
	testMAC3 := mustMAC(t, "3C:3C:3C:3C:3C:3C")

	// Note: test cases should not generate more than the amount of notifications the internal channel for HostMap can,
	//  excess notifications will be dropped by the HostMap and therefor are not testable.
	testCases := []struct {
		name            string
		hm              *hostmonitor.HostMap
		addrs           []hostmonitor.Addr
		expectedChanges []hostmonitor.Change
	}{
		{
			name: "coming online",
			hm:   hostmonitor.NewHostMap(),
			addrs: []hostmonitor.Addr{
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
			},
			expectedChanges: []hostmonitor.Change{
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online:       true,
					PreviousAddr: nil,
					LastSeen:     time.Time{}, // will ignore
				},
			},
		},
		{
			name: "not changing ip",
			hm:   hostmonitor.NewHostMap(),
			addrs: []hostmonitor.Addr{
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
			},
			expectedChanges: []hostmonitor.Change{
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online:       true,
					PreviousAddr: nil,
					LastSeen:     time.Time{}, // will ignore
				},
			},
		},
		{
			name: "changing ip",
			hm:   hostmonitor.NewHostMap(),
			addrs: []hostmonitor.Addr{
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.3"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.4"),
				},
			},
			expectedChanges: []hostmonitor.Change{
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online:       true,
					PreviousAddr: nil,
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.4"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
				},
			},
		},
		{
			name: "changing ip back",
			hm:   hostmonitor.NewHostMap(),
			addrs: []hostmonitor.Addr{
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.3"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
			},
			expectedChanges: []hostmonitor.Change{
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online:       true,
					PreviousAddr: nil,
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
				},
			},
		},
		{
			name: "changing ips new and old",
			hm:   hostmonitor.NewHostMap(),
			addrs: []hostmonitor.Addr{
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.3"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.4"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.3"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.4"),
				},
			},
			expectedChanges: []hostmonitor.Change{
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online:       true,
					PreviousAddr: nil,
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.4"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.4"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.3"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.4"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
				},
			},
		},

		// different macs
		{
			name: "multiple MACs",
			hm:   hostmonitor.NewHostMap(),
			addrs: []hostmonitor.Addr{
				{
					MAC: testMAC1,
					IP:  mustIP(t, "192.168.1.2"),
				},
				{
					MAC: testMAC2,
					IP:  mustIP(t, "192.168.1.100"),
				},
				{
					MAC: testMAC3,
					IP:  mustIP(t, "192.168.1.200"),
				},
				{
					MAC: testMAC2,
					IP:  mustIP(t, "192.168.1.200"),
				},
				{
					MAC: testMAC3,
					IP:  mustIP(t, "192.168.1.100"),
				},
			},
			expectedChanges: []hostmonitor.Change{
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC1,
						IP:  mustIP(t, "192.168.1.2"),
					},
					Online:       true,
					PreviousAddr: nil,
				},
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC2,
						IP:  mustIP(t, "192.168.1.100"),
					},
					Online:       true,
					PreviousAddr: nil,
				},
				{
					ChangeType: hostmonitor.OnlineChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC3,
						IP:  mustIP(t, "192.168.1.200"),
					},
					Online:       true,
					PreviousAddr: nil,
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC2,
						IP:  mustIP(t, "192.168.1.200"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC2,
						IP:  mustIP(t, "192.168.1.100"),
					},
				},
				{
					ChangeType: hostmonitor.IPChange,
					Addr: hostmonitor.Addr{
						MAC: testMAC3,
						IP:  mustIP(t, "192.168.1.100"),
					},
					Online: true,
					PreviousAddr: &hostmonitor.Addr{
						MAC: testMAC3,
						IP:  mustIP(t, "192.168.1.200"),
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			tc.hm.UpdateAddresses(tc.addrs)
			changes, err := drain(tc.hm.Notifications(), len(tc.expectedChanges))
			require.NoError(t, err, "error while draining")

			for i := 0; i < len(changes); i++ {
				assert.NotEmpty(t, changes[i].LastSeen)
				changes[i].LastSeen = time.Time{}
			}

			if !assert.Equal(t, tc.expectedChanges, changes) {
				for i, expected := range tc.expectedChanges {
					if len(changes) <= i {
						break
					}
					t.Logf("change: %s", changes[i].String())
					t.Logf("expected: %s", expected.String())
					require.Equalf(t, expected.ChangeType.String(), changes[i].ChangeType.String(), "change %d invalid", i+1)
					require.Equalf(t, expected.Addr.String(), changes[i].Addr.String(), "change %d invalid", i+1)
					require.Equalf(t, expected.Online, changes[i].Online, "change %d invalid", i+1)
					if expected.PreviousAddr != nil && assert.NotNil(t, changes[i].PreviousAddr, "change %d invalid", i+1) {
						require.Equal(t, expected.PreviousAddr.String(), changes[i].PreviousAddr.String(), "change %d invalid", i+1)
					} else {
						require.Nil(t, changes[i].PreviousAddr, "change %d invalid", i+1)
					}
				}
			}
		})
	}
}
