package main

import (
	"container/list"
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
	cp.leader.positions = make(map[string]*position)
	for i := range cp.followers {
		cp.followers[i].positions = make(map[string]*position)
	}
}

func (cp *copyPortfolio) joinNewFollower() int {
	cp.followers = append(cp.followers, trader{positions: make(map[string]*position)})
	return len(cp.followers) - 1
}

func (cp *copyPortfolio) String() string {
	lTotal := cp.leader.calcTotalPortValue()
	s := strings.Builder{}
	w := tabwriter.NewWriter(&s, 16, 24, 0, ' ', 0)

	w.Write([]byte("Leader:\n"))

	w.Write([]byte("Base:\t "))
	w.Write([]byte(cp.leader.baseCurrency.StringFixedBank(2)))
	w.Write([]byte("\t(%P: "))
	w.Write([]byte((cp.leader.baseCurrency.Div(lTotal).Mul(decimal.New(100, 0))).StringFixedBank(4)))

	for _, name := range order {
		w.Write([]byte(")\t " + strings.ToUpper(name) + ":\t"))
		if trade, ok := cp.leader.positions[name]; ok {
			w.Write([]byte(trade.totalAmount.StringFixedBank(8)))
			w.Write([]byte("\t(%P: "))
			w.Write([]byte((trade.totalAmount.Mul(marketPrices[name]).Div(lTotal).Mul(decimal.New(100, 0))).StringFixedBank(4)))
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
		w.Write([]byte((f.baseCurrency.Div(fTotal).Mul(decimal.New(100, 0))).StringFixedBank(4)))

		for _, name := range order {
			w.Write([]byte(")\t " + strings.ToUpper(name) + ":\t"))
			if trade, ok := f.positions[name]; ok {
				w.Write([]byte(trade.totalAmount.StringFixedBank(8)))
				w.Write([]byte("\t(%P: "))
				w.Write([]byte((trade.totalAmount.Mul(marketPrices[name]).Div(fTotal).Mul(decimal.New(100, 0))).StringFixedBank(4)))
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
	positions    map[string]*position
}

type position struct {
	totalAmount decimal.Decimal
	buys        *list.List
}

type buy struct {
	price  decimal.Decimal
	amount decimal.Decimal
}

// Calculates and return total portfolio value of trader, calculated in base currency.
func (t *trader) calcTotalPortValue() decimal.Decimal {
	total := t.baseCurrency
	for name, pos := range t.positions {
		if price, ok := marketPrices[name]; ok {
			total = total.Add(pos.totalAmount.Mul(price))
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

	currentPrice := marketPrices[asset]
	t.baseCurrency = t.baseCurrency.Sub(totalCost)
	if pos, ok := t.positions[asset]; ok {
		pos.totalAmount = pos.totalAmount.Add(amount)
		pos.buys.PushBack(&buy{price: currentPrice, amount: amount})
	} else {
		t.positions[asset] = &position{totalAmount: amount, buys: list.New()}
		t.positions[asset].buys.PushBack(&buy{price: currentPrice, amount: amount})
	}
}

// Low-level sell func
// Returns trader's gains, negative or positive
func (t *trader) sell(asset string, amount decimal.Decimal) decimal.Decimal {
	pos := t.positions[asset]

	toSell, currentPrice := amount, marketPrices[asset]
	var gains decimal.Decimal
	for b := pos.buys.Front(); b != nil; b = b.Next() {
		bu := b.Value.(*buy)

		if toSell.LessThanOrEqual(bu.amount) {
			cost, got := toSell.Mul(bu.price), toSell.Mul(currentPrice)
			gains = gains.Add(got.Sub(cost))
			bu.amount = bu.amount.Sub(toSell)
			break
		} else { // have to close this buy and move on to next
			cost, got := bu.amount.Mul(bu.price), bu.amount.Mul(currentPrice)
			gains = gains.Add(got.Sub(cost))
			toSell = toSell.Sub(bu.amount)
			pos.buys.Remove(b)
		}
	}

	pos.totalAmount = pos.totalAmount.Sub(amount)

	got := amount.Mul(currentPrice)
	proceeds := got.Sub(got.Mul(fee.Sub(decimal.New(1, 0))))
	t.baseCurrency = t.baseCurrency.Add(proceeds)
	return gains
}
