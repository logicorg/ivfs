package main

import (
	"testing"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestSymbol(t *testing.T) {
	tests := map[string]struct {
		ticker, exchange, kind string
		want                   string
		wantErr                error
	}{
		"a US stock":                       {ticker: "AAPL", exchange: "NASDAQ", kind: "stock", want: "AAPL"},
		"a US share class":                 {ticker: "BRK.B", exchange: "NYSE", kind: "stock", want: "BRK-B"},
		"an ETF on XETRA":                  {ticker: "VWCE", exchange: "XETRA", kind: "etf", want: "VWCE.DE"},
		"an exchange by its MIC":           {ticker: "VWCE", exchange: "XETR", kind: "etf", want: "VWCE.DE"},
		"an exchange as typed":             {ticker: " swda ", exchange: "  borsa   italiana ", kind: "etf", want: "SWDA.MI"},
		"a ticker with the suffix already": {ticker: "VWCE.DE", exchange: "XETRA", kind: "etf", want: "VWCE.DE"},
		"no exchange, as it is":            {ticker: "VWCE.DE", kind: "etf", want: "VWCE.DE"},
		"a crypto, in dollars":             {ticker: "BTC", kind: "crypto", want: "BTC-USD"},
		"a crypto with its pair":           {ticker: "BTC-EUR", kind: "crypto", want: "BTC-EUR"},
		"a crypto ignores the exchange":    {ticker: "ETH", exchange: "Kraken", kind: "crypto", want: "ETH-USD"},
		"no ticker": {
			exchange: "XETRA", kind: "fund", wantErr: errors.ErrNotFound{ResourceType: "Symbol", ID: "none, as the asset has no ticker"},
		},
		"a ticker with a slash": {
			ticker: "BRK/B", exchange: "NYSE", kind: "stock",
			wantErr: errors.ErrInvalidParameter{Description: `the ticker "BRK/B" is not one Yahoo lists: ` +
				"write it with letters, digits, dots and dashes, such as BRK.B"},
		},
		"an exchange it does not know": {
			ticker: "VWCE", exchange: "Bolsa de Lima", kind: "etf",
			wantErr: errors.ErrInvalidParameter{Description: `the exchange "BOLSA DE LIMA" is not one the Yahoo provider knows; ` +
				"enter its name or MIC, such as NASDAQ, XETRA, LSE or XMIL, or leave it empty to look the ticker up as it is"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := symbol(tt.ticker, tt.exchange, tt.kind)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
