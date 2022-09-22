package lnd

import (
	"context"
	"github.com/cockroachdb/errors"
	_ "github.com/lib/pq"
	"github.com/lightningnetwork/lnd/lnrpc"
	"github.com/lncapital/torq/testutil"
	"github.com/mixer/clock"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"testing"
	"time"
)

// mockLightningClientForwardingHistory is used to moc responses from GetNodeInfo
type mockLightningClientForwardingHistory struct {
	CustomMaxEvents  int32
	ForwardingEvents []*lnrpc.ForwardingEvent
	LastOffsetIndex  uint32
	Error            error
	CancelFunc       func()
}

// TODO: Use fuzzy tests:
//	 https://go.dev/doc/fuzz/
//   https://go.dev/blog/fuzz-beta

// ForwardingHistory mocks the response of LNDs lnrpc.ForwardingHistory
func (c *mockLightningClientForwardingHistory) ForwardingHistory(ctx context.Context,
	in *lnrpc.ForwardingHistoryRequest,
	opts ...grpc.CallOption) (*lnrpc.ForwardingHistoryResponse, error) {

	if c.Error != nil {
		return nil, c.Error
	}

	if c.ForwardingEvents == nil {
		c.CancelFunc()
		return &lnrpc.ForwardingHistoryResponse{}, nil
	}

	r := lnrpc.ForwardingHistoryResponse{
		ForwardingEvents: c.ForwardingEvents,
		LastOffsetIndex:  c.LastOffsetIndex,
	}

	c.ForwardingEvents = nil

	return &r, nil
}

func TestFetchForwardingHistoryError(t *testing.T) {

	mClient := mockLightningClientForwardingHistory{
		Error: errors.New("Some error"),
	}

	ctx := context.Background()
	_, err := fetchForwardingHistory(ctx, &mClient, 0, 1000)

	testutil.Given(t, "While fetching forwarding history")

	testutil.WhenF(t, "If lnrcp.ForwardingHistory returns an error.")

	if err != nil {
		testutil.Successf(t, "fetchForwardingHistory returns the error")
	} else {
		testutil.Errorf(t, "fetchForwardingHistory returns the error")
	}

}

func TestSubscribeForwardingEvents(t *testing.T) {

	ctx := context.Background()
	errs, ctx := errgroup.WithContext(ctx)
	ctx, stopSubFwE := context.WithCancel(ctx)
	c := clock.NewMockClock(time.Unix(0, 0))

	srv, err := testutil.InitTestDBConn()
	if err != nil {
		panic(err)
	}

	db, err := srv.NewTestDatabase(true)
	if err != nil {
		t.Fatal(err)
	}
	//defer db.Close()

	mockTickerInterval := 3000 * time.Second
	me := 1000
	opt := FwhOptions{
		MaxEvents: &me,
		Tick:      c.Tick(mockTickerInterval),
	}

	mclient := mockLightningClientForwardingHistory{
		CancelFunc: stopSubFwE,
		ForwardingEvents: []*lnrpc.ForwardingEvent{
			{
				ChanIdIn:    1234,
				ChanIdOut:   2345,
				AmtIn:       11,
				AmtOut:      10,
				Fee:         1,
				FeeMsat:     1000,
				AmtInMsat:   11000,
				AmtOutMsat:  10000,
				TimestampNs: uint64(c.Now().UnixNano()),
			},
			{
				ChanIdIn:    1234,
				ChanIdOut:   2345,
				AmtIn:       11,
				AmtOut:      10,
				Fee:         1,
				FeeMsat:     1000,
				AmtInMsat:   11000,
				AmtOutMsat:  10000,
				TimestampNs: uint64(c.Now().UnixNano()) + 500000000,
			},
			{ // Duplicate record used for testing
				ChanIdIn:    1234,
				ChanIdOut:   2345,
				AmtIn:       11,
				AmtOut:      10,
				Fee:         1,
				FeeMsat:     1000,
				AmtInMsat:   11000,
				AmtOutMsat:  10000,
				TimestampNs: uint64(c.Now().UnixNano()) + 500000000,
			},
			{
				ChanIdIn:    1234,
				ChanIdOut:   2345,
				AmtIn:       11,
				AmtOut:      10,
				Fee:         1,
				FeeMsat:     1000,
				AmtInMsat:   11000,
				AmtOutMsat:  10000,
				TimestampNs: uint64(c.Now().UnixNano()) + 1000000000,
			},
		},
		LastOffsetIndex: 0,
	}

	// Start subscribing in a goroutine to allow the test to continue simulating time through the
	// mocked time object.
	errs.Go(func() error {
		err := SubscribeForwardingEvents(ctx, &mclient, db, &opt)
		if err != nil {
			t.Fatal(errors.Wrapf(err, "SubscribeForwardingEvents(%v, %v, %v, %v)", ctx,
				mclient, db, &opt))
		}
		return nil
	})

	// Simulate passing intervals, one more than required to process
	numbTicks := 2
	for i := 0; i < numbTicks; i++ {
		c.AddTime(mockTickerInterval)
	}

	// Check for potential errors from the goroutine (SubscribeForwardingEvents)
	err = errs.Wait()
	if err != nil {
		t.Fatal(err)
	}

	testutil.Given(t, "While running SubscribeForwardingEvents")

	testutil.WhenF(t, "We need to check that fetchLastForwardTime returns the expected nanosecond.")
	{
		var expected uint64 = 1000000000
		returned, err := fetchLastForwardTime(db)
		switch {
		case err != nil:
			testutil.Fatalf(t, "We get an error: %v", err)
		case returned != expected:
			testutil.Errorf(t, " "+
				"We expected %d got %d", expected, returned)
		case returned == expected:
			testutil.Successf(t, "We got the expected nanosecond ")
		}

	}

	testutil.WhenF(t, "We need to check that storeForwardingHistory only stores unique records.")
	{
		var expectedUnique = 3
		var returned int
		err := db.QueryRow("select count(*) from forward;").Scan(&returned)

		switch {
		case err != nil:
			testutil.Fatalf(t, "We get an error: %v", err)
		case returned != expectedUnique:
			testutil.Errorf(t, "We expected to store %d records but stored %d", expectedUnique,
				returned)
		case returned == expectedUnique:
			testutil.Successf(t, "We stored the expected number of records")
		}
	}

	var returnedDate time.Time
	err = db.QueryRow("select \"time\" from forward order by time_ns desc limit 1;").Scan(&returnedDate)
	var expectedDate, _ = time.Parse("2006-01-02 15:04:05-0700 MST", "1970-01-01 00:00:01+0000 UTC")
	if returnedDate != expectedDate {
		t.Errorf("Time on the latest forward record (%v) isn't as expected (%v)", returnedDate, expectedDate)
	}

	db.Close()
	// err = srv.Cleanup()
	// if err != nil {
	// 	t.Fatal(err)
	// }

}
