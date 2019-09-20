package main

import (
	"github.com/shopspring/decimal"
	"strconv"
	"strings"
	"text/tabwriter"
)

type copyPortfolio struct {
	leader    trader
	followers []trader
}

func (cp *copyPortfolio) init() {
	cp.leader.openTrades = make(map[string]decimal.Decimal)
	for i := range cp.followers {
		cp.followers[i].openTrades = make(map[string]decimal.Decimal)
	}
}

func (cp *copyPortfolio) String() string {
	lTotal := cp.leader.calcTotalPortValue()
	s := strings.Builder{}
	w := tabwriter.NewWriter(&s, 12, 16, 0, ' ', 0)

	w.Write([]byte("Leader:\n"))

	w.Write([]byte("Base:\t "))
	w.Write([]byte(cp.leader.baseCurrency.StringFixedBank(2)))
	w.Write([]byte("\t(%P: "))
	w.Write([]byte((cp.leader.baseCurrency.Div(lTotal).Mul(decimal.New(100, 0))).StringFixedBank(2)))

	w.Write([]byte(")\t BTC:\t"))
	w.Write([]byte(cp.leader.openTrades["btc"].StringFixedBank(2)))
	w.Write([]byte("\t(%P: "))
	w.Write([]byte((cp.leader.openTrades["btc"].Mul(marketPrices["btc"]).Div(lTotal).Mul(decimal.New(100, 0))).StringFixedBank(2)))

	w.Write([]byte(")\n"))
	for i := range cp.followers {
		f := &cp.followers[i]
		fTotal := f.calcTotalPortValue()

		w.Write([]byte("Follower " + strconv.Itoa(i)))
		w.Write([]byte(":\n"))

		w.Write([]byte("Base:\t "))
		w.Write([]byte(f.baseCurrency.StringFixedBank(2)))
		w.Write([]byte("\t(%P: "))
		w.Write([]byte((f.baseCurrency.Div(fTotal).Mul(decimal.New(100, 0))).StringFixedBank(2)))

		w.Write([]byte(")\t BTC:\t"))
		w.Write([]byte(f.openTrades["btc"].StringFixedBank(2)))
		w.Write([]byte("\t(%P: "))
		w.Write([]byte((f.openTrades["btc"].Mul(marketPrices["btc"]).Div(fTotal).Mul(decimal.New(100, 0))).StringFixedBank(2)))

		w.Write([]byte(")\n"))
	}
	w.Flush()
	return s.String()
}

type trader struct {
	baseCurrency decimal.Decimal
	openTrades   map[string]decimal.Decimal
}


// Calculates and return total portfolio value of trader, calculated in base currency.
func (t *trader) calcTotalPortValue() decimal.Decimal {
	total := t.baseCurrency
	for name, amount := range t.openTrades {
		if price, ok := marketPrices[name]; ok {
			total = total.Add(amount.Mul(price))
		} else {
			panic("no price for asset " + name)
		}
	}
	return total
}

// Low-level buy func
func (t *trader) buy(asset string, amount, totalCost decimal.Decimal) {
	t.baseCurrency = t.baseCurrency.Sub(totalCost)
	if ass, ok := t.openTrades[asset]; ok {
		t.openTrades[asset] = ass.Add(amount)
	} else {
		t.openTrades[asset] = amount
	}
}

// Low-level sell func
func (t *trader) sell(asset string, amount decimal.Decimal) {
	t.openTrades[asset] = t.openTrades[asset].Sub(amount)
	got := amount.Mul(marketPrices[asset])
	proceeds := got.Sub(got.Mul(fee.Sub(decimal.New(1, 0))))
	t.baseCurrency = t.baseCurrency.Add(proceeds)
}
