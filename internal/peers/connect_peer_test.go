package peers

import (
	"github.com/lightningnetwork/lnd/lnrpc"
	"reflect"
	"testing"
)

func Test_processRequest(t *testing.T) {
	addr := LndAddress{
		PubKey: "03003a3c4df03c5a980589626a69c955126c828d51a58f700ef1c64e03bf3030b0",
		Host:   "127.0.0.1:10010",
	}
	var perm = true
	var timeOut uint64 = 10

	tests := []struct {
		name    string
		input   ConnectPeerRequest
		want    lnrpc.ConnectPeerRequest
		wantErr bool
	}{
		{
			"Missing node ID",
			ConnectPeerRequest{
				LndAddress: LndAddress{},
				Perm:       nil,
				TimeOut:    nil,
			},
			lnrpc.ConnectPeerRequest{
				Addr: &lnrpc.LightningAddress{
					Pubkey: "03003a3c4df03c5a980589626a69c955126c828d51a58f700ef1c64e03bf3030b0",
					Host:   "127.0.0.1:10010",
				},
				Perm:    false,
				Timeout: 0,
			},
			true,
		},
		{
			"Missing lnd address - pubkey",
			ConnectPeerRequest{
				NodeId: 1,
				LndAddress: LndAddress{
					PubKey: "",
					Host:   "127.0.0.1:10010",
				},
				Perm:    nil,
				TimeOut: nil,
			},
			lnrpc.ConnectPeerRequest{
				Addr: &lnrpc.LightningAddress{
					Pubkey: "",
					Host:   "",
				},
				Perm:    false,
				Timeout: 0,
			},
			true,
		},
		{
			"Missing lnd address - host",
			ConnectPeerRequest{
				NodeId: 1,
				LndAddress: LndAddress{
					PubKey: "03003a3c4df03c5a980589626a69c955126c828d51a58f700ef1c64e03bf3030b0",
					Host:   "",
				},
				Perm:    nil,
				TimeOut: nil,
			},
			lnrpc.ConnectPeerRequest{
				Addr: &lnrpc.LightningAddress{
					Pubkey: "",
					Host:   "",
				},
				Perm:    false,
				Timeout: 0,
			},
			true,
		},
		{
			"All params provided",
			ConnectPeerRequest{
				NodeId:     1,
				LndAddress: addr,
				Perm:       &perm,
				TimeOut:    &timeOut,
			},
			lnrpc.ConnectPeerRequest{
				Addr: &lnrpc.LightningAddress{
					Pubkey: "03003a3c4df03c5a980589626a69c955126c828d51a58f700ef1c64e03bf3030b0",
					Host:   "127.0.0.1:10010",
				},
				Perm:    true,
				Timeout: 10,
			},
			false,
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := processRequest(test.input)

			if err != nil {
				if test.wantErr {
					return
				}
				t.Errorf("processSendRequest: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%d: processSendRequest()\nGot:\n%v\nWant:\n%v\n", i, got, test.want)
			}
		})
	}
}
