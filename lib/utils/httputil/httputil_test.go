package httputil

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pressly/chi"
	"github.com/stretchr/testify/require"

	"github.com/uber/makisu/mocks/net/http"
)

const _testURL = "http://localhost:0/test"

func startServer(t *testing.T) (string, func()) {
	require := require.New(t)
	l, err := net.Listen("tcp", ":0")
	require.NoError(err)
	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})
	go http.Serve(l, r)
	return l.Addr().String(), func() { l.Close() }
}

func newResponse(status int) *http.Response {
	// We need to set a dummy request in the response so NewStatusError
	// can access the "original" URL.
	dummyReq, err := http.NewRequest("GET", _testURL, nil)
	if err != nil {
		panic(err)
	}

	rec := httptest.NewRecorder()
	rec.WriteHeader(status)
	resp := rec.Result()
	resp.Request = dummyReq

	return resp
}

func TestSendRetry(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transport := mockhttp.NewMockRoundTripper(ctrl)

	for _, status := range []int{503, 500, 200} {
		transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(status), nil)
	}

	start := time.Now()
	_, err := Get(
		_testURL,
		SendRetry(RetryMax(5), RetryInterval(200*time.Millisecond)),
		SendTransport(transport))
	require.NoError(err)
	require.InDelta(400*time.Millisecond, time.Since(start), float64(50*time.Millisecond))
}

func TestSendRetryOnTransportErrors(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transport := mockhttp.NewMockRoundTripper(ctrl)

	transport.EXPECT().RoundTrip(gomock.Any()).Return(nil, errors.New("some network error")).Times(3)

	start := time.Now()
	_, err := Get(
		_testURL,
		SendRetry(RetryMax(3), RetryInterval(200*time.Millisecond)),
		SendTransport(transport))
	require.Error(err)
	require.InDelta(400*time.Millisecond, time.Since(start), float64(50*time.Millisecond))
}

func TestSendRetryOn5XX(t *testing.T) {
	require := require.New(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	transport := mockhttp.NewMockRoundTripper(ctrl)

	transport.EXPECT().RoundTrip(gomock.Any()).Return(newResponse(503), nil).Times(3)

	start := time.Now()
	_, err := Get(
		_testURL,
		SendRetry(RetryMax(3), RetryInterval(200*time.Millisecond)),
		SendTransport(transport))
	require.Error(err)
	require.Equal(503, err.(StatusError).Status)
	require.InDelta(400*time.Millisecond, time.Since(start), float64(50*time.Millisecond))
}

func TestStatusChecking(t *testing.T) {
	err := StatusError{Status: http.StatusCreated}
	require.True(t, IsCreated(err))
	err = StatusError{Status: http.StatusNotFound}
	require.True(t, IsNotFound(err))
	err = StatusError{Status: http.StatusConflict}
	require.True(t, IsConflict(err))
	err = StatusError{Status: http.StatusAccepted}
	require.True(t, IsAccepted(err))
	err = StatusError{Status: http.StatusForbidden}
	require.True(t, IsForbidden(err))
}
