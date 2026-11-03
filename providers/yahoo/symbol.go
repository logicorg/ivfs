package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/logicorg/ivfs/internal/pkg/errors"
)

// suffixes maps an exchange, as a person may enter it (its name, a common
// abbreviation, or its MIC), to the suffix Yahoo lists its tickers under.
// The US exchanges have none.
var suffixes = map[string]string{}

func init() {
	for suffix, names := range map[string][]string{
		"":    {"NASDAQ", "NYSE", "NYSE ARCA", "NYSEARCA", "ARCA", "AMEX", "NYSE AMERICAN", "BATS", "CBOE", "XNAS", "XNYS", "ARCX"},
		".DE": {"XETRA", "XETR", "ETR", "GER"},
		".F":  {"FRANKFURT", "FRA", "XFRA"},
		".L":  {"LSE", "LON", "LONDON", "XLON"},
		".MI": {"MIL", "MILAN", "BIT", "BORSA ITALIANA", "XMIL"},
		".PA": {"EPA", "PARIS", "EURONEXT PARIS", "XPAR"},
		".AS": {"AMS", "AMSTERDAM", "EURONEXT AMSTERDAM", "XAMS"},
		".BR": {"EBR", "BRUSSELS", "EURONEXT BRUSSELS", "XBRU"},
		".LS": {"ELI", "LISBON", "EURONEXT LISBON", "XLIS"},
		".IR": {"ISE", "DUBLIN", "EURONEXT DUBLIN", "XDUB"},
		".MC": {"BME", "MADRID", "XMAD"},
		".SW": {"SIX", "SWX", "XSWX"},
		".VI": {"VIE", "VIENNA", "XWBO"},
		".ST": {"STO", "STOCKHOLM", "XSTO"},
		".CO": {"CPH", "COPENHAGEN", "XCSE"},
		".HE": {"HEL", "HELSINKI", "XHEL"},
		".OL": {"OSL", "OSLO", "XOSL"},
		".TO": {"TSX", "TOR", "TORONTO", "XTSE"},
		".AX": {"ASX", "XASX"},
		".T":  {"TSE", "TYO", "TOKYO", "XTKS"},
		".HK": {"HKEX", "HKG", "HONG KONG", "XHKG"},
	} {
		for _, name := range names {
			suffixes[name] = suffix
		}
	}
}

// symbolPattern is the characters a Yahoo symbol is written with, such as
// BRK-B, VWCE.DE, EURUSD=X or ^GSPC.
var symbolPattern = regexp.MustCompile(`^[A-Z0-9^][A-Z0-9.=-]*$`)

// symbol returns Yahoo's symbol for an asset with ticker, on exchange, of
// kind, all as the person entered them. An asset with no ticker is not
// found: Yahoo is looked up by symbol, and an ISIN alone does not give one.
func symbol(ticker, exchange, kind string) (string, error) {
	ticker = strings.ToUpper(strings.TrimSpace(ticker))
	if ticker == "" {
		return "", errors.ErrNotFound{ResourceType: "Symbol", ID: "none, as the asset has no ticker"}
	}
	if !symbolPattern.MatchString(ticker) {
		return "", errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the ticker %q is not one Yahoo lists: write it with letters, digits, dots and dashes, such as BRK.B", ticker)}
	}
	// Yahoo prices a crypto against a currency, and the spec asks for none,
	// so the dollar, which every coin Yahoo lists is quoted in.
	if kind == "crypto" {
		if strings.Contains(ticker, "-") {
			return ticker, nil
		}
		return ticker + "-USD", nil
	}
	exchange = strings.Join(strings.Fields(strings.ToUpper(exchange)), " ")
	if exchange == "" {
		return ticker, nil
	}
	suffix, ok := suffixes[exchange]
	switch {
	case !ok:
		return "", errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the exchange %q is not one the Yahoo provider knows; enter its name or MIC, such as NASDAQ, XETRA, LSE or XMIL, "+
				"or leave it empty to look the ticker up as it is", exchange)}
	case suffix == "":
		// Yahoo writes a share class with a dash: BRK.B is BRK-B.
		return strings.ReplaceAll(ticker, ".", "-"), nil
	case strings.HasSuffix(ticker, suffix):
		return ticker, nil
	default:
		return ticker + suffix, nil
	}
}
