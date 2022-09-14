package on_chain_tx

import (
	"github.com/lightningnetwork/lnd/lnrpc"
	"reflect"
	"testing"
)

func Test_processSendRequest(t *testing.T) {
	var targetConf int32 = 10
	var satPerVbyte uint64 = 11
	var amount int64 = 14
	var sendAll = true
	var label = "test"
	var minConfs int32 = 15
	var spendUnco = true

	tests := []struct {
		name    string
		input   sendCoinsRequest
		want    lnrpc.SendCoinsRequest
		wantErr bool
	}{
		{
			"Missing node ID",
			sendCoinsRequest{
				Addr:      "adadsdas",
				AmountSat: 12,
			},
			lnrpc.SendCoinsRequest{
				Addr:   "adadsdas",
				Amount: 12,
			},
			true,
		},
		{
			"Address not provided",
			sendCoinsRequest{
				NodeId:    1,
				Addr:      "",
				AmountSat: 12,
			},
			lnrpc.SendCoinsRequest{
				Addr:   "",
				Amount: 12,
			},
			true,
		},
		{
			"Invalid amount",
			sendCoinsRequest{
				NodeId:    1,
				Addr:      "test",
				AmountSat: 0,
			},
			lnrpc.SendCoinsRequest{
				Addr:   "test",
				Amount: 0,
			},
			true,
		},
		{
			"Both targetconf and satpervbyte provided",
			sendCoinsRequest{
				NodeId:      1,
				Addr:        "test",
				AmountSat:   12,
				TargetConf:  &targetConf,
				SatPerVbyte: &satPerVbyte,
			},
			lnrpc.SendCoinsRequest{
				Addr:        "",
				Amount:      0,
				TargetConf:  0,
				SatPerVbyte: 0,
			},
			true,
		},
		{
			"Only mandatory params",
			sendCoinsRequest{
				NodeId:    1,
				Addr:      "test",
				AmountSat: amount,
			},
			lnrpc.SendCoinsRequest{
				Addr:   "test",
				Amount: 14,
			},
			false,
		},
		{
			"All params",
			sendCoinsRequest{
				NodeId:           1,
				Addr:             "test",
				AmountSat:        amount,
				TargetConf:       &targetConf,
				SendAll:          &sendAll,
				Label:            &label,
				MinConfs:         &minConfs,
				SpendUnconfirmed: &spendUnco,
			},
			lnrpc.SendCoinsRequest{
				Addr:             "test",
				Amount:           14,
				TargetConf:       10,
				SendAll:          true,
				Label:            "test",
				MinConfs:         15,
				SpendUnconfirmed: true,
			},
			false,
		},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := processSendRequest(test.input)

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
