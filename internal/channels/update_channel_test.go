package channels

import (
	"github.com/lightningnetwork/lnd/lnrpc"
	"reflect"
	"testing"
)

func Test_processChannelPoint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *lnrpc.PolicyUpdateRequest_ChanPoint
		wantErr bool
	}{
		{
			"Test correct chann point",
			"e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce:0",
			&lnrpc.PolicyUpdateRequest_ChanPoint{
				ChanPoint: &lnrpc.ChannelPoint{
					FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{
						FundingTxidStr: "e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce",
					},
					OutputIndex: 0,
				},
			},
			false,
		},
		{
			"Test wrong chann point format",
			"e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce0",
			&lnrpc.PolicyUpdateRequest_ChanPoint{},
			true,
		},
		{
			"Output Index not int",
			"e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce:X",
			&lnrpc.PolicyUpdateRequest_ChanPoint{},
			true,
		},
	}

	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := processChannelPoint(test.input)
			if err != nil {
				if test.wantErr {
					return
				}
				t.Errorf("processChannelPoint error: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%d: processChannelPoint()\nGot:\n%v\nWant:\n%v\n", i, got, test.want)
			}
		})
	}
}

func Test_createPolicyRequest(t *testing.T) {
	var noChanPoint *string

	var feeRatePpm uint32 = 11
	var baseFeeMsat int64 = 12
	var maxHtlcMsat uint64 = 13
	var minHtlcMsat uint64 = 14

	chanPoint := "e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce:0"

	tests := []struct {
		name    string
		input   updateChanRequestBody
		want    lnrpc.PolicyUpdateRequest
		wantErr bool
	}{
		{
			"Missing node Id",
			updateChanRequestBody{
				NodeId:        1,
				ChannelPoint:  noChanPoint,
				TimeLockDelta: 18,
			},
			lnrpc.PolicyUpdateRequest{
				Scope:         &lnrpc.PolicyUpdateRequest_Global{Global: true},
				TimeLockDelta: 18,
			},
			true,
		},
		{
			"Channel point not provided - update all",
			updateChanRequestBody{
				NodeId:        1,
				ChannelPoint:  noChanPoint,
				TimeLockDelta: 18,
			},
			lnrpc.PolicyUpdateRequest{
				Scope:         &lnrpc.PolicyUpdateRequest_Global{Global: true},
				TimeLockDelta: 18,
			},
			false,
		},
		{
			"Channel point provided - update one",
			updateChanRequestBody{
				NodeId:        1,
				ChannelPoint:  &chanPoint,
				TimeLockDelta: 18,
			},
			lnrpc.PolicyUpdateRequest{
				Scope: &lnrpc.PolicyUpdateRequest_ChanPoint{
					ChanPoint: &lnrpc.ChannelPoint{
						FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{
							FundingTxidStr: "e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce",
						},
						OutputIndex: 0,
					},
				},
				TimeLockDelta: 18,
			},
			false,
		},
		{
			"TimeLockDelta < 18",
			updateChanRequestBody{
				NodeId:        1,
				ChannelPoint:  noChanPoint,
				TimeLockDelta: 0,
			},
			lnrpc.PolicyUpdateRequest{
				Scope:         &lnrpc.PolicyUpdateRequest_Global{Global: true},
				TimeLockDelta: 18,
			},
			false,
		},
		{
			"All params provided",
			updateChanRequestBody{
				NodeId:        1,
				ChannelPoint:  &chanPoint,
				FeeRatePpm:    &feeRatePpm,
				BaseFeeMsat:   &baseFeeMsat,
				MaxHtlcMsat:   &maxHtlcMsat,
				MinHtlcMsat:   &minHtlcMsat,
				TimeLockDelta: 18,
			},
			lnrpc.PolicyUpdateRequest{
				Scope: &lnrpc.PolicyUpdateRequest_ChanPoint{
					ChanPoint: &lnrpc.ChannelPoint{
						FundingTxid: &lnrpc.ChannelPoint_FundingTxidStr{
							FundingTxidStr: "e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce",
						},
						OutputIndex: 0,
					},
				},
				BaseFeeMsat:          int64(12),
				FeeRatePpm:           uint32(11),
				TimeLockDelta:        18,
				MaxHtlcMsat:          uint64(13),
				MinHtlcMsat:          uint64(14),
				MinHtlcMsatSpecified: true,
			},
			false,
		},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := createPolicyRequest(test.input)
			if err != nil {
				if test.wantErr {
					return
				}
				t.Errorf("createPolicyRequest error: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%d: createPolicyRequest()\nGot:\n%v\nWant:\n%v\n", i, got, test.want)
			}
		})
	}
}

func Test_processUpdateResponse(t *testing.T) {
	var noFailedUpdSlice []failedUpdate
	txidByte := []byte{206, 199, 33, 7, 91, 12, 79, 57, 217, 192, 219, 244, 131, 232, 102, 160, 188,
		3, 67, 142, 26, 122, 16, 45, 156, 23, 62, 240, 213, 240, 59, 228}
	failedUpdSlice := []failedUpdate{
		{
			OutPoint: struct {
				Txid    string
				OutIndx uint32
			}{"e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce", 0},
			Reason:      "not found",
			UpdateError: "not found",
		},
	}

	tests := []struct {
		name  string
		input *lnrpc.PolicyUpdateResponse
		want  updateResponse
	}{
		{
			"Update succeeded",
			&lnrpc.PolicyUpdateResponse{FailedUpdates: []*lnrpc.FailedUpdate{}},
			updateResponse{
				Status:        "Channel/s updated",
				FailedUpdates: noFailedUpdSlice,
			},
		},
		{
			"Update failed",
			&lnrpc.PolicyUpdateResponse{FailedUpdates: []*lnrpc.FailedUpdate{{
				Outpoint: &lnrpc.OutPoint{
					TxidBytes:   txidByte,
					TxidStr:     "e43bf0d5f03e179c2d107a1a8e4303bca066e883f4dbc0d9394f0c5b0721c7ce",
					OutputIndex: 0,
				},
				Reason:      2,
				UpdateError: "not found",
			}}},
			updateResponse{
				Status:        "Channel/s update failed",
				FailedUpdates: failedUpdSlice,
			},
		},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := processUpdateResponse(test.input)

			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%d: processUpdateResponse()\nGot:\n%v\nWant:\n%v\n", i, got, test.want)
			}
		})
	}
}
