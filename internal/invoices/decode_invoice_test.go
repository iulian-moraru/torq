package invoices

import (
	"github.com/lightningnetwork/lnd/lnrpc"
	"reflect"
	"testing"
)

func Test_constructDecodedInvoice(t *testing.T) {
	type args struct {
		decoded *lnrpc.PayReq
	}
	tests := []struct {
		name string
		args args
		want *DecodedInvoice
	}{
		{
			name: "Decodes a full decoded invoice",
			args: args{
				decoded: &lnrpc.PayReq{
					Destination:     "test",
					PaymentHash:     "payment_hash",
					NumSatoshis:     100,
					Timestamp:       1662812778,
					Expiry:          120,
					Description:     "Some description encoded in the invoice",
					DescriptionHash: "",
					FallbackAddr:    "something",
					CltvExpiry:      124,
					RouteHints: []*lnrpc.RouteHint{{
						HopHints: []*lnrpc.HopHint{{
							NodeId:                    "routing_hint_node_id_1",
							ChanId:                    72623859790382856,
							FeeBaseMsat:               12,
							FeeProportionalMillionths: 12000,
							CltvExpiryDelta:           3,
						},
							{
								NodeId:                    "routing_hint_node_id_2",
								ChanId:                    72623859790382857,
								FeeBaseMsat:               12,
								FeeProportionalMillionths: 12000,
								CltvExpiryDelta:           3,
							},
						},
					}},
					PaymentAddr: []byte{168, 213, 214, 148, 231, 104, 254, 123, 61, 242, 97, 45, 197, 193, 155, 230, 240, 120, 80, 14, 154,
						245, 118, 86, 87, 253, 15, 135, 236, 100, 231, 176},
					NumMsat: 100000,
					Features: map[uint32]*lnrpc.Feature{
						1: {
							Name:       "Feature A",
							IsRequired: false,
							IsKnown:    false,
						},
					},
				},
			},
			want: &DecodedInvoice{
				PaymentRequest:    "",
				DestinationPubKey: "test",
				RHash:             "payment_hash",
				Memo:              "Some description encoded in the invoice",
				ValueMsat:         100000,
				CreatedAt:         1662812778,
				PaymentAddr:       "a8d5d694e768fe7b3df2612dc5c19be6f078500e9af5765657fd0f87ec64e7b0",
				FallbackAddr:      "something",
				Expiry:            120,
				CltvExpiry:        124,
				Private:           false,
				Features: featureMap{
					1: {
						Name:       "Feature A",
						IsKnown:    false,
						IsRequired: false,
					},
				},
				RouteHints: []routeHint{{HopHints: []hopHint{
					{
						LNDShortChannelId: 72623859790382856,
						NodeId:            "routing_hint_node_id_1",
						ShortChannelId:    "66051x263430x1800",
						FeeBase:           12,
						CltvExpiryDelta:   3,
						FeeProportional:   12000,
					},
					{
						LNDShortChannelId: 72623859790382857,
						NodeId:            "routing_hint_node_id_2",
						ShortChannelId:    "66051x263430x1801",
						FeeBase:           12,
						CltvExpiryDelta:   3,
						FeeProportional:   12000,
					},
				}}},
			},
		},
		{
			name: "Invoice without features and route hints",
			args: args{
				decoded: &lnrpc.PayReq{
					Destination:     "test",
					PaymentHash:     "payment_hash",
					NumSatoshis:     100,
					Timestamp:       1662812779,
					Expiry:          120,
					Description:     "Some description encoded in the invoice",
					DescriptionHash: "",
					FallbackAddr:    "something",
					CltvExpiry:      124,
					RouteHints:      nil,
					PaymentAddr: []byte{168, 213, 214, 148, 231, 104, 254, 123, 61, 242, 97, 45, 197, 193, 155, 230, 240, 120, 80, 14, 154,
						245, 118, 86, 87, 253, 15, 135, 236, 100, 231, 176},
					NumMsat:  100000,
					Features: nil,
				},
			},
			want: &DecodedInvoice{
				PaymentRequest:    "",
				DestinationPubKey: "test",
				RHash:             "payment_hash",
				Memo:              "Some description encoded in the invoice",
				ValueMsat:         100000,
				CreatedAt:         1662812779,
				PaymentAddr:       "a8d5d694e768fe7b3df2612dc5c19be6f078500e9af5765657fd0f87ec64e7b0",
				FallbackAddr:      "something",
				Expiry:            120,
				CltvExpiry:        124,
				Private:           false,
				Features:          featureMap{},
				RouteHints:        nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := constructDecodedInvoice(tt.args.decoded)
			equal := reflect.DeepEqual(got, tt.want)
			if !equal {
				t.Errorf("constructDecodedInvoice() = \n%v\n, want\n%v", got, tt.want)
			}
		})
	}
}
