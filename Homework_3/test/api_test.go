//go:build api

package apitest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	pb "homework2/pkg/api/product"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	bookSKU   int64  = 1076963
	bookName         = "Теория нравственных чувств | Смит Адам"
	bookPrice uint32 = 3379

	cookSKU   int64  = 1148162
	cookName         = "Кулинар Гуров"
	cookPrice uint32 = 2931
)

var userSeq atomic.Int64

func nextUser() int64 {
	return time.Now().UnixNano() + userSeq.Add(1)
}

func postCart(t *testing.T, userID, sku int64, count uint32) int {
	t.Helper()
	body, err := json.Marshal(map[string]uint32{"count": count})
	require.NoError(t, err)
	url := fmt.Sprintf("http://localhost:8080/user/%d/cart/%d", userID, sku)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	resp, err := httpClient().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

type cartItem struct {
	SKU   int64  `json:"sku"`
	Name  string `json:"name"`
	Count uint32 `json:"count"`
	Price uint32 `json:"price"`
}

type cartView struct {
	Items      []cartItem `json:"items"`
	TotalPrice uint32     `json:"total_price"`
}

func getCart(t *testing.T, userID int64) (int, cartView) {
	t.Helper()
	url := fmt.Sprintf("http://localhost:8080/user/%d/cart", userID)
	resp, err := httpClient().Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	var view cartView
	if resp.StatusCode == http.StatusOK {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&view))
	}
	return resp.StatusCode, view
}

func TestProductAndCart(t *testing.T) {
	t.Parallel()

	t.Run("get product", func(t *testing.T) {
		t.Parallel()
		client := productClient(t)
		resp, err := client.GetProduct(t.Context(), &pb.GetProductRequest{Sku: bookSKU})
		require.NoError(t, err)
		require.Equal(t, bookSKU, resp.GetSku())
		require.Equal(t, bookName, resp.GetName())
		require.Equal(t, int32(bookPrice), resp.GetPrice())
	})

	t.Run("product rejects bad sku", func(t *testing.T) {
		t.Parallel()
		client := productClient(t)
		_, err := client.GetProduct(t.Context(), &pb.GetProductRequest{Sku: 0})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("product rejects negative sku", func(t *testing.T) {
		t.Parallel()
		client := productClient(t)
		_, err := client.GetProduct(t.Context(), &pb.GetProductRequest{Sku: -1})
		require.Equal(t, codes.InvalidArgument, status.Code(err))
	})

	t.Run("product missing sku", func(t *testing.T) {
		t.Parallel()
		client := productClient(t)
		_, err := client.GetProduct(t.Context(), &pb.GetProductRequest{Sku: 42})
		require.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("add and get cart", func(t *testing.T) {
		t.Parallel()
		userID := nextUser()
		require.Equal(t, http.StatusOK, postCart(t, userID, cookSKU, 1))
		require.Equal(t, http.StatusOK, postCart(t, userID, bookSKU, 2))
		require.Equal(t, http.StatusOK, postCart(t, userID, bookSKU, 1))

		code, view := getCart(t, userID)
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, []cartItem{
			{SKU: bookSKU, Name: bookName, Count: 3, Price: bookPrice},
			{SKU: cookSKU, Name: cookName, Count: 1, Price: cookPrice},
		}, view.Items)
		require.Equal(t, bookPrice*3+cookPrice, view.TotalPrice)
	})

	t.Run("add rejects bad user", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, http.StatusBadRequest, postCart(t, 0, bookSKU, 1))
	})

	t.Run("add rejects bad sku", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, http.StatusBadRequest, postCart(t, nextUser(), 0, 1))
	})

	t.Run("add rejects bad count", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, http.StatusBadRequest, postCart(t, nextUser(), bookSKU, 0))
	})

	t.Run("add missing sku", func(t *testing.T) {
		t.Parallel()
		require.Equal(t, http.StatusPreconditionFailed, postCart(t, nextUser(), 42, 1))
	})

	t.Run("get rejects bad user", func(t *testing.T) {
		t.Parallel()
		code, _ := getCart(t, 0)
		require.Equal(t, http.StatusBadRequest, code)
	})

	t.Run("empty cart", func(t *testing.T) {
		t.Parallel()
		code, _ := getCart(t, nextUser())
		require.Equal(t, http.StatusNotFound, code)
	})
}
