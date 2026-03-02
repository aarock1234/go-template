package client

import (
	"math/rand/v2"

	"github.com/saucesteals/fhttp/http2"
)

type http2Fingerprint struct {
	InitialWindowSize uint32
	TransportConnFlow uint32
	HeaderPriority    *http2.PriorityParam
	Settings          []http2.Setting
}

var (
	HTTP2FingerprintMac = http2Fingerprint{
		InitialWindowSize: 4194304,
		TransportConnFlow: 10485760,
		HeaderPriority: &http2.PriorityParam{
			Weight: 254,
		},
		Settings: []http2.Setting{
			{
				ID:  http2.SettingEnablePush,
				Val: 0,
			},
			{
				ID:  http2.SettingInitialWindowSize,
				Val: 4194304,
			},
			{
				ID:  http2.SettingMaxConcurrentStreams,
				Val: 100,
			},
		},
	}

	HTTP2FingerprintWindows = http2Fingerprint{
		InitialWindowSize: 65535,
		TransportConnFlow: 15663105,
		HeaderPriority: &http2.PriorityParam{
			Weight: 255,
		},
		Settings: []http2.Setting{
			{
				ID:  http2.SettingEnablePush,
				Val: 0,
			},
			{
				ID:  http2.SettingInitialWindowSize,
				Val: 6291456,
			},
			{
				ID:  http2.SettingMaxHeaderListSize,
				Val: 262144,
			},
		},
	}
)

func (f http2Fingerprint) Configure(transport *http2.Transport) {
	transport.InitialWindowSize = f.InitialWindowSize
	transport.TransportConnFlow = f.TransportConnFlow
	transport.HeaderPriority = f.HeaderPriority
	transport.Settings = f.Settings
}

const (
	minInitialWindowSize = 65535
	maxInitialWindowSize = 6291456

	minTransportConnFlow = 10485760
	maxTransportConnFlow = 15663105

	minHeaderPriority = 254
	maxHeaderPriority = 255
)

func RandomHTTP2Fingerprint() http2Fingerprint {
	initialWindowSize := rand.Uint32N(maxInitialWindowSize-minInitialWindowSize) + minInitialWindowSize
	transportConnFlow := rand.Uint32N(maxTransportConnFlow-minTransportConnFlow) + minTransportConnFlow
	headerPriority := rand.IntN(maxHeaderPriority-minHeaderPriority) + minHeaderPriority

	return http2Fingerprint{
		InitialWindowSize: initialWindowSize,
		TransportConnFlow: transportConnFlow,
		HeaderPriority: &http2.PriorityParam{
			Weight: uint8(headerPriority),
		},
		Settings: []http2.Setting{
			{
				ID:  http2.SettingEnablePush,
				Val: 0,
			},
			{
				ID:  http2.SettingInitialWindowSize,
				Val: initialWindowSize,
			},
			{
				ID:  http2.SettingMaxHeaderListSize,
				Val: transportConnFlow,
			},
		},
	}
}
