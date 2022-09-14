package invoices

import (
	"github.com/lightningnetwork/lnd/lnrpc"
	"reflect"
	"testing"
)

func Test_processInvoiceReq(t *testing.T) {
	var valueMsat int64 = 11
	var memo = "test"
	var rPreImage = "024bf894b017051472911cb3db5097a825e2fc9a5602c824ff7bbea2a625f40972"
	var rPreImageByte = []byte{2, 75, 248, 148, 176, 23, 5, 20, 114, 145, 28, 179, 219, 80, 151,
		168, 37, 226, 252, 154, 86, 2, 200, 36, 255, 123, 190, 162, 166, 37, 244, 9, 114}
	var expiry int64 = 12
	var fallBackAddress = "test"
	var private = true
	var amp = true

	tests := []struct {
		name    string
		input   newInvoiceRequest
		want    lnrpc.Invoice
		wantErr bool
	}{
		{
			"Node ID missing",
			newInvoiceRequest{
				ValueMsat: &valueMsat,
			},
			lnrpc.Invoice{
				ValueMsat: 11,
			},
			true,
		},
		{
			"Only ValueMSat provided",
			newInvoiceRequest{
				NodeId:    1,
				ValueMsat: &valueMsat,
			},
			lnrpc.Invoice{
				ValueMsat: 11,
			},
			false,
		},
		{
			"All params provided",
			newInvoiceRequest{
				NodeId:          1,
				Memo:            &memo,
				RPreImage:       &rPreImage,
				ValueMsat:       &valueMsat,
				Expiry:          &expiry,
				FallBackAddress: &fallBackAddress,
				Private:         &private,
				IsAmp:           &amp,
			},
			lnrpc.Invoice{
				Memo:         "test",
				RPreimage:    rPreImageByte,
				ValueMsat:    11,
				Expiry:       12,
				FallbackAddr: "test",
				Private:      true,
				IsAmp:        true,
			},
			false,
		},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := processInvoiceReq(test.input)

			if err != nil {
				if test.wantErr {
					return
				}
				t.Errorf("processInvoiceReq error: %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("%d: processInvoiceReq()\nGot:\n%v\nWant:\n%v\n", i, got, test.want)
			}

		})
	}
}
