package main

import (
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/shopspring/decimal"
)

type copyPortfolio struct {
	leader    trader
	followers []trader
}

func (cp *copyPortfolio) init() {
	cp.leader.openTrades = make(map[string]*openTrade)
	for i := range cp.followers {
		cp.followers[i].openTrades = make(map[string]*openTrade)
	}
}

func (cp *copyPortfolio) joinNewFollower() int {
	cp.followers = append(cp.followers, trader{openTrades: make(map[string]*openTrade)})
	return len(cp.followers) - 1
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

	for _, name := range order {
		w.Write([]byte(")\t " + strings.ToUpper(name) + ":\t"))
		if trade, ok := cp.leader.openTrades[name]; ok {
			w.Write([]byte(trade.amount.StringFixedBank(8)))
			w.Write([]byte("\t(%P: "))
			w.Write([]byte((trade.amount.Mul(marketPrices[name]).Div(lTotal).Mul(decimal.New(100, 0))).StringFixedBank(2)))
		} else {
			w.Write([]byte("0.00\t(%P: 0.00"))
		}
	}

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

		for _, name := range order {
			w.Write([]byte(")\t " + strings.ToUpper(name) + ":\t"))
			if trade, ok := f.openTrades[name]; ok {
				w.Write([]byte(trade.amount.StringFixedBank(8)))
				w.Write([]byte("\t(%P: "))
				w.Write([]byte((trade.amount.Mul(marketPrices[name]).Div(fTotal).Mul(decimal.New(100, 0))).StringFixedBank(2)))
			} else {
				w.Write([]byte("0.00\t(%P: 0.00"))
			}
		}

		w.Write([]byte(")\n"))
	}
	w.Flush()
	return s.String()
}

type trader struct {
	baseCurrency decimal.Decimal
	openTrades   map[string]*openTrade
}

type openTrade struct {
	entryPrice decimal.Decimal
	amount     decimal.Decimal
}

// Calculates and return total portfolio value of trader, calculated in base currency.
func (t *trader) calcTotalPortValue() decimal.Decimal {
	total := t.baseCurrency
	for name, trade := range t.openTrades {
		if price, ok := marketPrices[name]; ok {
			total = total.Add(trade.amount.Mul(price))
		} else {
			panic("no price for asset " + name)
		}
	}
	return total
}

// Low-level buy func
func (t *trader) buy(asset string, amount, totalCost decimal.Decimal) {
	if amount.IsZero() {
		return
	}

	assetPrice := marketPrices[asset]
	t.baseCurrency = t.baseCurrency.Sub(totalCost)
	if trade, ok := t.openTrades[asset]; ok {
		trade.amount = trade.amount.Add(amount)
		trade.entryPrice = assetPrice
	} else {
		t.openTrades[asset] = &openTrade{entryPrice: assetPrice, amount: amount}
	}
}

// Low-level sell func
func (t *trader) sell(asset string, amount decimal.Decimal) {
	openTrade := t.openTrades[asset]
	openTrade.amount = openTrade.amount.Sub(amount)
	got := amount.Mul(marketPrices[asset])
	proceeds := got.Sub(got.Mul(fee.Sub(decimal.New(1, 0))))

	cost := openTrade.entryPrice.Mul(openTrade.amount)
	profit := proceeds.Sub(cost)

	// share % of profit with leader
	// Assume follower profit split is 80F/20L
	leaderShare := profit.Mul(decimal.NewFromFloat(0.2))

	t.baseCurrency = t.baseCurrency.Add(proceeds.Sub(leaderShare))
}
